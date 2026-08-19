package main

import (
	"fmt"
	"os"
	"strings"
)

type CardChargeRequest struct {
	OrderID        string
	AmountUSD      float64
	CardNumber     string
	ExpirationYYYYMM string
	CVC            string
	CardholderName string
	CustomerPhone  string
}

type CardChargeResult struct {
	Success         bool
	TransactionID   string
	AuthorizationCode string
	RRN             string
	ResponseCode    string
	ResponseMessage string
	Gateway         string
	RawResponse     string
}

type PaymentGateway interface {
	Name() string
	Charge(req CardChargeRequest) (*CardChargeResult, error)
	Enabled() bool
}

func NewPaymentGateway() PaymentGateway {
	if os.Getenv("PAYMENT_GATEWAY") == "cardnet" {
		return NewCardNetGateway()
	}
	return NewAzulGateway()
}

func CardPaymentsEnabled() bool {
	return NewPaymentGateway().Enabled()
}

func normalizeCardNumber(n string) string {
	var b strings.Builder
	for _, c := range n {
		if c >= '0' && c <= '9' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

func formatExpiration(month, year string) (string, error) {
	m := strings.TrimSpace(month)
	y := strings.TrimSpace(year)
	if len(y) == 2 {
		y = "20" + y
	}
	if len(m) == 1 {
		m = "0" + m
	}
	if len(m) != 2 || len(y) != 4 {
		return "", fmt.Errorf("fecha de expiración inválida")
	}
	return y + m, nil
}

func amountToMinorUnits(amount float64) string {
	cents := int64(amount*100 + 0.5)
	return fmt.Sprintf("%d", cents)
}
