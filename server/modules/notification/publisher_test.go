package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/moduleapi"
	notificationcontract "graft/server/modules/notification/contract"
	notificationstore "graft/server/modules/notification/store"
)

func TestPublisherPersistsUserDeliveryAndDedupe(t *testing.T) {
	stack := newNotificationTestStack(t)

	result, err := stack.publisher.Publish(context.Background(), validPublishInput())
	if err != nil {
		t.Fatalf("publish notification: %v", err)
	}
	requireFirstPublishResult(t, result)

	duplicate, err := stack.publisher.Publish(context.Background(), validPublishInput())
	if err != nil {
		t.Fatalf("publish duplicate notification: %v", err)
	}
	requireDuplicatePublishResult(t, duplicate, result.EventID, result.DeliveryIDs[0])

	count, err := stack.service.UnreadCount(context.Background(), 42)
	if err != nil {
		t.Fatalf("unread count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one unread delivery, got %d", count)
	}

	page, err := stack.service.List(context.Background(), ListQuery{RecipientUserID: 42, PageSize: 10})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Event.NavigationKind != notificationcontract.NavigationAuditLog.String() {
		t.Fatalf("unexpected notification page: %#v", page)
	}
}

func TestPublisherCompensatesMissingDeliveryOnDedupeRetry(t *testing.T) {
	stack := newNotificationTestStack(t)
	input := validPublishInput()

	event, deduplicated, err := stack.repository.CreateEvent(context.Background(), createEventInputFromPublishInput(input))
	if err != nil {
		t.Fatalf("create event without deliveries: %v", err)
	}
	if event.ID == 0 || deduplicated {
		t.Fatalf("unexpected seeded event result: event=%#v deduplicated=%v", event, deduplicated)
	}

	result, err := stack.publisher.Publish(context.Background(), input)
	if err != nil {
		t.Fatalf("publish dedupe retry: %v", err)
	}
	if !result.Deduplicated || result.EventID != event.ID || result.RecipientCount != 1 || len(result.DeliveryIDs) != 1 {
		t.Fatalf("expected dedupe retry to compensate delivery fan-out, got %#v", result)
	}
	count, err := stack.service.UnreadCount(context.Background(), 42)
	if err != nil {
		t.Fatalf("unread count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one compensated delivery, got %d", count)
	}
}

func TestPublisherFansOutPermissionTarget(t *testing.T) {
	db := newNotificationTestDB(t)
	repository, err := notificationstore.NewSQLRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	publisher, err := NewPublisher(repository, permissionFanoutRBAC{userIDs: []uint64{42, 7, 42, 0}})
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}

	input := validPublishInput()
	input.Target = moduleapi.NotificationTarget{
		Type: moduleapi.NotificationTargetType(notificationcontract.TargetPermission),
		Ref:  "audit.read",
	}
	result, err := publisher.Publish(context.Background(), input)
	if err != nil {
		t.Fatalf("publish permission target: %v", err)
	}
	if result.RecipientCount != 2 {
		t.Fatalf("expected two recipients, got %#v", result)
	}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	page, err := service.List(context.Background(), ListQuery{RecipientUserID: 7, PageSize: 10})
	if err != nil {
		t.Fatalf("list permission notification: %v", err)
	}
	if page.Total != 1 || page.Items[0].Delivery.TargetType != notificationcontract.TargetPermission.String() {
		t.Fatalf("unexpected permission delivery: %#v", page)
	}
}

func TestPublisherSkipsPersistenceWhenNotificationDisabled(t *testing.T) {
	repository := &publisherSpyRepository{}
	publisher, err := NewPublisher(repository, permissionFanoutRBAC{userIDs: []uint64{42}})
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	if err := publisher.setConfigResolver(staticNotificationConfigResolver{values: map[string]bool{
		notificationEnabledKey: false,
	}}); err != nil {
		t.Fatalf("set config resolver: %v", err)
	}

	result, err := publisher.Publish(context.Background(), validPublishInput())
	if err != nil {
		t.Fatalf("publish disabled notification: %v", err)
	}
	if !result.Skipped || result.RecipientCount != 0 || result.EventID != 0 || len(result.DeliveryIDs) != 0 {
		t.Fatalf("expected skipped empty result, got %#v", result)
	}
	if repository.createEventCalls != 0 || repository.createDeliveriesCalls != 0 {
		t.Fatalf("expected no persistence calls, got events=%d deliveries=%d", repository.createEventCalls, repository.createDeliveriesCalls)
	}
}

func TestPublisherSkipsPersistenceWhenSourceDisabled(t *testing.T) {
	repository := &publisherSpyRepository{}
	publisher, err := NewPublisher(repository, permissionFanoutRBAC{userIDs: []uint64{42}})
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	if err := publisher.setConfigResolver(staticNotificationConfigResolver{values: map[string]bool{
		notificationSourceAuditIncidentEnabledKey: false,
	}}); err != nil {
		t.Fatalf("set config resolver: %v", err)
	}

	result, err := publisher.Publish(context.Background(), validPublishInput())
	if err != nil {
		t.Fatalf("publish source-disabled notification: %v", err)
	}
	if !result.Skipped {
		t.Fatalf("expected source-disabled publish to be skipped, got %#v", result)
	}
	if repository.createEventCalls != 0 || repository.createDeliveriesCalls != 0 {
		t.Fatalf("expected no persistence calls, got events=%d deliveries=%d", repository.createEventCalls, repository.createDeliveriesCalls)
	}
}

func TestPublisherUsesSchedulerSuccessSourceSwitch(t *testing.T) {
	repository := &publisherSpyRepository{}
	publisher, err := NewPublisher(repository, permissionFanoutRBAC{userIDs: []uint64{42}})
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	resolver := &recordingNotificationConfigResolver{values: map[string]bool{
		notificationSourceScheduledTaskSuccessEnabledKey: false,
	}}
	if err := publisher.setConfigResolver(resolver); err != nil {
		t.Fatalf("set config resolver: %v", err)
	}

	input := validPublishInput()
	input.SourceModule = "scheduler"
	input.EventType = "task_succeeded"
	input.DedupeKey = "scheduler:run_succeeded:99"
	result, err := publisher.Publish(context.Background(), input)
	if err != nil {
		t.Fatalf("publish scheduler success source-disabled notification: %v", err)
	}
	if !result.Skipped {
		t.Fatalf("expected scheduler success publish to be skipped, got %#v", result)
	}
	if repository.createEventCalls != 0 || repository.createDeliveriesCalls != 0 {
		t.Fatalf("expected no persistence calls, got events=%d deliveries=%d", repository.createEventCalls, repository.createDeliveriesCalls)
	}
	requireConfigKeyLookup(t, resolver, notificationSourceScheduledTaskSuccessEnabledKey)

	resolver.values[notificationSourceScheduledTaskSuccessEnabledKey] = true
	result, err = publisher.Publish(context.Background(), input)
	if err != nil {
		t.Fatalf("publish scheduler success source-enabled notification: %v", err)
	}
	if result.Skipped || repository.createEventCalls != 1 || repository.createDeliveriesCalls != 1 {
		t.Fatalf("expected scheduler success publish to persist once, result=%#v events=%d deliveries=%d", result, repository.createEventCalls, repository.createDeliveriesCalls)
	}
}

func TestPublisherSetConfigResolverRejectsInvalidInputs(t *testing.T) {
	repository := &publisherSpyRepository{}
	publisher, err := NewPublisher(repository)
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	if err := publisher.setConfigResolver(nil); err == nil {
		t.Fatalf("expected nil config resolver error")
	}

	var missingPublisher *Publisher
	if err := missingPublisher.setConfigResolver(staticNotificationConfigResolver{}); err == nil {
		t.Fatalf("expected nil publisher error")
	}
}

func TestPublisherRejectsEmptyPermissionFanoutBeforePersistingEvent(t *testing.T) {
	repository := &publisherSpyRepository{}
	publisher, err := NewPublisher(repository, permissionFanoutRBAC{userIDs: []uint64{0, 0}})
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}

	input := validPublishInput()
	input.Target = moduleapi.NotificationTarget{
		Type: moduleapi.NotificationTargetType(notificationcontract.TargetPermission),
		Ref:  "audit.read",
	}
	_, err = publisher.Publish(context.Background(), input)
	if !errors.Is(err, moduleapi.ErrNotificationInvalidInput) {
		t.Fatalf("expected invalid input for empty permission fan-out, got %v", err)
	}
	if repository.createEventCalls != 0 || repository.createDeliveriesCalls != 0 {
		t.Fatalf("expected no persistence calls, got events=%d deliveries=%d", repository.createEventCalls, repository.createDeliveriesCalls)
	}
}

func TestPublisherAllowsKeyOnlyLocalizedNotificationPayload(t *testing.T) {
	stack := newNotificationTestStack(t)
	input := validPublishInput()
	input.Title = ""
	input.Message = ""
	input.ActionLabel = ""
	input.TitleKey = "notification.title.scheduler.runSucceeded"
	input.MessageKey = "notification.message.scheduler.runSucceeded"
	input.ActionLabelKey = "notification.action.openRunRecord"

	result, err := stack.publisher.Publish(context.Background(), input)
	if err != nil {
		t.Fatalf("publish key-only notification: %v", err)
	}
	if result.EventID == 0 || result.RecipientCount != 1 {
		t.Fatalf("unexpected key-only publish result: %#v", result)
	}

	page, err := stack.service.List(context.Background(), ListQuery{RecipientUserID: 42, PageSize: 10})
	if err != nil {
		t.Fatalf("list key-only notification: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected one notification item, got %#v", page)
	}
	if page.Items[0].Event.Title != "" || page.Items[0].Event.Message != "" || page.Items[0].Event.ActionLabel != "" {
		t.Fatalf("expected stored fallback text to stay empty, got %#v", page.Items[0].Event)
	}
	if page.Items[0].Event.TitleKey != input.TitleKey ||
		page.Items[0].Event.MessageKey != input.MessageKey ||
		page.Items[0].Event.ActionLabelKey != input.ActionLabelKey {
		t.Fatalf("expected stored locale keys to match input, got %#v", page.Items[0].Event)
	}
}

func TestRepositoryCreateDeliveriesRejectsInvalidBatchWithoutPartialInsert(t *testing.T) {
	stack := newNotificationTestStack(t)
	event, _, err := stack.repository.CreateEvent(context.Background(), createEventInputFromPublishInput(validPublishInput()))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	_, err = stack.repository.CreateDeliveries(context.Background(), []notificationstore.CreateDeliveryInput{
		{
			EventID:         event.ID,
			RecipientUserID: 42,
			TargetType:      notificationcontract.TargetUser.String(),
			TargetRef:       "42",
		},
		{
			EventID:         event.ID,
			RecipientUserID: 7,
			TargetType:      "",
			TargetRef:       "7",
		},
	})
	if !errors.Is(err, notificationstore.ErrInvalidInput) {
		t.Fatalf("expected invalid delivery batch error, got %v", err)
	}

	var deliveryCount int
	if err := stack.db.QueryRow(`SELECT COUNT(*) FROM notification_deliveries WHERE event_id = ?`, event.ID).Scan(&deliveryCount); err != nil {
		t.Fatalf("count deliveries: %v", err)
	}
	if deliveryCount != 0 {
		t.Fatalf("expected invalid batch to insert no deliveries, got %d", deliveryCount)
	}
}

func TestServiceKeepsDeliveryMutationsUserScoped(t *testing.T) {
	stack := newNotificationTestStack(t)
	deletedAt := time.Date(2026, 6, 9, 11, 0, 0, 0, time.UTC)

	result, err := stack.publisher.Publish(context.Background(), validPublishInput())
	if err != nil {
		t.Fatalf("publish notification: %v", err)
	}
	deliveryID := result.DeliveryIDs[0]
	if _, err := stack.service.MarkRead(context.Background(), 7, deliveryID, time.Now().UTC()); !errors.Is(err, moduleapi.ErrNotificationDeliveryNotFound) {
		t.Fatalf("expected wrong-user read to be rejected, got %v", err)
	}
	requireUnreadDeliveryForUser(t, stack.db, deliveryID, 42)
	if err := stack.service.DeleteDelivery(context.Background(), 7, deliveryID, time.Now().UTC()); !errors.Is(err, moduleapi.ErrNotificationDeliveryNotFound) {
		t.Fatalf("expected wrong-user delete to be rejected, got %v", err)
	}

	result, err = stack.publisher.Publish(context.Background(), validPublishInputWithDedupe("audit.permission_denied.1002"))
	if err != nil {
		t.Fatalf("publish second notification: %v", err)
	}
	deliveryID = result.DeliveryIDs[0]
	if deliveryID == 0 {
		t.Fatal("expected second delivery id")
	}
	var storedUserID uint64
	if err := stack.db.QueryRow(`SELECT recipient_user_id FROM notification_deliveries WHERE id = ?`, deliveryID).Scan(&storedUserID); err != nil {
		t.Fatalf("expected second delivery row: %v", err)
	}
	if storedUserID != 42 {
		t.Fatalf("unexpected second delivery recipient: %d", storedUserID)
	}
	if _, err := stack.service.MarkRead(context.Background(), 42, deliveryID, time.Now().UTC()); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	count, err := stack.service.UnreadCount(context.Background(), 42)
	if err != nil {
		t.Fatalf("unread count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one unread delivery from the wrong-user rejection case, got %d", count)
	}
	if err := stack.service.DeleteDelivery(context.Background(), 42, deliveryID, deletedAt); err != nil {
		t.Fatalf("delete delivery: %v", err)
	}
	requireDeletedDeliveryEpoch(t, stack.db, deliveryID, deletedAt)
	count, err = stack.service.UnreadCount(context.Background(), 42)
	if err != nil {
		t.Fatalf("unread count after delete: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected deleted read delivery to stay hidden from unread count, got %d", count)
	}
}

func TestServiceMarkReadReturnsNotFoundWhenUpdateAffectsNoRows(t *testing.T) {
	stack := newNotificationTestStack(t)

	result, err := stack.publisher.Publish(context.Background(), validPublishInput())
	if err != nil {
		t.Fatalf("publish notification: %v", err)
	}
	deliveryID := result.DeliveryIDs[0]
	if _, err := stack.db.Exec(`CREATE TRIGGER notification_mark_read_noop
		BEFORE UPDATE OF read_at ON notification_deliveries
		BEGIN
			SELECT RAISE(IGNORE);
		END;`); err != nil {
		t.Fatalf("create no-op update trigger: %v", err)
	}

	if _, err := stack.service.MarkRead(context.Background(), 42, deliveryID, time.Now().UTC()); !errors.Is(err, moduleapi.ErrNotificationDeliveryNotFound) {
		t.Fatalf("expected no-row read update to return not found, got %v", err)
	}
	requireUnreadDeliveryForUser(t, stack.db, deliveryID, 42)
}

type notificationTestStack struct {
	db         *sql.DB
	repository notificationstore.Repository
	publisher  *Publisher
	service    *Service
}

func newNotificationTestStack(t *testing.T) notificationTestStack {
	t.Helper()
	db := newNotificationTestDB(t)
	repository, err := notificationstore.NewSQLRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	publisher, err := NewPublisher(repository)
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return notificationTestStack{
		db:         db,
		repository: repository,
		publisher:  publisher,
		service:    service,
	}
}

func requireFirstPublishResult(t *testing.T, result moduleapi.PublishNotificationResult) {
	t.Helper()
	if result.EventID == 0 || result.RecipientCount != 1 || len(result.DeliveryIDs) != 1 || result.Deduplicated {
		t.Fatalf("unexpected publish result: %#v", result)
	}
}

func requireDuplicatePublishResult(t *testing.T, result moduleapi.PublishNotificationResult, eventID uint64, deliveryID uint64) {
	t.Helper()
	if !result.Deduplicated || result.EventID != eventID || result.RecipientCount != 1 ||
		len(result.DeliveryIDs) != 1 || result.DeliveryIDs[0] != deliveryID {
		t.Fatalf("unexpected duplicate result: %#v", result)
	}
}

func requireUnreadDeliveryForUser(t *testing.T, db *sql.DB, deliveryID uint64, userID uint64) {
	t.Helper()
	var storedUserID uint64
	var storedReadAt sql.NullTime
	if err := db.QueryRow(`SELECT recipient_user_id, read_at FROM notification_deliveries WHERE id = ?`, deliveryID).Scan(&storedUserID, &storedReadAt); err != nil {
		t.Fatalf("read delivery state: %v", err)
	}
	if storedUserID != userID || storedReadAt.Valid {
		t.Fatalf("wrong-user read changed delivery state: user=%d read_valid=%v", storedUserID, storedReadAt.Valid)
	}
}

func requireDeletedDeliveryEpoch(t *testing.T, db *sql.DB, deliveryID uint64, deletedAt time.Time) {
	t.Helper()
	var storedDeletedAt int64
	if err := db.QueryRow(`SELECT deleted_at FROM notification_deliveries WHERE id = ?`, deliveryID).Scan(&storedDeletedAt); err != nil {
		t.Fatalf("read deleted delivery state: %v", err)
	}
	if storedDeletedAt != deletedAt.Unix() {
		t.Fatalf("expected deleted_at epoch %d, got %d", deletedAt.Unix(), storedDeletedAt)
	}
}

type publisherSpyRepository struct {
	createEventCalls      int
	createDeliveriesCalls int
}

type staticNotificationConfigResolver struct {
	values map[string]bool
}

func (r staticNotificationConfigResolver) Boolean(_ context.Context, key string, fallback bool) bool {
	value, ok := r.values[key]
	if !ok {
		return fallback
	}
	return value
}

type recordingNotificationConfigResolver struct {
	values map[string]bool
	keys   []string
}

func (r *recordingNotificationConfigResolver) Boolean(_ context.Context, key string, fallback bool) bool {
	r.keys = append(r.keys, key)
	value, ok := r.values[key]
	if !ok {
		return fallback
	}
	return value
}

func requireConfigKeyLookup(t *testing.T, resolver *recordingNotificationConfigResolver, key string) {
	t.Helper()
	for _, observed := range resolver.keys {
		if observed == key {
			return
		}
	}
	t.Fatalf("expected config key %q to be looked up, got %#v", key, resolver.keys)
}

func (r *publisherSpyRepository) CreateEvent(context.Context, notificationstore.CreateEventInput) (notificationstore.Event, bool, error) {
	r.createEventCalls++
	return notificationstore.Event{ID: 1001}, false, nil
}

func (r *publisherSpyRepository) CreateDeliveries(context.Context, []notificationstore.CreateDeliveryInput) ([]notificationstore.Delivery, error) {
	r.createDeliveriesCalls++
	return []notificationstore.Delivery{{ID: 2001}}, nil
}

func (r *publisherSpyRepository) List(context.Context, notificationstore.ListQuery) (notificationstore.ListResult, error) {
	return notificationstore.ListResult{}, nil
}

func (r *publisherSpyRepository) Get(context.Context, uint64, uint64) (notificationstore.Notification, error) {
	return notificationstore.Notification{}, nil
}

func (r *publisherSpyRepository) UnreadCount(context.Context, uint64) (int, error) {
	return 0, nil
}

func (r *publisherSpyRepository) MarkRead(context.Context, uint64, uint64, time.Time) (notificationstore.Delivery, error) {
	return notificationstore.Delivery{}, nil
}

func (r *publisherSpyRepository) MarkAllRead(context.Context, uint64, time.Time) (int, error) {
	return 0, nil
}

func (r *publisherSpyRepository) MarkAllReadMatching(context.Context, notificationstore.ListQuery, time.Time) (int, error) {
	return 0, nil
}

func (r *publisherSpyRepository) DeleteDelivery(context.Context, uint64, uint64, time.Time) error {
	return nil
}

func validPublishInput() moduleapi.PublishNotificationInput {
	return validPublishInputWithDedupe("audit.permission_denied.1001")
}

func validPublishInputWithDedupe(dedupeKey string) moduleapi.PublishNotificationInput {
	return moduleapi.PublishNotificationInput{
		Title:        "Permission denied",
		Message:      "A permission denial needs review.",
		Severity:     moduleapi.NotificationSeverity(notificationcontract.SeverityWarning),
		Category:     moduleapi.NotificationCategory(notificationcontract.CategorySecurity),
		SourceModule: "audit",
		EventType:    "permission_denied",
		ResourceType: "audit_log",
		ResourceID:   "1001",
		ResourceName: "Audit log 1001",
		Navigation: moduleapi.NotificationNavigation{
			Kind:    moduleapi.NotificationNavigationKind(notificationcontract.NavigationAuditLog),
			Payload: json.RawMessage(`{"audit_log_id":"1001"}`),
		},
		Metadata:   json.RawMessage(`{"request_id":"req-1"}`),
		DedupeKey:  dedupeKey,
		OccurredAt: time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
		Target: moduleapi.NotificationTarget{
			Type: moduleapi.NotificationTargetType(notificationcontract.TargetUser),
			Ref:  "42",
		},
	}
}

type permissionFanoutRBAC struct {
	userIDs []uint64
}

func (p permissionFanoutRBAC) ListRoleNamesByUserID(context.Context, uint64) ([]string, error) {
	return nil, nil
}

func (p permissionFanoutRBAC) ListPermissionCodesByUserID(context.Context, uint64) ([]string, error) {
	return nil, nil
}

func (p permissionFanoutRBAC) ListUserIDsByPermissionCode(context.Context, string) ([]uint64, error) {
	return p.userIDs, nil
}

func (p permissionFanoutRBAC) ListRoleSummariesByUserIDs(context.Context, []uint64) (map[uint64][]moduleapi.RoleSummary, error) {
	return nil, nil
}

func newNotificationTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close sqlite db: %v", err)
		}
	})

	schema := `CREATE TABLE notification_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title_key TEXT NOT NULL DEFAULT '',
		title TEXT NOT NULL,
		message_key TEXT NOT NULL DEFAULT '',
		message TEXT NOT NULL,
		category_key TEXT NOT NULL DEFAULT '',
		source_key TEXT NOT NULL DEFAULT '',
		level_key TEXT NOT NULL DEFAULT '',
		event_type_key TEXT NOT NULL DEFAULT '',
		resource_type_key TEXT NOT NULL DEFAULT '',
		action_label_key TEXT NOT NULL DEFAULT '',
		action_label TEXT NOT NULL DEFAULT '',
		severity TEXT NOT NULL,
		category TEXT NOT NULL,
		source_module TEXT NOT NULL,
		event_type TEXT NOT NULL,
		resource_type TEXT NOT NULL DEFAULT '',
		resource_id TEXT NOT NULL DEFAULT '',
		resource_name TEXT NOT NULL DEFAULT '',
		navigation_kind TEXT NOT NULL,
		navigation_payload BLOB NOT NULL DEFAULT '{}',
		metadata BLOB NOT NULL DEFAULT '{}',
		dedupe_key TEXT NULL UNIQUE,
		occurred_at TIMESTAMP NOT NULL,
		expires_at TIMESTAMP NULL,
		created_at TIMESTAMP NOT NULL
	);
	CREATE TABLE notification_deliveries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id INTEGER NOT NULL,
		recipient_user_id INTEGER NOT NULL,
		target_type TEXT NOT NULL,
		target_ref TEXT NOT NULL DEFAULT '',
		read_at TIMESTAMP NULL,
		deleted_at INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL,
		UNIQUE (event_id, recipient_user_id),
		FOREIGN KEY (event_id) REFERENCES notification_events(id)
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create notification schema: %v", err)
	}
	return db
}
