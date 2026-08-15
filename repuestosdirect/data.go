package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Part struct {
	ID           string
	Make         string
	Model        string
	Year         int
	Category     string
	Name         string
	Source       string
	PriceUSD     float64
	Stock        int
	Courier      string
	ReorderPoint int
	Availability string
}

type StockRow struct {
	Part   Part
	Status string
}

type Shop struct {
	ID           string
	Name         string
	Owner        string
	Phone        string
	Address      string
	CreditLimit  float64
	CreditUsed   float64
	CreditTerms  int
	PasswordDemo string
	Approved     bool
	Active       bool
}

type OrderStatus string

const (
	StatusPedido  OrderStatus = "Pedido"
	StatusEnviado OrderStatus = "Enviado"
	StatusAduana  OrderStatus = "En aduana"
	StatusLlegado OrderStatus = "Llegado"
	StatusListo   OrderStatus = "Listo para recoger"
)

type OrderItem struct {
	PartID   string
	PartName string
	Qty      int
	UnitUSD  float64
}

type Order struct {
	ID       string
	ShopID   string
	Items    []OrderItem
	Total    float64
	OnCredit bool
	Status   OrderStatus
	PlacedAt time.Time
	DueDate  time.Time
	Courier  string
	PaidAt   *time.Time
}

type ReportSummary struct {
	TotalOrders     int
	TotalSales      float64
	CreditOutstanding float64
	OverdueCount    int
	LowStockCount   int
	ActiveShops     int
}

type Store struct {
	db *sql.DB
}

func NewPostgresStore(connStr string) (*Store, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening db connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to db: %w", err)
	}

	s := &Store{db: db}
	if err := s.initTables(); err != nil {
		return nil, fmt.Errorf("error initializing tables: %w", err)
	}

	return s, nil
}

func (s *Store) initTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS shops (
		id VARCHAR(50) PRIMARY KEY,
		name VARCHAR(255) UNIQUE NOT NULL,
		owner VARCHAR(255) NOT NULL,
		phone VARCHAR(50) NOT NULL,
		credit_limit NUMERIC(10,2) NOT NULL DEFAULT 300.00,
		credit_used NUMERIC(10,2) NOT NULL DEFAULT 0.00,
		credit_terms INT NOT NULL DEFAULT 15,
		password_demo VARCHAR(255) NOT NULL
	);

	CREATE TABLE IF NOT EXISTS parts (
		id VARCHAR(50) PRIMARY KEY,
		make VARCHAR(100) NOT NULL,
		model VARCHAR(100) NOT NULL,
		year INT NOT NULL,
		category VARCHAR(100) NOT NULL,
		name VARCHAR(255) NOT NULL,
		source VARCHAR(50) NOT NULL,
		price_usd NUMERIC(10,2) NOT NULL,
		stock INT NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS orders (
		id VARCHAR(50) PRIMARY KEY,
		shop_id VARCHAR(50) REFERENCES shops(id) ON DELETE SET NULL,
		total NUMERIC(10,2) NOT NULL,
		on_credit BOOLEAN NOT NULL DEFAULT FALSE,
		status VARCHAR(50) NOT NULL,
		placed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		due_date TIMESTAMPTZ
	);

	CREATE TABLE IF NOT EXISTS order_items (
		id BIGSERIAL PRIMARY KEY,
		order_id VARCHAR(50) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
		part_id VARCHAR(50) NOT NULL REFERENCES parts(id),
		part_name VARCHAR(255) NOT NULL,
		qty INT NOT NULL,
		unit_usd NUMERIC(10,2) NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id VARCHAR(64) PRIMARY KEY,
		shop_id VARCHAR(50) NOT NULL DEFAULT '',
		cart JSONB NOT NULL DEFAULT '{}',
		csrf_token VARCHAR(64) NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL
	);

	CREATE SEQUENCE IF NOT EXISTS order_id_seq;
	CREATE SEQUENCE IF NOT EXISTS shop_id_seq;
	CREATE SEQUENCE IF NOT EXISTS part_id_seq;
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	migrations := []string{
		`ALTER TABLE parts ADD COLUMN IF NOT EXISTS reorder_point INT NOT NULL DEFAULT 5`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS courier VARCHAR(20) NOT NULL DEFAULT ''`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS delivered_at TIMESTAMPTZ`,
		`ALTER TABLE shops ADD COLUMN IF NOT EXISTS approved BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE shops ADD COLUMN IF NOT EXISTS address VARCHAR(500) NOT NULL DEFAULT ''`,
		`ALTER TABLE shops ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT TRUE`,
		`ALTER TABLE parts ADD COLUMN IF NOT EXISTS availability VARCHAR(100) NOT NULL DEFAULT 'En stock'`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ`,
		`ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_part_id_fkey`,
		`ALTER TABLE order_items ALTER COLUMN part_id DROP NOT NULL`,
		`ALTER TABLE order_items ADD CONSTRAINT order_items_part_id_fkey FOREIGN KEY (part_id) REFERENCES parts(id) ON DELETE SET NULL`,
	}
	for _, q := range migrations {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) seedInitialData() {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM parts").Scan(&count)
	if count > 0 {
		return
	}

	log.Println("Seeding initial catalog parts and demo shops...")

	s.db.Exec(`
		INSERT INTO shops (id, name, owner, phone, credit_limit, credit_used, credit_terms, password_demo, approved, active)
		VALUES ('S-0001', 'Taller Los Santos', 'Carlos Santos', '+18095550199', 300.00, 0.00, 15, '1234', TRUE, TRUE)
		ON CONFLICT (id) DO NOTHING;
	`)

	s.db.Exec(`
		INSERT INTO parts (id, make, model, year, category, name, source, price_usd, stock, availability) VALUES
		('P-001', 'Toyota', 'Corolla', 2015, 'Frenos', 'Pastillas de freno delanteras', 'OEM USA', 28.50, 12, 'En stock'),
		('P-002', 'Toyota', 'Corolla', 2015, 'Frenos', 'Disco de freno delantero', 'Aftermarket High Quality', 45.00, 8, 'En stock'),
		('P-003', 'Toyota', 'Corolla', 2015, 'Motor', 'Filtro de aceite', 'OEM Japan', 8.50, 25, 'En stock'),
		('P-004', 'Toyota', 'Corolla', 2015, 'Suspensión', 'Amortiguador delantero derecho', 'KYB OEM', 68.00, 6, 'En stock')
		ON CONFLICT (id) DO NOTHING;
	`)
}

// --- Sessions ---

func (s *Store) LoadSession(id string) (*Session, bool) {
	var shopID, csrf string
	var cartJSON []byte
	var expires time.Time
	err := s.db.QueryRow(
		`SELECT shop_id, cart, csrf_token, expires_at FROM sessions WHERE id = $1 AND expires_at > now()`,
		id,
	).Scan(&shopID, &cartJSON, &csrf, &expires)
	if err != nil {
		return nil, false
	}
	return &Session{
		ID:        id,
		ShopID:    shopID,
		Cart:      cartFromJSON(cartJSON),
		CSRFToken: csrf,
	}, true
}

func (s *Store) SaveSession(sess *Session) {
	sess.mu.RLock()
	shopID := sess.ShopID
	cart := cartToJSON(sess.Cart)
	csrf := sess.CSRFToken
	sess.mu.RUnlock()

	_, err := s.db.Exec(`
		INSERT INTO sessions (id, shop_id, cart, csrf_token, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET shop_id = $2, cart = $3, csrf_token = $4, expires_at = $5`,
		sess.ID, shopID, cart, csrf, sessionExpiry(),
	)
	if err != nil {
		logError("session save failed", err.Error())
	}
}

func (s *Store) CleanExpiredSessions() {
	s.db.Exec(`DELETE FROM sessions WHERE expires_at < now()`)
}

// --- Orders ---

func (s *Store) UpdateOrderStatus(orderID string, status OrderStatus) (*Order, error) {
	if status == StatusLlegado {
		_, err := s.db.Exec(
			`UPDATE orders SET status = $1, delivered_at = now() WHERE id = $2`,
			string(status), orderID,
		)
		if err != nil {
			return nil, err
		}
	} else {
		_, err := s.db.Exec(`UPDATE orders SET status = $1 WHERE id = $2`, string(status), orderID)
		if err != nil {
			return nil, err
		}
	}
	return s.Order(orderID)
}

func (s *Store) MarkOrderPaid(orderID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var shopID sql.NullString
	var total float64
	var onCredit bool
	var paidAt sql.NullTime
	err = tx.QueryRow(
		`SELECT shop_id, total, on_credit, paid_at FROM orders WHERE id = $1 FOR UPDATE`,
		orderID,
	).Scan(&shopID, &total, &onCredit, &paidAt)
	if err != nil {
		return fmt.Errorf("pedido no encontrado")
	}
	if !onCredit {
		return fmt.Errorf("solo pedidos a crédito pueden marcarse como pagados")
	}
	if paidAt.Valid {
		return fmt.Errorf("pedido ya fue pagado")
	}
	if !shopID.Valid {
		return fmt.Errorf("pedido sin taller asociado")
	}

	_, err = tx.Exec(`UPDATE orders SET paid_at = now() WHERE id = $1`, orderID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`UPDATE shops SET credit_used = GREATEST(credit_used - $1, 0) WHERE id = $2`,
		total, shopID.String,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeletePart(partID string) error {
	_, err := s.db.Exec("DELETE FROM parts WHERE id = $1", partID)
	return err
}

func (s *Store) Order(orderID string) (*Order, error) {
	o := &Order{}
	var nullShop sql.NullString
	var paidAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, shop_id, total, on_credit, status, placed_at, due_date, courier, paid_at FROM orders WHERE id = $1`,
		orderID,
	).Scan(&o.ID, &nullShop, &o.Total, &o.OnCredit, &o.Status, &o.PlacedAt, &o.DueDate, &o.Courier, &paidAt)
	if err != nil {
		return nil, err
	}
	if nullShop.Valid {
		o.ShopID = nullShop.String
	}
	if paidAt.Valid {
		t := paidAt.Time
		o.PaidAt = &t
	}
	o.Items = s.getOrderItems(o.ID)
	return o, nil
}

// --- Catalog ---

func (s *Store) Makes() []string {
	rows, err := s.db.Query("SELECT DISTINCT make FROM parts ORDER BY make ASC")
	if err != nil {
		log.Println("Makes query error:", err)
		return nil
	}
	defer rows.Close()

	var makes []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err == nil {
			makes = append(makes, m)
		}
	}
	return makes
}

func (s *Store) Models(make string) []string {
	rows, err := s.db.Query("SELECT DISTINCT model FROM parts WHERE make = $1 ORDER BY model ASC", make)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var models []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err == nil {
			models = append(models, m)
		}
	}
	return models
}

func (s *Store) Years(make, model string) []int {
	rows, err := s.db.Query("SELECT DISTINCT year FROM parts WHERE make = $1 AND model = $2 ORDER BY year DESC", make, model)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var years []int
	for rows.Next() {
		var y int
		if err := rows.Scan(&y); err == nil {
			years = append(years, y)
		}
	}
	return years
}

func (s *Store) SearchParts(query string) []Part {
	q := "%" + query + "%"
	rows, err := s.db.Query(`
		SELECT id, make, model, year, category, name, source, price_usd, stock, availability
		FROM parts
		WHERE name ILIKE $1 OR category ILIKE $1 OR make ILIKE $1 OR model ILIKE $1 OR id ILIKE $1
		ORDER BY name ASC LIMIT 50`, q)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanParts(rows)
}

func scanParts(rows *sql.Rows) []Part {
	var parts []Part
	for rows.Next() {
		var p Part
		if err := rows.Scan(&p.ID, &p.Make, &p.Model, &p.Year, &p.Category, &p.Name, &p.Source, &p.PriceUSD, &p.Stock, &p.Availability); err == nil {
			parts = append(parts, p)
		}
	}
	return parts
}

func (s *Store) StockReport() []StockRow {
	rows, _ := s.db.Query(`SELECT id, make, model, year, category, name, source, price_usd, stock, reorder_point, availability FROM parts ORDER BY name`)
	defer rows.Close()
	var out []StockRow
	for rows.Next() {
		var p Part
		rows.Scan(&p.ID, &p.Make, &p.Model, &p.Year, &p.Category, &p.Name, &p.Source, &p.PriceUSD, &p.Stock, &p.ReorderPoint, &p.Availability)
		status := "ok"
		if p.Stock == 0 && isInStock(p.Availability) {
			status = "out"
		} else if p.Stock <= p.ReorderPoint && isInStock(p.Availability) {
			status = "reorder"
		}
		out = append(out, StockRow{Part: p, Status: status})
	}
	return out
}

func (s *Store) LowStockParts() []StockRow {
	var out []StockRow
	for _, row := range s.StockReport() {
		if row.Status == "reorder" || row.Status == "out" {
			out = append(out, row)
		}
	}
	return out
}

func (s *Store) PendingDeliveries() []*Order {
	rows, err := s.db.Query(`
		SELECT id, shop_id, total, on_credit, status, courier, placed_at, due_date
		FROM orders
		WHERE courier = '' AND status = $1 AND shop_id IS NOT NULL
		ORDER BY placed_at ASC`, string(StatusListo))
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanOrders(rows, s)
}

func (s *Store) AssignCourier(orderID, courier string) error {
	res, err := s.db.Exec(`UPDATE orders SET courier = $1 WHERE id = $2`, courier, orderID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pedido no encontrado")
	}
	return nil
}

func (s *Store) DriverReadyOrders() []*Order {
	rows, err := s.db.Query(`
		SELECT id, shop_id, total, on_credit, status, placed_at, due_date
		FROM orders
		WHERE status = $1 AND shop_id IS NOT NULL
		ORDER BY placed_at ASC`, string(StatusListo))
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanOrders(rows, s)
}

func (s *Store) PartsFor(make, model string, year int) []Part {
	rows, _ := s.db.Query(`SELECT id, make, model, year, category, name, source, price_usd, stock, availability FROM parts WHERE make = $1 AND model = $2 AND year = $3`, make, model, year)
	defer rows.Close()
	return scanParts(rows)
}

func (s *Store) Part(id string) (Part, bool) {
	var p Part
	err := s.db.QueryRow(`SELECT id, make, model, year, category, name, source, price_usd, stock, availability FROM parts WHERE id = $1`, id).
		Scan(&p.ID, &p.Make, &p.Model, &p.Year, &p.Category, &p.Name, &p.Source, &p.PriceUSD, &p.Stock, &p.Availability)
	return p, err == nil
}

// --- Shops ---

func scanShop(row *sql.Row) (*Shop, error) {
	sh := &Shop{}
	err := row.Scan(&sh.ID, &sh.Name, &sh.Owner, &sh.Phone, &sh.Address, &sh.CreditLimit, &sh.CreditUsed, &sh.CreditTerms, &sh.PasswordDemo, &sh.Approved, &sh.Active)
	return sh, err
}

func (s *Store) Shop(id string) (*Shop, bool) {
	sh, err := scanShop(s.db.QueryRow(
		`SELECT id, name, owner, phone, address, credit_limit, credit_used, credit_terms, password_demo, approved, active FROM shops WHERE id = $1 AND active = TRUE`, id))
	return sh, err == nil
}

func (s *Store) ShopByName(name string) (*Shop, bool) {
	sh, err := scanShop(s.db.QueryRow(
		`SELECT id, name, owner, phone, address, credit_limit, credit_used, credit_terms, password_demo, approved, active FROM shops WHERE name = $1 AND active = TRUE`, name))
	return sh, err == nil
}

func (s *Store) ShopNameTaken(name string) bool {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM shops WHERE name = $1 AND active = TRUE)", name).Scan(&exists)
	return err == nil && exists
}

func (s *Store) AllShops() []*Shop {
	return s.queryShops(`SELECT id, name, owner, phone, address, credit_limit, credit_used, credit_terms, password_demo, approved, active FROM shops WHERE active = TRUE AND approved = TRUE ORDER BY name ASC`)
}

func (s *Store) ActiveClients() []*Shop {
	return s.queryShops(`SELECT id, name, owner, phone, address, credit_limit, credit_used, credit_terms, password_demo, approved, active FROM shops WHERE active = TRUE AND approved = TRUE ORDER BY name ASC`)
}

func (s *Store) PendingShops() []*Shop {
	return s.queryShops(`SELECT id, name, owner, phone, address, credit_limit, credit_used, credit_terms, password_demo, approved, active FROM shops WHERE active = TRUE AND approved = FALSE ORDER BY id ASC`)
}

func (s *Store) queryShops(q string) []*Shop {
	rows, err := s.db.Query(q)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var shops []*Shop
	for rows.Next() {
		sh := &Shop{}
		if err := rows.Scan(&sh.ID, &sh.Name, &sh.Owner, &sh.Phone, &sh.Address, &sh.CreditLimit, &sh.CreditUsed, &sh.CreditTerms, &sh.PasswordDemo, &sh.Approved, &sh.Active); err == nil {
			shops = append(shops, sh)
		}
	}
	return shops
}

func (s *Store) AddShop(name, owner, phone, address, password string) (*Shop, error) {
	hashed, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	var seq int
	s.db.QueryRow("SELECT nextval('shop_id_seq')").Scan(&seq)
	id := fmt.Sprintf("S-%04d", seq)

	query := `INSERT INTO shops (id, name, owner, phone, address, credit_limit, credit_used, credit_terms, password_demo, approved, active)
              VALUES ($1, $2, $3, $4, $5, 300.00, 0.00, 15, $6, FALSE, TRUE)
              RETURNING id, name, owner, phone, address, credit_limit, credit_used, credit_terms, password_demo, approved, active`
	sh, err := scanShop(s.db.QueryRow(query, id, name, owner, phone, address, hashed))
	return sh, err
}

func (s *Store) ApproveShop(id string) error {
	_, err := s.db.Exec("UPDATE shops SET approved = TRUE WHERE id = $1 AND active = TRUE", id)
	return err
}

func (s *Store) RemoveShop(id string) error {
	sh, ok := s.Shop(id)
	if !ok {
		// allow removing inactive
		var err error
		sh, err = scanShop(s.db.QueryRow(
			`SELECT id, name, owner, phone, address, credit_limit, credit_used, credit_terms, password_demo, approved, active FROM shops WHERE id = $1`, id))
		if err != nil {
			return fmt.Errorf("taller no encontrado")
		}
	}
	if sh.CreditUsed > 0 {
		return fmt.Errorf("no se puede eliminar: el taller tiene $%.2f de crédito pendiente", sh.CreditUsed)
	}
	var unpaid int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM orders
		WHERE shop_id = $1 AND on_credit = TRUE AND paid_at IS NULL`, id).Scan(&unpaid)
	if unpaid > 0 {
		return fmt.Errorf("no se puede eliminar: hay %d pedido(s) a crédito sin pagar", unpaid)
	}
	_, err := s.db.Exec(`UPDATE shops SET active = FALSE, approved = FALSE WHERE id = $1`, id)
	return err
}

func (s *Store) UpgradePasswordHash(shopID, hashed string) {
	s.db.Exec(`UPDATE shops SET password_demo = $1 WHERE id = $2`, hashed, shopID)
}

func (s *Store) AddPart(makeStr, model, category, name, source string, year, stock, reorderPoint int, price float64, availability string) error {
	var seq int
	s.db.QueryRow("SELECT nextval('part_id_seq')").Scan(&seq)
	id := fmt.Sprintf("P-%04d", seq)
	_, err := s.db.Exec(`INSERT INTO parts (id, make, model, year, category, name, source, price_usd, stock, reorder_point, availability) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		id, makeStr, model, year, category, name, source, price, stock, reorderPoint, availability)
	return err
}

func (s *Store) PlaceOrder(shopID string, items []OrderItem, onCredit bool) (*Order, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var total float64
	for _, it := range items {
		total += it.UnitUSD * float64(it.Qty)
	}

	for _, item := range items {
		var stock int
		var avail string
		var partName string
		err = tx.QueryRow(
			`SELECT stock, availability, name FROM parts WHERE id = $1 FOR UPDATE`,
			item.PartID,
		).Scan(&stock, &avail, &partName)
		if err != nil {
			return nil, fmt.Errorf("repuesto no encontrado: %s", item.PartID)
		}
		if isInStock(avail) && stock < item.Qty {
			return nil, fmt.Errorf("stock insuficiente para %s (disponible: %d)", partName, stock)
		}
		if isInStock(avail) {
			_, err = tx.Exec(`UPDATE parts SET stock = stock - $1 WHERE id = $2`, item.Qty, item.PartID)
			if err != nil {
				return nil, err
			}
		}
	}

	terms := 15
	if shopID != "" && onCredit {
		var creditLimit, creditUsed float64
		var creditTerms int
		err = tx.QueryRow(
			`SELECT credit_limit, credit_used, credit_terms FROM shops WHERE id = $1 AND active = TRUE FOR UPDATE`,
			shopID,
		).Scan(&creditLimit, &creditUsed, &creditTerms)
		if err != nil {
			return nil, fmt.Errorf("taller no encontrado")
		}
		if (creditLimit - creditUsed) < total {
			return nil, fmt.Errorf("crédito insuficiente ($%.2f disponible)", creditLimit-creditUsed)
		}
		terms = creditTerms
		_, err = tx.Exec("UPDATE shops SET credit_used = credit_used + $1 WHERE id = $2", total, shopID)
		if err != nil {
			return nil, err
		}
	}

	var seq int
	err = tx.QueryRow("SELECT nextval('order_id_seq')").Scan(&seq)
	if err != nil {
		return nil, fmt.Errorf("error generating order id: %w", err)
	}
	orderID := fmt.Sprintf("ORD-%04d", seq)

	placedAt := time.Now()
	dueDate := placedAt
	if onCredit {
		dueDate = placedAt.AddDate(0, 0, terms)
	}
	status := StatusPedido

	var nullShopID sql.NullString
	if shopID != "" {
		nullShopID = sql.NullString{String: shopID, Valid: true}
	}

	_, err = tx.Exec(
		"INSERT INTO orders (id, shop_id, total, on_credit, status, placed_at, due_date) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		orderID, nullShopID, total, onCredit, status, placedAt, dueDate,
	)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		_, err = tx.Exec(
			"INSERT INTO order_items (order_id, part_id, part_name, qty, unit_usd) VALUES ($1, $2, $3, $4, $5)",
			orderID, item.PartID, item.PartName, item.Qty, item.UnitUSD,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Order{
		ID:       orderID,
		ShopID:   shopID,
		Items:    items,
		Total:    total,
		OnCredit: onCredit,
		Status:   status,
		PlacedAt: placedAt,
		DueDate:  dueDate,
	}, nil
}

func scanOrders(rows *sql.Rows, s *Store) []*Order {
	var orders []*Order
	for rows.Next() {
		o := &Order{}
		var nullShop sql.NullString
		rows.Scan(&o.ID, &nullShop, &o.Total, &o.OnCredit, &o.Status, &o.PlacedAt, &o.DueDate)
		if nullShop.Valid {
			o.ShopID = nullShop.String
		}
		o.Items = s.getOrderItems(o.ID)
		orders = append(orders, o)
	}
	return orders
}

func (s *Store) OrdersFor(shopID string) []*Order {
	rows, err := s.db.Query("SELECT id, shop_id, total, on_credit, status, placed_at, due_date FROM orders WHERE shop_id = $1 ORDER BY placed_at DESC", shopID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanOrders(rows, s)
}

func (s *Store) AllOrdersSortedByDue() []*Order {
	rows, err := s.db.Query("SELECT id, shop_id, total, on_credit, status, placed_at, due_date FROM orders ORDER BY due_date ASC")
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanOrders(rows, s)
}

func (s *Store) CreditOrdersForAging() []*Order {
	rows, err := s.db.Query(`
		SELECT id, shop_id, total, on_credit, status, placed_at, due_date
		FROM orders
		WHERE on_credit = TRUE AND paid_at IS NULL AND shop_id IS NOT NULL
		ORDER BY due_date ASC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanOrders(rows, s)
}

func (s *Store) ReportSummary() ReportSummary {
	var r ReportSummary
	s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total),0) FROM orders`).Scan(&r.TotalOrders, &r.TotalSales)
	s.db.QueryRow(`
		SELECT COALESCE(SUM(total),0)
		FROM orders WHERE on_credit = TRUE AND paid_at IS NULL`).Scan(&r.CreditOutstanding)
	s.db.QueryRow(`
		SELECT COUNT(*)
		FROM orders WHERE on_credit = TRUE AND paid_at IS NULL AND due_date < now()`).Scan(&r.OverdueCount)
	s.db.QueryRow(`SELECT COUNT(*) FROM shops WHERE active = TRUE AND approved = TRUE`).Scan(&r.ActiveShops)
	r.LowStockCount = len(s.LowStockParts())
	return r
}

func (s *Store) getOrderItems(orderID string) []OrderItem {
	rows, err := s.db.Query("SELECT part_id, part_name, qty, unit_usd FROM order_items WHERE order_id = $1", orderID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.PartID, &item.PartName, &item.Qty, &item.UnitUSD); err == nil {
			items = append(items, item)
		}
	}
	return items
}
