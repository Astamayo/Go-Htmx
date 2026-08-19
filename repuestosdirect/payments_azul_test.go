package main

import (
	"os"
	"testing"
)

func TestAzulMockChargeApproved(t *testing.T) {
	os.Setenv("AZUL_ENVIRONMENT", "mock")
	gw := NewAzulGateway()
	result, err := gw.Charge(CardChargeRequest{
		OrderID: "ORD-TEST", AmountUSD: 100.00,
		CardNumber: "4111111111111111", ExpirationYYYYMM: "203012", CVC: "123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
}

func TestAzulMockChargeDeclined(t *testing.T) {
	os.Setenv("AZUL_ENVIRONMENT", "mock")
	gw := NewAzulGateway()
	_, err := gw.Charge(CardChargeRequest{
		OrderID: "ORD-TEST", AmountUSD: 10.00,
		CardNumber: "4000000000000002", ExpirationYYYYMM: "203012", CVC: "123",
	})
	if err == nil {
		t.Fatal("expected decline error")
	}
}

func TestFormatExpiration(t *testing.T) {
	exp, err := formatExpiration("3", "26")
	if err != nil || exp != "202603" {
		t.Fatalf("got %s err %v", exp, err)
	}
}

func TestAmountToMinorUnits(t *testing.T) {
	if amountToMinorUnits(50.00) != "5000" {
		t.Fatal("expected 5000 centavos")
	}
}
