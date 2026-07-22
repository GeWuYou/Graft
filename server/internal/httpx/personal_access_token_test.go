package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/moduleapi"
)

func TestRequirePersonalAccessTokenInjectsVerifiedCaller(t *testing.T) {
	gin.SetMode(gin.TestMode)
	caller := moduleapi.PersonalAccessTokenCaller{
		TokenID:   17,
		User:      moduleapi.CurrentUser{ID: 7, Username: "alice"},
		Scopes:    []string{"audit.read"},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	service := personalAccessTokenMiddlewareService{authenticate: func(_ context.Context, token string) (moduleapi.PersonalAccessTokenCaller, error) {
		if token != "gpat_test" {
			t.Fatalf("unexpected token %q", token)
		}
		return caller, nil
	}}
	engine := gin.New()
	engine.Use(RequirePersonalAccessToken(nil, service))
	engine.GET("/mcp", func(ginCtx *gin.Context) {
		actual, ok := moduleapi.PersonalAccessTokenCallerFromContext(ginCtx.Request.Context())
		if !ok || actual.TokenID != caller.TokenID || actual.User.ID != caller.User.ID {
			t.Fatalf("expected verified personal token caller, got %#v, ok=%t", actual, ok)
		}
		requestAuth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
		if !ok || requestAuth.User == nil || requestAuth.User.ID != caller.User.ID {
			t.Fatalf("expected request auth context from personal token caller, got %#v, ok=%t", requestAuth, ok)
		}
		ginCtx.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	request.Header.Set("Authorization", "Bearer gpat_test")
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

type personalAccessTokenMiddlewareService struct {
	authenticate func(context.Context, string) (moduleapi.PersonalAccessTokenCaller, error)
}

func (s personalAccessTokenMiddlewareService) CreateCurrentUserPersonalAccessToken(context.Context, moduleapi.PersonalAccessTokenCreateInput) (moduleapi.PersonalAccessTokenIssued, error) {
	return moduleapi.PersonalAccessTokenIssued{}, nil
}

func (s personalAccessTokenMiddlewareService) ListCurrentUserPersonalAccessTokens(context.Context, int) ([]moduleapi.PersonalAccessTokenSummary, error) {
	return nil, nil
}

func (s personalAccessTokenMiddlewareService) RevokeCurrentUserPersonalAccessToken(context.Context, uint64) error {
	return nil
}

func (s personalAccessTokenMiddlewareService) AuthenticatePersonalAccessToken(ctx context.Context, token string) (moduleapi.PersonalAccessTokenCaller, error) {
	return s.authenticate(ctx, token)
}

var _ moduleapi.PersonalAccessTokenService = personalAccessTokenMiddlewareService{}
