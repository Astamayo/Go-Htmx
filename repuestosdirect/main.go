package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var store *Store
var tmpl *template.Template

func loadTemplates() {
	funcMap := template.FuncMap{
		"availabilityLabel": availabilityLabel,
		"availabilityShort": availabilityShort,
		"isInStock":         isInStock,
		"mapsQuery":         mapsQuery,
		"statusBadgeClass":  statusBadgeClass,
		"lineTotal": func(qty int, unit float64) float64 {
			return float64(qty) * unit
		},
		"sub": func(a, b float64) float64 {
			return a - b
		},
		"creditTermsLabel": creditTermsLabel,
		"toJSON": func(v any) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
	}
	tmpl = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))
}

type PageData struct {
	Title     string
	Shop      *Shop
	CartCount int
	Error     string
	CSRFToken string
	Data      any
}

type inventoryData struct{ StockRows, LowStock []StockRow }
type partFormData struct {
	Part   Part
	IsEdit bool
}
type deliveryData struct{ Pending []*Order }
type deliveryPageData struct {
	Pending []*Order
	Drivers []*Driver
}
type pendingData struct{ Pending, Clients []*Shop }
type orderManageData struct{ Orders []*Order }
type orderDetailData struct {
	Order        *Order
	ShopName     string
	ShopPhone    string
	Installments []Installment
	History      []StatusHistoryEntry
}
type searchData struct {
	Query string
	Parts []Part
}
type reportsData struct {
	Summary       ReportSummary
	LowStock      []StockRow
	RecentOrders  []*Order
	SortBy        string
	RevenueMonths map[string]float64
	OrdersPerShop map[string]int
	TopParts      map[string]int
	AuditLog      []AuditEntry
}

type DriverRoute struct {
	Order *Order
	Shop  *Shop
}
type driverData struct{ Routes []DriverRoute }

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
	MinCredit       float64
}
type dashboardData struct {
	Orders          []*Order
	CreditAvailable float64
}
type confirmData struct{ Order *Order }
type adminData struct {
	Rows         []AgingInstallmentRow
	Summary      ReportSummary
	LowStock     []StockRow
}

func render(w http.ResponseWriter, r *http.Request, page string, pd PageData) {
	s := getSession(w, r)
	if pd.CSRFToken == "" {
		pd.CSRFToken = s.csrfToken()
	}

	type ctx struct {
		Title           string
		Shop            *Shop
		Role            string
		CartCount       int
		Error           string
		CSRFToken       string
		Makes           []string
		Shops           []*Shop
		Clients         []*Shop
		Items           []CartLine
		Total           float64
		CreditAvailable float64
		MinCredit       float64
		Orders          []*Order
		Order           *Order
		ShopName        string
		ShopPhone       string
		Installments    []Installment
		History         []StatusHistoryEntry
		Rows            []AgingInstallmentRow
		StockRows       []StockRow
		LowStock        []StockRow
		Pending         []*Order
		DriverRoutes    []DriverRoute
		Drivers         []*Driver
		Query           string
		Parts           []Part
		Summary         ReportSummary
		RecentOrders    []*Order
		Part            Part
		IsEdit          bool
		Completed       bool
		SortBy          string
		RevenueMonths   map[string]float64
		OrdersPerShop   map[string]int
		TopParts        map[string]int
		AuditLog        []AuditEntry
		StatusOptions   []OrderStatus
	}
	c := ctx{
		Title: pd.Title, Shop: pd.Shop, CartCount: pd.CartCount,
		Error: pd.Error, CSRFToken: pd.CSRFToken,
		Role: string(getSession(w, r).role()),
		StatusOptions: orderStatusOptions,
	}
	switch d := pd.Data.(type) {
	case catalogData:
		c.Makes = d.Makes
	case loginData:
		c.Shops = d.Shops
	case cartData:
		c.Items = d.Items
		c.Total = d.Total
		c.CreditAvailable = d.CreditAvailable
		c.MinCredit = d.MinCredit
	case dashboardData:
		c.Orders = d.Orders
		c.CreditAvailable = d.CreditAvailable
	case confirmData:
		c.Order = d.Order
	case adminData:
		c.Rows = d.Rows
		c.Summary = d.Summary
		c.LowStock = d.LowStock
	case inventoryData:
		c.StockRows = d.StockRows
		c.LowStock = d.LowStock
	case deliveryData:
		c.Pending = d.Pending
	case deliveryPageData:
		c.Pending = d.Pending
		c.Drivers = d.Drivers
	case pendingData:
		c.Shops = d.Pending
		c.Clients = d.Clients
	case orderManageData:
		c.Orders = d.Orders
	case orderDetailData:
		c.Order = d.Order
		c.ShopName = d.ShopName
		c.ShopPhone = d.ShopPhone
		c.Installments = d.Installments
		c.History = d.History
	case driverData:
		c.DriverRoutes = d.Routes
	case searchData:
		c.Query = d.Query
		c.Parts = d.Parts
	case reportsData:
		c.Summary = d.Summary
		c.LowStock = d.LowStock
		c.RecentOrders = d.RecentOrders
		c.SortBy = d.SortBy
		c.RevenueMonths = d.RevenueMonths
		c.OrdersPerShop = d.OrdersPerShop
		c.TopParts = d.TopParts
		c.AuditLog = d.AuditLog
	case partFormData:
		c.Part = d.Part
		c.IsEdit = d.IsEdit
	case tiendaOrdersData:
		c.Orders = d.Orders
		c.Completed = d.Completed
	case driversData:
		c.Drivers = d.Drivers
	case shopFormData:
		c.Shop = &d.Shop
		c.IsEdit = d.IsEdit
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, page, c); err != nil {
		logError("template error", err.Error())
	}
}

func requirePOST(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.ParseForm()
		s := getSession(w, r)
		if !validateCSRF(r, s) {
			http.Error(w, "token CSRF inválido", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func currentShop(s *Session) *Shop {
	shopID := s.shopID()
	if shopID == "" {
		return nil
	}
	sh, _ := store.Shop(shopID)
	return sh
}

func cartCount(s *Session) int {
	n := 0
	for _, q := range s.cartCopy() {
		n += q
	}
	return n
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := store.db.Ping(); err != nil {
		http.Error(w, "db unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/catalogo", http.StatusFound)
}

func handleCatalog(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, r, "page_catalog", PageData{
		Title: "Buscar repuestos", Shop: currentShop(s), CartCount: cartCount(s),
		Data: catalogData{Makes: store.Makes()},
	})
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	var parts []Part
	if q != "" {
		parts = store.SearchParts(q)
	}
	render(w, r, "page_search", PageData{
		Title: "Buscar", Shop: currentShop(s), CartCount: cartCount(s),
		Data: searchData{Query: q, Parts: parts},
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

func handlePartialSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	tmpl.ExecuteTemplate(w, "partial_parts", store.SearchParts(q))
}

func handleCartAdd(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	partID := r.FormValue("part_id")
	cart := s.cartCopy()
	if _, exists := cart[partID]; exists && r.FormValue("confirm") != "yes" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		p, _ := store.Part(partID)
		tmpl.ExecuteTemplate(w, "partial_cart_confirm", map[string]any{
			"Part": p, "Qty": cart[partID], "CSRFToken": s.csrfToken(),
		})
		return
	}
	if p, ok := store.Part(partID); ok {
		if isInStock(p.Availability) && p.Stock <= 0 {
			http.Error(w, "sin stock", http.StatusBadRequest)
			return
		}
		qty := s.addToCart(partID)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.ExecuteTemplate(w, "partial_cart_added", map[string]any{
			"Part": p, "Qty": qty, "CartCount": cartCount(s),
		})
		return
	}
	tmpl.ExecuteTemplate(w, "cart_badge", cartCount(s))
}

func handleCartRemove(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	s.removeFromCart(r.FormValue("part_id"))
	http.Redirect(w, r, "/carrito", http.StatusSeeOther)
}

func handleCartUpdate(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	partID := r.FormValue("part_id")
	qty, _ := strconv.Atoi(r.FormValue("qty"))
	s.updateCart(partID, qty)
	http.Redirect(w, r, "/carrito", http.StatusSeeOther)
}

func buildCartData(s *Session, sh *Shop) cartData {
	cartCopy := s.cartCopy()
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
	cd := cartData{Items: items, Total: total, MinCredit: minCreditOrderAmount()}
	if sh != nil {
		cd.CreditAvailable = sh.CreditLimit - sh.CreditUsed
	}
	return cd
}

func handleCartPage(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	sh := currentShop(s)
	render(w, r, "page_cart", PageData{
		Title: "Carrito", Shop: sh, CartCount: cartCount(s),
		Data: buildCartData(s, sh),
	})
}

func handleOrderPlace(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	sh := currentShop(s)
	payment := r.FormValue("payment")

	var items []OrderItem
	for pid, qty := range s.cartCopy() {
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
		render(w, r, "page_cart", PageData{
			Title: "Carrito", Shop: sh, CartCount: cartCount(s),
			Error: err.Error(), Data: cd,
		})
		return
	}

	s.clearCart()

	if sh != nil {
		msg := "Hola " + sh.Owner + ", tu pedido " + order.ID + " por $" + strconv.FormatFloat(order.Total, 'f', 2, 64) + " fue confirmado."
		if order.OnCredit {
			msg += " Plan de pago: " + creditTermsLabel(sh.CreditTerms, sh.PaymentSplits) + "."
			for _, inst := range store.InstallmentsForOrder(order.ID) {
				msg += fmt.Sprintf(" Cuota %d: $%.2f vence %s.", inst.Num, inst.Amount, inst.DueDate.Format("02/01/2006"))
			}
		}
		msg += " Estado: " + string(order.Status) + "."
		SendWhatsApp(sh.Phone, msg)
	}

	render(w, r, "page_confirmation", PageData{
		Title: "Pedido confirmado", Shop: sh, CartCount: 0,
		Data: confirmData{Order: order},
	})
}

func handleLoginGet(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	if s.role() == RoleShop {
		http.Redirect(w, r, "/tienda/pedidos", http.StatusSeeOther)
		return
	}
	render(w, r, "page_login", PageData{Title: "Entrar — Taller", CartCount: cartCount(s)})
}

func handleLoginPost(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	username := strings.TrimSpace(r.FormValue("username"))
	pass := r.FormValue("password")

	sh, ok := store.ShopByUsername(username)
	if !ok {
		render(w, r, "page_login", PageData{
			Title: "Entrar — Taller", CartCount: cartCount(s),
			Error: "Usuario o contraseña incorrectos.",
		})
		return
	}
	if !checkPassword(sh.PasswordDemo, pass) {
		render(w, r, "page_login", PageData{
			Title: "Entrar — Taller", CartCount: cartCount(s),
			Error: "Usuario o contraseña incorrectos.",
		})
		return
	}

	if len(sh.PasswordDemo) >= 4 && sh.PasswordDemo[:4] != "$2a$" {
		if hashed, err := hashPassword(pass); err == nil {
			store.UpgradePasswordHash(sh.ID, hashed)
		}
	}

	if !sh.Approved {
		render(w, r, "page_login", PageData{
			Title: "Entrar — Taller", CartCount: cartCount(s),
			Error: "Tu cuenta está pendiente de revisión.",
		})
		return
	}

	s.setAuth(RoleShop, sh.ID, "", "")
	http.Redirect(w, r, "/tienda/pedidos", http.StatusSeeOther)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	handleRoleLogout(w, r)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/tienda/pedidos", http.StatusSeeOther)
}

func handleOrderDetail(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	if s.role() == RoleShop {
		http.Redirect(w, r, "/tienda/pedidos/"+r.PathValue("id"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/catalogo", http.StatusSeeOther)
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, r, "page_admin", PageData{
		Title: "Administración", CartCount: cartCount(s),
		Data: adminData{Rows: store.AgingInstallments(), Summary: store.ReportSummary(), LowStock: store.LowStockParts()},
	})
}

func handleAdminReports(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	sortBy := r.URL.Query().Get("sort")
	orders := store.OrdersSorted(sortBy)
	recent := orders
	if len(recent) > 10 {
		recent = recent[:10]
	}
	render(w, r, "page_reports", PageData{
		Title: "Reportes", CartCount: cartCount(s),
		Data: reportsData{
			Summary: store.ReportSummary(), LowStock: store.LowStockParts(), RecentOrders: recent,
			SortBy: sortBy, RevenueMonths: store.RevenueByMonth(), OrdersPerShop: store.OrdersPerShop(),
			TopParts: store.TopSellingParts(), AuditLog: store.RecentAuditLog(30),
		},
	})
}

func handleInventory(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	low := store.LowStockParts()
	render(w, r, "page_inventory", PageData{
		Title: "Inventario", Shop: currentShop(s), CartCount: cartCount(s),
		Data: inventoryData{StockRows: store.StockReport(), LowStock: low},
	})
}

func handleDelivery(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, r, "page_delivery", PageData{
		Title: "Entregas", CartCount: cartCount(s),
		Data: deliveryPageData{Pending: store.PendingDeliveries(), Drivers: store.AllDrivers()},
	})
}

func handleDeliveryAssign(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	orderID := r.FormValue("order_id")
	driverID := r.FormValue("driver_id")
	courier := r.FormValue("courier")
	if driverID != "" {
		if d, ok := store.Driver(driverID); ok {
			courier = d.Name
		}
		store.AssignCourierAndDriver(orderID, courier, driverID)
	} else {
		store.AssignCourier(orderID, courier)
	}
	store.LogAudit("admin", s.adminID(), "order.assign_driver", "order", orderID, driverID)
	http.Redirect(w, r, "/admin/delivery", http.StatusSeeOther)
}

func parsePartForm(r *http.Request) (makeStr, model, category, name, source, availability, partNumber, oemRef, condition, description, photoURL string, year, stock, reorder int, price float64, b2bPrice *float64) {
	makeStr = r.FormValue("make")
	model = r.FormValue("model")
	category = r.FormValue("category")
	name = r.FormValue("name")
	source = r.FormValue("source")
	availability = r.FormValue("availability")
	partNumber = r.FormValue("part_number")
	oemRef = r.FormValue("oem_ref")
	condition = r.FormValue("part_condition")
	description = r.FormValue("description")
	photoURL = r.FormValue("photo_url")
	year, _ = strconv.Atoi(r.FormValue("year"))
	stock, _ = strconv.Atoi(r.FormValue("stock"))
	reorder, _ = strconv.Atoi(r.FormValue("reorder_point"))
	price, _ = strconv.ParseFloat(r.FormValue("price_usd"), 64)
	if v := strings.TrimSpace(r.FormValue("b2b_price")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			b2bPrice = &f
		}
	}
	if condition == "" {
		condition = "new"
	}
	return
}

func renderPartForm(w http.ResponseWriter, r *http.Request, title string, part Part, isEdit bool, errMsg string) {
	s := getSession(w, r)
	render(w, r, "page_inventory_add", PageData{
		Title: title, Shop: currentShop(s), CartCount: cartCount(s),
		Error: errMsg, Data: partFormData{Part: part, IsEdit: isEdit},
	})
}

func handleInventoryAddGet(w http.ResponseWriter, r *http.Request) {
	part := Part{}
	if from := strings.TrimSpace(r.URL.Query().Get("from")); from != "" {
		if p, ok := store.Part(from); ok {
			part = p
		}
	}
	renderPartForm(w, r, "Agregar Repuesto", part, false, "")
}

func handleInventoryAddPost(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	makeStr, model, category, name, source, availability, partNumber, oemRef, condition, description, photoURL, year, stock, reorder, price, b2bPrice := parsePartForm(r)
	if err := validatePartInput(makeStr, model, category, name, source, year, stock, reorder, price, availability); err != nil {
		renderPartForm(w, r, "Agregar Repuesto", Part{
			Make: makeStr, Model: model, Year: year, Category: category, Name: name,
			Source: source, PriceUSD: price, Stock: stock, ReorderPoint: reorder, Availability: availability,
			PartNumber: partNumber, OEMRef: oemRef, PartCondition: condition, Description: description, PhotoURL: photoURL, B2BPrice: b2bPrice,
		}, false, err.Error())
		return
	}
	store.AddPart(makeStr, model, category, name, source, year, stock, reorder, price, availability, partNumber, oemRef, condition, description, photoURL, b2bPrice)
	store.LogAudit("admin", s.adminID(), "part.create", "part", name, partNumber)
	http.Redirect(w, r, "/admin/inventory", http.StatusSeeOther)
}

func handleInventoryEditGet(w http.ResponseWriter, r *http.Request) {
	partID := r.PathValue("id")
	p, ok := store.Part(partID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	renderPartForm(w, r, "Editar Repuesto "+partID, p, true, "")
}

func handleInventoryEditPost(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	partID := r.PathValue("id")
	makeStr, model, category, name, source, availability, partNumber, oemRef, condition, description, photoURL, year, stock, reorder, price, b2bPrice := parsePartForm(r)
	part := Part{
		ID: partID, Make: makeStr, Model: model, Year: year, Category: category, Name: name,
		Source: source, PriceUSD: price, Stock: stock, ReorderPoint: reorder, Availability: availability,
		PartNumber: partNumber, OEMRef: oemRef, PartCondition: condition, Description: description, PhotoURL: photoURL, B2BPrice: b2bPrice,
	}
	if err := validatePartInput(makeStr, model, category, name, source, year, stock, reorder, price, availability); err != nil {
		renderPartForm(w, r, "Editar Repuesto "+partID, part, true, err.Error())
		return
	}
	if err := store.UpdatePart(partID, makeStr, model, category, name, source, year, stock, reorder, price, availability, partNumber, oemRef, condition, description, photoURL, b2bPrice); err != nil {
		renderPartForm(w, r, "Editar Repuesto "+partID, part, true, err.Error())
		return
	}
	store.LogAudit("admin", s.adminID(), "part.update", "part", partID, name)
	http.Redirect(w, r, "/admin/inventory", http.StatusSeeOther)
}

func handleSignupGet(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, r, "page_signup", PageData{Title: "Crear cuenta", CartCount: cartCount(s)})
}

func handleSignupPost(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	name := strings.TrimSpace(r.FormValue("name"))
	username := strings.TrimSpace(r.FormValue("username"))
	owner := strings.TrimSpace(r.FormValue("owner"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	address := strings.TrimSpace(r.FormValue("address"))
	password := r.FormValue("password")
	if username == "" {
		username = name
	}

	if err := validateSignup(name, owner, phone, address, password); err != nil {
		render(w, r, "page_signup", PageData{Title: "Crear cuenta", CartCount: cartCount(s), Error: err.Error()})
		return
	}
	if store.ShopNameTaken(name) {
		render(w, r, "page_signup", PageData{Title: "Crear cuenta", CartCount: cartCount(s), Error: "Ya existe un taller con ese nombre."})
		return
	}
	if _, err := store.AddShopWithCredentials(name, username, owner, phone, address, password); err != nil {
		render(w, r, "page_signup", PageData{Title: "Crear cuenta", CartCount: cartCount(s), Error: "Error al crear cuenta."})
		return
	}
	render(w, r, "page_login", PageData{
		Title: "Entrar — Taller", CartCount: cartCount(s),
		Error: "¡Cuenta creada! Espera a que un administrador la apruebe.",
	})
}

func handleAdminShops(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, r, "page_admin_shops", PageData{
		Title: "Talleres", Shop: currentShop(s), CartCount: cartCount(s),
		Data: pendingData{Pending: store.PendingShops(), Clients: store.ActiveClients()},
	})
}

func handleAdminShopApprove(w http.ResponseWriter, r *http.Request) {
	store.ApproveShop(r.FormValue("shop_id"))
	http.Redirect(w, r, "/admin/shops", http.StatusSeeOther)
}

func handleAdminShopCredit(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	shopID := r.FormValue("shop_id")
	limit, _ := strconv.ParseFloat(r.FormValue("credit_limit"), 64)
	terms, _ := strconv.Atoi(r.FormValue("credit_terms"))
	splits, _ := strconv.Atoi(r.FormValue("payment_splits"))
	reminder, _ := strconv.Atoi(r.FormValue("reminder_days"))

	if err := validateShopCredit(limit, terms, splits, reminder); err != nil {
		render(w, r, "page_admin_shops", PageData{
			Title: "Talleres", Shop: currentShop(s), CartCount: cartCount(s), Error: err.Error(),
			Data: pendingData{Pending: store.PendingShops(), Clients: store.ActiveClients()},
		})
		return
	}
	if err := store.UpdateShopCredit(shopID, limit, terms, splits, reminder); err != nil {
		render(w, r, "page_admin_shops", PageData{
			Title: "Talleres", Shop: currentShop(s), CartCount: cartCount(s), Error: err.Error(),
			Data: pendingData{Pending: store.PendingShops(), Clients: store.ActiveClients()},
		})
		return
	}
	http.Redirect(w, r, "/admin/shops", http.StatusSeeOther)
}

func handleAdminInstallmentPay(w http.ResponseWriter, r *http.Request) {
	installmentID, _ := strconv.ParseInt(r.FormValue("installment_id"), 10, 64)
	if err := store.MarkInstallmentPaid(installmentID); err != nil {
		logError("installment pay failed", err.Error())
	}
	redirect := r.FormValue("redirect")
	if redirect == "" {
		redirect = "/admin"
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func handleCronReminders(w http.ResponseWriter, r *http.Request) {
	secret := os.Getenv("CRON_SECRET")
	if secret == "" || r.URL.Query().Get("secret") != secret {
		http.Error(w, "no autorizado", http.StatusUnauthorized)
		return
	}
	sent, err := store.SendPaymentReminders()
	if err != nil {
		logError("reminders failed", err.Error())
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "sent %d reminders", sent)
}

func handleAdminShopRemove(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	shopID := r.FormValue("shop_id")
	if err := store.RemoveShop(shopID); err != nil {
		render(w, r, "page_admin_shops", PageData{
			Title: "Talleres", Shop: currentShop(s), CartCount: cartCount(s),
			Error: err.Error(),
			Data:  pendingData{Pending: store.PendingShops(), Clients: store.ActiveClients()},
		})
		return
	}
	http.Redirect(w, r, "/admin/shops", http.StatusSeeOther)
}

func handleDriverDeliver(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	orderID := r.FormValue("order_id")
	status := OrderStatus(r.FormValue("status"))
	if status == "" {
		status = StatusLlegado
	}
	order, err := store.Order(orderID)
	if err != nil {
		http.Redirect(w, r, "/driver", http.StatusSeeOther)
		return
	}
	if order.DriverID != "" && order.DriverID != s.driverID() {
		http.Redirect(w, r, "/driver", http.StatusSeeOther)
		return
	}
	if order.DriverID == "" {
		store.AssignDriverToOrder(orderID, s.driverID())
	}
	if _, err := store.UpdateOrderStatus(orderID, status, actorLabel(s)); err != nil {
		logError("driver deliver failed", err.Error())
	}
	if order.ShopID != "" && (status == StatusLlegado || status == StatusLlegadoLegacy) {
		if sh, ok := store.Shop(order.ShopID); ok {
			SendWhatsApp(sh.Phone, "Tu pedido "+orderID+" fue entregado. ¡Gracias!")
		}
	}
	http.Redirect(w, r, "/driver", http.StatusSeeOther)
}

func handleDriverDashboard(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	driverID := s.driverID()
	var routes []DriverRoute
	for _, o := range store.DriverActiveOrders(driverID) {
		sh, ok := store.Shop(o.ShopID)
		if !ok {
			continue
		}
		routes = append(routes, DriverRoute{Order: o, Shop: sh})
	}
	render(w, r, "page_driver", PageData{
		Title: "Ruta de Repartidor", CartCount: cartCount(s),
		Data: driverData{Routes: routes},
	})
}

func handleAdminOrders(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	completed := r.URL.Query().Get("view") == "completed"
	orders := store.ActiveOrdersAll()
	if completed {
		orders = store.CompletedOrdersAll()
	}
	render(w, r, "page_admin_orders", PageData{
		Title: "Gestión de Pedidos", CartCount: cartCount(s),
		Data: tiendaOrdersData{Orders: orders, Completed: completed},
	})
}

func handleAdminOrderDetail(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	orderID := r.PathValue("id")
	order, err := store.Order(orderID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	shopName := order.ShopID
	shopPhone := ""
	if sh, ok := store.Shop(order.ShopID); ok {
		shopName = sh.Name
		shopPhone = sh.Phone
	}
	render(w, r, "page_admin_order_detail", PageData{
		Title: "Pedido " + orderID, CartCount: cartCount(s),
		Data: orderDetailData{
			Order: order, ShopName: shopName, ShopPhone: shopPhone,
			Installments: order.Installments, History: store.OrderStatusHistory(orderID),
		},
	})
}

func handleAdminOrderStatus(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	orderID := r.FormValue("order_id")
	newStatus := OrderStatus(r.FormValue("status"))
	order, err := store.UpdateOrderStatus(orderID, newStatus, actorLabel(s))
	if err != nil {
		logError("order status update failed", err.Error())
		http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
		return
	}

	if newStatus == StatusListo {
		notifyDriverReady(order)
	}
	store.LogAudit("admin", s.adminID(), "order.status", "order", orderID, string(newStatus))

	if order != nil && order.ShopID != "" {
		if sh, ok := store.Shop(order.ShopID); ok {
			msg := "Actualización pedido " + orderID + ": ahora está en estado \"" + string(newStatus) + "\"."
			SendWhatsApp(sh.Phone, msg)
		}
	}

	redirect := r.FormValue("redirect")
	if redirect == "" {
		redirect = "/admin/orders"
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func handleAdminOrderPay(w http.ResponseWriter, r *http.Request) {
	orderID := r.FormValue("order_id")
	if err := store.MarkOrderPaid(orderID); err != nil {
		logError("mark paid failed", err.Error())
	}
	redirect := r.FormValue("redirect")
	if redirect == "" {
		redirect = "/admin"
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func handleAdminInventoryDelete(w http.ResponseWriter, r *http.Request) {
	if err := store.DeletePart(r.FormValue("part_id")); err != nil {
		logError("delete part failed", err.Error())
	}
	http.Redirect(w, r, "/admin/inventory", http.StatusSeeOther)
}

func main() {
	loadTemplates()

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		logError("startup failed", "DATABASE_URL required")
		os.Exit(1)
	}

	var err error
	store, err = NewPostgresStore(connStr)
	if err != nil {
		logError("db init failed", err.Error())
		os.Exit(1)
	}
	store.CleanExpiredSessions()

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	mux.HandleFunc("GET /healthz", handleHealth)
	mux.HandleFunc("GET /{$}", handleHome)
	mux.HandleFunc("GET /catalogo", handleCatalog)
	mux.HandleFunc("GET /buscar", handleSearch)
	mux.HandleFunc("GET /partials/models", handlePartialModels)
	mux.HandleFunc("GET /partials/years", handlePartialYears)
	mux.HandleFunc("GET /partials/parts", handlePartialParts)
	mux.HandleFunc("GET /partials/search", handlePartialSearch)
	mux.HandleFunc("POST /cart/add", requirePOST(handleCartAdd))
	mux.HandleFunc("POST /cart/remove", requirePOST(handleCartRemove))
	mux.HandleFunc("POST /cart/update", requirePOST(handleCartUpdate))
	mux.HandleFunc("GET /carrito", handleCartPage)
	mux.HandleFunc("POST /order/place", requirePOST(handleOrderPlace))
	mux.HandleFunc("GET /login", handleLoginGet)
	mux.HandleFunc("POST /login", requirePOST(handleLoginPost))
	mux.HandleFunc("GET /logout", handleLogout)

	mux.HandleFunc("GET /admin/login", handleAdminLoginGet)
	mux.HandleFunc("POST /admin/login", requirePOST(handleAdminLoginPost))
	mux.HandleFunc("GET /driver/login", handleDriverLoginGet)
	mux.HandleFunc("POST /driver/login", requirePOST(handleDriverLoginPost))

	mux.HandleFunc("GET /tienda/pedidos", requireShop(handleTiendaPedidos))
	mux.HandleFunc("GET /tienda/pedidos/historial", requireShop(handleTiendaHistorial))
	mux.HandleFunc("GET /tienda/pedidos/{id}", requireShop(handleTiendaOrderDetail))
	mux.HandleFunc("GET /tienda/entregas", requireShop(handleTiendaEntregas))

	mux.HandleFunc("GET /admin", requireAdmin(handleAdmin))
	mux.HandleFunc("GET /admin/reports", requireAdmin(handleAdminReports))
	mux.HandleFunc("GET /admin/inventory", requireAdmin(handleInventory))
	mux.HandleFunc("GET /admin/delivery", requireAdmin(handleDelivery))
	mux.HandleFunc("POST /admin/delivery/assign", requireAdmin(requirePOST(handleDeliveryAssign)))
	mux.HandleFunc("GET /admin/inventory/add", requireAdmin(handleInventoryAddGet))
	mux.HandleFunc("POST /admin/inventory/add", requireAdmin(requirePOST(handleInventoryAddPost)))
	mux.HandleFunc("GET /admin/inventory/edit/{id}", requireAdmin(handleInventoryEditGet))
	mux.HandleFunc("POST /admin/inventory/edit/{id}", requireAdmin(requirePOST(handleInventoryEditPost)))
	mux.HandleFunc("GET /admin/shops", requireAdmin(handleAdminShops))
	mux.HandleFunc("GET /admin/shops/create", requireAdmin(handleAdminShopCreateGet))
	mux.HandleFunc("POST /admin/shops/create", requireAdmin(requirePOST(handleAdminShopCreatePost)))
	mux.HandleFunc("GET /admin/shops/edit/{id}", requireAdmin(handleAdminShopEditGet))
	mux.HandleFunc("POST /admin/shops/edit/{id}", requireAdmin(requirePOST(handleAdminShopEditPost)))
	mux.HandleFunc("GET /admin/drivers", requireAdmin(handleAdminDrivers))
	mux.HandleFunc("POST /admin/drivers/add", requireAdmin(requirePOST(handleAdminDriverAddPost)))
	mux.HandleFunc("POST /admin/drivers/edit/{id}", requireAdmin(requirePOST(handleAdminDriverEditPost)))
	mux.HandleFunc("POST /admin/shops/approve", requireAdmin(requirePOST(handleAdminShopApprove)))
	mux.HandleFunc("POST /admin/shops/credit", requireAdmin(requirePOST(handleAdminShopCredit)))
	mux.HandleFunc("POST /admin/shops/remove", requireAdmin(requirePOST(handleAdminShopRemove)))
	mux.HandleFunc("POST /admin/installments/pay", requireAdmin(requirePOST(handleAdminInstallmentPay)))
	mux.HandleFunc("GET /cron/reminders", handleCronReminders)
	mux.HandleFunc("GET /admin/orders", requireAdmin(handleAdminOrders))
	mux.HandleFunc("GET /admin/orders/{id}", requireAdmin(handleAdminOrderDetail))
	mux.HandleFunc("POST /admin/orders/status", requireAdmin(requirePOST(handleAdminOrderStatus)))
	mux.HandleFunc("POST /admin/orders/pay", requireAdmin(requirePOST(handleAdminOrderPay)))
	mux.HandleFunc("POST /admin/inventory/delete", requireAdmin(requirePOST(handleAdminInventoryDelete)))
	mux.HandleFunc("GET /driver", requireDriver(handleDriverDashboard))
	mux.HandleFunc("POST /driver/deliver", requireDriver(requirePOST(handleDriverDeliver)))
	mux.HandleFunc("POST /driver/status", requireDriver(requirePOST(handleDriverStatus)))

	mux.HandleFunc("GET /dashboard", handleDashboard)
	mux.HandleFunc("GET /pedido/{id}", handleOrderDetail)
	mux.HandleFunc("GET /signup", handleSignupGet)
	mux.HandleFunc("POST /signup", requirePOST(handleSignupPost))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	logInfo("RepuestosDirect listening", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logError("server error", err.Error())
		os.Exit(1)
	}
}
