package announcement

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	announcementcontract "graft/server/modules/announcement/contract"
)

type announcementSavedViewServiceStub struct{}

func (announcementSavedViewServiceStub) List(context.Context, uint64, string) ([]moduleapi.SavedView, error) {
	return nil, nil
}
func (announcementSavedViewServiceStub) Create(context.Context, moduleapi.SavedViewCreateInput) (moduleapi.SavedView, error) {
	return moduleapi.SavedView{}, nil
}
func (announcementSavedViewServiceStub) Update(context.Context, moduleapi.SavedViewUpdateInput) (moduleapi.SavedView, error) {
	return moduleapi.SavedView{}, nil
}
func (announcementSavedViewServiceStub) Delete(context.Context, uint64, string, uint64) error {
	return nil
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
	registerAnnouncementSavedViewRoutes(engine.Group(announcementcontract.AnnouncementGroup), moduleCtx, announcementSavedViewServiceStub{}, func(ctx *gin.Context) {
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

var _ moduleapi.SavedViewService = announcementSavedViewServiceStub{}
