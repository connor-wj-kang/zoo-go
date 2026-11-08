package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"zoo-go/internal/domain"
)

func (s *Store) RequestRefund(orderID string, req domain.RefundRequest, now time.Time) (domain.Refund, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "主人申请取消服务并退款"
	}
	if len([]rune(reason)) > 80 {
		return domain.Refund{}, validation("退款原因不能超过 80 字")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Refund{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var ownerID, orderStatus, taskStatus, paymentID, paymentStatus string
	var amountFen, paymentAmount int64
	var startAt string
	err = tx.QueryRow(`
		SELECT o.owner_id, o.status, o.amount_fen, t.status, t.start_at,
		       COALESCE(p.id, ''), COALESCE(p.status, ''), COALESCE(p.amount_fen, 0)
		FROM orders o
		JOIN tasks t ON t.id = o.task_id
		LEFT JOIN payments p ON p.order_id = o.id
		WHERE o.id = ?
	`, orderID).Scan(&ownerID, &orderStatus, &amountFen, &taskStatus, &startAt, &paymentID, &paymentStatus, &paymentAmount)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Refund{}, notFound("订单不存在")
	}
	if err != nil {
		return domain.Refund{}, err
	}
	if ownerID != req.OwnerID {
		return domain.Refund{}, forbidden("无权申请该订单退款")
	}
	if orderStatus != domain.OrderPaid || taskStatus != domain.TaskBooked {
		return domain.Refund{}, conflict("仅已支付且尚未开始服务的订单可申请退款")
	}
	if parseTimestamp(startAt).Before(now) {
		return domain.Refund{}, conflict("服务已开始，请通过客服处理退款")
	}
	if paymentID == "" || paymentStatus != domain.PaymentPaid {
		return domain.Refund{}, conflict("未找到可退款的微信支付记录")
	}
	if paymentAmount != amountFen {
		return domain.Refund{}, conflict("支付金额与订单金额不一致")
	}

	existing, err := activeRefundByOrderTx(tx, orderID)
	if err != nil {
		return domain.Refund{}, err
	}
	if existing != nil {
		if err := tx.Commit(); err != nil {
			return domain.Refund{}, err
		}
		return *existing, nil
	}

	refundID := newID("refund")
	outRefundNo := "zooR_" + refundID
	if _, err := tx.Exec(`
		INSERT INTO refunds(
			id, order_id, payment_id, out_refund_no, amount_fen, total_fen, reason,
			status, requested_by, created_at, updated_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, 'REQUESTED', ?, ?, ?)
	`, refundID, orderID, paymentID, outRefundNo, amountFen, amountFen, reason,
		req.OwnerID, timestamp(now), timestamp(now)); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return domain.Refund{}, conflict("该订单已有退款申请")
		}
		return domain.Refund{}, err
	}
	refund, err := refundByIDTx(tx, refundID)
	if err != nil {
		return domain.Refund{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Refund{}, err
	}
	return refund, nil
}

func (s *Store) StartRefund(refundID, reviewer string, now time.Time) (domain.Refund, string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Refund{}, "", err
	}
	defer func() { _ = tx.Rollback() }()
	var orderID, taskID, status, outTradeNo, orderStatus, taskStatus, paymentStatus, startAt string
	err = tx.QueryRow(`
		SELECT r.order_id, o.task_id, r.status, p.out_trade_no,
		       o.status, t.status, p.status, t.start_at
		FROM refunds r
		JOIN orders o ON o.id = r.order_id
		JOIN payments p ON p.id = r.payment_id
		JOIN tasks t ON t.id = o.task_id
		WHERE r.id = ?
	`, refundID).Scan(&orderID, &taskID, &status, &outTradeNo, &orderStatus, &taskStatus, &paymentStatus, &startAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Refund{}, "", notFound("退款申请不存在")
	}
	if err != nil {
		return domain.Refund{}, "", err
	}
	if status != domain.RefundRequested && status != domain.RefundFailed {
		return domain.Refund{}, "", conflict("退款申请当前状态不能审核")
	}
	if paymentStatus != domain.PaymentPaidLate {
		if orderStatus != domain.OrderPaid || taskStatus != domain.TaskBooked {
			return domain.Refund{}, "", conflict("订单当前状态已不能退款")
		}
		if !parseTimestamp(startAt).After(now) {
			return domain.Refund{}, "", conflict("服务已开始，不能直接退款")
		}
	}
	result, err := tx.Exec(`
		UPDATE refunds
		SET status = 'PROCESSING', reviewed_by = ?, last_error = '', updated_at = ?
		WHERE id = ? AND status IN ('REQUESTED', 'FAILED')
	`, reviewer, timestamp(now), refundID)
	if err != nil {
		return domain.Refund{}, "", err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return domain.Refund{}, "", conflict("退款申请状态已变化")
	}
	if _, err := tx.Exec(`UPDATE orders SET status = 'REFUNDING' WHERE id = ? AND status = 'PAID'`, orderID); err != nil {
		return domain.Refund{}, "", err
	}
	if _, err := tx.Exec(`
		UPDATE tasks SET status = 'REFUNDING', version = version + 1
		WHERE id = ? AND status = 'BOOKED'
	`, taskID); err != nil {
		return domain.Refund{}, "", err
	}
	refund, err := refundByIDTx(tx, refundID)
	if err != nil {
		return domain.Refund{}, "", err
	}
	if err := tx.Commit(); err != nil {
		return domain.Refund{}, "", err
	}
	return refund, outTradeNo, nil
}

func (s *Store) ApplyRefundResult(refundID, wechatRefundID, status string, amountFen int64, successAt, now time.Time) (domain.Refund, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Refund{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var orderID, taskID, currentStatus, paymentStatus string
	var expectedAmount int64
	err = tx.QueryRow(`
		SELECT r.order_id, o.task_id, r.status, r.amount_fen, p.status
		FROM refunds r
		JOIN orders o ON o.id = r.order_id
		JOIN payments p ON p.id = r.payment_id
		WHERE r.id = ?
	`, refundID).Scan(&orderID, &taskID, &currentStatus, &expectedAmount, &paymentStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Refund{}, notFound("退款申请不存在")
	}
	if err != nil {
		return domain.Refund{}, err
	}
	if amountFen <= 0 || amountFen != expectedAmount {
		return domain.Refund{}, validation("退款金额与申请金额不一致")
	}
	normalizedStatus := strings.ToUpper(strings.TrimSpace(status))
	if currentStatus == domain.RefundSucceeded && normalizedStatus == domain.RefundSucceeded {
		refund, err := refundByIDTx(tx, refundID)
		if err != nil {
			return domain.Refund{}, err
		}
		return refund, tx.Commit()
	}
	if currentStatus == domain.RefundSucceeded {
		refund, err := refundByIDTx(tx, refundID)
		if err != nil {
			return domain.Refund{}, err
		}
		return refund, tx.Commit()
	}
	if currentStatus != domain.RefundProcessing && currentStatus != domain.RefundFailed {
		return domain.Refund{}, conflict("退款申请尚未进入处理阶段")
	}
	switch normalizedStatus {
	case domain.RefundSucceeded:
		when := successAt
		if when.IsZero() {
			when = now
		}
		if _, err := tx.Exec(`
			UPDATE refunds
			SET status = 'SUCCESS', wechat_refund_id = ?, succeeded_at = ?, updated_at = ?
			WHERE id = ?
		`, wechatRefundID, timestamp(when), timestamp(now), refundID); err != nil {
			return domain.Refund{}, err
		}
		if _, err := tx.Exec(`UPDATE orders SET status = 'REFUNDED' WHERE id = ?`, orderID); err != nil {
			return domain.Refund{}, err
		}
		if _, err := tx.Exec(`
			UPDATE tasks SET status = 'CANCELLED', version = version + 1
			WHERE id = ? AND status = 'REFUNDING'
		`, taskID); err != nil {
			return domain.Refund{}, err
		}
		if err := releaseStudentSlotsTx(tx, orderID); err != nil {
			return domain.Refund{}, err
		}
	case "PROCESSING":
		if _, err := tx.Exec(`
			UPDATE refunds SET status = 'PROCESSING', wechat_refund_id = ?, updated_at = ? WHERE id = ?
		`, wechatRefundID, timestamp(now), refundID); err != nil {
			return domain.Refund{}, err
		}
	case "CLOSED", "ABNORMAL", domain.RefundFailed:
		if _, err := tx.Exec(`
			UPDATE refunds SET status = 'FAILED', wechat_refund_id = ?, last_error = ?, updated_at = ? WHERE id = ?
		`, wechatRefundID, status, timestamp(now), refundID); err != nil {
			return domain.Refund{}, err
		}
		if paymentStatus != domain.PaymentPaidLate {
			if _, err := tx.Exec(`UPDATE orders SET status = 'PAID' WHERE id = ? AND status = 'REFUNDING'`, orderID); err != nil {
				return domain.Refund{}, err
			}
			if _, err := tx.Exec(`
				UPDATE tasks SET status = 'BOOKED', version = version + 1
				WHERE id = ? AND status = 'REFUNDING'
			`, taskID); err != nil {
				return domain.Refund{}, err
			}
		}
	default:
		return domain.Refund{}, validation("未知退款状态")
	}
	refund, err := refundByIDTx(tx, refundID)
	if err != nil {
		return domain.Refund{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Refund{}, err
	}
	return refund, nil
}

func (s *Store) FailRefund(refundID, message string, now time.Time) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var orderID, taskID, paymentStatus string
	if err := tx.QueryRow(`SELECT order_id FROM refunds WHERE id = ?`, refundID).Scan(&orderID); err != nil {
		return err
	}
	if err := tx.QueryRow(`SELECT task_id FROM orders WHERE id = ?`, orderID).Scan(&taskID); err != nil {
		return err
	}
	if err := tx.QueryRow(`SELECT p.status FROM refunds r JOIN payments p ON p.id = r.payment_id WHERE r.id = ?`, refundID).Scan(&paymentStatus); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE refunds SET status = 'FAILED', last_error = ?, updated_at = ? WHERE id = ? AND status = 'PROCESSING'
	`, truncateRunes(message, 500), timestamp(now), refundID); err != nil {
		return err
	}
	if paymentStatus != domain.PaymentPaidLate {
		if _, err := tx.Exec(`UPDATE orders SET status = 'PAID' WHERE id = ? AND status = 'REFUNDING'`, orderID); err != nil {
			return err
		}
		if _, err := tx.Exec(`
			UPDATE tasks SET status = 'BOOKED', version = version + 1
			WHERE id = ? AND status = 'REFUNDING'
		`, taskID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) RejectRefund(refundID, reviewer string, now time.Time) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var paymentStatus string
	err = tx.QueryRow(`
		SELECT p.status FROM refunds r JOIN payments p ON p.id = r.payment_id WHERE r.id = ?
	`, refundID).Scan(&paymentStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return notFound("退款申请不存在")
	}
	if err != nil {
		return err
	}
	if paymentStatus == domain.PaymentPaidLate {
		return conflict("超时支付必须原路退款，不能驳回")
	}
	result, err := tx.Exec(`
		UPDATE refunds
		SET status = 'REJECTED', reviewed_by = ?, updated_at = ?
		WHERE id = ? AND status IN ('REQUESTED', 'FAILED')
	`, reviewer, timestamp(now), refundID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return conflict("退款申请当前状态不能驳回")
	}
	return tx.Commit()
}

func activeRefundByOrderTx(tx *sql.Tx, orderID string) (*domain.Refund, error) {
	var refundID string
	err := tx.QueryRow(`
		SELECT id FROM refunds
		WHERE order_id = ? AND status IN ('REQUESTED', 'PROCESSING', 'SUCCESS')
		LIMIT 1
	`, orderID).Scan(&refundID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	refund, err := refundByIDTx(tx, refundID)
	if err != nil {
		return nil, err
	}
	return &refund, nil
}

func (s *Store) RefundByOutRefundNo(outRefundNo string) (domain.Refund, error) {
	var refund domain.Refund
	var createdAt string
	var succeededAt sql.NullString
	err := s.db.QueryRow(`
		SELECT id, order_id, payment_id, out_refund_no, wechat_refund_id,
		       amount_fen, total_fen, reason, status, created_at, succeeded_at
		FROM refunds WHERE out_refund_no = ?
	`, outRefundNo).Scan(
		&refund.ID, &refund.OrderID, &refund.PaymentID, &refund.OutRefundNo, &refund.WeChatRefundID,
		&refund.AmountFen, &refund.TotalFen, &refund.Reason, &refund.Status, &createdAt, &succeededAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Refund{}, notFound("退款申请不存在")
	}
	if err != nil {
		return domain.Refund{}, err
	}
	refund.CreatedAt = parseTimestamp(createdAt)
	refund.SucceededAt = nullableTimestamp(succeededAt)
	return refund, nil
}

func (s *Store) RefundPaymentInfo(refundID string) (domain.Refund, string, error) {
	var refund domain.Refund
	var outTradeNo, createdAt string
	var succeededAt sql.NullString
	err := s.db.QueryRow(`
		SELECT r.id, r.order_id, r.payment_id, r.out_refund_no, r.wechat_refund_id,
		       r.amount_fen, r.total_fen, r.reason, r.status, r.created_at, r.succeeded_at,
		       p.out_trade_no
		FROM refunds r JOIN payments p ON p.id = r.payment_id
		WHERE r.id = ?
	`, refundID).Scan(
		&refund.ID, &refund.OrderID, &refund.PaymentID, &refund.OutRefundNo, &refund.WeChatRefundID,
		&refund.AmountFen, &refund.TotalFen, &refund.Reason, &refund.Status, &createdAt, &succeededAt,
		&outTradeNo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Refund{}, "", notFound("退款申请不存在")
	}
	if err != nil {
		return domain.Refund{}, "", err
	}
	refund.CreatedAt = parseTimestamp(createdAt)
	refund.SucceededAt = nullableTimestamp(succeededAt)
	return refund, outTradeNo, nil
}

func (s *Store) ProcessingRefunds() ([]domain.Refund, error) {
	return s.ListRefunds(domain.RefundProcessing)
}

func (s *Store) ListRefunds(status string) ([]domain.Refund, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	query := `
		SELECT id, order_id, payment_id, out_refund_no, wechat_refund_id,
		       amount_fen, total_fen, reason, status, created_at, succeeded_at
		FROM refunds`
	args := []any{}
	if status != "" {
		switch status {
		case domain.RefundRequested, domain.RefundProcessing, domain.RefundSucceeded, domain.RefundFailed, domain.RefundRejected:
		default:
			return nil, validation("退款状态筛选无效")
		}
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at ASC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	refunds := []domain.Refund{}
	for rows.Next() {
		var refund domain.Refund
		var createdAt string
		var succeededAt sql.NullString
		if err := rows.Scan(
			&refund.ID, &refund.OrderID, &refund.PaymentID, &refund.OutRefundNo, &refund.WeChatRefundID,
			&refund.AmountFen, &refund.TotalFen, &refund.Reason, &refund.Status, &createdAt, &succeededAt,
		); err != nil {
			return nil, err
		}
		refund.CreatedAt = parseTimestamp(createdAt)
		refund.SucceededAt = nullableTimestamp(succeededAt)
		refunds = append(refunds, refund)
	}
	return refunds, rows.Err()
}

func refundByIDTx(tx *sql.Tx, refundID string) (domain.Refund, error) {
	var refund domain.Refund
	var createdAt string
	var succeededAt sql.NullString
	err := tx.QueryRow(`
		SELECT id, order_id, payment_id, out_refund_no, wechat_refund_id,
		       amount_fen, total_fen, reason, status, created_at, succeeded_at
		FROM refunds WHERE id = ?
	`, refundID).Scan(
		&refund.ID, &refund.OrderID, &refund.PaymentID, &refund.OutRefundNo, &refund.WeChatRefundID,
		&refund.AmountFen, &refund.TotalFen, &refund.Reason, &refund.Status, &createdAt, &succeededAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Refund{}, notFound("退款申请不存在")
	}
	if err != nil {
		return domain.Refund{}, err
	}
	refund.CreatedAt = parseTimestamp(createdAt)
	refund.SucceededAt = nullableTimestamp(succeededAt)
	return refund, nil
}
