package build

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/config"
	containerdi "graft/server/internal/container"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

func TestBuildListQueryBindsBuildOwnedHistoryFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	request := httptest.NewRequest("GET", "/api/build/jobs?limit=50&offset=100&application_id=app_01JZ5R6M7N8P9Q0R1S2T3V4W5X&image_repository=example%2Fapp&image_tag=v1&created_after=2026-08-01T00%3A00%3A00Z&created_before=2026-08-02T00%3A00%3A00Z", nil)
	context.Request = request

	query, ok := buildListQuery(context)
	if !ok {
		t.Fatal("expected valid Build history query")
	}
	if query.Limit != 50 || query.Offset != 100 || query.ApplicationID == nil || *query.ApplicationID != "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X" {
		t.Fatalf("unexpected pagination or application filter: %#v", query)
	}
	if query.ImageRepository == nil || *query.ImageRepository != "example/app" || query.ImageTag == nil || *query.ImageTag != "v1" {
		t.Fatalf("unexpected image filters: %#v", query)
	}
	if query.CreatedAfter == nil || query.CreatedBefore == nil || !query.CreatedAfter.Before(*query.CreatedBefore) {
		t.Fatalf("unexpected creation range: %#v", query)
	}
}

func TestBuildListQueryRejectsInvalidHistoryRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/api/build/jobs?created_after=2026-08-02T00%3A00%3A00Z&created_before=2026-08-01T00%3A00%3A00Z", nil)

	if _, ok := buildListQuery(context); ok {
		t.Fatal("expected reverse creation range to be rejected")
	}
}

type buildRouteAuthService struct{}

func (buildRouteAuthService) CurrentUser(context.Context) (*moduleapi.CurrentUser, error) {
	return &moduleapi.CurrentUser{ID: 7, Username: "admin", DisplayName: "Admin"}, nil
}

func (buildRouteAuthService) ParseAccessToken(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
	return &moduleapi.AccessTokenClaims{UserID: 7, SessionID: "session-1", ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil
}

type buildRouteAuthorizer struct{}

func (buildRouteAuthorizer) Authorize(context.Context, moduleapi.RequestAuthContext, string) error {
	return nil
}

func newBuildRouteTestEngine(t *testing.T, tasks *recordingBuildTasks, repository *recordingBuildRepository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	services := containerdi.New()
	if err := services.RegisterSingleton((*moduleapi.AuthService)(nil), func(containerdi.Resolver) (any, error) { return buildRouteAuthService{}, nil }); err != nil {
		t.Fatal(err)
	}
	if err := services.RegisterSingleton((*moduleapi.Authorizer)(nil), func(containerdi.Resolver) (any, error) { return buildRouteAuthorizer{}, nil }); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(&recordingBuildContexts{}, tasks, &recordingBuildDocker{}, repository)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &module.Context{Router: engine.Group(""), Services: services, EventBus: eventbus.New(zap.NewNop()), I18n: i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}})}
	if err := registerRoutes(ctx, service); err != nil {
		t.Fatal(err)
	}
	return engine
}

func buildAuthorizedRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer route-test-token")
	request.Header.Set("Idempotency-Key", "route-test-key")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func TestBuildRoutesRejectInvalidListQuery(t *testing.T) {
	engine := newBuildRouteTestEngine(t, &recordingBuildTasks{}, &recordingBuildRepository{})
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, buildAuthorizedRequest(http.MethodGet, "/api/build/jobs?limit=0", ""))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
}

func TestBuildRoutesMapBadBuildIDAndInternalReadFailureSeparately(t *testing.T) {
	t.Run("bad build id", func(t *testing.T) {
		engine := newBuildRouteTestEngine(t, &recordingBuildTasks{}, &recordingBuildRepository{})
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, buildAuthorizedRequest(http.MethodGet, "/api/build/jobs/%20", ""))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
		}
	})
	t.Run("internal repository failure", func(t *testing.T) {
		engine := newBuildRouteTestEngine(t, &recordingBuildTasks{}, &recordingBuildRepository{getBuildIDErr: errors.New("database unavailable")})
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, buildAuthorizedRequest(http.MethodGet, "/api/build/jobs/build_test", ""))
		if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "common.internalError") {
			t.Fatalf("unexpected response: status=%d body=%s", response.Code, response.Body.String())
		}
	})
}

func TestBuildSubmitRouteUsesInternalErrorKeyForRuntimeFailure(t *testing.T) {
	engine := newBuildRouteTestEngine(t, &recordingBuildTasks{err: errors.New("task runtime unavailable")}, &recordingBuildRepository{})
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, buildAuthorizedRequest(http.MethodPost, "/api/build/jobs", `{"application_id":"app_01JZ5R6M7N8P9Q0R1S2T3V4W5X","context_path":"src","dockerfile_path":"Dockerfile","image_repository":"example/app","image_tag":"v1"}`))
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "common.internalError") {
		t.Fatalf("unexpected response: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestBuildPaginationUsesStoreBounds(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodGet, "/api/build/jobs", nil)
	query, ok := buildPaginationQuery(context)
	if !ok || query.Limit != buildstore.DefaultListLimit {
		t.Fatalf("default query = %#v, ok=%v", query, ok)
	}
}
