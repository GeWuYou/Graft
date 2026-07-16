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

// NotificationSeverity 定义稳定的通知严重级别契约。
type NotificationSeverity string

// NotificationCategory 定义稳定的通知分类契约。
type NotificationCategory string

// NotificationTargetType 定义稳定的通知投递目标类型契约。
type NotificationTargetType string

// NotificationNavigationKind 定义稳定的通知业务导航契约。
type NotificationNavigationKind string

// NotificationTarget 描述来源模块请求的一种通知发布目标。
type NotificationTarget struct {
	Type NotificationTargetType
	Ref  string
}

// NotificationNavigation 描述结构化的业务导航目标。
type NotificationNavigation struct {
	Kind    NotificationNavigationKind
	Payload json.RawMessage
}

// PublishNotificationInput 描述稳定的跨模块通知发布请求。
//
// 来源模块负责事件检测与业务上下文；通知中心负责校验、持久化和投递状态。
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

// PublishNotificationResult 返回供来源模块记录日志的有界投递结果。
type PublishNotificationResult struct {
	EventID        uint64
	DeliveryIDs    []uint64
	RecipientCount int
	Deduplicated   bool
	Skipped        bool
}

// NotificationPublisher 暴露站内通知的稳定跨模块能力。
type NotificationPublisher interface {
	Publish(ctx context.Context, input PublishNotificationInput) (PublishNotificationResult, error)
}

// SystemConfigResolver 暴露跨模块的系统配置有效值读取能力。
//
// 调用方必须提供系统配置 authority 已注册的 key，并为布尔读取提供显式回退值。
// 实现必须继续以 configregistry 和 system-config service 为 authority，不得向消费者暴露
// 存储细节或覆盖表访问；模块应在装配阶段解析该能力，避免在请求热路径中反复查找。
type SystemConfigResolver interface {
	IsBooleanConfigEnabled(ctx context.Context, key string, fallback bool) bool
	ResolveDefaultConfig(ctx context.Context, key string) (string, error)
}
