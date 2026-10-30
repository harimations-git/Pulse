package monitor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckServiceStatucCodes(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		wantHealthy bool
	}{
		{"OK", 200, true},
		{"Created", 201, true},
		{"Bad Request", 400, false},
		{"Not Found", 404, false},
		{"Server Error", 500, false},
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.statusCode)
				}),
			)

			defer server.Close()

			result := CheckService(context.Background(), server.URL, client)

			if result.Healthy != tc.wantHealthy {
				t.Errorf("Expected healthy %t, got %t", tc.wantHealthy, result.Healthy)
			}

			if result.StatusCode != tc.statusCode {
				t.Errorf("Expected status %d, got %d", tc.statusCode, result.StatusCode)
			}
		})
	}
}
func TestCheckServiceHealthy(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	result := CheckService(context.Background(), server.URL, client)

	if !result.Healthy {
		t.Error("Exepcted service to be healthy")
	}

	if result.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", result.StatusCode)
	}
}

func TestCheckServiceUnhealthy(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
	)
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	result := CheckService(context.Background(), server.URL, client)

	if result.Healthy {
		t.Error("Exepected service to be unhealthy")
	}

	if result.StatusCode != 503 {
		t.Errorf("Expected status to be 503, got %d", result.StatusCode)
	}

}

func TestServerTimeout(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	client := &http.Client{Timeout: 100 * time.Millisecond}

	result := CheckService(context.Background(), server.URL, client)

	if result.Healthy {
		t.Error("Exepected service to be unhealthy")
	}

	if result.StatusCode != 0 {
		t.Errorf("Expected status to be 0, got %d", result.StatusCode)
	}

	if result.Err == nil {
		t.Fatal("Expected a timeout error, got nil")
	}

	if !errors.Is(result.Err, context.DeadlineExceeded) {
		t.Errorf("Expected deadline exceeded error, got %v", result.Err)
	}

}

func TestCheckServiceCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	result := CheckService(ctx, "https://example.com", client)

	if result.Healthy {
		t.Error("Expected cancelled request to be unhealthy")
	}

	if result.StatusCode != 0 {
		t.Errorf("Expected status 0, got %d", result.StatusCode)
	}

	if !errors.Is(result.Err, context.Canceled) {
		t.Errorf(
			"Expected context.Canceled, got %v",
			result.Err,
		)
	}
}
