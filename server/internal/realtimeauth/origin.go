// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package realtimeauth

import (
	"errors"
	"net/url"
	"strings"
)

var ErrOriginDenied = errors.New("websocket origin denied")

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
