package rbac

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
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

type recordedRBACSavedViewService struct {
	listOwner     uint64
	listSurface   string
	createInput   moduleapi.SavedViewCreateInput
	updateInput   moduleapi.SavedViewUpdateInput
	deleteOwner   uint64
	deleteSurface string
	deleteID      uint64
}

func (s *recordedRBACSavedViewService) List(_ context.Context, owner uint64, surface string) ([]moduleapi.SavedView, error) {
	s.listOwner, s.listSurface = owner, surface
	return []moduleapi.SavedView{{ID: 41, Name: "Saved", QueryState: json.RawMessage(`{"keyword":"role"}`), PageSize: 20, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(2, 0)}}, nil
}

func (s *recordedRBACSavedViewService) Create(_ context.Context, input moduleapi.SavedViewCreateInput) (moduleapi.SavedView, error) {
	s.createInput = input
	return moduleapi.SavedView{ID: 42, Name: input.Name, QueryState: input.QueryState, PageSize: input.PageSize, VisibleColumns: input.VisibleColumns, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(2, 0)}, nil
}

func (s *recordedRBACSavedViewService) Update(_ context.Context, input moduleapi.SavedViewUpdateInput) (moduleapi.SavedView, error) {
	s.updateInput = input
	return moduleapi.SavedView{ID: input.ID, Name: input.Name, QueryState: input.QueryState, PageSize: input.PageSize, VisibleColumns: input.VisibleColumns, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(2, 0)}, nil
}

func (s *recordedRBACSavedViewService) Delete(_ context.Context, owner uint64, surface string, id uint64) error {
	s.deleteOwner, s.deleteSurface, s.deleteID = owner, surface, id
	return nil
}

func TestValidateRBACSavedViews(t *testing.T) {
	role := httpx.SavedViewRequest{Name: "Builtin", QueryState: json.RawMessage(`{"type":"builtin"}`), PageSize: 20, VisibleColumns: []string{"role", "builtin"}}
	if err := validateRBACSavedView(role, roleSavedViewDefinition); err != nil {
		t.Fatalf("validate role saved view: %v", err)
	}
	permission := httpx.SavedViewRequest{Name: "Security", QueryState: json.RawMessage(`{"keyword":"read","module":"rbac"}`), PageSize: 50, VisibleColumns: []string{"permission", "module"}}
	if err := validateRBACSavedView(permission, permissionSavedViewDefinition); err != nil {
		t.Fatalf("validate permission saved view: %v", err)
	}
	invalid := permission
	invalid.QueryState = json.RawMessage(`{"unknown":true}`)
	if err := validateRBACSavedView(invalid, permissionSavedViewDefinition); err == nil {
		t.Fatal("expected unknown permission query field to fail")
	}
}

//nolint:gocyclo // CRUD handler 契约需要在同一请求序列中直接核对参数归属。
func TestRBACSavedViewHandlersServeCRUDAndRequireOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &recordedRBACSavedViewService{}
	engine := gin.New()
	ctx := &module.Context{Router: engine}
	group := engine.Group("/roles")
	registerRBACSavedViewRoutes(group, ctx, service, func(c *gin.Context) {
		c.Request = c.Request.WithContext(moduleapi.WithRequestAuthContext(c.Request.Context(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}}))
	}, roleSavedViewDefinition)

	body := `{"name":"Saved","query_state":{"keyword":"role"},"page_size":20,"visible_columns":["role"]}`
	request := func(method, path, payload string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(payload))
		if payload != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, req)
		return res
	}
	if got := request(http.MethodGet, "/roles/saved-views", "").Code; got != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", got, http.StatusOK)
	}
	if service.listOwner != 7 || service.listSurface != roleSavedViewDefinition.surface {
		t.Fatalf("list args = (%d, %q)", service.listOwner, service.listSurface)
	}
	if got := request(http.MethodPost, "/roles/saved-views", body).Code; got != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d", got, http.StatusCreated)
	}
	if service.createInput.OwnerUserID != 7 || service.createInput.SurfaceKey != roleSavedViewDefinition.surface {
		t.Fatalf("create input = %#v", service.createInput)
	}
	if got := request(http.MethodPut, "/roles/saved-views/42", body).Code; got != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d", got, http.StatusOK)
	}
	if service.updateInput.ID != 42 || service.updateInput.OwnerUserID != 7 || service.updateInput.SurfaceKey != roleSavedViewDefinition.surface {
		t.Fatalf("update input = %#v", service.updateInput)
	}
	if got := request(http.MethodDelete, "/roles/saved-views/42", "").Code; got != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want %d", got, http.StatusNoContent)
	}
	if service.deleteOwner != 7 || service.deleteSurface != roleSavedViewDefinition.surface || service.deleteID != 42 {
		t.Fatalf("delete args = (%d, %q, %d)", service.deleteOwner, service.deleteSurface, service.deleteID)
	}

	unauthenticated := httptest.NewRecorder()
	missingOwner := gin.New()
	missingOwner.GET("/saved-views", handleRBACSavedViewList(nil, service, roleSavedViewDefinition))
	missingOwner.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/saved-views", nil))
	if unauthenticated.Code != http.StatusBadRequest {
		t.Fatalf("unauthenticated GET status = %d, want %d", unauthenticated.Code, http.StatusBadRequest)
	}
}

var _ moduleapi.SavedViewService = (*recordedRBACSavedViewService)(nil)
