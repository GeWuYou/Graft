package container

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	containercontract "graft/server/modules/container/contract"
)

const (
	runtimeEventResourceTypeContainer = "container"
	defaultRuntimeEventsHistoryLimit  = 256
	defaultRuntimeEventsHistoryTTL    = 30 * time.Minute
)

var allowedRuntimeEventTypes = map[containercontract.RuntimeEventType]struct{}{
	containercontract.RuntimeEventTypeContainerCreated:             {},
	containercontract.RuntimeEventTypeContainerStarted:             {},
	containercontract.RuntimeEventTypeContainerRestarted:           {},
	containercontract.RuntimeEventTypeContainerStopped:             {},
	containercontract.RuntimeEventTypeContainerRemoved:             {},
	containercontract.RuntimeEventTypeContainerOOMKilled:           {},
	containercontract.RuntimeEventTypeContainerHealthStatusChanged: {},
	containercontract.RuntimeEventTypeContainerExecStarted:         {},
	containercontract.RuntimeEventTypeContainerExecFinished:        {},
}

// RuntimeEvent is the canonical container runtime event domain fact.
type RuntimeEvent struct {
	ID           string            `json:"id"`
	ResourceType string            `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	EventType    string            `json:"event_type"`
	Severity     string            `json:"severity"`
	OccurredAt   time.Time         `json:"occurred_at"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

// RuntimeEventRecord is one ordered replay/delivery record.
type RuntimeEventRecord struct {
	Seq   int64        `json:"seq"`
	Event RuntimeEvent `json:"event"`
}

// RuntimeEventStreamContext carries stream-local context that does not belong to the event fact.
type RuntimeEventStreamContext struct {
	Runtime string `json:"runtime"`
}

// RuntimeEventsHistory is the bounded history payload used by HTTP seed and reconnect backfill.
type RuntimeEventsHistory struct {
	ResourceID string                    `json:"resource_id"`
	Context    RuntimeEventStreamContext `json:"context"`
	Items      []RuntimeEventRecord      `json:"items"`
}

// RuntimeEventCandidate is the source-produced normalized input before canonical severity and ids are assigned.
type RuntimeEventCandidate struct {
	ResourceID string
	EventType  containercontract.RuntimeEventType
	OccurredAt time.Time
	Attributes map[string]string
}

// canonicalRuntimeEventSeverity 根据事件类型和健康状态属性确定运行时事件的规范严重级别。
// 对容器重启返回 Warning，对 OOMKilled 返回 Error；对健康状态变更事件，health_status 为 "unhealthy" 时返回 Warning，否则返回 Info；其他事件返回 Info。
func canonicalRuntimeEventSeverity(
	eventType containercontract.RuntimeEventType,
	attributes map[string]string,
) containercontract.RuntimeEventSeverity {
	switch eventType {
	case containercontract.RuntimeEventTypeContainerRestarted:
		return containercontract.RuntimeEventSeverityWarning
	case containercontract.RuntimeEventTypeContainerOOMKilled:
		return containercontract.RuntimeEventSeverityError
	case containercontract.RuntimeEventTypeContainerHealthStatusChanged:
		switch strings.ToLower(strings.TrimSpace(attributes["health_status"])) {
		case "unhealthy":
			return containercontract.RuntimeEventSeverityWarning
		default:
			return containercontract.RuntimeEventSeverityInfo
		}
	default:
		return containercontract.RuntimeEventSeverityInfo
	}
}

// normalizeRuntimeEventCandidate 规范化并校验运行时事件候选值。
// 它会裁剪资源 ID 和事件类型，校验事件类型是否允许，将时间统一为 UTC（零值则使用当前 UTC 时间），并清理属性键值对后返回规范化结果；任一必填字段无效时返回错误。
func normalizeRuntimeEventCandidate(candidate RuntimeEventCandidate) (RuntimeEventCandidate, error) {
	resourceID := strings.TrimSpace(candidate.ResourceID)
	if resourceID == "" {
		return RuntimeEventCandidate{}, fmt.Errorf("runtime event resource id is required")
	}
	eventType := containercontract.RuntimeEventType(strings.TrimSpace(candidate.EventType.String()))
	if eventType == "" {
		return RuntimeEventCandidate{}, fmt.Errorf("runtime event type is required")
	}
	if _, ok := allowedRuntimeEventTypes[eventType]; !ok {
		return RuntimeEventCandidate{}, fmt.Errorf("runtime event type %q is invalid", eventType)
	}
	occurredAt := candidate.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	attributes := map[string]string{}
	for key, value := range candidate.Attributes {
		normalizedKey := strings.TrimSpace(key)
		normalizedValue := strings.TrimSpace(value)
		if normalizedKey == "" || normalizedValue == "" {
			continue
		}
		attributes[normalizedKey] = normalizedValue
	}

	return RuntimeEventCandidate{
		ResourceID: resourceID,
		EventType:  eventType,
		OccurredAt: occurredAt,
		Attributes: attributes,
	}, nil
}

// newRuntimeEvent 将候选事件转换为规范化的运行时事件。
func newRuntimeEvent(candidate RuntimeEventCandidate) (RuntimeEvent, error) {
	normalized, err := normalizeRuntimeEventCandidate(candidate)
	if err != nil {
		return RuntimeEvent{}, err
	}
	severity := canonicalRuntimeEventSeverity(normalized.EventType, normalized.Attributes)

	event := RuntimeEvent{
		ResourceType: runtimeEventResourceTypeContainer,
		ResourceID:   normalized.ResourceID,
		EventType:    normalized.EventType.String(),
		Severity:     severity.String(),
		OccurredAt:   normalized.OccurredAt,
		Attributes:   normalized.Attributes,
	}
	event.ID = runtimeEventOpaqueID(event)
	return event, nil
}

// runtimeEventOpaqueID 为运行时事件生成确定性的不透明标识。
// 该标识由事件的资源 ID、事件类型、发生时间和属性内容计算得出。
func runtimeEventOpaqueID(event RuntimeEvent) string {
	body, _ := json.Marshal(struct {
		ResourceID string            `json:"resource_id"`
		EventType  string            `json:"event_type"`
		OccurredAt string            `json:"occurred_at"`
		Attributes map[string]string `json:"attributes,omitempty"`
	}{
		ResourceID: event.ResourceID,
		EventType:  event.EventType,
		OccurredAt: event.OccurredAt.UTC().Format(time.RFC3339Nano),
		Attributes: event.Attributes,
	})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:12])
}
