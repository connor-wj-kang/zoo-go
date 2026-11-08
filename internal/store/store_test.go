package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"zoo-go/internal/domain"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestStudentVerificationMigrationAddsExpiryColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			open_id TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			avatar TEXT NOT NULL DEFAULT '',
			phone_mask TEXT NOT NULL DEFAULT '',
			owner_verified INTEGER NOT NULL DEFAULT 0,
			student_verified INTEGER NOT NULL DEFAULT 0,
			student_status TEXT NOT NULL DEFAULT 'UNVERIFIED',
			school TEXT NOT NULL DEFAULT '',
			rating REAL NOT NULL DEFAULT 0,
			done_count INTEGER NOT NULL DEFAULT 0,
			service_radius_km REAL NOT NULL DEFAULT 3,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		INSERT INTO users(id, open_id, name, student_status, school, created_at, updated_at)
		VALUES('u_legacy', 'legacy-openid', '旧用户', 'PENDING', '旧学校', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO users(id, open_id, name, student_verified, student_status, school, created_at, updated_at)
		VALUES('u_legacy_verified', 'legacy-verified-openid', '旧认证用户', 1, 'VERIFIED', '旧学校', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
	`)
	if err != nil {
		db.Close()
		t.Fatalf("create legacy schema: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	st, err := New(path)
	if err != nil {
		t.Fatalf("New() migration error = %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	var columnCount int
	if err := st.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = 'verification_expires_at'`).Scan(&columnCount); err != nil {
		t.Fatalf("check migrated column: %v", err)
	}
	if columnCount != 1 {
		t.Fatalf("verification_expires_at column count = %d", columnCount)
	}
	profile, err := st.Profile("u_legacy")
	if err != nil {
		t.Fatalf("Profile() after migration error = %v", err)
	}
	if profile.StudentStatus != domain.StudentUnverified {
		t.Fatalf("legacy pending status = %s", profile.StudentStatus)
	}
	verifiedProfile, err := st.Profile("u_legacy_verified")
	if err != nil {
		t.Fatalf("Profile() legacy verified error = %v", err)
	}
	student, err := st.StudentProfileAt("u_legacy_verified", time.Now())
	if err != nil {
		t.Fatalf("StudentProfileAt() legacy verified error = %v", err)
	}
	if !verifiedProfile.StudentVerified || student.VerificationExpiresAt == nil {
		t.Fatalf("legacy verified migration = profile=%+v student=%+v", verifiedProfile, student)
	}
}

func createTestUser(t *testing.T, st *Store, name string) domain.Profile {
	t.Helper()
	session, err := st.Login("test:"+name+"-"+newID("openid"), name, "", "", time.Now())
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	return session.Profile
}

func createWeChatTestUser(t *testing.T, st *Store, name string) domain.Profile {
	t.Helper()
	session, err := st.Login("wx_"+name+"_"+newID("openid"), name, "", "", time.Now())
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	return session.Profile
}

func testChsiCode() string {
	return strings.ToUpper(strings.TrimPrefix(newID(""), "_"))
}

func createVerifiedStudent(t *testing.T, st *Store, name string) domain.Profile {
	t.Helper()
	student := createTestUser(t, st, name)
	if _, err := st.SubmitStudentVerification(student.ID, domain.SubmitStudentVerificationRequest{
		RealName: name, School: "中国农业大学", StudentNo: "20240001", Major: "动物医学",
		EnrollmentYear: 2024, ChsiCode: testChsiCode(),
	}, time.Now()); err != nil {
		t.Fatalf("SubmitStudentVerification() error = %v", err)
	}
	if err := st.ApproveStudent(student.ID, time.Now()); err != nil {
		t.Fatalf("ApproveStudent() error = %v", err)
	}
	if _, err := st.UpdateStudentLocation(student.ID, domain.UpdateLocationRequest{
		Latitude: 39.9998, Longitude: 116.3269, RadiusKM: 5,
	}, time.Now()); err != nil {
		t.Fatalf("UpdateStudentLocation() error = %v", err)
	}
	student, err := st.Profile(student.ID)
	if err != nil {
		t.Fatalf("Profile() error = %v", err)
	}
	return student
}

func TestStudentVerificationReviewAndResubmission(t *testing.T) {
	st := newTestStore(t)
	student := createTestUser(t, st, "审核学生")
	now := time.Now()
	request := domain.SubmitStudentVerificationRequest{
		RealName: "审核学生", School: "中国农业大学", StudentNo: "20240011",
		Major: "动物医学", EnrollmentYear: 2024, ChsiCode: testChsiCode(),
	}
	profile, err := st.SubmitStudentVerification(student.ID, request, now)
	if err != nil {
		t.Fatalf("SubmitStudentVerification() error = %v", err)
	}
	if profile.VerificationStatus != domain.StudentPending || profile.Verified {
		t.Fatalf("submitted profile = %+v", profile)
	}
	applications, err := st.StudentVerificationApplications(domain.StudentPending)
	if err != nil {
		t.Fatalf("StudentVerificationApplications() error = %v", err)
	}
	if len(applications) != 1 || applications[0].ChsiCode != request.ChsiCode {
		t.Fatalf("pending applications = %+v", applications)
	}
	if err := st.RejectStudentVerification(applications[0].ID, "reviewer", "验证码与学籍报告不一致", now.Add(time.Minute)); err != nil {
		t.Fatalf("RejectStudentVerification() error = %v", err)
	}
	profile, err = st.StudentProfileAt(student.ID, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("StudentProfileAt() error = %v", err)
	}
	if profile.VerificationStatus != domain.StudentRejected || profile.VerificationNote == "" {
		t.Fatalf("rejected profile = %+v", profile)
	}

	request.ChsiCode = testChsiCode()
	if _, err := st.SubmitStudentVerification(student.ID, request, now.Add(3*time.Minute)); err != nil {
		t.Fatalf("resubmit StudentVerification() error = %v", err)
	}
	applications, err = st.StudentVerificationApplications(domain.StudentPending)
	if err != nil || len(applications) != 1 {
		t.Fatalf("pending applications after resubmit = %+v, err=%v", applications, err)
	}
	validUntil := now.AddDate(1, 0, 0)
	if err := st.ApproveStudentVerification(applications[0].ID, "reviewer", validUntil, now.Add(4*time.Minute)); err != nil {
		t.Fatalf("ApproveStudentVerification() error = %v", err)
	}
	profile, err = st.StudentProfileAt(student.ID, now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("StudentProfileAt() error = %v", err)
	}
	if !profile.Verified || profile.VerificationStatus != domain.StudentVerified || profile.VerificationExpiresAt == nil {
		t.Fatalf("approved profile = %+v", profile)
	}
}

func TestStudentVerificationRejectsDuplicateCHSICode(t *testing.T) {
	st := newTestStore(t)
	first := createTestUser(t, st, "学生甲")
	second := createTestUser(t, st, "学生乙")
	request := domain.SubmitStudentVerificationRequest{
		RealName: "学生甲", School: "中国农业大学", StudentNo: "20240021",
		EnrollmentYear: 2024, ChsiCode: testChsiCode(),
	}
	if _, err := st.SubmitStudentVerification(first.ID, request, time.Now()); err != nil {
		t.Fatalf("first SubmitStudentVerification() error = %v", err)
	}
	request.RealName = "学生乙"
	request.StudentNo = "20240022"
	_, err := st.SubmitStudentVerification(second.ID, request, time.Now())
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != "STATE_CONFLICT" {
		t.Fatalf("duplicate code error = %v", err)
	}
}

func TestExpiredStudentVerificationCannotBid(t *testing.T) {
	st := newTestStore(t)
	owner := createTestUser(t, st, "任务主人")
	student := createTestUser(t, st, "过期学生")
	now := time.Now()
	request := domain.SubmitStudentVerificationRequest{
		RealName: "过期学生", School: "中国农业大学", StudentNo: "20240031",
		EnrollmentYear: 2024, ChsiCode: testChsiCode(),
	}
	if _, err := st.SubmitStudentVerification(student.ID, request, now); err != nil {
		t.Fatalf("SubmitStudentVerification() error = %v", err)
	}
	applications, err := st.StudentVerificationApplications(domain.StudentPending)
	if err != nil || len(applications) != 1 {
		t.Fatalf("pending applications = %+v, err=%v", applications, err)
	}
	if err := st.ApproveStudentVerification(applications[0].ID, "reviewer", now.Add(time.Hour), now); err != nil {
		t.Fatalf("ApproveStudentVerification() error = %v", err)
	}
	pet := createTestPet(t, st, owner.ID)
	task := publishTestTask(t, st, owner.ID, pet.ID)
	_, err = st.SubmitBid(task.ID, domain.SubmitBidRequest{
		StudentID: student.ID, PriceFen: 5000, Message: "可以按时上门",
	}, now.Add(2*time.Hour))
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != "FORBIDDEN" {
		t.Fatalf("expired bid error = %v", err)
	}
}

func createTestPet(t *testing.T, st *Store, ownerID string) domain.Pet {
	t.Helper()
	pet, err := st.CreatePet(ownerID, domain.CreatePetRequest{
		Name: "豆豆", Species: "狗", Breed: "金毛", Age: "3岁",
		Note: "每餐一勺半", Flags: []string{"温顺"},
	}, time.Now())
	if err != nil {
		t.Fatalf("CreatePet() error = %v", err)
	}
	return pet
}

func publishTestTask(t *testing.T, st *Store, ownerID, petID string) domain.TaskView {
	t.Helper()
	now := time.Now()
	view, err := st.PublishTask(domain.PublishTaskRequest{
		OwnerID: ownerID, PetID: petID, Services: []string{"喂食", "换水", "拍照反馈"},
		ServiceDate: now.AddDate(0, 0, 2).Format("2006-01-02"), TimeWindow: "09:00-10:00",
		Mode: domain.ModeBid, BudgetFen: 6000, BidDurationMin: 60,
		Community: "马连洼·朗润园附近", LocationName: "朗润园东门",
		Latitude: 39.9998, Longitude: 116.3269, AddressDetail: "3号楼2单元401",
		Note: "请按清单完成", SafetyConfirmed: true, AgreementAccepted: true,
	}, now)
	if err != nil {
		t.Fatalf("PublishTask() error = %v", err)
	}
	return view
}

func TestDatabaseStartsWithoutMockBusinessData(t *testing.T) {
	st := newTestStore(t)
	owner := createTestUser(t, st, "主人A")
	home, err := st.OwnerHome(owner.ID)
	if err != nil {
		t.Fatalf("OwnerHome() error = %v", err)
	}
	if len(home.Pets) != 0 || len(home.Tasks) != 0 || home.Community.Total != 0 {
		t.Fatalf("new database contains business data: pets=%d tasks=%d posts=%d", len(home.Pets), len(home.Tasks), home.Community.Total)
	}
	if home.Address != nil {
		t.Fatalf("new database contains address: %+v", home.Address)
	}
}

func TestStudentCannotSeeOtherBidsOrAddress(t *testing.T) {
	st := newTestStore(t)
	owner := createTestUser(t, st, "主人A")
	studentA := createVerifiedStudent(t, st, "学生A")
	studentB := createVerifiedStudent(t, st, "学生B")
	pet := createTestPet(t, st, owner.ID)
	task := publishTestTask(t, st, owner.ID, pet.ID)

	if _, err := st.SubmitBid(task.ID, domain.SubmitBidRequest{StudentID: studentA.ID, PriceFen: 4500, Message: "可以准时到"}, time.Now()); err != nil {
		t.Fatalf("SubmitBid(A) error = %v", err)
	}
	if _, err := st.SubmitBid(task.ID, domain.SubmitBidRequest{StudentID: studentB.ID, PriceFen: 4200, Message: "本小区"}, time.Now()); err != nil {
		t.Fatalf("SubmitBid(B) error = %v", err)
	}
	view, err := st.Task(task.ID, studentA.ID, "student")
	if err != nil {
		t.Fatalf("Task() error = %v", err)
	}
	if len(view.Bids) != 0 {
		t.Fatalf("student saw %d other bids", len(view.Bids))
	}
	if view.AddressDetail != "" {
		t.Fatalf("student saw detailed address %q", view.AddressDetail)
	}
	if view.MyBid == nil || view.MyBid.PriceFen != 4500 {
		t.Fatalf("MyBid = %+v, want price 4500", view.MyBid)
	}
	if view.BidCount != 2 {
		t.Fatalf("BidCount = %d, want 2", view.BidCount)
	}
	ownerView, err := st.Task(task.ID, owner.ID, "owner")
	if err != nil {
		t.Fatalf("owner Task() error = %v", err)
	}
	if len(ownerView.Bids) != 2 || ownerView.AddressDetail == "" {
		t.Fatalf("owner view missing bids or address: %+v", ownerView)
	}
}

func TestNearbyTasksAndBidRadiusEnforcement(t *testing.T) {
	st := newTestStore(t)
	owner := createTestUser(t, st, "附近任务主人")
	student := createVerifiedStudent(t, st, "附近学生")
	pet := createTestPet(t, st, owner.ID)
	task := publishTestTask(t, st, owner.ID, pet.ID)

	if _, err := st.UpdateStudentLocation(student.ID, domain.UpdateLocationRequest{
		Latitude: 40.0020, Longitude: 116.3271, RadiusKM: 5,
	}, time.Now()); err != nil {
		t.Fatalf("UpdateStudentLocation() error = %v", err)
	}
	tasks, err := st.AvailableTasks(student.ID)
	if err != nil {
		t.Fatalf("AvailableTasks() error = %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != task.ID {
		t.Fatalf("nearby tasks = %+v", tasks)
	}
	if tasks[0].DistanceKM <= 0 || tasks[0].DistanceKM > 5 {
		t.Fatalf("distance = %.2f, want within (0, 5]", tasks[0].DistanceKM)
	}
	if tasks[0].LocationName != "" || tasks[0].AddressDetail != "" {
		t.Fatalf("student saw private location before award: %+v", tasks[0])
	}

	if _, err := st.UpdateStudentLocation(student.ID, domain.UpdateLocationRequest{
		Latitude: 40.1, Longitude: 116.4, RadiusKM: 5,
	}, time.Now()); err != nil {
		t.Fatalf("move student error = %v", err)
	}
	tasks, err = st.AvailableTasks(student.ID)
	if err != nil {
		t.Fatalf("AvailableTasks(far) error = %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("far tasks should be hidden: %+v", tasks)
	}
	_, err = st.Task(task.ID, student.ID, "student")
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != "FORBIDDEN" {
		t.Fatalf("far task detail error = %v", err)
	}
	_, err = st.SubmitBid(task.ID, domain.SubmitBidRequest{StudentID: student.ID, PriceFen: 4500}, time.Now())
	if !errors.As(err, &appErr) || appErr.Code != "FORBIDDEN" {
		t.Fatalf("far bid error = %v", err)
	}
}

func TestStudentLocationValidation(t *testing.T) {
	st := newTestStore(t)
	student := createTestUser(t, st, "定位学生")
	_, err := st.UpdateStudentLocation(student.ID, domain.UpdateLocationRequest{
		Latitude: 39.9998, Longitude: 116.3269, RadiusKM: 6,
	}, time.Now())
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Fatalf("radius validation error = %v", err)
	}
	_, err = st.AvailableTasks(student.ID)
	if !errors.As(err, &appErr) || appErr.Code != "LOCATION_REQUIRED" {
		t.Fatalf("available without location error = %v", err)
	}
}

func TestConcurrentAwardOnlyOneWinner(t *testing.T) {
	st := newTestStore(t)
	owner := createTestUser(t, st, "主人A")
	student := createVerifiedStudent(t, st, "学生A")
	pet := createTestPet(t, st, owner.ID)
	task := publishTestTask(t, st, owner.ID, pet.ID)
	bidView, err := st.SubmitBid(task.ID, domain.SubmitBidRequest{StudentID: student.ID, PriceFen: 4500}, time.Now())
	if err != nil {
		t.Fatalf("SubmitBid() error = %v", err)
	}
	bidID := bidView.MyBid.ID
	const attempts = 20
	var wg sync.WaitGroup
	results := make(chan error, attempts)
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := st.AwardTask(task.ID, domain.AwardRequest{OwnerID: owner.ID, BidID: bidID}, time.Now())
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("successful awards = %d, want 1", success)
	}
	view, err := st.Task(task.ID, owner.ID, "owner")
	if err != nil {
		t.Fatalf("Task() error = %v", err)
	}
	if view.Status != domain.TaskPendingPayment || view.OrderID == "" {
		t.Fatalf("unexpected awarded task: status=%s order=%s", view.Status, view.OrderID)
	}
}

func TestPaymentAndFulfillmentLifecycle(t *testing.T) {
	st := newTestStore(t)
	owner := createWeChatTestUser(t, st, "主人A")
	student := createVerifiedStudent(t, st, "学生A")
	pet := createTestPet(t, st, owner.ID)
	task := publishTestTask(t, st, owner.ID, pet.ID)
	bidView, err := st.SubmitBid(task.ID, domain.SubmitBidRequest{StudentID: student.ID, PriceFen: 4500}, time.Now())
	if err != nil {
		t.Fatalf("SubmitBid() error = %v", err)
	}
	awarded, err := st.AwardTask(task.ID, domain.AwardRequest{OwnerID: owner.ID, BidID: bidView.MyBid.ID}, time.Now())
	if err != nil {
		t.Fatalf("AwardTask() error = %v", err)
	}
	attempt, err := st.PreparePayment(awarded.OrderID, owner.ID, time.Now())
	if err != nil {
		t.Fatalf("PreparePayment() error = %v", err)
	}
	notice := domain.PaymentNotice{
		OutTradeNo: attempt.OutTradeNo, TransactionID: "wx-txn-payment-lifecycle",
		TradeState: "SUCCESS", AmountFen: 4500, SuccessAt: time.Now(),
	}
	for range 8 {
		paid, late, err := st.ApplyPaymentNotice(notice, time.Now())
		if err != nil {
			t.Fatalf("ApplyPaymentNotice() error = %v", err)
		}
		if late || paid.Status != domain.TaskBooked || paid.Order == nil || paid.Order.Status != domain.OrderPaid {
			t.Fatalf("unexpected paid view: %+v", paid)
		}
	}
	if _, err := st.CheckIn(awarded.OrderID, student.ID, time.Now()); err != nil {
		t.Fatalf("CheckIn() error = %v", err)
	}
	if _, err := st.Complete(awarded.OrderID, domain.CompleteRequest{StudentID: student.ID, Photos: []string{"/uploads/one.jpg"}}, time.Now()); err == nil {
		t.Fatal("Complete() with one photo succeeded, want validation error")
	}
	if _, err := st.Complete(awarded.OrderID, domain.CompleteRequest{StudentID: student.ID, Photos: []string{"/tmp/one.jpg", "/tmp/two.jpg"}}, time.Now()); err == nil {
		t.Fatal("Complete() with non-platform photos succeeded, want validation error")
	}
	completed, err := st.Complete(awarded.OrderID, domain.CompleteRequest{
		StudentID: student.ID, Photos: []string{"/uploads/one.jpg", "/uploads/two.jpg"},
	}, time.Now())
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if completed.Status != domain.TaskPendingConfirm {
		t.Fatalf("status = %s, want %s", completed.Status, domain.TaskPendingConfirm)
	}
	confirmed, err := st.Confirm(awarded.OrderID, owner.ID, time.Now())
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	if confirmed.Status != domain.TaskCompleted {
		t.Fatalf("status = %s, want %s", confirmed.Status, domain.TaskCompleted)
	}
	rated, err := st.Rate(awarded.OrderID, domain.RateRequest{OwnerID: owner.ID, Stars: 5})
	if err != nil {
		t.Fatalf("Rate() error = %v", err)
	}
	if !rated.Rated {
		t.Fatal("task was not marked rated")
	}
	earnings, err := st.StudentEarnings(student.ID)
	if err != nil {
		t.Fatalf("StudentEarnings() error = %v", err)
	}
	if earnings.AvailableFen != 4050 || earnings.CompletedOrders != 1 {
		t.Fatalf("unexpected earnings: %+v", earnings)
	}
}

func TestConcurrentAwardsForSameStudentSlotHaveSingleWinner(t *testing.T) {
	st := newTestStore(t)
	ownerA := createTestUser(t, st, "主人A")
	ownerB := createTestUser(t, st, "主人B")
	student := createVerifiedStudent(t, st, "共享时段学生")
	petA := createTestPet(t, st, ownerA.ID)
	petB := createTestPet(t, st, ownerB.ID)
	taskA := publishTestTask(t, st, ownerA.ID, petA.ID)
	taskB := publishTestTask(t, st, ownerB.ID, petB.ID)
	bidA, err := st.SubmitBid(taskA.ID, domain.SubmitBidRequest{StudentID: student.ID, PriceFen: 4500}, time.Now())
	if err != nil {
		t.Fatalf("SubmitBid(A) error = %v", err)
	}
	bidB, err := st.SubmitBid(taskB.ID, domain.SubmitBidRequest{StudentID: student.ID, PriceFen: 4600}, time.Now())
	if err != nil {
		t.Fatalf("SubmitBid(B) error = %v", err)
	}

	type award struct {
		taskID  string
		ownerID string
		bidID   string
	}
	awards := []award{
		{taskID: taskA.ID, ownerID: ownerA.ID, bidID: bidA.MyBid.ID},
		{taskID: taskB.ID, ownerID: ownerB.ID, bidID: bidB.MyBid.ID},
	}
	results := make(chan error, len(awards))
	var wg sync.WaitGroup
	for _, item := range awards {
		wg.Add(1)
		go func(item award) {
			defer wg.Done()
			_, err := st.AwardTask(item.taskID, domain.AwardRequest{OwnerID: item.ownerID, BidID: item.bidID}, time.Now())
			results <- err
		}(item)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("successful awards = %d, want 1", success)
	}
}

func TestPendingPaymentBlocksSlotUntilExpiry(t *testing.T) {
	st := newTestStore(t)
	ownerA := createTestUser(t, st, "主人A")
	ownerB := createTestUser(t, st, "主人B")
	student := createVerifiedStudent(t, st, "共享时段学生")
	petA := createTestPet(t, st, ownerA.ID)
	petB := createTestPet(t, st, ownerB.ID)
	taskA := publishTestTask(t, st, ownerA.ID, petA.ID)
	taskB := publishTestTask(t, st, ownerB.ID, petB.ID)
	bidA, err := st.SubmitBid(taskA.ID, domain.SubmitBidRequest{StudentID: student.ID, PriceFen: 4500}, time.Now())
	if err != nil {
		t.Fatalf("SubmitBid(A) error = %v", err)
	}
	bidB, err := st.SubmitBid(taskB.ID, domain.SubmitBidRequest{StudentID: student.ID, PriceFen: 4600}, time.Now())
	if err != nil {
		t.Fatalf("SubmitBid(B) error = %v", err)
	}
	awarded, err := st.AwardTask(taskA.ID, domain.AwardRequest{OwnerID: ownerA.ID, BidID: bidA.MyBid.ID}, time.Now())
	if err != nil {
		t.Fatalf("AwardTask(A) error = %v", err)
	}
	if _, err := st.AwardTask(taskB.ID, domain.AwardRequest{OwnerID: ownerB.ID, BidID: bidB.MyBid.ID}, time.Now()); err == nil {
		t.Fatal("AwardTask(B) succeeded while student slot was occupied")
	}
	if _, err := st.PreparePayment(awarded.OrderID, ownerA.ID, time.Now()); err != nil {
		t.Fatalf("PreparePayment() error = %v", err)
	}
	expireAt := awarded.PayDeadlineAt.Add(time.Second)
	expired, err := st.ExpirePendingPayment(awarded.OrderID, expireAt)
	if err != nil {
		t.Fatalf("ExpirePendingPayment() error = %v", err)
	}
	if !expired {
		t.Fatal("ExpirePendingPayment() = false, want true")
	}
	if _, err := st.AwardTask(taskB.ID, domain.AwardRequest{OwnerID: ownerB.ID, BidID: bidB.MyBid.ID}, expireAt); err != nil {
		t.Fatalf("AwardTask(B) after expiry error = %v", err)
	}
}

func TestLatePaymentBecomesRefundInsteadOfBooking(t *testing.T) {
	st := newTestStore(t)
	owner := createWeChatTestUser(t, st, "主人A")
	student := createVerifiedStudent(t, st, "学生A")
	pet := createTestPet(t, st, owner.ID)
	task := publishTestTask(t, st, owner.ID, pet.ID)
	bid, err := st.SubmitBid(task.ID, domain.SubmitBidRequest{StudentID: student.ID, PriceFen: 4500}, time.Now())
	if err != nil {
		t.Fatalf("SubmitBid() error = %v", err)
	}
	awarded, err := st.AwardTask(task.ID, domain.AwardRequest{OwnerID: owner.ID, BidID: bid.MyBid.ID}, time.Now())
	if err != nil {
		t.Fatalf("AwardTask() error = %v", err)
	}
	attempt, err := st.PreparePayment(awarded.OrderID, owner.ID, time.Now())
	if err != nil {
		t.Fatalf("PreparePayment() error = %v", err)
	}
	lateAt := awarded.PayDeadlineAt.Add(time.Minute)
	notice := domain.PaymentNotice{
		OutTradeNo: attempt.OutTradeNo, TransactionID: "wx-late-transaction",
		TradeState: "SUCCESS", AmountFen: 4500, SuccessAt: lateAt,
	}
	view, late, err := st.ApplyPaymentNotice(notice, lateAt)
	if err != nil {
		t.Fatalf("ApplyPaymentNotice() error = %v", err)
	}
	if !late || view.Status != domain.TaskRefunding || view.Order == nil || view.Order.PaymentStatus != domain.PaymentPaidLate {
		t.Fatalf("late payment did not enter refunding: late=%v view=%+v", late, view)
	}
	if view.Refund == nil || view.Refund.Status != domain.RefundRequested {
		t.Fatalf("late payment refund = %+v", view.Refund)
	}
	if _, late, err := st.ApplyPaymentNotice(notice, lateAt.Add(time.Second)); err != nil || !late {
		t.Fatalf("duplicate late callback: late=%v err=%v", late, err)
	}
	if _, _, err := st.StartRefund(view.Refund.ID, "admin", lateAt); err != nil {
		t.Fatalf("StartRefund() error = %v", err)
	}
	refund, err := st.ApplyRefundResult(view.Refund.ID, "wx-refund-late", domain.RefundSucceeded, 4500, lateAt, lateAt)
	if err != nil {
		t.Fatalf("ApplyRefundResult() error = %v", err)
	}
	if refund.Status != domain.RefundSucceeded {
		t.Fatalf("refund status = %s", refund.Status)
	}
	again, err := st.ApplyRefundResult(view.Refund.ID, "wx-refund-late", domain.RefundSucceeded, 4500, lateAt, lateAt.Add(time.Second))
	if err != nil || again.Status != domain.RefundSucceeded {
		t.Fatalf("duplicate successful refund callback: refund=%+v err=%v", again, err)
	}
	order, err := st.Order(awarded.OrderID, owner.ID, "owner")
	if err != nil {
		t.Fatalf("Order() error = %v", err)
	}
	if order.Status != domain.TaskCancelled || order.Order.Status != domain.OrderRefunded {
		t.Fatalf("late refund final state = %+v", order)
	}
}

func TestLatePaymentForOldOrderDoesNotConflictWithNewAward(t *testing.T) {
	st := newTestStore(t)
	owner := createWeChatTestUser(t, st, "主人A")
	student := createVerifiedStudent(t, st, "学生A")
	pet := createTestPet(t, st, owner.ID)
	task := publishTestTask(t, st, owner.ID, pet.ID)
	bid, err := st.SubmitBid(task.ID, domain.SubmitBidRequest{StudentID: student.ID, PriceFen: 4500}, time.Now())
	if err != nil {
		t.Fatalf("SubmitBid() error = %v", err)
	}
	first, err := st.AwardTask(task.ID, domain.AwardRequest{OwnerID: owner.ID, BidID: bid.MyBid.ID}, time.Now())
	if err != nil {
		t.Fatalf("first AwardTask() error = %v", err)
	}
	oldPayment, err := st.PreparePayment(first.OrderID, owner.ID, time.Now())
	if err != nil {
		t.Fatalf("PreparePayment() error = %v", err)
	}
	expireAt := first.PayDeadlineAt.Add(time.Second)
	expired, err := st.ExpirePendingPayment(first.OrderID, expireAt)
	if err != nil || !expired {
		t.Fatalf("ExpirePendingPayment() expired=%v err=%v", expired, err)
	}
	second, err := st.AwardTask(task.ID, domain.AwardRequest{OwnerID: owner.ID, BidID: bid.MyBid.ID}, expireAt)
	if err != nil {
		t.Fatalf("second AwardTask() error = %v", err)
	}
	if second.OrderID == first.OrderID {
		t.Fatal("re-award reused the cancelled order")
	}
	lateAt := expireAt.Add(time.Minute)
	view, late, err := st.ApplyPaymentNotice(domain.PaymentNotice{
		OutTradeNo: oldPayment.OutTradeNo, TransactionID: "wx-late-old-order",
		TradeState: "SUCCESS", AmountFen: 4500, SuccessAt: lateAt,
	}, lateAt)
	if err != nil {
		t.Fatalf("ApplyPaymentNotice(old) error = %v", err)
	}
	if !late || view.Refund == nil || view.Refund.Status != domain.RefundRequested {
		t.Fatalf("old late payment refund state = %+v", view)
	}
	var oldOrderStatus, newOrderStatus string
	if err := st.db.QueryRow(`SELECT status FROM orders WHERE id = ?`, first.OrderID).Scan(&oldOrderStatus); err != nil {
		t.Fatalf("query old order: %v", err)
	}
	if err := st.db.QueryRow(`SELECT status FROM orders WHERE id = ?`, second.OrderID).Scan(&newOrderStatus); err != nil {
		t.Fatalf("query new order: %v", err)
	}
	if oldOrderStatus != domain.OrderRefunding || newOrderStatus != domain.OrderUnpaid {
		t.Fatalf("order statuses after old late payment: old=%s new=%s", oldOrderStatus, newOrderStatus)
	}
}

func TestPaymentRejectsWrongAmountAndDuplicateTransaction(t *testing.T) {
	st := newTestStore(t)
	owner := createWeChatTestUser(t, st, "主人A")
	student := createVerifiedStudent(t, st, "学生A")
	pet := createTestPet(t, st, owner.ID)
	task := publishTestTask(t, st, owner.ID, pet.ID)
	bid, err := st.SubmitBid(task.ID, domain.SubmitBidRequest{StudentID: student.ID, PriceFen: 4500}, time.Now())
	if err != nil {
		t.Fatalf("SubmitBid() error = %v", err)
	}
	awarded, err := st.AwardTask(task.ID, domain.AwardRequest{OwnerID: owner.ID, BidID: bid.MyBid.ID}, time.Now())
	if err != nil {
		t.Fatalf("AwardTask() error = %v", err)
	}
	attempt, err := st.PreparePayment(awarded.OrderID, owner.ID, time.Now())
	if err != nil {
		t.Fatalf("PreparePayment() error = %v", err)
	}
	wrongAmount := domain.PaymentNotice{
		OutTradeNo: attempt.OutTradeNo, TransactionID: "wx-wrong-amount",
		TradeState: "SUCCESS", AmountFen: 1, SuccessAt: time.Now(),
	}
	if _, _, err := st.ApplyPaymentNotice(wrongAmount, time.Now()); err == nil {
		t.Fatal("ApplyPaymentNotice() accepted a wrong amount")
	}
	valid := wrongAmount
	valid.AmountFen = 4500
	if _, _, err := st.ApplyPaymentNotice(valid, time.Now()); err != nil {
		t.Fatalf("ApplyPaymentNotice() error = %v", err)
	}
	conflict := valid
	conflict.TransactionID = "wx-conflicting-transaction"
	if _, _, err := st.ApplyPaymentNotice(conflict, time.Now()); err == nil {
		t.Fatal("ApplyPaymentNotice() accepted a different transaction id for a paid order")
	}
}

func TestOwnerRefundRequestAndFailureRestoresBooking(t *testing.T) {
	st := newTestStore(t)
	owner := createWeChatTestUser(t, st, "主人A")
	student := createVerifiedStudent(t, st, "学生A")
	pet := createTestPet(t, st, owner.ID)
	task := publishTestTask(t, st, owner.ID, pet.ID)
	bid, err := st.SubmitBid(task.ID, domain.SubmitBidRequest{StudentID: student.ID, PriceFen: 4500}, time.Now())
	if err != nil {
		t.Fatalf("SubmitBid() error = %v", err)
	}
	awarded, err := st.AwardTask(task.ID, domain.AwardRequest{OwnerID: owner.ID, BidID: bid.MyBid.ID}, time.Now())
	if err != nil {
		t.Fatalf("AwardTask() error = %v", err)
	}
	attempt, err := st.PreparePayment(awarded.OrderID, owner.ID, time.Now())
	if err != nil {
		t.Fatalf("PreparePayment() error = %v", err)
	}
	if _, _, err := st.ApplyPaymentNotice(domain.PaymentNotice{
		OutTradeNo: attempt.OutTradeNo, TransactionID: "wx-refund-normal",
		TradeState: "SUCCESS", AmountFen: 4500, SuccessAt: time.Now(),
	}, time.Now()); err != nil {
		t.Fatalf("ApplyPaymentNotice() error = %v", err)
	}
	first, err := st.RequestRefund(awarded.OrderID, domain.RefundRequest{OwnerID: owner.ID, Reason: "临时取消"}, time.Now())
	if err != nil {
		t.Fatalf("RequestRefund() error = %v", err)
	}
	second, err := st.RequestRefund(awarded.OrderID, domain.RefundRequest{OwnerID: owner.ID, Reason: "临时取消"}, time.Now())
	if err != nil {
		t.Fatalf("duplicate RequestRefund() error = %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("duplicate refund id = %s, want %s", second.ID, first.ID)
	}
	if _, _, err := st.StartRefund(first.ID, "admin", time.Now()); err != nil {
		t.Fatalf("StartRefund() error = %v", err)
	}
	if err := st.FailRefund(first.ID, "测试退款失败", time.Now()); err != nil {
		t.Fatalf("FailRefund() error = %v", err)
	}
	view, err := st.Order(awarded.OrderID, owner.ID, "owner")
	if err != nil {
		t.Fatalf("Order() error = %v", err)
	}
	if view.Status != domain.TaskBooked || view.Order.Status != domain.OrderPaid || view.Refund == nil || view.Refund.Status != domain.RefundFailed {
		t.Fatalf("failed refund state = %+v", view)
	}
}

func TestBidValidation(t *testing.T) {
	st := newTestStore(t)
	owner := createTestUser(t, st, "主人A")
	student := createVerifiedStudent(t, st, "学生A")
	pet := createTestPet(t, st, owner.ID)
	task := publishTestTask(t, st, owner.ID, pet.ID)
	cases := []struct {
		name string
		req  domain.SubmitBidRequest
	}{
		{name: "over budget", req: domain.SubmitBidRequest{StudentID: student.ID, PriceFen: 6001}},
		{name: "below floor", req: domain.SubmitBidRequest{StudentID: student.ID, PriceFen: 1999}},
		{name: "unverified", req: domain.SubmitBidRequest{StudentID: createTestUser(t, st, "未认证学生").ID, PriceFen: 4000}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := st.SubmitBid(task.ID, tc.req, time.Now())
			if err == nil {
				t.Fatal("SubmitBid() error = nil, want error")
			}
			var appErr *AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("error type = %T, want *AppError", err)
			}
		})
	}
}

func TestCommunityPostsPersistAndPaginate(t *testing.T) {
	st := newTestStore(t)
	owner := createTestUser(t, st, "主人A")
	for i := 1; i <= 3; i++ {
		if _, err := st.CreateCommunityPost(owner.ID, domain.CreatePostRequest{
			Content: fmt.Sprintf("社区动态 %d", i),
		}, time.Now().Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("CreateCommunityPost() error = %v", err)
		}
	}
	first, err := st.CommunityPosts(owner.ID, "", 2)
	if err != nil {
		t.Fatalf("CommunityPosts() error = %v", err)
	}
	if len(first.Items) != 2 || !first.HasMore || first.NextCursor == "" {
		t.Fatalf("unexpected first page: %+v", first)
	}
	second, err := st.CommunityPosts(owner.ID, first.NextCursor, 2)
	if err != nil {
		t.Fatalf("CommunityPosts(second) error = %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].Content != "社区动态 1" {
		t.Fatalf("unexpected second page: %+v", second)
	}
	if _, err := st.CreateCommunityPost(owner.ID, domain.CreatePostRequest{Images: []string{"https://evil.example/a.jpg"}}, time.Now()); err == nil {
		t.Fatal("external image URL was accepted")
	}
}
