// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package store defines Notification Center module persistence contracts.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	// ErrInvalidInput indicates a notification store input violates the module persistence contract.
	ErrInvalidInput = errors.New("notification invalid input")
	// ErrDeliveryNotFound indicates no non-deleted delivery exists in the requested user scope.
	ErrDeliveryNotFound = errors.New("notification delivery not found")
)

// Event stores the immutable notification fact.
type Event struct {
	ID                uint64
	TitleKey          string
	Title             string
	MessageKey        string
	Message           string
	CategoryKey       string
	SourceKey         string
	LevelKey          string
	EventTypeKey      string
	ResourceTypeKey   string
	ActionLabelKey    string
	ActionLabel       string
	Severity          string
	Category          string
	SourceModule      string
	EventType         string
	ResourceType      string
	ResourceID        string
	ResourceName      string
	NavigationKind    string
	NavigationPayload json.RawMessage
	Metadata          json.RawMessage
	DedupeKey         string
	OccurredAt        time.Time
	ExpiresAt         *time.Time
	CreatedAt         time.Time
}

// Delivery stores per-user notification state.
type Delivery struct {
	ID              uint64
	EventID         uint64
	RecipientUserID uint64
	TargetType      string
	TargetRef       string
	ReadAt          *time.Time
	DeletedAt       *time.Time
	CreatedAt       time.Time
}

// Notification joins one delivery with its event fact for current-user reads.
type Notification struct {
	Event    Event
	Delivery Delivery
}

// CreateEventInput describes one notification event insert.
type CreateEventInput struct {
	TitleKey          string
	Title             string
	MessageKey        string
	Message           string
	CategoryKey       string
	SourceKey         string
	LevelKey          string
	EventTypeKey      string
	ResourceTypeKey   string
	ActionLabelKey    string
	ActionLabel       string
	Severity          string
	Category          string
	SourceModule      string
	EventType         string
	ResourceType      string
	ResourceID        string
	ResourceName      string
	NavigationKind    string
	NavigationPayload json.RawMessage
	Metadata          json.RawMessage
	DedupeKey         string
	OccurredAt        time.Time
	ExpiresAt         *time.Time
}

// CreateDeliveryInput describes one user delivery insert.
type CreateDeliveryInput struct {
	EventID         uint64
	RecipientUserID uint64
	TargetType      string
	TargetRef       string
}

// ListQuery describes current-user notification filters.
type ListQuery struct {
	RecipientUserID uint64
	Status          string
	Severity        string
	Category        string
	SourceModule    string
	OccurredFrom    *time.Time
	OccurredTo      *time.Time
	Limit           int
	Offset          int
}

// ListResult returns a paginated current-user notification page.
type ListResult struct {
	Items []Notification
	Total int
}

// Repository persists notification events and deliveries.
type Repository interface {
	CreateEvent(ctx context.Context, input CreateEventInput) (Event, bool, error)
	CreateDeliveries(ctx context.Context, inputs []CreateDeliveryInput) ([]Delivery, error)
	List(ctx context.Context, query ListQuery) (ListResult, error)
	Get(ctx context.Context, recipientUserID uint64, deliveryID uint64) (Notification, error)
	UnreadCount(ctx context.Context, recipientUserID uint64) (int, error)
	MarkRead(ctx context.Context, recipientUserID uint64, deliveryID uint64, readAt time.Time) (Delivery, error)
	MarkAllRead(ctx context.Context, recipientUserID uint64, readAt time.Time) (int, error)
	MarkAllReadMatching(ctx context.Context, query ListQuery, readAt time.Time) (int, error)
	DeleteDelivery(ctx context.Context, recipientUserID uint64, deliveryID uint64, deletedAt time.Time) error
}
