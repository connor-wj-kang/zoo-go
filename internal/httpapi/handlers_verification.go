package httpapi

import (
	"net/http"
	"strings"
	"time"

	"zoo-go/internal/domain"
)

func (s *Server) adminStudentVerifications(w http.ResponseWriter, r *http.Request) {
	applications, err := s.store.StudentVerificationApplications(query(r, "status", domain.StudentPending))
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, applications)
}

func (s *Server) adminApproveStudentVerification(w http.ResponseWriter, r *http.Request) {
	var req domain.ApproveStudentVerificationRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	var validUntil time.Time
	if value := strings.TrimSpace(req.ValidUntil); value != "" {
		parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorEnvelope{Error: apiError{Code: "VALIDATION_ERROR", Message: "认证有效期格式应为 YYYY-MM-DD"}})
			return
		}
		validUntil = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	if err := s.store.ApproveStudentVerification(r.PathValue("id"), "admin", validUntil, s.now()); err != nil {
		writeError(w, err)
		return
	}
	detail := "默认一年有效期"
	if !validUntil.IsZero() {
		detail = "有效期至 " + validUntil.Format("2006-01-02")
	}
	s.recordAdminAudit("STUDENT_VERIFICATION_APPROVED", "student_verification", r.PathValue("id"), detail)
	writeData(w, map[string]bool{"ok": true})
}

func (s *Server) adminRejectStudentVerification(w http.ResponseWriter, r *http.Request) {
	var req domain.RejectStudentVerificationRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := s.store.RejectStudentVerification(r.PathValue("id"), "admin", req.Reason, s.now()); err != nil {
		writeError(w, err)
		return
	}
	s.recordAdminAudit("STUDENT_VERIFICATION_REJECTED", "student_verification", r.PathValue("id"), req.Reason)
	writeData(w, map[string]bool{"ok": true})
}
