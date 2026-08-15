package main

import (
	"math"
	"testing"
	"time"
)

func TestBuildInstallmentScheduleTwoPayments30Days(t *testing.T) {
	placed := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	schedule := buildInstallmentSchedule(100.0, 30, 2, placed)
	if len(schedule) != 2 {
		t.Fatalf("expected 2 installments, got %d", len(schedule))
	}
	if schedule[0].Amount != 50.0 {
		t.Fatalf("first payment should be 50, got %v", schedule[0].Amount)
	}
	if schedule[1].Amount != 50.0 {
		t.Fatalf("second payment should be 50, got %v", schedule[1].Amount)
	}
	day15 := placed.AddDate(0, 0, 15)
	day30 := placed.AddDate(0, 0, 30)
	if !schedule[0].DueDate.Equal(day15) {
		t.Fatalf("first due expected day 15, got %v", schedule[0].DueDate)
	}
	if !schedule[1].DueDate.Equal(day30) {
		t.Fatalf("second due expected day 30, got %v", schedule[1].DueDate)
	}
}

func TestBuildInstallmentScheduleRounding(t *testing.T) {
	placed := time.Now()
	schedule := buildInstallmentSchedule(100.0, 30, 3, placed)
	var sum float64
	for _, s := range schedule {
		sum += s.Amount
	}
	if math.Abs(sum-100.0) > 0.01 {
		t.Fatalf("installments should sum to 100, got %v", sum)
	}
}

func TestValidateShopCredit(t *testing.T) {
	if err := validateShopCredit(500, 30, 2, 3); err != nil {
		t.Fatal(err)
	}
	if err := validateShopCredit(0, 30, 2, 3); err == nil {
		t.Fatal("expected error for zero limit")
	}
	if err := validateShopCredit(500, 0, 2, 3); err == nil {
		t.Fatal("expected error for zero terms")
	}
}
