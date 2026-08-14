package main

import (
	"crypto/rand"
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Global store reference
var store *Store

// ---------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------

type Session struct {
	Mu     sync.RWMutex
	ShopID string
	Cart   map[string]int // partID -> qty
}

var (
	sessMu sync.Mutex
	sess   = map[string]*Session{}
)

func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func getSession(w http.ResponseWriter, r *http.Request) *Session {
	c, err := r.Cookie("sid")
	if err == nil {
		sessMu.Lock()
		s, ok := sess[c.Value]
		sessMu.Unlock()
		if ok {
			return s
		}
	}
	sid := newSessionID()
	http.SetCookie(w, &http.Cookie{Name: "sid", Value: sid, Path: "/", HttpOnly: true, MaxAge: 86400 * 7})
	s := &Session{Cart: map[string]int{}}
	sessMu.Lock()
	sess[sid] = s
	sessMu.Unlock()
	return s
}

// ---------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------

var tmpl *template.Template

func loadTemplates() {
	tmpl = template.Must(template.ParseGlob("templates/*.html"))
}

type PageData struct {
	Title     string
	Shop      *Shop
	CartCount int
	Error     string
	Data      any
}

type inventoryData struct{ StockRows []StockRow }
type deliveryData struct{ Pending []*Order }

func render(w http.ResponseWriter, page string, pd PageData) {
	type ctx struct {
		Title           string
		Shop            *Shop
		CartCount       int
		Error           string
		Makes           []string
		Shops           []*Shop
		Items           []CartLine
		Total           float64
		CreditAvailable float64
		Orders          []*Order
		Order           *Order
		Rows            []AgingRow
	}
	c := ctx{Title: pd.Title, Shop: pd.Shop, CartCount: pd.CartCount, Error: pd.Error}
	switch d := pd.Data.(type) {
	case catalogData:
		c.Makes = d.Makes
	case loginData:
		c.Shops = d.Shops
	case cartData:
		c.Items = d.Items
		c.Total = d.Total
		c.CreditAvailable = d.CreditAvailable
	case dashboardData:
		c.Orders = d.Orders
		c.CreditAvailable = d.CreditAvailable
	case confirmData:
		c.Order = d.Order
	case adminData:
		c.Rows = d.Rows
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, page, c); err != nil {
		log.Println("template error:", err)
	}
}

type catalogData struct{ Makes []string }
type loginData struct{ Shops []*Shop }
type CartLine struct {
	Part     Part
	Qty      int
	Subtotal float64
}
type cartData struct {
	Items           []CartLine
	Total           float64
	CreditAvailable float64
}
type dashboardData struct {
	Orders          []*Order
	CreditAvailable float64
}
type confirmData struct{ Order *Order }
type AgingRow struct {
	Order       *Order
	ShopName    string
	Overdue     bool
	DaysOverdue int
}
type adminData struct{ Rows []AgingRow }

// ---------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------

func handleInventory(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, "page_inventory", PageData{Title: "Inventario", Shop: currentShop(r, s), CartCount: cartCount(s),
		Data: inventoryData{StockRows: store.StockReport()}})
}

func handleDelivery(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, "page_delivery", PageData{Title: "Entregas", Shop: currentShop(r, s), CartCount: cartCount(s),
		Data: deliveryData{Pending: store.PendingDeliveries()}})
}

func handleDeliveryAssign(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	store.AssignCourier(r.FormValue("order_id"), r.FormValue("courier"))
	http.Redirect(w, r, "/admin/delivery", http.StatusSeeOther)
}

// Basic auth gate for everything under /admin -- minimal but real.
// Set ADMIN_PASSWORD in Render's env vars.
func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		want := os.Getenv("ADMIN_PASSWORD")
		if want == "" {
			http.Error(w, "ADMIN_PASSWORD no configurado", http.StatusInternalServerError)
			return
		}
		_, pass, ok := r.BasicAuth()
		if !ok || pass != want {
			w.Header().Set("WWW-Authenticate", `Basic realm="admin"`)
			http.Error(w, "no autorizado", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func currentShop(_ *http.Request, s *Session) *Shop {
	s.Mu.RLock()
	shopID := s.ShopID
	s.Mu.RUnlock()

	if shopID == "" {
		return nil
	}
	sh, _ := store.Shop(shopID)
	return sh
}

func cartCount(s *Session) int {
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	n := 0
	for _, q := range s.Cart {
		n += q
	}
	return n
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/catalogo", http.StatusFound)
}

func handleCatalog(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, "page_catalog", PageData{
		Title: "Buscar repuestos", Shop: currentShop(r, s), CartCount: cartCount(s),
		Data: catalogData{Makes: store.Makes()},
	})
}

func handlePartialModels(w http.ResponseWriter, r *http.Request) {
	make := r.URL.Query().Get("make")
	tmpl.ExecuteTemplate(w, "partial_models", store.Models(make))
}

func handlePartialYears(w http.ResponseWriter, r *http.Request) {
	make := r.URL.Query().Get("make")
	model := r.URL.Query().Get("model")
	tmpl.ExecuteTemplate(w, "partial_years", store.Years(make, model))
}

func handlePartialParts(w http.ResponseWriter, r *http.Request) {
	make := r.URL.Query().Get("make")
	model := r.URL.Query().Get("model")
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	tmpl.ExecuteTemplate(w, "partial_parts", store.PartsFor(make, model, year))
}

func handleCartAdd(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	r.ParseForm()
	partID := r.FormValue("part_id")
	if _, ok := store.Part(partID); ok {
		s.Mu.Lock()
		s.Cart[partID]++
		s.Mu.Unlock()
	}
	tmpl.ExecuteTemplate(w, "cart_badge", cartCount(s))
}

func handleCartUpdate(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	r.ParseForm()
	partID := r.FormValue("part_id")
	qty, _ := strconv.Atoi(r.FormValue("qty"))

	// Add Lock here!
	s.Mu.Lock()
	if qty <= 0 {
		delete(s.Cart, partID)
	} else {
		s.Cart[partID] = qty
	}
	s.Mu.Unlock()
	// Unlock before redirecting

	http.Redirect(w, r, "/carrito", http.StatusSeeOther)
}

func buildCartData(s *Session, sh *Shop) cartData {
	s.Mu.RLock()
	cartCopy := make(map[string]int, len(s.Cart))
	for k, v := range s.Cart {
		cartCopy[k] = v
	}
	s.Mu.RUnlock()

	var items []CartLine
	var total float64
	for pid, qty := range cartCopy {
		p, ok := store.Part(pid)
		if !ok {
			continue
		}
		sub := p.PriceUSD * float64(qty)
		items = append(items, CartLine{Part: p, Qty: qty, Subtotal: sub})
		total += sub
	}
	cd := cartData{Items: items, Total: total}
	if sh != nil {
		cd.CreditAvailable = sh.CreditLimit - sh.CreditUsed
	}
	return cd
}

func handleCartPage(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	sh := currentShop(r, s)
	render(w, "page_cart", PageData{
		Title: "Carrito", Shop: sh, CartCount: cartCount(s),
		Data: buildCartData(s, sh),
	})
}

func handleOrderPlace(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	sh := currentShop(r, s)
	r.ParseForm()
	payment := r.FormValue("payment")

	var items []OrderItem

	// Add the read lock here!
	s.Mu.RLock()
	for pid, qty := range s.Cart {
		p, ok := store.Part(pid)
		if !ok {
			continue
		}
		items = append(items, OrderItem{PartID: p.ID, PartName: p.Name, Qty: qty, UnitUSD: p.PriceUSD})
	}
	s.Mu.RUnlock()

	if len(items) == 0 {
		http.Redirect(w, r, "/catalogo", http.StatusSeeOther)
		return
	}

	shopID := ""
	onCredit := false
	if sh != nil {
		shopID = sh.ID
		onCredit = payment == "credit"
	}

	order, err := store.PlaceOrder(shopID, items, onCredit)
	if err != nil {
		cd := buildCartData(s, sh)
		render(w, "page_cart", PageData{Title: "Carrito", Shop: sh, CartCount: cartCount(s), Error: err.Error(), Data: cd})
		return
	}

	// Clear the cart after placing the order
	s.Mu.Lock()
	s.Cart = map[string]int{}
	s.Mu.Unlock()

	if sh != nil {
		msg := "Hola " + sh.Owner + ", tu pedido " + order.ID + " por $" + strconv.FormatFloat(order.Total, 'f', 2, 64) + " fue confirmado. Estado: " + string(order.Status) + "."
		SendWhatsApp(sh.Phone, msg)
	}

	render(w, "page_confirmation", PageData{
		Title: "Pedido confirmado", Shop: sh, CartCount: 0,
		Data: confirmData{Order: order},
	})
}

func handleLoginGet(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	shops := store.AllShops()
	render(w, "page_login", PageData{Title: "Entrar", CartCount: cartCount(s), Data: loginData{Shops: shops}})
}

func handleLoginPost(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	r.ParseForm()

	name := r.FormValue("name")
	pass := r.FormValue("password")
	sh, ok := store.ShopByLogin(name, pass)
	if !ok {
		shops := store.AllShops()
		render(w, "page_login", PageData{Title: "Entrar", CartCount: cartCount(s), Error: "Usuario o contraseña incorrectos.", Data: loginData{Shops: shops}})
		return
	}

	s.Mu.Lock()
	s.ShopID = sh.ID
	s.Mu.Unlock()

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)

	s.Mu.Lock()
	s.ShopID = ""
	s.Mu.Unlock()

	http.Redirect(w, r, "/catalogo", http.StatusSeeOther)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	sh := currentShop(r, s)

	if sh == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	orders := store.OrdersFor(sh.ID)
	render(w, "page_dashboard", PageData{
		Title: "Panel", Shop: sh, CartCount: cartCount(s),
		Data: dashboardData{Orders: orders, CreditAvailable: sh.CreditLimit - sh.CreditUsed},
	})
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	sh := currentShop(r, s)
	var rows []AgingRow
	now := time.Now()
	for _, o := range store.AllOrdersSortedByDue() {
		shx, _ := store.Shop(o.ShopID)
		name := o.ShopID
		if shx != nil {
			name = shx.Name
		}
		overdue := now.After(o.DueDate)
		days := 0
		if overdue {
			days = int(now.Sub(o.DueDate).Hours() / 24)
		}
		rows = append(rows, AgingRow{Order: o, ShopName: name, Overdue: overdue, DaysOverdue: days})
	}
	render(w, "page_admin", PageData{
		Title: "Administración", Shop: sh, CartCount: cartCount(s),
		Data: adminData{Rows: rows},
	})
}

func handleInventoryAddGet(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, "page_inventory_add", PageData{
		Title:     "Agregar Repuesto",
		Shop:      currentShop(r, s),
		CartCount: cartCount(s),
	})
}

func handleInventoryAddPost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	makeStr := r.FormValue("make")
	model := r.FormValue("model")
	category := r.FormValue("category")
	name := r.FormValue("name")
	source := r.FormValue("source")

	year, _ := strconv.Atoi(r.FormValue("year"))
	stock, _ := strconv.Atoi(r.FormValue("stock"))
	reorder, _ := strconv.Atoi(r.FormValue("reorder_point"))
	price, _ := strconv.ParseFloat(r.FormValue("price_usd"), 64)

	err := store.AddPart(makeStr, model, category, name, source, year, stock, reorder, price)
	if err != nil {
		log.Println("Error adding part:", err)
		// You could render the form again with an error message here
	}

	// Redirect back to the inventory table after saving
	http.Redirect(w, r, "/admin/inventory", http.StatusSeeOther)
}

func handleSignupGet(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, "page_signup", PageData{Title: "Crear cuenta", CartCount: cartCount(s)})
}

func handleSignupPost(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	owner := strings.TrimSpace(r.FormValue("owner"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	password := r.FormValue("password")

	if name == "" || owner == "" || phone == "" || password == "" {
		render(w, "page_signup", PageData{Title: "Crear cuenta", CartCount: cartCount(s), Error: "Completa todos los campos."})
		return
	}
	if store.ShopNameTaken(name) {
		render(w, "page_signup", PageData{Title: "Crear cuenta", CartCount: cartCount(s), Error: "Ya existe un taller con ese nombre."})
		return
	}

	sh := store.AddShop(name, owner, phone, password)
	s.Mu.Lock()
	s.ShopID = sh.ID
	s.Mu.Unlock()
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// ---------------------------------------------------------------------
// main
// ---------------------------------------------------------------------

func main() {
	loadTemplates()

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	var err error
	store, err = NewPostgresStore(connStr)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleHome)
	mux.HandleFunc("GET /catalogo", handleCatalog)
	mux.HandleFunc("GET /partials/models", handlePartialModels)
	mux.HandleFunc("GET /partials/years", handlePartialYears)
	mux.HandleFunc("GET /partials/parts", handlePartialParts)
	mux.HandleFunc("POST /cart/add", handleCartAdd)
	mux.HandleFunc("POST /cart/update", handleCartUpdate)
	mux.HandleFunc("GET /carrito", handleCartPage)
	mux.HandleFunc("POST /order/place", handleOrderPlace)
	mux.HandleFunc("GET /login", handleLoginGet)
	mux.HandleFunc("POST /login", handleLoginPost)
	mux.HandleFunc("GET /logout", handleLogout)
	mux.HandleFunc("GET /dashboard", handleDashboard)
	mux.HandleFunc("GET /signup", handleSignupGet)
	mux.HandleFunc("POST /signup", handleSignupPost)

	// Admin routes protected by BasicAuth middleware
	mux.HandleFunc("GET /admin", requireAdmin(handleAdmin))
	mux.HandleFunc("GET /admin/inventory", requireAdmin(handleInventory))
	mux.HandleFunc("GET /admin/delivery", requireAdmin(handleDelivery))
	mux.HandleFunc("POST /admin/delivery/assign", requireAdmin(handleDeliveryAssign))

	// page_inventory_add
	mux.HandleFunc("GET /admin/inventory/add", requireAdmin(handleInventoryAddGet))
	mux.HandleFunc("POST /admin/inventory/add", requireAdmin(handleInventoryAddPost))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("RepuestosDirect escuchando en %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
