package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"zoo-go/internal/domain"
)

func (s *Store) SubmitStudentVerification(userID string, req domain.SubmitStudentVerificationRequest, now time.Time) (domain.Student, error) {
	req.RealName = strings.TrimSpace(req.RealName)
	req.School = strings.TrimSpace(req.School)
	req.StudentNo = strings.TrimSpace(req.StudentNo)
	req.Major = strings.TrimSpace(req.Major)
	req.ChsiCode = strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(req.ChsiCode), " ", ""), "-", ""))
	if len([]rune(req.RealName)) < 2 || len([]rune(req.RealName)) > 40 {
		return domain.Student{}, validation("请填写真实姓名")
	}
	if req.School == "" || len([]rune(req.School)) > 80 {
		return domain.Student{}, validation("请填写有效学校名称")
	}
	if len([]rune(req.StudentNo)) < 4 || len([]rune(req.StudentNo)) > 40 || !verificationCodeCharacters(req.StudentNo, true) {
		return domain.Student{}, validation("请填写有效学号")
	}
	if len([]rune(req.Major)) > 80 {
		return domain.Student{}, validation("专业名称不能超过 80 字")
	}
	if req.EnrollmentYear < 1980 || req.EnrollmentYear > now.Year() {
		return domain.Student{}, validation("请选择有效的入学年份")
	}
	if len([]rune(req.ChsiCode)) < 6 || len([]rune(req.ChsiCode)) > 32 || !verificationCodeCharacters(req.ChsiCode, false) {
		return domain.Student{}, validation("请填写学信网在线验证码")
	}
	if err := s.expireStudentVerifications(now); err != nil {
		return domain.Student{}, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return domain.Student{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var user dbUser
	var ownerVerified, studentVerified int
	err = tx.QueryRow(`
		SELECT id, open_id, name, avatar, phone_mask, owner_verified, student_verified,
		       student_status, school, verification_expires_at, rating, done_count, service_radius_km
		FROM users WHERE id = ?
	`, userID).Scan(
		&user.ID, &user.OpenID, &user.Name, &user.Avatar, &user.PhoneMask,
		&ownerVerified, &studentVerified, &user.StudentStatus, &user.School,
		&user.VerificationExpiresAt, &user.Rating, &user.DoneCount, &user.ServiceRadiusKM,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Student{}, notFound("用户不存在")
	}
	if err != nil {
		return domain.Student{}, err
	}
	user.OwnerVerified = ownerVerified == 1
	user.StudentVerified = studentVerified == 1
	if user.StudentStatus == domain.StudentPending {
		return domain.Student{}, conflict("学籍资料正在审核，请勿重复提交")
	}
	if studentVerificationActive(user, now) {
		return domain.Student{}, conflict("学籍认证仍在有效期内，无需重复提交")
	}

	nowText := timestamp(now)
	applicationID := newID("sv")
	_, err = tx.Exec(`
		INSERT INTO student_verifications(
			id, user_id, real_name, school, student_no, major, enrollment_year,
			chsi_code, chsi_code_hash, status, submitted_at, created_at, updated_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?, ?)
	`, applicationID, userID, req.RealName, req.School, req.StudentNo, req.Major,
		req.EnrollmentYear, req.ChsiCode, hashToken(req.ChsiCode), nowText, nowText, nowText)
	if err != nil {
		message := strings.ToLower(err.Error())
		if strings.Contains(message, "chsi_code_hash") {
			return domain.Student{}, conflict("该学信网验证码已被其他账号使用")
		}
		if strings.Contains(message, "unique") {
			return domain.Student{}, conflict("学籍资料正在审核，请勿重复提交")
		}
		return domain.Student{}, err
	}
	if _, err := tx.Exec(`
		UPDATE users SET school = ?, student_status = 'PENDING', student_verified = 0,
		                 verification_expires_at = '', updated_at = ?
		WHERE id = ?
	`, req.School, nowText, userID); err != nil {
		return domain.Student{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Student{}, err
	}
	return s.StudentProfileAt(userID, now)
}

func (s *Store) StudentVerificationApplications(status string) ([]domain.StudentVerificationApplication, error) {
	if err := s.expireStudentVerifications(time.Now()); err != nil {
		return nil, err
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "" {
		status = domain.StudentPending
	}
	if status != "ALL" && status != domain.StudentPending && status != domain.StudentVerified &&
		status != domain.StudentRejected && status != domain.StudentExpired {
		return nil, validation("学籍审核状态无效")
	}
	where := "WHERE v.status = ?"
	args := []any{status}
	if status == "ALL" {
		where = ""
		args = nil
	}
	rows, err := s.db.Query(`
		SELECT v.id, v.user_id, u.name, v.real_name, v.school, v.student_no, v.major,
		       v.enrollment_year, v.chsi_code, v.status, v.review_note,
		       v.submitted_at, v.reviewed_at, v.valid_until
		FROM student_verifications v JOIN users u ON u.id = v.user_id
		`+where+`
		ORDER BY v.submitted_at ASC, v.id ASC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applications := []domain.StudentVerificationApplication{}
	for rows.Next() {
		var application domain.StudentVerificationApplication
		var submittedAt string
		var reviewedAt, validUntil sql.NullString
		if err := rows.Scan(
			&application.ID, &application.UserID, &application.UserName,
			&application.RealName, &application.School, &application.StudentNo,
			&application.Major, &application.EnrollmentYear, &application.ChsiCode,
			&application.Status, &application.ReviewNote, &submittedAt, &reviewedAt, &validUntil,
		); err != nil {
			return nil, err
		}
		application.SubmittedAt = parseTimestamp(submittedAt)
		application.ReviewedAt = nullableTimestamp(reviewedAt)
		application.ValidUntil = nullableTimestamp(validUntil)
		applications = append(applications, application)
	}
	return applications, rows.Err()
}

func (s *Store) ApproveStudentVerification(applicationID, reviewer string, validUntil, now time.Time) error {
	applicationID = strings.TrimSpace(applicationID)
	reviewer = strings.TrimSpace(reviewer)
	if reviewer == "" {
		reviewer = "admin"
	}
	if validUntil.IsZero() {
		validUntil = now.AddDate(1, 0, 0)
	}
	validUntil = validUntil.UTC()
	if !validUntil.After(now) || validUntil.After(now.AddDate(5, 0, 0)) {
		return validation("认证有效期必须在未来 5 年内")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var userID, school, status string
	err = tx.QueryRow(`
		SELECT user_id, school, status FROM student_verifications WHERE id = ?
	`, applicationID).Scan(&userID, &school, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return notFound("学籍认证申请不存在")
	}
	if err != nil {
		return err
	}
	if status != domain.StudentPending {
		return conflict("该学籍认证申请已处理")
	}
	nowText := timestamp(now)
	validUntilText := timestamp(validUntil)
	result, err := tx.Exec(`
		UPDATE student_verifications SET status = 'VERIFIED', review_note = '', reviewed_by = ?,
		       reviewed_at = ?, valid_until = ?, updated_at = ?
		WHERE id = ? AND status = 'PENDING'
	`, reviewer, nowText, validUntilText, nowText, applicationID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return conflict("该学籍认证申请已处理")
	}
	if _, err := tx.Exec(`
		UPDATE users SET school = ?, student_status = 'VERIFIED', student_verified = 1,
		                 verification_expires_at = ?, updated_at = ?
		WHERE id = ?
	`, school, validUntilText, nowText, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) RejectStudentVerification(applicationID, reviewer, reason string, now time.Time) error {
	reason = strings.TrimSpace(reason)
	reviewer = strings.TrimSpace(reviewer)
	if reviewer == "" {
		reviewer = "admin"
	}
	if reason == "" || len([]rune(reason)) > 300 {
		return validation("请填写 300 字以内的驳回原因")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var userID, status string
	err = tx.QueryRow(`SELECT user_id, status FROM student_verifications WHERE id = ?`, applicationID).Scan(&userID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return notFound("学籍认证申请不存在")
	}
	if err != nil {
		return err
	}
	if status != domain.StudentPending {
		return conflict("该学籍认证申请已处理")
	}
	nowText := timestamp(now)
	result, err := tx.Exec(`
		UPDATE student_verifications SET status = 'REJECTED', review_note = ?, reviewed_by = ?,
		       reviewed_at = ?, updated_at = ?
		WHERE id = ? AND status = 'PENDING'
	`, reason, reviewer, nowText, nowText, applicationID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return conflict("该学籍认证申请已处理")
	}
	if _, err := tx.Exec(`
		UPDATE users SET student_status = 'REJECTED', student_verified = 0,
		                 verification_expires_at = '', updated_at = ?
		WHERE id = ?
	`, nowText, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ApproveStudent(userID string, now time.Time) error {
	var applicationID string
	err := s.db.QueryRow(`
		SELECT id FROM student_verifications
		WHERE user_id = ? AND status = 'PENDING'
		ORDER BY submitted_at DESC, id DESC LIMIT 1
	`, userID).Scan(&applicationID)
	if errors.Is(err, sql.ErrNoRows) {
		return conflict("该用户没有待审核的学籍认证")
	}
	if err != nil {
		return err
	}
	return s.ApproveStudentVerification(applicationID, "admin", now.AddDate(1, 0, 0), now)
}

func (s *Store) expireStudentVerifications(now time.Time) error {
	nowText := timestamp(now)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`
		UPDATE student_verifications SET status = 'EXPIRED', updated_at = ?
		WHERE status = 'VERIFIED' AND valid_until <> '' AND valid_until <= ?
	`, nowText, nowText); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE users SET student_verified = 0, student_status = 'EXPIRED', updated_at = ?
		WHERE student_verified = 1 AND verification_expires_at <> '' AND verification_expires_at <= ?
	`, nowText, nowText); err != nil {
		return err
	}
	return tx.Commit()
}

func studentVerificationActive(user dbUser, now time.Time) bool {
	if !user.StudentVerified || user.StudentStatus != domain.StudentVerified {
		return false
	}
	if user.VerificationExpiresAt == "" {
		return true
	}
	return parseTimestamp(user.VerificationExpiresAt).After(now)
}

func verificationCodeCharacters(value string, allowHyphen bool) bool {
	for _, char := range value {
		if char >= '0' && char <= '9' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		if char >= 'a' && char <= 'z' {
			continue
		}
		if allowHyphen && char == '-' {
			continue
		}
		return false
	}
	return true
}
