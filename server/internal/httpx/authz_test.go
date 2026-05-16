package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/i18n"
	"graft/server/internal/pluginapi"
)

func newTestLocalizer() *i18n.Service {
	return i18n.New(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
}

type testAuthService struct {
	parseAccessToken func(ctx context.Context, token string) (*pluginapi.AccessTokenClaims, error)
	currentUser      func(ctx context.Context) (*pluginapi.CurrentUser, error)
}

func (s testAuthService) CurrentUser(ctx context.Context) (*pluginapi.CurrentUser, error) {
	return s.currentUser(ctx)
}

func (s testAuthService) ParseAccessToken(ctx context.Context, token string) (*pluginapi.AccessTokenClaims, error) {
	return s.parseAccessToken(ctx, token)
}

type testAuthorizer struct {
	authorize func(ctx context.Context, request pluginapi.RequestAuthContext, permission string) error
}

func (a testAuthorizer) Authorize(ctx context.Context, request pluginapi.RequestAuthContext, permission string) error {
	return a.authorize(ctx, request, permission)
}

func newAuthzTestResolver(t *testing.T, auth pluginapi.AuthService, authorizer pluginapi.Authorizer) container.Resolver {
	t.Helper()

	services := container.New()
	if err := services.RegisterSingleton((*pluginapi.AuthService)(nil), func(_ container.Resolver) (any, error) {
		return auth, nil
	}); err != nil {
		t.Fatalf("register auth service: %v", err)
	}
	if err := services.RegisterSingleton((*pluginapi.Authorizer)(nil), func(_ container.Resolver) (any, error) {
		return authorizer, nil
	}); err != nil {
		t.Fatalf("register authorizer: %v", err)
	}

	return services
}

func newAuthOnlyTestResolver(t *testing.T, auth pluginapi.AuthService) container.Resolver {
	t.Helper()

	services := container.New()
	if err := services.RegisterSingleton((*pluginapi.AuthService)(nil), func(_ container.Resolver) (any, error) {
		return auth, nil
	}); err != nil {
		t.Fatalf("register auth service: %v", err)
	}

	return services
}

func newBearerRequest(path string, token string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	return request
}

func newAuthenticatedClaims() *pluginapi.AccessTokenClaims {
	return &pluginapi.AccessTokenClaims{
		UserID:    7,
		SessionID: "session-1",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Minute),
	}
}

func newAuthenticatedUser() *pluginapi.CurrentUser {
	return &pluginapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"}
}

func newAuthenticatedAuthService() testAuthService {
	return testAuthService{
		parseAccessToken: func(context.Context, string) (*pluginapi.AccessTokenClaims, error) {
			return newAuthenticatedClaims(), nil
		},
		currentUser: func(context.Context) (*pluginapi.CurrentUser, error) {
			return newAuthenticatedUser(), nil
		},
	}
}

func assertAuthenticatedRequestAllowed(t *testing.T, permission string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(nil, newAuthOnlyTestResolver(t, newAuthenticatedAuthService()), permission))
	engine.GET("/api/profile", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	ctx.Request = newBearerRequest("/api/profile", "token-1")
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func assertPermissionRejectsTokenError(t *testing.T, requestToken string, parseErr error, wantKey string, wantCode string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	localizer := newTestLocalizer()
	resolver := newAuthzTestResolver(t,
		testAuthService{
			parseAccessToken: func(context.Context, string) (*pluginapi.AccessTokenClaims, error) {
				return nil, parseErr
			},
			currentUser: func(context.Context) (*pluginapi.CurrentUser, error) {
				t.Fatal("current user should not be called when token parse fails")
				return nil, nil
			},
		},
		testAuthorizer{
			authorize: func(context.Context, pluginapi.RequestAuthContext, string) error {
				t.Fatal("authorize should not be called when token parse fails")
				return nil
			},
		},
	)

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(localizer, resolver, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	ctx.Request = newBearerRequest("/api/users/1", requestToken)
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var payload ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != wantKey || payload.Code != wantCode {
		t.Fatalf("expected token error contract %s/%s, got %#v", wantKey, wantCode, payload)
	}
}

// TestRequirePermissionRejectsMissingBearerToken 验证缺少访问令牌时会被后端守卫直接拒绝。
func TestRequirePermissionRejectsMissingBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	localizer := newTestLocalizer()
	resolver := newAuthzTestResolver(t,
		testAuthService{
			parseAccessToken: func(context.Context, string) (*pluginapi.AccessTokenClaims, error) {
				t.Fatal("parse access token should not be called without bearer token")
				return nil, nil
			},
			currentUser: func(context.Context) (*pluginapi.CurrentUser, error) {
				t.Fatal("current user should not be called without bearer token")
				return nil, nil
			},
		},
		testAuthorizer{
			authorize: func(context.Context, pluginapi.RequestAuthContext, string) error {
				t.Fatal("authorize should not be called without bearer token")
				return nil
			},
		},
	)

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(localizer, resolver, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var payload ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.token_missing" || payload.Code != "AUTH_TOKEN_MISSING" {
		t.Fatalf("expected missing token contract, got %#v", payload)
	}
}

// TestRequirePermissionRejectsPermissionDenied 验证认证成功但缺少权限时会返回 403。
func TestRequirePermissionRejectsPermissionDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	localizer := newTestLocalizer()
	resolver := newAuthzTestResolver(t,
		testAuthService{
			parseAccessToken: func(context.Context, string) (*pluginapi.AccessTokenClaims, error) {
				return newAuthenticatedClaims(), nil
			},
			currentUser: func(context.Context) (*pluginapi.CurrentUser, error) {
				return newAuthenticatedUser(), nil
			},
		},
		testAuthorizer{
			authorize: func(context.Context, pluginapi.RequestAuthContext, string) error {
				return pluginapi.ErrPermissionDenied
			},
		},
	)

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(localizer, resolver, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	request := newBearerRequest("/api/users/1", "token-1")
	request.Header.Set(i18n.LocaleHeader, "en-US")
	ctx.Request = request
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}

	var payload ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.forbidden" || payload.Code != "AUTH_FORBIDDEN" {
		t.Fatalf("expected forbidden message key, got %#v", payload)
	}
	if payload.Locale != "en-US" {
		t.Fatalf("expected requested locale to be echoed, got %#v", payload)
	}
	if payload.Details["permission"] != "user.read" {
		t.Fatalf("expected denied permission to be echoed, got %#v", payload)
	}
}

// TestRequirePermissionAllowsAuthorizedRequest 验证 bearer token 和授权判断都通过时，请求可以继续执行。
func TestRequirePermissionAllowsAuthorizedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resolver := newAuthzTestResolver(t,
		testAuthService{
			parseAccessToken: func(context.Context, string) (*pluginapi.AccessTokenClaims, error) {
				return newAuthenticatedClaims(), nil
			},
			currentUser: func(ctx context.Context) (*pluginapi.CurrentUser, error) {
				requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
				if !ok || requestAuth.Claims == nil || requestAuth.Claims.UserID != 7 {
					t.Fatalf("expected request auth claims to be populated before CurrentUser, got %#v, ok=%v", requestAuth, ok)
				}
				return newAuthenticatedUser(), nil
			},
		},
		testAuthorizer{
			authorize: func(_ context.Context, request pluginapi.RequestAuthContext, _ string) error {
				if request.User == nil || request.User.ID != 7 {
					t.Fatalf("expected request user to be populated before Authorize, got %#v", request.User)
				}
				return nil
			},
		},
	)

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(nil, resolver, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		requestAuth, ok := pluginapi.RequestAuthContextFromContext(inner.Request.Context())
		if !ok || requestAuth.User == nil || requestAuth.User.ID != 7 {
			t.Fatalf("expected handler context to carry current user, got %#v, ok=%v", requestAuth, ok)
		}
		inner.Status(http.StatusOK)
	})

	ctx.Request = newBearerRequest("/api/users/1", "token-1")
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

// TestRequirePermissionAllowsAuthenticatedRequestWhenPermissionCodeBlank 验证空权限码只要求建立登录态。
func TestRequirePermissionAllowsAuthenticatedRequestWhenPermissionCodeBlank(t *testing.T) {
	assertAuthenticatedRequestAllowed(t, " ")
}

// TestRequirePermissionAllowsBlankPermissionWithoutAuthorizer 验证空权限码路由不会隐式要求 Authorizer 已注册。
func TestRequirePermissionAllowsBlankPermissionWithoutAuthorizer(t *testing.T) {
	assertAuthenticatedRequestAllowed(t, "")
}

// TestRequirePermissionMapsInvalidTokenToUnauthorized 验证无效 token 会收敛为未登录响应。
func TestRequirePermissionMapsInvalidTokenToUnauthorized(t *testing.T) {
	assertPermissionRejectsTokenError(t, "bad-token", pluginapi.ErrInvalidAccessToken, "auth.token_invalid", "AUTH_TOKEN_INVALID")
}

// TestRequirePermissionMapsExpiredTokenToUnauthorized 验证过期 token 会收敛为稳定的过期响应，
// 以便前端仅对该分支触发 refresh。
func TestRequirePermissionMapsExpiredTokenToUnauthorized(t *testing.T) {
	assertPermissionRejectsTokenError(t, "expired-token", pluginapi.ErrExpiredAccessToken, "auth.token_expired", "AUTH_TOKEN_EXPIRED")
}

// TestRequirePermissionFailsClosedWhenAuthDependenciesMissing 验证未装配 auth 依赖时会拒绝请求而不是继续执行。
func TestRequirePermissionFailsClosedWhenAuthDependenciesMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	localizer := newTestLocalizer()

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	handled := false
	engine.Use(RequirePermission(localizer, container.New(), "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		handled = true
		inner.Status(http.StatusOK)
	})

	ctx.Request = newBearerRequest("/api/users/1", "token-1")
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if handled {
		t.Fatal("expected request to fail closed before reaching handler")
	}
}

var (
	_ pluginapi.AuthService = testAuthService{}
	_ pluginapi.Authorizer  = testAuthorizer{}
	_                       = errors.Is
)
