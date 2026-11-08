package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"zoo-go/internal/domain"
	"zoo-go/internal/geo"
)

func (s *Store) PublishTask(req domain.PublishTaskRequest, now time.Time) (domain.TaskView, error) {
	if _, err := s.userByID(req.OwnerID); err != nil {
		return domain.TaskView{}, err
	}
	pet, err := s.petByOwner(req.PetID, req.OwnerID)
	if err != nil {
		return domain.TaskView{}, err
	}
	if len(req.Services) == 0 {
		return domain.TaskView{}, validation("请至少选择一项服务")
	}
	for _, service := range req.Services {
		if !contains(serviceTypes, service) {
			return domain.TaskView{}, validation("包含不支持的服务项目：" + service)
		}
	}
	if !contains(timeWindows, req.TimeWindow) {
		return domain.TaskView{}, validation("请选择有效的服务时段")
	}
	if req.Mode != domain.ModeBid && req.Mode != domain.ModeFixed {
		return domain.TaskView{}, validation("请选择竞价或一口价模式")
	}
	if req.BudgetFen < budgetMinFen || req.BudgetFen > budgetMaxFen {
		return domain.TaskView{}, validation(fmt.Sprintf("预算需在 ¥%d–¥%d 之间", budgetMinFen/100, budgetMaxFen/100))
	}
	if !req.SafetyConfirmed || !req.AgreementAccepted {
		return domain.TaskView{}, validation("请确认宠物安全情况并同意服务协议")
	}
	areas, err := s.serviceAreaNames()
	if err != nil {
		return domain.TaskView{}, err
	}
	community := strings.TrimSpace(req.Community)
	if community == "" && len(areas) > 0 {
		community = areas[0]
	}
	if !contains(areas, community) {
		return domain.TaskView{}, validation("当前地址不在开放服务区")
	}
	locationPoint := geo.Point{Latitude: req.Latitude, Longitude: req.Longitude}
	if !geo.Valid(locationPoint) {
		return domain.TaskView{}, validation("请在地图中选择服务地址位置")
	}
	locationName := strings.TrimSpace(req.LocationName)
	if locationName == "" {
		locationName = community
	}
	if len([]rune(locationName)) > 120 {
		return domain.TaskView{}, validation("地址位置名称不能超过 120 字")
	}
	addressDetail := strings.TrimSpace(req.AddressDetail)
	if addressDetail == "" {
		saved, err := s.defaultAddress(req.OwnerID)
		if err != nil {
			return domain.TaskView{}, err
		}
		if saved != nil {
			addressDetail = saved.Detail
		}
	}
	if addressDetail == "" || len([]rune(addressDetail)) > 120 {
		return domain.TaskView{}, validation("请填写有效门牌地址，最多 120 字")
	}
	startDate, err := time.ParseInLocation("2006-01-02", req.ServiceDate, now.Location())
	if err != nil {
		return domain.TaskView{}, validation("服务日期格式不正确")
	}
	windowStart := strings.Split(req.TimeWindow, "-")[0]
	startAt, err := time.ParseInLocation("2006-01-02 15:04", req.ServiceDate+" "+windowStart, now.Location())
	if err != nil {
		return domain.TaskView{}, validation("服务时段格式不正确")
	}
	if startAt.Before(now.Add(6*time.Hour)) || startAt.After(now.AddDate(0, 0, 14)) {
		return domain.TaskView{}, validation("预约需至少提前 6 小时，且在 14 天内")
	}
	if req.BidDurationMin == 0 {
		req.BidDurationMin = 60
	}
	if !contains([]string{"15", "30", "60", "120", "720"}, fmt.Sprint(req.BidDurationMin)) {
		return domain.TaskView{}, validation("竞价时长无效")
	}
	deadline := now.Add(time.Duration(req.BidDurationMin) * time.Minute)
	if !deadline.Before(startAt) {
		return domain.TaskView{}, validation("竞价截止时间必须早于服务开始时间")
	}
	dateText := strings.TrimSpace(req.ServiceDateText)
	if dateText == "" {
		weekdays := [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
		dateText = fmt.Sprintf("%s %s", startDate.Format("01-02"), weekdays[startDate.Weekday()])
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		note = "请按清单完成照护并拍照反馈。"
	}
	if len([]rune(note)) > 300 {
		return domain.TaskView{}, validation("备注不能超过 300 字")
	}
	services := nonEmptyStrings(req.Services)
	task := domain.Task{
		ID: newID("task"), OwnerID: req.OwnerID, PetID: pet.ID, Pet: snapshotPet(pet),
		Mode: req.Mode, Status: domain.TaskBidding, BudgetFen: req.BudgetFen,
		ServiceDate: req.ServiceDate, ServiceDateText: dateText, TimeWindow: req.TimeWindow, StartAt: startAt,
		Services: services, Community: community, AddressMasked: community + " · 门牌成交后可见",
		LocationName: locationName, Latitude: locationPoint.Latitude, Longitude: locationPoint.Longitude,
		AddressDetail: addressDetail, Note: note, Checklist: append([]string(nil), services...),
		BidDeadlineAt: deadline, Photos: []string{}, Version: 1, CreatedAt: now,
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.TaskView{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`
		INSERT INTO tasks(id, owner_id, pet_id, pet_snapshot, mode, status, budget_fen, price_fen,
		                  service_date, service_date_text, time_window, start_at, services_json,
		                  community, location_name, latitude, longitude, address_masked, address_detail, note, checklist_json,
		                  bid_deadline_at, pay_deadline_at, winner_bid_id, order_id, photos_json,
		                  rated, version, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, '', '', '[]', 0, 1, ?)
	`, task.ID, task.OwnerID, task.PetID, marshalJSON(task.Pet), task.Mode, task.Status,
		task.BudgetFen, task.ServiceDate, task.ServiceDateText, task.TimeWindow, timestamp(task.StartAt),
		marshalJSON(task.Services), task.Community, task.LocationName, task.Latitude, task.Longitude,
		task.AddressMasked, task.AddressDetail, task.Note,
		marshalJSON(task.Checklist), timestamp(task.BidDeadlineAt), timestamp(task.CreatedAt)); err != nil {
		return domain.TaskView{}, err
	}
	if _, err := tx.Exec(`
		INSERT INTO owner_addresses(owner_id, community, detail, location_name, latitude, longitude, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(owner_id) DO UPDATE SET community = excluded.community, detail = excluded.detail,
			location_name = excluded.location_name, latitude = excluded.latitude,
			longitude = excluded.longitude, updated_at = excluded.updated_at
	`, req.OwnerID, community, addressDetail, locationName, locationPoint.Latitude, locationPoint.Longitude, timestamp(now)); err != nil {
		return domain.TaskView{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.TaskView{}, err
	}
	return s.taskView(task, req.OwnerID, true, true, now)
}

func (s *Store) petByOwner(petID, ownerID string) (domain.Pet, error) {
	var pet domain.Pet
	var flags string
	err := s.db.QueryRow(`
		SELECT id, owner_id, name, species, breed, age, image, note, flags_json
		FROM pets WHERE id = ? AND owner_id = ?
	`, petID, ownerID).Scan(&pet.ID, &pet.OwnerID, &pet.Name, &pet.Species, &pet.Breed,
		&pet.Age, &pet.Image, &pet.Note, &flags)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Pet{}, validation("请选择有效的宠物")
	}
	if err != nil {
		return domain.Pet{}, err
	}
	pet.Flags = decodeStrings(flags)
	return pet, nil
}

func snapshotPet(pet domain.Pet) domain.PetSnapshot {
	return domain.PetSnapshot{
		ID: pet.ID, Name: pet.Name, Species: pet.Species, Breed: pet.Breed, Age: pet.Age,
		Image: pet.Image, Note: pet.Note, Flags: append([]string(nil), pet.Flags...),
	}
}

func (s *Store) SubmitBid(taskID string, req domain.SubmitBidRequest, now time.Time) (domain.TaskView, error) {
	if err := s.expireStudentVerifications(now); err != nil {
		return domain.TaskView{}, err
	}
	student, err := s.userByID(req.StudentID)
	if err != nil {
		return domain.TaskView{}, err
	}
	if !studentVerificationActive(student, now) {
		if student.StudentStatus == domain.StudentExpired {
			return domain.TaskView{}, forbidden("学籍认证已过期，请重新提交")
		}
		return domain.TaskView{}, forbidden("学生认证未通过，暂不能报价")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.TaskView{}, err
	}
	defer func() { _ = tx.Rollback() }()
	task, err := scanTask(tx.QueryRow(`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, taskID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TaskView{}, notFound("任务不存在")
	}
	if err != nil {
		return domain.TaskView{}, err
	}
	if task.Status != domain.TaskBidding {
		return domain.TaskView{}, conflict("任务已停止竞价")
	}
	if !now.Before(task.BidDeadlineAt) {
		return domain.TaskView{}, conflict("报价已截止")
	}
	studentPoint := geo.Point{Latitude: student.ServiceLatitude, Longitude: student.ServiceLongitude}
	if !geo.Valid(studentPoint) {
		return domain.TaskView{}, locationRequired("请先开启定位后再报价")
	}
	taskPoint := geo.Point{Latitude: task.Latitude, Longitude: task.Longitude}
	if !geo.Valid(taskPoint) {
		return domain.TaskView{}, conflict("该任务缺少有效服务位置，无法报价")
	}
	radius := student.ServiceRadiusKM
	if radius <= 0 || radius > geo.MaxDispatchRadiusKM {
		radius = geo.DefaultDispatchRadius
	}
	if geo.DistanceKM(studentPoint, taskPoint) > radius {
		return domain.TaskView{}, forbidden("任务距离超过接单范围，仅可承接 5 公里内任务")
	}
	if req.PriceFen < minBidFen {
		return domain.TaskView{}, validation("报价不能低于 ¥20 底价")
	}
	if req.PriceFen > task.BudgetFen {
		return domain.TaskView{}, validation("报价不能超过主人预算")
	}
	if len([]rune(req.Message)) > 200 {
		return domain.TaskView{}, validation("报价说明不能超过 200 字")
	}
	rows, err := tx.Query(`
		SELECT t.start_at FROM orders o JOIN tasks t ON t.id = o.task_id
		WHERE o.student_id = ? AND o.status IN ('UNPAID', 'PAID', 'IN_SERVICE', 'PENDING_CONFIRM', 'REFUNDING')
	`, req.StudentID)
	if err != nil {
		return domain.TaskView{}, err
	}
	for rows.Next() {
		var startAt string
		if err := rows.Scan(&startAt); err != nil {
			rows.Close()
			return domain.TaskView{}, err
		}
		if absDuration(parseTimestamp(startAt).Sub(task.StartAt)) < time.Hour {
			rows.Close()
			return domain.TaskView{}, conflict("该时段已有冲突订单，暂不能报价")
		}
	}
	if err := rows.Close(); err != nil {
		return domain.TaskView{}, err
	}

	var bidID string
	var revision int
	err = tx.QueryRow(`SELECT id, revision FROM bids WHERE task_id = ? AND student_id = ? AND status = 'ACTIVE'`, taskID, req.StudentID).Scan(&bidID, &revision)
	if err == nil {
		if revision >= 2 {
			return domain.TaskView{}, conflict("报价最多修改 2 次")
		}
		if _, err := tx.Exec(`
			UPDATE bids SET price_fen = ?, message = ?, revision = revision + 1, updated_at = ?
			WHERE id = ?
		`, req.PriceFen, strings.TrimSpace(req.Message), timestamp(now), bidID); err != nil {
			return domain.TaskView{}, err
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		bidID = newID("bid")
		if _, err := tx.Exec(`
			INSERT INTO bids(id, task_id, student_id, price_fen, message, revision, status, created_at, updated_at)
			VALUES(?, ?, ?, ?, ?, 0, 'ACTIVE', ?, ?)
		`, bidID, taskID, req.StudentID, req.PriceFen, strings.TrimSpace(req.Message), timestamp(now), timestamp(now)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return domain.TaskView{}, conflict("你已经提交过报价，请刷新后修改")
			}
			return domain.TaskView{}, err
		}
	} else {
		return domain.TaskView{}, err
	}
	if _, err := tx.Exec(`UPDATE tasks SET version = version + 1 WHERE id = ?`, taskID); err != nil {
		return domain.TaskView{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.TaskView{}, err
	}
	task, err = s.taskByID(taskID)
	if err != nil {
		return domain.TaskView{}, err
	}
	return s.taskView(task, req.StudentID, false, true, now)
}

func (s *Store) AwardTask(taskID string, req domain.AwardRequest, now time.Time) (domain.TaskView, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return domain.TaskView{}, err
	}
	defer func() { _ = tx.Rollback() }()
	task, err := scanTask(tx.QueryRow(`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, taskID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TaskView{}, notFound("任务不存在")
	}
	if err != nil {
		return domain.TaskView{}, err
	}
	if task.OwnerID != req.OwnerID {
		return domain.TaskView{}, forbidden("无权操作该任务")
	}
	if task.Status != domain.TaskBidding {
		return domain.TaskView{}, conflict("任务已被选定或已停止竞价")
	}
	if !now.Before(task.BidDeadlineAt) {
		return domain.TaskView{}, conflict("报价已截止")
	}
	var bid domain.Bid
	var bidCreated string
	err = tx.QueryRow(`
		SELECT id, task_id, student_id, price_fen, message, revision, status, created_at
		FROM bids WHERE id = ? AND task_id = ? AND status = 'ACTIVE'
	`, req.BidID, taskID).Scan(&bid.ID, &bid.TaskID, &bid.StudentID, &bid.PriceFen,
		&bid.Message, &bid.Revision, &bid.Status, &bidCreated)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TaskView{}, conflict("报价已失效，请刷新后重试")
	}
	if err != nil {
		return domain.TaskView{}, err
	}
	var verified int
	var studentStatus, verificationExpiresAt string
	if err := tx.QueryRow(`
		SELECT student_verified, student_status, verification_expires_at
		FROM users WHERE id = ?
	`, bid.StudentID).Scan(&verified, &studentStatus, &verificationExpiresAt); err != nil {
		return domain.TaskView{}, err
	}
	if !studentVerificationActive(dbUser{
		StudentVerified: verified == 1, StudentStatus: studentStatus,
		VerificationExpiresAt: verificationExpiresAt,
	}, now) {
		return domain.TaskView{}, conflict("该学生认证状态异常")
	}
	payDeadline := now.Add(paymentTimeout)
	result, err := tx.Exec(`
		UPDATE tasks SET status = 'PENDING_PAYMENT', price_fen = ?, winner_bid_id = ?, pay_deadline_at = ?,
		                 version = version + 1
		WHERE id = ? AND status = 'BIDDING' AND version = ?
	`, bid.PriceFen, bid.ID, timestamp(payDeadline), taskID, task.Version)
	if err != nil {
		return domain.TaskView{}, err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return domain.TaskView{}, conflict("任务状态已变化，请刷新后重试")
	}
	if _, err := tx.Exec(`UPDATE bids SET status = 'WON', updated_at = ? WHERE id = ?`, timestamp(now), bid.ID); err != nil {
		return domain.TaskView{}, err
	}
	if _, err := tx.Exec(`UPDATE bids SET status = 'INVALIDATED', updated_at = ? WHERE task_id = ? AND id <> ? AND status = 'ACTIVE'`, timestamp(now), taskID, bid.ID); err != nil {
		return domain.TaskView{}, err
	}
	fee := bid.PriceFen * feeRateBasisPoints / 10000
	orderID := newID("order")
	if _, err := tx.Exec(`
		INSERT INTO orders(id, task_id, bid_id, owner_id, student_id, amount_fen, fee_fen, student_fen, status, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, 'UNPAID', ?)
	`, orderID, taskID, bid.ID, req.OwnerID, bid.StudentID, bid.PriceFen, fee, bid.PriceFen-fee, timestamp(now)); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return domain.TaskView{}, conflict("任务已经成交，请刷新")
		}
		return domain.TaskView{}, err
	}
	if _, err := tx.Exec(`UPDATE tasks SET order_id = ? WHERE id = ?`, orderID, taskID); err != nil {
		return domain.TaskView{}, err
	}
	if err := reserveStudentSlotsTx(tx, orderID, bid.StudentID, task.StartAt, task.TimeWindow, timestamp(now)); err != nil {
		return domain.TaskView{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.TaskView{}, err
	}
	task, err = s.taskByID(taskID)
	if err != nil {
		return domain.TaskView{}, err
	}
	return s.taskView(task, req.OwnerID, true, true, now)
}

func (s *Store) ReopenTask(taskID, ownerID string, now time.Time) (domain.TaskView, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return domain.TaskView{}, err
	}
	defer func() { _ = tx.Rollback() }()
	task, err := scanTask(tx.QueryRow(`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, taskID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TaskView{}, notFound("任务不存在")
	}
	if err != nil {
		return domain.TaskView{}, err
	}
	if task.OwnerID != ownerID {
		return domain.TaskView{}, forbidden("无权操作该任务")
	}
	if task.Status != domain.TaskPendingPayment {
		return domain.TaskView{}, conflict("仅待付款任务可重新竞价")
	}
	if task.OrderID != "" {
		var orderStatus string
		if err := tx.QueryRow(`SELECT status FROM orders WHERE id = ?`, task.OrderID).Scan(&orderStatus); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return domain.TaskView{}, err
		}
		if orderStatus == domain.OrderPaid || orderStatus == domain.OrderInService || orderStatus == domain.OrderPendingConfirm || orderStatus == domain.OrderCompleted || orderStatus == domain.OrderRefunding {
			return domain.TaskView{}, conflict("订单已支付，不能重新竞价")
		}
		if orderStatus != "" {
			if _, err := tx.Exec(`UPDATE orders SET status = 'CANCELLED' WHERE id = ?`, task.OrderID); err != nil {
				return domain.TaskView{}, err
			}
		}
		if err := releaseStudentSlotsTx(tx, task.OrderID); err != nil {
			return domain.TaskView{}, err
		}
		if err := s.ClosePaymentForOrder(tx, task.OrderID, now); err != nil {
			return domain.TaskView{}, err
		}
	}
	if _, err := tx.Exec(`
		UPDATE tasks SET status = 'BIDDING', winner_bid_id = '', order_id = '', price_fen = 0,
		                 pay_deadline_at = NULL, bid_deadline_at = ?, version = version + 1
		WHERE id = ?
	`, timestamp(now.Add(30*time.Minute)), taskID); err != nil {
		return domain.TaskView{}, err
	}
	if _, err := tx.Exec(`UPDATE bids SET status = 'ACTIVE', updated_at = ? WHERE task_id = ? AND status IN ('WON', 'INVALIDATED')`, timestamp(now), taskID); err != nil {
		return domain.TaskView{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.TaskView{}, err
	}
	task, err = s.taskByID(taskID)
	if err != nil {
		return domain.TaskView{}, err
	}
	return s.taskView(task, ownerID, true, true, now)
}

func (s *Store) CancelTask(taskID, ownerID string, now time.Time) (domain.TaskView, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return domain.TaskView{}, err
	}
	defer func() { _ = tx.Rollback() }()
	task, err := scanTask(tx.QueryRow(`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, taskID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TaskView{}, notFound("任务不存在")
	}
	if err != nil {
		return domain.TaskView{}, err
	}
	if task.OwnerID != ownerID {
		return domain.TaskView{}, forbidden("无权操作该任务")
	}
	if task.Status != domain.TaskBidding && task.Status != domain.TaskPendingPayment {
		return domain.TaskView{}, conflict("当前状态不能直接撤单")
	}
	if task.OrderID != "" {
		var orderStatus string
		if err := tx.QueryRow(`SELECT status FROM orders WHERE id = ?`, task.OrderID).Scan(&orderStatus); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return domain.TaskView{}, err
		}
		if orderStatus == domain.OrderPaid || orderStatus == domain.OrderInService || orderStatus == domain.OrderPendingConfirm || orderStatus == domain.OrderCompleted || orderStatus == domain.OrderRefunding {
			return domain.TaskView{}, conflict("订单已支付，请走退款/客服流程")
		}
		if orderStatus != "" {
			if _, err := tx.Exec(`UPDATE orders SET status = 'CANCELLED' WHERE id = ?`, task.OrderID); err != nil {
				return domain.TaskView{}, err
			}
		}
		if err := releaseStudentSlotsTx(tx, task.OrderID); err != nil {
			return domain.TaskView{}, err
		}
		if err := s.ClosePaymentForOrder(tx, task.OrderID, now); err != nil {
			return domain.TaskView{}, err
		}
	}
	if _, err := tx.Exec(`UPDATE tasks SET status = 'CANCELLED', version = version + 1 WHERE id = ?`, taskID); err != nil {
		return domain.TaskView{}, err
	}
	if _, err := tx.Exec(`UPDATE bids SET status = 'INVALIDATED', updated_at = ? WHERE task_id = ? AND status IN ('ACTIVE', 'WON')`, timestamp(now), taskID); err != nil {
		return domain.TaskView{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.TaskView{}, err
	}
	task, err = s.taskByID(taskID)
	if err != nil {
		return domain.TaskView{}, err
	}
	return s.taskView(task, ownerID, true, true, now)
}

func (s *Store) CheckIn(orderID, studentID string, now time.Time) (domain.TaskView, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return domain.TaskView{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var taskID, ownerID, status string
	err = tx.QueryRow(`SELECT task_id, student_id, status FROM orders WHERE id = ?`, orderID).Scan(&taskID, &ownerID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TaskView{}, notFound("订单不存在")
	}
	if err != nil {
		return domain.TaskView{}, err
	}
	if ownerID != studentID {
		return domain.TaskView{}, forbidden("无权操作该订单")
	}
	if status == domain.OrderInService || status == domain.OrderPendingConfirm || status == domain.OrderCompleted {
		task, err := s.taskByID(taskID)
		if err != nil {
			return domain.TaskView{}, err
		}
		return s.taskView(task, studentID, false, true, now)
	}
	if status != domain.OrderPaid {
		return domain.TaskView{}, conflict("仅已支付订单可签到")
	}
	if _, err := tx.Exec(`UPDATE orders SET status = 'IN_SERVICE', check_in_at = ? WHERE id = ? AND status = 'PAID'`, timestamp(now), orderID); err != nil {
		return domain.TaskView{}, err
	}
	if _, err := tx.Exec(`UPDATE tasks SET status = 'IN_SERVICE', version = version + 1 WHERE id = ? AND status = 'BOOKED'`, taskID); err != nil {
		return domain.TaskView{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.TaskView{}, err
	}
	task, err := s.taskByID(taskID)
	if err != nil {
		return domain.TaskView{}, err
	}
	return s.taskView(task, studentID, false, true, now)
}

func (s *Store) Complete(orderID string, req domain.CompleteRequest, now time.Time) (domain.TaskView, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return domain.TaskView{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var taskID, studentID, status string
	err = tx.QueryRow(`SELECT task_id, student_id, status FROM orders WHERE id = ?`, orderID).Scan(&taskID, &studentID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TaskView{}, notFound("订单不存在")
	}
	if err != nil {
		return domain.TaskView{}, err
	}
	if studentID != req.StudentID {
		return domain.TaskView{}, forbidden("无权操作该订单")
	}
	if status == domain.OrderPendingConfirm || status == domain.OrderCompleted {
		task, err := s.taskByID(taskID)
		if err != nil {
			return domain.TaskView{}, err
		}
		return s.taskView(task, req.StudentID, false, true, now)
	}
	if status != domain.OrderInService {
		return domain.TaskView{}, conflict("请先到场签到")
	}
	photos := nonEmptyStrings(req.Photos)
	if len(photos) < 2 {
		return domain.TaskView{}, validation("完工需上传至少 2 张服务凭证")
	}
	if len(photos) > 9 {
		return domain.TaskView{}, validation("服务凭证最多 9 张")
	}
	for _, photo := range photos {
		if !strings.HasPrefix(photo, "/uploads/") {
			return domain.TaskView{}, validation("服务凭证必须先上传到平台")
		}
	}
	if _, err := tx.Exec(`UPDATE orders SET status = 'PENDING_CONFIRM', completed_at = ? WHERE id = ? AND status = 'IN_SERVICE'`, timestamp(now), orderID); err != nil {
		return domain.TaskView{}, err
	}
	if _, err := tx.Exec(`UPDATE tasks SET status = 'PENDING_CONFIRM', photos_json = ?, version = version + 1 WHERE id = ?`, marshalJSON(photos), taskID); err != nil {
		return domain.TaskView{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.TaskView{}, err
	}
	task, err := s.taskByID(taskID)
	if err != nil {
		return domain.TaskView{}, err
	}
	return s.taskView(task, req.StudentID, false, true, now)
}

func (s *Store) Confirm(orderID, ownerID string, now time.Time) (domain.TaskView, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return domain.TaskView{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var taskID, orderOwner, status string
	err = tx.QueryRow(`SELECT task_id, owner_id, status FROM orders WHERE id = ?`, orderID).Scan(&taskID, &orderOwner, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TaskView{}, notFound("订单不存在")
	}
	if err != nil {
		return domain.TaskView{}, err
	}
	if orderOwner != ownerID {
		return domain.TaskView{}, forbidden("无权操作该订单")
	}
	if status == domain.OrderCompleted {
		task, err := s.taskByID(taskID)
		if err != nil {
			return domain.TaskView{}, err
		}
		return s.taskView(task, ownerID, true, true, now)
	}
	if status != domain.OrderPendingConfirm {
		return domain.TaskView{}, conflict("订单尚未提交验收")
	}
	if _, err := tx.Exec(`UPDATE orders SET status = 'COMPLETED', confirmed_at = ? WHERE id = ? AND status = 'PENDING_CONFIRM'`, timestamp(now), orderID); err != nil {
		return domain.TaskView{}, err
	}
	if _, err := tx.Exec(`UPDATE tasks SET status = 'COMPLETED', version = version + 1 WHERE id = ? AND status = 'PENDING_CONFIRM'`, taskID); err != nil {
		return domain.TaskView{}, err
	}
	if _, err := tx.Exec(`UPDATE users SET done_count = done_count + 1 WHERE id = (SELECT student_id FROM orders WHERE id = ?)`, orderID); err != nil {
		return domain.TaskView{}, err
	}
	if err := releaseStudentSlotsTx(tx, orderID); err != nil {
		return domain.TaskView{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.TaskView{}, err
	}
	task, err := s.taskByID(taskID)
	if err != nil {
		return domain.TaskView{}, err
	}
	return s.taskView(task, ownerID, true, true, now)
}

func (s *Store) Rate(orderID string, req domain.RateRequest) (domain.TaskView, error) {
	if req.Stars < 1 || req.Stars > 5 {
		return domain.TaskView{}, validation("评分需为 1–5 星")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.TaskView{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var taskID, ownerID, status string
	var rated int
	err = tx.QueryRow(`
		SELECT o.task_id, o.owner_id, o.status, t.rated
		FROM orders o JOIN tasks t ON t.id = o.task_id WHERE o.id = ?
	`, orderID).Scan(&taskID, &ownerID, &status, &rated)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TaskView{}, notFound("订单不存在")
	}
	if err != nil {
		return domain.TaskView{}, err
	}
	if ownerID != req.OwnerID {
		return domain.TaskView{}, forbidden("无权评价该订单")
	}
	if status != domain.OrderCompleted {
		return domain.TaskView{}, conflict("仅已完成订单可评价")
	}
	if rated == 0 {
		if _, err := tx.Exec(`UPDATE tasks SET rated = 1 WHERE id = ?`, taskID); err != nil {
			return domain.TaskView{}, err
		}
		if _, err := tx.Exec(`UPDATE orders SET rating_stars = ? WHERE id = ?`, req.Stars, orderID); err != nil {
			return domain.TaskView{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.TaskView{}, err
	}
	task, err := s.taskByID(taskID)
	if err != nil {
		return domain.TaskView{}, err
	}
	return s.taskView(task, req.OwnerID, true, true, time.Now())
}
