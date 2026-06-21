package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/dashboard"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	notificationcontract "graft/server/modules/notification/contract"
	notificationstore "graft/server/modules/notification/store"
)

func TestNotificationRoutesScopeToCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 6, 9, 8, 0, 0, 0, time.UTC)
	repo := &routeTestNotificationRepository{
		items: []notificationstore.Notification{
			routeTestNotification(100, 42, "Unread item", now, nil),
			routeTestNotification(101, 7, "Wrong user", now, nil),
			routeTestNotification(102, 42, "Read item", now, &now),
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	fixture := newNotificationRouteTestContext()
	registerNotificationRoutes(fixture.ctx, service, notificationGuards{view: routeTestAuth(42), read: routeTestAuth(42)})

	request := httptest.NewRequest(http.MethodGet, "/api/notifications?status=unread&page_size=10", nil)
	recorder := httptest.NewRecorder()
	fixture.engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var response httpx.SuccessResponse[map[string]any]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := response.Data
	items, ok := data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one current-user unread item, got %#v", data["items"])
	}
	if repo.listQuery.RecipientUserID != 42 || repo.listQuery.Status != "unread" {
		t.Fatalf("unexpected list query: %#v", repo.listQuery)
	}
}

func TestNotificationRoutesRejectInvalidListQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &routeTestNotificationRepository{}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	fixture := newNotificationRouteTestContext()
	registerNotificationRoutes(fixture.ctx, service, notificationGuards{view: routeTestAuth(42), read: routeTestAuth(42)})

	request := httptest.NewRequest(http.MethodGet, "/api/notifications?page=bad", nil)
	recorder := httptest.NewRecorder()
	fixture.engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid query, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestNotificationReadRoutePersistsReadState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 6, 9, 8, 0, 0, 0, time.UTC)
	repo := &routeTestNotificationRepository{
		items: []notificationstore.Notification{routeTestNotification(100, 42, "Unread item", now, nil)},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	fixture := newNotificationRouteTestContext()
	registerNotificationRoutes(fixture.ctx, service, notificationGuards{view: routeTestAuth(42), read: routeTestAuth(42)})

	request := httptest.NewRequest(http.MethodPost, "/api/notifications/100/read", nil)
	recorder := httptest.NewRecorder()
	fixture.engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if repo.items[0].Delivery.ReadAt == nil {
		t.Fatal("expected route test repository to persist read state")
	}
}

func TestNotificationRoutesRejectWrongUserDeliveryMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 6, 9, 8, 0, 0, 0, time.UTC)
	repo := &routeTestNotificationRepository{
		items: []notificationstore.Notification{routeTestNotification(100, 7, "Wrong user", now, nil)},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	fixture := newNotificationRouteTestContext()
	registerNotificationRoutes(fixture.ctx, service, notificationGuards{view: routeTestAuth(42), read: routeTestAuth(42)})

	request := httptest.NewRequest(http.MethodPost, "/api/notifications/100/read", nil)
	recorder := httptest.NewRecorder()
	fixture.engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong-user delivery, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestNotificationUnreadCountUsesCanonicalCountField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 6, 9, 8, 0, 0, 0, time.UTC)
	repo := &routeTestNotificationRepository{
		items: []notificationstore.Notification{
			routeTestNotification(100, 42, "Unread item", now, nil),
			routeTestNotification(101, 42, "Read item", now, &now),
			routeTestNotification(102, 7, "Wrong user", now, nil),
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	fixture := newNotificationRouteTestContext()
	registerNotificationRoutes(fixture.ctx, service, notificationGuards{view: routeTestAuth(42), read: routeTestAuth(42)})

	request := httptest.NewRequest(http.MethodGet, "/api/notifications/unread-count", nil)
	recorder := httptest.NewRecorder()
	fixture.engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data["count"] != float64(1) {
		t.Fatalf("expected canonical count=1, got %#v", response.Data)
	}
	if _, ok := response.Data["unread_count"]; ok {
		t.Fatalf("unexpected legacy unread_count field in %#v", response.Data)
	}
}

type routeTestNotificationRepository struct {
	items     []notificationstore.Notification
	listQuery notificationstore.ListQuery
}

func (r *routeTestNotificationRepository) CreateEvent(context.Context, notificationstore.CreateEventInput) (notificationstore.Event, bool, error) {
	return notificationstore.Event{}, false, nil
}

func (r *routeTestNotificationRepository) CreateDeliveries(context.Context, []notificationstore.CreateDeliveryInput) ([]notificationstore.Delivery, error) {
	return nil, nil
}

func (r *routeTestNotificationRepository) List(_ context.Context, query notificationstore.ListQuery) (notificationstore.ListResult, error) {
	r.listQuery = query
	items := make([]notificationstore.Notification, 0)
	for _, item := range r.items {
		if item.Delivery.RecipientUserID != query.RecipientUserID {
			continue
		}
		if query.Status == "unread" && item.Delivery.ReadAt != nil {
			continue
		}
		items = append(items, item)
	}
	return notificationstore.ListResult{Items: items, Total: len(items)}, nil
}

func (r *routeTestNotificationRepository) Get(_ context.Context, recipientUserID uint64, deliveryID uint64) (notificationstore.Notification, error) {
	for _, item := range r.items {
		if item.Delivery.ID == deliveryID && item.Delivery.RecipientUserID == recipientUserID {
			return item, nil
		}
	}
	return notificationstore.Notification{}, notificationstore.ErrDeliveryNotFound
}

func (r *routeTestNotificationRepository) UnreadCount(_ context.Context, recipientUserID uint64) (int, error) {
	count := 0
	for _, item := range r.items {
		if item.Delivery.RecipientUserID == recipientUserID && item.Delivery.ReadAt == nil {
			count++
		}
	}
	return count, nil
}

func (r *routeTestNotificationRepository) MarkRead(_ context.Context, recipientUserID uint64, deliveryID uint64, readAt time.Time) (notificationstore.Delivery, error) {
	for index := range r.items {
		if r.items[index].Delivery.ID == deliveryID && r.items[index].Delivery.RecipientUserID == recipientUserID {
			r.items[index].Delivery.ReadAt = &readAt
			return r.items[index].Delivery, nil
		}
	}
	return notificationstore.Delivery{}, notificationstore.ErrDeliveryNotFound
}

func (r *routeTestNotificationRepository) MarkAllRead(context.Context, uint64, time.Time) (int, error) {
	return 0, nil
}

func (r *routeTestNotificationRepository) MarkAllReadMatching(context.Context, notificationstore.ListQuery, time.Time) (int, error) {
	return 0, nil
}

func (r *routeTestNotificationRepository) DeleteDelivery(context.Context, uint64, uint64, time.Time) error {
	return nil
}

type notificationRouteTestContext struct {
	ctx    *module.Context
	engine *gin.Engine
}

func newNotificationRouteTestContext() notificationRouteTestContext {
	localizer, err := i18n.New(config.I18nConfig{
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
	if err != nil {
		panic(err)
	}
	engine := gin.New()
	return notificationRouteTestContext{
		engine: engine,
		ctx: &module.Context{
			Config:             &config.Config{},
			Router:             engine.Group("/api"),
			I18n:               localizer,
			Services:           container.New(),
			MenuRegistry:       menu.NewRegistry(),
			PermissionRegistry: permission.NewRegistry(),
			CronRegistry:       cronx.NewRegistry(),
			ConfigRegistry:     configregistry.NewRegistry(),
			DashboardRegistry:  dashboard.NewRegistry(),
		},
	}
}

func routeTestAuth(userID uint64) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Request = ctx.Request.WithContext(moduleapi.WithRequestAuthContext(ctx.Request.Context(), moduleapi.RequestAuthContext{
			User: &moduleapi.CurrentUser{ID: userID, Username: "alice"},
		}))
		ctx.Next()
	}
}

func routeTestNotification(deliveryID uint64, recipientUserID uint64, title string, now time.Time, readAt *time.Time) notificationstore.Notification {
	return notificationstore.Notification{
		Event: notificationstore.Event{
			ID:                deliveryID + 1000,
			Title:             title,
			Message:           "message",
			Severity:          notificationcontract.SeverityWarning.String(),
			Category:          notificationcontract.CategorySecurity.String(),
			SourceModule:      "audit",
			EventType:         "permission_denied",
			NavigationKind:    notificationcontract.NavigationAuditLog.String(),
			NavigationPayload: json.RawMessage(`{"audit_log_id":1}`),
			OccurredAt:        now,
			CreatedAt:         now,
		},
		Delivery: notificationstore.Delivery{
			ID:              deliveryID,
			EventID:         deliveryID + 1000,
			RecipientUserID: recipientUserID,
			TargetType:      notificationcontract.TargetUser.String(),
			TargetRef:       "42",
			ReadAt:          readAt,
			CreatedAt:       now,
		},
	}
}
