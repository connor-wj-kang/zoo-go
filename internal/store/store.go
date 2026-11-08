package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"zoo-go/internal/domain"
	"zoo-go/internal/geo"
)

const (
	feeRateBasisPoints = int64(1000)
	budgetMinFen       = int64(3000)
	budgetMaxFen       = int64(20000)
	minBidFen          = int64(2000)
	paymentTimeout     = 10 * time.Minute
	sessionTTL         = 30 * 24 * time.Hour
)

var (
	serviceTypes = []string{"喂食", "换水", "清理猫砂", "清理生活区", "拍照反馈"}
	timeWindows  = []string{"09:00-10:00", "12:00-13:00", "15:00-16:00", "18:00-19:00", "20:00-21:00"}
)

type Store struct {
	db   *sql.DB
	path string
}

type dbUser struct {
	ID                    string
	OpenID                string
	Name                  string
	Avatar                string
	PhoneMask             string
	OwnerVerified         bool
	StudentVerified       bool
	StudentStatus         string
	School                string
	VerificationExpiresAt string
	Rating                float64
	DoneCount             int
	ServiceRadiusKM       float64
	ServiceLatitude       float64
	ServiceLongitude      float64
	LocationUpdatedAt     string
}

func New(path string) (*Store, error) {
	dsn := path
	if path != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	} else {
		dsn = ":memory:"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	if path != "" {
		if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
			db.Close()
			return nil, fmt.Errorf("enable WAL: %w", err)
		}
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	store := &Store{db: db, path: path}
	if err := store.migrateStudentVerificationSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate student verification: %w", err)
	}
	if err := store.migrateAdminSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate admin schema: %w", err)
	}
	if err := store.migrateLocationSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate location schema: %w", err)
	}
	if err := store.migrateOrdersTaskUniqueness(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate orders: %w", err)
	}
	if err := store.ensureActiveOrderIndex(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ensure active order index: %w", err)
	}
	if err := store.rebuildStudentSlots(); err != nil {
		db.Close()
		return nil, fmt.Errorf("rebuild student slots: %w", err)
	}
	now := timestamp(time.Now())
	for _, area := range defaultServiceAreas {
		if _, err := db.Exec(`
			INSERT INTO service_areas(id, name, district, enabled, sort_order, created_at)
			VALUES(?, ?, ?, 1, ?, ?)
			ON CONFLICT(id) DO NOTHING
		`, area.ID, area.Name, area.District, area.Order, now); err != nil {
			db.Close()
			return nil, fmt.Errorf("configure service area: %w", err)
		}
	}
	return store, nil
}

func (s *Store) ensureActiveOrderIndex() error {
	var definition string
	err := s.db.QueryRow(`SELECT COALESCE(sql, '') FROM sqlite_master WHERE type = 'index' AND name = 'ux_orders_active_task'`).Scan(&definition)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if definition != "" && !strings.Contains(definition, "'REFUNDING'") && strings.Contains(definition, "'PENDING_CONFIRM'") {
		return nil
	}
	if _, err := s.db.Exec(`DROP INDEX IF EXISTS ux_orders_active_task`); err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE UNIQUE INDEX ux_orders_active_task
		ON orders(task_id) WHERE status IN ('UNPAID', 'PAID', 'IN_SERVICE', 'PENDING_CONFIRM')
	`)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

func newID(prefix string) string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		panic(fmt.Sprintf("secure random id: %v", err))
	}
	return prefix + "_" + hex.EncodeToString(raw)
}

func newToken() string {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		panic(fmt.Sprintf("secure random token: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func timestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTimestamp(value string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, value)
	return t
}

func nullableTimestamp(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	t := parseTimestamp(value.String)
	return &t
}

func marshalJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func decodeStrings(value string) []string {
	out := []string{}
	if strings.TrimSpace(value) == "" {
		return out
	}
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return []string{}
	}
	return out
}

func scanUser(scanner interface{ Scan(...any) error }) (dbUser, error) {
	var user dbUser
	var ownerVerified, studentVerified int
	err := scanner.Scan(
		&user.ID, &user.OpenID, &user.Name, &user.Avatar, &user.PhoneMask,
		&ownerVerified, &studentVerified, &user.StudentStatus, &user.School,
		&user.VerificationExpiresAt, &user.Rating, &user.DoneCount, &user.ServiceRadiusKM,
		&user.ServiceLatitude, &user.ServiceLongitude, &user.LocationUpdatedAt,
	)
	user.OwnerVerified = ownerVerified == 1
	user.StudentVerified = studentVerified == 1
	return user, err
}

func (s *Store) userByID(userID string) (dbUser, error) {
	row := s.db.QueryRow(`
		SELECT id, open_id, name, avatar, phone_mask, owner_verified, student_verified,
		       student_status, school, verification_expires_at, rating, done_count, service_radius_km,
		       service_latitude, service_longitude, location_updated_at
		FROM users WHERE id = ?
	`, userID)
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return dbUser{}, notFound("用户不存在")
	}
	return user, err
}

func (s *Store) profileFromUser(user dbUser) (domain.Profile, error) {
	var completed int
	if err := s.db.QueryRow(`
		SELECT COUNT(*) FROM tasks WHERE owner_id = ? AND status IN ('COMPLETED', 'SETTLED')
	`, user.ID).Scan(&completed); err != nil {
		return domain.Profile{}, err
	}
	return domain.Profile{
		ID: user.ID, Name: user.Name, Avatar: user.Avatar, PhoneMask: user.PhoneMask,
		Verified: user.OwnerVerified, CompletedOrders: completed,
		StudentVerified: user.StudentVerified, StudentStatus: user.StudentStatus,
		School: user.School, Rating: user.Rating, DoneCount: user.DoneCount,
		ServiceRadiusKM: user.ServiceRadiusKM,
	}, nil
}

func (s *Store) Profile(userID string) (domain.Profile, error) {
	user, err := s.userByID(userID)
	if err != nil {
		return domain.Profile{}, err
	}
	return s.profileFromUser(user)
}

func (s *Store) UpdateProfile(userID string, req domain.UpdateProfileRequest, now time.Time) (domain.Profile, error) {
	if _, err := s.userByID(userID); err != nil {
		return domain.Profile{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.Profile{}, validation("昵称不能为空")
	}
	if len([]rune(name)) > 24 {
		return domain.Profile{}, validation("昵称不能超过 24 字")
	}
	avatar := strings.TrimSpace(req.Avatar)
	if avatar != "" && !strings.HasPrefix(avatar, "/uploads/") && !strings.HasPrefix(avatar, "https://") {
		return domain.Profile{}, validation("头像必须先上传到平台")
	}
	if _, err := s.db.Exec(`UPDATE users SET name = ?, avatar = ?, updated_at = ? WHERE id = ?`,
		name, avatar, timestamp(now), userID); err != nil {
		return domain.Profile{}, err
	}
	return s.Profile(userID)
}

func (s *Store) StudentProfile(studentID string) (domain.Student, error) {
	return s.StudentProfileAt(studentID, time.Now())
}

func (s *Store) StudentProfileAt(studentID string, now time.Time) (domain.Student, error) {
	if err := s.expireStudentVerifications(now); err != nil {
		return domain.Student{}, err
	}
	user, err := s.userByID(studentID)
	if err != nil {
		return domain.Student{}, err
	}
	verification := "待提交学籍认证"
	switch user.StudentStatus {
	case domain.StudentPending:
		verification = "学籍审核中"
	case domain.StudentVerified:
		verification = "姓名+在校学籍核验"
	case domain.StudentRejected:
		verification = "学籍认证未通过"
	case domain.StudentExpired:
		verification = "学籍认证已过期"
	}
	var reviewNote string
	err = s.db.QueryRow(`
		SELECT review_note FROM student_verifications
		WHERE user_id = ? ORDER BY submitted_at DESC, id DESC LIMIT 1
	`, studentID).Scan(&reviewNote)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.Student{}, err
	}
	var expiresAt *time.Time
	if user.VerificationExpiresAt != "" {
		parsed := parseTimestamp(user.VerificationExpiresAt)
		expiresAt = &parsed
	}
	return domain.Student{
		ID: user.ID, Name: user.Name, Avatar: user.Avatar, Rating: user.Rating,
		DoneCount: user.DoneCount, Verified: user.StudentVerified, Verification: verification,
		VerificationStatus: user.StudentStatus, VerificationNote: reviewNote,
		VerificationExpiresAt: expiresAt, School: user.School, ServiceRadius: user.ServiceRadiusKM,
		LocationEnabled: geo.Valid(geo.Point{Latitude: user.ServiceLatitude, Longitude: user.ServiceLongitude}),
	}, nil
}

func (s *Store) UpdateStudentLocation(userID string, req domain.UpdateLocationRequest, now time.Time) (domain.StudentLocation, error) {
	if _, err := s.userByID(userID); err != nil {
		return domain.StudentLocation{}, err
	}
	point := geo.Point{Latitude: req.Latitude, Longitude: req.Longitude}
	if !geo.Valid(point) {
		return domain.StudentLocation{}, validation("定位坐标无效，请重新选择当前位置")
	}
	radius := req.RadiusKM
	if radius == 0 {
		radius = geo.DefaultDispatchRadius
	}
	if radius < 1 || radius > geo.MaxDispatchRadiusKM {
		return domain.StudentLocation{}, validation("接单范围必须在 1–5 公里之间")
	}
	if _, err := s.db.Exec(`
		UPDATE users SET service_latitude = ?, service_longitude = ?, service_radius_km = ?,
		                 location_updated_at = ?, updated_at = ?
		WHERE id = ?
	`, point.Latitude, point.Longitude, radius, timestamp(now), timestamp(now), userID); err != nil {
		return domain.StudentLocation{}, err
	}
	return domain.StudentLocation{
		Latitude: point.Latitude, Longitude: point.Longitude, RadiusKM: radius,
		UpdatedAt: now, LocationEnabled: true,
	}, nil
}

func (s *Store) Pets(ownerID string) ([]domain.Pet, error) {
	if _, err := s.userByID(ownerID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`
		SELECT id, owner_id, name, species, breed, age, image, note, flags_json
		FROM pets WHERE owner_id = ? ORDER BY created_at ASC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pets := []domain.Pet{}
	for rows.Next() {
		var pet domain.Pet
		var flags string
		if err := rows.Scan(&pet.ID, &pet.OwnerID, &pet.Name, &pet.Species, &pet.Breed, &pet.Age, &pet.Image, &pet.Note, &flags); err != nil {
			return nil, err
		}
		pet.Flags = decodeStrings(flags)
		pets = append(pets, pet)
	}
	return pets, rows.Err()
}

func (s *Store) CreatePet(ownerID string, req domain.CreatePetRequest, now time.Time) (domain.Pet, error) {
	if _, err := s.userByID(ownerID); err != nil {
		return domain.Pet{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 30 {
		return domain.Pet{}, validation("宠物名称不能为空且不能超过 30 字")
	}
	if req.Species != "猫" && req.Species != "狗" {
		return domain.Pet{}, validation("目前仅支持猫、狗")
	}
	if len([]rune(req.Note)) > 300 {
		return domain.Pet{}, validation("宠物说明不能超过 300 字")
	}
	image := strings.TrimSpace(req.Image)
	if image == "" {
		if req.Species == "猫" {
			image = "/assets/pets/cat.jpg"
		} else {
			image = "/assets/pets/dog.jpg"
		}
	}
	pet := domain.Pet{
		ID: newID("pet"), OwnerID: ownerID, Name: name, Species: req.Species,
		Breed: strings.TrimSpace(req.Breed), Age: strings.TrimSpace(req.Age), Image: image,
		Note: strings.TrimSpace(req.Note), Flags: nonEmptyStrings(req.Flags),
	}
	nowText := timestamp(now)
	if _, err := s.db.Exec(`
		INSERT INTO pets(id, owner_id, name, species, breed, age, image, note, flags_json, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, pet.ID, pet.OwnerID, pet.Name, pet.Species, pet.Breed, pet.Age, pet.Image, pet.Note, marshalJSON(pet.Flags), nowText, nowText); err != nil {
		return domain.Pet{}, err
	}
	return pet, nil
}

func (s *Store) Login(openID, nickname, avatar, phoneMask string, now time.Time) (domain.Session, error) {
	openID = strings.TrimSpace(openID)
	if openID == "" {
		return domain.Session{}, validation("登录凭证无效")
	}
	if err := s.expireStudentVerifications(now); err != nil {
		return domain.Session{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Session{}, err
	}
	defer tx.Rollback()
	nowText := timestamp(now)
	if _, err := tx.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, nowText); err != nil {
		return domain.Session{}, err
	}

	var user dbUser
	row := tx.QueryRow(`
		SELECT id, open_id, name, avatar, phone_mask, owner_verified, student_verified,
		       student_status, school, verification_expires_at, rating, done_count, service_radius_km,
		       service_latitude, service_longitude, location_updated_at
		FROM users WHERE open_id = ?
	`, openID)
	user, err = scanUser(row)
	isNew := errors.Is(err, sql.ErrNoRows)
	if isNew {
		user = dbUser{
			ID: newID("u"), OpenID: openID, Name: defaultName(nickname), Avatar: strings.TrimSpace(avatar),
			PhoneMask: strings.TrimSpace(phoneMask), StudentStatus: "UNVERIFIED",
			ServiceRadiusKM: geo.DefaultDispatchRadius,
		}
		_, err = tx.Exec(`
			INSERT INTO users(id, open_id, name, avatar, phone_mask, owner_verified, student_verified,
			                  student_status, school, rating, done_count, service_radius_km, created_at, updated_at)
			VALUES(?, ?, ?, ?, ?, 0, 0, 'UNVERIFIED', '', 0, 0, ?, ?, ?)
		`, user.ID, user.OpenID, user.Name, user.Avatar, user.PhoneMask, user.ServiceRadiusKM, nowText, nowText)
		if err != nil {
			return domain.Session{}, err
		}
	} else if err != nil {
		return domain.Session{}, err
	} else {
		var accountStatus string
		if err := tx.QueryRow(`SELECT account_status FROM users WHERE id = ?`, user.ID).Scan(&accountStatus); err != nil {
			return domain.Session{}, err
		}
		if accountStatus == domain.AccountDisabled {
			return domain.Session{}, forbidden("账号已被停用，请联系平台客服")
		}
		if nickname = strings.TrimSpace(nickname); nickname != "" {
			user.Name = nickname
		}
		if avatar = strings.TrimSpace(avatar); avatar != "" {
			user.Avatar = avatar
		}
		if phoneMask = strings.TrimSpace(phoneMask); phoneMask != "" {
			user.PhoneMask = phoneMask
		}
		if _, err := tx.Exec(`UPDATE users SET name = ?, avatar = ?, phone_mask = ?, updated_at = ? WHERE id = ?`,
			user.Name, user.Avatar, user.PhoneMask, nowText, user.ID); err != nil {
			return domain.Session{}, err
		}
	}

	token := newToken()
	expiresAt := now.Add(sessionTTL)
	if _, err := tx.Exec(`
		INSERT INTO sessions(token_hash, user_id, expires_at, created_at) VALUES(?, ?, ?, ?)
	`, hashToken(token), user.ID, timestamp(expiresAt), nowText); err != nil {
		return domain.Session{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Session{}, err
	}
	profile, err := s.Profile(user.ID)
	if err != nil {
		return domain.Session{}, err
	}
	return domain.Session{Token: token, Profile: profile, ExpiresAt: expiresAt, IsNewUser: isNew}, nil
}

func defaultName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "喂宠用户"
	}
	runes := []rune(value)
	if len(runes) > 24 {
		runes = runes[:24]
	}
	return string(runes)
}

func (s *Store) Authenticate(token string, now time.Time) (domain.Profile, error) {
	if strings.TrimSpace(token) == "" {
		return domain.Profile{}, &AppError{Status: 401, Code: "UNAUTHORIZED", Message: "请先登录"}
	}
	var userID string
	err := s.db.QueryRow(`
		SELECT user_id FROM sessions WHERE token_hash = ? AND expires_at > ?
	`, hashToken(token), timestamp(now)).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Profile{}, &AppError{Status: 401, Code: "UNAUTHORIZED", Message: "登录已过期，请重新登录"}
	}
	if err != nil {
		return domain.Profile{}, err
	}
	if err := s.expireStudentVerifications(now); err != nil {
		return domain.Profile{}, err
	}
	profile, err := s.Profile(userID)
	if err != nil {
		return domain.Profile{}, err
	}
	status, _, err := s.accountStatus(userID)
	if err != nil {
		return domain.Profile{}, err
	}
	if status == domain.AccountDisabled {
		return domain.Profile{}, forbidden("账号已被停用，请联系平台客服")
	}
	return profile, nil
}

func (s *Store) Logout(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hashToken(token))
	return err
}

func (s *Store) serviceAreaNames() ([]string, error) {
	rows, err := s.db.Query(`SELECT name FROM service_areas WHERE enabled = 1 ORDER BY sort_order ASC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (s *Store) defaultAddress(ownerID string) (*domain.OwnerAddress, error) {
	var address domain.OwnerAddress
	err := s.db.QueryRow(`
		SELECT community, detail, location_name, latitude, longitude
		FROM owner_addresses WHERE owner_id = ?
	`, ownerID).Scan(&address.Community, &address.Detail, &address.LocationName, &address.Latitude, &address.Longitude)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &address, nil
}

func (s *Store) saveDefaultAddress(ownerID, community, detail string, now time.Time) error {
	_, err := s.db.Exec(`
		INSERT INTO owner_addresses(owner_id, community, detail, updated_at) VALUES(?, ?, ?, ?)
		ON CONFLICT(owner_id) DO UPDATE SET community = excluded.community, detail = excluded.detail, updated_at = excluded.updated_at
	`, ownerID, community, detail, timestamp(now))
	return err
}

func (s *Store) Meta(now time.Time) domain.Meta {
	weekdays := [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	days := make([]domain.DateOption, 0, 5)
	for i := 1; i <= 5; i++ {
		d := now.AddDate(0, 0, i)
		days = append(days, domain.DateOption{
			Value: d.Format("2006-01-02"),
			Label: fmt.Sprintf("%s %s", d.Format("01-02"), weekdays[d.Weekday()]),
		})
	}
	communities, _ := s.serviceAreaNames()
	return domain.Meta{
		ServiceTypes:       append([]string(nil), serviceTypes...),
		TimeWindows:        append([]string(nil), timeWindows...),
		Communities:        communities,
		Days:               days,
		Deadlines:          []domain.DeadlineOption{{Label: "15分钟", Minutes: 15}, {Label: "30分钟", Minutes: 30}, {Label: "60分钟", Minutes: 60}, {Label: "2小时", Minutes: 120}},
		FeeRateBasisPoints: feeRateBasisPoints, BudgetMinFen: budgetMinFen, BudgetMaxFen: budgetMaxFen,
		PaymentTimeoutMinute: int(paymentTimeout / time.Minute),
	}
}

func nonEmptyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
