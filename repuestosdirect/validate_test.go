package main

import "testing"

func TestIsInStock(t *testing.T) {
	if !isInStock("En stock") {
		t.Fatal("expected En stock to be in stock")
	}
	if isInStock("Por pedido — 3 a 4 semanas") {
		t.Fatal("expected order-only availability to not be in stock")
	}
}

func TestAvailabilityLabel(t *testing.T) {
	if got := availabilityLabel("En stock"); got == "" {
		t.Fatal("expected non-empty label")
	}
	if got := availabilityShort("Por pedido — 3 a 4 semanas"); got != "por pedido, 3 a 4 semanas" {
		t.Fatalf("unexpected short label: %q", got)
	}
}

func TestValidatePartInput(t *testing.T) {
	err := validatePartInput("Toyota", "Corolla", "Frenos", "Pastillas", "OEM", 2015, 5, 2, 10.0, "En stock")
	if err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
	err = validatePartInput("", "Corolla", "Frenos", "Pastillas", "OEM", 2015, 5, 2, 10.0, "En stock")
	if err == nil {
		t.Fatal("expected validation error for empty make")
	}
	err = validatePartInput("Toyota", "Corolla", "Frenos", "Pastillas", "OEM", 2015, -1, 2, 10.0, "En stock")
	if err == nil {
		t.Fatal("expected validation error for negative stock")
	}
	err = validatePartInput("Toyota", "Corolla", "Frenos", "Pastillas", "OEM", 2015, 5, 2, 0, "En stock")
	if err == nil {
		t.Fatal("expected validation error for zero price")
	}
}

func TestValidateSignup(t *testing.T) {
	if err := validateSignup("Taller", "Juan", "+18095550100", "Calle 1", "1234"); err != nil {
		t.Fatalf("valid signup rejected: %v", err)
	}
	if err := validateSignup("", "Juan", "+18095550100", "Calle 1", "1234"); err == nil {
		t.Fatal("expected error for empty name")
	}
	if err := validateSignup("Taller", "Juan", "+18095550100", "Calle 1", "12"); err == nil {
		t.Fatal("expected error for short password")
	}
}


func TestMapsQueryPrefersAddress(t *testing.T) {
	got := mapsQuery("Av. Winston Churchill 123", "Taller X")
	if got == "" || got == "https://maps.google.com/?q=Taller+X" {
		t.Fatalf("expected address in maps query, got %q", got)
	}
}
