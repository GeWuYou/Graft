package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
)

func TestRegisterSummaryRouteReturnsVisibleWidgets(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	registry := NewRegistry()
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:                  "core.module-runtime-health",
		ModuleKey:           "core",
		Type:                WidgetTypeHealth,
		Size:                WidgetSizeMedium,
		RequiredPermissions: []string{"modules.runtime.read"},
		Loader: WidgetLoaderFunc(func(context.Context, WidgetRequest) (WidgetPayload, error) {
			return WidgetPayload{"items": []HealthItem{}}, nil
		}),
	})

	if err := Register(
		Registration{
			Config:   &config.Config{App: config.AppConfig{Env: "test"}},
			Registry: registry,
		},
		router.Group("/api"),
		routeAuthService{},
		testAuthorizer{allow: map[string]bool{"modules.runtime.read": true}},
	); err != nil {
		t.Fatalf("register dashboard routes: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/dashboard/summary", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var envelope struct {
		Data struct {
			Widgets []struct {
				ID string `json:"id"`
			} `json:"widgets"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(envelope.Data.Widgets) != 1 || envelope.Data.Widgets[0].ID != "core.module-runtime-health" {
		t.Fatalf("unexpected widgets: %#v", envelope.Data.Widgets)
	}
}

func TestRegisterWidgetRouteHidesUnauthorizedWidget(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	registry := NewRegistry()
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:                  "core.hidden",
		ModuleKey:           "core",
		Type:                WidgetTypeHealth,
		Size:                WidgetSizeSmall,
		RequiredPermissions: []string{"modules.runtime.read"},
		Loader:              noopLoader(),
	})

	if err := Register(
		Registration{Registry: registry},
		router.Group("/api"),
		routeAuthService{},
		testAuthorizer{allow: map[string]bool{}},
	); err != nil {
		t.Fatalf("register dashboard routes: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/dashboard/widgets/core.hidden", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unauthorized widget, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

type routeAuthService struct{}

func (routeAuthService) CurrentUser(context.Context) (*moduleapi.CurrentUser, error) {
	return &moduleapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"}, nil
}

func (routeAuthService) ParseAccessToken(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
	return &moduleapi.AccessTokenClaims{
		UserID:    7,
		SessionID: "session-1",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Minute),
	}, nil
}
