// Package realtimeauth issues and validates short-lived realtime access tickets.
package realtimeauth

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

// ErrOriginDenied indicates that the websocket request origin is not allowlisted.
var ErrOriginDenied = errors.New("websocket origin denied")

// ValidateOrigin validates that the provided WebSocket Origin header matches an entry in the allowlist. It returns nil if a match is found, and ErrOriginDenied otherwise.
func ValidateOrigin(requestOrigin string, allowedOrigins []string) error {
	normalizedOrigin, ok := normalizeOrigin(requestOrigin)
	if !ok {
		return ErrOriginDenied
	}
	for _, allowed := range allowedOrigins {
		normalizedAllowed, allowedOK := normalizeOrigin(allowed)
		if allowedOK && strings.EqualFold(normalizedOrigin, normalizedAllowed) {
			return nil
		}
	}
	return ErrOriginDenied
}

func normalizeOrigin(raw string) (string, bool) {
	origin := strings.TrimSpace(raw)
	if origin == "" {
		return "", false
	}
	parsedOrigin, err := url.Parse(origin)
	if err != nil || !isSupportedOrigin(parsedOrigin) {
		return "", false
	}

	scheme := strings.ToLower(parsedOrigin.Scheme)
	host := strings.ToLower(parsedOrigin.Hostname())
	if host == "" {
		return "", false
	}

	return scheme + "://" + normalizedOriginHost(host, scheme, parsedOrigin.Port()), true
}

func isSupportedOrigin(parsedOrigin *url.URL) bool {
	if parsedOrigin.Scheme == "" || parsedOrigin.Host == "" {
		return false
	}
	return parsedOrigin.Path == "" && parsedOrigin.RawQuery == "" && parsedOrigin.Fragment == ""
}

func normalizedOriginHost(host, scheme, port string) string {
	if port == "" {
		port = defaultOriginPort(scheme)
	}
	if port == "" {
		return host
	}
	return net.JoinHostPort(host, port)
}

func defaultOriginPort(scheme string) string {
	switch scheme {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}
