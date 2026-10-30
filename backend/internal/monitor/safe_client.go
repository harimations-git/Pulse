package monitor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
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

func isAllowedIP(ip netip.Addr) bool {
	ip = ip.Unmap()

	if !ip.IsValid() ||
		!ip.IsGlobalUnicast() ||
		ip.IsPrivate() ||
		ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() {
		return false
	}

	//Additional reserved and non-public address ranges
	blocked := []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("100.64.0.0/10"),
		netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("224.0.0.0/4"),
		netip.MustParsePrefix("240.0.0.0/4"),
		netip.MustParsePrefix("2001:db8::/32"),
	}

	for _, prefix := range blocked {
		if prefix.Contains(ip) {
			return false
		}
	}

	return true
}

func NewSafeHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	transport := &http.Transport{

		Proxy: nil,

		DialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {

			hostname, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}

			//Resolve DNS
			addresses, err := net.DefaultResolver.LookupIPAddr(ctx, hostname)
			if err != nil {
				return nil, err
			}

			if len(addresses) == 0 {
				return nil, fmt.Errorf("no IP addresses found")
			}

			//Fail closed if any resolved addfress is disallowed
			for _, resolved := range addresses {
				ip, ok := netip.AddrFromSlice(resolved.IP)

				if !ok || !isAllowedIP(ip) {
					return nil, fmt.Errorf("blocked destination: %s", resolved.IP)
				}
			}

			//Connect directly to a validated IP
			target := net.JoinHostPort(addresses[0].IP.String(), port)

			return dialer.DialContext(ctx, network, target)

		},

		TLSHandshakeTimeout: 5 * time.Second,
	}

	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,

		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
