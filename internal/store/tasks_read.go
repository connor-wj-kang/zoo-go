package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"zoo-go/internal/domain"
	"zoo-go/internal/geo"
)

const taskColumns = `
	id, owner_id, pet_id, pet_snapshot, mode, status, budget_fen, price_fen,
	service_date, service_date_text, time_window, start_at, services_json,
	community, location_name, latitude, longitude, address_masked, address_detail, note, checklist_json,
	bid_deadline_at, pay_deadline_at, winner_bid_id, order_id, photos_json,
	rated, version, created_at
`

type taskScanner interface {
	Scan(...any) error
}

func scanTask(scanner taskScanner) (domain.Task, error) {
	var task domain.Task
	var petJSON, servicesJSON, checklistJSON, photosJSON string
	var payDeadline sql.NullString
	var startAt, bidDeadline, createdAt string
	var rated int
	err := scanner.Scan(
		&task.ID, &task.OwnerID, &task.PetID, &petJSON, &task.Mode, &task.Status,
		&task.BudgetFen, &task.PriceFen, &task.ServiceDate, &task.ServiceDateText,
		&task.TimeWindow, &startAt, &servicesJSON, &task.Community, &task.LocationName,
		&task.Latitude, &task.Longitude, &task.AddressMasked,
		&task.AddressDetail, &task.Note, &checklistJSON, &bidDeadline, &payDeadline,
		&task.WinnerBidID, &task.OrderID, &photosJSON, &rated, &task.Version, &createdAt,
	)
	if err != nil {
		return domain.Task{}, err
	}
	if err := json.Unmarshal([]byte(petJSON), &task.Pet); err != nil {
		return domain.Task{}, fmt.Errorf("decode pet snapshot: %w", err)
	}
	task.Services = decodeStrings(servicesJSON)
	task.Checklist = decodeStrings(checklistJSON)
	task.Photos = decodeStrings(photosJSON)
	task.StartAt = parseTimestamp(startAt)
	task.BidDeadlineAt = parseTimestamp(bidDeadline)
	task.CreatedAt = parseTimestamp(createdAt)
	task.PayDeadlineAt = nullableTimestamp(payDeadline)
	task.Rated = rated == 1
	return task, nil
}

func (s *Store) taskByID(taskID string) (domain.Task, error) {
	task, err := scanTask(s.db.QueryRow(`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, taskID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, notFound("任务不存在")
	}
	return task, err
}

func (s *Store) queryTasks(query string, args ...any) ([]domain.Task, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := []domain.Task{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (s *Store) OwnerHome(ownerID string) (domain.OwnerHome, error) {
	user, err := s.userByID(ownerID)
	if err != nil {
		return domain.OwnerHome{}, err
	}
	profile, err := s.profileFromUser(user)
	if err != nil {
		return domain.OwnerHome{}, err
	}
	address, err := s.defaultAddress(ownerID)
	if err != nil {
		return domain.OwnerHome{}, err
	}
	pets, err := s.Pets(ownerID)
	if err != nil {
		return domain.OwnerHome{}, err
	}
	tasks, err := s.queryTasks(`
		SELECT `+taskColumns+` FROM tasks
		WHERE owner_id = ? AND status NOT IN ('COMPLETED', 'SETTLED', 'CANCELLED', 'EXPIRED')
		ORDER BY created_at DESC
	`, ownerID)
	if err != nil {
		return domain.OwnerHome{}, err
	}
	now := time.Now()
	taskViews := make([]domain.TaskView, 0, len(tasks))
	for _, task := range tasks {
		view, err := s.taskView(task, ownerID, true, false, now)
		if err != nil {
			return domain.OwnerHome{}, err
		}
		taskViews = append(taskViews, view)
	}
	summary, err := s.communitySummary(ownerID, 2)
	if err != nil {
		return domain.OwnerHome{}, err
	}
	return domain.OwnerHome{
		Profile: profile,
		Area:    domain.Area{Name: "马连洼街道", District: "海淀区", Enabled: true},
		Tasks:   taskViews, Pets: pets, Address: address,
		Steps: []domain.Step{
			{Title: "发布任务", Description: "设预算与时间", Icon: "edit"},
			{Title: "学生报价", Description: "择优选人", Icon: "bid"},
			{Title: "付款预约", Description: "10分钟内支付", Icon: "pay"},
			{Title: "凭证验收", Description: "照片+清单", Icon: "camera"},
		},
		Guarantees: []domain.Guarantee{
			{Title: "实名+在校双重核验", Description: "仅展示“已核验”，不展示证件信息", Icon: "shield"},
			{Title: "成交前隐藏门牌", Description: "付款后仅向中选学生短期开放", Icon: "eye"},
			{Title: "争议冻结结算", Description: "异常上报后人工客服介入", Icon: "snow"},
		},
		Community: summary,
	}, nil
}

func (s *Store) OwnerTasks(ownerID, filter string) ([]domain.TaskView, error) {
	if _, err := s.userByID(ownerID); err != nil {
		return nil, err
	}
	condition := ""
	switch filter {
	case "", "all":
	case "bidding":
		condition = "AND status IN ('BIDDING', 'PENDING_PAYMENT')"
	case "fulfilling":
		condition = "AND status IN ('BOOKED', 'IN_SERVICE', 'PENDING_CONFIRM')"
	case "completed":
		condition = "AND status IN ('COMPLETED', 'SETTLED')"
	default:
		return nil, validation("订单筛选条件无效")
	}
	tasks, err := s.queryTasks(`SELECT `+taskColumns+` FROM tasks WHERE owner_id = ? `+condition+` ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]domain.TaskView, 0, len(tasks))
	for _, task := range tasks {
		view, err := s.taskView(task, ownerID, true, false, now)
		if err != nil {
			return nil, err
		}
		out = append(out, view)
	}
	return out, nil
}

func (s *Store) Task(taskID, viewerID, role string) (domain.TaskView, error) {
	task, err := s.taskByID(taskID)
	if err != nil {
		return domain.TaskView{}, err
	}
	now := time.Now()
	if role == "owner" {
		if task.OwnerID != viewerID {
			return domain.TaskView{}, forbidden("无权查看该任务")
		}
		return s.taskView(task, viewerID, true, true, now)
	}
	viewer, err := s.userByID(viewerID)
	if err != nil {
		return domain.TaskView{}, err
	}
	if task.Status == domain.TaskBidding {
		var hasBid int
		if err := s.db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM bids WHERE task_id = ? AND student_id = ? AND status IN ('ACTIVE', 'WON')
			)
		`, task.ID, viewerID).Scan(&hasBid); err != nil {
			return domain.TaskView{}, err
		}
		if hasBid == 0 {
			studentPoint := geo.Point{Latitude: viewer.ServiceLatitude, Longitude: viewer.ServiceLongitude}
			if !geo.Valid(studentPoint) {
				return domain.TaskView{}, locationRequired("请先开启定位，系统只会展示 5 公里内的任务")
			}
			taskPoint := geo.Point{Latitude: task.Latitude, Longitude: task.Longitude}
			radius := viewer.ServiceRadiusKM
			if radius <= 0 || radius > geo.MaxDispatchRadiusKM {
				radius = geo.DefaultDispatchRadius
			}
			if !geo.Valid(taskPoint) || geo.DistanceKM(studentPoint, taskPoint) > radius {
				return domain.TaskView{}, forbidden("任务距离超过接单范围，仅可查看 5 公里内任务")
			}
		}
	}
	view, err := s.taskView(task, viewerID, false, true, now)
	if err != nil {
		return domain.TaskView{}, err
	}
	setTaskDistance(&view, task, viewer)
	return view, nil
}

func (s *Store) AvailableTasks(studentID string) ([]domain.TaskView, error) {
	student, err := s.userByID(studentID)
	if err != nil {
		return nil, err
	}
	studentPoint := geo.Point{Latitude: student.ServiceLatitude, Longitude: student.ServiceLongitude}
	if !geo.Valid(studentPoint) {
		return nil, locationRequired("请先开启定位，系统只会展示 5 公里内的任务")
	}
	radius := student.ServiceRadiusKM
	if radius <= 0 || radius > geo.MaxDispatchRadiusKM {
		radius = geo.DefaultDispatchRadius
	}
	now := time.Now()
	tasks, err := s.queryTasks(`
		SELECT `+taskColumns+` FROM tasks
		WHERE status = 'BIDDING' AND bid_deadline_at > ?
		ORDER BY start_at ASC, created_at DESC
	`, timestamp(now))
	if err != nil {
		return nil, err
	}
	out := make([]domain.TaskView, 0, len(tasks))
	for _, task := range tasks {
		taskPoint := geo.Point{Latitude: task.Latitude, Longitude: task.Longitude}
		if !geo.Valid(taskPoint) {
			continue
		}
		distance := geo.DistanceKM(studentPoint, taskPoint)
		if distance > radius {
			continue
		}
		view, err := s.taskView(task, studentID, false, false, now)
		if err != nil {
			return nil, err
		}
		view.DistanceKM = geo.RoundKM(distance)
		out = append(out, view)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DistanceKM == out[j].DistanceKM {
			return out[i].StartAt.Before(out[j].StartAt)
		}
		return out[i].DistanceKM < out[j].DistanceKM
	})
	return out, nil
}

func (s *Store) StudentTasks(studentID string) ([]domain.TaskView, error) {
	if _, err := s.userByID(studentID); err != nil {
		return nil, err
	}
	now := time.Now()
	tasks, err := s.queryTasks(`
		SELECT `+taskColumns+` FROM tasks
		WHERE id IN (SELECT task_id FROM bids WHERE student_id = ?)
		ORDER BY start_at ASC
	`, studentID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TaskView, 0, len(tasks))
	for _, task := range tasks {
		view, err := s.taskView(task, studentID, false, true, now)
		if err != nil {
			return nil, err
		}
		out = append(out, view)
	}
	return out, nil
}

func (s *Store) Order(orderID, viewerID, role string) (domain.TaskView, error) {
	var taskID string
	err := s.db.QueryRow(`SELECT task_id FROM orders WHERE id = ?`, orderID).Scan(&taskID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TaskView{}, notFound("订单不存在")
	}
	if err != nil {
		return domain.TaskView{}, err
	}
	task, err := s.taskByID(taskID)
	if err != nil {
		return domain.TaskView{}, err
	}
	if role == "owner" {
		if task.OwnerID != viewerID {
			return domain.TaskView{}, forbidden("无权查看该订单")
		}
		return s.taskViewWithOrder(task, orderID, viewerID, true, true, time.Now())
	}
	var studentID string
	if err := s.db.QueryRow(`SELECT student_id FROM orders WHERE id = ?`, orderID).Scan(&studentID); err != nil {
		return domain.TaskView{}, err
	}
	if studentID != viewerID {
		return domain.TaskView{}, forbidden("无权查看该订单")
	}
	return s.taskViewWithOrder(task, orderID, viewerID, false, true, time.Now())
}

func (s *Store) StudentEarnings(studentID string) (domain.Earnings, error) {
	if _, err := s.userByID(studentID); err != nil {
		return domain.Earnings{}, err
	}
	rows, err := s.db.Query(`
		SELECT o.id, o.task_id, t.pet_snapshot, o.amount_fen, o.fee_fen, o.student_fen,
		       o.status, t.service_date_text, t.time_window, o.completed_at
		FROM orders o JOIN tasks t ON t.id = o.task_id
		WHERE o.student_id = ?
		ORDER BY o.created_at DESC
	`, studentID)
	if err != nil {
		return domain.Earnings{}, err
	}
	defer rows.Close()
	result := domain.Earnings{StudentID: studentID, Records: []domain.EarningRecord{}}
	for rows.Next() {
		var record domain.EarningRecord
		var petJSON string
		var completedAt sql.NullString
		if err := rows.Scan(&record.OrderID, &record.TaskID, &petJSON, &record.AmountFen,
			&record.FeeFen, &record.StudentFen, &record.Status, &record.ServiceDate,
			&record.TimeWindow, &completedAt); err != nil {
			return domain.Earnings{}, err
		}
		var pet domain.PetSnapshot
		if err := json.Unmarshal([]byte(petJSON), &pet); err != nil {
			return domain.Earnings{}, err
		}
		record.PetName = pet.Name
		record.CompletedAt = nullableTimestamp(completedAt)
		switch record.Status {
		case domain.OrderPaid, domain.OrderInService, domain.OrderPendingConfirm:
			result.PendingFen += record.StudentFen
		case domain.OrderCompleted:
			result.AvailableFen += record.StudentFen
			result.CompletedOrders++
		default:
			continue
		}
		result.TotalFen += record.StudentFen
		result.Records = append(result.Records, record)
	}
	return result, rows.Err()
}

func (s *Store) taskView(task domain.Task, viewerID string, ownerView, detail bool, now time.Time) (domain.TaskView, error) {
	return s.taskViewWithOrder(task, task.OrderID, viewerID, ownerView, detail, now)
}

func (s *Store) taskViewWithOrder(task domain.Task, viewOrderID, viewerID string, ownerView, detail bool, now time.Time) (domain.TaskView, error) {
	view := domain.TaskView{
		ID: task.ID, OwnerID: task.OwnerID, Pet: task.Pet, Mode: task.Mode, Status: task.Status,
		BudgetFen: task.BudgetFen, PriceFen: task.PriceFen, ServiceDate: task.ServiceDate,
		ServiceDateText: task.ServiceDateText, TimeWindow: task.TimeWindow, StartAt: task.StartAt,
		Services: append([]string(nil), task.Services...), Community: task.Community,
		LocationName:  task.LocationName,
		AddressMasked: task.AddressMasked, Note: task.Note, Checklist: append([]string(nil), task.Checklist...),
		BidDeadlineAt: task.BidDeadlineAt, PayDeadlineAt: task.PayDeadlineAt, WinnerBidID: task.WinnerBidID,
		OrderID: task.OrderID, Photos: append([]string(nil), task.Photos...), Rated: task.Rated,
		Version: task.Version, CreatedAt: task.CreatedAt,
	}
	if task.BidDeadlineAt.After(now) {
		view.RemainingSec = int64(task.BidDeadlineAt.Sub(now).Seconds())
	}
	if task.PayDeadlineAt != nil && task.PayDeadlineAt.After(now) {
		view.PayRemainingSec = int64(task.PayDeadlineAt.Sub(now).Seconds())
	}
	if ownerView {
		view.AddressDetail = task.AddressDetail
		view.LocationName = task.LocationName
	} else {
		view.LocationName = ""
	}
	order, err := s.orderByTaskID(task.ID, viewOrderID)
	if err != nil {
		return domain.TaskView{}, err
	}
	if order != nil {
		view.Order = order
		refund, err := s.latestRefundByOrder(order.ID)
		if err != nil {
			return domain.TaskView{}, err
		}
		view.Refund = refund
		if !ownerView && order.StudentID == viewerID &&
			order.Status != domain.OrderUnpaid && order.Status != domain.OrderCancelled &&
			order.Status != domain.OrderRefunding && order.Status != domain.OrderRefunded {
			view.AddressDetail = task.AddressDetail
			view.LocationName = task.LocationName
		}
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM bids WHERE task_id = ? AND status = 'ACTIVE'`, task.ID).Scan(&view.BidCount); err != nil {
		return domain.TaskView{}, err
	}
	rows, err := s.db.Query(`
		SELECT b.id, b.task_id, b.student_id, u.name, u.avatar, u.student_verified,
		       u.rating, u.done_count, u.school, u.service_latitude, u.service_longitude,
		       b.price_fen, b.message, b.revision,
		       b.status, b.created_at
		FROM bids b JOIN users u ON u.id = b.student_id
		WHERE b.task_id = ?
		ORDER BY b.price_fen ASC, b.created_at ASC
	`, task.ID)
	if err != nil {
		return domain.TaskView{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var bid domain.BidView
		var verified int
		var studentLatitude, studentLongitude float64
		var createdAt string
		if err := rows.Scan(&bid.ID, &bid.TaskID, &bid.Student.ID, &bid.Student.Name, &bid.Student.Avatar,
			&verified, &bid.Student.Rating, &bid.Student.DoneCount, &bid.Student.School,
			&studentLatitude, &studentLongitude,
			&bid.PriceFen, &bid.Message, &bid.Revision, &bid.Status, &createdAt); err != nil {
			return domain.TaskView{}, err
		}
		bid.Student.Verified = verified == 1
		bid.Student.Verification = "实名+在校核验"
		taskPoint := geo.Point{Latitude: task.Latitude, Longitude: task.Longitude}
		studentPoint := geo.Point{Latitude: studentLatitude, Longitude: studentLongitude}
		if geo.Valid(taskPoint) && geo.Valid(studentPoint) {
			bid.Student.DistanceKM = geo.RoundKM(geo.DistanceKM(taskPoint, studentPoint))
		}
		bid.CreatedAt = parseTimestamp(createdAt)
		if ownerView && detail {
			view.Bids = append(view.Bids, bid)
		} else if !ownerView && bid.Student.ID == viewerID && (bid.Status == domain.BidActive || bid.Status == domain.BidWon) {
			copy := bid
			view.MyBid = &copy
		}
		if bid.ID == task.WinnerBidID {
			copy := bid.Student
			view.Winner = &copy
		}
	}
	if err := rows.Err(); err != nil {
		return domain.TaskView{}, err
	}
	sort.Slice(view.Bids, func(i, j int) bool {
		if view.Bids[i].PriceFen == view.Bids[j].PriceFen {
			return view.Bids[i].CreatedAt.Before(view.Bids[j].CreatedAt)
		}
		return view.Bids[i].PriceFen < view.Bids[j].PriceFen
	})
	if order != nil && order.ID != task.OrderID {
		view.Status = taskStatusForOrder(order.Status, task.Status)
		view.PriceFen = order.AmountFen
		view.OrderID = order.ID
		view.WinnerBidID = order.BidID
		view.PayRemainingSec = 0
		student, err := s.StudentProfile(order.StudentID)
		if err != nil {
			return domain.TaskView{}, err
		}
		view.Winner = &student
	}
	return view, nil
}

func setTaskDistance(view *domain.TaskView, task domain.Task, student dbUser) {
	taskPoint := geo.Point{Latitude: task.Latitude, Longitude: task.Longitude}
	studentPoint := geo.Point{Latitude: student.ServiceLatitude, Longitude: student.ServiceLongitude}
	if geo.Valid(taskPoint) && geo.Valid(studentPoint) {
		view.DistanceKM = geo.RoundKM(geo.DistanceKM(studentPoint, taskPoint))
	}
}

func taskStatusForOrder(orderStatus, fallback string) string {
	switch orderStatus {
	case domain.OrderUnpaid:
		return domain.TaskPendingPayment
	case domain.OrderPaid:
		return domain.TaskBooked
	case domain.OrderInService:
		return domain.TaskInService
	case domain.OrderPendingConfirm:
		return domain.TaskPendingConfirm
	case domain.OrderCompleted:
		return domain.TaskCompleted
	case domain.OrderRefunding:
		return domain.TaskRefunding
	case domain.OrderCancelled, domain.OrderRefunded:
		return domain.TaskCancelled
	default:
		return fallback
	}
}

func (s *Store) orderByTaskID(taskID, orderID string) (*domain.Order, error) {
	if orderID == "" {
		return nil, nil
	}
	var order domain.Order
	var createdAt string
	var paymentStatus, outTradeNo string
	var paidAt, checkInAt, completedAt, confirmedAt sql.NullString
	err := s.db.QueryRow(`
		SELECT o.id, o.task_id, o.bid_id, o.owner_id, o.student_id, o.amount_fen, o.fee_fen, o.student_fen,
		       o.status, o.payment_key, COALESCE(p.status, ''), COALESCE(p.out_trade_no, ''),
		       o.created_at, o.paid_at, o.check_in_at, o.completed_at, o.confirmed_at, o.rating_stars
		FROM orders o LEFT JOIN payments p ON p.order_id = o.id
		WHERE o.task_id = ? AND o.id = ?
	`, taskID, orderID).Scan(
		&order.ID, &order.TaskID, &order.BidID, &order.OwnerID, &order.StudentID,
		&order.AmountFen, &order.FeeFen, &order.StudentFen, &order.Status, &order.PaymentKey,
		&paymentStatus, &outTradeNo,
		&createdAt, &paidAt, &checkInAt, &completedAt, &confirmedAt, &order.RatingStars,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	order.CreatedAt = parseTimestamp(createdAt)
	order.PaymentStatus = paymentStatus
	order.OutTradeNo = outTradeNo
	order.PaidAt = nullableTimestamp(paidAt)
	order.CheckInAt = nullableTimestamp(checkInAt)
	order.CompletedAt = nullableTimestamp(completedAt)
	order.ConfirmedAt = nullableTimestamp(confirmedAt)
	return &order, nil
}
