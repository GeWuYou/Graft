// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
	notificationcontract "graft/server/modules/notification/contract"
	notificationstore "graft/server/modules/notification/store"
)

// Publisher implements the cross-module NotificationPublisher capability.
type Publisher struct {
	repository notificationstore.Repository
	rbac       moduleapi.RBACAccessService
}

// NewPublisher creates a moduleapi.NotificationPublisher implementation.
func NewPublisher(repository notificationstore.Repository, rbac ...moduleapi.RBACAccessService) (*Publisher, error) {
	if repository == nil {
		return nil, errors.New("notification repository is unavailable")
	}
	publisher := &Publisher{repository: repository}
	if len(rbac) > 0 {
		publisher.rbac = rbac[0]
	}
	return publisher, nil
}

func (p *Publisher) setRBACAccessService(rbac moduleapi.RBACAccessService) error {
	if p == nil {
		return errors.New("notification publisher is unavailable")
	}
	if rbac == nil {
		return errors.New("rbac access service is required")
	}

	p.rbac = rbac
	return nil
}

// Publish validates, persists, and fans out one notification event.
func (p *Publisher) Publish(ctx context.Context, input moduleapi.PublishNotificationInput) (moduleapi.PublishNotificationResult, error) {
	if p == nil || p.repository == nil {
		return moduleapi.PublishNotificationResult{}, errors.New("notification publisher is unavailable")
	}
	normalized, err := normalizePublishInput(input)
	if err != nil {
		return moduleapi.PublishNotificationResult{}, err
	}

	recipients, err := p.resolveRecipients(ctx, normalized.Target)
	if err != nil {
		return moduleapi.PublishNotificationResult{}, err
	}
	if len(recipients) == 0 {
		return moduleapi.PublishNotificationResult{}, fmt.Errorf("%w: recipients", moduleapi.ErrNotificationInvalidInput)
	}

	event, deduplicated, err := p.repository.CreateEvent(ctx, notificationstore.CreateEventInput{
		TitleKey:          normalized.TitleKey,
		Title:             normalized.Title,
		MessageKey:        normalized.MessageKey,
		Message:           normalized.Message,
		Severity:          string(normalized.Severity),
		Category:          string(normalized.Category),
		SourceModule:      normalized.SourceModule,
		EventType:         normalized.EventType,
		ResourceType:      normalized.ResourceType,
		ResourceID:        normalized.ResourceID,
		ResourceName:      normalized.ResourceName,
		NavigationKind:    string(normalized.Navigation.Kind),
		NavigationPayload: normalized.Navigation.Payload,
		Metadata:          normalized.Metadata,
		DedupeKey:         normalized.DedupeKey,
		OccurredAt:        normalized.OccurredAt,
		ExpiresAt:         normalized.ExpiresAt,
	})
	if err != nil {
		return moduleapi.PublishNotificationResult{}, mapStoreError(err)
	}

	deliveryInputs := make([]notificationstore.CreateDeliveryInput, 0, len(recipients))
	for _, userID := range recipients {
		deliveryInputs = append(deliveryInputs, notificationstore.CreateDeliveryInput{
			EventID:         event.ID,
			RecipientUserID: userID,
			TargetType:      string(normalized.Target.Type),
			TargetRef:       normalized.Target.Ref,
		})
	}
	deliveries, err := p.repository.CreateDeliveries(ctx, deliveryInputs)
	if err != nil {
		return moduleapi.PublishNotificationResult{}, mapStoreError(err)
	}

	deliveryIDs := make([]uint64, 0, len(deliveries))
	for _, delivery := range deliveries {
		deliveryIDs = append(deliveryIDs, delivery.ID)
	}
	return moduleapi.PublishNotificationResult{
		EventID:        event.ID,
		DeliveryIDs:    deliveryIDs,
		RecipientCount: len(deliveryIDs),
		Deduplicated:   deduplicated,
	}, nil
}

func normalizePublishInput(input moduleapi.PublishNotificationInput) (moduleapi.PublishNotificationInput, error) {
	input = normalizePublishTextFields(input)
	input = normalizePublishJSONFields(input)
	input = normalizePublishTimes(input)
	if err := validatePublishInput(input); err != nil {
		return moduleapi.PublishNotificationInput{}, err
	}
	return input, nil
}

func normalizePublishTextFields(input moduleapi.PublishNotificationInput) moduleapi.PublishNotificationInput {
	input.TitleKey = strings.TrimSpace(input.TitleKey)
	input.Title = strings.TrimSpace(input.Title)
	input.MessageKey = strings.TrimSpace(input.MessageKey)
	input.Message = strings.TrimSpace(input.Message)
	input.SourceModule = strings.TrimSpace(input.SourceModule)
	input.EventType = strings.TrimSpace(input.EventType)
	input.ResourceType = strings.TrimSpace(input.ResourceType)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	input.ResourceName = strings.TrimSpace(input.ResourceName)
	input.DedupeKey = strings.TrimSpace(input.DedupeKey)
	input.Target.Ref = strings.TrimSpace(input.Target.Ref)
	return input
}

func normalizePublishJSONFields(input moduleapi.PublishNotificationInput) moduleapi.PublishNotificationInput {
	if len(input.Navigation.Payload) == 0 {
		input.Navigation.Payload = json.RawMessage(`{}`)
	}
	if len(input.Metadata) == 0 {
		input.Metadata = json.RawMessage(`{}`)
	}
	return input
}

func normalizePublishTimes(input moduleapi.PublishNotificationInput) moduleapi.PublishNotificationInput {
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	} else {
		input.OccurredAt = input.OccurredAt.UTC()
	}
	if input.ExpiresAt != nil {
		expiresAt := input.ExpiresAt.UTC()
		input.ExpiresAt = &expiresAt
	}
	return input
}

func validatePublishInput(input moduleapi.PublishNotificationInput) error {
	if input.Title == "" || input.Message == "" || input.SourceModule == "" || input.EventType == "" {
		return moduleapi.ErrNotificationInvalidInput
	}
	if err := validatePublishContract(input); err != nil {
		return err
	}
	if !json.Valid(input.Navigation.Payload) || !json.Valid(input.Metadata) {
		return fmt.Errorf("%w: json payload", moduleapi.ErrNotificationInvalidInput)
	}
	return nil
}

func validatePublishContract(input moduleapi.PublishNotificationInput) error {
	if !notificationcontract.ValidSeverity(notificationcontract.Severity(input.Severity)) {
		return fmt.Errorf("%w: severity", moduleapi.ErrNotificationInvalidInput)
	}
	if !notificationcontract.ValidCategory(notificationcontract.Category(input.Category)) {
		return fmt.Errorf("%w: category", moduleapi.ErrNotificationInvalidInput)
	}
	if !notificationcontract.ValidTargetType(notificationcontract.TargetType(input.Target.Type)) {
		return fmt.Errorf("%w: target_type", moduleapi.ErrNotificationInvalidInput)
	}
	if !notificationcontract.ValidNavigationKind(notificationcontract.NavigationKind(input.Navigation.Kind)) {
		return fmt.Errorf("%w: navigation_kind", moduleapi.ErrNotificationInvalidInput)
	}
	return nil
}

func (p *Publisher) resolveRecipients(ctx context.Context, target moduleapi.NotificationTarget) ([]uint64, error) {
	switch notificationcontract.TargetType(target.Type) {
	case notificationcontract.TargetUser:
		userID, err := strconv.ParseUint(strings.TrimSpace(target.Ref), 10, 64)
		if err != nil || userID == 0 {
			return nil, fmt.Errorf("%w: target_ref", moduleapi.ErrNotificationInvalidInput)
		}
		return []uint64{userID}, nil
	case notificationcontract.TargetPermission:
		if p.rbac == nil {
			return nil, fmt.Errorf("%w: rbac access service", moduleapi.ErrNotificationTargetUnsupported)
		}
		permissionCode := strings.TrimSpace(target.Ref)
		if permissionCode == "" {
			return nil, fmt.Errorf("%w: target_ref", moduleapi.ErrNotificationInvalidInput)
		}
		userIDs, err := p.rbac.ListUserIDsByPermissionCode(ctx, permissionCode)
		if err != nil {
			return nil, err
		}
		return stableRecipientUserIDs(userIDs), nil
	case notificationcontract.TargetRole, notificationcontract.TargetSystem:
		return nil, fmt.Errorf("%w: %s", moduleapi.ErrNotificationTargetUnsupported, target.Type)
	default:
		return nil, fmt.Errorf("%w: target_type", moduleapi.ErrNotificationInvalidInput)
	}
}

func stableRecipientUserIDs(userIDs []uint64) []uint64 {
	recipients := make([]uint64, 0, len(userIDs))
	seen := make(map[uint64]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if userID == 0 {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		recipients = append(recipients, userID)
	}
	return recipients
}

var _ moduleapi.NotificationPublisher = (*Publisher)(nil)
