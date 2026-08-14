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
	CreditLimit  float64
	CreditUsed   float64
	CreditTerms  int
	PasswordDemo string
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
	CREATE SEQUENCE IF NOT EXISTS order_id_seq;
	CREATE SEQUENCE IF NOT EXISTS shop_id_seq;
	CREATE SEQUENCE IF NOT EXISTS part_id_seq;
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	s.db.Exec(`ALTER TABLE parts ADD COLUMN IF NOT EXISTS reorder_point INT NOT NULL DEFAULT 5`)
	s.db.Exec(`ALTER TABLE orders ADD COLUMN IF NOT EXISTS courier VARCHAR(20) NOT NULL DEFAULT ''`)
	s.db.Exec(`ALTER TABLE orders ADD COLUMN IF NOT EXISTS delivered_at TIMESTAMPTZ`)

	s.seedInitialData()
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
		INSERT INTO shops (id, name, owner, phone, credit_limit, credit_used, credit_terms, password_demo)
		VALUES ('S-0001', 'Taller Los Santos', 'Carlos Santos', '+18095550199', 300.00, 0.00, 15, '1234')
		ON CONFLICT (id) DO NOTHING;
	`)

	s.db.Exec(`
		INSERT INTO parts (id, make, model, year, category, name, source, price_usd, stock) VALUES
		('P-001', 'Toyota', 'Corolla', 2015, 'Frenos', 'Pastillas de freno delanteras', 'OEM USA', 28.50, 12),
		('P-002', 'Toyota', 'Corolla', 2015, 'Frenos', 'Disco de freno delantero', 'Aftermarket High Quality', 45.00, 8),
		('P-003', 'Toyota', 'Corolla', 2015, 'Motor', 'Filtro de aceite', 'OEM Japan', 8.50, 25),
		('P-004', 'Toyota', 'Corolla', 2015, 'Suspensión', 'Amortiguador delantero derecho', 'KYB OEM', 68.00, 6)
		ON CONFLICT (id) DO NOTHING;
	`)
}

// --- Store Queries ---

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
		log.Println("Models query error:", err)
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
		log.Println("Years query error:", err)
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

func (s *Store) StockReport() []StockRow {
	rows, err := s.db.Query(`SELECT id, make, model, year, category, name, source, price_usd, stock, reorder_point
		FROM parts WHERE stock > 0 OR source = 'OEM USA' ORDER BY name`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []StockRow
	for rows.Next() {
		var p Part
		rows.Scan(&p.ID, &p.Make, &p.Model, &p.Year, &p.Category, &p.Name, &p.Source, &p.PriceUSD, &p.Stock, &p.ReorderPoint)
		status := "ok"
		if p.Stock == 0 {
			status = "out"
		} else if p.Stock <= p.ReorderPoint {
			status = "reorder"
		}
		out = append(out, StockRow{Part: p, Status: status})
	}
	return out
}

func (s *Store) PendingDeliveries() []*Order {
	rows, err := s.db.Query(`SELECT id, shop_id, total, on_credit, status, courier, placed_at, due_date
		FROM orders WHERE courier = '' ORDER BY placed_at ASC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*Order
	for rows.Next() {
		o := &Order{}
		var nullShop sql.NullString
		rows.Scan(&o.ID, &nullShop, &o.Total, &o.OnCredit, &o.Status, &o.Courier, &o.PlacedAt, &o.DueDate)
		if nullShop.Valid {
			o.ShopID = nullShop.String
		}
		out = append(out, o)
	}
	return out
}

func (s *Store) AssignCourier(orderID, courier string) error {
	res, err := s.db.Exec(`UPDATE orders SET courier = $1, delivered_at = now() WHERE id = $2`, courier, orderID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pedido no encontrado")
	}
	return nil
}

func (s *Store) PartsFor(make, model string, year int) []Part {
	query := `SELECT id, make, model, year, category, name, source, price_usd, stock 
	          FROM parts WHERE make = $1 AND model = $2 AND year = $3`
	rows, err := s.db.Query(query, make, model, year)
	if err != nil {
		log.Println("PartsFor query error:", err)
		return nil
	}
	defer rows.Close()

	var parts []Part
	for rows.Next() {
		var p Part
		if err := rows.Scan(&p.ID, &p.Make, &p.Model, &p.Year, &p.Category, &p.Name, &p.Source, &p.PriceUSD, &p.Stock); err == nil {
			parts = append(parts, p)
		}
	}
	return parts
}

func (s *Store) Part(id string) (Part, bool) {
	query := `SELECT id, make, model, year, category, name, source, price_usd, stock FROM parts WHERE id = $1`
	var p Part
	err := s.db.QueryRow(query, id).Scan(&p.ID, &p.Make, &p.Model, &p.Year, &p.Category, &p.Name, &p.Source, &p.PriceUSD, &p.Stock)
	if err != nil {
		return Part{}, false
	}
	return p, true
}

func (s *Store) Shop(id string) (*Shop, bool) {
	query := `SELECT id, name, owner, phone, credit_limit, credit_used, credit_terms, password_demo FROM shops WHERE id = $1`
	sh := &Shop{}
	err := s.db.QueryRow(query, id).Scan(&sh.ID, &sh.Name, &sh.Owner, &sh.Phone, &sh.CreditLimit, &sh.CreditUsed, &sh.CreditTerms, &sh.PasswordDemo)
	if err != nil {
		return nil, false
	}
	return sh, true
}

func (s *Store) ShopByLogin(name, password string) (*Shop, bool) {
	query := `SELECT id, name, owner, phone, credit_limit, credit_used, credit_terms, password_demo FROM shops WHERE name = $1 AND password_demo = $2`
	sh := &Shop{}
	err := s.db.QueryRow(query, name, password).Scan(&sh.ID, &sh.Name, &sh.Owner, &sh.Phone, &sh.CreditLimit, &sh.CreditUsed, &sh.CreditTerms, &sh.PasswordDemo)
	if err != nil {
		return nil, false
	}
	return sh, true
}

func (s *Store) ShopNameTaken(name string) bool {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM shops WHERE name = $1)", name).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func (s *Store) AllShops() []*Shop {
	rows, err := s.db.Query("SELECT id, name, owner, phone, credit_limit, credit_used, credit_terms, password_demo FROM shops ORDER BY name ASC")
	if err != nil {
		log.Println("AllShops error:", err)
		return nil
	}
	defer rows.Close()

	var shops []*Shop
	for rows.Next() {
		sh := &Shop{}
		if err := rows.Scan(&sh.ID, &sh.Name, &sh.Owner, &sh.Phone, &sh.CreditLimit, &sh.CreditUsed, &sh.CreditTerms, &sh.PasswordDemo); err == nil {
			shops = append(shops, sh)
		}
	}
	return shops
}

func (s *Store) AddShop(name, owner, phone, password string) *Shop {
	var seq int
	err := s.db.QueryRow("SELECT nextval('shop_id_seq')").Scan(&seq)
	if err != nil {
		log.Println("Error generating shop ID:", err)
		return nil
	}
	id := fmt.Sprintf("S-%04d", seq)

	query := `INSERT INTO shops (id, name, owner, phone, credit_limit, credit_used, credit_terms, password_demo)
              VALUES ($1, $2, $3, $4, 300.00, 0.00, 15, $5)
              RETURNING id, name, owner, phone, credit_limit, credit_used, credit_terms, password_demo`

	sh := &Shop{}
	err = s.db.QueryRow(query, id, name, owner, phone, password).Scan(
		&sh.ID, &sh.Name, &sh.Owner, &sh.Phone, &sh.CreditLimit, &sh.CreditUsed, &sh.CreditTerms, &sh.PasswordDemo,
	)
	if err != nil {
		log.Println("AddShop error:", err)
		return nil
	}
	return sh
}

func (s *Store) AddPart(make, model, category, name, source string, year, stock, reorderPoint int, price float64) error {
	var seq int
	err := s.db.QueryRow("SELECT nextval('part_id_seq')").Scan(&seq)
	if err != nil {
		return fmt.Errorf("error generating part id: %w", err)
	}
	id := fmt.Sprintf("P-%04d", seq)

	query := `INSERT INTO parts (id, make, model, year, category, name, source, price_usd, stock, reorder_point)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err = s.db.Exec(query, id, make, model, year, category, name, source, price, stock, reorderPoint)
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

	terms := 15
	if shopID != "" && onCredit {
		sh, ok := s.Shop(shopID)
		if !ok {
			return nil, fmt.Errorf("taller no encontrado")
		}
		if (sh.CreditLimit - sh.CreditUsed) < total {
			return nil, fmt.Errorf("crédito insuficiente ($%.2f disponible)", sh.CreditLimit-sh.CreditUsed)
		}
		terms = sh.CreditTerms

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
	dueDate := placedAt.AddDate(0, 0, terms)
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

func (s *Store) OrdersFor(shopID string) []*Order {
	rows, err := s.db.Query("SELECT id, shop_id, total, on_credit, status, placed_at, due_date FROM orders WHERE shop_id = $1 ORDER BY placed_at DESC", shopID)
	if err != nil {
		return nil
	}
	defer rows.Close()

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

func (s *Store) AllOrdersSortedByDue() []*Order {
	rows, err := s.db.Query("SELECT id, shop_id, total, on_credit, status, placed_at, due_date FROM orders ORDER BY due_date ASC")
	if err != nil {
		return nil
	}
	defer rows.Close()

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
