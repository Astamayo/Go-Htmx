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
