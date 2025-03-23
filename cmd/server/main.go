package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"enclose/internal/hub"
	"enclose/internal/handler"
)

func main() {
	// Initialize game hub with capacity limits
	gameHub := hub.NewHub(100) // Allow max 100 concurrent games
	defer gameHub.Stop()

	// Create WebSocket handler with hub reference
	wsHandler := &handler.WebSocketHandler{
		Hub: gameHub,
	}


	// Configure HTTP server with timeouts
	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler:      loggingMiddleware(http.DefaultServeMux),
	}

	// Register WebSocket endpoint
	http.HandleFunc("/ws", rateLimitMiddleware(wsHandler.Handle))

	// Graceful shutdown setup
	shutdown := make(chan struct{})
	go handleSignals(server, gameHub, shutdown)

	// Start game maintenance goroutine
	go gameHub.MaintainGames(30*time.Second)

	// Start server
	log.Printf("Server starting on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}

	// Wait for shutdown completion
	<-shutdown
	log.Println("Server stopped gracefully")
}

func handleSignals(server *http.Server, hub *hub.Hub, shutdown chan struct{}) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	
	// Wait for shutdown signal
	<-sig
	log.Println("\nReceived shutdown signal")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop accepting new connections
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Stop game hub
	hub.Stop()

	// Close shutdown channel to signal completion
	close(shutdown)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	limiter := NewRateLimiter(100, time.Minute) // 100 requests/min per IP
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow(r.RemoteAddr) {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	}
}

//Bonus
type RateLimiter struct {
    requests int
    interval time.Duration
    mu       sync.Mutex
    counters map[string]int
}

func NewRateLimiter(requests int, interval time.Duration) *RateLimiter {
    rl := &RateLimiter{
        requests: requests,
        interval: interval,
        counters: make(map[string]int),
    }
    go rl.resetCounters()
    return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    if rl.counters[ip] >= rl.requests {
        return false
    }
    rl.counters[ip]++
    return true
}

func (rl *RateLimiter) resetCounters() {
    for range time.Tick(rl.interval) {
        rl.mu.Lock()
        rl.counters = make(map[string]int)
        rl.mu.Unlock()
    }
}