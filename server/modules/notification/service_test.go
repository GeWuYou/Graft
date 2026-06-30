package notification

import (
	"context"
	"math"
	"testing"
	"time"

	notificationstore "graft/server/modules/notification/store"
)

type serviceTestRepository struct {
	listQuery notificationstore.ListQuery
}

func (r *serviceTestRepository) CreateEvent(context.Context, notificationstore.CreateEventInput) (notificationstore.Event, bool, error) {
	return notificationstore.Event{}, false, nil
}

func (r *serviceTestRepository) CreateDeliveries(context.Context, []notificationstore.CreateDeliveryInput) ([]notificationstore.Delivery, error) {
	return nil, nil
}

func (r *serviceTestRepository) List(_ context.Context, query notificationstore.ListQuery) (notificationstore.ListResult, error) {
	r.listQuery = query
	return notificationstore.ListResult{}, nil
}

func (r *serviceTestRepository) Get(context.Context, uint64, uint64) (notificationstore.Notification, error) {
	return notificationstore.Notification{}, nil
}

func (r *serviceTestRepository) UnreadCount(context.Context, uint64) (int, error) {
	return 0, nil
}

func (r *serviceTestRepository) MarkRead(context.Context, uint64, uint64, time.Time) (notificationstore.Delivery, error) {
	return notificationstore.Delivery{}, nil
}

func (r *serviceTestRepository) MarkAllRead(context.Context, uint64, time.Time) (int, error) {
	return 0, nil
}

func (r *serviceTestRepository) MarkAllReadMatching(context.Context, notificationstore.ListQuery, time.Time) (int, error) {
	return 0, nil
}

func (r *serviceTestRepository) DeleteDelivery(context.Context, uint64, uint64, time.Time) error {
	return nil
}

func TestNormalizePageClampsOversizedPageBeforeOffsetOverflow(t *testing.T) {
	repository := &serviceTestRepository{}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	pageSize := 100
	inputPage := math.MaxInt
	result, err := service.List(context.Background(), ListQuery{
		RecipientUserID: 42,
		Page:            inputPage,
		PageSize:        pageSize,
	})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}

	expectedPage := math.MaxInt/pageSize + 1
	if result.Page != expectedPage {
		t.Fatalf("expected clamped page %d, got %d", expectedPage, result.Page)
	}
	expectedOffset := (expectedPage - 1) * pageSize
	if repository.listQuery.Offset != expectedOffset {
		t.Fatalf("expected clamped offset %d, got %d", expectedOffset, repository.listQuery.Offset)
	}
	if repository.listQuery.Offset < 0 {
		t.Fatalf("expected non-negative offset, got %d", repository.listQuery.Offset)
	}
}

func TestNormalizePageClampsPageSizeOneWithoutOverflow(t *testing.T) {
	repository := &serviceTestRepository{}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.List(context.Background(), ListQuery{
		RecipientUserID: 42,
		Page:            math.MaxInt,
		PageSize:        1,
	})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}

	if result.Page != math.MaxInt {
		t.Fatalf("expected clamped page %d, got %d", math.MaxInt, result.Page)
	}
	expectedOffset := math.MaxInt - 1
	if repository.listQuery.Offset != expectedOffset {
		t.Fatalf("expected clamped offset %d, got %d", expectedOffset, repository.listQuery.Offset)
	}
	if repository.listQuery.Offset < 0 {
		t.Fatalf("expected non-negative offset, got %d", repository.listQuery.Offset)
	}
}
