package logger

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/apperror"
	"graft/server/internal/logger/logsafe"
)

func TestLoggerFacadesAttributeCallerToExternalCallSite(t *testing.T) {
	testCases := []struct {
		name string
		log  func(*testing.T, *zap.Logger)
	}{
		{name: "app logger", log: func(_ *testing.T, base *zap.Logger) {
			NewAppLogger(base).Error(context.Background(), "failed")
		}},
		{name: "logsafe", log: func(_ *testing.T, base *zap.Logger) {
			logsafe.Error(base, "failed")
		}},
		{name: "category logger", log: func(_ *testing.T, base *zap.Logger) {
			Category(base, CategoryApplication).Error("failed")
		}},
		{name: "report error", log: func(t *testing.T, base *zap.Logger) {
			reported := ReportError(context.Background(), NewAppLogger(base), "failed", errors.New("secret detail"))
			if !apperror.IsReported(reported) {
				t.Fatal("expected ReportError to return a reported error")
			}
		}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			core, observed := observer.New(zapcore.ErrorLevel)
			base := zap.New(core, zap.AddCaller())
			testCase.log(t, base)

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
