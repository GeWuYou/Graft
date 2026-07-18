package logger

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/apperror"
	"graft/server/internal/contract/errorcode"
	messagecontract "graft/server/internal/contract/message"
)

func TestReportErrorRecordsOnceAndPreservesCause(t *testing.T) {
	core, observed := observer.New(zapcore.ErrorLevel)
	appLogger := NewAppLogger(zap.New(core))
	cause := errors.New("runtime unavailable")
	err := apperror.Wrap(cause, apperror.Descriptor{
		Kind:       apperror.KindInternal,
		Code:       errorcode.CommonInternalError,
		MessageKey: messagecontract.CommonInternalError,
	})

	reported := ReportError(context.Background(), appLogger, "create redeploy workflow failed", err,
		StringField(FieldOperation, "application_redeploy"),
	)
	reported = ReportError(context.Background(), appLogger, "duplicate", reported)

	if !apperror.IsReported(reported) || !errors.Is(reported, cause) {
		t.Fatal("expected reported marker and preserved cause")
	}
	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected one error record, got %d", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields[FieldError] != cause.Error() || fields["error_kind"] != string(apperror.KindInternal) {
		t.Fatalf("expected canonical error fields, got %#v", fields)
	}
}

func TestReportErrorLeavesErrorUnreportedWithoutLogger(t *testing.T) {
	err := errors.New("unreported")
	if got := ReportError(context.Background(), nil, "ignored", err); got != err || apperror.IsReported(got) {
		t.Fatal("expected nil logger to leave error available for fallback reporting")
	}
}
