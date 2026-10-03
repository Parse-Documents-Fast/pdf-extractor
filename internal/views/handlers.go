package views

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds application metrics
type Metrics struct {
	StartTime      time.Time `json:"start_time"`
	Uptime         string    `json:"uptime"`
	Goroutines     int       `json:"goroutines"`
	MemoryMB       float64   `json:"memory_mb"`
	RequestsTotal  int64     `json:"requests_total"`
	RequestsActive int64     `json:"requests_active"`
}

// MetricsCollector manages application metrics
type MetricsCollector struct {
	startTime      time.Time
	requestsTotal  atomic.Int64
	requestsActive atomic.Int64
	mu             sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now(),
	}
}

// IncrementRequest increments active request counter
func (mc *MetricsCollector) IncrementRequest() {
	mc.requestsActive.Add(1)
	mc.requestsTotal.Add(1)
}

// DecrementRequest decrements active request counter
func (mc *MetricsCollector) DecrementRequest() {
	mc.requestsActive.Add(-1)
}

// GetMetrics returns current metrics snapshot
func (mc *MetricsCollector) GetMetrics() Metrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return Metrics{
		StartTime:      mc.startTime,
		Uptime:         time.Since(mc.startTime).String(),
		Goroutines:     runtime.NumGoroutine(),
		MemoryMB:       float64(m.Alloc) / 1024 / 1024,
		RequestsTotal:  mc.requestsTotal.Load(),
		RequestsActive: mc.requestsActive.Load(),
	}
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status  string    `json:"status"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

// Handlers manages HTTP handlers
type Handlers struct {
	metrics *MetricsCollector
}

// NewHandlers creates a new handlers instance
func NewHandlers(metrics *MetricsCollector) *Handlers {
	return &Handlers{metrics: metrics}
}

// HealthHandler handles GET /health
func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:  "healthy",
		Message: "PDF Extractor API is running",
		Time:    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// MetricsHandler handles GET /metrics
func (h *Handlers) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := h.metrics.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

// MetricsMiddleware wraps an HTTP handler with metrics collection
func (h *Handlers) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.metrics.IncrementRequest()
		defer h.metrics.DecrementRequest()
		next.ServeHTTP(w, r)
	})
}