package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type AzulGateway struct {
	store       string
	auth1       string
	auth2       string
	environment string
	ecommerceURL string
	itbis       string
	client      *http.Client
}

func NewAzulGateway() *AzulGateway {
	env := strings.ToLower(os.Getenv("AZUL_ENVIRONMENT"))
	if env == "" {
		env = "sandbox"
	}
	return &AzulGateway{
		store:        os.Getenv("AZUL_MERCHANT_ID"),
		auth1:        firstNonEmpty(os.Getenv("AZUL_AUTH1"), os.Getenv("AZUL_API_KEY")),
		auth2:        os.Getenv("AZUL_AUTH2"),
		environment:  env,
		ecommerceURL: firstNonEmpty(os.Getenv("AZUL_ECOMMERCE_URL"), "https://repuestosdirect.com"),
		itbis:        firstNonEmpty(os.Getenv("AZUL_ITBIS"), "0"),
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

func (g *AzulGateway) Name() string { return "azul" }

func (g *AzulGateway) Enabled() bool {
	if g.environment == "mock" {
		return true
	}
	return g.store != "" && g.auth1 != "" && g.auth2 != ""
}

func (g *AzulGateway) apiURL() string {
	if g.environment == "production" {
		return "https://pagos.azul.com.do/WebServices/JSON/Default.aspx"
	}
	return "https://pruebas.azul.com.do/WebServices/JSON/Default.aspx"
}

func (g *AzulGateway) Charge(req CardChargeRequest) (*CardChargeResult, error) {
	card := normalizeCardNumber(req.CardNumber)
	if len(card) < 13 {
		return nil, fmt.Errorf("número de tarjeta inválido")
	}
	if g.environment == "mock" || (!g.Enabled() && isProduction()) {
		return g.mockCharge(card, req)
	}
	if !g.Enabled() {
		return g.mockCharge(card, req)
	}

	payload := map[string]any{
		"Channel":             "EC",
		"Store":               g.store,
		"CardNumber":          card,
		"Expiration":          req.ExpirationYYYYMM,
		"CVC":                 req.CVC,
		"PosInputMode":        "E-Commerce",
		"TrxType":             "Sale",
		"Amount":              amountToMinorUnits(req.AmountUSD),
		"Itbis":               g.itbis,
		"CurrencyPosCode":     firstNonEmpty(os.Getenv("AZUL_CURRENCY_CODE"), "US$"),
		"Payments":            "1",
		"Plan":                "0",
		"AcquirerRefData":     "1",
		"CustomerServicePhone": normalizePhoneDR(req.CustomerPhone),
		"OrderNumber":         req.OrderID,
		"ECommerceUrl":        g.ecommerceURL,
		"CustomOrderId":       req.OrderID,
		"ForceNo3DS":          "1",
	}

	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequest(http.MethodPost, g.apiURL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Auth1", g.auth1)
	httpReq.Header.Set("Auth2", g.auth2)

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error de conexión con Azul: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var azulResp struct {
		IsoCode           string `json:"IsoCode"`
		ResponseMessage   string `json:"ResponseMessage"`
		ResponseCode      string `json:"ResponseCode"`
		AuthorizationCode string `json:"AuthorizationCode"`
		RRN               string `json:"RRN"`
		ErrorDescription  string `json:"ErrorDescription"`
		Ticket            string `json:"Ticket"`
	}
	if err := json.Unmarshal(raw, &azulResp); err != nil {
		return nil, fmt.Errorf("respuesta Azul inválida")
	}

	result := &CardChargeResult{
		Success:           azulResp.IsoCode == "00",
		TransactionID:     azulResp.Ticket,
		AuthorizationCode: azulResp.AuthorizationCode,
		RRN:               azulResp.RRN,
		ResponseCode:      azulResp.ResponseCode,
		ResponseMessage:   firstNonEmpty(azulResp.ResponseMessage, azulResp.ErrorDescription),
		Gateway:           "azul",
		RawResponse:       string(raw),
	}
	if !result.Success {
		if result.ResponseMessage == "" {
			result.ResponseMessage = "Pago rechazado por el banco emisor"
		}
		return result, fmt.Errorf("%s", result.ResponseMessage)
	}
	return result, nil
}

func (g *AzulGateway) mockCharge(card string, req CardChargeRequest) (*CardChargeResult, error) {
	if strings.HasPrefix(card, "4000") {
		return &CardChargeResult{
			Success: false, Gateway: "azul-mock",
			ResponseMessage: "Tarjeta rechazada (prueba)",
		}, fmt.Errorf("tarjeta rechazada (sandbox)")
	}
	return &CardChargeResult{
		Success: true, Gateway: "azul-mock",
		TransactionID: "MOCK-" + req.OrderID,
		AuthorizationCode: "OK0000",
		RRN: "999999999999",
		ResponseCode: "00",
		ResponseMessage: "APROBADA (modo prueba)",
	}, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func normalizePhoneDR(phone string) string {
	p := strings.TrimSpace(phone)
	if p == "" {
		return "809-000-0000"
	}
	return p
}
