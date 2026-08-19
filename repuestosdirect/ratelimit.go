package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type rateBucket struct {
	count   int
	resetAt time.Time
}

var rateLimitMu sync.Mutex
var rateLimitBuckets = map[string]*rateBucket{}

func rateLimit(max int, window time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			if !allowRate(ip, max, window) {
				http.Error(w, "demasiadas solicitudes — intente más tarde", http.StatusTooManyRequests)
				return
			}
			next(w, r)
		}
	}
}

func allowRate(key string, max int, window time.Duration) bool {
	now := time.Now()
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()
	b, ok := rateLimitBuckets[key]
	if !ok || now.After(b.resetAt) {
		rateLimitBuckets[key] = &rateBucket{count: 1, resetAt: now.Add(window)}
		return true
	}
	if b.count >= max {
		return false
	}
	b.count++
	return true
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
