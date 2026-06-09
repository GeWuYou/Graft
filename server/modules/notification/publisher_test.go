// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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
	requireDuplicatePublishResult(t, duplicate, result.EventID)

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

func TestServiceKeepsDeliveryMutationsUserScoped(t *testing.T) {
	stack := newNotificationTestStack(t)

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
	if err := stack.service.DeleteDelivery(context.Background(), 42, deliveryID, time.Now().UTC()); err != nil {
		t.Fatalf("delete delivery: %v", err)
	}
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

func requireDuplicatePublishResult(t *testing.T, result moduleapi.PublishNotificationResult, eventID uint64) {
	t.Helper()
	if !result.Deduplicated || result.EventID != eventID || result.RecipientCount != 0 || len(result.DeliveryIDs) != 0 {
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
		deleted_at TIMESTAMP NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY (event_id) REFERENCES notification_events(id)
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create notification schema: %v", err)
	}
	return db
}
