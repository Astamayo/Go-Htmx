package main

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string { return e.Message }

func validatePartInput(makeStr, model, category, name, source string, year, stock, reorder int, price float64, availability string) error {
	if strings.TrimSpace(makeStr) == "" || strings.TrimSpace(model) == "" ||
		strings.TrimSpace(category) == "" || strings.TrimSpace(name) == "" ||
		strings.TrimSpace(source) == "" {
		return ValidationError{Message: "Completa todos los campos del repuesto."}
	}
	if year < 1980 || year > 2030 {
		return ValidationError{Message: "Año de vehículo inválido."}
	}
	if stock < 0 || reorder < 0 {
		return ValidationError{Message: "Stock y punto de reorden no pueden ser negativos."}
	}
	if price <= 0 {
		return ValidationError{Message: "El precio debe ser mayor a cero."}
	}
	if availability != "En stock" && availability != "Por pedido — 3 a 4 semanas" {
		return ValidationError{Message: "Disponibilidad inválida."}
	}
	return nil
}

func validateSignup(name, owner, phone, address, password string) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(owner) == "" ||
		strings.TrimSpace(phone) == "" || strings.TrimSpace(address) == "" ||
		password == "" {
		return ValidationError{Message: "Completa todos los campos."}
	}
	if len(password) < 4 {
		return ValidationError{Message: "La contraseña debe tener al menos 4 caracteres."}
	}
	return nil
}

func isInStock(availability string) bool {
	return availability == "En stock"
}

func availabilityLabel(availability string) string {
	if isInStock(availability) {
		return "En stock local — mismo día/siguiente"
	}
	return "Por pedido — 3 a 4 semanas"
}

func availabilityShort(availability string) string {
	if isInStock(availability) {
		return "en stock local"
	}
	return "por pedido, 3 a 4 semanas"
}

func validateShopCredit(limit float64, terms, splits, reminderDays int) error {
	if limit <= 0 {
		return ValidationError{Message: "El límite de crédito debe ser mayor a cero."}
	}
	if terms < 1 || terms > 365 {
		return ValidationError{Message: "Los términos deben ser entre 1 y 365 días."}
	}
	if splits < 1 || splits > 12 {
		return ValidationError{Message: "Los pagos deben ser entre 1 y 12 cuotas."}
	}
	if reminderDays < 0 || reminderDays > 30 {
		return ValidationError{Message: "El recordatorio debe ser entre 0 y 30 días antes."}
	}
	return nil
}

func creditTermsLabel(terms, splits int) string {
	if splits <= 1 {
		return fmt.Sprintf("%d días", terms)
	}
	daysPer := terms / splits
	return fmt.Sprintf("%d días en %d pagos (cada %d días)", terms, splits, daysPer)
}

func mapsQuery(address, name string) string {
	q := strings.TrimSpace(address)
	if q == "" {
		q = name
	}
	return fmt.Sprintf("https://maps.google.com/?q=%s", strings.ReplaceAll(q, " ", "+"))
}
