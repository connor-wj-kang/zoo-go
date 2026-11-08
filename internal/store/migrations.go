package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *Store) migrateStudentVerificationSchema() error {
	rows, err := s.db.Query(`PRAGMA table_info(users)`)
	if err != nil {
		return err
	}
	hasExpiry := false
	for rows.Next() {
		var cid, notNull, pk int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			rows.Close()
			return err
		}
		if name == "verification_expires_at" {
			hasExpiry = true
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	now := time.Now()
	nowText := timestamp(now)
	if !hasExpiry {
		if _, err := s.db.Exec(`ALTER TABLE users ADD COLUMN verification_expires_at TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`
			UPDATE users SET verification_expires_at = ?
			WHERE student_verified = 1 AND student_status = 'VERIFIED'
		`, timestamp(now.AddDate(1, 0, 0))); err != nil {
			return err
		}
	}
	_, err = s.db.Exec(`
		UPDATE users SET student_status = 'UNVERIFIED', student_verified = 0, updated_at = ?
		WHERE student_status = 'PENDING'
		  AND NOT EXISTS (SELECT 1 FROM student_verifications v WHERE v.user_id = users.id)
	`, nowText)
	return err
}

func (s *Store) migrateAdminSchema() error {
	columns := map[string]bool{}
	rows, err := s.db.Query(`PRAGMA table_info(users)`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var cid, notNull, pk int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			rows.Close()
			return err
		}
		columns[name] = true
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for name, definition := range map[string]string{
		"account_status":  `ALTER TABLE users ADD COLUMN account_status TEXT NOT NULL DEFAULT 'ACTIVE'`,
		"disabled_reason": `ALTER TABLE users ADD COLUMN disabled_reason TEXT NOT NULL DEFAULT ''`,
		"disabled_at":     `ALTER TABLE users ADD COLUMN disabled_at TEXT NOT NULL DEFAULT ''`,
	} {
		if columns[name] {
			continue
		}
		if _, err := s.db.Exec(definition); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) migrateLocationSchema() error {
	columns := []struct {
		table      string
		name       string
		definition string
	}{
		{"users", "service_latitude", `ALTER TABLE users ADD COLUMN service_latitude REAL NOT NULL DEFAULT 0`},
		{"users", "service_longitude", `ALTER TABLE users ADD COLUMN service_longitude REAL NOT NULL DEFAULT 0`},
		{"users", "location_updated_at", `ALTER TABLE users ADD COLUMN location_updated_at TEXT NOT NULL DEFAULT ''`},
		{"owner_addresses", "location_name", `ALTER TABLE owner_addresses ADD COLUMN location_name TEXT NOT NULL DEFAULT ''`},
		{"owner_addresses", "latitude", `ALTER TABLE owner_addresses ADD COLUMN latitude REAL NOT NULL DEFAULT 0`},
		{"owner_addresses", "longitude", `ALTER TABLE owner_addresses ADD COLUMN longitude REAL NOT NULL DEFAULT 0`},
		{"tasks", "location_name", `ALTER TABLE tasks ADD COLUMN location_name TEXT NOT NULL DEFAULT ''`},
		{"tasks", "latitude", `ALTER TABLE tasks ADD COLUMN latitude REAL NOT NULL DEFAULT 0`},
		{"tasks", "longitude", `ALTER TABLE tasks ADD COLUMN longitude REAL NOT NULL DEFAULT 0`},
	}
	for _, column := range columns {
		exists, err := tableHasColumn(s.db, column.table, column.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := s.db.Exec(column.definition); err != nil {
			return err
		}
	}
	return nil
}

func tableHasColumn(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, pk int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (s *Store) migrateOrdersTaskUniqueness() error {
	hasConstraint, err := hasUniqueTaskConstraint(s.db)
	if err != nil || !hasConstraint {
		return err
	}
	if _, err := s.db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer s.db.Exec(`PRAGMA foreign_keys = ON`)

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`
		CREATE TABLE orders_migrated (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			bid_id TEXT NOT NULL,
			owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			student_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			amount_fen INTEGER NOT NULL CHECK (amount_fen >= 0),
			fee_fen INTEGER NOT NULL CHECK (fee_fen >= 0),
			student_fen INTEGER NOT NULL CHECK (student_fen >= 0),
			status TEXT NOT NULL,
			payment_key TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			paid_at TEXT,
			check_in_at TEXT,
			completed_at TEXT,
			confirmed_at TEXT,
			rating_stars INTEGER NOT NULL DEFAULT 0
		)
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO orders_migrated(
			id, task_id, bid_id, owner_id, student_id, amount_fen, fee_fen, student_fen,
			status, payment_key, created_at, paid_at, check_in_at, completed_at, confirmed_at, rating_stars
		)
		SELECT id, task_id, bid_id, owner_id, student_id, amount_fen, fee_fen, student_fen,
		       status, payment_key, created_at, paid_at, check_in_at, completed_at, confirmed_at, rating_stars
		FROM orders
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE orders`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE orders_migrated RENAME TO orders`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		CREATE UNIQUE INDEX ux_orders_payment_key
		ON orders(payment_key) WHERE payment_key <> '';
		CREATE UNIQUE INDEX ux_orders_active_task
		ON orders(task_id) WHERE status IN ('UNPAID', 'PAID', 'IN_SERVICE', 'PENDING_CONFIRM');
		CREATE INDEX idx_orders_student ON orders(student_id, created_at DESC);
	`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	rows, err := s.db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("orders migration left inconsistent foreign keys")
	}
	return rows.Err()
}

func hasUniqueTaskConstraint(db *sql.DB) (bool, error) {
	rows, err := db.Query(`PRAGMA index_list('orders')`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	indexes := []string{}
	for rows.Next() {
		var seq, unique, partial int
		var name, origin string
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return false, err
		}
		if unique == 1 && origin == "u" {
			indexes = append(indexes, name)
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	for _, name := range indexes {
		info, err := db.Query(`PRAGMA index_info(` + quoteSQLiteIdentifier(name) + `)`)
		if err != nil {
			return false, err
		}
		columns := []string{}
		for info.Next() {
			var seq, cid int
			var column string
			if err := info.Scan(&seq, &cid, &column); err != nil {
				info.Close()
				return false, err
			}
			columns = append(columns, column)
		}
		if err := info.Close(); err != nil {
			return false, err
		}
		if len(columns) == 1 && columns[0] == "task_id" {
			return true, nil
		}
	}
	return false, nil
}

func quoteSQLiteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func (s *Store) rebuildStudentSlots() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM student_slots`); err != nil {
		return err
	}
	rows, err := tx.Query(`
		SELECT o.id, o.student_id, t.start_at, t.time_window
		FROM orders o JOIN tasks t ON t.id = o.task_id
		WHERE o.status IN ('UNPAID', 'PAID', 'IN_SERVICE', 'PENDING_CONFIRM', 'REFUNDING')
		ORDER BY t.start_at ASC, o.created_at ASC
	`)
	if err != nil {
		return err
	}
	type reservation struct {
		orderID    string
		studentID  string
		startAt    string
		timeWindow string
	}
	reservations := []reservation{}
	for rows.Next() {
		var item reservation
		if err := rows.Scan(&item.orderID, &item.studentID, &item.startAt, &item.timeWindow); err != nil {
			rows.Close()
			return err
		}
		reservations = append(reservations, item)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	now := timestamp(time.Now())
	for _, item := range reservations {
		if err := reserveStudentSlotsTx(tx, item.orderID, item.studentID, parseTimestamp(item.startAt), item.timeWindow, now); err != nil {
			return fmt.Errorf("rebuild student slots for order %s: %w", item.orderID, err)
		}
	}
	return tx.Commit()
}

func reserveStudentSlotsTx(tx *sql.Tx, orderID, studentID string, startAt time.Time, timeWindow, now string) error {
	slots, err := reservationSlots(startAt, timeWindow)
	if err != nil {
		return err
	}
	for _, slot := range slots {
		if _, err := tx.Exec(`
			INSERT INTO student_slots(student_id, slot_at, order_id, created_at)
			VALUES(?, ?, ?, ?)
		`, studentID, slot, orderID, now); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return conflict("该学生当前时段已有其他有效订单")
			}
			return err
		}
	}
	return nil
}

func releaseStudentSlotsTx(tx *sql.Tx, orderID string) error {
	_, err := tx.Exec(`DELETE FROM student_slots WHERE order_id = ?`, orderID)
	return err
}

func reservationSlots(startAt time.Time, timeWindow string) ([]string, error) {
	parts := strings.Split(timeWindow, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid time window %q", timeWindow)
	}
	startClock, err := time.Parse("15:04", strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, err
	}
	endClock, err := time.Parse("15:04", strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, err
	}
	startDelta := time.Duration(startClock.Hour())*time.Hour + time.Duration(startClock.Minute())*time.Minute
	endDelta := time.Duration(endClock.Hour())*time.Hour + time.Duration(endClock.Minute())*time.Minute
	if endDelta <= startDelta {
		endDelta += 24 * time.Hour
	}
	slots := []string{}
	for slot := startAt; slot.Before(startAt.Add(endDelta - startDelta)); slot = slot.Add(30 * time.Minute) {
		slots = append(slots, timestamp(slot))
	}
	return slots, nil
}
