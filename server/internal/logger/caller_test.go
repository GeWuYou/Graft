package logger

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/logger/logsafe"
)

func TestLoggerFacadesAttributeCallerToExternalCallSite(t *testing.T) {
	testCases := []struct {
		name string
		log  func(*zap.Logger)
	}{
		{name: "app logger", log: func(base *zap.Logger) {
			NewAppLogger(base).Error(context.Background(), "failed")
		}},
		{name: "logsafe", log: func(base *zap.Logger) {
			logsafe.Error(base, "failed")
		}},
		{name: "category logger", log: func(base *zap.Logger) {
			Category(base, CategoryApplication).Error("failed")
		}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			core, observed := observer.New(zapcore.ErrorLevel)
			base := zap.New(core, zap.AddCaller())
			testCase.log(base)

			entries := observed.All()
			if len(entries) != 1 {
				t.Fatalf("expected one entry, got %d", len(entries))
			}
			caller := entries[0].Caller
			if !caller.Defined || !strings.HasSuffix(caller.File, "caller_test.go") {
				t.Fatalf("expected caller_test.go attribution, got %#v", caller)
			}
		})
	}
}
