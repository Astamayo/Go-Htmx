package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type Session struct {
	mu        sync.RWMutex
	ID        string
	ShopID    string
	Cart      map[string]int
	CSRFToken string
}

func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func newCSRFToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func getSession(w http.ResponseWriter, r *http.Request) *Session {
	c, err := r.Cookie("sid")
	if err == nil && c.Value != "" {
		if s, ok := store.LoadSession(c.Value); ok {
			return s
		}
	}

	sid := newSessionID()
	s := &Session{
		ID:        sid,
		Cart:      map[string]int{},
		CSRFToken: newCSRFToken(),
	}
	store.SaveSession(s)

	http.SetCookie(w, &http.Cookie{
		Name:     "sid",
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400 * 7,
		SameSite: http.SameSiteLaxMode,
		Secure:   isProduction(),
	})
	return s
}

func (s *Session) persist() {
	store.SaveSession(s)
}

func (s *Session) setShopID(id string) {
	s.mu.Lock()
	s.ShopID = id
	s.mu.Unlock()
	s.persist()
}

func (s *Session) clearShopID() {
	s.setShopID("")
}

func (s *Session) setCart(cart map[string]int) {
	s.mu.Lock()
	s.Cart = cart
	s.mu.Unlock()
	s.persist()
}

func (s *Session) addToCart(partID string) {
	s.mu.Lock()
	s.Cart[partID]++
	s.mu.Unlock()
	s.persist()
}

func (s *Session) updateCart(partID string, qty int) {
	s.mu.Lock()
	if qty <= 0 {
		delete(s.Cart, partID)
	} else {
		s.Cart[partID] = qty
	}
	s.mu.Unlock()
	s.persist()
}

func (s *Session) clearCart() {
	s.setCart(map[string]int{})
}

func (s *Session) cartCopy() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]int, len(s.Cart))
	for k, v := range s.Cart {
		out[k] = v
	}
	return out
}

func (s *Session) shopID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ShopID
}

func (s *Session) csrfToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CSRFToken
}

func cartToJSON(cart map[string]int) []byte {
	b, _ := json.Marshal(cart)
	return b
}

func cartFromJSON(b []byte) map[string]int {
	cart := map[string]int{}
	if len(b) == 0 {
		return cart
	}
	_ = json.Unmarshal(b, &cart)
	return cart
}

func sessionExpiry() time.Time {
	return time.Now().Add(7 * 24 * time.Hour)
}
