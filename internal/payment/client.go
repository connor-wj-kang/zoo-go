package payment

import (
	"bytes"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"zoo-go/internal/domain"
)

const defaultAPIBaseURL = "https://api.mch.weixin.qq.com"

type Config struct {
	AppID                  string
	MchID                  string
	MerchantSerialNo       string
	APIv3Key               string
	MerchantPrivateKeyPath string
	PlatformPublicKeyPath  string
	PlatformSerialNo       string
	PaymentNotifyURL       string
	RefundNotifyURL        string
	APIBaseURL             string
	HTTPClient             *http.Client
	Now                    func() time.Time
}

type Client struct {
	appID              string
	mchID              string
	merchantSerialNo   string
	apiV3Key           []byte
	merchantPrivateKey *rsa.PrivateKey
	platformPublicKey  *rsa.PublicKey
	platformSerialNo   string
	paymentNotifyURL   string
	refundNotifyURL    string
	baseURL            string
	httpClient         *http.Client
	now                func() time.Time
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("wechat pay %d %s: %s", e.StatusCode, e.Code, e.Message)
}

func (e *APIError) IsCode(codes ...string) bool {
	for _, code := range codes {
		if strings.EqualFold(e.Code, code) {
			return true
		}
	}
	return false
}

type PaymentEvent struct {
	EventType string
	Notice    domain.PaymentNotice
}

type RefundEvent struct {
	EventType string
	Notice    domain.RefundNotice
}

type RefundRequest struct {
	OutTradeNo  string
	OutRefundNo string
	Reason      string
	RefundFen   int64
	TotalFen    int64
}

func NewClient(config Config) (*Client, error) {
	config.AppID = strings.TrimSpace(config.AppID)
	config.MchID = strings.TrimSpace(config.MchID)
	config.MerchantSerialNo = strings.TrimSpace(config.MerchantSerialNo)
	config.PlatformSerialNo = strings.TrimSpace(config.PlatformSerialNo)
	config.PaymentNotifyURL = strings.TrimSpace(config.PaymentNotifyURL)
	config.RefundNotifyURL = strings.TrimSpace(config.RefundNotifyURL)
	required := map[string]string{
		"WECHAT_APP_ID":                       config.AppID,
		"WECHAT_MCH_ID":                       config.MchID,
		"WECHAT_PAY_SERIAL_NO":                config.MerchantSerialNo,
		"WECHAT_PAY_API_V3_KEY":               config.APIv3Key,
		"WECHAT_PAY_PRIVATE_KEY_PATH":         config.MerchantPrivateKeyPath,
		"WECHAT_PAY_PLATFORM_PUBLIC_KEY_PATH": config.PlatformPublicKeyPath,
		"WECHAT_PAY_PLATFORM_SERIAL_NO":       config.PlatformSerialNo,
		"WECHAT_PAY_NOTIFY_URL":               config.PaymentNotifyURL,
		"WECHAT_REFUND_NOTIFY_URL":            config.RefundNotifyURL,
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("missing payment configuration: %s", name)
		}
	}
	if len([]byte(config.APIv3Key)) != 32 {
		return nil, errors.New("WECHAT_PAY_API_V3_KEY must be exactly 32 bytes")
	}
	merchantKey, err := loadRSAPrivateKey(config.MerchantPrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load merchant private key: %w", err)
	}
	platformKey, err := loadRSAPublicKey(config.PlatformPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load platform public key: %w", err)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(config.APIBaseURL), "/")
	if baseURL == "" {
		baseURL = defaultAPIBaseURL
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	return &Client{
		appID: config.AppID, mchID: config.MchID, merchantSerialNo: config.MerchantSerialNo,
		apiV3Key: []byte(config.APIv3Key), merchantPrivateKey: merchantKey,
		platformPublicKey: platformKey, platformSerialNo: config.PlatformSerialNo,
		paymentNotifyURL: config.PaymentNotifyURL, refundNotifyURL: config.RefundNotifyURL,
		baseURL: baseURL, httpClient: httpClient, now: now,
	}, nil
}

func loadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("private key is not PEM encoded")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not RSA")
		}
		return rsaKey, nil
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func loadRSAPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("public key is not PEM encoded")
	}
	if block.Type == "CERTIFICATE" {
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		key, ok := certificate.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("certificate public key is not RSA")
		}
		return key, nil
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not RSA")
		}
		return rsaKey, nil
	}
	return x509.ParsePKCS1PublicKey(block.Bytes)
}

func (c *Client) CreateJSAPI(ctx context.Context, attempt domain.PaymentAttempt) (domain.PaymentParams, error) {
	if attempt.AmountFen <= 0 {
		return domain.PaymentParams{}, errors.New("payment amount must be positive")
	}
	body := map[string]any{
		"appid":        c.appID,
		"mchid":        c.mchID,
		"description":  attempt.Description,
		"out_trade_no": attempt.OutTradeNo,
		"time_expire":  attempt.ExpireAt.In(time.FixedZone("CST", 8*60*60)).Format(time.RFC3339),
		"notify_url":   c.paymentNotifyURL,
		"amount":       map[string]any{"total": attempt.AmountFen, "currency": "CNY"},
		"payer":        map[string]any{"openid": attempt.PayerOpenID},
	}
	var response struct {
		PrepayID string `json:"prepay_id"`
	}
	if err := c.do(ctx, http.MethodPost, "/v3/pay/transactions/jsapi", body, &response); err != nil {
		return domain.PaymentParams{}, err
	}
	if response.PrepayID == "" {
		return domain.PaymentParams{}, errors.New("wechat pay returned an empty prepay_id")
	}
	timestamp := strconv.FormatInt(c.now().Unix(), 10)
	nonce := randomNonce()
	packageValue := "prepay_id=" + response.PrepayID
	message := c.appID + "\n" + timestamp + "\n" + nonce + "\n" + packageValue + "\n"
	signature, err := sign(c.merchantPrivateKey, message)
	if err != nil {
		return domain.PaymentParams{}, err
	}
	return domain.PaymentParams{
		TimeStamp: timestamp, NonceStr: nonce, Package: packageValue, SignType: "RSA", PaySign: signature,
	}, nil
}

func (c *Client) QueryPayment(ctx context.Context, outTradeNo string) (domain.PaymentNotice, error) {
	path := "/v3/pay/transactions/out-trade-no/" + url.PathEscape(outTradeNo) + "?mchid=" + url.QueryEscape(c.mchID)
	var response struct {
		AppID         string `json:"appid"`
		MchID         string `json:"mchid"`
		OutTradeNo    string `json:"out_trade_no"`
		TransactionID string `json:"transaction_id"`
		TradeState    string `json:"trade_state"`
		SuccessTime   string `json:"success_time"`
		Amount        struct {
			Total int64 `json:"total"`
		} `json:"amount"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &response); err != nil {
		return domain.PaymentNotice{}, err
	}
	if response.AppID != c.appID || response.MchID != c.mchID {
		return domain.PaymentNotice{}, errors.New("wechat payment query returned another merchant")
	}
	if response.OutTradeNo != outTradeNo {
		return domain.PaymentNotice{}, errors.New("wechat payment query returned another out_trade_no")
	}
	return domain.PaymentNotice{
		OutTradeNo: response.OutTradeNo, TransactionID: response.TransactionID,
		TradeState: response.TradeState, AppID: response.AppID, MchID: response.MchID,
		AmountFen: response.Amount.Total, SuccessAt: parseWeChatTime(response.SuccessTime),
	}, nil
}

func (c *Client) ClosePayment(ctx context.Context, outTradeNo string) error {
	path := "/v3/pay/transactions/out-trade-no/" + url.PathEscape(outTradeNo) + "/close"
	return c.do(ctx, http.MethodPost, path, map[string]any{"mchid": c.mchID}, nil)
}

func (c *Client) CreateRefund(ctx context.Context, request RefundRequest) (domain.RefundNotice, error) {
	if request.RefundFen <= 0 || request.TotalFen <= 0 || request.RefundFen > request.TotalFen {
		return domain.RefundNotice{}, errors.New("invalid refund amount")
	}
	body := map[string]any{
		"out_trade_no":  request.OutTradeNo,
		"out_refund_no": request.OutRefundNo,
		"reason":        request.Reason,
		"notify_url":    c.refundNotifyURL,
		"amount": map[string]any{
			"refund": request.RefundFen, "total": request.TotalFen, "currency": "CNY",
		},
	}
	notice, err := c.refundRequest(ctx, http.MethodPost, "/v3/refund/domestic/refunds", body)
	if err != nil {
		return domain.RefundNotice{}, err
	}
	if notice.OutRefundNo == "" {
		notice.OutRefundNo = request.OutRefundNo
	}
	if notice.OutRefundNo != request.OutRefundNo {
		return domain.RefundNotice{}, errors.New("wechat refund returned another out_refund_no")
	}
	if notice.AmountFen == 0 {
		notice.AmountFen = request.RefundFen
	}
	return notice, nil
}

func (c *Client) QueryRefund(ctx context.Context, outRefundNo string) (domain.RefundNotice, error) {
	notice, err := c.refundRequest(ctx, http.MethodGet, "/v3/refund/domestic/refunds/"+url.PathEscape(outRefundNo), nil)
	if err != nil {
		return domain.RefundNotice{}, err
	}
	if notice.OutRefundNo != "" && notice.OutRefundNo != outRefundNo {
		return domain.RefundNotice{}, errors.New("wechat refund query returned another out_refund_no")
	}
	return notice, nil
}

func (c *Client) refundRequest(ctx context.Context, method, path string, body any) (domain.RefundNotice, error) {
	var response struct {
		RefundID    string `json:"refund_id"`
		OutRefundNo string `json:"out_refund_no"`
		Status      string `json:"status"`
		SuccessTime string `json:"success_time"`
		Amount      struct {
			Refund int64 `json:"refund"`
			Total  int64 `json:"total"`
		} `json:"amount"`
	}
	if err := c.do(ctx, method, path, body, &response); err != nil {
		return domain.RefundNotice{}, err
	}
	return domain.RefundNotice{
		OutRefundNo: response.OutRefundNo, WeChatRefundID: response.RefundID,
		Status: response.Status, AmountFen: response.Amount.Refund,
		SuccessAt: parseWeChatTime(response.SuccessTime),
	}, nil
}

func (c *Client) ParsePaymentNotification(header http.Header, body []byte) (PaymentEvent, error) {
	var envelope notificationEnvelope
	if err := c.verifyAndDecodeNotification(header, body, &envelope); err != nil {
		return PaymentEvent{}, err
	}
	var transaction struct {
		AppID         string `json:"appid"`
		MchID         string `json:"mchid"`
		OutTradeNo    string `json:"out_trade_no"`
		TransactionID string `json:"transaction_id"`
		TradeState    string `json:"trade_state"`
		SuccessTime   string `json:"success_time"`
		Amount        struct {
			Total int64 `json:"total"`
		} `json:"amount"`
	}
	if err := json.Unmarshal(envelope.Plaintext, &transaction); err != nil {
		return PaymentEvent{}, fmt.Errorf("decode payment notification: %w", err)
	}
	if transaction.AppID != c.appID || transaction.MchID != c.mchID {
		return PaymentEvent{}, errors.New("payment notification is for another merchant")
	}
	return PaymentEvent{
		EventType: envelope.EventType,
		Notice: domain.PaymentNotice{
			OutTradeNo: transaction.OutTradeNo, TransactionID: transaction.TransactionID,
			TradeState: transaction.TradeState, AppID: transaction.AppID, MchID: transaction.MchID,
			AmountFen: transaction.Amount.Total, SuccessAt: parseWeChatTime(transaction.SuccessTime),
		},
	}, nil
}

func (c *Client) ParseRefundNotification(header http.Header, body []byte) (RefundEvent, error) {
	var envelope notificationEnvelope
	if err := c.verifyAndDecodeNotification(header, body, &envelope); err != nil {
		return RefundEvent{}, err
	}
	var refund struct {
		MchID        string `json:"mchid"`
		OutRefundNo  string `json:"out_refund_no"`
		RefundID     string `json:"refund_id"`
		RefundStatus string `json:"refund_status"`
		SuccessTime  string `json:"success_time"`
		Amount       struct {
			Refund int64 `json:"refund"`
		} `json:"amount"`
	}
	if err := json.Unmarshal(envelope.Plaintext, &refund); err != nil {
		return RefundEvent{}, fmt.Errorf("decode refund notification: %w", err)
	}
	if refund.MchID != "" && refund.MchID != c.mchID {
		return RefundEvent{}, errors.New("refund notification is for another merchant")
	}
	return RefundEvent{
		EventType: envelope.EventType,
		Notice: domain.RefundNotice{
			OutRefundNo: refund.OutRefundNo, WeChatRefundID: refund.RefundID,
			Status: refund.RefundStatus, AmountFen: refund.Amount.Refund,
			SuccessAt: parseWeChatTime(refund.SuccessTime),
		},
	}, nil
}

type notificationEnvelope struct {
	EventType string `json:"event_type"`
	Resource  struct {
		Algorithm      string `json:"algorithm"`
		Ciphertext     string `json:"ciphertext"`
		AssociatedData string `json:"associated_data"`
		Nonce          string `json:"nonce"`
	} `json:"resource"`
	Plaintext []byte `json:"-"`
}

func (c *Client) verifyAndDecodeNotification(header http.Header, body []byte, envelope *notificationEnvelope) error {
	timestamp := strings.TrimSpace(header.Get("Wechatpay-Timestamp"))
	nonce := strings.TrimSpace(header.Get("Wechatpay-Nonce"))
	signature := strings.TrimSpace(header.Get("Wechatpay-Signature"))
	serial := strings.TrimSpace(header.Get("Wechatpay-Serial"))
	if timestamp == "" || nonce == "" || signature == "" || serial == "" {
		return errors.New("missing wechat pay signature headers")
	}
	if !strings.EqualFold(serial, c.platformSerialNo) {
		return fmt.Errorf("unexpected wechat pay platform serial: %s", serial)
	}
	unix, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("invalid wechat pay timestamp")
	}
	if delta := c.now().Sub(time.Unix(unix, 0)); delta < -5*time.Minute || delta > 5*time.Minute {
		return errors.New("wechat pay notification timestamp is outside the allowed window")
	}
	message := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	if err := verify(c.platformPublicKey, message, signature); err != nil {
		return errors.New("invalid wechat pay notification signature")
	}
	if err := json.Unmarshal(body, envelope); err != nil {
		return fmt.Errorf("decode notification envelope: %w", err)
	}
	if envelope.Resource.Algorithm != "AEAD_AES_256_GCM" {
		return fmt.Errorf("unsupported notification algorithm: %s", envelope.Resource.Algorithm)
	}
	plaintext, err := decryptResource(c.apiV3Key, envelope.Resource.Nonce, envelope.Resource.Ciphertext, envelope.Resource.AssociatedData)
	if err != nil {
		return err
	}
	envelope.Plaintext = plaintext
	return nil
}

func decryptResource(key []byte, nonce, ciphertext, associatedData string) ([]byte, error) {
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode notification ciphertext: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("invalid notification nonce")
	}
	plaintext, err := gcm.Open(nil, []byte(nonce), ciphertextBytes, []byte(associatedData))
	if err != nil {
		return nil, fmt.Errorf("decrypt notification: %w", err)
	}
	return plaintext, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, target any) error {
	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	endpoint := c.baseURL + path
	request, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "zoo-go/1.0")
	if body != nil {
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	if err := c.authorize(request, method, path, bodyBytes); err != nil {
		return err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: response.StatusCode}
		var payload struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(responseBody, &payload) == nil {
			apiErr.Code = payload.Code
			apiErr.Message = payload.Message
		}
		if apiErr.Code == "" {
			apiErr.Message = http.StatusText(response.StatusCode)
		}
		return apiErr
	}
	if len(responseBody) == 0 || target == nil {
		return nil
	}
	if err := c.verifyResponse(response.Header, responseBody); err != nil {
		return err
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("decode wechat pay response: %w", err)
	}
	return nil
}

func (c *Client) authorize(request *http.Request, method, canonicalPath string, body []byte) error {
	timestamp := strconv.FormatInt(c.now().Unix(), 10)
	nonce := randomNonce()
	message := method + "\n" + canonicalPath + "\n" + timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	signature, err := sign(c.merchantPrivateKey, message)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",signature="%s",timestamp="%s",serial_no="%s"`,
		c.mchID, nonce, signature, timestamp, c.merchantSerialNo,
	))
	return nil
}

func (c *Client) verifyResponse(header http.Header, body []byte) error {
	timestamp := strings.TrimSpace(header.Get("Wechatpay-Timestamp"))
	nonce := strings.TrimSpace(header.Get("Wechatpay-Nonce"))
	signature := strings.TrimSpace(header.Get("Wechatpay-Signature"))
	serial := strings.TrimSpace(header.Get("Wechatpay-Serial"))
	if timestamp == "" || nonce == "" || signature == "" || serial == "" {
		return errors.New("wechat pay response is missing signature headers")
	}
	if !strings.EqualFold(serial, c.platformSerialNo) {
		return fmt.Errorf("unexpected wechat pay platform serial: %s", serial)
	}
	unix, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("invalid wechat pay response timestamp")
	}
	if delta := c.now().Sub(time.Unix(unix, 0)); delta < -5*time.Minute || delta > 5*time.Minute {
		return errors.New("wechat pay response timestamp is outside the allowed window")
	}
	message := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	if err := verify(c.platformPublicKey, message, signature); err != nil {
		return errors.New("invalid wechat pay response signature")
	}
	return nil
}

func sign(privateKey *rsa.PrivateKey, message string) (string, error) {
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func verify(publicKey *rsa.PublicKey, message, signature string) error {
	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(message))
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signatureBytes)
}

func randomNonce() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		panic(fmt.Sprintf("generate wechat pay nonce: %v", err))
	}
	return hex.EncodeToString(raw)
}

func parseWeChatTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, _ := time.Parse(time.RFC3339, value)
	return parsed
}
