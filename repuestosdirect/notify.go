package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// SendWhatsApp sends a message using the official Meta WhatsApp Cloud API.
func SendWhatsApp(toPhone, message string) {
	token := os.Getenv("WHATSAPP_TOKEN")
	phoneID := os.Getenv("WHATSAPP_PHONE_ID")

	// Fallback to console logging if API keys aren't set yet
	if token == "" || phoneID == "" {
		log.Printf("[WhatsApp MOCK -> %s] %s", toPhone, message)
		return
	}

	url := fmt.Sprintf("https://graph.facebook.com/v17.0/%s/messages", phoneID)

	payload := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                toPhone,
		"type":              "text",
		"text": map[string]string{
			"preview_url": "false",
			"body":        message,
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending WhatsApp message:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("WhatsApp API error: status code %d", resp.StatusCode)
	}
}
