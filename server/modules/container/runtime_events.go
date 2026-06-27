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

func normalizeRuntimeEventCandidate(candidate RuntimeEventCandidate) (RuntimeEventCandidate, error) {
	resourceID := strings.TrimSpace(candidate.ResourceID)
	if resourceID == "" {
		return RuntimeEventCandidate{}, fmt.Errorf("runtime event resource id is required")
	}
	eventType := containercontract.RuntimeEventType(strings.TrimSpace(candidate.EventType.String()))
	if eventType == "" {
		return RuntimeEventCandidate{}, fmt.Errorf("runtime event type is required")
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
