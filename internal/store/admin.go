package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"zoo-go/internal/domain"
	"zoo-go/internal/geo"
)

func (s *Store) AdminOverview(now time.Time) (domain.AdminOverview, error) {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var overview domain.AdminOverview
	err := s.db.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM users WHERE account_status = 'DISABLED'),
			(SELECT COUNT(*) FROM users WHERE student_verified = 1 AND student_status = 'VERIFIED'),
			(SELECT COUNT(*) FROM student_verifications WHERE status = 'PENDING'),
			(SELECT COUNT(*) FROM refunds WHERE status = 'REQUESTED'),
			(SELECT COUNT(*) FROM orders WHERE status IN ('UNPAID', 'PAID', 'IN_SERVICE', 'PENDING_CONFIRM')),
			(SELECT COUNT(*) FROM orders WHERE status = 'COMPLETED'),
			(SELECT COUNT(*) FROM orders WHERE created_at >= ?),
			(SELECT COALESCE(SUM(amount_fen), 0) FROM orders WHERE status IN ('PAID', 'IN_SERVICE', 'PENDING_CONFIRM', 'COMPLETED'))
	`, timestamp(dayStart)).Scan(
		&overview.Users, &overview.DisabledUsers, &overview.VerifiedStudents,
		&overview.PendingVerifications, &overview.PendingRefunds,
		&overview.ActiveOrders, &overview.CompletedOrders, &overview.TodayOrders,
		&overview.GrossPaidFen,
	)
	return overview, err
}

func (s *Store) AdminUsers(search, status string, limit int) ([]domain.AdminUser, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	search = strings.TrimSpace(search)
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "ALL" {
		status = ""
	}
	if status != "" && status != domain.AccountActive && status != domain.AccountDisabled {
		return nil, validation("用户状态筛选无效")
	}
	like := "%" + search + "%"
	rows, err := s.db.Query(`
		SELECT u.id, u.name, u.avatar, u.phone_mask, u.owner_verified, u.student_verified,
		       u.student_status, u.school, u.account_status, u.disabled_reason,
		       u.rating, u.done_count, u.created_at,
		       (SELECT COUNT(*) FROM orders o WHERE (o.owner_id = u.id OR o.student_id = u.id)
		         AND o.status IN ('UNPAID', 'PAID', 'IN_SERVICE', 'PENDING_CONFIRM'))
		FROM users u
		WHERE (? = '' OR u.name LIKE ? OR u.id LIKE ? OR u.phone_mask LIKE ? OR u.school LIKE ?)
		  AND (? = '' OR u.account_status = ?)
		ORDER BY u.created_at DESC, u.id DESC
		LIMIT ?
	`, search, like, like, like, like, status, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []domain.AdminUser{}
	for rows.Next() {
		var user domain.AdminUser
		var ownerVerified, studentVerified int
		var createdAt string
		if err := rows.Scan(
			&user.ID, &user.Name, &user.Avatar, &user.PhoneMask, &ownerVerified, &studentVerified,
			&user.StudentStatus, &user.School, &user.AccountStatus, &user.DisabledReason,
			&user.Rating, &user.DoneCount, &createdAt, &user.ActiveOrders,
		); err != nil {
			return nil, err
		}
		user.OwnerVerified = ownerVerified == 1
		user.StudentVerified = studentVerified == 1
		user.CreatedAt = parseTimestamp(createdAt)
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) UpdateUserAccountStatus(userID, status, reason string, now time.Time) error {
	userID = strings.TrimSpace(userID)
	status = strings.ToUpper(strings.TrimSpace(status))
	reason = strings.TrimSpace(reason)
	if status != domain.AccountActive && status != domain.AccountDisabled {
		return validation("账号状态无效")
	}
	if status == domain.AccountDisabled && reason == "" {
		return validation("停用账号必须填写原因")
	}
	if len([]rune(reason)) > 200 {
		return validation("停用原因不能超过 200 字")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	nowText := timestamp(now)
	var disabledAt string
	if status == domain.AccountDisabled {
		disabledAt = nowText
	} else {
		reason = ""
		disabledAt = ""
	}
	result, err := tx.Exec(`
		UPDATE users SET account_status = ?, disabled_reason = ?, disabled_at = ?, updated_at = ?
		WHERE id = ?
	`, status, reason, disabledAt, nowText, userID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return notFound("用户不存在")
	}
	if status == domain.AccountDisabled {
		if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) AdminOrders(status string, limit int) ([]domain.AdminOrder, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "ALL" {
		status = ""
	}
	if status != "" {
		valid := map[string]bool{
			domain.OrderUnpaid: true, domain.OrderPaid: true, domain.OrderInService: true,
			domain.OrderPendingConfirm: true, domain.OrderCompleted: true,
			domain.OrderRefunding: true, domain.OrderCancelled: true, domain.OrderRefunded: true,
		}
		if !valid[status] {
			return nil, validation("订单状态筛选无效")
		}
	}
	rows, err := s.db.Query(`
		SELECT o.id, o.task_id, o.status, t.status, o.owner_id, owner.name,
		       o.student_id, student.name, COALESCE(pet.name, ''), t.service_date, t.time_window,
		       t.community, o.amount_fen, o.fee_fen, o.student_fen,
		       COALESCE(payment.status, ''), o.created_at
		FROM orders o
		JOIN tasks t ON t.id = o.task_id
		JOIN users owner ON owner.id = o.owner_id
		JOIN users student ON student.id = o.student_id
		LEFT JOIN pets pet ON pet.id = t.pet_id
		LEFT JOIN payments payment ON payment.order_id = o.id
		WHERE (? = '' OR o.status = ?)
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT ?
	`, status, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := []domain.AdminOrder{}
	for rows.Next() {
		var order domain.AdminOrder
		var createdAt string
		if err := rows.Scan(
			&order.ID, &order.TaskID, &order.Status, &order.TaskStatus,
			&order.OwnerID, &order.OwnerName, &order.StudentID, &order.StudentName,
			&order.PetName, &order.ServiceDate, &order.TimeWindow, &order.Community,
			&order.AmountFen, &order.FeeFen, &order.StudentFen, &order.PaymentStatus, &createdAt,
		); err != nil {
			return nil, err
		}
		order.CreatedAt = parseTimestamp(createdAt)
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (s *Store) AddAdminAudit(actor, action, resourceType, resourceID, detail string, now time.Time) error {
	actor = strings.TrimSpace(actor)
	action = strings.TrimSpace(action)
	resourceType = strings.TrimSpace(resourceType)
	resourceID = strings.TrimSpace(resourceID)
	detail = strings.TrimSpace(detail)
	if actor == "" || action == "" || resourceType == "" || resourceID == "" {
		return validation("审计日志字段不完整")
	}
	if len([]rune(detail)) > 1000 {
		detail = string([]rune(detail)[:1000])
	}
	_, err := s.db.Exec(`
		INSERT INTO admin_audit_logs(id, actor, action, resource_type, resource_id, detail, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
	`, newID("audit"), actor, action, resourceType, resourceID, detail, timestamp(now))
	return err
}

func (s *Store) AdminAuditLogs(limit int) ([]domain.AdminAuditLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(`
		SELECT id, actor, action, resource_type, resource_id, detail, created_at
		FROM admin_audit_logs
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	logs := []domain.AdminAuditLog{}
	for rows.Next() {
		var item domain.AdminAuditLog
		var createdAt string
		if err := rows.Scan(&item.ID, &item.Actor, &item.Action, &item.ResourceType,
			&item.ResourceID, &item.Detail, &createdAt); err != nil {
			return nil, err
		}
		item.CreatedAt = parseTimestamp(createdAt)
		logs = append(logs, item)
	}
	return logs, rows.Err()
}

func (s *Store) AdminDispatchInputs(now time.Time, limit int) ([]domain.DispatchTaskCandidate, []domain.DispatchStudentCandidate, error) {
	if err := s.expireStudentVerifications(now); err != nil {
		return nil, nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	taskRows, err := s.db.Query(`
		SELECT id, community, latitude, longitude
		FROM tasks
		WHERE status = 'BIDDING' AND bid_deadline_at > ?
		ORDER BY start_at ASC, id ASC
		LIMIT ?
	`, timestamp(now), limit)
	if err != nil {
		return nil, nil, err
	}
	tasks := []domain.DispatchTaskCandidate{}
	for taskRows.Next() {
		var task domain.DispatchTaskCandidate
		if err := taskRows.Scan(&task.ID, &task.Community, &task.Latitude, &task.Longitude); err != nil {
			taskRows.Close()
			return nil, nil, err
		}
		if geo.Valid(geo.Point{Latitude: task.Latitude, Longitude: task.Longitude}) {
			tasks = append(tasks, task)
		}
	}
	if err := taskRows.Close(); err != nil {
		return nil, nil, err
	}

	studentRows, err := s.db.Query(`
		SELECT id, name, service_latitude, service_longitude, rating, done_count
		FROM users
		WHERE account_status = 'ACTIVE' AND student_verified = 1 AND student_status = 'VERIFIED'
		  AND service_latitude <> 0 AND service_longitude <> 0
		ORDER BY rating DESC, done_count DESC, id ASC
		LIMIT ?
	`, limit*2)
	if err != nil {
		return nil, nil, err
	}
	students := []domain.DispatchStudentCandidate{}
	for studentRows.Next() {
		var student domain.DispatchStudentCandidate
		if err := studentRows.Scan(&student.ID, &student.Name, &student.Latitude, &student.Longitude,
			&student.Rating, &student.DoneCount); err != nil {
			studentRows.Close()
			return nil, nil, err
		}
		if geo.Valid(geo.Point{Latitude: student.Latitude, Longitude: student.Longitude}) {
			students = append(students, student)
		}
	}
	if err := studentRows.Close(); err != nil {
		return nil, nil, err
	}
	return tasks, students, nil
}

func (s *Store) accountStatus(userID string) (string, string, error) {
	var status, reason string
	err := s.db.QueryRow(`SELECT account_status, disabled_reason FROM users WHERE id = ?`, userID).Scan(&status, &reason)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", notFound("用户不存在")
	}
	return status, reason, err
}
