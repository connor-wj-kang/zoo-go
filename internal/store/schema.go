package store

const schemaSQL = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    open_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    avatar TEXT NOT NULL DEFAULT '',
    phone_mask TEXT NOT NULL DEFAULT '',
    owner_verified INTEGER NOT NULL DEFAULT 0,
    student_verified INTEGER NOT NULL DEFAULT 0,
    student_status TEXT NOT NULL DEFAULT 'UNVERIFIED',
    school TEXT NOT NULL DEFAULT '',
    verification_expires_at TEXT NOT NULL DEFAULT '',
    account_status TEXT NOT NULL DEFAULT 'ACTIVE',
    disabled_reason TEXT NOT NULL DEFAULT '',
    disabled_at TEXT NOT NULL DEFAULT '',
    rating REAL NOT NULL DEFAULT 0,
    done_count INTEGER NOT NULL DEFAULT 0,
    service_radius_km REAL NOT NULL DEFAULT 5,
    service_latitude REAL NOT NULL DEFAULT 0,
    service_longitude REAL NOT NULL DEFAULT 0,
    location_updated_at TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    token_hash TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expiry ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS service_areas (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    district TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS owner_addresses (
    owner_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    community TEXT NOT NULL,
    detail TEXT NOT NULL,
    location_name TEXT NOT NULL DEFAULT '',
    latitude REAL NOT NULL DEFAULT 0,
    longitude REAL NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pets (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    species TEXT NOT NULL,
    breed TEXT NOT NULL DEFAULT '',
    age TEXT NOT NULL DEFAULT '',
    image TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    flags_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_pets_owner ON pets(owner_id, created_at);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pet_id TEXT NOT NULL,
    pet_snapshot TEXT NOT NULL,
    mode TEXT NOT NULL,
    status TEXT NOT NULL,
    budget_fen INTEGER NOT NULL CHECK (budget_fen >= 0),
    price_fen INTEGER NOT NULL DEFAULT 0 CHECK (price_fen >= 0),
    service_date TEXT NOT NULL,
    service_date_text TEXT NOT NULL,
    time_window TEXT NOT NULL,
    start_at TEXT NOT NULL,
    services_json TEXT NOT NULL,
    community TEXT NOT NULL,
    location_name TEXT NOT NULL DEFAULT '',
    latitude REAL NOT NULL DEFAULT 0,
    longitude REAL NOT NULL DEFAULT 0,
    address_masked TEXT NOT NULL,
    address_detail TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    checklist_json TEXT NOT NULL DEFAULT '[]',
    bid_deadline_at TEXT NOT NULL,
    pay_deadline_at TEXT,
    winner_bid_id TEXT NOT NULL DEFAULT '',
    order_id TEXT NOT NULL DEFAULT '',
    photos_json TEXT NOT NULL DEFAULT '[]',
    rated INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tasks_owner_created ON tasks(owner_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tasks_status_deadline ON tasks(status, bid_deadline_at);

CREATE TABLE IF NOT EXISTS bids (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    student_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    price_fen INTEGER NOT NULL CHECK (price_fen >= 0),
    message TEXT NOT NULL DEFAULT '',
    revision INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_bids_task_price ON bids(task_id, price_fen);
CREATE INDEX IF NOT EXISTS idx_bids_student ON bids(student_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS ux_bids_active_student
    ON bids(task_id, student_id) WHERE status IN ('ACTIVE', 'WON');

CREATE TABLE IF NOT EXISTS orders (
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
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_orders_payment_key
    ON orders(payment_key) WHERE payment_key <> '';
CREATE UNIQUE INDEX IF NOT EXISTS ux_orders_active_task
    ON orders(task_id) WHERE status IN ('UNPAID', 'PAID', 'IN_SERVICE', 'PENDING_CONFIRM');
CREATE INDEX IF NOT EXISTS idx_orders_student ON orders(student_id, created_at DESC);

CREATE TABLE IF NOT EXISTS student_slots (
    student_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    slot_at TEXT NOT NULL,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    PRIMARY KEY(student_id, slot_at)
);
CREATE INDEX IF NOT EXISTS idx_student_slots_order ON student_slots(order_id);

CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    out_trade_no TEXT NOT NULL UNIQUE,
    amount_fen INTEGER NOT NULL CHECK (amount_fen > 0),
    status TEXT NOT NULL,
    prepay_id TEXT NOT NULL DEFAULT '',
    wechat_transaction_id TEXT NOT NULL DEFAULT '',
    last_error TEXT NOT NULL DEFAULT '',
    paid_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_payments_transaction
    ON payments(wechat_transaction_id) WHERE wechat_transaction_id <> '';

CREATE TABLE IF NOT EXISTS refunds (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    payment_id TEXT NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    out_refund_no TEXT NOT NULL UNIQUE,
    wechat_refund_id TEXT NOT NULL DEFAULT '',
    amount_fen INTEGER NOT NULL CHECK (amount_fen > 0),
    total_fen INTEGER NOT NULL CHECK (total_fen > 0),
    reason TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    requested_by TEXT NOT NULL REFERENCES users(id),
    reviewed_by TEXT NOT NULL DEFAULT '',
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    succeeded_at TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_refunds_active_order
    ON refunds(order_id) WHERE status IN ('REQUESTED', 'PROCESSING', 'SUCCESS');
CREATE INDEX IF NOT EXISTS idx_refunds_order_created ON refunds(order_id, created_at DESC);

CREATE TABLE IF NOT EXISTS community_posts (
    id TEXT PRIMARY KEY,
    author_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    images_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_community_created ON community_posts(created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS uploads (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename TEXT NOT NULL UNIQUE,
    original_name TEXT NOT NULL DEFAULT '',
    mime_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_uploads_owner ON uploads(owner_id, created_at DESC);

CREATE TABLE IF NOT EXISTS student_verifications (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    real_name TEXT NOT NULL,
    school TEXT NOT NULL,
    student_no TEXT NOT NULL,
    major TEXT NOT NULL DEFAULT '',
    enrollment_year INTEGER NOT NULL DEFAULT 0,
    chsi_code TEXT NOT NULL,
    chsi_code_hash TEXT NOT NULL,
    status TEXT NOT NULL,
    review_note TEXT NOT NULL DEFAULT '',
    reviewed_by TEXT NOT NULL DEFAULT '',
    submitted_at TEXT NOT NULL,
    reviewed_at TEXT,
    valid_until TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_student_verifications_user
    ON student_verifications(user_id, submitted_at DESC);
CREATE INDEX IF NOT EXISTS idx_student_verifications_status
    ON student_verifications(status, submitted_at ASC);
CREATE UNIQUE INDEX IF NOT EXISTS ux_student_verifications_pending
    ON student_verifications(user_id) WHERE status = 'PENDING';
CREATE UNIQUE INDEX IF NOT EXISTS ux_student_verifications_chsi_active
    ON student_verifications(chsi_code_hash) WHERE status IN ('PENDING', 'VERIFIED');

CREATE TABLE IF NOT EXISTS admin_audit_logs (
    id TEXT PRIMARY KEY,
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_admin_audit_created ON admin_audit_logs(created_at DESC, id DESC);
`

var defaultServiceAreas = []struct {
	ID       string
	Name     string
	District string
	Order    int
}{
	{ID: "area_lry", Name: "马连洼·朗润园附近", District: "海淀区", Order: 10},
	{ID: "area_ndnl", Name: "马连洼·农大南路附近", District: "海淀区", Order: 20},
	{ID: "area_ymyxl", Name: "马连洼·圆明园西路附近", District: "海淀区", Order: 30},
}
