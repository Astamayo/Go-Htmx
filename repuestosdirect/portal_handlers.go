package main

import (
	"net/http"
	"os"
)

func handleTiendaPedidos(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	sh := currentShop(s)
	render(w, r, "page_tienda_pedidos", PageData{
		Title: "Pedidos", Shop: sh, CartCount: cartCount(s),
		Data: tiendaOrdersData{Orders: store.ActiveOrdersForShop(sh.ID), Completed: false},
	})
}

func handleTiendaHistorial(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	sh := currentShop(s)
	render(w, r, "page_tienda_pedidos", PageData{
		Title: "Historial", Shop: sh, CartCount: cartCount(s),
		Data: tiendaOrdersData{Orders: store.CompletedOrdersForShop(sh.ID), Completed: true},
	})
}

func handleTiendaOrderDetail(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	sh := currentShop(s)
	orderID := r.PathValue("id")
	order, err := store.Order(orderID)
	if err != nil || order.ShopID != sh.ID {
		http.NotFound(w, r)
		return
	}
	render(w, r, "page_tienda_order_detail", PageData{
		Title: "Pedido " + orderID, Shop: sh, CartCount: cartCount(s),
		Data: orderDetailData{
			Order: order, ShopName: sh.Name, ShopPhone: sh.Phone,
			Installments: order.Installments, History: store.OrderStatusHistory(orderID),
		},
	})
}

func handleTiendaEntregas(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	sh := currentShop(s)
	var pending []*Order
	for _, o := range store.ActiveOrdersForShop(sh.ID) {
		if o.Status == StatusListo || o.Status == StatusEnCamino {
			pending = append(pending, o)
		}
	}
	render(w, r, "page_tienda_entregas", PageData{
		Title: "Entregas", Shop: sh, CartCount: cartCount(s),
		Data: deliveryData{Pending: pending},
	})
}

func handleDriverStatus(w http.ResponseWriter, r *http.Request) {
	s := getSession(w, r)
	orderID := r.FormValue("order_id")
	newStatus := OrderStatus(r.FormValue("status"))
	order, err := store.Order(orderID)
	if err != nil {
		http.Redirect(w, r, "/driver", http.StatusSeeOther)
		return
	}
	if order.DriverID != "" && order.DriverID != s.driverID() {
		http.Error(w, "no autorizado", http.StatusForbidden)
		return
	}
	if newStatus == StatusEnCamino && order.DriverID == "" {
		store.AssignDriverToOrder(orderID, s.driverID())
	}
	if _, err := store.UpdateOrderStatus(orderID, newStatus, actorLabel(s)); err != nil {
		logError("driver status failed", err.Error())
	}
	if order.ShopID != "" {
		if sh, ok := store.Shop(order.ShopID); ok {
			notifyOrderStatus(sh.Phone, orderID, string(newStatus))
		}
	}
	http.Redirect(w, r, "/driver", http.StatusSeeOther)
}

type tiendaOrdersData struct {
	Orders    []*Order
	Completed bool
}

func notifyDriverReady(order *Order) {
	if driverPhone := os.Getenv("DRIVER_PHONE"); driverPhone != "" {
		SendWhatsApp(driverPhone, "📦 Nuevo pedido listo para recoger: "+order.ID)
	}
}
