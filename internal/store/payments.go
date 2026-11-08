package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"zoo-go/internal/domain"
)

func paymentOutTradeNo(orderID string) string {
	return "zoo_" + orderID
}

func paymentDescription(petJSON string) string {
	var pet domain.PetSnapshot
	_ = json.Unmarshal([]byte(petJSON), &pet)
	name := strings.TrimSpace(pet.Name)
	description := "上门喂宠服务"
	if name != "" {
		description += "-" + name
	}
	return truncateRunes(description, 127)
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) > max {
		return string(runes[:max])
	}
	return value
}

func (s *Store) PreparePayment(orderID, ownerID string, now time.Time) (domain.PaymentAttempt, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return domain.PaymentAttempt{}, err
	}
	defer tx.Rollback()

	var taskID, orderStatus, taskStatus, taskOrderID string
	var amountFen int64
	var deadline sql.NullString
	var petJSON, openID string
	err = tx.QueryRow(`
		SELECT o.task_id, o.status, o.amount_fen, t.status, t.order_id, t.pay_deadline_at, t.pet_snapshot, u.open_id
		FROM orders o
		JOIN tasks t ON t.id = o.task_id
		JOIN users u ON u.id = o.owner_id
		WHERE o.id = ? AND o.owner_id = ?
	`, orderID, ownerID).Scan(&taskID, &orderStatus, &amountFen, &taskStatus, &taskOrderID, &deadline, &petJSON, &openID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PaymentAttempt{}, notFound("订单不存在")
	}
	if err != nil {
		return domain.PaymentAttempt{}, err
	}
	if orderStatus != domain.OrderUnpaid {
		return domain.PaymentAttempt{}, conflict("订单当前状态不能发起支付")
	}
	if taskStatus != domain.TaskPendingPayment {
		return domain.PaymentAttempt{}, conflict("任务状态已变化，不能支付")
	}
	if taskOrderID != orderID {
		return domain.PaymentAttempt{}, conflict("该订单已不是任务当前成交订单")
	}
	if amountFen <= 0 {
		return domain.PaymentAttempt{}, conflict("订单金额无效")
	}
	expireAt := nullableTimestamp(deadline)
	if expireAt == nil || !now.Before(*expireAt) {
		return domain.PaymentAttempt{}, conflict("中选资格已超时，请重新选择")
	}
	if strings.TrimSpace(openID) == "" || strings.HasPrefix(openID, "dev:") {
		return domain.PaymentAttempt{}, conflict("当前登录不是可支付的微信账号")
	}

	outTradeNo := paymentOutTradeNo(orderID)
	var paymentID, paymentStatus string
	err = tx.QueryRow(`SELECT id, status FROM payments WHERE order_id = ?`, orderID).Scan(&paymentID, &paymentStatus)
	if errors.Is(err, sql.ErrNoRows) {
		paymentID = newID("pay")
		if _, err := tx.Exec(`
			INSERT INTO payments(id, order_id, out_trade_no, amount_fen, status, created_at, updated_at)
			VALUES(?, ?, ?, ?, 'PREPARING', ?, ?)
		`, paymentID, orderID, outTradeNo, amountFen, timestamp(now), timestamp(now)); err != nil {
			return domain.PaymentAttempt{}, err
		}
	} else if err != nil {
		return domain.PaymentAttempt{}, err
	} else {
		if paymentStatus == domain.PaymentPaid || paymentStatus == domain.PaymentPaidLate {
			return domain.PaymentAttempt{}, conflict("订单已经支付")
		}
		if _, err := tx.Exec(`UPDATE payments SET status = 'PREPARING', last_error = '', updated_at = ? WHERE id = ?`,
			timestamp(now), paymentID); err != nil {
			return domain.PaymentAttempt{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.PaymentAttempt{}, err
	}
	return domain.PaymentAttempt{
		ID: paymentID, OrderID: orderID, OutTradeNo: outTradeNo,
		Description: paymentDescription(petJSON), PayerOpenID: openID,
		AmountFen: amountFen, ExpireAt: *expireAt,
	}, nil
}

func (s *Store) MarkPaymentPrepared(paymentID, prepayID string, now time.Time) error {
	_, err := s.db.Exec(`
		UPDATE payments
		SET status = 'PREPARED', prepay_id = ?, last_error = '', updated_at = ?
		WHERE id = ? AND status IN ('PREPARING', 'FAILED')
	`, prepayID, timestamp(now), paymentID)
	return err
}

func (s *Store) MarkPaymentFailed(paymentID, message string, now time.Time) error {
	_, err := s.db.Exec(`
		UPDATE payments
		SET status = 'FAILED', last_error = ?, updated_at = ?
		WHERE id = ? AND status = 'PREPARING' AND prepay_id = ''
	`, truncateRunes(strings.TrimSpace(message), 500), timestamp(now), paymentID)
	return err
}

func (s *Store) ApplyPaymentNotice(notice domain.PaymentNotice, now time.Time) (domain.TaskView, bool, error) {
	if notice.TradeState != "SUCCESS" {
		return domain.TaskView{}, false, conflict("支付尚未成功")
	}
	if strings.TrimSpace(notice.TransactionID) == "" {
		return domain.TaskView{}, false, validation("支付交易号不能为空")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.TaskView{}, false, err
	}
	defer tx.Rollback()

	var paymentID, paymentStatus, transactionID, orderID, ownerID, taskID, taskStatus, taskOrderID, orderStatus string
	var paymentAmount, orderAmount int64
	var payDeadline sql.NullString
	err = tx.QueryRow(`
		SELECT p.id, p.status, p.wechat_transaction_id, p.amount_fen,
		       o.id, o.owner_id, o.task_id, o.status, o.amount_fen,
		       t.status, t.order_id, t.pay_deadline_at
		FROM payments p
		JOIN orders o ON o.id = p.order_id
		JOIN tasks t ON t.id = o.task_id
		WHERE p.out_trade_no = ?
	`, notice.OutTradeNo).Scan(
		&paymentID, &paymentStatus, &transactionID, &paymentAmount,
		&orderID, &ownerID, &taskID, &orderStatus, &orderAmount,
		&taskStatus, &taskOrderID, &payDeadline,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TaskView{}, false, notFound("支付单不存在")
	}
	if err != nil {
		return domain.TaskView{}, false, err
	}
	if paymentAmount != notice.AmountFen || orderAmount != notice.AmountFen {
		return domain.TaskView{}, false, validation("支付金额与订单金额不一致")
	}
	if (paymentStatus == domain.PaymentPaid || paymentStatus == domain.PaymentPaidLate) && transactionID == notice.TransactionID {
		if err := tx.Commit(); err != nil {
			return domain.TaskView{}, false, err
		}
		view, err := s.Order(orderID, ownerID, "owner")
		if err != nil {
			return domain.TaskView{}, false, err
		}
		return view, paymentStatus == domain.PaymentPaidLate, nil
	}
	if transactionID != "" && transactionID != notice.TransactionID {
		return domain.TaskView{}, false, conflict("支付交易号冲突")
	}

	paidAt := notice.SuccessAt
	if paidAt.IsZero() {
		paidAt = now
	}
	deadline := nullableTimestamp(payDeadline)
	late := orderStatus != domain.OrderUnpaid ||
		taskStatus != domain.TaskPendingPayment ||
		taskOrderID != orderID ||
		deadline == nil || paidAt.After(*deadline)

	if !late {
		result, err := tx.Exec(`
			UPDATE orders SET status = 'PAID', paid_at = ?, payment_key = ?
			WHERE id = ? AND status = 'UNPAID'
		`, timestamp(paidAt), notice.OutTradeNo, orderID)
		if err != nil {
			return domain.TaskView{}, false, err
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			late = true
		}
	}
	if !late {
		result, err := tx.Exec(`
			UPDATE tasks SET status = 'BOOKED', version = version + 1
			WHERE id = ? AND status = 'PENDING_PAYMENT' AND order_id = ?
		`, taskID, orderID)
		if err != nil {
			return domain.TaskView{}, false, err
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			return domain.TaskView{}, false, conflict("任务状态已变化，支付需人工退款")
		}
	}

	if late {
		if orderStatus == domain.OrderUnpaid || orderStatus == domain.OrderCancelled {
			if _, err := tx.Exec(`UPDATE orders SET status = 'REFUNDING' WHERE id = ?`, orderID); err != nil {
				return domain.TaskView{}, false, err
			}
		}
		if taskStatus == domain.TaskPendingPayment && taskOrderID == orderID {
			if _, err := tx.Exec(`
				UPDATE tasks SET status = 'REFUNDING', version = version + 1
				WHERE id = ? AND status = 'PENDING_PAYMENT' AND order_id = ?
			`, taskID, orderID); err != nil {
				return domain.TaskView{}, false, err
			}
		}
		if err := createLatePaymentRefundTx(tx, orderID, paymentID, paymentAmount, now); err != nil {
			return domain.TaskView{}, false, err
		}
	}
	paymentFinalStatus := domain.PaymentPaid
	if late {
		paymentFinalStatus = domain.PaymentPaidLate
	}
	result, err := tx.Exec(`
		UPDATE payments
		SET status = ?, wechat_transaction_id = ?, paid_at = ?, last_error = '', updated_at = ?
		WHERE id = ? AND status IN ('PREPARING', 'PREPARED', 'FAILED', 'CLOSED')
	`, paymentFinalStatus, notice.TransactionID, timestamp(paidAt), timestamp(now), paymentID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return domain.TaskView{}, false, conflict("微信支付交易号已被使用")
		}
		return domain.TaskView{}, false, err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return domain.TaskView{}, false, conflict("支付单状态已变化")
	}
	if err := tx.Commit(); err != nil {
		return domain.TaskView{}, false, err
	}
	view, err := s.Order(orderID, ownerID, "owner")
	if err != nil {
		return domain.TaskView{}, false, err
	}
	return view, late, nil
}

func createLatePaymentRefundTx(tx *sql.Tx, orderID, paymentID string, amountFen int64, now time.Time) error {
	var existing string
	err := tx.QueryRow(`
		SELECT id FROM refunds
		WHERE order_id = ? AND status IN ('REQUESTED', 'PROCESSING', 'SUCCESS')
		LIMIT 1
	`, orderID).Scan(&existing)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var ownerID string
	if err := tx.QueryRow(`SELECT owner_id FROM orders WHERE id = ?`, orderID).Scan(&ownerID); err != nil {
		return err
	}
	refundID := newID("refund")
	_, err = tx.Exec(`
		INSERT INTO refunds(
			id, order_id, payment_id, out_refund_no, amount_fen, total_fen, reason,
			status, requested_by, created_at, updated_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, 'REQUESTED', ?, ?, ?)
	`, refundID, orderID, paymentID, "zooR_"+refundID, amountFen, amountFen,
		"支付成功但订单已关闭，需人工审核原路退款", ownerID, timestamp(now), timestamp(now))
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "unique") {
		return nil
	}
	return err
}

type ExpiredPayment struct {
	OrderID    string
	TaskID     string
	OutTradeNo string
}

func (s *Store) ExpiredPendingPayments(now time.Time) ([]ExpiredPayment, error) {
	rows, err := s.db.Query(`
		SELECT o.id, o.task_id, COALESCE(p.out_trade_no, '')
		FROM orders o
		JOIN tasks t ON t.id = o.task_id
		LEFT JOIN payments p ON p.order_id = o.id
		WHERE o.status = 'UNPAID' AND t.status = 'PENDING_PAYMENT'
		  AND t.order_id = o.id AND t.pay_deadline_at IS NOT NULL AND t.pay_deadline_at <= ?
		ORDER BY t.pay_deadline_at ASC
	`, timestamp(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ExpiredPayment{}
	for rows.Next() {
		var item ExpiredPayment
		if err := rows.Scan(&item.OrderID, &item.TaskID, &item.OutTradeNo); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ExpirePendingPayment(orderID string, now time.Time) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var taskID, orderStatus, taskStatus, taskOrderID, startAt, winnerBID string
	var deadline sql.NullString
	err = tx.QueryRow(`
		SELECT o.task_id, o.status, t.status, t.order_id, t.start_at, t.winner_bid_id, t.pay_deadline_at
		FROM orders o JOIN tasks t ON t.id = o.task_id
		WHERE o.id = ?
	`, orderID).Scan(&taskID, &orderStatus, &taskStatus, &taskOrderID, &startAt, &winnerBID, &deadline)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	payDeadline := nullableTimestamp(deadline)
	if orderStatus != domain.OrderUnpaid || taskStatus != domain.TaskPendingPayment ||
		taskOrderID != orderID || payDeadline == nil || now.Before(*payDeadline) {
		return false, nil
	}
	result, err := tx.Exec(`UPDATE orders SET status = 'CANCELLED' WHERE id = ? AND status = 'UNPAID'`, orderID)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return false, nil
	}
	if _, err := tx.Exec(`
		UPDATE payments SET status = 'CLOSED', updated_at = ?
		WHERE order_id = ? AND status IN ('PREPARING', 'PREPARED', 'FAILED')
	`, timestamp(now), orderID); err != nil {
		return false, err
	}
	if err := releaseStudentSlotsTx(tx, orderID); err != nil {
		return false, err
	}
	if parseTimestamp(startAt).After(now.Add(time.Hour)) {
		if _, err := tx.Exec(`
			UPDATE tasks
			SET status = 'BIDDING', winner_bid_id = '', order_id = '', price_fen = 0,
			    pay_deadline_at = NULL, bid_deadline_at = ?, version = version + 1
			WHERE id = ? AND status = 'PENDING_PAYMENT' AND order_id = ?
		`, timestamp(now.Add(30*time.Minute)), taskID, orderID); err != nil {
			return false, err
		}
		if _, err := tx.Exec(`
			UPDATE bids SET status = 'ACTIVE', updated_at = ?
			WHERE task_id = ? AND status IN ('WON', 'INVALIDATED')
		`, timestamp(now), taskID); err != nil {
			return false, err
		}
	} else {
		if _, err := tx.Exec(`
			UPDATE tasks SET status = 'EXPIRED', winner_bid_id = '', order_id = '', pay_deadline_at = NULL,
			                  version = version + 1
			WHERE id = ? AND status = 'PENDING_PAYMENT' AND order_id = ?
		`, taskID, orderID); err != nil {
			return false, err
		}
		if _, err := tx.Exec(`UPDATE bids SET status = 'EXPIRED', updated_at = ? WHERE id = ? AND status = 'WON'`,
			timestamp(now), winnerBID); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) ExpirePendingPayments(now time.Time) (int, error) {
	items, err := s.ExpiredPendingPayments(now)
	if err != nil {
		return 0, err
	}
	expired := 0
	for _, item := range items {
		ok, err := s.ExpirePendingPayment(item.OrderID, now)
		if err != nil {
			return expired, err
		}
		if ok {
			expired++
		}
	}
	return expired, nil
}

func (s *Store) latestRefundByOrder(orderID string) (*domain.Refund, error) {
	var refund domain.Refund
	var createdAt string
	var succeededAt sql.NullString
	err := s.db.QueryRow(`
		SELECT id, order_id, payment_id, out_refund_no, wechat_refund_id,
		       amount_fen, total_fen, reason, status, created_at, succeeded_at
		FROM refunds WHERE order_id = ? ORDER BY created_at DESC LIMIT 1
	`, orderID).Scan(
		&refund.ID, &refund.OrderID, &refund.PaymentID, &refund.OutRefundNo, &refund.WeChatRefundID,
		&refund.AmountFen, &refund.TotalFen, &refund.Reason, &refund.Status, &createdAt, &succeededAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	refund.CreatedAt = parseTimestamp(createdAt)
	refund.SucceededAt = nullableTimestamp(succeededAt)
	return &refund, nil
}

func (s *Store) PaymentForOwner(orderID, paymentID, ownerID string) (domain.PaymentRef, error) {
	var payment domain.PaymentRef
	err := s.db.QueryRow(`
		SELECT p.id, p.order_id, p.out_trade_no, p.status
		FROM payments p JOIN orders o ON o.id = p.order_id
		WHERE p.id = ? AND p.order_id = ? AND o.owner_id = ?
	`, paymentID, orderID, ownerID).Scan(&payment.ID, &payment.OrderID, &payment.OutTradeNo, &payment.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PaymentRef{}, notFound("支付单不存在")
	}
	return payment, err
}

func (s *Store) ClosePaymentForOrder(tx *sql.Tx, orderID string, now time.Time) error {
	_, err := tx.Exec(`
		UPDATE payments SET status = 'CLOSED', updated_at = ?
		WHERE order_id = ? AND status IN ('PREPARING', 'PREPARED', 'FAILED')
	`, timestamp(now), orderID)
	return err
}

func (s *Store) PaymentOutTradeNoForOrder(orderID string) (string, error) {
	var outTradeNo string
	err := s.db.QueryRow(`SELECT out_trade_no FROM payments WHERE order_id = ?`, orderID).Scan(&outTradeNo)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return outTradeNo, err
}
