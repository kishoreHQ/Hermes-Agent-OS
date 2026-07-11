// Package ratelimit is a simple token-bucket rate limiter for Host API.
package ratelimit

import (
	"net/http"
	"sync"
	"time"
)

// Limiter tracks requests per key (IP).
type Limiter struct {
	mu       sync.Mutex
	perMin   int
	hits     map[string][]time.Time
}

func New(perMin int) *Limiter {
	if perMin <= 0 {
		perMin = 120
	}
	return &Limiter{perMin: perMin, hits: map[string][]time.Time{}}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cut := now.Add(-time.Minute)
	arr := l.hits[key]
	j := 0
	for _, t := range arr {
		if t.After(cut) {
			arr[j] = t
			j++
		}
	}
	arr = arr[:j]
	if len(arr) >= l.perMin {
		l.hits[key] = arr
		return false
	}
	l.hits[key] = append(arr, now)
	return true
}

// Middleware wraps HTTP handlers.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/api/v1/health" {
			next.ServeHTTP(w, r)
			return
		}
		key := r.RemoteAddr
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			key = xff
		}
		if !l.Allow(key) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(429)
			_, _ = w.Write([]byte(`{"data":null,"error":{"code":"rate_limited","message":"too many requests"}}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
