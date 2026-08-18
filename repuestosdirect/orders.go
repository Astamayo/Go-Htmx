package main

import "strings"

func isOrderComplete(status OrderStatus) bool {
	return status == StatusLlegado || status == StatusLlegadoLegacy || status == StatusNoEntregado
}

var orderStatusOptions = []OrderStatus{
	StatusPedido,
	StatusConfirmado,
	StatusEnviado,
	StatusAduana,
	StatusEnCamino,
	StatusListo,
	StatusLlegado,
	StatusNoEntregado,
}

func statusBadgeClass(status OrderStatus) string {
	switch status {
	case StatusPedido:
		return "bg-stone-100 text-stone-700"
	case StatusConfirmado:
		return "bg-blue-100 text-blue-800"
	case StatusEnviado, StatusAduana:
		return "bg-indigo-100 text-indigo-800"
	case StatusEnCamino:
		return "bg-amber-100 text-amber-800"
	case StatusListo:
		return "bg-yellow-100 text-yellow-800"
	case StatusLlegado, StatusLlegadoLegacy:
		return "bg-emerald-100 text-emerald-800"
	case StatusNoEntregado:
		return "bg-red-100 text-red-800"
	default:
		return "bg-stone-100 text-stone-600"
	}
}

const activeOrderFilter = `status NOT IN ('Entregado', 'Llegado', 'No se pudo entregar')`

func (s *Store) ActiveOrdersAll() []*Order {
	rows, err := s.db.Query(`SELECT id, shop_id, total, on_credit, status, placed_at, due_date FROM orders WHERE ` + activeOrderFilter + ` ORDER BY placed_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanOrders(rows, s)
}

func (s *Store) CompletedOrdersAll() []*Order {
	rows, err := s.db.Query(`SELECT id, shop_id, total, on_credit, status, placed_at, due_date FROM orders WHERE NOT (` + activeOrderFilter + `) ORDER BY placed_at DESC LIMIT 100`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanOrders(rows, s)
}

func (s *Store) ActiveOrdersForShop(shopID string) []*Order {
	rows, err := s.db.Query(`SELECT id, shop_id, total, on_credit, status, placed_at, due_date FROM orders WHERE shop_id = $1 AND `+activeOrderFilter+` ORDER BY placed_at DESC`, shopID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanOrders(rows, s)
}

func (s *Store) CompletedOrdersForShop(shopID string) []*Order {
	rows, err := s.db.Query(`SELECT id, shop_id, total, on_credit, status, placed_at, due_date FROM orders WHERE shop_id = $1 AND NOT (`+activeOrderFilter+`) ORDER BY placed_at DESC LIMIT 50`, shopID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanOrders(rows, s)
}

func (s *Store) DriverAssignedOrders(driverID string) []*Order {
	rows, err := s.db.Query(`
		SELECT id, shop_id, total, on_credit, status, placed_at, due_date
		FROM orders
		WHERE driver_id = $1 AND `+activeOrderFilter+` AND status IN ('Listo para recoger', 'En camino')
		ORDER BY placed_at ASC`, driverID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanOrders(rows, s)
}

func (s *Store) OrdersSorted(sortBy string) []*Order {
	orderClause := "placed_at DESC"
	switch strings.ToLower(sortBy) {
	case "date":
		orderClause = "placed_at DESC"
	case "revenue":
		orderClause = "total DESC"
	case "shop":
		orderClause = "shop_id ASC, placed_at DESC"
	case "due":
		orderClause = "due_date ASC"
	}
	q := `SELECT id, shop_id, total, on_credit, status, placed_at, due_date FROM orders ORDER BY ` + orderClause
	rows, err := s.db.Query(q)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanOrders(rows, s)
}

func (s *Store) RevenueByMonth() map[string]float64 {
	rows, err := s.db.Query(`
		SELECT to_char(placed_at, 'YYYY-MM') AS month, COALESCE(SUM(total), 0)
		FROM orders
		GROUP BY month
		ORDER BY month ASC
		LIMIT 12`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var month string
		var total float64
		if rows.Scan(&month, &total) == nil {
			out[month] = total
		}
	}
	return out
}

func (s *Store) OrdersPerShop() map[string]int {
	rows, err := s.db.Query(`
		SELECT COALESCE(s.name, 'Sin taller'), COUNT(o.id)
		FROM orders o
		LEFT JOIN shops s ON s.id = o.shop_id
		GROUP BY s.name
		ORDER BY COUNT(o.id) DESC
		LIMIT 10`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var name string
		var n int
		if rows.Scan(&name, &n) == nil {
			out[name] = n
		}
	}
	return out
}

func (s *Store) TopSellingParts() map[string]int {
	rows, err := s.db.Query(`
		SELECT part_name, SUM(qty)::int
		FROM order_items
		GROUP BY part_name
		ORDER BY SUM(qty) DESC
		LIMIT 10`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var name string
		var n int
		if rows.Scan(&name, &n) == nil {
			out[name] = n
		}
	}
	return out
}
