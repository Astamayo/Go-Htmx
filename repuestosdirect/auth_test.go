package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	hashed, err := hashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if !checkPassword(hashed, "secret123") {
		t.Fatal("bcrypt password should match")
	}
	if checkPassword(hashed, "wrong") {
		t.Fatal("bcrypt password should not match wrong password")
	}
}

func TestLegacyPlainPassword(t *testing.T) {
	if !checkPassword("1234", "1234") {
		t.Fatal("legacy plain password should still work")
	}
	if checkPassword("1234", "9999") {
		t.Fatal("legacy plain password should reject wrong password")
	}
}

func TestValidateCSRFRequest(t *testing.T) {
	s := &Session{CSRFToken: "tok"}
	req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader("csrf_token=tok"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ParseForm()
	if !validateCSRF(req, s) {
		t.Fatal("expected valid token")
	}
	req, _ = http.NewRequest(http.MethodPost, "/", strings.NewReader("csrf_token=bad"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ParseForm()
	if validateCSRF(req, s) {
		t.Fatal("expected invalid token")
	}
}

func TestMapsQueryUsesAddress(t *testing.T) {
	got := mapsQuery("Av. Winston Churchill 123", "Taller X")
	if !strings.Contains(got, "Winston") {
		t.Fatalf("expected address in query, got %q", got)
	}
}
