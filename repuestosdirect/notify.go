package main

import "log"

// ---------------------------------------------------------------------
// WhatsApp notification stub.
//
// In production this calls the WhatsApp Business Cloud API (Meta) or a
// provider like Twilio. For the demo it just logs to the console so you
// can see exactly when and what would be sent. Swap the body of
// SendWhatsApp() for a real HTTP call when you have API credentials —
// everything that calls this function stays the same.
// ---------------------------------------------------------------------

func SendWhatsApp(toPhone, message string) {
	log.Printf("[WhatsApp -> %s] %s", toPhone, message)
}
