package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/apperror"
	"graft/server/internal/config"
	"graft/server/internal/contract/errorcode"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/i18n"
)

func TestRequestIDMiddlewareAttachesCorrelationBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestIDMiddleware())
	engine.GET("/items/:id", func(ctx *gin.Context) {
		correlation, ok := RequestAuditContextFromContext(ctx.Request.Context())
		if !ok {
			t.Fatal("expected correlation context before handler execution")
		}
		if correlation.RequestID != "req-1" || correlation.TraceID != "trace-1" {
			t.Fatalf("expected incoming correlation ids, got %#v", correlation)
		}
		if correlation.Route != "/items/:id" || correlation.Method != http.MethodGet {
			t.Fatalf("expected route and method, got %#v", correlation)
		}
		ctx.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/items/7", nil)
	request.Header.Set(RequestIDHeader, "req-1")
	request.Header.Set(traceIDFallbackHeader, "trace-1")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestWriteAppErrorLogsOnlyUnreportedInternalCause(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantLogs   int
		wantStatus int
	}{
		{
			name: "expected not found",
			err: apperror.New(apperror.Descriptor{
				Kind: apperror.KindNotFound, Code: errorcode.CommonNotFound, MessageKey: messagecontract.CommonNotFound,
			}),
			wantStatus: http.StatusNotFound,
		},
		{name: "unknown internal", err: errors.New("sql: connection refused"), wantLogs: 1, wantStatus: http.StatusInternalServerError},
		{name: "reported internal", err: apperror.MarkReported(errors.New("docker unavailable")), wantStatus: http.StatusInternalServerError},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			core, observed := observer.New(zapcore.ErrorLevel)
			engine := gin.New()
			engine.Use(RequestIDMiddleware())
			engine.GET("/failure", func(ctx *gin.Context) {
				WriteAppError(ctx, nil, zap.New(core), testCase.err)
			})
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/failure", nil))

			if recorder.Code != testCase.wantStatus {
				t.Fatalf("expected status %d, got %d", testCase.wantStatus, recorder.Code)
			}
			if len(observed.All()) != testCase.wantLogs {
				t.Fatalf("expected %d fallback logs, got %d", testCase.wantLogs, len(observed.All()))
			}
			if testCase.wantStatus == http.StatusInternalServerError && strings.Contains(recorder.Body.String(), testCase.err.Error()) {
				t.Fatalf("expected internal cause to stay out of response: %s", recorder.Body.String())
			}
		})
	}
}

func TestWriteLocalizedErrorUsesTraceIDWhenRequestAndTraceDiffer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.GET("/failure", func(inner *gin.Context) {
		WriteLocalizedError(inner, nil, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
	})
	request := httptest.NewRequest(http.MethodGet, "/failure", nil)
	request.Header.Set(RequestIDHeader, "request-1")
	request.Header.Set(traceIDFallbackHeader, "trace-1")
	ctx.Request = request
	engine.HandleContext(ctx)

	var payload ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.TraceID != "trace-1" {
		t.Fatalf("expected trace id in error response, got %#v", payload)
	}
	if recorder.Header().Get(RequestIDHeader) != "request-1" {
		t.Fatalf("expected request id response header, got %q", recorder.Header().Get(RequestIDHeader))
	}
}

func TestWriteAppErrorRejectsUnsupportedDescriptorKind(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	err := apperror.New(apperror.Descriptor{
		Kind:       apperror.Kind("unexpected"),
		Code:       errorcode.Code("PRIVATE_CODE"),
		MessageKey: messagecontract.Key("private.detail"),
		PublicData: map[string]any{"secret": "must not leak"},
	})
	engine.GET("/failure", func(inner *gin.Context) {
		WriteAppError(inner, nil, zap.NewNop(), err)
	})
	ctx.Request = httptest.NewRequest(http.MethodGet, "/failure", nil)
	engine.HandleContext(ctx)

	var payload ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != errorcode.CommonInternalError.String() || payload.MessageKey != messagecontract.CommonInternalError.String() {
		t.Fatalf("expected safe internal descriptor, got %#v", payload)
	}
	if payload.Data != nil || strings.Contains(recorder.Body.String(), "must not leak") {
		t.Fatalf("expected unsupported descriptor data to stay out of response: %s", recorder.Body.String())
	}
}

func TestWriteSavedViewErrorLogsUnexpectedCause(t *testing.T) {
	core, entries := observer.New(zapcore.ErrorLevel)
	restoreGlobals := zap.ReplaceGlobals(zap.New(core))
	defer restoreGlobals()

	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/api/access-log/saved-views", nil)
	ginCtx.Request.Header.Set(RequestIDHeader, "request-saved-view")
	unexpected := errors.New("column saved_views.is_default does not exist")

	WriteSavedViewError(ginCtx, nil, unexpected)

	if got := entries.Len(); got != 1 {
		t.Fatalf("expected one internal-error log entry, got %d", got)
	}
	entry := entries.All()[0]
	if entry.Message != "unreported internal error" {
		t.Fatalf("unexpected log message %q", entry.Message)
	}
	if got := entry.ContextMap()["error"]; got != unexpected.Error() {
		t.Fatalf("logged cause = %#v, want %q", got, unexpected.Error())
	}
}

func assertLocalizedErrorEnvelope(t *testing.T, payload ErrorResponse) {
	t.Helper()

	if payload.MessageKey != "common.invalid_argument" {
		t.Fatalf("expected message key, got %#v", payload)
	}
	if payload.Code != "COMMON_INVALID_ARGUMENT" || payload.Success {
		t.Fatalf("expected stable error envelope code/success, got %#v", payload)
	}
	if payload.Locale != "en-US" {
		t.Fatalf("expected requested locale to be echoed, got %#v", payload)
	}
	if payload.Message != "Invalid request parameters" || payload.Error != payload.Message {
		t.Fatalf("expected en-US localized message, got %#v", payload)
	}
}

func assertLocalizedErrorDetails(t *testing.T, recorder *httptest.ResponseRecorder, payload ErrorResponse) {
	t.Helper()

	if payload.Details["field"] != "id" {
		t.Fatalf("expected details field id, got %#v", payload)
	}
	if payload.TraceID == "" {
		t.Fatalf("expected trace id to be generated, got %#v", payload)
	}
	if recorder.Header().Get(RequestIDHeader) != payload.TraceID {
		t.Fatalf("expected trace id header to match payload, got %#v", payload)
	}
}

// TestWriteLocalizedErrorUsesResolvedLocaleAndFallbackMessage 验证统一错误响应
// 会保留解析后的 locale，并优先返回对应语言的稳定文案。
func TestWriteLocalizedErrorUsesResolvedLocaleAndFallbackMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := i18n.MustNew(config.I18nConfig{
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
	assertLocalizedErrorEnvelope(t, payload)
	assertLocalizedErrorDetails(t, recorder, payload)
}

// TestWriteSuccessReusesIncomingRequestID 验证成功响应会复用上游透传的 request-id，
// 避免 auth/bootstrap 等链路在前后端排障时丢失统一 trace。
func TestWriteSuccessReusesIncomingRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.GET("/healthz", func(inner *gin.Context) {
		WriteSuccess(inner, http.StatusOK, map[string]any{
			"ok": true,
		})
	})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set(RequestIDHeader, "req-from-upstream")
	ctx.Request = request
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload SuccessResponse[map[string]any]
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Success || payload.Code != "OK" || payload.Message != "OK" {
		t.Fatalf("expected stable success envelope, got %#v", payload)
	}
	if payload.TraceID != "req-from-upstream" || recorder.Header().Get(RequestIDHeader) != payload.TraceID {
		t.Fatalf("expected request id to be reused, got %#v", payload)
	}
	if ok, exists := payload.Data["ok"]; !exists || ok != true {
		t.Fatalf("expected success payload data, got %#v", payload.Data)
	}
}
