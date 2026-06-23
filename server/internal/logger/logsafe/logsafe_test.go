package logsafe

import (
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestErrorSanitizesControlCharacters(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	logger := zap.New(WrapCore(core))

	Error(
		logger,
		"write request audit log failed\r\nfor user",
		zap.String("path", "/api/users\r\nx-injected: yes"),
		zap.Error(errors.New("db timeout\r\ntrace=1")),
	)

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected one log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "write request audit log failed for user" {
		t.Fatalf("expected sanitized message, got %q", entry.Message)
	}

	fields := entry.ContextMap()
	if fields["path"] != "/api/users x-injected: yes" {
		t.Fatalf("expected sanitized path field, got %#v", fields["path"])
	}
	if fields["error"] != "db timeout trace=1" {
		t.Fatalf("expected sanitized error field, got %#v", fields["error"])
	}
}

func TestWrapCoreSanitizesWithFields(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	logger := zap.New(WrapCore(core)).With(zap.String("requestId", "req-1\r\nforged"))

	logger.Info("http access\tcompleted")

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected one log entry, got %d", len(entries))
	}

	fields := entries[0].ContextMap()
	if fields["requestId"] != "req-1 forged" {
		t.Fatalf("expected sanitized With field, got %#v", fields["requestId"])
	}
	if entries[0].Message != "http access completed" {
		t.Fatalf("expected sanitized info message, got %q", entries[0].Message)
	}
}
