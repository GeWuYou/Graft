// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package moduleapi

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	// ErrNotificationInvalidInput 表示发布方提交的通知载荷不满足稳定契约。
	ErrNotificationInvalidInput = errors.New("notification invalid input")
	// ErrNotificationTargetUnsupported 表示目标类型已进入 contract，但当前阶段尚未支持 fan-out。
	ErrNotificationTargetUnsupported = errors.New("notification target unsupported")
	// ErrNotificationDeliveryNotFound 表示当前用户范围内找不到目标投递记录。
	ErrNotificationDeliveryNotFound = errors.New("notification delivery not found")
	// ErrNotificationDisabled 表示通知总开关、来源开关或站内投递开关当前关闭。
	ErrNotificationDisabled = errors.New("notification disabled")
)

// NotificationSeverity identifies the stable notification severity contract.
type NotificationSeverity string

// NotificationCategory identifies the stable notification category contract.
type NotificationCategory string

// NotificationTargetType identifies the stable notification delivery target contract.
type NotificationTargetType string

// NotificationNavigationKind identifies the stable notification navigation contract.
type NotificationNavigationKind string

// NotificationTarget describes one publication target requested by a source module.
type NotificationTarget struct {
	Type NotificationTargetType
	Ref  string
}

// NotificationNavigation describes the structured business navigation target.
type NotificationNavigation struct {
	Kind    NotificationNavigationKind
	Payload json.RawMessage
}

// PublishNotificationInput describes the stable cross-module notification publication request.
//
// Source modules own event detection and business context. Notification Center owns validation,
// persistence, and delivery state.
type PublishNotificationInput struct {
	TitleKey        string
	Title           string
	MessageKey      string
	Message         string
	CategoryKey     string
	SourceKey       string
	LevelKey        string
	EventTypeKey    string
	ResourceTypeKey string
	ActionLabelKey  string
	ActionLabel     string
	Severity        NotificationSeverity
	Category        NotificationCategory
	SourceModule    string
	EventType       string
	ResourceType    string
	ResourceID      string
	ResourceName    string
	Navigation      NotificationNavigation
	Metadata        json.RawMessage
	DedupeKey       string
	OccurredAt      time.Time
	ExpiresAt       *time.Time
	Target          NotificationTarget
}

// PublishNotificationResult returns bounded delivery information for source-module logging.
type PublishNotificationResult struct {
	EventID        uint64
	DeliveryIDs    []uint64
	RecipientCount int
	Deduplicated   bool
	Skipped        bool
}

// NotificationPublisher exposes the stable cross-module capability for in-app notifications.
type NotificationPublisher interface {
	Publish(ctx context.Context, input PublishNotificationInput) (PublishNotificationResult, error)
}

// SystemConfigResolver 暴露跨模块布尔配置开关的有效值查询能力。
//
// 调用方必须传入已由配置 authority 注册的布尔配置 key，并提供该配置在不可解析时的显式 fallback。
// 实现方不得暴露配置存储或模块内部细节；当 key 不存在、类型不是布尔值、底层读取失败或有效值无法解析时，
// 返回调用方提供的 fallback。依赖该 capability 的模块仍应在装配阶段显式解析服务，避免业务路径中反复查找。
type SystemConfigResolver interface {
	IsBooleanConfigEnabled(ctx context.Context, key string, fallback bool) bool
}
