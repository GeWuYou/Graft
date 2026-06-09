// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// Severity identifies a stable notification severity contract.
type Severity string

// String returns the canonical severity value.
func (s Severity) String() string {
	return string(s)
}

const (
	// SeverityInfo indicates informational notifications.
	SeverityInfo Severity = "info"
	// SeverityWarning indicates notifications that need attention.
	SeverityWarning Severity = "warning"
	// SeverityError indicates explicit failures.
	SeverityError Severity = "error"
	// SeverityCritical indicates high-risk or high-impact events.
	SeverityCritical Severity = "critical"
)

// ValidSeverity reports whether value is a known severity contract.
func ValidSeverity(value Severity) bool {
	switch value {
	case SeverityInfo, SeverityWarning, SeverityError, SeverityCritical:
		return true
	default:
		return false
	}
}
