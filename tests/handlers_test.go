package views

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "GET /health returns 200 with healthy status",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   "healthy",
		},
		{
			name:       "POST /health returns 405 Method Not Allowed",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE /health returns 405 Method Not Allowed",
			method:     http.MethodDelete,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := NewMetricsCollector()
			handlers := NewHandlers(metrics)

			req := httptest.NewRequest(tt.method, "/health", nil)
			rec := httptest.NewRecorder()

			handlers.HealthHandler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantBody != "" && rec.Body.String() != "" {
				var resp HealthResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Errorf("failed to decode response: %v", err)
				}
				if resp.Status != tt.wantBody {
					t.Errorf("status = %s, want %s", resp.Status, tt.wantBody)
				}
			}
		})
	}
}

func TestHealthHandlerContentType(t *testing.T) {
	metrics := NewMetricsCollector()
	handlers := NewHandlers(metrics)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handlers.HealthHandler(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", contentType)
	}
}

func TestHealthHandlerResponse(t *testing.T) {
	metrics := NewMetricsCollector()
	handlers := NewHandlers(metrics)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handlers.HealthHandler(rec, req)

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "healthy" {
		t.Errorf("status = %s, want healthy", resp.Status)
	}

	if resp.Message == "" {
		t.Error("message is empty")
	}

	if resp.Time.IsZero() {
		t.Error("time is zero")
	}
}

func TestMetricsHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "GET /metrics returns 200",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST /metrics returns 405 Method Not Allowed",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT /metrics returns 405 Method Not Allowed",
			method:     http.MethodPut,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := NewMetricsCollector()
			handlers := NewHandlers(metrics)

			req := httptest.NewRequest(tt.method, "/metrics", nil)
			rec := httptest.NewRecorder()

			handlers.MetricsHandler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestMetricsHandlerContentType(t *testing.T) {
	metrics := NewMetricsCollector()
	handlers := NewHandlers(metrics)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handlers.MetricsHandler(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", contentType)
	}
}

func TestMetricsHandlerResponse(t *testing.T) {
	metrics := NewMetricsCollector()
	handlers := NewHandlers(metrics)

	// Simulate some requests
	metrics.IncrementRequest()
	metrics.IncrementRequest()
	metrics.DecrementRequest()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handlers.MetricsHandler(rec, req)

	var resp Metrics
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.StartTime.IsZero() {
		t.Error("start_time is zero")
	}

	if resp.Uptime == "" {
		t.Error("uptime is empty")
	}

	if resp.Goroutines <= 0 {
		t.Errorf("goroutines = %d, want > 0", resp.Goroutines)
	}

	if resp.MemoryMB <= 0 {
		t.Errorf("memory_mb = %f, want > 0", resp.MemoryMB)
	}

	if resp.RequestsTotal != 2 {
		t.Errorf("requests_total = %d, want 2", resp.RequestsTotal)
	}

	if resp.RequestsActive != 1 {
		t.Errorf("requests_active = %d, want 1", resp.RequestsActive)
	}
}

func TestMetricsCollector(t *testing.T) {
	t.Run("IncrementRequest increments total and active", func(t *testing.T) {
		mc := NewMetricsCollector()

		mc.IncrementRequest()
		mc.IncrementRequest()

		metrics := mc.GetMetrics()
		if metrics.RequestsTotal != 2 {
			t.Errorf("requests_total = %d, want 2", metrics.RequestsTotal)
		}
		if metrics.RequestsActive != 2 {
			t.Errorf("requests_active = %d, want 2", metrics.RequestsActive)
		}
	})

	t.Run("DecrementRequest decrements active only", func(t *testing.T) {
		mc := NewMetricsCollector()

		mc.IncrementRequest()
		mc.IncrementRequest()
		mc.DecrementRequest()

		metrics := mc.GetMetrics()
		if metrics.RequestsTotal != 2 {
			t.Errorf("requests_total = %d, want 2", metrics.RequestsTotal)
		}
		if metrics.RequestsActive != 1 {
			t.Errorf("requests_active = %d, want 1", metrics.RequestsActive)
		}
	})

	t.Run("GetMetrics returns valid metrics", func(t *testing.T) {
		mc := NewMetricsCollector()
		time.Sleep(10 * time.Millisecond)

		metrics := mc.GetMetrics()

		if metrics.StartTime.IsZero() {
			t.Error("start_time is zero")
		}
		if metrics.Uptime == "" {
			t.Error("uptime is empty")
		}
		if metrics.Goroutines <= 0 {
			t.Errorf("goroutines = %d, want > 0", metrics.Goroutines)
		}
		if metrics.MemoryMB <= 0 {
			t.Errorf("memory_mb = %f, want > 0", metrics.MemoryMB)
		}
	})
}

func TestMetricsMiddleware(t *testing.T) {
	mc := NewMetricsCollector()
	handlers := NewHandlers(mc)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := handlers.MetricsMiddleware(testHandler)

	// Make multiple requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}

	metrics := mc.GetMetrics()
	if metrics.RequestsTotal != 3 {
		t.Errorf("requests_total = %d, want 3", metrics.RequestsTotal)
	}
	if metrics.RequestsActive != 0 {
		t.Errorf("requests_active = %d, want 0", metrics.RequestsActive)
	}
}

func TestMetricsMiddlewareConcurrency(t *testing.T) {
	mc := NewMetricsCollector()
	handlers := NewHandlers(mc)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := handlers.MetricsMiddleware(testHandler)

	// Simulate concurrent requests
	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rec, req)
			done <- struct{}{}
		}()
	}

	// Wait for requests to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	metrics := mc.GetMetrics()
	if metrics.RequestsTotal != 5 {
		t.Errorf("requests_total = %d, want 5", metrics.RequestsTotal)
	}
	if metrics.RequestsActive != 0 {
		t.Errorf("requests_active = %d, want 0", metrics.RequestsActive)
	}
}