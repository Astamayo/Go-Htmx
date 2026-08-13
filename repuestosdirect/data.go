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
		// ============================================================
		// TOYOTA COROLLA
		// ============================================================

		// Corolla 2015
		{ID: "P-0001", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 28.00, Stock: 14},
		{ID: "P-0002", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 26.00, Stock: 11},
		{ID: "P-0003", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 68.00, Stock: 0},
		{ID: "P-0004", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 6.50, Stock: 40},
		{ID: "P-0005", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 9.00, Stock: 22},
		{ID: "P-0006", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Filtros", Name: "Filtro de cabina", Source: "local", PriceUSD: 8.50, Stock: 18},
		{ID: "P-0007", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 62.00, Stock: 0},
		{ID: "P-0008", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Suspensión", Name: "Amortiguador trasero", Source: "import", PriceUSD: 48.00, Stock: 0},
		{ID: "P-0009", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 32.00, Stock: 0},
		{ID: "P-0010", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Suspensión", Name: "Link de barra estabilizadora", Source: "import", PriceUSD: 19.00, Stock: 0},
		{ID: "P-0011", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Suspensión", Name: "Terminal de dirección", Source: "import", PriceUSD: 27.00, Stock: 0},
		{ID: "P-0012", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 24.00, Stock: 10},
		{ID: "P-0013", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Motor", Name: "Bobina de encendido", Source: "import", PriceUSD: 38.00, Stock: 4},
		{ID: "P-0014", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 58.00, Stock: 0},
		{ID: "P-0015", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Motor", Name: "Termostato", Source: "import", PriceUSD: 21.00, Stock: 0},
		{ID: "P-0016", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Correas", Name: "Correa serpentina", Source: "local", PriceUSD: 18.00, Stock: 12},
		{ID: "P-0017", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 125.00, Stock: 0},
		{ID: "P-0018", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Eléctrico", Name: "Motor de arranque", Source: "import", PriceUSD: 110.00, Stock: 0},
		{ID: "P-0019", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Eléctrico", Name: "Sensor de oxígeno", Source: "import", PriceUSD: 45.00, Stock: 0},
		{ID: "P-0020", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "Enfriamiento", Name: "Radiador", Source: "import", PriceUSD: 115.00, Stock: 0},
		{ID: "P-0021", Make: "Toyota", Model: "Corolla", Year: 2015, Category: "A/C", Name: "Compresor de A/C", Source: "import", PriceUSD: 175.00, Stock: 0},

		// Corolla 2018
		{ID: "P-0022", Make: "Toyota", Model: "Corolla", Year: 2018, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 29.00, Stock: 12},
		{ID: "P-0023", Make: "Toyota", Model: "Corolla", Year: 2018, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 26.00, Stock: 11},
		{ID: "P-0024", Make: "Toyota", Model: "Corolla", Year: 2018, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 6.50, Stock: 35},
		{ID: "P-0025", Make: "Toyota", Model: "Corolla", Year: 2018, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 9.00, Stock: 22},
		{ID: "P-0026", Make: "Toyota", Model: "Corolla", Year: 2018, Category: "Filtros", Name: "Filtro de cabina", Source: "local", PriceUSD: 8.50, Stock: 18},
		{ID: "P-0027", Make: "Toyota", Model: "Corolla", Year: 2018, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 65.00, Stock: 0},
		{ID: "P-0028", Make: "Toyota", Model: "Corolla", Year: 2018, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 34.00, Stock: 0},
		{ID: "P-0029", Make: "Toyota", Model: "Corolla", Year: 2018, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 25.00, Stock: 10},
		{ID: "P-0030", Make: "Toyota", Model: "Corolla", Year: 2018, Category: "Motor", Name: "Bobina de encendido", Source: "import", PriceUSD: 40.00, Stock: 4},

		// ============================================================
		// TOYOTA RAV4
		// ============================================================

		// RAV4 2015
		{ID: "P-0031", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 32.00, Stock: 10},
		{ID: "P-0032", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 29.00, Stock: 8},
		{ID: "P-0033", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 82.00, Stock: 0},
		{ID: "P-0034", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 7.00, Stock: 30},
		{ID: "P-0035", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 11.00, Stock: 15},
		{ID: "P-0036", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "Filtros", Name: "Filtro de cabina", Source: "local", PriceUSD: 10.00, Stock: 15},
		{ID: "P-0037", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 75.00, Stock: 0},
		{ID: "P-0038", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 38.00, Stock: 0},
		{ID: "P-0039", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "Suspensión", Name: "Link de barra estabilizadora", Source: "import", PriceUSD: 22.00, Stock: 0},
		{ID: "P-0040", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "Suspensión", Name: "Cojinete de rueda", Source: "import", PriceUSD: 55.00, Stock: 0},
		{ID: "P-0041", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 65.00, Stock: 0},
		{ID: "P-0042", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 28.00, Stock: 8},
		{ID: "P-0043", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 145.00, Stock: 0},
		{ID: "P-0044", Make: "Toyota", Model: "RAV4", Year: 2015, Category: "A/C", Name: "Compresor de A/C", Source: "import", PriceUSD: 190.00, Stock: 0},

		// ============================================================
		// TOYOTA CAMRY
		// ============================================================

		// Camry 2015
		{ID: "P-0045", Make: "Toyota", Model: "Camry", Year: 2015, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 32.00, Stock: 10},
		{ID: "P-0046", Make: "Toyota", Model: "Camry", Year: 2015, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 28.00, Stock: 8},
		{ID: "P-0047", Make: "Toyota", Model: "Camry", Year: 2015, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 78.00, Stock: 0},
		{ID: "P-0048", Make: "Toyota", Model: "Camry", Year: 2015, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 7.00, Stock: 30},
		{ID: "P-0049", Make: "Toyota", Model: "Camry", Year: 2015, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 10.00, Stock: 15},
		{ID: "P-0050", Make: "Toyota", Model: "Camry", Year: 2015, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 68.00, Stock: 0},
		{ID: "P-0051", Make: "Toyota", Model: "Camry", Year: 2015, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 35.00, Stock: 0},
		{ID: "P-0052", Make: "Toyota", Model: "Camry", Year: 2015, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 62.00, Stock: 0},
		{ID: "P-0053", Make: "Toyota", Model: "Camry", Year: 2015, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 28.00, Stock: 8},
		{ID: "P-0054", Make: "Toyota", Model: "Camry", Year: 2015, Category: "Eléctrico", Name: "Bobina de encendido", Source: "import", PriceUSD: 40.00, Stock: 4},

		// ============================================================
		// HONDA CIVIC
		// ============================================================

		// Civic 2016
		{ID: "P-0055", Make: "Honda", Model: "Civic", Year: 2016, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 30.00, Stock: 8},
		{ID: "P-0056", Make: "Honda", Model: "Civic", Year: 2016, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 27.00, Stock: 8},
		{ID: "P-0057", Make: "Honda", Model: "Civic", Year: 2016, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 72.00, Stock: 0},
		{ID: "P-0058", Make: "Honda", Model: "Civic", Year: 2016, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 7.00, Stock: 30},
		{ID: "P-0059", Make: "Honda", Model: "Civic", Year: 2016, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 10.00, Stock: 15},
		{ID: "P-0060", Make: "Honda", Model: "Civic", Year: 2016, Category: "Filtros", Name: "Filtro de cabina", Source: "local", PriceUSD: 9.00, Stock: 15},
		{ID: "P-0061", Make: "Honda", Model: "Civic", Year: 2016, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 34.00, Stock: 0},
		{ID: "P-0062", Make: "Honda", Model: "Civic", Year: 2016, Category: "Suspensión", Name: "Link de barra estabilizadora", Source: "import", PriceUSD: 20.00, Stock: 0},
		{ID: "P-0063", Make: "Honda", Model: "Civic", Year: 2016, Category: "Suspensión", Name: "Terminal de dirección", Source: "import", PriceUSD: 29.00, Stock: 0},
		{ID: "P-0064", Make: "Honda", Model: "Civic", Year: 2016, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 58.00, Stock: 0},
		{ID: "P-0065", Make: "Honda", Model: "Civic", Year: 2016, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 28.00, Stock: 8},
		{ID: "P-0066", Make: "Honda", Model: "Civic", Year: 2016, Category: "Motor", Name: "Bobina de encendido", Source: "import", PriceUSD: 42.00, Stock: 4},
		{ID: "P-0067", Make: "Honda", Model: "Civic", Year: 2016, Category: "Eléctrico", Name: "Sensor de oxígeno", Source: "import", PriceUSD: 48.00, Stock: 0},
		{ID: "P-0068", Make: "Honda", Model: "Civic", Year: 2016, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 135.00, Stock: 0},
		{ID: "P-0069", Make: "Honda", Model: "Civic", Year: 2016, Category: "A/C", Name: "Compresor de A/C", Source: "import", PriceUSD: 185.00, Stock: 0},

		// ============================================================
		// HONDA CR-V
		// ============================================================

		// CR-V 2016
		{ID: "P-0070", Make: "Honda", Model: "CR-V", Year: 2016, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 32.00, Stock: 10},
		{ID: "P-0071", Make: "Honda", Model: "CR-V", Year: 2016, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 29.00, Stock: 8},
		{ID: "P-0072", Make: "Honda", Model: "CR-V", Year: 2016, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 78.00, Stock: 0},
		{ID: "P-0073", Make: "Honda", Model: "CR-V", Year: 2016, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 7.00, Stock: 30},
		{ID: "P-0074", Make: "Honda", Model: "CR-V", Year: 2016, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 11.00, Stock: 15},
		{ID: "P-0075", Make: "Honda", Model: "CR-V", Year: 2016, Category: "Filtros", Name: "Filtro de cabina", Source: "local", PriceUSD: 10.00, Stock: 15},
		{ID: "P-0076", Make: "Honda", Model: "CR-V", Year: 2016, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 72.00, Stock: 0},
		{ID: "P-0077", Make: "Honda", Model: "CR-V", Year: 2016, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 38.00, Stock: 0},
		{ID: "P-0078", Make: "Honda", Model: "CR-V", Year: 2016, Category: "Suspensión", Name: "Link de barra estabilizadora", Source: "import", PriceUSD: 22.00, Stock: 0},
		{ID: "P-0079", Make: "Honda", Model: "CR-V", Year: 2016, Category: "Suspensión", Name: "Cojinete de rueda", Source: "import", PriceUSD: 52.00, Stock: 0},
		{ID: "P-0080", Make: "Honda", Model: "CR-V", Year: 2016, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 62.00, Stock: 0},
		{ID: "P-0081", Make: "Honda", Model: "CR-V", Year: 2016, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 28.00, Stock: 8},
		{ID: "P-0082", Make: "Honda", Model: "CR-V", Year: 2016, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 145.00, Stock: 0},
		{ID: "P-0083", Make: "Honda", Model: "CR-V", Year: 2016, Category: "A/C", Name: "Compresor de A/C", Source: "import", PriceUSD: 195.00, Stock: 0},

		// ============================================================
		// HYUNDAI TUCSON
		// ============================================================

		// Tucson 2017
		{ID: "P-0084", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 30.00, Stock: 10},
		{ID: "P-0085", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 28.00, Stock: 8},
		{ID: "P-0086", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 76.00, Stock: 0},
		{ID: "P-0087", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 7.00, Stock: 30},
		{ID: "P-0088", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 10.00, Stock: 15},
		{ID: "P-0089", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Filtros", Name: "Filtro de cabina", Source: "local", PriceUSD: 9.00, Stock: 15},
		{ID: "P-0090", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 68.00, Stock: 0},
		{ID: "P-0091", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 35.00, Stock: 0},
		{ID: "P-0092", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Suspensión", Name: "Link de barra estabilizadora", Source: "import", PriceUSD: 21.00, Stock: 0},
		{ID: "P-0093", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Suspensión", Name: "Cojinete de rueda", Source: "import", PriceUSD: 55.00, Stock: 0},
		{ID: "P-0094", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 27.00, Stock: 8},
		{ID: "P-0095", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Motor", Name: "Bobina de encendido", Source: "import", PriceUSD: 40.00, Stock: 4},
		{ID: "P-0096", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 60.00, Stock: 0},
		{ID: "P-0097", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Eléctrico", Name: "Sensor de oxígeno", Source: "import", PriceUSD: 42.00, Stock: 0},
		{ID: "P-0098", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 135.00, Stock: 0},
		{ID: "P-0099", Make: "Hyundai", Model: "Tucson", Year: 2017, Category: "A/C", Name: "Compresor de A/C", Source: "import", PriceUSD: 180.00, Stock: 0},

		// ============================================================
		// HYUNDAI ELANTRA
		// ============================================================

		// Elantra 2017
		{ID: "P-0100", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 7.00, Stock: 30},
		{ID: "P-0101", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 9.00, Stock: 20},
		{ID: "P-0102", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Filtros", Name: "Filtro de cabina", Source: "local", PriceUSD: 9.00, Stock: 15},
		{ID: "P-0103", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 28.00, Stock: 10},
		{ID: "P-0104", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 25.00, Stock: 8},
		{ID: "P-0105", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 71.00, Stock: 0},
		{ID: "P-0106", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 60.00, Stock: 0},
		{ID: "P-0107", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 32.00, Stock: 0},
		{ID: "P-0108", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Suspensión", Name: "Link de barra estabilizadora", Source: "import", PriceUSD: 19.00, Stock: 0},
		{ID: "P-0109", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 25.00, Stock: 8},
		{ID: "P-0110", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Motor", Name: "Bobina de encendido", Source: "import", PriceUSD: 38.00, Stock: 4},
		{ID: "P-0111", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 55.00, Stock: 0},
		{ID: "P-0112", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 125.00, Stock: 0},
		{ID: "P-0113", Make: "Hyundai", Model: "Elantra", Year: 2017, Category: "Carrocería", Name: "Espejo lateral izquierdo", Source: "import", PriceUSD: 55.00, Stock: 0},

		// ============================================================
		// HYUNDAI SANTA FE
		// ============================================================

		// Santa Fe 2017
		{ID: "P-0114", Make: "Hyundai", Model: "Santa Fe", Year: 2017, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 35.00, Stock: 8},
		{ID: "P-0115", Make: "Hyundai", Model: "Santa Fe", Year: 2017, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 30.00, Stock: 8},
		{ID: "P-0116", Make: "Hyundai", Model: "Santa Fe", Year: 2017, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 90.00, Stock: 0},
		{ID: "P-0117", Make: "Hyundai", Model: "Santa Fe", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 8.00, Stock: 25},
		{ID: "P-0118", Make: "Hyundai", Model: "Santa Fe", Year: 2017, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 12.00, Stock: 15},
		{ID: "P-0119", Make: "Hyundai", Model: "Santa Fe", Year: 2017, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 78.00, Stock: 0},
		{ID: "P-0120", Make: "Hyundai", Model: "Santa Fe", Year: 2017, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 40.00, Stock: 0},
		{ID: "P-0121", Make: "Hyundai", Model: "Santa Fe", Year: 2017, Category: "Suspensión", Name: "Cojinete de rueda", Source: "import", PriceUSD: 60.00, Stock: 0},
		{ID: "P-0122", Make: "Hyundai", Model: "Santa Fe", Year: 2017, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 70.00, Stock: 0},
		{ID: "P-0123", Make: "Hyundai", Model: "Santa Fe", Year: 2017, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 155.00, Stock: 0},
		{ID: "P-0124", Make: "Hyundai", Model: "Santa Fe", Year: 2017, Category: "A/C", Name: "Compresor de A/C", Source: "import", PriceUSD: 210.00, Stock: 0},

		// ============================================================
		// KIA SPORTAGE
		// ============================================================

		// Sportage 2017
		{ID: "P-0125", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 30.00, Stock: 10},
		{ID: "P-0126", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 28.00, Stock: 8},
		{ID: "P-0127", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 78.00, Stock: 0},
		{ID: "P-0128", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 7.00, Stock: 25},
		{ID: "P-0129", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 10.00, Stock: 15},
		{ID: "P-0130", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Filtros", Name: "Filtro de cabina", Source: "local", PriceUSD: 9.00, Stock: 15},
		{ID: "P-0131", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 68.00, Stock: 0},
		{ID: "P-0132", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 35.00, Stock: 0},
		{ID: "P-0133", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Suspensión", Name: "Link de barra estabilizadora", Source: "import", PriceUSD: 21.00, Stock: 0},
		{ID: "P-0134", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Suspensión", Name: "Cojinete de rueda", Source: "import", PriceUSD: 55.00, Stock: 0},
		{ID: "P-0135", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 27.00, Stock: 8},
		{ID: "P-0136", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Motor", Name: "Bobina de encendido", Source: "import", PriceUSD: 40.00, Stock: 4},
		{ID: "P-0137", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 60.00, Stock: 0},
		{ID: "P-0138", Make: "Kia", Model: "Sportage", Year: 2017, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 135.00, Stock: 0},
		{ID: "P-0139", Make: "Kia", Model: "Sportage", Year: 2017, Category: "A/C", Name: "Compresor de A/C", Source: "import", PriceUSD: 180.00, Stock: 0},

		// ============================================================
		// KIA RIO
		// ============================================================

		// Rio 2014
		{ID: "P-0140", Make: "Kia", Model: "Rio", Year: 2014, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 26.00, Stock: 10},
		{ID: "P-0141", Make: "Kia", Model: "Rio", Year: 2014, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 24.00, Stock: 8},
		{ID: "P-0142", Make: "Kia", Model: "Rio", Year: 2014, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 7.00, Stock: 30},
		{ID: "P-0143", Make: "Kia", Model: "Rio", Year: 2014, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 9.00, Stock: 20},
		{ID: "P-0144", Make: "Kia", Model: "Rio", Year: 2014, Category: "Filtros", Name: "Filtro de cabina", Source: "local", PriceUSD: 8.00, Stock: 15},
		{ID: "P-0145", Make: "Kia", Model: "Rio", Year: 2014, Category: "Correas", Name: "Correa serpentina", Source: "local", PriceUSD: 15.00, Stock: 17},
		{ID: "P-0146", Make: "Kia", Model: "Rio", Year: 2014, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 55.00, Stock: 0},
		{ID: "P-0147", Make: "Kia", Model: "Rio", Year: 2014, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 30.00, Stock: 0},
		{ID: "P-0148", Make: "Kia", Model: "Rio", Year: 2014, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 24.00, Stock: 8},
		{ID: "P-0149", Make: "Kia", Model: "Rio", Year: 2014, Category: "Motor", Name: "Bobina de encendido", Source: "import", PriceUSD: 35.00, Stock: 4},
		{ID: "P-0150", Make: "Kia", Model: "Rio", Year: 2014, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 52.00, Stock: 0},
		{ID: "P-0151", Make: "Kia", Model: "Rio", Year: 2014, Category: "Motor", Name: "Alternador (usado)", Source: "import", PriceUSD: 95.00, Stock: 0},

		// ============================================================
		// KIA SORENTO
		// ============================================================

		// Sorento 2017
		{ID: "P-0152", Make: "Kia", Model: "Sorento", Year: 2017, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 35.00, Stock: 8},
		{ID: "P-0153", Make: "Kia", Model: "Sorento", Year: 2017, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 30.00, Stock: 8},
		{ID: "P-0154", Make: "Kia", Model: "Sorento", Year: 2017, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 90.00, Stock: 0},
		{ID: "P-0155", Make: "Kia", Model: "Sorento", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 8.00, Stock: 25},
		{ID: "P-0156", Make: "Kia", Model: "Sorento", Year: 2017, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 12.00, Stock: 15},
		{ID: "P-0157", Make: "Kia", Model: "Sorento", Year: 2017, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 78.00, Stock: 0},
		{ID: "P-0158", Make: "Kia", Model: "Sorento", Year: 2017, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 40.00, Stock: 0},
		{ID: "P-0159", Make: "Kia", Model: "Sorento", Year: 2017, Category: "Suspensión", Name: "Cojinete de rueda", Source: "import", PriceUSD: 60.00, Stock: 0},
		{ID: "P-0160", Make: "Kia", Model: "Sorento", Year: 2017, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 70.00, Stock: 0},
		{ID: "P-0161", Make: "Kia", Model: "Sorento", Year: 2017, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 155.00, Stock: 0},
		{ID: "P-0162", Make: "Kia", Model: "Sorento", Year: 2017, Category: "A/C", Name: "Compresor de A/C", Source: "import", PriceUSD: 210.00, Stock: 0},

		// ============================================================
		// NISSAN SENTRA
		// ============================================================

		// Sentra 2017
		{ID: "P-0163", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 28.00, Stock: 10},
		{ID: "P-0164", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 26.00, Stock: 8},
		{ID: "P-0165", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 70.00, Stock: 0},
		{ID: "P-0166", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 7.00, Stock: 30},
		{ID: "P-0167", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 10.00, Stock: 15},
		{ID: "P-0168", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Filtros", Name: "Filtro de cabina", Source: "local", PriceUSD: 9.00, Stock: 15},
		{ID: "P-0169", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 62.00, Stock: 0},
		{ID: "P-0170", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 32.00, Stock: 0},
		{ID: "P-0171", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Suspensión", Name: "Link de barra estabilizadora", Source: "import", PriceUSD: 20.00, Stock: 0},
		{ID: "P-0172", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 27.00, Stock: 8},
		{ID: "P-0173", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Motor", Name: "Bobina de encendido", Source: "import", PriceUSD: 40.00, Stock: 4},
		{ID: "P-0174", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 58.00, Stock: 0},
		{ID: "P-0175", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Eléctrico", Name: "Sensor de oxígeno", Source: "import", PriceUSD: 45.00, Stock: 0},
		{ID: "P-0176", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 130.00, Stock: 0},
		{ID: "P-0177", Make: "Nissan", Model: "Sentra", Year: 2017, Category: "A/C", Name: "Compresor de A/C", Source: "import", PriceUSD: 180.00, Stock: 0},

		// ============================================================
		// NISSAN ROGUE
		// ============================================================

		// Rogue 2017
		{ID: "P-0178", Make: "Nissan", Model: "Rogue", Year: 2017, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 32.00, Stock: 10},
		{ID: "P-0179", Make: "Nissan", Model: "Rogue", Year: 2017, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 29.00, Stock: 8},
		{ID: "P-0180", Make: "Nissan", Model: "Rogue", Year: 2017, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 80.00, Stock: 0},
		{ID: "P-0181", Make: "Nissan", Model: "Rogue", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 7.00, Stock: 30},
		{ID: "P-0182", Make: "Nissan", Model: "Rogue", Year: 2017, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 11.00, Stock: 15},
		{ID: "P-0183", Make: "Nissan", Model: "Rogue", Year: 2017, Category: "Filtros", Name: "Filtro de cabina", Source: "local", PriceUSD: 10.00, Stock: 15},
		{ID: "P-0184", Make: "Nissan", Model: "Rogue", Year: 2017, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 70.00, Stock: 0},
		{ID: "P-0185", Make: "Nissan", Model: "Rogue", Year: 2017, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 36.00, Stock: 0},
		{ID: "P-0186", Make: "Nissan", Model: "Rogue", Year: 2017, Category: "Suspensión", Name: "Cojinete de rueda", Source: "import", PriceUSD: 58.00, Stock: 0},
		{ID: "P-0187", Make: "Nissan", Model: "Rogue", Year: 2017, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 28.00, Stock: 8},
		{ID: "P-0188", Make: "Nissan", Model: "Rogue", Year: 2017, Category: "Motor", Name: "Bobina de encendido", Source: "import", PriceUSD: 42.00, Stock: 4},
		{ID: "P-0189", Make: "Nissan", Model: "Rogue", Year: 2017, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 145.00, Stock: 0},
		{ID: "P-0190", Make: "Nissan", Model: "Rogue", Year: 2017, Category: "A/C", Name: "Compresor de A/C", Source: "import", PriceUSD: 195.00, Stock: 0},

		// ============================================================
		// NISSAN FRONTIER
		// ============================================================

		// Frontier 2017
		{ID: "P-0191", Make: "Nissan", Model: "Frontier", Year: 2017, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 35.00, Stock: 8},
		{ID: "P-0192", Make: "Nissan", Model: "Frontier", Year: 2017, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 95.00, Stock: 0},
		{ID: "P-0193", Make: "Nissan", Model: "Frontier", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 8.00, Stock: 25},
		{ID: "P-0194", Make: "Nissan", Model: "Frontier", Year: 2017, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 12.00, Stock: 15},
		{ID: "P-0195", Make: "Nissan", Model: "Frontier", Year: 2017, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 75.00, Stock: 0},
		{ID: "P-0196", Make: "Nissan", Model: "Frontier", Year: 2017, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 42.00, Stock: 0},
		{ID: "P-0197", Make: "Nissan", Model: "Frontier", Year: 2017, Category: "Suspensión", Name: "Link de barra estabilizadora", Source: "import", PriceUSD: 25.00, Stock: 0},
		{ID: "P-0198", Make: "Nissan", Model: "Frontier", Year: 2017, Category: "Suspensión", Name: "Cojinete de rueda", Source: "import", PriceUSD: 65.00, Stock: 0},
		{ID: "P-0199", Make: "Nissan", Model: "Frontier", Year: 2017, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 75.00, Stock: 0},
		{ID: "P-0200", Make: "Nissan", Model: "Frontier", Year: 2017, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 155.00, Stock: 0},

		// ============================================================
		// MAZDA CX-5
		// ============================================================

		// CX-5 2017
		{ID: "P-0201", Make: "Mazda", Model: "CX-5", Year: 2017, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 32.00, Stock: 8},
		{ID: "P-0202", Make: "Mazda", Model: "CX-5", Year: 2017, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 29.00, Stock: 8},
		{ID: "P-0203", Make: "Mazda", Model: "CX-5", Year: 2017, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 82.00, Stock: 0},
		{ID: "P-0204", Make: "Mazda", Model: "CX-5", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 7.00, Stock: 25},
		{ID: "P-0205", Make: "Mazda", Model: "CX-5", Year: 2017, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 11.00, Stock: 15},
		{ID: "P-0206", Make: "Mazda", Model: "CX-5", Year: 2017, Category: "Filtros", Name: "Filtro de cabina", Source: "local", PriceUSD: 10.00, Stock: 15},
		{ID: "P-0207", Make: "Mazda", Model: "CX-5", Year: 2017, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 72.00, Stock: 0},
		{ID: "P-0208", Make: "Mazda", Model: "CX-5", Year: 2017, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 38.00, Stock: 0},
		{ID: "P-0209", Make: "Mazda", Model: "CX-5", Year: 2017, Category: "Suspensión", Name: "Link de barra estabilizadora", Source: "import", PriceUSD: 22.00, Stock: 0},
		{ID: "P-0210", Make: "Mazda", Model: "CX-5", Year: 2017, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 30.00, Stock: 8},
		{ID: "P-0211", Make: "Mazda", Model: "CX-5", Year: 2017, Category: "Motor", Name: "Bobina de encendido", Source: "import", PriceUSD: 45.00, Stock: 4},
		{ID: "P-0212", Make: "Mazda", Model: "CX-5", Year: 2017, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 145.00, Stock: 0},
		{ID: "P-0213", Make: "Mazda", Model: "CX-5", Year: 2017, Category: "A/C", Name: "Compresor de A/C", Source: "import", PriceUSD: 190.00, Stock: 0},

		// ============================================================
		// MITSUBISHI OUTLANDER
		// ============================================================

		// Outlander 2017
		{ID: "P-0214", Make: "Mitsubishi", Model: "Outlander", Year: 2017, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 32.00, Stock: 8},
		{ID: "P-0215", Make: "Mitsubishi", Model: "Outlander", Year: 2017, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 28.00, Stock: 8},
		{ID: "P-0216", Make: "Mitsubishi", Model: "Outlander", Year: 2017, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 80.00, Stock: 0},
		{ID: "P-0217", Make: "Mitsubishi", Model: "Outlander", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 7.00, Stock: 25},
		{ID: "P-0218", Make: "Mitsubishi", Model: "Outlander", Year: 2017, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 11.00, Stock: 15},
		{ID: "P-0219", Make: "Mitsubishi", Model: "Outlander", Year: 2017, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 68.00, Stock: 0},
		{ID: "P-0220", Make: "Mitsubishi", Model: "Outlander", Year: 2017, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 35.00, Stock: 0},
		{ID: "P-0221", Make: "Mitsubishi", Model: "Outlander", Year: 2017, Category: "Suspensión", Name: "Link de barra estabilizadora", Source: "import", PriceUSD: 21.00, Stock: 0},
		{ID: "P-0222", Make: "Mitsubishi", Model: "Outlander", Year: 2017, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 28.00, Stock: 8},
		{ID: "P-0223", Make: "Mitsubishi", Model: "Outlander", Year: 2017, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 65.00, Stock: 0},
		{ID: "P-0224", Make: "Mitsubishi", Model: "Outlander", Year: 2017, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 140.00, Stock: 0},

		// ============================================================
		// SUZUKI SWIFT
		// ============================================================

		// Swift 2017
		{ID: "P-0225", Make: "Suzuki", Model: "Swift", Year: 2017, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 24.00, Stock: 10},
		{ID: "P-0226", Make: "Suzuki", Model: "Swift", Year: 2017, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 22.00, Stock: 8},
		{ID: "P-0227", Make: "Suzuki", Model: "Swift", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 6.00, Stock: 25},
		{ID: "P-0228", Make: "Suzuki", Model: "Swift", Year: 2017, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 8.00, Stock: 15},
		{ID: "P-0229", Make: "Suzuki", Model: "Swift", Year: 2017, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 52.00, Stock: 0},
		{ID: "P-0230", Make: "Suzuki", Model: "Swift", Year: 2017, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 28.00, Stock: 0},
		{ID: "P-0231", Make: "Suzuki", Model: "Swift", Year: 2017, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 22.00, Stock: 8},
		{ID: "P-0232", Make: "Suzuki", Model: "Swift", Year: 2017, Category: "Motor", Name: "Bobina de encendido", Source: "import", PriceUSD: 35.00, Stock: 4},
		{ID: "P-0233", Make: "Suzuki", Model: "Swift", Year: 2017, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 110.00, Stock: 0},

		// ============================================================
		// TOYOTA HILUX
		// ============================================================

		// Hilux 2017
		{ID: "P-0234", Make: "Toyota", Model: "Hilux", Year: 2017, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 38.00, Stock: 8},
		{ID: "P-0235", Make: "Toyota", Model: "Hilux", Year: 2017, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 32.00, Stock: 6},
		{ID: "P-0236", Make: "Toyota", Model: "Hilux", Year: 2017, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 105.00, Stock: 0},
		{ID: "P-0237", Make: "Toyota", Model: "Hilux", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 8.00, Stock: 25},
		{ID: "P-0238", Make: "Toyota", Model: "Hilux", Year: 2017, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 13.00, Stock: 15},
		{ID: "P-0239", Make: "Toyota", Model: "Hilux", Year: 2017, Category: "Filtros", Name: "Filtro de combustible", Source: "import", PriceUSD: 22.00, Stock: 5},
		{ID: "P-0240", Make: "Toyota", Model: "Hilux", Year: 2017, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 75.00, Stock: 0},
		{ID: "P-0241", Make: "Toyota", Model: "Hilux", Year: 2017, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 42.00, Stock: 0},
		{ID: "P-0242", Make: "Toyota", Model: "Hilux", Year: 2017, Category: "Suspensión", Name: "Link de barra estabilizadora", Source: "import", PriceUSD: 25.00, Stock: 0},
		{ID: "P-0243", Make: "Toyota", Model: "Hilux", Year: 2017, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 78.00, Stock: 0},
		{ID: "P-0244", Make: "Toyota", Model: "Hilux", Year: 2017, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 165.00, Stock: 0},

		// ============================================================
		// TOYOTA 4RUNNER
		// ============================================================

		// 4Runner 2017
		{ID: "P-0245", Make: "Toyota", Model: "4Runner", Year: 2017, Category: "Frenos", Name: "Pastillas de freno delanteras", Source: "local", PriceUSD: 42.00, Stock: 6},
		{ID: "P-0246", Make: "Toyota", Model: "4Runner", Year: 2017, Category: "Frenos", Name: "Pastillas de freno traseras", Source: "local", PriceUSD: 35.00, Stock: 6},
		{ID: "P-0247", Make: "Toyota", Model: "4Runner", Year: 2017, Category: "Frenos", Name: "Discos de freno delanteros (par)", Source: "import", PriceUSD: 120.00, Stock: 0},
		{ID: "P-0248", Make: "Toyota", Model: "4Runner", Year: 2017, Category: "Filtros", Name: "Filtro de aceite", Source: "local", PriceUSD: 8.00, Stock: 20},
		{ID: "P-0249", Make: "Toyota", Model: "4Runner", Year: 2017, Category: "Filtros", Name: "Filtro de aire del motor", Source: "local", PriceUSD: 13.00, Stock: 12},
		{ID: "P-0250", Make: "Toyota", Model: "4Runner", Year: 2017, Category: "Suspensión", Name: "Amortiguador delantero", Source: "import", PriceUSD: 85.00, Stock: 0},
		{ID: "P-0251", Make: "Toyota", Model: "4Runner", Year: 2017, Category: "Suspensión", Name: "Rótula de suspensión", Source: "import", PriceUSD: 45.00, Stock: 0},
		{ID: "P-0252", Make: "Toyota", Model: "4Runner", Year: 2017, Category: "Suspensión", Name: "Link de barra estabilizadora", Source: "import", PriceUSD: 28.00, Stock: 0},
		{ID: "P-0253", Make: "Toyota", Model: "4Runner", Year: 2017, Category: "Motor", Name: "Bujías (juego)", Source: "import", PriceUSD: 32.00, Stock: 6},
		{ID: "P-0254", Make: "Toyota", Model: "4Runner", Year: 2017, Category: "Motor", Name: "Bomba de agua", Source: "import", PriceUSD: 75.00, Stock: 0},
		{ID: "P-0255", Make: "Toyota", Model: "4Runner", Year: 2017, Category: "Eléctrico", Name: "Alternador", Source: "import", PriceUSD: 165.00, Stock: 0},

		// ============================================================
		// MOTORCYCLES
		// ============================================================

		// Honda CG
		{ID: "P-0256", Make: "Honda", Model: "CG", Year: 2018, Category: "Frenos", Name: "Pastillas de freno", Source: "local", PriceUSD: 12.00, Stock: 20},
		{ID: "P-0257", Make: "Honda", Model: "CG", Year: 2018, Category: "Motor", Name: "Bujía", Source: "local", PriceUSD: 5.00, Stock: 30},
		{ID: "P-0258", Make: "Honda", Model: "CG", Year: 2018, Category: "Filtros", Name: "Filtro de aire", Source: "local", PriceUSD: 7.00, Stock: 20},
		{ID: "P-0259", Make: "Honda", Model: "CG", Year: 2018, Category: "Transmisión", Name: "Cadena", Source: "import", PriceUSD: 25.00, Stock: 10},
		{ID: "P-0260", Make: "Honda", Model: "CG", Year: 2018, Category: "Transmisión", Name: "Kit cadena y sprockets", Source: "import", PriceUSD: 45.00, Stock: 8},
		{ID: "P-0261", Make: "Honda", Model: "CG", Year: 2018, Category: "Eléctrico", Name: "Batería", Source: "local", PriceUSD: 30.00, Stock: 5},
		{ID: "P-0262", Make: "Honda", Model: "CG", Year: 2018, Category: "Eléctrico", Name: "Regulador/rectificador", Source: "import", PriceUSD: 25.00, Stock: 4},
		{ID: "P-0263", Make: "Honda", Model: "CG", Year: 2018, Category: "Motor", Name: "Cable de clutch", Source: "local", PriceUSD: 8.00, Stock: 10},

		// Yamaha YBR
		{ID: "P-0264", Make: "Yamaha", Model: "YBR", Year: 2018, Category: "Frenos", Name: "Pastillas de freno", Source: "local", PriceUSD: 12.00, Stock: 20},
		{ID: "P-0265", Make: "Yamaha", Model: "YBR", Year: 2018, Category: "Motor", Name: "Bujía", Source: "local", PriceUSD: 5.00, Stock: 30},
		{ID: "P-0266", Make: "Yamaha", Model: "YBR", Year: 2018, Category: "Filtros", Name: "Filtro de aire", Source: "local", PriceUSD: 7.00, Stock: 20},
		{ID: "P-0267", Make: "Yamaha", Model: "YBR", Year: 2018, Category: "Transmisión", Name: "Cadena", Source: "import", PriceUSD: 25.00, Stock: 10},
		{ID: "P-0268", Make: "Yamaha", Model: "YBR", Year: 2018, Category: "Transmisión", Name: "Kit cadena y sprockets", Source: "import", PriceUSD: 45.00, Stock: 8},
		{ID: "P-0269", Make: "Yamaha", Model: "YBR", Year: 2018, Category: "Eléctrico", Name: "Batería", Source: "local", PriceUSD: 30.00, Stock: 5},
		{ID: "P-0270", Make: "Yamaha", Model: "YBR", Year: 2018, Category: "Eléctrico", Name: "Regulador/rectificador", Source: "import", PriceUSD: 25.00, Stock: 4},
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
