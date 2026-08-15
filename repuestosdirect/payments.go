package main

import (
	"database/sql"
	"fmt"
	"math"
	"time"
)

const (
	DefaultCreditLimit   = 300.00
	DefaultCreditTerms   = 30
	DefaultPaymentSplits = 2
	DefaultReminderDays  = 3
)

type Installment struct {
	ID             int64
	OrderID        string
	Num            int
	Amount         float64
	DueDate        time.Time
	PaidAt         *time.Time
	ReminderSentAt *time.Time
}

type AgingInstallmentRow struct {
	Order       *Order
	Installment Installment
	ShopName    string
	ShopPhone   string
	Overdue     bool
	DaysOverdue int
}

type installmentDraft struct {
	Num     int
	Amount  float64
	DueDate time.Time
}

// buildInstallmentSchedule splits total across terms days into equal payments.
// Example: 30 days, 2 splits → 50% due day 15, 50% due day 30.
func buildInstallmentSchedule(total float64, terms, splits int, placedAt time.Time) []installmentDraft {
	if splits < 1 {
		splits = 1
	}
	if terms < 1 {
		terms = DefaultCreditTerms
	}

	daysPer := float64(terms) / float64(splits)
	base := math.Floor(total/float64(splits)*100) / 100
	out := make([]installmentDraft, splits)
	allocated := 0.0

	for i := 1; i <= splits; i++ {
		amount := base
		if i == splits {
			amount = math.Round((total-allocated)*100) / 100
		}
		allocated += amount
		dueDay := int(math.Round(daysPer * float64(i)))
		out[i-1] = installmentDraft{
			Num:     i,
			Amount:  amount,
			DueDate: placedAt.AddDate(0, 0, dueDay),
		}
	}
	return out
}

func (s *Store) createInstallmentsTx(tx *sql.Tx, orderID string, total float64, terms, splits int, placedAt time.Time) (time.Time, error) {
	schedule := buildInstallmentSchedule(total, terms, splits, placedAt)
	var finalDue time.Time
	for _, inst := range schedule {
		_, err := tx.Exec(`
			INSERT INTO order_installments (order_id, installment_num, amount, due_date)
			VALUES ($1, $2, $3, $4)`,
			orderID, inst.Num, inst.Amount, inst.DueDate,
		)
		if err != nil {
			return time.Time{}, err
		}
		finalDue = inst.DueDate
	}
	return finalDue, nil
}

func (s *Store) InstallmentsForOrder(orderID string) []Installment {
	rows, err := s.db.Query(`
		SELECT id, order_id, installment_num, amount, due_date, paid_at, reminder_sent_at
		FROM order_installments WHERE order_id = $1 ORDER BY installment_num ASC`, orderID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanInstallments(rows)
}

func scanInstallments(rows *sql.Rows) []Installment {
	var out []Installment
	for rows.Next() {
		inst := Installment{}
		var paidAt, reminder sql.NullTime
		if err := rows.Scan(&inst.ID, &inst.OrderID, &inst.Num, &inst.Amount, &inst.DueDate, &paidAt, &reminder); err == nil {
			if paidAt.Valid {
				t := paidAt.Time
				inst.PaidAt = &t
			}
			if reminder.Valid {
				t := reminder.Time
				inst.ReminderSentAt = &t
			}
			out = append(out, inst)
		}
	}
	return out
}

func (s *Store) MarkInstallmentPaid(installmentID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var orderID, shopID sql.NullString
	var amount float64
	var paidAt sql.NullTime
	err = tx.QueryRow(`
		SELECT oi.order_id, oi.amount, oi.paid_at, o.shop_id
		FROM order_installments oi
		JOIN orders o ON o.id = oi.order_id
		WHERE oi.id = $1 FOR UPDATE OF oi, o`, installmentID,
	).Scan(&orderID, &amount, &paidAt, &shopID)
	if err != nil {
		return fmt.Errorf("cuota no encontrada")
	}
	if paidAt.Valid {
		return fmt.Errorf("esta cuota ya fue pagada")
	}
	if !shopID.Valid {
		return fmt.Errorf("pedido sin taller")
	}

	_, err = tx.Exec(`UPDATE order_installments SET paid_at = now() WHERE id = $1`, installmentID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE shops SET credit_used = GREATEST(credit_used - $1, 0) WHERE id = $2`, amount, shopID.String)
	if err != nil {
		return err
	}

	var unpaid int
	tx.QueryRow(`SELECT COUNT(*) FROM order_installments WHERE order_id = $1 AND paid_at IS NULL`, orderID.String).Scan(&unpaid)
	if unpaid == 0 {
		_, err = tx.Exec(`UPDATE orders SET paid_at = now() WHERE id = $1`, orderID.String)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) MarkOrderPaid(orderID string) error {
	insts := s.InstallmentsForOrder(orderID)
	if len(insts) > 0 {
		for _, inst := range insts {
			if inst.PaidAt == nil {
				if err := s.MarkInstallmentPaid(inst.ID); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return s.markOrderPaidLegacy(orderID)
}

func (s *Store) markOrderPaidLegacy(orderID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var shopID sql.NullString
	var total float64
	var onCredit bool
	var paidAt sql.NullTime
	err = tx.QueryRow(
		`SELECT shop_id, total, on_credit, paid_at FROM orders WHERE id = $1 FOR UPDATE`,
		orderID,
	).Scan(&shopID, &total, &onCredit, &paidAt)
	if err != nil {
		return fmt.Errorf("pedido no encontrado")
	}
	if !onCredit {
		return fmt.Errorf("solo pedidos a crédito pueden marcarse como pagados")
	}
	if paidAt.Valid {
		return fmt.Errorf("pedido ya fue pagado")
	}
	if !shopID.Valid {
		return fmt.Errorf("pedido sin taller asociado")
	}

	_, err = tx.Exec(`UPDATE orders SET paid_at = now() WHERE id = $1`, orderID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE shops SET credit_used = GREATEST(credit_used - $1, 0) WHERE id = $2`, total, shopID.String)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) UpdateShopCredit(shopID string, limit float64, terms, splits, reminderDays int) error {
	if limit <= 0 {
		return fmt.Errorf("el límite de crédito debe ser mayor a cero")
	}
	if terms < 1 || terms > 365 {
		return fmt.Errorf("términos inválidos (1-365 días)")
	}
	if splits < 1 || splits > 12 {
		return fmt.Errorf("número de pagos inválido (1-12)")
	}
	if reminderDays < 0 || reminderDays > 30 {
		return fmt.Errorf("días de recordatorio inválidos (0-30)")
	}

	sh, ok := s.Shop(shopID)
	if !ok {
		return fmt.Errorf("taller no encontrado")
	}
	if limit < sh.CreditUsed {
		return fmt.Errorf("no puedes bajar el límite por debajo del crédito usado ($%.2f)", sh.CreditUsed)
	}

	_, err := s.db.Exec(`
		UPDATE shops SET credit_limit = $1, credit_terms = $2, payment_splits = $3, reminder_days = $4
		WHERE id = $5 AND active = TRUE`,
		limit, terms, splits, reminderDays, shopID,
	)
	return err
}

func (s *Store) AgingInstallments() []AgingInstallmentRow {
	rows, err := s.db.Query(`
		SELECT oi.id, oi.order_id, oi.installment_num, oi.amount, oi.due_date, oi.paid_at, oi.reminder_sent_at,
		       o.id, o.shop_id, o.total, o.on_credit, o.status, o.placed_at, o.due_date,
		       sh.name, sh.phone
		FROM order_installments oi
		JOIN orders o ON o.id = oi.order_id
		JOIN shops sh ON sh.id = o.shop_id
		WHERE oi.paid_at IS NULL AND o.on_credit = TRUE AND sh.active = TRUE
		ORDER BY oi.due_date ASC`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	now := time.Now()
	var out []AgingInstallmentRow
	for rows.Next() {
		var inst Installment
		var order Order
		var shopName, shopPhone string
		var nullShop sql.NullString
		var paidAt, reminder sql.NullTime
		var orderDue time.Time

		err := rows.Scan(
			&inst.ID, &inst.OrderID, &inst.Num, &inst.Amount, &inst.DueDate, &paidAt, &reminder,
			&order.ID, &nullShop, &order.Total, &order.OnCredit, &order.Status, &order.PlacedAt, &orderDue,
			&shopName, &shopPhone,
		)
		if err != nil {
			continue
		}
		if paidAt.Valid {
			t := paidAt.Time
			inst.PaidAt = &t
		}
		if reminder.Valid {
			t := reminder.Time
			inst.ReminderSentAt = &t
		}
		if nullShop.Valid {
			order.ShopID = nullShop.String
		}
		order.DueDate = orderDue

		overdue := now.After(inst.DueDate)
		days := 0
		if overdue {
			days = int(now.Sub(inst.DueDate).Hours() / 24)
		}
		out = append(out, AgingInstallmentRow{
			Order: &order, Installment: inst,
			ShopName: shopName, ShopPhone: shopPhone,
			Overdue: overdue, DaysOverdue: days,
		})
	}
	return out
}

func (s *Store) SendPaymentReminders() (int, error) {
	rows, err := s.db.Query(`
		SELECT oi.id, oi.installment_num, oi.amount, oi.due_date,
		       o.id, sh.owner, sh.phone, sh.reminder_days
		FROM order_installments oi
		JOIN orders o ON o.id = oi.order_id
		JOIN shops sh ON sh.id = o.shop_id
		WHERE oi.paid_at IS NULL
		  AND o.on_credit = TRUE
		  AND sh.active = TRUE
		  AND (
		    (oi.due_date::date - CURRENT_DATE) <= sh.reminder_days
		    AND (oi.due_date::date - CURRENT_DATE) >= 0
		    AND (oi.reminder_sent_at IS NULL OR oi.reminder_sent_at::date < CURRENT_DATE)
		  )
		  OR (
		    oi.due_date < now()
		    AND (oi.reminder_sent_at IS NULL OR oi.reminder_sent_at::date < CURRENT_DATE)
		  )`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	sent := 0
	for rows.Next() {
		var instID int64
		var num int
		var amount float64
		var dueDate time.Time
		var orderID, owner, phone string
		var reminderDays int
		if err := rows.Scan(&instID, &num, &amount, &dueDate, &orderID, &owner, &phone, &reminderDays); err != nil {
			continue
		}

		daysLeft := int(time.Until(dueDate).Hours() / 24)
		var msg string
		if daysLeft < 0 {
			msg = fmt.Sprintf("Hola %s, la cuota %d de tu pedido %s por $%.2f está VENCIDA desde hace %d día(s). Por favor coordina el pago.",
				owner, num, orderID, amount, -daysLeft)
		} else if daysLeft == 0 {
			msg = fmt.Sprintf("Hola %s, hoy vence la cuota %d de tu pedido %s por $%.2f. Total pendiente de esta cuota.",
				owner, num, orderID, amount)
		} else {
			msg = fmt.Sprintf("Hola %s, recuerda que la cuota %d de tu pedido %s por $%.2f vence en %d día(s) (%s).",
				owner, num, orderID, amount, daysLeft, dueDate.Format("02 Jan 2006"))
		}
		SendWhatsApp(phone, msg)
		s.db.Exec(`UPDATE order_installments SET reminder_sent_at = now() WHERE id = $1`, instID)
		sent++
	}
	return sent, nil
}

func (s *Store) creditOutstanding() float64 {
	var total float64
	s.db.QueryRow(`
		SELECT COALESCE(SUM(oi.amount), 0)
		FROM order_installments oi
		JOIN orders o ON o.id = oi.order_id
		WHERE oi.paid_at IS NULL AND o.on_credit = TRUE`).Scan(&total)
	if total == 0 {
		s.db.QueryRow(`
			SELECT COALESCE(SUM(total), 0)
			FROM orders WHERE on_credit = TRUE AND paid_at IS NULL`).Scan(&total)
	}
	return total
}

func (s *Store) overdueInstallmentCount() int {
	var n int
	s.db.QueryRow(`
		SELECT COUNT(*)
		FROM order_installments oi
		JOIN orders o ON o.id = oi.order_id
		WHERE oi.paid_at IS NULL AND o.on_credit = TRUE AND oi.due_date < now()`).Scan(&n)
	if n == 0 {
		s.db.QueryRow(`
			SELECT COUNT(*)
			FROM orders WHERE on_credit = TRUE AND paid_at IS NULL AND due_date < now()`).Scan(&n)
	}
	return n
}
