package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"zoo-go/internal/domain"
	"zoo-go/internal/payment"
	"zoo-go/internal/store"
)

type testAPI struct {
	base string
	st   *store.Store
}

type fakePaymentGateway struct{}

func (fakePaymentGateway) CreateJSAPI(_ context.Context, attempt domain.PaymentAttempt) (domain.PaymentParams, error) {
	return domain.PaymentParams{
		TimeStamp: "1", NonceStr: "nonce", Package: "prepay_id=test-" + attempt.ID,
		SignType: "RSA", PaySign: "signature",
	}, nil
}

func (fakePaymentGateway) QueryPayment(_ context.Context, outTradeNo string) (domain.PaymentNotice, error) {
	return domain.PaymentNotice{OutTradeNo: outTradeNo, TradeState: "NOTPAY"}, nil
}

func (fakePaymentGateway) ClosePayment(_ context.Context, _ string) error { return nil }

func (fakePaymentGateway) CreateRefund(_ context.Context, request payment.RefundRequest) (domain.RefundNotice, error) {
	return domain.RefundNotice{
		OutRefundNo: request.OutRefundNo, WeChatRefundID: "wx-refund-test",
		Status: domain.RefundSucceeded, AmountFen: request.RefundFen, SuccessAt: time.Now(),
	}, nil
}

func (fakePaymentGateway) QueryRefund(_ context.Context, outRefundNo string) (domain.RefundNotice, error) {
	return domain.RefundNotice{OutRefundNo: outRefundNo, Status: domain.RefundSucceeded}, nil
}

func (fakePaymentGateway) ParsePaymentNotification(http.Header, []byte) (payment.PaymentEvent, error) {
	return payment.PaymentEvent{}, errors.New("not implemented in test gateway")
}

func (fakePaymentGateway) ParseRefundNotification(http.Header, []byte) (payment.RefundEvent, error) {
	return payment.RefundEvent{}, errors.New("not implemented in test gateway")
}

func newTestAPI(t *testing.T) *testAPI {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	cfg := Config{
		DevAuth: true, AdminToken: "admin-test", UploadDir: filepath.Join(t.TempDir(), "uploads"),
		AdminDir: filepath.Join("..", "..", "admin"),
		Payments: payment.NewService(st, fakePaymentGateway{}, nil),
	}
	server := httptest.NewServer(New(st, cfg, nil).Handler())
	t.Cleanup(server.Close)
	return &testAPI{base: server.URL, st: st}
}

func (api *testAPI) loginWeChat(t *testing.T, openID, name string) (string, string) {
	t.Helper()
	session, err := api.st.Login(openID, name, "", "", time.Now())
	if err != nil {
		t.Fatalf("store.Login() error = %v", err)
	}
	return session.Token, session.Profile.ID
}

func (api *testAPI) request(t *testing.T, method, path, token string, body any, wantStatus int) map[string]any {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, api.base+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s status = %d, want %d, body=%s", method, path, resp.StatusCode, wantStatus, data)
	}
	if len(data) == 0 {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode response: %v, body=%s", err, data)
	}
	return value
}

func (api *testAPI) adminRequest(t *testing.T, method, path string, body any, wantStatus int) map[string]any {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal admin request: %v", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, api.base+path, reader)
	if err != nil {
		t.Fatalf("new admin request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Admin-Token", "admin-test")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("admin request %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s status = %d, want %d, body=%s", method, path, resp.StatusCode, wantStatus, data)
	}
	if len(data) == 0 {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode admin response: %v, body=%s", err, data)
	}
	return value
}

func login(t *testing.T, api *testAPI, device string, wantStatus int) (string, string) {
	t.Helper()
	result := api.request(t, http.MethodPost, "/v1/auth/wechat/login", "", map[string]any{
		"code": "dev-code", "deviceId": device, "nickname": device,
	}, wantStatus)
	if wantStatus != http.StatusOK {
		return "", ""
	}
	data := result["data"].(map[string]any)
	profile := data["profile"].(map[string]any)
	return data["token"].(string), profile["id"].(string)
}

func TestAuthenticationAndEmptyDatabase(t *testing.T) {
	api := newTestAPI(t)
	api.request(t, http.MethodGet, "/v1/owner/home", "", nil, http.StatusUnauthorized)
	token, _ := login(t, api, "owner-device", http.StatusOK)
	result := api.request(t, http.MethodGet, "/v1/owner/home", token, nil, http.StatusOK)
	home := result["data"].(map[string]any)
	if len(home["pets"].([]any)) != 0 || len(home["tasks"].([]any)) != 0 {
		t.Fatalf("new account should not contain mock business data: %+v", home)
	}
	community := home["community"].(map[string]any)
	if community["total"].(float64) != 0 {
		t.Fatalf("new community should be empty: %+v", community)
	}
}

func TestPaymentUnavailableWithoutMerchantConfiguration(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	server := httptest.NewServer(New(st, Config{DevAuth: true, AdminToken: "admin-test"}, nil).Handler())
	t.Cleanup(server.Close)
	api := &testAPI{base: server.URL, st: st}
	token, _ := login(t, api, "no-payment-device", http.StatusOK)
	result := api.request(t, http.MethodPost, "/v1/orders/missing/payments", token, nil, http.StatusServiceUnavailable)
	code := result["error"].(map[string]any)["code"]
	if code != "PAYMENT_NOT_CONFIGURED" {
		t.Fatalf("error code = %v", code)
	}
}

func TestProfileUpdate(t *testing.T) {
	api := newTestAPI(t)
	token, userID := login(t, api, "profile-device", http.StatusOK)

	result := api.request(t, http.MethodPost, "/v1/profile", token, map[string]any{
		"name": "小园", "avatar": "/uploads/avatar.png",
	}, http.StatusOK)
	profile := result["data"].(map[string]any)
	if profile["id"] != userID || profile["name"] != "小园" || profile["avatar"] != "/uploads/avatar.png" {
		t.Fatalf("profile update response = %+v", profile)
	}

	me := api.request(t, http.MethodGet, "/v1/auth/me", token, nil, http.StatusOK)
	if me["data"].(map[string]any)["name"] != "小园" {
		t.Fatalf("auth/me did not return updated profile: %+v", me)
	}

	api.request(t, http.MethodPost, "/v1/profile", token, map[string]any{"name": "", "avatar": ""}, http.StatusBadRequest)
}

func TestAuthenticatedEndToEndAndCommunity(t *testing.T) {
	api := newTestAPI(t)
	ownerToken, _ := api.loginWeChat(t, "wx_owner_e2e", "主人A")
	studentToken, studentID := api.loginWeChat(t, "wx_student_e2e", "学生A")

	api.request(t, http.MethodPost, "/v1/student/verification", studentToken, map[string]any{
		"realName": "学生A", "school": "中国农业大学", "studentNo": "20240001",
		"major": "动物医学", "enrollmentYear": 2024, "chsiCode": "E2E20261105A1",
	}, http.StatusOK)
	adminReq, _ := http.NewRequest(http.MethodPost, api.base+"/v1/admin/students/"+studentID+"/approve", nil)
	adminReq.Header.Set("X-Admin-Token", "admin-test")
	adminResp, err := http.DefaultClient.Do(adminReq)
	if err != nil {
		t.Fatalf("admin approve: %v", err)
	}
	adminResp.Body.Close()
	if adminResp.StatusCode != http.StatusOK {
		t.Fatalf("admin approve status = %d", adminResp.StatusCode)
	}
	api.request(t, http.MethodPost, "/v1/student/location", studentToken, map[string]any{
		"latitude": 39.9998, "longitude": 116.3269, "radiusKm": 5,
	}, http.StatusOK)

	petResult := api.request(t, http.MethodPost, "/v1/pets", ownerToken, map[string]any{
		"name": "豆豆", "species": "狗", "breed": "金毛", "age": "3岁", "note": "每餐一勺半", "flags": []string{"温顺"},
	}, http.StatusCreated)
	petID := petResult["data"].(map[string]any)["id"].(string)

	serviceDate := time.Now().AddDate(0, 0, 2).Format("2006-01-02")
	taskResult := api.request(t, http.MethodPost, "/v1/tasks", ownerToken, map[string]any{
		"petId": petID, "services": []string{"喂食", "换水", "拍照反馈"},
		"serviceDate": serviceDate, "timeWindow": "09:00-10:00", "mode": "bid",
		"budgetFen": 6000, "bidDurationMinutes": 60, "community": "马连洼·朗润园附近",
		"locationName": "朗润园东门", "latitude": 39.9998, "longitude": 116.3269,
		"addressDetail": "3号楼2单元401", "note": "按清单完成",
		"safetyConfirmed": true, "agreementAccepted": true,
	}, http.StatusCreated)
	task := taskResult["data"].(map[string]any)
	taskID := task["id"].(string)

	bidResult := api.request(t, http.MethodPost, "/v1/tasks/"+taskID+"/bids", studentToken, map[string]any{
		"priceFen": 4500, "message": "可以准时到",
	}, http.StatusOK)
	bidID := bidResult["data"].(map[string]any)["myBid"].(map[string]any)["id"].(string)

	studentTask := api.request(t, http.MethodGet, "/v1/tasks/"+taskID+"?role=student", studentToken, nil, http.StatusOK)
	studentTaskData := studentTask["data"].(map[string]any)
	if detail, ok := studentTaskData["addressDetail"]; ok && detail != "" {
		t.Fatalf("student saw detailed address: %+v", studentTaskData)
	}

	awarded := api.request(t, http.MethodPost, "/v1/tasks/"+taskID+"/award", ownerToken, map[string]any{"bidId": bidID}, http.StatusOK)
	orderID := awarded["data"].(map[string]any)["orderId"].(string)
	challenge := api.request(t, http.MethodPost, "/v1/orders/"+orderID+"/payments", ownerToken, nil, http.StatusCreated)
	challengeData := challenge["data"].(map[string]any)
	if _, _, err := api.st.ApplyPaymentNotice(domain.PaymentNotice{
		OutTradeNo: challengeData["outTradeNo"].(string), TransactionID: "wx-e2e-transaction",
		TradeState: "SUCCESS", AmountFen: 4500, SuccessAt: time.Now(),
	}, time.Now()); err != nil {
		t.Fatalf("ApplyPaymentNotice() error = %v", err)
	}
	api.request(t, http.MethodPost, "/v1/orders/"+orderID+"/checkin", studentToken, map[string]any{}, http.StatusOK)

	firstURL := api.uploadTestImage(t, studentToken)
	secondURL := api.uploadTestImage(t, studentToken)
	api.request(t, http.MethodPost, "/v1/orders/"+orderID+"/complete", studentToken, map[string]any{
		"photos": []string{firstURL, secondURL},
	}, http.StatusOK)
	api.request(t, http.MethodPost, "/v1/orders/"+orderID+"/confirm", ownerToken, map[string]any{}, http.StatusOK)
	api.request(t, http.MethodPost, "/v1/orders/"+orderID+"/rate", ownerToken, map[string]any{"stars": 5}, http.StatusOK)

	post := api.request(t, http.MethodPost, "/v1/community/posts", ownerToken, map[string]any{
		"content": "今天豆豆顺利完成上门喂养，感谢平台。", "images": []string{firstURL},
	}, http.StatusCreated)
	if post["data"].(map[string]any)["content"] == "" {
		t.Fatal("community post missing content")
	}
	feed := api.request(t, http.MethodGet, "/v1/community/posts?limit=20", studentToken, nil, http.StatusOK)
	items := feed["data"].(map[string]any)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("community feed length = %d, want 1", len(items))
	}
}

func TestAdminStudentVerificationReview(t *testing.T) {
	api := newTestAPI(t)
	studentToken, studentID := api.loginWeChat(t, "wx_student_verification", "认证学生")
	submit := map[string]any{
		"realName": "认证学生", "school": "中国农业大学", "studentNo": "20240041",
		"major": "动物医学", "enrollmentYear": 2024, "chsiCode": "HTTPVERIFY2026A",
	}
	api.request(t, http.MethodPost, "/v1/student/verification", studentToken, submit, http.StatusOK)

	list := api.adminRequest(t, http.MethodGet, "/v1/admin/student-verifications?status=PENDING", nil, http.StatusOK)
	applications := list["data"].([]any)
	if len(applications) != 1 {
		t.Fatalf("pending applications = %+v", applications)
	}
	applicationID := applications[0].(map[string]any)["id"].(string)
	api.adminRequest(t, http.MethodPost, "/v1/admin/student-verifications/"+applicationID+"/reject", map[string]any{
		"reason": "验证码已失效，请重新申请学籍报告",
	}, http.StatusOK)
	profile := api.request(t, http.MethodGet, "/v1/student/profile", studentToken, nil, http.StatusOK)
	if profile["data"].(map[string]any)["verificationStatus"] != domain.StudentRejected {
		t.Fatalf("rejected student profile = %+v", profile)
	}

	submit["chsiCode"] = "HTTPVERIFY2026B"
	api.request(t, http.MethodPost, "/v1/student/verification", studentToken, submit, http.StatusOK)
	list = api.adminRequest(t, http.MethodGet, "/v1/admin/student-verifications?status=PENDING", nil, http.StatusOK)
	applicationID = list["data"].([]any)[0].(map[string]any)["id"].(string)
	api.adminRequest(t, http.MethodPost, "/v1/admin/student-verifications/"+applicationID+"/approve", map[string]any{
		"validUntil": "2027-12-31",
	}, http.StatusOK)
	profile = api.request(t, http.MethodGet, "/v1/student/profile", studentToken, nil, http.StatusOK)
	student := profile["data"].(map[string]any)
	if student["id"] != studentID || student["verified"] != true || student["verificationStatus"] != domain.StudentVerified {
		t.Fatalf("approved student profile = %+v", student)
	}
}

func TestAdminDashboardUserControlsAndAudit(t *testing.T) {
	api := newTestAPI(t)
	loginBody := map[string]any{
		"code": "dev-code", "deviceId": "admin-user-control-device", "nickname": "风控用户",
	}
	loginResult := api.request(t, http.MethodPost, "/v1/auth/wechat/login", "", loginBody, http.StatusOK)
	loginData := loginResult["data"].(map[string]any)
	token := loginData["token"].(string)
	userID := loginData["profile"].(map[string]any)["id"].(string)

	overview := api.adminRequest(t, http.MethodGet, "/v1/admin/overview", nil, http.StatusOK)
	if overview["data"].(map[string]any)["users"].(float64) < 1 {
		t.Fatalf("admin overview = %+v", overview)
	}
	users := api.adminRequest(t, http.MethodGet, "/v1/admin/users?q=风控用户&status=ALL", nil, http.StatusOK)
	if len(users["data"].([]any)) != 1 {
		t.Fatalf("admin users = %+v", users)
	}
	api.adminRequest(t, http.MethodPost, "/v1/admin/users/"+userID+"/status", map[string]any{
		"status": domain.AccountDisabled, "reason": "风控测试停用",
	}, http.StatusOK)
	api.request(t, http.MethodGet, "/v1/auth/me", token, nil, http.StatusUnauthorized)
	api.request(t, http.MethodPost, "/v1/auth/wechat/login", "", loginBody, http.StatusForbidden)

	api.adminRequest(t, http.MethodPost, "/v1/admin/users/"+userID+"/status", map[string]any{
		"status": domain.AccountActive,
	}, http.StatusOK)
	loginResult = api.request(t, http.MethodPost, "/v1/auth/wechat/login", "", loginBody, http.StatusOK)
	token = loginResult["data"].(map[string]any)["token"].(string)
	api.request(t, http.MethodGet, "/v1/auth/me", token, nil, http.StatusOK)

	audits := api.adminRequest(t, http.MethodGet, "/v1/admin/audit-logs?limit=20", nil, http.StatusOK)
	items := audits["data"].([]any)
	if len(items) < 2 {
		t.Fatalf("admin audit logs = %+v", audits)
	}
	response, err := http.Get(api.base + "/admin/")
	if err != nil {
		t.Fatalf("GET /admin/: %v", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("运营管理平台")) {
		t.Fatalf("admin page status=%d body=%s", response.StatusCode, body)
	}
}

func TestAdminDispatchComparisonUsesRealLocatedData(t *testing.T) {
	api := newTestAPI(t)
	now := time.Now()
	_, ownerID := api.loginWeChat(t, "wx_dispatch_owner", "派单主人")
	_, firstStudentID := api.loginWeChat(t, "wx_dispatch_student_1", "就近学生")
	_, secondStudentID := api.loginWeChat(t, "wx_dispatch_student_2", "远一些学生")
	for index, studentID := range []string{firstStudentID, secondStudentID} {
		_, err := api.st.SubmitStudentVerification(studentID, domain.SubmitStudentVerificationRequest{
			RealName: "派单学生", School: "中国农业大学", StudentNo: fmt.Sprintf("20260%03d", index+1),
			Major: "动物医学", EnrollmentYear: 2024, ChsiCode: fmt.Sprintf("DISPATCH2026%02d", index+1),
		}, now)
		if err != nil {
			t.Fatalf("SubmitStudentVerification(student %d) error = %v", index, err)
		}
		if err := api.st.ApproveStudent(studentID, now); err != nil {
			t.Fatalf("ApproveStudent(student %d) error = %v", index, err)
		}
	}
	if _, err := api.st.UpdateStudentLocation(firstStudentID, domain.UpdateLocationRequest{
		Latitude: 40.0001, Longitude: 116.3270, RadiusKM: 5,
	}, now); err != nil {
		t.Fatalf("UpdateStudentLocation(first) error = %v", err)
	}
	if _, err := api.st.UpdateStudentLocation(secondStudentID, domain.UpdateLocationRequest{
		Latitude: 40.0120, Longitude: 116.3350, RadiusKM: 5,
	}, now); err != nil {
		t.Fatalf("UpdateStudentLocation(second) error = %v", err)
	}
	pet, err := api.st.CreatePet(ownerID, domain.CreatePetRequest{
		Name: "派单测试猫", Species: "猫", Breed: "狸花", Age: "2岁",
	}, now)
	if err != nil {
		t.Fatalf("CreatePet() error = %v", err)
	}
	if _, err := api.st.PublishTask(domain.PublishTaskRequest{
		OwnerID: ownerID, PetID: pet.ID, Services: []string{"喂食", "换水"},
		ServiceDate: now.AddDate(0, 0, 2).Format("2006-01-02"), TimeWindow: "09:00-10:00",
		Mode: domain.ModeBid, BudgetFen: 6000, BidDurationMin: 60,
		Community: "马连洼·朗润园附近", LocationName: "朗润园东门",
		Latitude: 39.9998, Longitude: 116.3269, AddressDetail: "3号楼2单元401",
		SafetyConfirmed: true, AgreementAccepted: true,
	}, now); err != nil {
		t.Fatalf("PublishTask() error = %v", err)
	}

	result := api.adminRequest(t, http.MethodGet, "/v1/admin/dispatch/compare", nil, http.StatusOK)
	data := result["data"].(map[string]any)
	if data["taskCount"].(float64) != 1 || data["studentCount"].(float64) != 2 ||
		data["candidateEdges"].(float64) != 2 || data["readOnly"] != true {
		t.Fatalf("dispatch comparison = %+v", data)
	}
	km := data["km"].(map[string]any)
	if km["assigned"].(float64) != 1 || len(km["assignments"].([]any)) != 1 {
		t.Fatalf("KM result = %+v", km)
	}
}

func TestRefundRequestAndAdminApproval(t *testing.T) {
	api := newTestAPI(t)
	now := time.Now()
	ownerSession, err := api.st.Login("wx_owner_refund", "退款主人", "", "", now)
	if err != nil {
		t.Fatalf("owner Login() error = %v", err)
	}
	studentSession, err := api.st.Login("wx_student_refund", "退款学生", "", "", now)
	if err != nil {
		t.Fatalf("student Login() error = %v", err)
	}
	if _, err := api.st.SubmitStudentVerification(studentSession.Profile.ID, domain.SubmitStudentVerificationRequest{
		RealName: "退款学生", School: "中国农业大学", StudentNo: "20240002",
		Major: "动物医学", EnrollmentYear: 2024, ChsiCode: "REFUND20261105A1",
	}, now); err != nil {
		t.Fatalf("SubmitStudentVerification() error = %v", err)
	}
	if err := api.st.ApproveStudent(studentSession.Profile.ID, now); err != nil {
		t.Fatalf("ApproveStudent() error = %v", err)
	}
	if _, err := api.st.UpdateStudentLocation(studentSession.Profile.ID, domain.UpdateLocationRequest{
		Latitude: 39.9998, Longitude: 116.3269, RadiusKM: 5,
	}, now); err != nil {
		t.Fatalf("UpdateStudentLocation() error = %v", err)
	}
	pet, err := api.st.CreatePet(ownerSession.Profile.ID, domain.CreatePetRequest{
		Name: "豆豆", Species: "狗", Breed: "金毛", Age: "3岁",
	}, now)
	if err != nil {
		t.Fatalf("CreatePet() error = %v", err)
	}
	task, err := api.st.PublishTask(domain.PublishTaskRequest{
		OwnerID: ownerSession.Profile.ID, PetID: pet.ID, Services: []string{"喂食", "换水"},
		ServiceDate: now.AddDate(0, 0, 2).Format("2006-01-02"), TimeWindow: "09:00-10:00",
		Mode: domain.ModeBid, BudgetFen: 6000, BidDurationMin: 60,
		Community: "马连洼·朗润园附近", LocationName: "朗润园东门",
		Latitude: 39.9998, Longitude: 116.3269, AddressDetail: "3号楼2单元401",
		SafetyConfirmed: true, AgreementAccepted: true,
	}, now)
	if err != nil {
		t.Fatalf("PublishTask() error = %v", err)
	}
	bid, err := api.st.SubmitBid(task.ID, domain.SubmitBidRequest{StudentID: studentSession.Profile.ID, PriceFen: 4500}, now)
	if err != nil {
		t.Fatalf("SubmitBid() error = %v", err)
	}
	awarded, err := api.st.AwardTask(task.ID, domain.AwardRequest{OwnerID: ownerSession.Profile.ID, BidID: bid.MyBid.ID}, now)
	if err != nil {
		t.Fatalf("AwardTask() error = %v", err)
	}
	attempt, err := api.st.PreparePayment(awarded.OrderID, ownerSession.Profile.ID, now)
	if err != nil {
		t.Fatalf("PreparePayment() error = %v", err)
	}
	if _, _, err := api.st.ApplyPaymentNotice(domain.PaymentNotice{
		OutTradeNo: attempt.OutTradeNo, TransactionID: "wx-refund-http",
		TradeState: "SUCCESS", AmountFen: 4500, SuccessAt: now,
	}, now); err != nil {
		t.Fatalf("ApplyPaymentNotice() error = %v", err)
	}

	refundResult := api.request(t, http.MethodPost, "/v1/orders/"+awarded.OrderID+"/refunds", ownerSession.Token, map[string]any{
		"reason": "临时取消",
	}, http.StatusCreated)
	refundID := refundResult["data"].(map[string]any)["id"].(string)
	approveReq, _ := http.NewRequest(http.MethodPost, api.base+"/v1/admin/refunds/"+refundID+"/approve", nil)
	approveReq.Header.Set("X-Admin-Token", "admin-test")
	approveResp, err := http.DefaultClient.Do(approveReq)
	if err != nil {
		t.Fatalf("approve refund: %v", err)
	}
	approveResp.Body.Close()
	if approveResp.StatusCode != http.StatusOK {
		t.Fatalf("approve refund status = %d", approveResp.StatusCode)
	}
	orderResult := api.request(t, http.MethodGet, "/v1/orders/"+awarded.OrderID+"?role=owner", ownerSession.Token, nil, http.StatusOK)
	orderView := orderResult["data"].(map[string]any)
	if orderView["status"] != domain.TaskCancelled {
		t.Fatalf("task status after refund = %v", orderView["status"])
	}
	refund := orderView["refund"].(map[string]any)
	if refund["status"] != domain.RefundSucceeded {
		t.Fatalf("refund status = %v", refund["status"])
	}
}

func (api *testAPI) uploadTestImage(t *testing.T, token string) string {
	t.Helper()
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Wl2nWQAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "evidence.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(png); err != nil {
		t.Fatalf("write png: %v", err)
	}
	writer.Close()
	req, err := http.NewRequest(http.MethodPost, api.base+"/v1/uploads", &body)
	if err != nil {
		t.Fatalf("upload request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload status = %d body=%s", resp.StatusCode, data)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode upload: %v", err)
	}
	url := result["data"].(map[string]any)["url"].(string)
	if url == "" {
		t.Fatal("upload returned empty URL")
	}
	return url
}

func TestHandlerRejectsBadBidJSON(t *testing.T) {
	api := newTestAPI(t)
	token, _ := login(t, api, "owner-device", http.StatusOK)
	req, _ := http.NewRequest(http.MethodPost, api.base+"/v1/tasks/missing/bids", bytes.NewBufferString(`{"priceFen":"bad"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST bid: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", resp.StatusCode, fmt.Sprint(resp.Status))
	}
}
