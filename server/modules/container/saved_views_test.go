package container

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
)

type savedViewServiceStub struct{}

func (savedViewServiceStub) List(context.Context, uint64, string) ([]moduleapi.SavedView, error) {
	return nil, nil
}
func (savedViewServiceStub) Create(context.Context, moduleapi.SavedViewCreateInput) (moduleapi.SavedView, error) {
	return moduleapi.SavedView{}, nil
}
func (savedViewServiceStub) Update(context.Context, moduleapi.SavedViewUpdateInput) (moduleapi.SavedView, error) {
	return moduleapi.SavedView{}, nil
}
func (savedViewServiceStub) Delete(context.Context, uint64, string, uint64) error { return nil }

type recordedSavedViewService struct {
	listSurface   string
	createInput   moduleapi.SavedViewCreateInput
	updateInput   moduleapi.SavedViewUpdateInput
	deleteOwnerID uint64
	deleteSurface string
	deleteID      uint64
}

func (s *recordedSavedViewService) List(_ context.Context, ownerID uint64, surface string) ([]moduleapi.SavedView, error) {
	if ownerID != 7 {
		return nil, moduleapi.ErrSavedViewInvalidInput
	}
	s.listSurface = surface
	return []moduleapi.SavedView{{ID: 41, Name: "Saved", QueryState: json.RawMessage(`{"keyword":"api"}`), PageSize: 20, VisibleColumns: []string{"name"}, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(2, 0)}}, nil
}

func (s *recordedSavedViewService) Create(_ context.Context, input moduleapi.SavedViewCreateInput) (moduleapi.SavedView, error) {
	s.createInput = input
	return moduleapi.SavedView{ID: 42, Name: input.Name, QueryState: input.QueryState, PageSize: input.PageSize, VisibleColumns: input.VisibleColumns, IsDefault: input.IsDefault, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(2, 0)}, nil
}

func (s *recordedSavedViewService) Update(_ context.Context, input moduleapi.SavedViewUpdateInput) (moduleapi.SavedView, error) {
	s.updateInput = input
	return moduleapi.SavedView{ID: input.ID, Name: input.Name, QueryState: input.QueryState, PageSize: input.PageSize, VisibleColumns: input.VisibleColumns, IsDefault: input.IsDefault, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(2, 0)}, nil
}

func (s *recordedSavedViewService) Delete(_ context.Context, ownerID uint64, surface string, id uint64) error {
	s.deleteOwnerID, s.deleteSurface, s.deleteID = ownerID, surface, id
	return nil
}

func TestContainerSavedViewValidationUsesListBinders(t *testing.T) {
	t.Parallel()
	moduleCtx := newTestContext()
	validStates := []struct {
		name       string
		state      string
		column     string
		definition containerSavedViewDefinition
	}{
		{"container", `{"state":"running","runtime_target_id":1}`, "state", containerSavedViewDefinitions[0]},
		{"image", `{"unused":true}`, "tags", containerSavedViewDefinitions[1]},
		{"network", `{"usage":"used","source":"docker"}`, "name", containerSavedViewDefinitions[2]},
		{"volume", `{"size_min_bytes":1,"sort_order":"asc"}`, "size_bytes", containerSavedViewDefinitions[3]},
	}
	for _, test := range validStates {
		t.Run(test.name, func(t *testing.T) {
			request := validSavedViewRequest(test.state, test.column)
			if err := validateContainerSavedView(request, moduleCtx, test.definition); err != nil {
				t.Fatalf("valid saved view rejected: %v", err)
			}
			unknown := validSavedViewRequest(`{"unknown":"value"}`, test.column)
			if err := validateContainerSavedView(unknown, moduleCtx, test.definition); err == nil {
				t.Fatal("unknown query field must be rejected")
			}
			badColumn := validSavedViewRequest(test.state, "unknown")
			if err := validateContainerSavedView(badColumn, moduleCtx, test.definition); err == nil {
				t.Fatal("unknown visible column must be rejected")
			}
		})
	}
}

func validSavedViewRequest(state, column string) httpx.SavedViewRequest {
	return httpx.SavedViewRequest{Name: "Test", QueryState: json.RawMessage(state), PageSize: 20, VisibleColumns: []string{column}}
}

func TestContainerSavedViewRoutesUseCanonicalImagePath(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	moduleCtx := newTestContext()
	moduleCtx.Router = engine
	registerContainerSavedViewRoutes(moduleCtx, savedViewServiceStub{}, func(ctx *gin.Context) { ctx.Next() })

	routes := make(map[string]struct{})
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, route := range []string{
		http.MethodGet + " /ops/containers/saved-views",
		http.MethodGet + " /docker/images/saved-views",
		http.MethodGet + " /ops/docker/networks/saved-views",
		http.MethodGet + " /ops/docker/volumes/saved-views",
	} {
		if _, ok := routes[route]; !ok {
			t.Fatalf("expected route %s, got %#v", route, routes)
		}
	}
	if _, ok := routes[http.MethodGet+" /ops/docker/images/saved-views"]; ok {
		t.Fatal("deprecated ops image saved-view route must not be registered")
	}
}

func TestContainerSavedViewRoutesServeCRUDForEachSurface(t *testing.T) {
	gin.SetMode(gin.TestMode)
	moduleCtx := newTestContext()
	service := &recordedSavedViewService{}
	engine := gin.New()
	moduleCtx.Router = engine
	registerContainerSavedViewRoutes(moduleCtx, service, func(ctx *gin.Context) {
		ctx.Request = ctx.Request.WithContext(moduleapi.WithRequestAuthContext(ctx.Request.Context(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}}))
	})

	surfaces := []struct {
		name       string
		path       string
		surface    string
		queryState string
		column     string
	}{
		{"container", "/ops/containers/saved-views", containerListSavedViewSurface, `{"state":"running"}`, "name"},
		{"image", "/docker/images/saved-views", dockerImageListSavedViewSurface, `{"unused":true}`, "tags"},
		{"network", "/ops/docker/networks/saved-views", dockerNetworkListSavedViewSurface, `{"usage":"used"}`, "name"},
		{"volume", "/ops/docker/volumes/saved-views", dockerVolumeListSavedViewSurface, `{"sort_order":"asc"}`, "name"},
	}
	for _, test := range surfaces {
		t.Run(test.name, func(t *testing.T) {
			requestBody := `{"name":"Saved","query_state":` + test.queryState + `,"page_size":20,"visible_columns":["` + test.column + `"]}`
			assertSavedViewHTTPStatus(t, engine, http.MethodGet, test.path, "", http.StatusOK)
			if service.listSurface != test.surface {
				t.Fatalf("list surface = %q, want %q", service.listSurface, test.surface)
			}
			assertSavedViewHTTPStatus(t, engine, http.MethodPost, test.path, requestBody, http.StatusCreated)
			if service.createInput.OwnerUserID != 7 || service.createInput.SurfaceKey != test.surface {
				t.Fatalf("create input = %#v, want owner 7 and surface %q", service.createInput, test.surface)
			}
			assertSavedViewHTTPStatus(t, engine, http.MethodPut, test.path+"/42", requestBody, http.StatusOK)
			if service.updateInput.ID != 42 || service.updateInput.OwnerUserID != 7 || service.updateInput.SurfaceKey != test.surface {
				t.Fatalf("update input = %#v, want id 42, owner 7 and surface %q", service.updateInput, test.surface)
			}
			assertSavedViewHTTPStatus(t, engine, http.MethodDelete, test.path+"/42", "", http.StatusNoContent)
			if service.deleteOwnerID != 7 || service.deleteSurface != test.surface || service.deleteID != 42 {
				t.Fatalf("delete input = (%d, %q, %d), want (7, %q, 42)", service.deleteOwnerID, service.deleteSurface, service.deleteID, test.surface)
			}
		})
	}
}

func assertSavedViewHTTPStatus(t *testing.T, engine http.Handler, method, path, body string, wantStatus int) {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d: %s", method, path, response.Code, wantStatus, response.Body.String())
	}
}

var _ moduleapi.SavedViewService = savedViewServiceStub{}
