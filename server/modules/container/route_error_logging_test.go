package container

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/logger"
	"graft/server/internal/module"
)

func TestWriteRouteErrorReportsUnexpectedContainerFailureOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, observed := observer.New(zapcore.ErrorLevel)
	runtimeLogger := zap.New(core)
	route := routeRuntime{ctx: &module.Context{
		Logger:    runtimeLogger,
		AppLogger: logger.NewAppLogger(runtimeLogger),
	}}
	engine := gin.New()
	engine.GET("/containers/:id", func(ctx *gin.Context) {
		route.writeRouteError(ctx, errors.New("docker daemon unavailable"))
	})
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/containers/demo", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if len(observed.All()) != 1 {
		t.Fatalf("expected one owner error log without HTTP duplicate, got %#v", observed.All())
	}
	if observed.All()[0].Message != "container request failed" {
		t.Fatalf("expected semantic error message, got %#v", observed.All()[0])
	}
}
