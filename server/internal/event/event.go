package event

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrInvalidEvent 表示事件 envelope 缺少稳定路由或不可序列化的载荷。
	ErrInvalidEvent = errors.New("invalid event")
)

const eventIDByteLength = 16

// Type 是由事件 owner 定义的稳定事件类型。
type Type string

// DeliveryMode 描述发布方需要的平台投递保证，而不是事件业务内容。
type DeliveryMode string

const (
	// DeliveryBestEffort 表示事件仅需被本地内存 dispatcher 接收。
	DeliveryBestEffort DeliveryMode = "best_effort"
	// DeliveryDurable 表示事件需写入 PostgreSQL Outbox，并由 consumer delivery 恢复。
	DeliveryDurable DeliveryMode = "durable"
)

// Event 是跨模块异步处理的不可变事件 envelope。
//
// Payload 与 Metadata 必须已经完成 JSON 编码。接收事件时会复制两者，避免发布方
// 在入队后修改底层切片。Retry、状态和错误归属于某个 consumer delivery，不在此模型中。
type Event struct {
	ID             string          `json:"id"`
	Type           Type            `json:"type"`
	Version        uint16          `json:"version"`
	Source         string          `json:"source"`
	Payload        json.RawMessage `json:"payload"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	OccurredAt     time.Time       `json:"occurred_at"`
	CreatedAt      time.Time       `json:"created_at"`
	CorrelationID  string          `json:"correlation_id,omitempty"`
	CausationID    string          `json:"causation_id,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
}

// NewID 返回适合作为事件稳定标识的随机十六进制值。
func NewID() (string, error) {
	bytes := make([]byte, eventIDByteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate event id: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (e Event) normalized(now time.Time) (Event, error) {
	e.ID = strings.TrimSpace(e.ID)
	e.Type = Type(strings.TrimSpace(string(e.Type)))
	e.Source = strings.TrimSpace(e.Source)
	e.CorrelationID = strings.TrimSpace(e.CorrelationID)
	e.CausationID = strings.TrimSpace(e.CausationID)
	e.IdempotencyKey = strings.TrimSpace(e.IdempotencyKey)
	if e.ID == "" || e.Type == "" || e.Source == "" || e.Version == 0 {
		return Event{}, fmt.Errorf("%w: id, type, version and source are required", ErrInvalidEvent)
	}
	if !json.Valid(e.Payload) {
		return Event{}, fmt.Errorf("%w: payload must be valid JSON", ErrInvalidEvent)
	}
	if len(e.Metadata) > 0 && !json.Valid(e.Metadata) {
		return Event{}, fmt.Errorf("%w: metadata must be valid JSON", ErrInvalidEvent)
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = now.UTC()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now.UTC()
	}
	e.Payload = append(json.RawMessage(nil), e.Payload...)
	e.Metadata = append(json.RawMessage(nil), e.Metadata...)
	return e, nil
}
