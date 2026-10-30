package monitor

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
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
		{"HTTP custom port", "http://example.com:8080", true},
		{"HTTPS database port", "https://example.com:5432", true},
		{"HTTP standard port", "http://example.com:80", false},
		{"HTTPS standard port", "https://example.com:443", false},
		{"IPv4 mapped loopback", "http://[::ffff:127.0.0.1]", true},
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
		{"IPv6 documentation", "2001:db8::1", false},
		{"IPv6 unspecified", "::", false},
		{"IPv6 multicast", "ff02::1", false},
		{"IPv4 unspecified", "0.0.0.0", false},
		{"IPv4 broadcast", "255.255.255.255", false},
		{"Benchmarking range", "198.19.0.1", false},
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

func TestSafeHTTPClientDoesNotFollowRedirects(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "http://127.0.0.1:8080")
			w.WriteHeader(http.StatusFound)
		}),
	)
	defer server.Close()

	client := NewSafeHTTPClient()

	// Use a test-only transport that allows local connections.
	// Production transport remains protected.
	client.Transport = &http.Transport{
		Proxy:       nil,
		DialContext: (&net.Dialer{}).DialContext,
	}

	result := CheckService(
		context.Background(),
		server.URL,
		client,
	)

	if result.StatusCode != http.StatusFound {
		t.Errorf("Expected 302, got %d", result.StatusCode)
	}

	if result.Healthy {
		t.Error("Expected redirect to be considered unhealthy")
	}

	if result.Err != nil {
		t.Errorf("Unexpected HTTP error: %v", result.Err)
	}
}

func TestSafeHTTPClientBlocksLocalhostDNS(t *testing.T) {
	client := NewSafeHTTPClient()

	result := CheckService(
		context.Background(),
		"http://localhost:8080",
		client,
	)

	if result.Err == nil {
		t.Fatal("Expected localhost connection to be blocked")
	}

	if result.Healthy {
		t.Error("Expected blocked destination to be unhealthy")
	}

	if result.StatusCode != 0 {
		t.Errorf("Expected status 0, got %d", result.StatusCode)
	}
}

func TestSafeHTTPClientRejectsInvalidURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"Localhost", "http://localhost:8080"},
		{"Private IP", "http://192.168.1.1"},
		{"Embedded credentials", "https://user:pass@example.com"},
		{"Unsupported scheme", "ftp://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewSafeHTTPClient()

			result := CheckService(
				context.Background(),
				tt.url,
				client,
			)

			if result.Err == nil {
				t.Fatal("Expected unsafe URL to be rejected")
			}

			if result.Healthy {
				t.Error("Expected unhealthy result")
			}

			if result.StatusCode != 0 {
				t.Errorf(
					"Expected status 0, got %d",
					result.StatusCode,
				)
			}
		})
	}
}

func TestSafeHTTPClientBlocksPrivateDNS(t *testing.T) {
	// Simulate DNS resolving a hostname to a private IP
	fakeLookup := func(
		ctx context.Context,
		hostname string,
	) ([]net.IPAddr, error) {
		return []net.IPAddr{
			{
				IP: net.ParseIP("192.168.1.10"),
			},
		}, nil
	}

	client := newSafeHTTPClientWithResolver(fakeLookup)

	result := CheckService(
		context.Background(),
		"https://monitor.example",
		client,
	)

	if result.Err == nil {
		t.Fatal("Expected private DNS destination to be rejected")
	}

	if result.Healthy {
		t.Error("Expected blocked request to be unhealthy")
	}

	if result.StatusCode != 0 {
		t.Errorf(
			"Expected status code 0, got %d",
			result.StatusCode,
		)
	}

	if !strings.Contains(result.Err.Error(), "blocked destination") {
		t.Errorf(
			"Expected blocked destination error, got %v",
			result.Err,
		)
	}
}

func TestSafeHTTPClientBlocksMixedDNS(t *testing.T) {
	fakeLookup := func(
		ctx context.Context,
		hostname string,
	) ([]net.IPAddr, error) {
		return []net.IPAddr{
			{IP: net.ParseIP("8.8.8.8")},
			{IP: net.ParseIP("10.0.0.5")},
		}, nil
	}

	client := newSafeHTTPClientWithResolver(fakeLookup)

	result := CheckService(
		context.Background(),
		"https://monitor.example",
		client,
	)

	if result.Err == nil {
		t.Fatal("Expected mixed DNS response to be rejected")
	}

	if !strings.Contains(result.Err.Error(), "blocked destination") {
		t.Errorf("Unexpected error: %v", result.Err)
	}
}
