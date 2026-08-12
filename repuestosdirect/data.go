package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// ---------------------------------------------------------------------
// In-memory data store.
//
// This stands in for PostgreSQL in the demo so the whole app runs as a
// single dependency-free binary. Every function below is written so it
// maps cleanly onto a real SQL table + query when you're ready to swap
// in Postgres (see README.md, section "Moving to Postgres").
// ---------------------------------------------------------------------

type Part struct {
	ID       string
	Make     string
	Model    string
	Year     int
	Category string // e.g. "Frenos", "Filtros", "Suspensión"
	Name     string
	Source   string // "local" (same/next day) or "import" (3-4 semanas)
	PriceUSD float64
	Stock    int // only meaningful for "local" parts
}

type Shop struct {
	ID           string
	Name         string
	Owner        string
	Phone        string // WhatsApp number
	CreditLimit  float64
	CreditUsed   float64
	CreditTerms  int // days
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
	DueDate  time.Time // only relevant if OnCredit
}

// Store holds all in-memory data behind a mutex. In production this
// entire struct is replaced by a *sql.DB / pgx pool.
type Store struct {
	mu     sync.RWMutex
	Parts  []Part
	Shops  map[string]*Shop
	Orders map[string]*Order
	nextID int
}

var store = NewStore()

func NewStore() *Store {
	s := &Store{
		Shops:  map[string]*Shop{},
		Orders: map[string]*Order{},
	}
	s.seed()
	return s
}

func (s *Store) genID(prefix string) string {
	s.nextID++
	return fmt.Sprintf("%s-%04d", prefix, s.nextID)
}

func (s *Store) seed() {
	// --- Catalog: normalized Make -> Model -> Year -> Parts, per the plan ---
	s.Parts = []Part{
		// Toyota Corolla 2015
		{ID: "P-0001", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 28.00, Stock: 14},
		{ID: "P-0002", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 6.50, Stock: 40},
		{ID: "P-0003", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 62.00, Stock: 0},
		{ID: "P-0004", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Correas", Name: "Correa de distribución", Source: "local", PriceUSD: 22.00, Stock: 9},
		{ID: "P-0005", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Carrocería", Name: "Guardafango delantero derecho (usado)", Source: "import", PriceUSD: 45.00, Stock: 0},

		// Toyota Corolla 2018
		{ID: "P-0006", Make: "Toyota", Model: "Corolla", Year: 2018, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 26.00, Stock: 11},
		{ID: "P-0007", Make: "Toyota", Model: "Corolla", Year: 2018, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 9.00, Stock: 22},

		// Honda Civic 2016
		{ID: "P-0008", Make: "Honda", Model: "Civic", Year: 2016, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 30.00, Stock: 8},
		{ID: "P-0009", Make: "Honda", Model: "Civic", Year: 2016, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 34.00, Stock: 0},
		{ID: "P-0010", Make: "Honda", Model: "Civic", Year: 2016, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 58.00, Stock: 0},

		// Hyundai Elantra 2017
		{ID: "P-0011", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 7.00, Stock: 30},
		{ID: "P-0012", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 71.00, Stock: 0},
		{ID: "P-0013", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Carrocería", Name: "Espejo lateral izquierdo (usado)", Source: "import", PriceUSD: 25.00, Stock: 0},

		// Kia Rio 2014
		{ID: "P-0014", Make: "Kia", Model: "Rio", Year: 2014, Category: "Correas", Name: "Correa serpentina", Source: "local", PriceUSD: 15.00, Stock: 17},
		{ID: "P-0015", Make: "Kia", Model: "Rio", Year: 2014, Category: "Motor", Name: "Alternador (usado)", Source: "import", PriceUSD: 95.00, Stock: 0},
	}

	// --- Shops (demo accounts) ---
	shops := []*Shop{
		{ID: "S-1001", Name: "Taller El Bravo", Owner: "Ramón Peña", Phone: "+1 809-555-0101", CreditLimit: 800, CreditUsed: 220, CreditTerms: 30, PasswordDemo: "1234"},
		{ID: "S-1002", Name: "Auto Servicios Núñez", Owner: "Carla Núñez", Phone: "+1 829-555-0142", CreditLimit: 500, CreditUsed: 0, CreditTerms: 15, PasswordDemo: "1234"},
		{ID: "S-1003", Name: "Mecánica Los Hermanos", Owner: "Julio y Freddy Matos", Phone: "+1 849-555-0187", CreditLimit: 1000, CreditUsed: 940, CreditTerms: 30, PasswordDemo: "1234"},
	}
	for _, sh := range shops {
		s.Shops[sh.ID] = sh
	}

	// --- A few historical orders so the dashboard isn't empty ---
	s.Orders["O-0001"] = &Order{
		ID:     "O-0001",
		ShopID: "S-1001",
		Items: []OrderItem{
			{PartID: "P-0001", PartName: "Pastillas de freno delanteras", Qty: 2, UnitUSD: 28.00},
			{PartID: "P-0002", PartName: "Filtro de aceite", Qty: 3, UnitUSD: 6.50},
		},
		Total:    75.50,
		OnCredit: true,
		Status:   StatusListo,
		PlacedAt: time.Now().AddDate(0, 0, -18),
		DueDate:  time.Now().AddDate(0, 0, 12),
	}
	s.Orders["O-0002"] = &Order{
		ID:     "O-0002",
		ShopID: "S-1001",
		Items: []OrderItem{
			{PartID: "P-0003", PartName: "Amortiguador delantero", Qty: 2, UnitUSD: 62.00},
		},
		Total:    124.00,
		OnCredit: true,
		Status:   StatusAduana,
		PlacedAt: time.Now().AddDate(0, 0, -9),
		DueDate:  time.Now().AddDate(0, 0, 21),
	}
	s.Orders["O-0003"] = &Order{
		ID:     "O-0003",
		ShopID: "S-1003",
		Items: []OrderItem{
			{PartID: "P-0012", PartName: "Discos de freno delanteros (par)", Qty: 4, UnitUSD: 71.00},
		},
		Total:    284.00,
		OnCredit: true,
		Status:   StatusEnviado,
		PlacedAt: time.Now().AddDate(0, 0, -25),
		DueDate:  time.Now().AddDate(0, 0, -1), // overdue on purpose, shows aging logic
	}
	s.nextID = 3
}

// ----- Query helpers (mirror what SQL queries would look like) -----

func (s *Store) Makes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[string]bool{}
	var out []string
	for _, p := range s.Parts {
		if !seen[p.Make] {
			seen[p.Make] = true
			out = append(out, p.Make)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Store) Models(make string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[string]bool{}
	var out []string
	for _, p := range s.Parts {
		if p.Make == make && !seen[p.Model] {
			seen[p.Model] = true
			out = append(out, p.Model)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Store) Years(make, model string) []int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[int]bool{}
	var out []int
	for _, p := range s.Parts {
		if p.Make == make && p.Model == model && !seen[p.Year] {
			seen[p.Year] = true
			out = append(out, p.Year)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

func (s *Store) PartsFor(make, model string, year int) []Part {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Part
	for _, p := range s.Parts {
		if p.Make == make && p.Model == model && p.Year == year {
			out = append(out, p)
		}
	}
	return out
}

func (s *Store) Part(id string) (Part, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.Parts {
		if p.ID == id {
			return p, true
		}
	}
	return Part{}, false
}

func (s *Store) Shop(id string) (*Shop, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sh, ok := s.Shops[id]
	return sh, ok
}

func (s *Store) ShopByLogin(name, password string) (*Shop, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sh := range s.Shops {
		if sh.Name == name && sh.PasswordDemo == password {
			return sh, true
		}
	}
	return nil, false
}

func (s *Store) OrdersFor(shopID string) []*Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Order
	for _, o := range s.Orders {
		if o.ShopID == shopID {
			out = append(out, o)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlacedAt.After(out[j].PlacedAt) })
	return out
}

func (s *Store) AllOrdersSortedByDue() []*Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Order
	for _, o := range s.Orders {
		if o.OnCredit {
			out = append(out, o)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DueDate.Before(out[j].DueDate) })
	return out
}

// PlaceOrder creates an order, updates credit usage, and (in production)
// would trigger the WhatsApp confirmation. See notify.go.
// shopID == "" means a guest, cash-only checkout with no account.
func (s *Store) PlaceOrder(shopID string, items []OrderItem, onCredit bool) (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var sh *Shop
	if shopID != "" {
		var ok bool
		sh, ok = s.Shops[shopID]
		if !ok {
			return nil, fmt.Errorf("taller no encontrado")
		}
	} else {
		onCredit = false // guests can never buy on credit
	}

	var total float64
	allLocal := true
	for _, it := range items {
		total += it.UnitUSD * float64(it.Qty)
	}
	for _, p := range s.Parts {
		for _, it := range items {
			if p.ID == it.PartID && p.Source == "import" {
				allLocal = false
			}
		}
	}

	if onCredit {
		if sh.CreditUsed+total > sh.CreditLimit {
			return nil, fmt.Errorf("este pedido excede el límite de crédito disponible del taller")
		}
		sh.CreditUsed += total
	}

	status := StatusPedido
	due := time.Time{}
	if sh != nil {
		due = time.Now().AddDate(0, 0, sh.CreditTerms)
	}
	if allLocal {
		status = StatusListo
	}

	o := &Order{
		ID:       s.genID("O"),
		ShopID:   shopID,
		Items:    items,
		Total:    total,
		OnCredit: onCredit,
		Status:   status,
		PlacedAt: time.Now(),
		DueDate:  due,
	}
	s.Orders[o.ID] = o
	return o, nil
}

// Add new shop to database

func (s *Store) ShopNameTaken(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sh := range s.Shops {
		if sh.Name == name {
			return true
		}
	}
	return false
}

func (s *Store) AddShop(name, owner, phone, password string) *Shop {
	s.mu.Lock()
	defer s.mu.Unlock()
	sh := &Shop{
		ID:           s.genID("S"),
		Name:         name,
		Owner:        owner,
		Phone:        phone,
		CreditLimit:  300, // starter limit; raise manually after a track record
		CreditUsed:   0,
		CreditTerms:  15,
		PasswordDemo: password,
	}
	s.Shops[sh.ID] = sh
	return sh
}
