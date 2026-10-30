package monitor

import "testing"

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
