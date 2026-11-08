package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zoo-go/internal/domain"
	"zoo-go/internal/payment"
	"zoo-go/internal/store"
)

type Config struct {
	DevAuth         bool
	WeChatAppID     string
	WeChatAppSecret string
	AdminToken      string
	UploadDir       string
	AdminDir        string
	Payments        *payment.Service
}

type Server struct {
	store    *store.Store
	logger   *slog.Logger
	config   Config
	payments *payment.Service
	now      func() time.Time
}

type contextKey string

const userContextKey contextKey = "authenticated-user"

func New(s *store.Store, config Config, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	if config.UploadDir == "" {
		config.UploadDir = "data/uploads"
	}
	if config.AdminDir == "" {
		config.AdminDir = "admin"
	}
	_ = os.MkdirAll(config.UploadDir, 0o755)
	return &Server{store: s, logger: logger, config: config, payments: config.Payments, now: time.Now}
}

func (s *Server) Handler() http.Handler {
	api := http.NewServeMux()
	api.HandleFunc("GET /v1/auth/me", s.authMe)
	api.HandleFunc("POST /v1/auth/logout", s.logout)
	api.HandleFunc("POST /v1/profile", s.updateProfile)
	api.HandleFunc("GET /v1/meta", s.meta)
	api.HandleFunc("GET /v1/owner/home", s.ownerHome)
	api.HandleFunc("GET /v1/owner/profile", s.ownerProfile)
	api.HandleFunc("GET /v1/owner/pets", s.ownerPets)
	api.HandleFunc("POST /v1/pets", s.createPet)
	api.HandleFunc("GET /v1/owner/tasks", s.ownerTasks)
	api.HandleFunc("POST /v1/tasks", s.publishTask)
	api.HandleFunc("GET /v1/tasks/{id}", s.task)
	api.HandleFunc("POST /v1/tasks/{id}/bids", s.submitBid)
	api.HandleFunc("POST /v1/tasks/{id}/award", s.awardTask)
	api.HandleFunc("POST /v1/tasks/{id}/reopen", s.reopenTask)
	api.HandleFunc("POST /v1/tasks/{id}/cancel", s.cancelTask)
	api.HandleFunc("GET /v1/orders/{id}", s.order)
	api.HandleFunc("POST /v1/orders/{id}/payments", s.createPayment)
	api.HandleFunc("GET /v1/orders/{id}/payments/{paymentId}", s.syncPayment)
	api.HandleFunc("POST /v1/orders/{id}/refunds", s.requestRefund)
	api.HandleFunc("POST /v1/orders/{id}/checkin", s.checkIn)
	api.HandleFunc("POST /v1/orders/{id}/complete", s.complete)
	api.HandleFunc("POST /v1/orders/{id}/confirm", s.confirm)
	api.HandleFunc("POST /v1/orders/{id}/rate", s.rate)
	api.HandleFunc("GET /v1/student/tasks/available", s.availableTasks)
	api.HandleFunc("POST /v1/student/location", s.updateStudentLocation)
	api.HandleFunc("GET /v1/student/tasks", s.studentTasks)
	api.HandleFunc("GET /v1/student/earnings", s.studentEarnings)
	api.HandleFunc("GET /v1/student/profile", s.studentProfile)
	api.HandleFunc("POST /v1/student/verification", s.submitStudentVerification)
	api.HandleFunc("GET /v1/community/posts", s.communityPosts)
	api.HandleFunc("POST /v1/community/posts", s.createCommunityPost)
	api.HandleFunc("POST /v1/uploads", s.upload)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/healthz", s.health)
	mux.HandleFunc("POST /v1/auth/wechat/login", s.login)
	mux.HandleFunc("POST /v1/admin/students/{id}/approve", s.adminOnly(s.adminApproveStudent))
	mux.HandleFunc("GET /v1/admin/overview", s.adminOnly(s.adminOverview))
	mux.HandleFunc("GET /v1/admin/dispatch/compare", s.adminOnly(s.adminDispatchCompare))
	mux.HandleFunc("GET /v1/admin/users", s.adminOnly(s.adminUsers))
	mux.HandleFunc("POST /v1/admin/users/{id}/status", s.adminOnly(s.adminUpdateUserStatus))
	mux.HandleFunc("GET /v1/admin/orders", s.adminOnly(s.adminOrders))
	mux.HandleFunc("GET /v1/admin/audit-logs", s.adminOnly(s.adminAuditLogs))
	mux.HandleFunc("GET /v1/admin/student-verifications", s.adminOnly(s.adminStudentVerifications))
	mux.HandleFunc("POST /v1/admin/student-verifications/{id}/approve", s.adminOnly(s.adminApproveStudentVerification))
	mux.HandleFunc("POST /v1/admin/student-verifications/{id}/reject", s.adminOnly(s.adminRejectStudentVerification))
	mux.HandleFunc("GET /v1/admin/refunds", s.adminOnly(s.adminRefunds))
	mux.HandleFunc("POST /v1/admin/refunds/{id}/approve", s.adminOnly(s.approveRefund))
	mux.HandleFunc("POST /v1/admin/refunds/{id}/reject", s.adminOnly(s.rejectRefund))
	mux.HandleFunc("POST /v1/payments/wechat/notify", s.wechatPaymentNotify)
	mux.HandleFunc("POST /v1/refunds/wechat/notify", s.wechatRefundNotify)
	mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusTemporaryRedirect)
	})
	adminFiles := http.StripPrefix("/admin/", http.FileServer(http.Dir(s.config.AdminDir)))
	mux.Handle("GET /admin/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		adminFiles.ServeHTTP(w, r)
	}))
	uploads := http.StripPrefix("/uploads/", http.FileServer(http.Dir(s.config.UploadDir)))
	mux.Handle("GET /uploads/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/uploads/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=86400")
		uploads.ServeHTTP(w, r)
	}))
	mux.Handle("/v1/", s.authenticate(api))
	return s.middleware(mux)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Token")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("panic recovered", "panic", recovered, "path", r.URL.Path)
				writeJSON(w, http.StatusInternalServerError, errorEnvelope{Error: apiError{Code: "INTERNAL_ERROR", Message: "服务暂时不可用"}})
			}
			s.logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String())
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		profile, err := s.store.Authenticate(token, s.now())
		if err != nil {
			writeError(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, profile)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.config.AdminToken == "" || r.Header.Get("X-Admin-Token") != s.config.AdminToken {
			writeJSON(w, http.StatusForbidden, errorEnvelope{Error: apiError{Code: "FORBIDDEN", Message: "管理员凭证无效"}})
			return
		}
		next(w, r)
	}
}

func currentUser(r *http.Request) domain.Profile {
	profile, _ := r.Context().Value(userContextKey).(domain.Profile)
	return profile
}

func bearerToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": s.now().UTC().Format(time.RFC3339)})
}

type envelope struct {
	Data any `json:"data,omitempty"`
}

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeData(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, envelope{Data: data})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	var appErr *store.AppError
	if errors.As(err, &appErr) {
		writeJSON(w, appErr.Status, errorEnvelope{Error: apiError{Code: appErr.Code, Message: appErr.Message}})
		return
	}
	var httpErr *httpError
	if errors.As(err, &httpErr) {
		writeJSON(w, httpErr.status, errorEnvelope{Error: apiError{Code: httpErr.code, Message: httpErr.message}})
		return
	}
	var uploadErr *uploadError
	if errors.As(err, &uploadErr) {
		writeJSON(w, http.StatusBadRequest, errorEnvelope{Error: apiError{Code: "UPLOAD_ERROR", Message: uploadErr.message}})
		return
	}
	writeJSON(w, http.StatusInternalServerError, errorEnvelope{Error: apiError{Code: "INTERNAL_ERROR", Message: "服务暂时不可用"}})
}

func decode(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return &store.AppError{Status: http.StatusBadRequest, Code: "INVALID_JSON", Message: "请求内容格式不正确"}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return &store.AppError{Status: http.StatusBadRequest, Code: "INVALID_JSON", Message: "请求只能包含一个 JSON 对象"}
	}
	return nil
}

func decodeOptional(r *http.Request, target any) error {
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(data)) == "" {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return &store.AppError{Status: http.StatusBadRequest, Code: "INVALID_JSON", Message: "请求内容格式不正确"}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return &store.AppError{Status: http.StatusBadRequest, Code: "INVALID_JSON", Message: "请求只能包含一个 JSON 对象"}
	}
	return nil
}

func query(r *http.Request, key, fallback string) string {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	return value
}

func (s *Server) uploadPath(name string) string {
	return filepath.Join(s.config.UploadDir, name)
}
