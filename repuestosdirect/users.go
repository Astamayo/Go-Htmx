package main

import (
	"fmt"
	"os"
	"time"
)

type Admin struct {
	ID       string
	Username string
	Name     string
	Active   bool
}

type Driver struct {
	ID       string
	Username string
	Name     string
	Phone    string
	Zone     string
	Active   bool
}

func (s *Store) initAuthTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS admins (
			id VARCHAR(50) PRIMARY KEY,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL DEFAULT '',
			active BOOLEAN NOT NULL DEFAULT TRUE
		)`,
		`CREATE TABLE IF NOT EXISTS drivers (
			id VARCHAR(50) PRIMARY KEY,
			username VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			phone VARCHAR(50) NOT NULL DEFAULT '',
			zone VARCHAR(100) NOT NULL DEFAULT '',
			active BOOLEAN NOT NULL DEFAULT TRUE
		)`,
		`CREATE SEQUENCE IF NOT EXISTS admin_id_seq`,
		`CREATE SEQUENCE IF NOT EXISTS driver_id_seq`,
		`ALTER TABLE shops ADD COLUMN IF NOT EXISTS username VARCHAR(100) UNIQUE`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'guest'`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS admin_id VARCHAR(50) NOT NULL DEFAULT ''`,
		`ALTER TABLE sessions ADD COLUMN IF NOT EXISTS driver_id VARCHAR(50) NOT NULL DEFAULT ''`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS driver_id VARCHAR(50)`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ`,
		`CREATE TABLE IF NOT EXISTS order_status_history (
			id BIGSERIAL PRIMARY KEY,
			order_id VARCHAR(50) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
			status VARCHAR(50) NOT NULL,
			changed_by VARCHAR(100) NOT NULL DEFAULT '',
			changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			note TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id BIGSERIAL PRIMARY KEY,
			actor_role VARCHAR(20) NOT NULL,
			actor_id VARCHAR(50) NOT NULL DEFAULT '',
			action VARCHAR(100) NOT NULL,
			entity_type VARCHAR(50) NOT NULL DEFAULT '',
			entity_id VARCHAR(50) NOT NULL DEFAULT '',
			detail TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE parts ADD COLUMN IF NOT EXISTS part_number VARCHAR(100) NOT NULL DEFAULT ''`,
		`ALTER TABLE parts ADD COLUMN IF NOT EXISTS oem_ref VARCHAR(100) NOT NULL DEFAULT ''`,
		`ALTER TABLE parts ADD COLUMN IF NOT EXISTS part_condition VARCHAR(50) NOT NULL DEFAULT 'new'`,
		`ALTER TABLE parts ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE parts ADD COLUMN IF NOT EXISTS photo_url VARCHAR(500) NOT NULL DEFAULT ''`,
		`ALTER TABLE parts ADD COLUMN IF NOT EXISTS b2b_price NUMERIC(10,2)`,
		`UPDATE shops SET username = id WHERE username IS NULL OR username = ''`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return s.bootstrapAdmin()
}

func (s *Store) bootstrapAdmin() error {
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&n)
	if n > 0 {
		return nil
	}
	pass := os.Getenv("ADMIN_PASSWORD")
	if pass == "" {
		pass = "admin123"
	}
	hashed, err := hashPassword(pass)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO admins (id, username, password_hash, name) VALUES ('A-0001', 'admin', $1, 'Administrador') ON CONFLICT DO NOTHING`, hashed)
	return err
}

func (s *Store) AdminLogin(username, password string) (*Admin, bool) {
	a := &Admin{}
	var hash string
	err := s.db.QueryRow(`SELECT id, username, name, active, password_hash FROM admins WHERE username = $1 AND active = TRUE`, username).
		Scan(&a.ID, &a.Username, &a.Name, &a.Active, &hash)
	if err != nil || !checkPassword(hash, password) {
		return nil, false
	}
	return a, true
}

func (s *Store) Admin(id string) (*Admin, bool) {
	a := &Admin{}
	err := s.db.QueryRow(`SELECT id, username, name, active FROM admins WHERE id = $1 AND active = TRUE`, id).
		Scan(&a.ID, &a.Username, &a.Name, &a.Active)
	return a, err == nil
}

func (s *Store) AllAdmins() []*Admin {
	rows, _ := s.db.Query(`SELECT id, username, name, active FROM admins ORDER BY username`)
	defer rows.Close()
	var out []*Admin
	for rows.Next() {
		a := &Admin{}
		if rows.Scan(&a.ID, &a.Username, &a.Name, &a.Active) == nil {
			out = append(out, a)
		}
	}
	return out
}

func (s *Store) DriverLogin(username, password string) (*Driver, bool) {
	d := &Driver{}
	var hash string
	err := s.db.QueryRow(`SELECT id, username, name, phone, zone, active, password_hash FROM drivers WHERE username = $1 AND active = TRUE`, username).
		Scan(&d.ID, &d.Username, &d.Name, &d.Phone, &d.Zone, &d.Active, &hash)
	if err != nil || !checkPassword(hash, password) {
		return nil, false
	}
	return d, true
}

func (s *Store) Driver(id string) (*Driver, bool) {
	d := &Driver{}
	err := s.db.QueryRow(`SELECT id, username, name, phone, zone, active FROM drivers WHERE id = $1 AND active = TRUE`, id).
		Scan(&d.ID, &d.Username, &d.Name, &d.Phone, &d.Zone, &d.Active)
	return d, err == nil
}

func (s *Store) AllDrivers() []*Driver {
	rows, _ := s.db.Query(`SELECT id, username, name, phone, zone, active FROM drivers ORDER BY name`)
	defer rows.Close()
	var out []*Driver
	for rows.Next() {
		d := &Driver{}
		if rows.Scan(&d.ID, &d.Username, &d.Name, &d.Phone, &d.Zone, &d.Active) == nil {
			out = append(out, d)
		}
	}
	return out
}

func (s *Store) AddDriver(username, name, phone, zone, password string) (*Driver, error) {
	hashed, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	var seq int
	s.db.QueryRow(`SELECT nextval('driver_id_seq')`).Scan(&seq)
	id := fmt.Sprintf("D-%04d", seq)
	d := &Driver{}
	err = s.db.QueryRow(`
		INSERT INTO drivers (id, username, password_hash, name, phone, zone)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, username, name, phone, zone, active`,
		id, username, hashed, name, phone, zone).Scan(&d.ID, &d.Username, &d.Name, &d.Phone, &d.Zone, &d.Active)
	return d, err
}

func (s *Store) UpdateDriver(id, name, phone, zone string, active bool) error {
	_, err := s.db.Exec(`UPDATE drivers SET name=$2, phone=$3, zone=$4, active=$5 WHERE id=$1`, id, name, phone, zone, active)
	return err
}

func (s *Store) UpdateDriverPassword(id, password string) error {
	hashed, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE drivers SET password_hash=$2 WHERE id=$1`, id, hashed)
	return err
}

func (s *Store) ShopByUsername(username string) (*Shop, bool) {
	sh, err := scanShop(s.db.QueryRow(
		`SELECT `+shopColumns+` FROM shops WHERE (username = $1 OR name = $1) AND active = TRUE`, username))
	return sh, err == nil
}

func (s *Store) AddShopWithCredentials(name, username, owner, phone, address, password string) (*Shop, error) {
	if username == "" {
		username = name
	}
	hashed, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	var seq int
	s.db.QueryRow("SELECT nextval('shop_id_seq')").Scan(&seq)
	id := fmt.Sprintf("S-%04d", seq)
	query := `INSERT INTO shops (id, name, username, owner, phone, address, credit_limit, credit_used, credit_terms, payment_splits, reminder_days, password_demo, approved, active)
              VALUES ($1, $2, $3, $4, $5, $6, $7, 0.00, $8, $9, $10, $11, TRUE, TRUE)
              RETURNING ` + shopColumns
	return scanShop(s.db.QueryRow(query, id, name, username, owner, phone, address,
		DefaultCreditLimit, DefaultCreditTerms, DefaultPaymentSplits, DefaultReminderDays, hashed))
}

func (s *Store) UpdateShopDetails(id, name, username, owner, phone, address string) error {
	_, err := s.db.Exec(`UPDATE shops SET name=$2, username=$3, owner=$4, phone=$5, address=$6 WHERE id=$1 AND active=TRUE`,
		id, name, username, owner, phone, address)
	return err
}

func (s *Store) LogAudit(actorRole, actorID, action, entityType, entityID, detail string) {
	s.db.Exec(`INSERT INTO audit_log (actor_role, actor_id, action, entity_type, entity_id, detail) VALUES ($1,$2,$3,$4,$5,$6)`,
		actorRole, actorID, action, entityType, entityID, detail)
}

func minCreditOrderAmount() float64 {
	v := os.Getenv("MIN_CREDIT_ORDER")
	if v == "" {
		return 50.0
	}
	f, err := parseFloat(v)
	if err != nil || f <= 0 {
		return 50.0
	}
	return f
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func (s *Store) RecordOrderStatus(orderID string, status OrderStatus, changedBy, note string) {
	s.db.Exec(`INSERT INTO order_status_history (order_id, status, changed_by, note) VALUES ($1,$2,$3,$4)`,
		orderID, string(status), changedBy, note)
}

func (s *Store) OrderStatusHistory(orderID string) []StatusHistoryEntry {
	rows, err := s.db.Query(`SELECT status, changed_by, changed_at, note FROM order_status_history WHERE order_id=$1 ORDER BY changed_at ASC`, orderID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []StatusHistoryEntry
	for rows.Next() {
		var e StatusHistoryEntry
		if rows.Scan(&e.Status, &e.ChangedBy, &e.ChangedAt, &e.Note) == nil {
			out = append(out, e)
		}
	}
	return out
}

type StatusHistoryEntry struct {
	Status    string
	ChangedBy string
	ChangedAt time.Time
	Note      string
}

type AuditEntry struct {
	ActorRole  string
	ActorID    string
	Action     string
	EntityType string
	EntityID   string
	Detail     string
	CreatedAt  time.Time
}

func (s *Store) RecentAuditLog(limit int) []AuditEntry {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT actor_role, actor_id, action, entity_type, entity_id, detail, created_at FROM audit_log ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if rows.Scan(&e.ActorRole, &e.ActorID, &e.Action, &e.EntityType, &e.EntityID, &e.Detail, &e.CreatedAt) == nil {
			out = append(out, e)
		}
	}
	return out
}

func (s *Store) UpdateShopPassword(id, password string) error {
	hashed, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE shops SET password_demo=$2 WHERE id=$1`, id, hashed)
	return err
}

func (s *Store) DeactivateShop(id string) error {
	_, err := s.db.Exec(`UPDATE shops SET active=FALSE, approved=FALSE WHERE id=$1`, id)
	return err
}

func (s *Store) AssignDriverToOrder(orderID, driverID string) error {
	_, err := s.db.Exec(`UPDATE orders SET driver_id=$2 WHERE id=$1`, orderID, driverID)
	return err
}
