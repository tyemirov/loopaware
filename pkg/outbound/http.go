package outbound

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

var nonPublicIPPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("255.255.255.255/32"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("::ffff:0:0/96"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:20::/28"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

// NewSafeHTTPClient copies an HTTP client and installs a public-network-only transport when needed.
func NewSafeHTTPClient(httpClient *http.Client, timeout time.Duration) *http.Client {
	if httpClient == nil {
		return &http.Client{
			Timeout:   timeout,
			Transport: NewSafeHTTPTransport(timeout),
		}
	}
	clientCopy := *httpClient
	if timeout > 0 && clientCopy.Timeout == 0 {
		clientCopy.Timeout = timeout
	}
	if clientCopy.Transport == nil {
		clientCopy.Transport = NewSafeHTTPTransport(timeout)
	}
	return &clientCopy
}

// NewSafeHTTPTransport returns a transport that rejects private, loopback, link-local, and special-use targets.
func NewSafeHTTPTransport(timeout time.Duration) http.RoundTripper {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	baseTransport := http.DefaultTransport.(*http.Transport).Clone()
	baseTransport.Proxy = nil
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
	}
	baseTransport.DialContext = func(ctx context.Context, network string, address string) (net.Conn, error) {
		host, port, splitErr := net.SplitHostPort(address)
		if splitErr != nil {
			return nil, splitErr
		}
		resolvedAddress, resolveErr := ResolvePublicDialAddress(ctx, host, port)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return dialer.DialContext(ctx, network, resolvedAddress)
	}
	return baseTransport
}

// ResolvePublicDialAddress resolves host to a dial address only when the target is public-routable.
func ResolvePublicDialAddress(ctx context.Context, host string, port string) (string, error) {
	normalizedHost := strings.TrimSpace(host)
	if normalizedHost == "" {
		return "", errors.New("empty outbound host")
	}
	if ip := net.ParseIP(normalizedHost); ip != nil {
		if !IsPublicIP(ip) {
			return "", fmt.Errorf("outbound target resolves to non-public address %s", ip.String())
		}
		return net.JoinHostPort(ip.String(), port), nil
	}

	addresses, lookupErr := net.DefaultResolver.LookupIPAddr(ctx, normalizedHost)
	if lookupErr != nil {
		return "", lookupErr
	}
	for _, address := range addresses {
		if IsPublicIP(address.IP) {
			return net.JoinHostPort(address.IP.String(), port), nil
		}
	}
	return "", fmt.Errorf("outbound target has no public DNS address: %s", normalizedHost)
}

// ValidatePublicHTTPURL rejects malformed, non-HTTP(S), or direct non-public outbound URLs.
func ValidatePublicHTTPURL(target *url.URL) error {
	if target == nil || target.Scheme == "" || target.Host == "" {
		return errors.New("invalid outbound url")
	}
	if !IsHTTPScheme(target.Scheme) {
		return errors.New("unsupported outbound url scheme")
	}
	if target.User != nil {
		return errors.New("outbound url userinfo is not allowed")
	}
	host := strings.TrimSpace(target.Hostname())
	if host == "" {
		return errors.New("empty outbound host")
	}
	if ip := net.ParseIP(host); ip != nil && !IsPublicIP(ip) {
		return fmt.Errorf("outbound target is non-public address %s", ip.String())
	}
	return nil
}

// IsHTTPScheme reports whether scheme is HTTP or HTTPS.
func IsHTTPScheme(scheme string) bool {
	return strings.EqualFold(scheme, "http") || strings.EqualFold(scheme, "https")
}

// IsPublicIP reports whether an IP address is globally routable and outside blocked special-use ranges.
func IsPublicIP(ip net.IP) bool {
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	address = address.Unmap()
	if !address.IsValid() || !address.IsGlobalUnicast() {
		return false
	}
	for _, prefix := range nonPublicIPPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}
