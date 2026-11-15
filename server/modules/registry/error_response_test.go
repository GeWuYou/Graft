package registry

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/logger"
	"graft/server/internal/module"
)

func TestRegistryInternalErrorReportsFailureWithSafeResponse(t *testing.T) {
	core, logs := observer.New(zap.ErrorLevel)
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/registries/primary/repositories", nil)
	cause := errors.New("repository parameter binding failed")
	writeRegistryError(ctx, &module.Context{AppLogger: logger.NewAppLogger(zap.New(core))}, cause)
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "COMMON_INTERNALERROR") {
		t.Fatalf("internal error response = %d %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), cause.Error()) {
		t.Fatal("public response contains internal error details")
	}
	if logs.Len() != 1 || logs.All()[0].ContextMap()["error_fingerprint"] == "" {
		t.Fatalf("internal failure diagnostics = %#v", logs.All())
	}
	if logs.All()[0].ContextMap()["error"] == cause.Error() {
		t.Fatal("diagnostics contain raw internal error details")
	}
}
