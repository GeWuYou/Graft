// Package realtimeauth issues and validates short-lived realtime access tickets.
package realtimeauth

import (
	"errors"
	"net/url"
	"strings"
)

// ErrOriginDenied indicates that the websocket request origin is not allowlisted.
var ErrOriginDenied = errors.New("websocket origin denied")

// ValidateOrigin validates that the provided WebSocket Origin header matches an entry in the allowlist. It returns nil if a match is found, and ErrOriginDenied otherwise.
func ValidateOrigin(requestOrigin string, allowedOrigins []string) error {
	origin := strings.TrimSpace(requestOrigin)
	if origin == "" {
		return ErrOriginDenied
	}
	parsedOrigin, err := url.Parse(origin)
	if err != nil || parsedOrigin.Scheme == "" || parsedOrigin.Host == "" {
		return ErrOriginDenied
	}
	for _, allowed := range allowedOrigins {
		if strings.EqualFold(origin, strings.TrimSpace(allowed)) {
			return nil
		}
	}
	return ErrOriginDenied
}
