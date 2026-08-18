package main

import (
	"net/http"
	"strings"
)

func handleAdminLoginGet(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	if s.role() == RoleAdmin {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	render(w, r, "page_admin_login", PageData{Title: "Admin", CartCount: cartCount(s)})
}

func handleAdminLoginPost(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	user := strings.TrimSpace(r.FormValue("username"))
	pass := r.FormValue("password")
	admin, ok := store.AdminLogin(user, pass)
	if !ok {
		render(w, r, "page_admin_login", PageData{
			Title: "Admin", CartCount: cartCount(s),
			Error: "Usuario o contraseña incorrectos.",
		})
		return
	}
	s.setAuth(RoleAdmin, "", admin.ID, "")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func handleDriverLoginGet(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	if s.role() == RoleDriver {
		http.Redirect(w, r, "/driver", http.StatusSeeOther)
		return
	}
	render(w, r, "page_driver_login", PageData{Title: "Repartidor", CartCount: cartCount(s)})
}

func handleDriverLoginPost(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	user := strings.TrimSpace(r.FormValue("username"))
	pass := r.FormValue("password")
	driver, ok := store.DriverLogin(user, pass)
	if !ok {
		render(w, r, "page_driver_login", PageData{
			Title: "Repartidor", CartCount: cartCount(s),
			Error: "Usuario o contraseña incorrectos.",
		})
		return
	}
	s.setAuth(RoleDriver, "", "", driver.ID)
	http.Redirect(w, r, "/driver", http.StatusSeeOther)
}

func handleRoleLogout(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	role := s.role()
	s.clearAuth()
	switch role {
	case RoleAdmin:
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
	case RoleDriver:
		http.Redirect(w, r, "/driver/login", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/catalogo", http.StatusSeeOther)
	}
}

func handleAdminDrivers(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, r, "page_admin_drivers", PageData{
		Title: "Repartidores", CartCount: cartCount(s),
		Data: driversData{Drivers: store.AllDrivers()},
	})
}

func handleAdminDriverAddPost(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	username := strings.TrimSpace(r.FormValue("username"))
	name := strings.TrimSpace(r.FormValue("name"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	zone := strings.TrimSpace(r.FormValue("zone"))
	password := r.FormValue("password")
	if username == "" || name == "" || password == "" {
		render(w, r, "page_admin_drivers", PageData{
			Title: "Repartidores", CartCount: cartCount(s), Error: "Usuario, nombre y contraseña son obligatorios.",
			Data: driversData{Drivers: store.AllDrivers()},
		})
		return
	}
	if _, err := store.AddDriver(username, name, phone, zone, password); err != nil {
		render(w, r, "page_admin_drivers", PageData{
			Title: "Repartidores", CartCount: cartCount(s), Error: err.Error(),
			Data: driversData{Drivers: store.AllDrivers()},
		})
		return
	}
	store.LogAudit("admin", s.adminID(), "driver.create", "driver", username, name)
	http.Redirect(w, r, "/admin/drivers", http.StatusSeeOther)
}

func handleAdminDriverEditPost(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	id := r.PathValue("id")
	name := strings.TrimSpace(r.FormValue("name"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	zone := strings.TrimSpace(r.FormValue("zone"))
	active := r.FormValue("active") == "on"
	if err := store.UpdateDriver(id, name, phone, zone, active); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if pass := r.FormValue("password"); pass != "" {
		store.UpdateDriverPassword(id, pass)
	}
	store.LogAudit("admin", s.adminID(), "driver.update", "driver", id, name)
	http.Redirect(w, r, "/admin/drivers", http.StatusSeeOther)
}

func handleAdminShopCreateGet(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, r, "page_admin_shop_form", PageData{
		Title: "Crear taller", CartCount: cartCount(s),
		Data: shopFormData{IsEdit: false},
	})
}

func handleAdminShopCreatePost(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	name := strings.TrimSpace(r.FormValue("name"))
	username := strings.TrimSpace(r.FormValue("username"))
	owner := strings.TrimSpace(r.FormValue("owner"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	address := strings.TrimSpace(r.FormValue("address"))
	password := r.FormValue("password")
	if name == "" || username == "" || password == "" {
		render(w, r, "page_admin_shop_form", PageData{
			Title: "Crear taller", CartCount: cartCount(s), Error: "Nombre, usuario y contraseña son obligatorios.",
			Data: shopFormData{Shop: Shop{Name: name, Username: username, Owner: owner, Phone: phone, Address: address}, IsEdit: false},
		})
		return
	}
	sh, err := store.AddShopWithCredentials(name, username, owner, phone, address, password)
	if err != nil {
		render(w, r, "page_admin_shop_form", PageData{
			Title: "Crear taller", CartCount: cartCount(s), Error: err.Error(),
			Data: shopFormData{Shop: Shop{Name: name, Username: username, Owner: owner, Phone: phone, Address: address}, IsEdit: false},
		})
		return
	}
	store.LogAudit("admin", s.adminID(), "shop.create", "shop", sh.ID, name)
	http.Redirect(w, r, "/admin/shops", http.StatusSeeOther)
}

func handleAdminShopEditGet(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	id := r.PathValue("id")
	sh, ok := store.Shop(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	render(w, r, "page_admin_shop_form", PageData{
		Title: "Editar taller", CartCount: cartCount(s),
		Data: shopFormData{Shop: *sh, IsEdit: true},
	})
}

func handleAdminShopEditPost(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	id := r.PathValue("id")
	name := strings.TrimSpace(r.FormValue("name"))
	username := strings.TrimSpace(r.FormValue("username"))
	owner := strings.TrimSpace(r.FormValue("owner"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	address := strings.TrimSpace(r.FormValue("address"))
	active := r.FormValue("active") == "on"
	if err := store.UpdateShopDetails(id, name, username, owner, phone, address); err != nil {
		render(w, r, "page_admin_shop_form", PageData{
			Title: "Editar taller", CartCount: cartCount(s), Error: err.Error(),
			Data: shopFormData{Shop: Shop{ID: id, Name: name, Username: username, Owner: owner, Phone: phone, Address: address}, IsEdit: true},
		})
		return
	}
	if pass := r.FormValue("password"); pass != "" {
		store.UpdateShopPassword(id, pass)
	}
	if !active {
		store.DeactivateShop(id)
	}
	store.LogAudit("admin", s.adminID(), "shop.update", "shop", id, name)
	http.Redirect(w, r, "/admin/shops", http.StatusSeeOther)
}

type driversData struct {
	Drivers []*Driver
}

type shopFormData struct {
	Shop   Shop
	IsEdit bool
}
