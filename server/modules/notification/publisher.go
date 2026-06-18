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

	"go.uber.org/zap"

	"graft/server/internal/moduleapi"
	notificationcontract "graft/server/modules/notification/contract"
	notificationstore "graft/server/modules/notification/store"
)

// Publisher implements the cross-module NotificationPublisher capability.
type Publisher struct {
	repository notificationstore.Repository
	rbac       moduleapi.RBACAccessService
	config     ConfigResolver
	logger     *zap.Logger
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

func (p *Publisher) setConfigResolver(resolver ConfigResolver) error {
	if p == nil {
		return errors.New("notification publisher is unavailable")
	}
	if resolver == nil {
		return errors.New("notification config resolver is required")
	}

	p.config = resolver
	return nil
}

func (p *Publisher) setLogger(logger *zap.Logger) {
	if p == nil || logger == nil {
		return
	}
	p.logger = logger
}

// Publish validates, persists, and fans out one notification event.
func (p *Publisher) Publish(ctx context.Context, input moduleapi.PublishNotificationInput) (moduleapi.PublishNotificationResult, error) {
	normalized, enabled, err := p.preparePublish(ctx, input)
	if err != nil {
		return moduleapi.PublishNotificationResult{}, err
	}
	if !enabled {
		p.debugPublishDecision(ctx, normalized, false, moduleapi.PublishNotificationResult{Skipped: true}, nil)
		return moduleapi.PublishNotificationResult{Skipped: true}, nil
	}

	recipients, err := p.resolveRecipients(ctx, normalized.Target)
	if err != nil {
		return moduleapi.PublishNotificationResult{}, err
	}
	if len(recipients) == 0 {
		return moduleapi.PublishNotificationResult{}, fmt.Errorf("%w: recipients", moduleapi.ErrNotificationInvalidInput)
	}

	event, deduplicated, err := p.repository.CreateEvent(ctx, createEventInputFromPublishInput(normalized))
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
	result := moduleapi.PublishNotificationResult{
		EventID:        event.ID,
		DeliveryIDs:    deliveryIDs,
		RecipientCount: len(deliveryIDs),
		Deduplicated:   deduplicated,
	}
	p.debugPublishDecision(ctx, normalized, true, result, nil)
	return result, nil
}

func createEventInputFromPublishInput(input moduleapi.PublishNotificationInput) notificationstore.CreateEventInput {
	return notificationstore.CreateEventInput{
		TitleKey:          input.TitleKey,
		Title:             input.Title,
		MessageKey:        input.MessageKey,
		Message:           input.Message,
		CategoryKey:       input.CategoryKey,
		SourceKey:         input.SourceKey,
		LevelKey:          input.LevelKey,
		EventTypeKey:      input.EventTypeKey,
		ResourceTypeKey:   input.ResourceTypeKey,
		ActionLabelKey:    input.ActionLabelKey,
		ActionLabel:       input.ActionLabel,
		Severity:          string(input.Severity),
		Category:          string(input.Category),
		SourceModule:      input.SourceModule,
		EventType:         input.EventType,
		ResourceType:      input.ResourceType,
		ResourceID:        input.ResourceID,
		ResourceName:      input.ResourceName,
		NavigationKind:    string(input.Navigation.Kind),
		NavigationPayload: input.Navigation.Payload,
		Metadata:          input.Metadata,
		DedupeKey:         input.DedupeKey,
		OccurredAt:        input.OccurredAt,
		ExpiresAt:         input.ExpiresAt,
	}
}

func (p *Publisher) preparePublish(
	ctx context.Context,
	input moduleapi.PublishNotificationInput,
) (moduleapi.PublishNotificationInput, bool, error) {
	if p == nil || p.repository == nil {
		return moduleapi.PublishNotificationInput{}, false, errors.New("notification publisher is unavailable")
	}
	normalized, err := normalizePublishInput(input)
	if err != nil {
		return moduleapi.PublishNotificationInput{}, false, err
	}
	return normalized, p.notificationEnabled(ctx, normalized), nil
}

func (p *Publisher) notificationEnabled(ctx context.Context, input moduleapi.PublishNotificationInput) bool {
	if p == nil || p.config == nil {
		return true
	}
	if !p.config.Boolean(ctx, notificationEnabledKey, true) {
		return false
	}
	if !p.config.Boolean(ctx, notificationDeliveryInAppEnabledKey, true) {
		return false
	}
	sourceKey := notificationSourceEnabledKey(input.SourceModule, input.EventType)
	return sourceKey == "" || p.config.Boolean(ctx, sourceKey, true)
}

func (p *Publisher) debugPublishDecision(
	ctx context.Context,
	input moduleapi.PublishNotificationInput,
	enabled bool,
	result moduleapi.PublishNotificationResult,
	err error,
) {
	if p == nil || p.logger == nil {
		return
	}
	sourceKey := notificationSourceEnabledKey(input.SourceModule, input.EventType)
	fields := []zap.Field{
		zap.String("module", moduleID),
		zap.String("sourceModule", input.SourceModule),
		zap.String("eventType", input.EventType),
		zap.String("sourceConfigKey", sourceKey),
		zap.Bool("notificationEnabled", p.configBool(ctx, notificationEnabledKey, true)),
		zap.Bool("inAppDeliveryEnabled", p.configBool(ctx, notificationDeliveryInAppEnabledKey, true)),
		zap.Bool("sourceEnabled", sourceKey == "" || p.configBool(ctx, sourceKey, true)),
		zap.String("targetType", string(input.Target.Type)),
		zap.String("targetRef", input.Target.Ref),
		zap.String("dedupeKey", input.DedupeKey),
		zap.Bool("enabled", enabled),
		zap.Bool("skipped", result.Skipped),
		zap.Bool("deduplicated", result.Deduplicated),
		zap.Uint64("notificationEventID", result.EventID),
		zap.Int("recipientCount", result.RecipientCount),
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	p.logger.Debug("notification publish decision", fields...)
}

func (p *Publisher) configBool(ctx context.Context, key string, fallback bool) bool {
	if p == nil || p.config == nil {
		return fallback
	}
	return p.config.Boolean(ctx, key, fallback)
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
	input.CategoryKey = strings.TrimSpace(input.CategoryKey)
	input.SourceKey = strings.TrimSpace(input.SourceKey)
	input.LevelKey = strings.TrimSpace(input.LevelKey)
	input.EventTypeKey = strings.TrimSpace(input.EventTypeKey)
	input.ResourceTypeKey = strings.TrimSpace(input.ResourceTypeKey)
	input.ActionLabelKey = strings.TrimSpace(input.ActionLabelKey)
	input.ActionLabel = strings.TrimSpace(input.ActionLabel)
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
	if (input.TitleKey == "" && input.Title == "") ||
		(input.MessageKey == "" && input.Message == "") ||
		input.SourceModule == "" ||
		input.EventType == "" {
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
