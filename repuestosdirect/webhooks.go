package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
)

func handleWhatsAppWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		mode := r.URL.Query().Get("hub.mode")
		token := r.URL.Query().Get("hub.verify_token")
		challenge := r.URL.Query().Get("hub.challenge")
		verifyToken := os.Getenv("WHATSAPP_VERIFY_TOKEN")
		if mode == "subscribe" && token == verifyToken && verifyToken != "" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	body, _ := io.ReadAll(r.Body)
	if sig := r.Header.Get("X-Hub-Signature-256"); sig != "" {
		if !verifyWhatsAppSignature(body, sig) {
			http.Error(w, "invalid signature", http.StatusForbidden)
			return
		}
	}
	// Delivery/read receipts — log for now; extend for customer read tracking
	logInfo("whatsapp webhook", string(body))
	w.WriteHeader(http.StatusOK)
}

func verifyWhatsAppSignature(body []byte, header string) bool {
	secret := os.Getenv("WHATSAPP_APP_SECRET")
	if secret == "" {
		return true
	}
	const prefix = "sha256="
	if len(header) <= len(prefix) {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := prefix + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(header))
}

func handleLegalTerms(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, r, "page_legal", PageData{Title: "Términos y condiciones", CartCount: cartCount(s), Data: legalPageData{Slug: "terms"}})
}

func handleLegalPrivacy(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, r, "page_legal", PageData{Title: "Política de privacidad", CartCount: cartCount(s), Data: legalPageData{Slug: "privacy"}})
}

func handleLegalRefund(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	render(w, r, "page_legal", PageData{Title: "Devoluciones y reembolsos", CartCount: cartCount(s), Data: legalPageData{Slug: "refund"}})
}

type legalPageData struct {
	Slug string
}
