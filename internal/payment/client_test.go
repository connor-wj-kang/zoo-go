package payment

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"zoo-go/internal/domain"
)

func TestCreateJSAPIRequestAndResponseSignatures(t *testing.T) {
	merchantKey := generateRSAKey(t)
	platformKey := generateRSAKey(t)
	apiKey := "01234567890123456789012345678901"
	now := time.Unix(1_800_000_000, 0)
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		verifyAuthorization(t, r.Header.Get("Authorization"), &merchantKey.PublicKey, http.MethodPost, requestPath, body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload["mchid"] != "1900000001" || payload["out_trade_no"] != "zoo_order_1" {
			t.Fatalf("unexpected request body: %+v", payload)
		}
		responseBody := []byte(`{"prepay_id":"wx-prepay-123"}`)
		writeSignedResponse(t, w, platformKey, "PUB_KEY_ID_TEST", responseBody)
	}))
	t.Cleanup(server.Close)
	client := newTestClient(t, merchantKey, platformKey, apiKey, server.URL, now)

	params, err := client.CreateJSAPI(context.Background(), domain.PaymentAttempt{
		ID: "pay_1", OrderID: "order_1", OutTradeNo: "zoo_order_1", Description: "上门喂宠服务",
		PayerOpenID: "wx_openid", AmountFen: 4500, ExpireAt: now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateJSAPI() error = %v", err)
	}
	if requestPath != "/v3/pay/transactions/jsapi" {
		t.Fatalf("request path = %s", requestPath)
	}
	if params.Package != "prepay_id=wx-prepay-123" || params.SignType != "RSA" {
		t.Fatalf("unexpected payment params: %+v", params)
	}
	message := "wx-app-id\n" + params.TimeStamp + "\n" + params.NonceStr + "\n" + params.Package + "\n"
	verifySignature(t, &merchantKey.PublicKey, message, params.PaySign)
}

func TestParsePaymentNotificationDecryptsAndVerifies(t *testing.T) {
	merchantKey := generateRSAKey(t)
	platformKey := generateRSAKey(t)
	apiKey := "01234567890123456789012345678901"
	now := time.Unix(1_800_000_000, 0)
	client := newTestClient(t, merchantKey, platformKey, apiKey, "https://example.invalid", now)

	plaintext := []byte(`{
		"appid":"wx-app-id","mchid":"1900000001","out_trade_no":"zoo_order_1",
		"transaction_id":"4200000001","trade_state":"SUCCESS","success_time":"2027-01-15T08:00:00+08:00",
		"amount":{"total":4500}
	}`)
	nonce := "0123456789ab"
	associatedData := "transaction"
	ciphertext := encryptNotification(t, []byte(apiKey), nonce, plaintext, associatedData)
	envelope, _ := json.Marshal(map[string]any{
		"id": "notify-1", "event_type": "TRANSACTION.SUCCESS",
		"resource": map[string]any{
			"algorithm": "AEAD_AES_256_GCM", "ciphertext": ciphertext,
			"associated_data": associatedData, "nonce": nonce, "original_type": "transaction",
		},
	})
	timestamp := now.Unix()
	headerNonce := "callback-nonce"
	message := strings.Join([]string{formatInt(timestamp), headerNonce, string(envelope), ""}, "\n")
	signature := signForTest(t, platformKey, message)
	header := http.Header{}
	header.Set("Wechatpay-Timestamp", formatInt(timestamp))
	header.Set("Wechatpay-Nonce", headerNonce)
	header.Set("Wechatpay-Signature", signature)
	header.Set("Wechatpay-Serial", "PUB_KEY_ID_TEST")

	event, err := client.ParsePaymentNotification(header, envelope)
	if err != nil {
		t.Fatalf("ParsePaymentNotification() error = %v", err)
	}
	if event.EventType != "TRANSACTION.SUCCESS" || event.Notice.TransactionID != "4200000001" || event.Notice.AmountFen != 4500 {
		t.Fatalf("unexpected payment event: %+v", event)
	}
	if event.Notice.SuccessAt.IsZero() {
		t.Fatal("success time was not parsed")
	}
}

func newTestClient(t *testing.T, merchantKey, platformKey *rsa.PrivateKey, apiKey, baseURL string, now time.Time) *Client {
	t.Helper()
	dir := t.TempDir()
	merchantPath := filepath.Join(dir, "merchant.pem")
	platformPath := filepath.Join(dir, "platform.pem")
	writePrivateKey(t, merchantPath, merchantKey)
	publicDER, err := x509.MarshalPKIXPublicKey(&platformKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal platform public key: %v", err)
	}
	if err := os.WriteFile(platformPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}), 0o600); err != nil {
		t.Fatalf("write platform key: %v", err)
	}
	client, err := NewClient(Config{
		AppID: "wx-app-id", MchID: "1900000001", MerchantSerialNo: "MERCHANT_SERIAL",
		APIv3Key: apiKey, MerchantPrivateKeyPath: merchantPath, PlatformPublicKeyPath: platformPath,
		PlatformSerialNo: "PUB_KEY_ID_TEST", PaymentNotifyURL: "https://example.com/notify",
		RefundNotifyURL: "https://example.com/refund-notify", APIBaseURL: baseURL,
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return key
}

func writePrivateKey(t *testing.T, path string, key *rsa.PrivateKey) {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
}

func verifyAuthorization(t *testing.T, authorization string, publicKey *rsa.PublicKey, method, path string, body []byte) {
	t.Helper()
	authorization = strings.TrimPrefix(authorization, "WECHATPAY2-SHA256-RSA2048 ")
	fields := map[string]string{}
	for _, part := range strings.Split(authorization, ",") {
		keyValue := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(keyValue) != 2 {
			continue
		}
		fields[keyValue[0]] = strings.Trim(keyValue[1], `"`)
	}
	if fields["mchid"] != "1900000001" || fields["serial_no"] != "MERCHANT_SERIAL" {
		t.Fatalf("unexpected authorization: %s", authorization)
	}
	message := strings.Join([]string{method, path, fields["timestamp"], fields["nonce_str"], string(body), ""}, "\n")
	verifySignature(t, publicKey, message, fields["signature"])
}

func writeSignedResponse(t *testing.T, w http.ResponseWriter, platformKey *rsa.PrivateKey, serial string, body []byte) {
	t.Helper()
	timestamp := "1800000000"
	nonce := "response-nonce"
	message := strings.Join([]string{timestamp, nonce, string(body), ""}, "\n")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Wechatpay-Timestamp", timestamp)
	w.Header().Set("Wechatpay-Nonce", nonce)
	w.Header().Set("Wechatpay-Signature", signForTest(t, platformKey, message))
	w.Header().Set("Wechatpay-Serial", serial)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func signForTest(t *testing.T, key *rsa.PrivateKey, message string) string {
	t.Helper()
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return base64.StdEncoding.EncodeToString(signature)
}

func verifySignature(t *testing.T, key *rsa.PublicKey, message, signature string) {
	t.Helper()
	bytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	digest := sha256.Sum256([]byte(message))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], bytes); err != nil {
		t.Fatalf("verify signature: %v", err)
	}
}

func encryptNotification(t *testing.T, key []byte, nonce string, plaintext []byte, associatedData string) string {
	t.Helper()
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("aes cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("gcm: %v", err)
	}
	ciphertext := gcm.Seal(nil, []byte(nonce), plaintext, []byte(associatedData))
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func formatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}
