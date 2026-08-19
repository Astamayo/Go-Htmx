package main

import (
	"fmt"
	"os"
)

// CardNetGateway placeholder — configure when CardNet credentials are available.
// Set PAYMENT_GATEWAY=cardnet and CARDNET_* env vars.
type CardNetGateway struct{}

func NewCardNetGateway() *CardNetGateway { return &CardNetGateway{} }

func (g *CardNetGateway) Name() string { return "cardnet" }

func (g *CardNetGateway) Enabled() bool {
	return os.Getenv("CARDNET_MERCHANT_ID") != "" && os.Getenv("CARDNET_API_KEY") != ""
}

func (g *CardNetGateway) Charge(req CardChargeRequest) (*CardChargeResult, error) {
	if !g.Enabled() {
		return NewAzulGateway().Charge(req)
	}
	return nil, fmt.Errorf("integración CardNet pendiente — contacte soporte")
}
