package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestValidateTargetURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		{"HTTPS website", "https://example.com", false},
		{"HTTP website", "http://example.com", false},
		{"Localhost", "http://localhost:8080", true},
		{"Loopback IP", "http://127.0.0.1", true},
		{"Private IP", "http://192.168.1.1", true},
		{"Cloud metadata IP", "http://169.254.169.254", true},
		{"IPv6 loopback", "http://[::1]", true},
		{"FTP protocol", "ftp://example.com", true},
		{"Missing hostname", "https://", true},
		{"Embedded credentials", "https://user:pass@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTargetURL(tt.url)

			if tt.wantError && err == nil {
				t.Errorf("Expected URL %s to be rejected", tt.url)
			}

			if !tt.wantError && err != nil {
				t.Errorf("Expected URL %s to be accepted: %v", tt.url, err)
			}
		})
	}
}

func TestIsAllowedIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		allowed bool
	}{
		{"Public IPv4", "8.8.8.8", true},
		{"Another public IPv4", "1.1.1.1", true},
		{"Loopback", "127.0.0.1", false},
		{"Private IPv4", "192.168.1.1", false},
		{"Private 10 range", "10.0.0.1", false},
		{"Cloud metadata", "169.254.169.254", false},
		{"Shared address space", "100.64.0.1", false},
		{"Documentation range", "192.0.2.1", false},
		{"IPv6 loopback", "::1", false},
		{"IPv6 private", "fd00::1", false},
		{"IPv4 mapped loopback", "::ffff:127.0.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := netip.MustParseAddr(tt.ip)

			actual := isAllowedIP(ip)

			if actual != tt.allowed {
				t.Errorf(
					"IP %s: expected allowed=%t, got %t",
					tt.ip,
					tt.allowed,
					actual,
				)
			}
		})
	}
}

func TestSafeHTTPClientBlocksLoopback(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	client := NewSafeHTTPClient()

	result := CheckService(
		context.Background(),
		server.URL,
		client,
	)

	if result.Err == nil {
		t.Fatal("Expected loopback request to be blocked")
	}

	if result.Healthy {
		t.Error("Expected blocked request to be unhealthy")
	}

	if result.StatusCode != 0 {
		t.Errorf("Expected status 0, got %d", result.StatusCode)
	}
}
