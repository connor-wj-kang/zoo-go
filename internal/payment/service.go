package payment

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"zoo-go/internal/domain"
	"zoo-go/internal/store"
)

type Gateway interface {
	CreateJSAPI(context.Context, domain.PaymentAttempt) (domain.PaymentParams, error)
	QueryPayment(context.Context, string) (domain.PaymentNotice, error)
	ClosePayment(context.Context, string) error
	CreateRefund(context.Context, RefundRequest) (domain.RefundNotice, error)
	QueryRefund(context.Context, string) (domain.RefundNotice, error)
	ParsePaymentNotification(http.Header, []byte) (PaymentEvent, error)
	ParseRefundNotification(http.Header, []byte) (RefundEvent, error)
}

type InvalidNotificationError struct {
	Reason string
}

func (e *InvalidNotificationError) Error() string {
	return "invalid wechat pay notification: " + e.Reason
}

type Service struct {
	store   *store.Store
	gateway Gateway
	logger  *slog.Logger
	now     func() time.Time
}

func NewService(st *store.Store, gateway Gateway, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{store: st, gateway: gateway, logger: logger, now: time.Now}
}

func (s *Service) CreateChallenge(ctx context.Context, orderID, ownerID string) (domain.PaymentChallenge, error) {
	attempt, err := s.store.PreparePayment(orderID, ownerID, s.now())
	if err != nil {
		return domain.PaymentChallenge{}, err
	}
	params, err := s.gateway.CreateJSAPI(ctx, attempt)
	if err != nil {
		_ = s.store.MarkPaymentFailed(attempt.ID, "微信支付预下单失败", s.now())
		return domain.PaymentChallenge{}, err
	}
	if err := s.store.MarkPaymentPrepared(attempt.ID, strings.TrimPrefix(params.Package, "prepay_id="), s.now()); err != nil {
		return domain.PaymentChallenge{}, err
	}
	return domain.PaymentChallenge{
		PaymentID: attempt.ID, OrderID: orderID, OutTradeNo: attempt.OutTradeNo, Params: params,
	}, nil
}

func (s *Service) SyncPayment(ctx context.Context, orderID, paymentID, ownerID string) (domain.TaskView, error) {
	ref, err := s.store.PaymentForOwner(orderID, paymentID, ownerID)
	if err != nil {
		return domain.TaskView{}, err
	}
	switch ref.Status {
	case domain.PaymentPaid, domain.PaymentPaidLate, domain.PaymentClosed:
		return s.store.Order(orderID, ownerID, "owner")
	}
	notice, err := s.gateway.QueryPayment(ctx, ref.OutTradeNo)
	if err != nil {
		return domain.TaskView{}, err
	}
	switch notice.TradeState {
	case "SUCCESS":
		view, _, err := s.store.ApplyPaymentNotice(notice, s.now())
		return view, err
	case "CLOSED", "REVOKED", "PAYERROR":
		_ = s.store.MarkPaymentFailed(ref.ID, "微信支付状态："+notice.TradeState, s.now())
	}
	return s.store.Order(orderID, ownerID, "owner")
}

func (s *Service) RequestRefund(orderID string, request domain.RefundRequest) (domain.Refund, error) {
	return s.store.RequestRefund(orderID, request, s.now())
}

func (s *Service) ApproveRefund(ctx context.Context, refundID, reviewer string) (domain.Refund, error) {
	refund, outTradeNo, err := s.store.StartRefund(refundID, reviewer, s.now())
	if err != nil {
		return domain.Refund{}, err
	}
	notice, err := s.gateway.CreateRefund(ctx, RefundRequest{
		OutTradeNo: outTradeNo, OutRefundNo: refund.OutRefundNo, Reason: refund.Reason,
		RefundFen: refund.AmountFen, TotalFen: refund.TotalFen,
	})
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			_ = s.store.FailRefund(refund.ID, apiErr.Error(), s.now())
		} else {
			s.logger.Error("wechat refund result is uncertain", "refund", refund.ID, "error", err)
		}
		return domain.Refund{}, err
	}
	amount := notice.AmountFen
	if amount <= 0 {
		amount = refund.AmountFen
	}
	status := strings.ToUpper(notice.Status)
	if status == "" {
		status = domain.RefundProcessing
	}
	return s.store.ApplyRefundResult(refund.ID, notice.WeChatRefundID, status, amount, notice.SuccessAt, s.now())
}

func (s *Service) RejectRefund(refundID, reviewer string) error {
	return s.store.RejectRefund(refundID, reviewer, s.now())
}

func (s *Service) HandlePaymentNotification(header http.Header, body []byte) (bool, error) {
	event, err := s.gateway.ParsePaymentNotification(header, body)
	if err != nil {
		return false, &InvalidNotificationError{Reason: err.Error()}
	}
	if event.EventType != "TRANSACTION.SUCCESS" || event.Notice.TradeState != "SUCCESS" {
		return false, nil
	}
	_, late, err := s.store.ApplyPaymentNotice(event.Notice, s.now())
	if err != nil {
		return false, err
	}
	if late {
		s.logger.Warn("late wechat payment converted to refund", "out_trade_no", event.Notice.OutTradeNo)
	}
	return true, nil
}

func (s *Service) HandleRefundNotification(header http.Header, body []byte) (bool, error) {
	event, err := s.gateway.ParseRefundNotification(header, body)
	if err != nil {
		return false, &InvalidNotificationError{Reason: err.Error()}
	}
	if !strings.HasPrefix(event.EventType, "REFUND.") || event.Notice.OutRefundNo == "" {
		return false, nil
	}
	refund, err := s.store.RefundByOutRefundNo(event.Notice.OutRefundNo)
	if err != nil {
		return false, err
	}
	status := strings.ToUpper(event.Notice.Status)
	if status == "" {
		return false, nil
	}
	if _, err := s.store.ApplyRefundResult(refund.ID, event.Notice.WeChatRefundID, status, event.Notice.AmountFen, event.Notice.SuccessAt, s.now()); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) ClosePaymentForOrder(ctx context.Context, orderID string) {
	outTradeNo, err := s.store.PaymentOutTradeNoForOrder(orderID)
	if err != nil || outTradeNo == "" {
		return
	}
	if err := s.gateway.ClosePayment(ctx, outTradeNo); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.IsCode("ORDERPAID") {
			notice, queryErr := s.gateway.QueryPayment(ctx, outTradeNo)
			if queryErr == nil && notice.TradeState == "SUCCESS" {
				if _, _, applyErr := s.store.ApplyPaymentNotice(notice, s.now()); applyErr == nil {
					return
				} else {
					s.logger.Error("apply payment discovered while closing order", "order", orderID, "error", applyErr)
				}
			}
		}
		s.logger.Warn("close wechat payment order", "order", orderID, "out_trade_no", outTradeNo, "error", err)
	}
}

func (s *Service) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	s.RunOnce(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.RunOnce(ctx)
		}
	}
}

func (s *Service) RunOnce(ctx context.Context) {
	s.expirePayments(ctx)
	s.reconcileRefunds(ctx)
}

func (s *Service) expirePayments(ctx context.Context) {
	candidates, err := s.store.ExpiredPendingPayments(s.now())
	if err != nil {
		s.logger.Error("list expired payments", "error", err)
		return
	}
	for _, candidate := range candidates {
		if ctx.Err() != nil {
			return
		}
		if candidate.OutTradeNo != "" {
			closeErr := s.gateway.ClosePayment(ctx, candidate.OutTradeNo)
			if closeErr != nil {
				notice, queryErr := s.gateway.QueryPayment(ctx, candidate.OutTradeNo)
				if queryErr == nil && notice.TradeState == "SUCCESS" {
					if _, _, err := s.store.ApplyPaymentNotice(notice, s.now()); err != nil {
						s.logger.Error("apply late payment during expiry", "order", candidate.OrderID, "error", err)
					}
					continue
				}
				if queryErr != nil && !isOrderNotExist(closeErr) {
					s.logger.Warn("skip expiring payment because close result is uncertain", "order", candidate.OrderID, "error", closeErr)
					continue
				}
			}
		}
		expired, err := s.store.ExpirePendingPayment(candidate.OrderID, s.now())
		if err != nil {
			s.logger.Error("expire pending payment", "order", candidate.OrderID, "error", err)
			continue
		}
		if expired {
			s.logger.Info("expired unpaid order", "order", candidate.OrderID)
		}
	}
}

func (s *Service) reconcileRefunds(ctx context.Context) {
	refunds, err := s.store.ProcessingRefunds()
	if err != nil {
		s.logger.Error("list processing refunds", "error", err)
		return
	}
	for _, refund := range refunds {
		if ctx.Err() != nil {
			return
		}
		refund, outTradeNo, err := s.store.RefundPaymentInfo(refund.ID)
		if err != nil {
			s.logger.Error("load processing refund", "refund", refund.ID, "error", err)
			continue
		}
		notice, err := s.gateway.QueryRefund(ctx, refund.OutRefundNo)
		if err != nil && isRefundNotExist(err) {
			notice, err = s.gateway.CreateRefund(ctx, RefundRequest{
				OutTradeNo: outTradeNo, OutRefundNo: refund.OutRefundNo, Reason: refund.Reason,
				RefundFen: refund.AmountFen, TotalFen: refund.TotalFen,
			})
		}
		if err != nil {
			s.logger.Warn("query processing refund", "refund", refund.ID, "error", err)
			continue
		}
		amount := notice.AmountFen
		if amount <= 0 {
			amount = refund.AmountFen
		}
		if _, err := s.store.ApplyRefundResult(refund.ID, notice.WeChatRefundID, notice.Status, amount, notice.SuccessAt, s.now()); err != nil {
			s.logger.Error("apply queried refund", "refund", refund.ID, "status", notice.Status, "error", err)
		}
	}
}

func isOrderNotExist(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.IsCode("ORDER_NOT_EXIST", "ORDER_NOT_EXISTS")
}

func isRefundNotExist(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.IsCode("RESOURCE_NOT_EXISTS", "RESOURCE_NOT_FOUND")
}
