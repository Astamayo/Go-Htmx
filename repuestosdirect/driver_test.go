package main

import "testing"

func TestDriverStatusNotifyLabel(t *testing.T) {
	if got := driverStatusNotifyLabel(StatusLlegado); got != "Entregado" {
		t.Fatalf("expected Entregado, got %q", got)
	}
	if got := driverStatusNotifyLabel(StatusEnCamino); got != "En camino" {
		t.Fatalf("expected En camino, got %q", got)
	}
}

func TestIsOrderComplete(t *testing.T) {
	for _, st := range []OrderStatus{StatusPedido, StatusEnCamino, StatusListo} {
		if isOrderComplete(st) {
			t.Fatalf("status %q should not be complete", st)
		}
	}
	for _, st := range []OrderStatus{StatusLlegado, StatusLlegadoLegacy, StatusNoEntregado} {
		if !isOrderComplete(st) {
			t.Fatalf("status %q should be complete", st)
		}
	}
}
