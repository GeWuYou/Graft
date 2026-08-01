package container

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

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

var _ moduleapi.SavedViewService = savedViewServiceStub{}
