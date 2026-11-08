package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"zoo-go/internal/domain"
	"zoo-go/internal/payment"
	"zoo-go/internal/store"
)

func (s *Server) createPayment(w http.ResponseWriter, r *http.Request) {
	if s.payments == nil {
		s.paymentUnavailable(w)
		return
	}
	challenge, err := s.payments.CreateChallenge(r.Context(), r.PathValue("id"), currentUser(r).ID)
	if err != nil {
		s.writePaymentError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{Data: challenge})
}

func (s *Server) syncPayment(w http.ResponseWriter, r *http.Request) {
	if s.payments == nil {
		s.paymentUnavailable(w)
		return
	}
	view, err := s.payments.SyncPayment(r.Context(), r.PathValue("id"), r.PathValue("paymentId"), currentUser(r).ID)
	if err != nil {
		s.writePaymentError(w, err)
		return
	}
	writeData(w, view)
}

func (s *Server) requestRefund(w http.ResponseWriter, r *http.Request) {
	var req domain.RefundRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if s.payments == nil {
		s.paymentUnavailable(w)
		return
	}
	req.OwnerID = currentUser(r).ID
	refund, err := s.payments.RequestRefund(r.PathValue("id"), req)
	if err != nil {
		s.writePaymentError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{Data: refund})
}

func (s *Server) adminRefunds(w http.ResponseWriter, r *http.Request) {
	status := query(r, "status", domain.RefundRequested)
	if strings.EqualFold(status, "ALL") {
		status = ""
	}
	refunds, err := s.store.ListRefunds(status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, refunds)
}

func (s *Server) approveRefund(w http.ResponseWriter, r *http.Request) {
	if s.payments == nil {
		s.paymentUnavailable(w)
		return
	}
	refund, err := s.payments.ApproveRefund(r.Context(), r.PathValue("id"), "admin")
	if err != nil {
		s.writePaymentError(w, err)
		return
	}
	s.recordAdminAudit("REFUND_APPROVED", "refund", refund.ID, "退款已提交微信处理")
	writeData(w, refund)
}

func (s *Server) rejectRefund(w http.ResponseWriter, r *http.Request) {
	if s.payments == nil {
		s.paymentUnavailable(w)
		return
	}
	var req domain.RejectRefundRequest
	if err := decodeOptional(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := s.payments.RejectRefund(r.PathValue("id"), "admin"); err != nil {
		s.writePaymentError(w, err)
		return
	}
	s.recordAdminAudit("REFUND_REJECTED", "refund", r.PathValue("id"), req.Reason)
	writeData(w, map[string]bool{"ok": true})
}

func (s *Server) wechatPaymentNotify(w http.ResponseWriter, r *http.Request) {
	if s.payments == nil {
		s.paymentUnavailable(w)
		return
	}
	body, err := readNotificationBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope{Error: apiError{Code: "INVALID_NOTIFICATION", Message: "回调内容无效"}})
		return
	}
	if _, err := s.payments.HandlePaymentNotification(r.Header, body); err != nil {
		s.writeNotificationError(w, err, "payment")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"code": "SUCCESS", "message": "成功"})
}

func (s *Server) wechatRefundNotify(w http.ResponseWriter, r *http.Request) {
	if s.payments == nil {
		s.paymentUnavailable(w)
		return
	}
	body, err := readNotificationBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope{Error: apiError{Code: "INVALID_NOTIFICATION", Message: "回调内容无效"}})
		return
	}
	if _, err := s.payments.HandleRefundNotification(r.Header, body); err != nil {
		s.writeNotificationError(w, err, "refund")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"code": "SUCCESS", "message": "成功"})
}

func readNotificationBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(io.LimitReader(r.Body, 2<<20))
}

func (s *Server) writeNotificationError(w http.ResponseWriter, err error, kind string) {
	var invalid *payment.InvalidNotificationError
	if errors.As(err, &invalid) {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope{Error: apiError{Code: "INVALID_NOTIFICATION", Message: "微信支付回调验签失败"}})
		return
	}
	s.logger.Error("handle wechat "+kind+" notification", "error", err)
	writeJSON(w, http.StatusInternalServerError, errorEnvelope{Error: apiError{Code: "NOTIFICATION_FAILED", Message: "回调处理失败"}})
}

func (s *Server) paymentUnavailable(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, errorEnvelope{Error: apiError{
		Code: "PAYMENT_NOT_CONFIGURED", Message: "微信支付尚未配置，请联系平台管理员",
	}})
}

func (s *Server) writePaymentError(w http.ResponseWriter, err error) {
	var appErr *store.AppError
	if errors.As(err, &appErr) {
		writeError(w, err)
		return
	}
	s.logger.Error("wechat payment operation", "error", err)
	writeJSON(w, http.StatusBadGateway, errorEnvelope{Error: apiError{
		Code: "WECHAT_PAY_FAILED", Message: "微信支付服务暂时不可用，请稍后重试",
	}})
}
