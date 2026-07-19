package audit

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/apperror"
	"graft/server/internal/logger"
	"graft/server/internal/module"
)

func TestReportAuditRouteErrorMarksSingleOwnerReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, observed := observer.New(zapcore.ErrorLevel)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/audit/logs", nil)
	moduleCtx := &module.Context{AppLogger: logger.NewAppLogger(zap.New(core))}

	reported := reportAuditRouteError(ctx, moduleCtx, "list audit logs failed", errors.New("audit repository unavailable"), logger.StringField("module", moduleID))
	if !apperror.IsReported(reported) {
		t.Fatal("expected semantic audit report to suppress HTTP fallback duplication")
	}
	if len(observed.All()) != 1 || observed.All()[0].Message != "list audit logs failed" {
		t.Fatalf("expected one semantic audit error log, got %#v", observed.All())
	}
}
