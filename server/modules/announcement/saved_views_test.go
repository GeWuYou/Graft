package announcement

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
	announcementcontract "graft/server/modules/announcement/contract"
)

type announcementSavedViewServiceStub struct {
	listOwner     uint64
	listSurface   string
	createInput   moduleapi.SavedViewCreateInput
	updateInput   moduleapi.SavedViewUpdateInput
	deleteOwner   uint64
	deleteID      uint64
	deleteSurface string
}

func (s *announcementSavedViewServiceStub) List(_ context.Context, owner uint64, surface string) ([]moduleapi.SavedView, error) {
	s.listOwner, s.listSurface = owner, surface
	return []moduleapi.SavedView{s.view(1)}, nil
}
func (s *announcementSavedViewServiceStub) Create(_ context.Context, input moduleapi.SavedViewCreateInput) (moduleapi.SavedView, error) {
	s.createInput = input
	return s.view(2), nil
}
func (s *announcementSavedViewServiceStub) Update(_ context.Context, input moduleapi.SavedViewUpdateInput) (moduleapi.SavedView, error) {
	s.updateInput = input
	return s.view(input.ID), nil
}
func (s *announcementSavedViewServiceStub) Delete(_ context.Context, owner uint64, surface string, id uint64) error {
	s.deleteOwner, s.deleteSurface, s.deleteID = owner, surface, id
	return nil
}

func (announcementSavedViewServiceStub) view(id uint64) moduleapi.SavedView {
	now := time.Now().UTC()
	return moduleapi.SavedView{ID: id, Name: "Saved announcements", QueryState: json.RawMessage(`{"status":"published"}`), PageSize: 20, VisibleColumns: []string{"title"}, CreatedAt: now, UpdatedAt: now}
}

func TestValidateAnnouncementManagementSavedView(t *testing.T) {
	t.Parallel()
	moduleCtx := newAnnouncementTestContext(nil)
	valid := httpx.SavedViewRequest{
		Name:           "Published warnings",
		QueryState:     json.RawMessage(`{"status":"published","level":"warning","pinned":true,"sort":"updated_desc","keyword":"maintenance"}`),
		PageSize:       20,
		VisibleColumns: []string{"title", "status", "operation"},
	}
	if err := validateAnnouncementManagementSavedView(valid, moduleCtx); err != nil {
		t.Fatalf("valid saved view rejected: %v", err)
	}
	for _, invalid := range []httpx.SavedViewRequest{
		{Name: "bad page", QueryState: valid.QueryState, PageSize: 25},
		{Name: "unknown field", QueryState: json.RawMessage(`{"delivery_mode":"popup"}`), PageSize: 20},
		{Name: "bad status", QueryState: json.RawMessage(`{"status":"visible"}`), PageSize: 20},
		{Name: "bad column", QueryState: valid.QueryState, PageSize: 20, VisibleColumns: []string{"unknown"}},
	} {
		if err := validateAnnouncementManagementSavedView(invalid, moduleCtx); err == nil {
			t.Fatalf("invalid saved view accepted: %#v", invalid)
		}
	}
}

func TestRegisterAnnouncementSavedViewRoutes(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	moduleCtx := newAnnouncementTestContext(engine)
	registerAnnouncementSavedViewRoutes(engine.Group(announcementcontract.AnnouncementGroup), moduleCtx, &announcementSavedViewServiceStub{}, func(ctx *gin.Context) {
		ctx.Next()
	})
	want := map[string]struct{}{
		http.MethodGet + " /announcements/saved-views":            {},
		http.MethodPost + " /announcements/saved-views":           {},
		http.MethodPut + " /announcements/saved-views/:viewId":    {},
		http.MethodDelete + " /announcements/saved-views/:viewId": {},
	}
	got := make(map[string]struct{})
	for _, route := range engine.Routes() {
		got[route.Method+" "+route.Path] = struct{}{}
	}
	for route := range want {
		if _, ok := got[route]; !ok {
			t.Fatalf("missing announcement saved-view route %s; routes=%v", route, got)
		}
	}
}

func TestAnnouncementSavedViewRoutesRegisterBeforeDetailRoute(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	moduleCtx := newAnnouncementTestContext(engine)
	service, err := NewService(testAnnouncementRepository{})
	if err != nil {
		t.Fatalf("new announcement service: %v", err)
	}
	guard := announcementRouteTestAuth(42)
	if err := registerAnnouncementRoutes(moduleCtx, service, announcementGuards{
		authenticated: guard,
		read:          guard,
		create:        guard,
		update:        guard,
		publish:       guard,
		delete:        guard,
	}, &announcementSavedViewServiceStub{}); err != nil {
		t.Fatalf("register announcement routes with saved views: %v", err)
	}

	want := map[string]struct{}{
		http.MethodGet + " /api/announcements/saved-views":            {},
		http.MethodPost + " /api/announcements/saved-views":           {},
		http.MethodPut + " /api/announcements/saved-views/:viewId":    {},
		http.MethodDelete + " /api/announcements/saved-views/:viewId": {},
		http.MethodGet + " /api/announcements/:id":                    {},
	}
	for _, route := range engine.Routes() {
		delete(want, route.Method+" "+route.Path)
	}
	if len(want) != 0 {
		t.Fatalf("missing static saved-view or detail routes: %v", want)
	}
}

func TestAnnouncementSavedViewHandlersUseOwnerAndSurface(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	moduleCtx := newAnnouncementTestContext(engine)
	service := &announcementSavedViewServiceStub{}
	registerAnnouncementSavedViewRoutes(engine.Group(announcementcontract.AnnouncementGroup), moduleCtx, service, announcementRouteTestAuth(42))

	body := `{"name":"Published warnings","query_state":{"status":"published"},"page_size":20,"visible_columns":["title"],"is_default":true}`
	if response := performAnnouncementSavedViewRequest(engine, http.MethodGet, "/announcements/saved-views", ""); response.Code != http.StatusOK {
		t.Fatalf("GET saved views status = %d, want %d", response.Code, http.StatusOK)
	}
	if service.listOwner != 42 || service.listSurface != announcementManagementSavedViewSurface {
		t.Fatalf("list ownership = (%d, %q)", service.listOwner, service.listSurface)
	}
	if response := performAnnouncementSavedViewRequest(engine, http.MethodPost, "/announcements/saved-views", body); response.Code != http.StatusCreated {
		t.Fatalf("POST saved views status = %d, want %d", response.Code, http.StatusCreated)
	}
	if service.createInput.OwnerUserID != 42 || service.createInput.SurfaceKey != announcementManagementSavedViewSurface {
		t.Fatalf("create input = %#v", service.createInput)
	}
	if response := performAnnouncementSavedViewRequest(engine, http.MethodPut, "/announcements/saved-views/7", body); response.Code != http.StatusOK {
		t.Fatalf("PUT saved view status = %d, want %d", response.Code, http.StatusOK)
	}
	if service.updateInput.ID != 7 || service.updateInput.OwnerUserID != 42 || service.updateInput.SurfaceKey != announcementManagementSavedViewSurface {
		t.Fatalf("update input = %#v", service.updateInput)
	}
	if response := performAnnouncementSavedViewRequest(engine, http.MethodDelete, "/announcements/saved-views/7", ""); response.Code != http.StatusNoContent {
		t.Fatalf("DELETE saved view status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if service.deleteOwner != 42 || service.deleteSurface != announcementManagementSavedViewSurface || service.deleteID != 7 {
		t.Fatalf("delete input = (%d, %q, %d)", service.deleteOwner, service.deleteSurface, service.deleteID)
	}
}

func performAnnouncementSavedViewRequest(engine *gin.Engine, method, path, requestBody string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(requestBody))
	if requestBody != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	engine.ServeHTTP(recorder, req)
	return recorder
}

var _ moduleapi.SavedViewService = (*announcementSavedViewServiceStub)(nil)
