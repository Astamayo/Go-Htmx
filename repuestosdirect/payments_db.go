package main

import (
	"fmt"
	"os"
	"time"
)

func (s *Store) initPaymentTables() error {
	queries := []string{
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS payment_method VARCHAR(20) NOT NULL DEFAULT ''`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS payment_status VARCHAR(30) NOT NULL DEFAULT ''`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS payment_ref VARCHAR(100) NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS payment_transactions (
			id BIGSERIAL PRIMARY KEY,
			order_id VARCHAR(50) NOT NULL,
			gateway VARCHAR(30) NOT NULL DEFAULT '',
			amount NUMERIC(10,2) NOT NULL,
			currency VARCHAR(3) NOT NULL DEFAULT 'USD',
			status VARCHAR(30) NOT NULL,
			auth_code VARCHAR(50) NOT NULL DEFAULT '',
			rrn VARCHAR(50) NOT NULL DEFAULT '',
			response_code VARCHAR(30) NOT NULL DEFAULT '',
			response_message TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS ncf_receipts (
			id BIGSERIAL PRIMARY KEY,
			order_id VARCHAR(50) NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
			ncf_number VARCHAR(50) NOT NULL DEFAULT '',
			ncf_type VARCHAR(20) NOT NULL DEFAULT 'B02',
			issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			customer_rnc VARCHAR(20) NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS whatsapp_message_log (
			id BIGSERIAL PRIMARY KEY,
			phone VARCHAR(30) NOT NULL,
			template_name VARCHAR(100) NOT NULL DEFAULT '',
			status VARCHAR(30) NOT NULL,
			wa_message_id VARCHAR(100) NOT NULL DEFAULT '',
			error_text TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) RecordPaymentTransaction(orderID string, result *CardChargeResult, amount float64, status string) error {
	_, err := s.db.Exec(`
		INSERT INTO payment_transactions (order_id, gateway, amount, currency, status, auth_code, rrn, response_code, response_message)
		VALUES ($1, $2, $3, 'USD', $4, $5, $6, $7, $8)`,
		orderID, result.Gateway, amount, status,
		result.AuthorizationCode, result.RRN, result.ResponseCode, result.ResponseMessage,
	)
	return err
}

func (s *Store) SetOrderPayment(orderID, method, payStatus, ref string, paidAt *time.Time) error {
	_, err := s.db.Exec(`
		UPDATE orders SET payment_method=$2, payment_status=$3, payment_ref=$4, paid_at=COALESCE($5, paid_at)
		WHERE id=$1`, orderID, method, payStatus, ref, paidAt)
	return err
}

func (s *Store) LogWhatsAppSend(phone, template, status, msgID, errText string) {
	s.db.Exec(`INSERT INTO whatsapp_message_log (phone, template_name, status, wa_message_id, error_text) VALUES ($1,$2,$3,$4,$5)`,
		phone, template, status, msgID, errText)
}

func (s *Store) NextNCFNumber() (string, error) {
	var seq int
	err := s.db.QueryRow(`SELECT COUNT(*) + 1 FROM ncf_receipts`).Scan(&seq)
	if err != nil {
		return "", err
	}
	prefix := firstNonEmpty(os.Getenv("NCF_PREFIX"), "B02")
	return fmt.Sprintf("%s-%08d", prefix, seq), nil
}

func (s *Store) IssueNCFReceipt(orderID, customerRNC string) (string, error) {
	var exists bool
	s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM ncf_receipts WHERE order_id=$1)`, orderID).Scan(&exists)
	if exists {
		var ncf string
		s.db.QueryRow(`SELECT ncf_number FROM ncf_receipts WHERE order_id=$1`, orderID).Scan(&ncf)
		return ncf, nil
	}
	ncf, err := s.NextNCFNumber()
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(`INSERT INTO ncf_receipts (order_id, ncf_number, ncf_type, customer_rnc) VALUES ($1,$2,'B02',$3)`,
		orderID, ncf, customerRNC)
	return ncf, err
}

// ReserveOrderID generates the next order id without creating the order.
func (s *Store) ReserveOrderID() (string, error) {
	var seq int
	err := s.db.QueryRow("SELECT nextval('order_id_seq')").Scan(&seq)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ORD-%04d", seq), nil
}
