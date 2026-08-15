package main

import (
	"crypto/subtle"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func checkPassword(stored, plain string) bool {
	if len(stored) >= 4 && stored[:4] == "$2a$" {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(plain)) == nil
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(plain)) == 1
}

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

func requireDriver(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		want := os.Getenv("DRIVER_PASSWORD")
		if want == "" {
			// Fall back to admin password if driver password not set.
			requireAdmin(next)(w, r)
			return
		}
		_, pass, ok := r.BasicAuth()
		if !ok || pass != want {
			w.Header().Set("WWW-Authenticate", `Basic realm="driver"`)
			http.Error(w, "no autorizado", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func validateCSRF(r *http.Request, s *Session) bool {
	if s == nil {
		return false
	}
	token := r.FormValue("csrf_token")
	if token == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.CSRFToken)) == 1
}

func isProduction() bool {
	return os.Getenv("RENDER") != "" || os.Getenv("GO_ENV") == "production"
}
