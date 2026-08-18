package main

import (
	"net/http"
)

type Role string

const (
	RoleGuest  Role = "guest"
	RoleShop   Role = "shop"
	RoleAdmin  Role = "admin"
	RoleDriver Role = "driver"
)

func (s *Session) role() Role {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Role == "" {
		return RoleGuest
	}
	return s.Role
}

func (s *Session) setAuth(role Role, shopID, adminID, driverID string) {
	s.mu.Lock()
	s.Role = role
	s.ShopID = shopID
	s.AdminID = adminID
	s.DriverID = driverID
	s.mu.Unlock()
	s.persist()
}

func (s *Session) clearAuth() {
	s.setAuth(RoleGuest, "", "", "")
}

func (s *Session) adminID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AdminID
}

func (s *Session) driverID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.DriverID
}

func requireRole(roles ...Role) func(http.HandlerFunc) http.HandlerFunc {
	allowed := map[Role]bool{}
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			s := getSession(w, r)
			if !allowed[s.role()] {
				http.Redirect(w, r, loginPathForRole(roles[0]), http.StatusSeeOther)
				return
			}
			next(w, r)
		}
	}
}

func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return requireRole(RoleAdmin)(next)
}

func requireShop(next http.HandlerFunc) http.HandlerFunc {
	return requireRole(RoleShop)(next)
}

func requireDriver(next http.HandlerFunc) http.HandlerFunc {
	return requireRole(RoleDriver)(next)
}

func loginPathForRole(role Role) string {
	switch role {
	case RoleAdmin:
		return "/admin/login"
	case RoleDriver:
		return "/driver/login"
	case RoleShop:
		return "/login"
	default:
		return "/login"
	}
}

func currentAdmin(s *Session) *Admin {
	if s.adminID() == "" {
		return nil
	}
	a, _ := store.Admin(s.adminID())
	return a
}

func currentDriver(s *Session) *Driver {
	if s.driverID() == "" {
		return nil
	}
	d, _ := store.Driver(s.driverID())
	return d
}
