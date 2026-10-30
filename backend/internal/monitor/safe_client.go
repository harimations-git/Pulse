package monitor

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func ValidateTargetURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)

	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	//Allow HTTP and HTTPS only
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("Only HTTP and HTTPS are allowed")
	}

	hostname := strings.ToLower(parsed.Hostname())

	if hostname == "" {
		return fmt.Errorf("UTL must contain a hostname")
	}

	//Prevent URLs containing embedded credentials
	if parsed.User != nil {
		return fmt.Errorf("URL credentials are not allowed")
	}

	//Block local hosts
	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") {
		return fmt.Errorf("local hostnames are not allowed")
	}

	// Block explicitly supplied non-public IP addresses
	if ip := net.ParseIP(hostname); ip != nil {
		if !ip.IsGlobalUnicast() || ip.IsPrivate() {
			return fmt.Errorf("non-public IP addresses are not allowed")
		}
	}

	return nil
}
