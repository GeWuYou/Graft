package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	"graft/server/internal/i18n"
)

// TestWriteLocalizedErrorUsesResolvedLocaleAndFallbackMessage 验证统一错误响应
// 会保留解析后的 locale，同时在缺少目标语言词条时回退到默认文案。
func TestWriteLocalizedErrorUsesResolvedLocaleAndFallbackMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := i18n.New(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.GET("/healthz", func(inner *gin.Context) {
		WriteLocalizedError(inner, service, http.StatusBadRequest, "common.invalid_argument", map[string]any{
			"field": "id",
		})
	})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	ctx.Request = request
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var payload ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "common.invalid_argument" {
		t.Fatalf("expected message key, got %#v", payload)
	}
	if payload.Locale != "en-US" {
		t.Fatalf("expected requested locale to be echoed, got %#v", payload)
	}
	if payload.Message != "请求参数不合法" || payload.Error != payload.Message {
		t.Fatalf("expected fallback localized message, got %#v", payload)
	}
	if payload.Details["field"] != "id" {
		t.Fatalf("expected details field id, got %#v", payload)
	}
}
