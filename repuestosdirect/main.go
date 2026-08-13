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

// ---------------------------------------------------------------------
// Sessions (in-memory, cookie-based). Stands in for Redis-backed
// sessions in production. See README.md.
// ---------------------------------------------------------------------

type Session struct {
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

func render(w http.ResponseWriter, page string, pd PageData) {
	// Each page is its own top-level named template (e.g. "page_catalog")
	// that includes the shared "header"/"footer" partials. Named blocks
	// in Go's html/template are global to the parsed set, so giving each
	// page a unique name avoids collisions between files.
	type ctx struct {
		Title     string
		Shop      *Shop
		CartCount int
		Error     string
		// dashboard/admin/cart/catalog specific
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

func currentShop(_ *http.Request, s *Session) *Shop {
	if s.ShopID == "" {
		return nil
	}
	sh, _ := store.Shop(s.ShopID)
	return sh
}

func cartCount(s *Session) int {
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
		s.Cart[partID]++
	}
	tmpl.ExecuteTemplate(w, "cart_badge", cartCount(s))
}

func handleCartUpdate(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	r.ParseForm()
	partID := r.FormValue("part_id")
	qty, _ := strconv.Atoi(r.FormValue("qty"))
	if qty <= 0 {
		delete(s.Cart, partID)
	} else {
		s.Cart[partID] = qty
	}
	http.Redirect(w, r, "/carrito", http.StatusSeeOther)
}

func buildCartData(s *Session, sh *Shop) cartData {
	var items []CartLine
	var total float64
	for pid, qty := range s.Cart {
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
	for pid, qty := range s.Cart {
		p, ok := store.Part(pid)
		if !ok {
			continue
		}
		items = append(items, OrderItem{PartID: p.ID, PartName: p.Name, Qty: qty, UnitUSD: p.PriceUSD})
	}
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

	// clear cart
	s.Cart = map[string]int{}

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
	var shops []*Shop
	for _, sh := range store.Shops {
		shops = append(shops, sh)
	}
	render(w, "page_login", PageData{Title: "Entrar", CartCount: cartCount(s), Data: loginData{Shops: shops}})
}

func handleLoginPost(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	r.ParseForm()
	name := r.FormValue("name")
	pass := r.FormValue("password")
	sh, ok := store.ShopByLogin(name, pass)
	if !ok {
		var shops []*Shop
		for _, x := range store.Shops {
			shops = append(shops, x)
		}
		render(w, "page_login", PageData{Title: "Entrar", CartCount: cartCount(s), Error: "Usuario o contraseña incorrectos.", Data: loginData{Shops: shops}})
		return
	}
	s.ShopID = sh.ID
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	s.ShopID = ""
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
	s.ShopID = sh.ID
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// ---------------------------------------------------------------------
// main
// ---------------------------------------------------------------------

func main() {
	loadTemplates()

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
	mux.HandleFunc("GET /admin", handleAdmin)
	mux.HandleFunc("GET /signup", handleSignupGet)
	mux.HandleFunc("POST /signup", handleSignupPost)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("RepuestosDirect escuchando en %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
