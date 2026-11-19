package systemconfig

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	configregistry "graft/server/internal/configregistry"
	containerdi "graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	systemconfigcontract "graft/server/modules/system-config/contract"
	systemconfiglocales "graft/server/modules/system-config/locales"
)

// 本文件通过 gin 测试引擎验证 strict server 迁移后的路由行为与 envelope 结构：
// 生成 wrapper 承担参数与请求体绑定，权限中间件与错误映射保持迁移前语义。

type routeContractAuthService struct{}

func (routeContractAuthService) CurrentUser(ctx context.Context) (*moduleapi.CurrentUser, error) {
	auth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok {
		return nil, moduleapi.ErrUnauthenticated
	}
	if auth.User != nil {
		return &moduleapi.CurrentUser{ID: auth.User.ID, Username: "route-contract"}, nil
	}
	if auth.Claims == nil {
		return nil, moduleapi.ErrUnauthenticated
	}
	return &moduleapi.CurrentUser{ID: 42, Username: "route-contract"}, nil
}

func (routeContractAuthService) ParseAccessToken(_ context.Context, token string) (*moduleapi.AccessTokenClaims, error) {
	if token != "route-contract-token" {
		return nil, moduleapi.ErrInvalidAccessToken
	}
	return &moduleapi.AccessTokenClaims{}, nil
}

type routeContractAuthorizer struct{ denyWrites bool }

func (a routeContractAuthorizer) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, permissionCode string) error {
	if a.denyWrites && permissionCode == systemconfigcontract.SystemConfigWritePermission.String() {
		return moduleapi.ErrPermissionDenied
	}
	return nil
}

func newRouteContractContext(t *testing.T, engine *gin.Engine, authorizer routeContractAuthorizer) *module.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	localizer, err := i18n.New(config.I18nConfig{
		DefaultLocale:  "zh-CN",
		FallbackLocale: "zh-CN",
		SupportedLocales: []string{
			"zh-CN",
			"en-US",
		},
	})
	if err != nil {
		t.Fatalf("new i18n service: %v", err)
	}
	resources, err := systemconfiglocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("load system-config locale resources: %v", err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register system-config locale resources: %v", err)
	}
	services := containerdi.New()
	registerSingleton := func(target any, value any) {
		t.Helper()
		if err := services.RegisterSingleton(target, func(containerdi.Resolver) (any, error) {
			return value, nil
		}); err != nil {
			t.Fatalf("register route contract dependency: %v", err)
		}
	}
	registerSingleton((*moduleapi.AuthService)(nil), moduleapi.AuthService(routeContractAuthService{}))
	registerSingleton((*moduleapi.Authorizer)(nil), moduleapi.Authorizer(authorizer))
	return &module.Context{
		Config:             &config.Config{},
		Router:             engine.Group("/api"),
		I18n:               localizer,
		Services:           services,
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
	}
}

func newRouteContractEngine(t *testing.T, denyWrites bool) *gin.Engine {
	t.Helper()
	engine := gin.New()
	ctx := newRouteContractContext(t, engine, routeContractAuthorizer{denyWrites: denyWrites})
	service := newTestService(t, configregistry.Definition{
		Key:          "scheduler.timeout",
		Module:       "scheduler",
		Group:        "runtime",
		Title:        "Timeout",
		Type:         configregistry.ValueTypeString,
		DefaultValue: json.RawMessage(`"30s"`),
	})
	if err := registerSystemConfigRoutes(ctx, "system-config", service); err != nil {
		t.Fatalf("register system config routes: %v", err)
	}
	return engine
}

func doRouteContractRequest(method, target, token, body string) (*http.Request, *httptest.ResponseRecorder) {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	req.Header.Set("X-Request-Id", "route-contract-request-id")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, httptest.NewRecorder()
}

type routeContractEnvelope struct {
	Success bool           `json:"success"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	TraceID string         `json:"traceId"`
	Data    map[string]any `json:"data"`
}

func assertRouteContractEnvelope(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string, wantMessageKey string) routeContractEnvelope {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d, body %s", rec.Code, wantStatus, rec.Body.String())
	}
	var envelope routeContractEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope %s: %v", rec.Body.String(), err)
	}
	if envelope.Code != wantCode {
		t.Fatalf("code = %q, want %q", envelope.Code, wantCode)
	}
	if envelope.TraceID != "route-contract-request-id" {
		t.Fatalf("traceId = %q, want echoed request id", envelope.TraceID)
	}
	if wantMessageKey != "" && !strings.Contains(rec.Body.String(), `"messageKey":"`+wantMessageKey+`"`) {
		t.Fatalf("messageKey missing for %q in %s", wantMessageKey, rec.Body.String())
	}
	return envelope
}

func TestSystemConfigStrictRoutesKeepEnvelopeContract(t *testing.T) {
	engine := newRouteContractEngine(t, false)

	req, rec := doRouteContractRequest(http.MethodGet, "/api/system-configs", "route-contract-token", "")
	engine.ServeHTTP(rec, req)
	envelope := assertRouteContractEnvelope(t, rec, http.StatusOK, "OK", "")
	items, ok := envelope.Data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one config item, got %#v", envelope.Data)
	}
	if envelope.Data["total"] != float64(1) {
		t.Fatalf("expected total 1, got %#v", envelope.Data["total"])
	}

	req, rec = doRouteContractRequest(http.MethodGet, "/api/system-configs/nope.key", "route-contract-token", "")
	engine.ServeHTTP(rec, req)
	assertRouteContractEnvelope(t, rec, http.StatusNotFound, "SYSTEM_CONFIG_NOT_FOUND", systemconfigcontract.SystemConfigNotFound.String())

	req, rec = doRouteContractRequest(http.MethodGet, "/api/system-configs", "", "")
	engine.ServeHTTP(rec, req)
	assertRouteContractEnvelope(t, rec, http.StatusUnauthorized, "AUTH_TOKEN_MISSING", "")

	req, rec = doRouteContractRequest(http.MethodPut, "/api/system-configs/scheduler.timeout", "route-contract-token", `not-json`)
	engine.ServeHTTP(rec, req)
	envelope = assertRouteContractEnvelope(t, rec, http.StatusBadRequest, "SYSTEM_CONFIG_INVALID_REQUEST", systemconfigcontract.SystemConfigInvalidRequest.String())
	if envelope.Data["field"] != "body" {
		t.Fatalf("expected body field error data, got %#v", envelope.Data)
	}

	req, rec = doRouteContractRequest(http.MethodPost, "/api/system-configs/nope.key/reset", "route-contract-token", "")
	engine.ServeHTTP(rec, req)
	assertRouteContractEnvelope(t, rec, http.StatusNotFound, "SYSTEM_CONFIG_NOT_FOUND", systemconfigcontract.SystemConfigNotFound.String())
}

func TestSystemConfigStrictRoutesKeepPermissionDeniedContract(t *testing.T) {
	engine := newRouteContractEngine(t, true)

	req, rec := doRouteContractRequest(http.MethodPut, "/api/system-configs/scheduler.timeout", "route-contract-token", `{"value":"45s"}`)
	engine.ServeHTTP(rec, req)
	envelope := assertRouteContractEnvelope(t, rec, http.StatusForbidden, "AUTH_FORBIDDEN", "")
	if envelope.Data["permission"] != systemconfigcontract.SystemConfigWritePermission.String() {
		t.Fatalf("expected denied permission metadata, got %#v", envelope.Data)
	}
}

func TestSystemConfigStrictRoutesUpdateOverride(t *testing.T) {
	engine := newRouteContractEngine(t, false)

	req, rec := doRouteContractRequest(http.MethodPut, "/api/system-configs/scheduler.timeout", "route-contract-token", `{"value":"45s"}`)
	engine.ServeHTTP(rec, req)
	envelope := assertRouteContractEnvelope(t, rec, http.StatusOK, "OK", "")
	if envelope.Data["status"] != "modified" || envelope.Data["effective_value"] != `"45s"` {
		t.Fatalf("expected modified override item, got %#v", envelope.Data)
	}
}
