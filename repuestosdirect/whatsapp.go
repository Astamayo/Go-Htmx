package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const whatsappGraphVersion = "v21.0"

func whatsappConfigured() bool {
	return os.Getenv("WHATSAPP_TOKEN") != "" && os.Getenv("WHATSAPP_PHONE_ID") != ""
}

func whatsappUseTemplates() bool {
	return os.Getenv("WHATSAPP_USE_TEMPLATES") == "1" || os.Getenv("WHATSAPP_USE_TEMPLATES") == "true"
}

// SendWhatsApp queues a message asynchronously; never blocks order processing.
func SendWhatsApp(toPhone, message string) {
	go sendWhatsAppWithRetry(toPhone, message, "", nil)
}

func SendWhatsAppTemplate(toPhone, templateName string, bodyParams []string) {
	go sendWhatsAppWithRetry(toPhone, "", templateName, bodyParams)
}

func sendWhatsAppWithRetry(toPhone, text, templateName string, bodyParams []string) {
	to := normalizeWhatsAppPhone(toPhone)
	if to == "" {
		return
	}
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		msgID, err := sendWhatsAppOnce(to, text, templateName, bodyParams)
		if err == nil {
			store.LogWhatsAppSend(to, templateName, "sent", msgID, "")
			return
		}
		lastErr = err
		store.LogWhatsAppSend(to, templateName, "failed", "", err.Error())
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	log.Printf("[WhatsApp FAILED -> %s] %v", to, lastErr)
}

func sendWhatsAppOnce(toPhone, text, templateName string, bodyParams []string) (string, error) {
	token := os.Getenv("WHATSAPP_TOKEN")
	phoneID := os.Getenv("WHATSAPP_PHONE_ID")
	if token == "" || phoneID == "" {
		log.Printf("[WhatsApp MOCK -> %s] %s", toPhone, firstNonEmpty(text, templateName))
		return "mock", nil
	}

	var payload map[string]any
	if templateName != "" && whatsappUseTemplates() {
		components := []any{}
		if len(bodyParams) > 0 {
			params := make([]map[string]any, len(bodyParams))
			for i, p := range bodyParams {
				params[i] = map[string]any{"type": "text", "text": p}
			}
			components = append(components, map[string]any{
				"type": "body", "parameters": params,
			})
		}
		payload = map[string]any{
			"messaging_product": "whatsapp",
			"to":                toPhone,
			"type":              "template",
			"template": map[string]any{
				"name": templateName,
				"language": map[string]string{"code": firstNonEmpty(os.Getenv("WHATSAPP_TEMPLATE_LANG"), "es")},
				"components": components,
			},
		}
	} else {
		payload = map[string]any{
			"messaging_product": "whatsapp",
			"recipient_type":    "individual",
			"to":                toPhone,
			"type":              "text",
			"text": map[string]string{
				"preview_url": "false",
				"body":        text,
			},
		}
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", whatsappGraphVersion, phoneID)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("whatsapp api %d: %s", resp.StatusCode, string(raw))
	}
	var parsed struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	_ = json.Unmarshal(raw, &parsed)
	msgID := ""
	if len(parsed.Messages) > 0 {
		msgID = parsed.Messages[0].ID
	}
	return msgID, nil
}

func normalizeWhatsAppPhone(phone string) string {
	p := strings.TrimSpace(phone)
	p = strings.TrimPrefix(p, "+")
	p = strings.ReplaceAll(p, " ", "")
	p = strings.ReplaceAll(p, "-", "")
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "1") && len(p) == 11 {
		return p
	}
	if len(p) == 10 {
		return "1" + p
	}
	return p
}

func notifyOrderConfirmed(phone, owner, orderID string, total float64, onCredit bool) {
	if whatsappUseTemplates() {
		tpl := os.Getenv("WHATSAPP_TEMPLATE_ORDER_CONFIRM")
		if tpl != "" {
			SendWhatsAppTemplate(phone, tpl, []string{owner, orderID, fmt.Sprintf("%.2f", total)})
			return
		}
	}
	msg := fmt.Sprintf("Hola %s, tu pedido %s por $%.2f fue confirmado.", owner, orderID, total)
	if onCredit {
		msg += " Pago a crédito según tu plan."
	}
	SendWhatsApp(phone, msg)
}

func notifyOrderStatus(phone, orderID, status string) {
	if whatsappUseTemplates() {
		tpl := os.Getenv("WHATSAPP_TEMPLATE_ORDER_STATUS")
		if tpl != "" {
			SendWhatsAppTemplate(phone, tpl, []string{orderID, status})
			return
		}
	}
	SendWhatsApp(phone, "Actualización pedido "+orderID+": "+status)
}

func notifyPaymentReminder(phone, shopName, orderID string, amount float64, due string) {
	if whatsappUseTemplates() {
		tpl := os.Getenv("WHATSAPP_TEMPLATE_PAYMENT_REMINDER")
		if tpl != "" {
			SendWhatsAppTemplate(phone, tpl, []string{shopName, orderID, fmt.Sprintf("%.2f", amount), due})
			return
		}
	}
	SendWhatsApp(phone, fmt.Sprintf("Recordatorio %s: cuota $%.2f del pedido %s vence %s.", shopName, amount, orderID, due))
}
