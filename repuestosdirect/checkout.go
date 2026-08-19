package main

import (
	"fmt"
	"net/http"
	"strings"
)

func parseCheckoutItems(s *Session) ([]OrderItem, float64, error) {
	var items []OrderItem
	var total float64
	for pid, qty := range s.cartCopy() {
		p, ok := store.Part(pid)
		if !ok {
			continue
		}
		sub := p.PriceUSD * float64(qty)
		items = append(items, OrderItem{PartID: p.ID, PartName: p.Name, Qty: qty, UnitUSD: p.PriceUSD})
		total += sub
	}
	if len(items) == 0 {
		return nil, 0, fmt.Errorf("carrito vacío")
	}
	return items, total, nil
}

func parseCardFromRequest(r *http.Request) (CardChargeRequest, error) {
	exp, err := formatExpiration(r.FormValue("card_exp_month"), r.FormValue("card_exp_year"))
	if err != nil {
		return CardChargeRequest{}, err
	}
	req := CardChargeRequest{
		CardNumber:       r.FormValue("card_number"),
		ExpirationYYYYMM: exp,
		CVC:              strings.TrimSpace(r.FormValue("card_cvc")),
		CardholderName:   strings.TrimSpace(r.FormValue("cardholder_name")),
		CustomerPhone:    strings.TrimSpace(r.FormValue("customer_phone")),
	}
	if req.CardholderName == "" {
		return CardChargeRequest{}, fmt.Errorf("nombre del titular requerido")
	}
	if len(normalizeCardNumber(req.CardNumber)) < 13 {
		return CardChargeRequest{}, fmt.Errorf("número de tarjeta inválido")
	}
	if len(req.CVC) < 3 {
		return CardChargeRequest{}, fmt.Errorf("CVC inválido")
	}
	return req, nil
}

func processCardPayment(orderID string, total float64, card CardChargeRequest) (*CardChargeResult, error) {
	gw := NewPaymentGateway()
	if !gw.Enabled() && isProduction() {
		return nil, fmt.Errorf("pagos con tarjeta no configurados — contacte soporte")
	}
	card.OrderID = orderID
	card.AmountUSD = total
	result, err := gw.Charge(card)
	if err != nil {
		return result, err
	}
	return result, nil
}
