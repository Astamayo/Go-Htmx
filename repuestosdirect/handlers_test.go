package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHandleHealthNoDB(t *testing.T) {
	// health handler needs store; skip if no DATABASE_URL
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("no database configured")
	}
	conn := os.Getenv("TEST_DATABASE_URL")
	if conn == "" {
		conn = os.Getenv("DATABASE_URL")
	}
	var err error
	store, err = NewPostgresStore(conn)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handleHealth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequirePOSTBlocksGET(t *testing.T) {
	called := false
	handler := requirePOST(func(w http.ResponseWriter, r *http.Request) { called = true })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusMethodNotAllowed || called {
		t.Fatal("GET should be blocked")
	}
}
