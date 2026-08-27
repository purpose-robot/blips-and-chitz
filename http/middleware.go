package http

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	httpx "github.com/purpose-robot/blips-and-chitz/internal/http"
	slogx "github.com/purpose-robot/blips-and-chitz/internal/slog"
	"golang.org/x/time/rate"
)

func (s *Server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()

		rwx := httpx.NewResponseWriter(w)
		ctx := slogx.WithFields(r.Context())

		next.ServeHTTP(rwx, r.WithContext(ctx))

		attrs := append([]slog.Attr{
			slogx.Request(r, rwx.Status, rwx.BytesCount, now),
		}, slogx.Fields(ctx)...)

		level := slog.LevelInfo
		if rwx.Status >= 500 {
			level = slog.LevelError
		}

		s.logger.LogAttrs(ctx, level, "request handled", attrs...)
	})
}

func (s *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if pv := recover(); pv != nil {
				w.Header().Set("Connection", "close")
				httpx.Error(w, r, fmt.Errorf("recovered from panic: %v", pv))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (s *Server) rateLimit(next http.Handler) http.Handler {
	if !s.limiter.Enabled {
		return next
	}

	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for range time.Tick(time.Minute) {
			mu.Lock()

			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			httpx.Error(w, r, fmt.Errorf("failed to parse remote address %q: %v", r.RemoteAddr, err))
			return
		}

		mu.Lock()

		c, found := clients[ip]
		if !found {
			c = &client{limiter: rate.NewLimiter(rate.Limit(s.limiter.RPS), s.limiter.Burst)}
			clients[ip] = c
		}

		allowed := c.limiter.Allow()
		c.lastSeen = time.Now()

		mu.Unlock()

		if !allowed {
			httpx.Error(w, r, httpx.NewSafeError(nil, http.StatusTooManyRequests, "rate limit exceeded"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
