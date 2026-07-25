package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"graft/server/internal/event"
)

const (
	// AccessLogPersistEventType 是 HTTP runtime 拥有的访问日志持久化事件类型。
	AccessLogPersistEventType    event.Type = "httpx.access-log.persist.v1"
	accessLogPersistEventVersion uint16     = 1
	accessLogPersistEventSource  string     = "internal.httpx"
)

type accessLogPersistEventPayload struct {
	Record CreateAccessLogInput `json:"record"`
}

type accessLogEventPersistSink struct {
	publisher event.Publisher
	fallback  AccessLogRepository
}

func (s accessLogEventPersistSink) PersistAccessLog(ctx context.Context, record CreateAccessLogInput) error {
	payload, err := json.Marshal(accessLogPersistEventPayload{Record: record})
	if err != nil {
		return fmt.Errorf("encode access log persistence event: %w", err)
	}
	eventID, err := event.NewID()
	if err != nil {
		return fmt.Errorf("create access log persistence event id: %w", err)
	}
	_, err = s.publisher.PublishAsync(event.Event{
		ID:             eventID,
		Type:           AccessLogPersistEventType,
		Version:        accessLogPersistEventVersion,
		Source:         accessLogPersistEventSource,
		Payload:        payload,
		OccurredAt:     record.OccurredAt,
		CorrelationID:  record.RequestID,
		IdempotencyKey: record.RequestID,
	}, event.PublishOptions{Delivery: event.DeliveryBestEffort})
	if errors.Is(err, event.ErrDispatcherStopped) && s.fallback != nil {
		_, fallbackErr := s.fallback.CreateAccessLog(ctx, record)
		return fallbackErr
	}
	if err != nil {
		return fmt.Errorf("publish access log persistence event: %w", err)
	}
	return nil
}

// NewAccessLogEventHandler 创建把访问日志事件写入现有 repository 的消费者。
// Handler 的重试由 Runtime event dispatcher 负责；repository 不应在此处自行重试或启动后台任务。
func NewAccessLogEventHandler(repo AccessLogRepository) event.Handler {
	return accessLogEventHandler{repo: repo}
}

type accessLogEventHandler struct {
	repo AccessLogRepository
}

func (accessLogEventHandler) ID() string { return "httpx.access-log-persist" }

func (accessLogEventHandler) Types() []event.Type { return []event.Type{AccessLogPersistEventType} }

func (h accessLogEventHandler) Handle(ctx context.Context, current event.Event) error {
	if h.repo == nil {
		return fmt.Errorf("access log repository is unavailable")
	}
	var payload accessLogPersistEventPayload
	if err := json.Unmarshal(current.Payload, &payload); err != nil {
		return fmt.Errorf("decode access log persistence event: %w", err)
	}
	if _, err := h.repo.CreateAccessLog(ctx, payload.Record); err != nil {
		return fmt.Errorf("persist access log: %w", err)
	}
	return nil
}
