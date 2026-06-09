// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package auth defines stable authentication contract values shared by the server runtime.
package auth

// Scheme identifies a stable HTTP authentication scheme token.
type Scheme string

// String returns the wire-format authentication scheme.
func (s Scheme) String() string {
	return string(s)
}

// Prefix returns the canonical scheme prefix used in Authorization headers.
func (s Scheme) Prefix() string {
	return s.String() + " "
}

const (
	// Bearer identifies the HTTP bearer-token authorization scheme.
	Bearer Scheme = "Bearer"
)
