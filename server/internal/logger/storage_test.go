// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"testing"
	"time"
)

func TestDefaultAppLogStoragePolicyMatchesCurrentAuthority(t *testing.T) {
	policy := DefaultAppLogStoragePolicy(3 * 24 * time.Hour)
	if policy.Mode != AppLogStorageModeRepositoryDurableStore {
		t.Fatalf("expected durable store mode, got %q", policy.Mode)
	}
	if policy.RetentionOwner != AppLogRetentionOwnerLogger {
		t.Fatalf("expected logger retention owner, got %q", policy.RetentionOwner)
	}
	if policy.DefaultWindow != 3*24*time.Hour {
		t.Fatalf("expected configured retention window, got %s", policy.DefaultWindow)
	}
	if err := policy.Validate(); err != nil {
		t.Fatalf("validate default policy: %v", err)
	}
}

func TestAppLogStoragePolicyRejectsRetentionWithoutDurableStore(t *testing.T) {
	policy := AppLogStoragePolicy{
		Mode:           AppLogStorageModeProcessOutput,
		RetentionOwner: AppLogRetentionOwnerLogger,
		DefaultWindow:  24 * time.Hour,
	}

	if err := policy.Validate(); err == nil {
		t.Fatal("expected process-output policy to reject repository retention")
	}
}

func TestAppLogRecordNormalizeRejectsForbiddenField(t *testing.T) {
	record := AppLogRecord{
		OccurredAt: time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC),
		Severity:   AppLogSeverityError,
		Component:  "modules.user.route",
		Message:    " map user response failed ",
		Fields: map[string]string{
			"status_code": "500",
		},
	}

	if _, err := record.Normalize(); err == nil {
		t.Fatal("expected forbidden field validation error")
	}
}

func TestAppLogRecordNormalizeSanitizesCanonicalFields(t *testing.T) {
	record := AppLogRecord{
		OccurredAt: time.Date(2026, 5, 30, 10, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		Severity:   AppLogSeverityWarn,
		Component:  " modules.user.route ",
		Message:    " map\tuser response \nfailed ",
		Operation:  " map user ",
		RequestID:  " req-1 ",
		TraceID:    " trace-1 ",
		Route:      " /api/users/:id ",
		Method:     " patch ",
		Error:      " bad \n request ",
		Fields: map[string]string{
			"module name":  " user ",
			"access_token": "secret",
		},
	}

	normalized, err := record.Normalize()
	if err != nil {
		t.Fatalf("normalize app log record: %v", err)
	}
	if normalized.Component != "modules.user.route" {
		t.Fatalf("expected sanitized component, got %q", normalized.Component)
	}
	if normalized.Message != "map user response failed" {
		t.Fatalf("expected sanitized message, got %q", normalized.Message)
	}
	if normalized.Operation != "map user" {
		t.Fatalf("expected sanitized operation, got %q", normalized.Operation)
	}
	if normalized.Method != "patch" {
		t.Fatalf("expected sanitized method, got %q", normalized.Method)
	}
	if got := normalized.Fields["module_name"]; got != "user" {
		t.Fatalf("expected sanitized module_name, got %q", got)
	}
	if got := normalized.Fields["access_token"]; got != redactedValue {
		t.Fatalf("expected redacted access_token, got %q", got)
	}
}

func TestIsForbiddenAppLogPersistedField(t *testing.T) {
	if !IsForbiddenAppLogPersistedField("client_ip") {
		t.Fatal("expected client_ip to be forbidden")
	}
	if IsForbiddenAppLogPersistedField("operation") {
		t.Fatal("expected operation to remain allowed")
	}
}
