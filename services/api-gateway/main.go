package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics
var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "endpoint", "status"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "api_request_duration_seconds",
			Help: "API request duration in seconds",
		},
		[]string{"method", "endpoint"},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(requestDuration)
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

type QueryRequest struct {
	Prompt string `json:"prompt"`
	UserID string `json:"user_id,omitempty"`
}

type QueryResponse struct {
	Response  string    `json:"response"`
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

// Middleware to track metrics
func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapper := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start).Seconds()
		statusCode := fmt.Sprintf("%d", wrapper.statusCode)

		requestsTotal.WithLabelValues(r.Method, r.URL.Path, statusCode).Inc()
		requestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Health endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "OK",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Readiness endpoint
func readyHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Add checks for dependent services (Redis, MCP Server)
	response := HealthResponse{
		Status:    "READY",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Query endpoint (placeholder - will route to MCP Server)
func queryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	// TODO: Implement rate limiting with Redis
	// TODO: Route to MCP Server

	response := QueryResponse{
		Response:  "API Gateway is working! (MCP integration pending)",
		RequestID: generateRequestID(),
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Simple request ID generator
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

func main() {
	// Initialize rate limiter
	rateLimiter := NewRateLimiter()
	defer rateLimiter.Close()

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Health and metrics endpoints (no rate limiting)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readyHandler)
	mux.Handle("/metrics", promhttp.Handler())

	// API endpoints with rate limiting
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/v1/query", queryHandler)

	// Apply rate limiting to API endpoints only
	rateLimitedAPI := rateLimiter.Middleware(apiMux)
	mux.Handle("/api/", rateLimitedAPI)

	// Wrap with metrics middleware
	handler := metricsMiddleware(mux)

	// Setup server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
		// Security timeouts
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("API Gateway starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down API Gateway...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("API Gateway stopped")
}
