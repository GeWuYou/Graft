package announcement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/container"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/cronx"
	"graft/server/internal/dashboard"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	"graft/server/internal/testassert"
	announcementcontract "graft/server/modules/announcement/contract"
	announcementlocales "graft/server/modules/announcement/locales"
	announcementstore "graft/server/modules/announcement/store"
)

func TestModuleRegistersAnnouncementMetadata(t *testing.T) {
	service, err := NewService(testAnnouncementRepository{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := newAnnouncementTestContext(nil)
	moduleInstance := NewModule(service)
	if err := moduleInstance.Register(ctx); err != nil {
		t.Fatalf("register announcement module: %v", err)
	}

	assertAnnouncementPermissionsRegistered(t, ctx.PermissionRegistry)
	assertAnnouncementMenuRegistered(t, ctx.MenuRegistry)
	assertAnnouncementMessageRegistered(t, ctx.I18n, i18n.LocaleZHCN, "公告管理")
	assertAnnouncementMessageRegistered(t, ctx.I18n, i18n.LocaleENUS, "Announcements")
	assertRegisteredAnnouncementErrorMessage(
		t,
		ctx.I18n,
		i18n.LocaleZHCN,
		announcementcontract.AnnouncementPublishedDeleteForbidden.String(),
		"已发布公告需先归档后删除",
	)
	assertRegisteredAnnouncementErrorMessage(
		t,
		ctx.I18n,
		i18n.LocaleENUS,
		announcementcontract.AnnouncementPublishedDeleteForbidden.String(),
		"Archive the published announcement before deleting it",
	)
}

func TestNewModuleSpecDeclaresMigrationAndDependencies(t *testing.T) {
	spec := NewModuleSpec()
	if spec.ID != moduleID {
		t.Fatalf("unexpected module id %q", spec.ID)
	}
	if len(spec.Dependencies) != 2 || spec.Dependencies[0] != "user" || spec.Dependencies[1] != "rbac" {
		t.Fatalf("unexpected dependencies %#v", spec.Dependencies)
	}
	if len(spec.MigrationPath) != 1 || spec.MigrationPath[0] != "modules/announcement/migrations" {
		t.Fatalf("unexpected migration paths %#v", spec.MigrationPath)
	}
}

func TestAnnouncementEmbeddedLocaleResources(t *testing.T) {
	resources, err := announcementlocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("embedded locale resources: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 locale resources, got %d", len(resources))
	}
	if resources[0].Namespace != i18n.Namespace("announcement") || resources[0].Locale != i18n.LocaleENUS {
		t.Fatalf("unexpected first locale resource %#v", resources[0])
	}
	if resources[1].Namespace != i18n.Namespace("announcement") || resources[1].Locale != i18n.LocaleZHCN {
		t.Fatalf("unexpected second locale resource %#v", resources[1])
	}
}

func TestAnnouncementContractValidators(t *testing.T) {
	if !announcementcontract.ValidAnnouncementStatus(announcementcontract.AnnouncementStatusPublished) {
		t.Fatal("expected published status to be valid")
	}
	if announcementcontract.ValidAnnouncementStatus(announcementcontract.AnnouncementStatus("visible")) {
		t.Fatal("unexpected ad-hoc announcement status accepted")
	}
	if !announcementcontract.ValidAnnouncementLevel(announcementcontract.AnnouncementLevelWarning) {
		t.Fatal("expected warning level to be valid")
	}
	if announcementcontract.ValidAnnouncementLevel(announcementcontract.AnnouncementLevel("critical")) {
		t.Fatal("unexpected ad-hoc announcement level accepted")
	}
	if !announcementcontract.ValidAnnouncementDeliveryMode(announcementcontract.AnnouncementDeliveryModePopup) {
		t.Fatal("expected popup delivery mode to be valid")
	}
	if announcementcontract.ValidAnnouncementDeliveryMode(announcementcontract.AnnouncementDeliveryMode("toast")) {
		t.Fatal("unexpected ad-hoc announcement delivery mode accepted")
	}
}

func TestAnnouncementUserRoutesReturnCurrentUserAnnouncements(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	publishAt := time.Now().UTC().Add(-time.Hour)
	created, err := service.Create(ctx, announcementstore.CreateInput{
		Title:        "Visible",
		Content:      "Visible content",
		Level:        announcementcontract.AnnouncementLevelInfo.String(),
		DeliveryMode: announcementcontract.AnnouncementDeliveryModePopup.String(),
		PublishAt:    &publishAt,
		ActorID:      &actorID,
	})
	if err != nil {
		t.Fatalf("create announcement: %v", err)
	}
	if _, err := service.Publish(ctx, created.ID, nil, &actorID); err != nil {
		t.Fatalf("publish announcement: %v", err)
	}

	engine := gin.New()
	moduleCtx := newAnnouncementTestContext(engine)
	if err := registerAnnouncementRoutes(moduleCtx, service, announcementGuards{
		authenticated: announcementRouteTestAuth(42),
		read:          announcementRouteTestAuth(42),
		create:        announcementRouteTestAuth(42),
		update:        announcementRouteTestAuth(42),
		publish:       announcementRouteTestAuth(42),
		delete:        announcementRouteTestAuth(42),
	}); err != nil {
		t.Fatalf("register announcement routes: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/my/announcements", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 user route response, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var response httpx.SuccessResponse[map[string]any]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	items, ok := response.Data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one visible announcement, got %#v", response.Data["items"])
	}
	item, ok := items[0].(map[string]any)
	if !ok || item["delivery_mode"] != announcementcontract.AnnouncementDeliveryModePopup.String() {
		t.Fatalf("expected user route to return popup delivery mode, got %#v", items[0])
	}
}

func TestAnnouncementUserRoutesReturnTypedDTO(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	publishAt := time.Now().UTC().Add(-time.Hour)
	created, err := service.Create(ctx, announcementstore.CreateInput{
		Title:        "Typed",
		Content:      "Typed content",
		Level:        announcementcontract.AnnouncementLevelInfo.String(),
		DeliveryMode: announcementcontract.AnnouncementDeliveryModePopup.String(),
		PublishAt:    &publishAt,
		ActorID:      &actorID,
	})
	if err != nil {
		t.Fatalf("create announcement: %v", err)
	}
	if _, err := service.Publish(ctx, created.ID, nil, &actorID); err != nil {
		t.Fatalf("publish announcement: %v", err)
	}

	engine := gin.New()
	moduleCtx := newAnnouncementTestContext(engine)
	if err := registerAnnouncementRoutes(moduleCtx, service, announcementGuards{
		authenticated: announcementRouteTestAuth(42),
		read:          announcementRouteTestAuth(42),
		create:        announcementRouteTestAuth(42),
		update:        announcementRouteTestAuth(42),
		publish:       announcementRouteTestAuth(42),
		delete:        announcementRouteTestAuth(42),
	}); err != nil {
		t.Fatalf("register announcement routes: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/my/announcements", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 user route response, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var response httpx.SuccessResponse[myAnnouncementListResponse]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode typed response: %v", err)
	}
	if len(response.Data.Items) != 1 {
		t.Fatalf("expected one typed announcement, got %#v", response.Data.Items)
	}
	item := response.Data.Items[0]
	if item.DeliveryMode != "popup" || !item.Unread || item.ReadAt != nil {
		t.Fatalf("unexpected typed announcement response: %#v", item)
	}
}

func TestAnnouncementManagementServiceLifecycle(t *testing.T) {
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	publishAt := time.Date(2026, 6, 12, 8, 0, 0, 0, time.FixedZone("cst", 8*60*60))
	expireAt := publishAt.Add(2 * time.Hour)

	created, err := service.Create(ctx, announcementstore.CreateInput{
		Title:     " Maintenance ",
		Content:   "Window",
		Level:     announcementcontract.AnnouncementLevelWarning.String(),
		Pinned:    true,
		PublishAt: &publishAt,
		ExpireAt:  &expireAt,
		ActorID:   &actorID,
	})
	if err != nil {
		t.Fatalf("create announcement: %v", err)
	}
	if created.Status != announcementcontract.AnnouncementStatusDraft.String() {
		t.Fatalf("expected draft create status, got %q", created.Status)
	}

	beforePublish := time.Now().UTC()
	published, err := service.Publish(ctx, created.ID, &publishAt, &actorID)
	if err != nil {
		t.Fatalf("publish announcement: %v", err)
	}
	if published.Status != announcementcontract.AnnouncementStatusPublished.String() {
		t.Fatalf("expected published status, got %q", published.Status)
	}
	if published.PublishAt == nil || !published.PublishAt.Equal(publishAt.UTC()) {
		t.Fatalf("expected publish_at to keep UTC input, got %#v", published.PublishAt)
	}
	assertAnnouncementPublishedAtAfter(t, published, beforePublish)
	assertAnnouncementPublishedBy(t, published, actorID)
	assertAnnouncementArchivedAtCleared(t, published)
	if err := service.Delete(ctx, created.ID, actorID); !errors.Is(err, errAnnouncementPublishedDelete) {
		t.Fatalf("expected published delete guard, got %v", err)
	}

	archived, err := service.Archive(ctx, created.ID, &actorID)
	if err != nil {
		t.Fatalf("archive announcement: %v", err)
	}
	if archived.Status != announcementcontract.AnnouncementStatusArchived.String() {
		t.Fatalf("expected archived status, got %q", archived.Status)
	}
	assertAnnouncementArchivedAtSet(t, archived)
	if _, err := service.Update(ctx, created.ID, announcementstore.UpdateInput{
		Title:   "New",
		Content: "New",
		Level:   announcementcontract.AnnouncementLevelInfo.String(),
	}); !errors.Is(err, errAnnouncementInvalidTransition) {
		t.Fatalf("expected archived update guard, got %v", err)
	}
	if err := service.Delete(ctx, created.ID, actorID); err != nil {
		t.Fatalf("delete archived announcement: %v", err)
	}
	if _, err := service.GetAdmin(ctx, created.ID); !errors.Is(err, errAnnouncementNotFound) {
		t.Fatalf("expected deleted announcement not found, got %v", err)
	}
}

func TestAnnouncementRepublishArchivedClearsEffectiveTimeAndPreservesReadState(t *testing.T) {
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	userID := uint64(42)
	publishAt := time.Now().UTC().Add(-2 * time.Hour)
	created := createAnnouncementForUserTest(t, service, "Republish", publishAt, nil, actorID)
	if _, err := service.MarkRead(ctx, userID, created.ID); err != nil {
		t.Fatalf("mark read before archive: %v", err)
	}
	archived, err := service.Archive(ctx, created.ID, &actorID)
	if err != nil {
		t.Fatalf("archive announcement: %v", err)
	}
	if archived.ArchivedAt == nil {
		t.Fatal("expected archived_at before republish")
	}

	beforeRepublish := time.Now().UTC()
	republished, err := service.Publish(ctx, created.ID, nil, &actorID)
	if err != nil {
		t.Fatalf("republish archived announcement: %v", err)
	}
	if republished.Status != announcementcontract.AnnouncementStatusPublished.String() {
		t.Fatalf("expected republished status, got %q", republished.Status)
	}
	assertAnnouncementArchivedAtCleared(t, republished)
	assertAnnouncementPublishedBy(t, republished, actorID)
	if republished.PublishAt != nil {
		t.Fatalf("expected republish without publish_at to store null effective time, got %#v", republished.PublishAt)
	}
	assertAnnouncementPublishedAtAfter(t, republished, beforeRepublish)

	result, err := service.ListCurrentUser(ctx, UserListQuery{UserID: userID, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list current user after republish: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ReadAt == nil {
		t.Fatalf("expected republished announcement to preserve read state, got total=%d items=%#v", result.Total, result.Items)
	}
}

func TestAnnouncementPublishOverridesStoredPublishAt(t *testing.T) {
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	publishAt := time.Date(2026, 6, 12, 8, 0, 0, 0, time.FixedZone("cst", 8*60*60))
	created, err := service.Create(ctx, announcementstore.CreateInput{
		Title:     " Maintenance ",
		Content:   "Window",
		Level:     announcementcontract.AnnouncementLevelWarning.String(),
		PublishAt: &publishAt,
		ActorID:   &actorID,
	})
	if err != nil {
		t.Fatalf("create announcement: %v", err)
	}
	overridePublishAt := publishAt.Add(time.Hour)
	overridden, err := service.Publish(ctx, created.ID, &overridePublishAt, &actorID)
	if err != nil {
		t.Fatalf("publish announcement with override: %v", err)
	}
	if overridden.PublishAt == nil || !overridden.PublishAt.Equal(overridePublishAt.UTC()) {
		t.Fatalf("expected explicit publish_at override, got %#v", overridden.PublishAt)
	}
	assertAnnouncementPublishedAtSet(t, overridden)
}

func TestAnnouncementPublishWithoutEffectiveTimeIsImmediatelyVisible(t *testing.T) {
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	created, err := service.Create(ctx, announcementstore.CreateInput{
		Title:   "Immediate",
		Content: "Immediate",
		Level:   announcementcontract.AnnouncementLevelInfo.String(),
		ActorID: &actorID,
	})
	if err != nil {
		t.Fatalf("create announcement: %v", err)
	}
	published, err := service.Publish(ctx, created.ID, nil, &actorID)
	if err != nil {
		t.Fatalf("publish announcement: %v", err)
	}
	if published.PublishAt != nil {
		t.Fatalf("expected publish_at to stay null for immediate visibility, got %#v", published.PublishAt)
	}
	assertAnnouncementPublishedAtSet(t, published)
	result, err := service.ListCurrentUser(ctx, UserListQuery{UserID: 42, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list current-user announcements: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Announcement.ID != published.ID {
		t.Fatalf("expected immediate announcement to be visible, got total=%d items=%#v", result.Total, result.Items)
	}
}

func TestAnnouncementUpdatePublishedEffectiveTimeDoesNotChangePublishedAt(t *testing.T) {
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	publishAt := time.Now().UTC().Add(-time.Hour)
	published := createAnnouncementForUserTest(t, service, "Edit effective", publishAt, nil, actorID)
	if published.PublishedAt == nil {
		t.Fatal("expected published_at after publish")
	}
	originalPublishedAt := *published.PublishedAt
	nextPublishAt := publishAt.Add(time.Hour)
	updated, err := service.Update(ctx, published.ID, announcementstore.UpdateInput{
		Title:        published.Title,
		Content:      published.Content,
		Level:        published.Level,
		DeliveryMode: published.DeliveryMode,
		Pinned:       published.Pinned,
		PublishAt:    &nextPublishAt,
		ActorID:      &actorID,
	})
	if err != nil {
		t.Fatalf("update published announcement: %v", err)
	}
	if updated.PublishedAt == nil || !updated.PublishedAt.Equal(originalPublishedAt) {
		t.Fatalf("expected edit to keep published_at %s, got %#v", originalPublishedAt, updated.PublishedAt)
	}
	if updated.PublishAt == nil || !updated.PublishAt.Equal(nextPublishAt.UTC()) {
		t.Fatalf("expected edit to update effective publish_at, got %#v", updated.PublishAt)
	}
}

func TestAnnouncementManagementServiceDeliveryMode(t *testing.T) {
	service, err := NewService(newMemoryAnnouncementRepository())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	created, err := service.Create(ctx, announcementstore.CreateInput{
		Title:   "Delivery",
		Content: "Delivery",
		Level:   announcementcontract.AnnouncementLevelInfo.String(),
		ActorID: &actorID,
	})
	if err != nil {
		t.Fatalf("create announcement: %v", err)
	}
	if created.DeliveryMode != announcementcontract.AnnouncementDeliveryModeSilent.String() {
		t.Fatalf("expected empty delivery mode to default to silent, got %q", created.DeliveryMode)
	}
	updated, err := service.Update(ctx, created.ID, announcementstore.UpdateInput{
		Title:        "Delivery",
		Content:      "Delivery",
		Level:        announcementcontract.AnnouncementLevelInfo.String(),
		DeliveryMode: announcementcontract.AnnouncementDeliveryModePopup.String(),
		ActorID:      &actorID,
	})
	if err != nil {
		t.Fatalf("update announcement delivery mode: %v", err)
	}
	if updated.DeliveryMode != announcementcontract.AnnouncementDeliveryModePopup.String() {
		t.Fatalf("expected popup delivery mode after update, got %q", updated.DeliveryMode)
	}
}

func TestAnnouncementManagementServiceDeleteDraftAndInvalidPublishWindow(t *testing.T) {
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	publishAt := time.Date(2026, 6, 12, 8, 0, 0, 0, time.UTC)
	expireAt := publishAt

	if _, err := service.Create(ctx, announcementstore.CreateInput{
		Title:     "Bad window",
		Content:   "Bad window",
		Level:     announcementcontract.AnnouncementLevelInfo.String(),
		PublishAt: &publishAt,
		ExpireAt:  &expireAt,
		ActorID:   &actorID,
	}); !errors.Is(err, errAnnouncementInvalidInput) {
		t.Fatalf("expected invalid expire_at guard, got %v", err)
	}
	if _, err := service.Create(ctx, announcementstore.CreateInput{
		Title:        "Bad delivery",
		Content:      "Bad delivery",
		Level:        announcementcontract.AnnouncementLevelInfo.String(),
		DeliveryMode: "toast",
		ActorID:      &actorID,
	}); !errors.Is(err, errAnnouncementInvalidInput) {
		t.Fatalf("expected invalid delivery mode guard, got %v", err)
	}

	created, err := service.Create(ctx, announcementstore.CreateInput{
		Title:   "Draft",
		Content: "Draft",
		Level:   announcementcontract.AnnouncementLevelInfo.String(),
		ActorID: &actorID,
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if err := service.Delete(ctx, created.ID, actorID); err != nil {
		t.Fatalf("delete draft: %v", err)
	}
}

func TestAnnouncementManagementServiceListFiltersAndSort(t *testing.T) {
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	create := func(title string, level announcementcontract.AnnouncementLevel, pinned bool) announcementstore.Announcement {
		item, createErr := service.Create(ctx, announcementstore.CreateInput{
			Title:   title,
			Content: "content " + title,
			Level:   level.String(),
			Pinned:  pinned,
		})
		if createErr != nil {
			t.Fatalf("create %s: %v", title, createErr)
		}
		return item
	}
	create("Alpha maintenance", announcementcontract.AnnouncementLevelWarning, true)
	create("Beta release", announcementcontract.AnnouncementLevelInfo, false)
	create("Gamma maintenance", announcementcontract.AnnouncementLevelWarning, false)

	pinned := true
	result, err := service.ListAdmin(ctx, AdminListQuery{
		Level:    announcementcontract.AnnouncementLevelWarning.String(),
		Pinned:   &pinned,
		Keyword:  "maintenance",
		Page:     1,
		PageSize: 10,
		Sort:     "updated_desc",
	})
	if err != nil {
		t.Fatalf("list filtered announcements: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Title != "Alpha maintenance" {
		t.Fatalf("unexpected filtered result: total=%d items=%#v", result.Total, result.Items)
	}
}

func TestAnnouncementUserListExcludesDraftFutureExpiredAndArchived(t *testing.T) {
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	now := time.Now().UTC()

	visible := createAnnouncementForUserTest(t, service, "Visible", now.Add(-time.Hour), nil, actorID)
	createDraftForUserTest(t, service, "Draft", now.Add(-time.Hour), nil, actorID)
	createAnnouncementForUserTest(t, service, "Future", now.Add(time.Hour), nil, actorID)
	expiredAt := now.Add(-time.Minute)
	createAnnouncementForUserTest(t, service, "Expired", now.Add(-time.Hour), &expiredAt, actorID)
	archived := createAnnouncementForUserTest(t, service, "Archived", now.Add(-time.Hour), nil, actorID)
	if _, err := service.Archive(ctx, archived.ID, &actorID); err != nil {
		t.Fatalf("archive test announcement: %v", err)
	}

	result, err := service.ListCurrentUser(ctx, UserListQuery{UserID: 42, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list current-user announcements: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Announcement.ID != visible.ID {
		t.Fatalf("expected only visible announcement %d, got total=%d items=%#v", visible.ID, result.Total, result.Items)
	}
}

func TestAnnouncementReadStateIsIsolatedByUser(t *testing.T) {
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	published := createAnnouncementForUserTest(t, service, "Visible", time.Now().UTC().Add(-time.Hour), nil, actorID)

	if _, err := service.MarkRead(ctx, 42, published.ID); err != nil {
		t.Fatalf("mark read user 42: %v", err)
	}
	user42, err := service.ListCurrentUser(ctx, UserListQuery{UserID: 42, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list user 42: %v", err)
	}
	user7, err := service.ListCurrentUser(ctx, UserListQuery{UserID: 7, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list user 7: %v", err)
	}
	if user42.Items[0].ReadAt == nil {
		t.Fatal("expected user 42 read state to be present")
	}
	if user7.Items[0].ReadAt != nil {
		t.Fatalf("expected user 7 read state to stay unread, got %#v", user7.Items[0].ReadAt)
	}
}

func TestAnnouncementReadAllOnlyAffectsCurrentUserAndVisibleAnnouncements(t *testing.T) {
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	now := time.Now().UTC()
	visibleA := createAnnouncementForUserTest(t, service, "Visible A", now.Add(-time.Hour), nil, actorID)
	visibleB := createAnnouncementForUserTest(t, service, "Visible B", now.Add(-2*time.Hour), nil, actorID)
	createDraftForUserTest(t, service, "Draft", now.Add(-time.Hour), nil, actorID)
	future := createAnnouncementForUserTest(t, service, "Future", now.Add(time.Hour), nil, actorID)

	updated, err := service.MarkAllRead(ctx, 42)
	if err != nil {
		t.Fatalf("mark all read: %v", err)
	}
	if updated != 2 {
		t.Fatalf("expected two visible announcements marked read, got %d", updated)
	}
	user42, err := service.ListCurrentUser(ctx, UserListQuery{UserID: 42, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list user 42: %v", err)
	}
	for _, item := range user42.Items {
		if item.Announcement.ID == visibleA.ID || item.Announcement.ID == visibleB.ID {
			if item.ReadAt == nil {
				t.Fatalf("expected visible announcement %d to be read", item.Announcement.ID)
			}
		}
		if item.Announcement.ID == future.ID {
			t.Fatal("future announcement should not be visible after read-all")
		}
	}
	user7Count, err := service.UnreadCount(ctx, 7)
	if err != nil {
		t.Fatalf("unread count user 7: %v", err)
	}
	if user7Count != 2 {
		t.Fatalf("expected user 7 unread count to stay isolated at 2, got %d", user7Count)
	}
}

func TestAnnouncementUnreadCountRespectsVisibilityAndUserIsolation(t *testing.T) {
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	actorID := uint64(7)
	now := time.Now().UTC()
	visibleA := createAnnouncementForUserTest(t, service, "Visible A", now.Add(-time.Hour), nil, actorID)
	createAnnouncementForUserTest(t, service, "Visible B", now.Add(-2*time.Hour), nil, actorID)
	createDraftForUserTest(t, service, "Draft", now.Add(-time.Hour), nil, actorID)
	expiredAt := now.Add(-time.Minute)
	createAnnouncementForUserTest(t, service, "Expired", now.Add(-time.Hour), &expiredAt, actorID)

	if _, err := service.MarkRead(ctx, 42, visibleA.ID); err != nil {
		t.Fatalf("mark read user 42: %v", err)
	}
	user42Count, err := service.UnreadCount(ctx, 42)
	if err != nil {
		t.Fatalf("unread count user 42: %v", err)
	}
	if user42Count != 1 {
		t.Fatalf("expected user 42 one unread visible announcement, got %d", user42Count)
	}
	user7Count, err := service.UnreadCount(ctx, 7)
	if err != nil {
		t.Fatalf("unread count user 7: %v", err)
	}
	if user7Count != 2 {
		t.Fatalf("expected user 7 two unread visible announcements, got %d", user7Count)
	}
}

func TestAnnouncementManagementRoutePermissionDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, err := NewService(newMemoryAnnouncementRepository())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	engine := gin.New()
	ctx := newAnnouncementTestContext(engine)
	if err := registerAnnouncementRoutes(ctx, service, announcementGuards{
		read: func(ginCtx *gin.Context) {
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusForbidden, messagecontract.AuthForbidden.String(), nil)
		},
	}); err != nil {
		t.Fatalf("register announcement routes: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/announcements", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403 permission denial, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAnnouncementRoutesRejectInvalidQueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, err := NewService(newMemoryAnnouncementRepository())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	engine := gin.New()
	ctx := newAnnouncementTestContext(engine)
	if err := registerAnnouncementRoutes(ctx, service, announcementGuards{
		authenticated: announcementRouteTestAuth(42),
		read:          announcementRouteTestAuth(42),
		create:        announcementRouteTestAuth(42),
		update:        announcementRouteTestAuth(42),
		publish:       announcementRouteTestAuth(42),
		delete:        announcementRouteTestAuth(42),
	}); err != nil {
		t.Fatalf("register announcement routes: %v", err)
	}

	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "admin pinned", path: "/api/announcements?pinned=maybe"},
		{name: "admin page", path: "/api/announcements?page=zero"},
		{name: "admin page size", path: "/api/announcements?page_size=101"},
		{name: "user unread", path: "/api/my/announcements?unread_only=maybe"},
		{name: "user page", path: "/api/my/announcements?page=0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tc.path, nil)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for %s, got %d body=%s", tc.path, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestAnnouncementDeletePublishedRouteReturnsDomainConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repository := newMemoryAnnouncementRepository()
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	actorID := uint64(7)
	published := createAnnouncementForUserTest(t, service, "Published", time.Now().UTC().Add(-time.Hour), nil, actorID)
	engine := gin.New()
	ctx := newAnnouncementTestContext(engine)
	if err := registerAnnouncementRoutes(ctx, service, announcementGuards{
		authenticated: announcementRouteTestAuth(42),
		read:          announcementRouteTestAuth(42),
		create:        announcementRouteTestAuth(42),
		update:        announcementRouteTestAuth(42),
		publish:       announcementRouteTestAuth(42),
		delete:        announcementRouteTestAuth(42),
	}); err != nil {
		t.Fatalf("register announcement routes: %v", err)
	}

	request := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/announcements/%d", published.ID), nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409 published delete conflict, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	payload := testassert.DecodeErrorResponse(t, recorder)
	testassert.AssertErrorPayload(
		t,
		payload,
		announcementcontract.AnnouncementPublishedDeleteForbidden.String(),
		"ANNOUNCEMENT_PUBLISHED_DELETE_FORBIDDEN",
		"zh-CN",
	)
}

func TestAnnouncementMapperRejectsInt64Overflow(t *testing.T) {
	if _, err := toAnnouncementItem(announcementstore.Announcement{ID: uint64(1) << 63}); !errors.Is(err, errAnnouncementIDOverflow) {
		t.Fatalf("expected announcement id overflow, got %v", err)
	}
	overflowActorID := uint64(1) << 63
	if _, err := toAnnouncementItem(announcementstore.Announcement{ID: 1, CreatedBy: &overflowActorID}); !errors.Is(err, errAnnouncementIDOverflow) {
		t.Fatalf("expected actor id overflow, got %v", err)
	}
	if _, err := toMyAnnouncementItem(announcementstore.UserAnnouncement{Announcement: announcementstore.Announcement{ID: uint64(1) << 63}}); !errors.Is(err, errAnnouncementIDOverflow) {
		t.Fatalf("expected current-user announcement id overflow, got %v", err)
	}
}

func assertAnnouncementPermissionsRegistered(t *testing.T, registry *permission.Registry) {
	t.Helper()
	registered := make(map[string]struct{}, len(registry.Items()))
	for _, item := range registry.Items() {
		registered[item.Code] = struct{}{}
	}
	for _, code := range []string{
		announcementcontract.AnnouncementReadPermission.String(),
		announcementcontract.AnnouncementCreatePermission.String(),
		announcementcontract.AnnouncementUpdatePermission.String(),
		announcementcontract.AnnouncementPublishPermission.String(),
		announcementcontract.AnnouncementDeletePermission.String(),
	} {
		if _, ok := registered[code]; !ok {
			t.Fatalf("expected announcement permission %s to be registered", code)
		}
	}
}

func assertAnnouncementMenuRegistered(t *testing.T, registry *menu.Registry) {
	t.Helper()
	for _, item := range registry.Items() {
		if item.Code == "announcement.list" &&
			item.Path == announcementcontract.AnnouncementMenuPath &&
			item.Permission == announcementcontract.AnnouncementReadPermission.String() &&
			item.TitleKey == announcementcontract.AnnouncementMenuTitle.String() {
			return
		}
	}
	t.Fatalf("expected announcement management menu, got %#v", registry.Items())
}

func assertAnnouncementMessageRegistered(t *testing.T, localizer *i18n.Service, locale i18n.LocaleTag, expected string) {
	t.Helper()
	matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(announcementcontract.AnnouncementMenuTitle.String()))
	if len(matches) != 1 || matches[0].Text != expected {
		t.Fatalf("expected announcement menu title %q for %s, got %#v", expected, locale, matches)
	}
}

func assertRegisteredAnnouncementErrorMessage(
	t *testing.T,
	localizer *i18n.Service,
	locale i18n.LocaleTag,
	key string,
	expected string,
) {
	t.Helper()

	matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key))
	if len(matches) != 1 || matches[0].Text != expected {
		t.Fatalf("expected announcement message %q for %s %q, got %#v", expected, locale, key, matches)
	}
}

func newAnnouncementTestContext(engine *gin.Engine) *module.Context {
	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:  "zh-CN",
		FallbackLocale: "zh-CN",
		SupportedLocales: []string{
			"zh-CN",
			"en-US",
		},
	})
	resources, err := announcementlocales.EmbeddedLocaleResources()
	if err != nil {
		panic(fmt.Sprintf("load announcement locale resources: %v", err))
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		panic(fmt.Sprintf("register announcement locale resources: %v", err))
	}
	var router gin.IRouter
	if engine != nil {
		router = engine.Group("/api")
	}
	return &module.Context{
		Config:             &config.Config{},
		Router:             router,
		I18n:               localizer,
		Services:           container.New(),
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
		ConfigRegistry:     configregistry.NewRegistry(),
		DashboardRegistry:  dashboard.NewRegistry(),
	}
}

func announcementRouteTestAuth(userID uint64) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Request = ctx.Request.WithContext(moduleapi.WithRequestAuthContext(ctx.Request.Context(), moduleapi.RequestAuthContext{
			User: &moduleapi.CurrentUser{ID: userID, Username: "alice"},
		}))
		ctx.Next()
	}
}

func createAnnouncementForUserTest(
	t *testing.T,
	service *Service,
	title string,
	publishAt time.Time,
	expireAt *time.Time,
	actorID uint64,
) announcementstore.Announcement {
	t.Helper()
	ctx := context.Background()
	item, err := service.Create(ctx, announcementstore.CreateInput{
		Title:     title,
		Content:   title + " content",
		Level:     announcementcontract.AnnouncementLevelInfo.String(),
		PublishAt: &publishAt,
		ExpireAt:  expireAt,
		ActorID:   &actorID,
	})
	if err != nil {
		t.Fatalf("create %s: %v", title, err)
	}
	published, err := service.Publish(ctx, item.ID, &publishAt, &actorID)
	if err != nil {
		t.Fatalf("publish %s: %v", title, err)
	}
	return published
}

func createDraftForUserTest(
	t *testing.T,
	service *Service,
	title string,
	publishAt time.Time,
	expireAt *time.Time,
	actorID uint64,
) announcementstore.Announcement {
	t.Helper()
	item, err := service.Create(context.Background(), announcementstore.CreateInput{
		Title:     title,
		Content:   title + " content",
		Level:     announcementcontract.AnnouncementLevelInfo.String(),
		PublishAt: &publishAt,
		ExpireAt:  expireAt,
		ActorID:   &actorID,
	})
	if err != nil {
		t.Fatalf("create draft %s: %v", title, err)
	}
	return item
}

func assertAnnouncementPublishedBy(t *testing.T, item announcementstore.Announcement, actorID uint64) {
	t.Helper()
	if item.PublishedBy == nil || *item.PublishedBy != actorID {
		t.Fatalf("expected published_by actor %d, got %#v", actorID, item.PublishedBy)
	}
}

func assertAnnouncementPublishedAtSet(t *testing.T, item announcementstore.Announcement) {
	t.Helper()
	if item.PublishedAt == nil {
		t.Fatal("expected published_at to be set")
	}
}

func assertAnnouncementPublishedAtAfter(t *testing.T, item announcementstore.Announcement, before time.Time) {
	t.Helper()
	if item.PublishedAt == nil || item.PublishedAt.Before(before) {
		t.Fatalf("expected published_at after %s, got %#v", before, item.PublishedAt)
	}
}

func assertAnnouncementArchivedAtSet(t *testing.T, item announcementstore.Announcement) {
	t.Helper()
	if item.ArchivedAt == nil {
		t.Fatal("expected archived_at to be set")
	}
}

func assertAnnouncementArchivedAtCleared(t *testing.T, item announcementstore.Announcement) {
	t.Helper()
	if item.ArchivedAt != nil {
		t.Fatalf("expected archived_at to be empty, got %#v", item.ArchivedAt)
	}
}

type testAnnouncementRepository struct{}

func (testAnnouncementRepository) Ping(context.Context) error {
	return nil
}

func (testAnnouncementRepository) ListAdmin(context.Context, announcementstore.ListQuery) (announcementstore.ListResult, error) {
	return announcementstore.ListResult{}, announcementstore.ErrAnnouncementNotFound
}

func (testAnnouncementRepository) ListCurrentUser(context.Context, announcementstore.UserListQuery) (announcementstore.UserListResult, error) {
	return announcementstore.UserListResult{}, announcementstore.ErrAnnouncementNotFound
}

func (testAnnouncementRepository) Create(context.Context, announcementstore.CreateInput) (announcementstore.Announcement, error) {
	return announcementstore.Announcement{}, announcementstore.ErrAnnouncementNotFound
}

func (testAnnouncementRepository) GetAdmin(context.Context, uint64) (announcementstore.Announcement, error) {
	return announcementstore.Announcement{}, announcementstore.ErrAnnouncementNotFound
}

func (testAnnouncementRepository) Update(context.Context, uint64, announcementstore.UpdateInput) (announcementstore.Announcement, error) {
	return announcementstore.Announcement{}, announcementstore.ErrAnnouncementNotFound
}

func (testAnnouncementRepository) Publish(context.Context, uint64, *time.Time, time.Time, *uint64) (announcementstore.Announcement, error) {
	return announcementstore.Announcement{}, announcementstore.ErrAnnouncementNotFound
}

func (testAnnouncementRepository) Archive(context.Context, uint64, *uint64) (announcementstore.Announcement, error) {
	return announcementstore.Announcement{}, announcementstore.ErrAnnouncementNotFound
}

func (testAnnouncementRepository) Delete(context.Context, uint64, uint64, time.Time) error {
	return announcementstore.ErrAnnouncementNotFound
}

func (testAnnouncementRepository) MarkRead(context.Context, uint64, uint64, time.Time) (announcementstore.UserAnnouncement, error) {
	return announcementstore.UserAnnouncement{}, announcementstore.ErrAnnouncementNotFound
}

func (testAnnouncementRepository) MarkAllRead(context.Context, uint64, time.Time, time.Time) (int, error) {
	return 0, announcementstore.ErrAnnouncementNotFound
}

func (testAnnouncementRepository) UnreadCount(context.Context, uint64, time.Time) (int, error) {
	return 0, announcementstore.ErrAnnouncementNotFound
}

var _ announcementstore.Repository = testAnnouncementRepository{}
