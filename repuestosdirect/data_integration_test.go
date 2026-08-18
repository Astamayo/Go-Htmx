//go:build integration

package main

import (
	"os"
	"testing"
)

func TestStorePlaceOrderAndRemoveShop(t *testing.T) {
	conn := os.Getenv("TEST_DATABASE_URL")
	if conn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	store, err := NewPostgresStore(conn)
	if err != nil {
		t.Fatal(err)
	}

	sh, err := store.AddShop("Test Shop IT", "Owner IT", "+18095559999", "Test Address", "testpass123")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ApproveShop(sh.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.AddPart("Toyota", "Test", "Motor", "Filtro IT", "OEM", 2020, 10, 2, 5.0, "En stock", "", "", "new", "", "", nil); err != nil {
		t.Fatal(err)
	}
	parts := store.SearchParts("Filtro IT")
	if len(parts) == 0 {
		t.Fatal("expected seeded part")
	}

	order, err := store.PlaceOrder(sh.ID, []OrderItem{{
		PartID: parts[0].ID, PartName: parts[0].Name, Qty: 2, UnitUSD: parts[0].PriceUSD,
	}}, false)
	if err != nil {
		t.Fatalf("place order: %v", err)
	}
	if order.Total != 10.0 {
		t.Fatalf("expected total 10, got %v", order.Total)
	}

	p, _ := store.Part(parts[0].ID)
	if p.Stock != 8 {
		t.Fatalf("expected stock 8 after order, got %d", p.Stock)
	}

	if err := store.MarkOrderPaid(order.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.RemoveShop(sh.ID); err != nil {
		t.Fatalf("remove shop: %v", err)
	}
}
