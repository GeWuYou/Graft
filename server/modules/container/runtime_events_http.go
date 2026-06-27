package container

import "time"

type containerRuntimeEventResponse struct {
	ID           string            `json:"id"`
	ResourceType string            `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	EventType    string            `json:"event_type"`
	Severity     string            `json:"severity"`
	OccurredAt   time.Time         `json:"occurred_at"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

type containerRuntimeEventRecordResponse struct {
	Seq   int64                         `json:"seq"`
	Event containerRuntimeEventResponse `json:"event"`
}

type containerRuntimeEventStreamContextResponse struct {
	Runtime string `json:"runtime"`
}

type containerRuntimeEventsHistoryResponse struct {
	ResourceID string                                     `json:"resource_id"`
	Context    containerRuntimeEventStreamContextResponse `json:"context"`
	Items      []containerRuntimeEventRecordResponse      `json:"items"`
}

// toRuntimeEventsHistoryResponse 将运行时事件历史转换为 HTTP 历史响应。
//
// 返回值包含资源 ID、运行时上下文以及按顺序排列的事件记录；事件属性会被拷贝到响应中。
func toRuntimeEventsHistoryResponse(history RuntimeEventsHistory) containerRuntimeEventsHistoryResponse {
	items := make([]containerRuntimeEventRecordResponse, 0, len(history.Items))
	for _, item := range history.Items {
		items = append(items, containerRuntimeEventRecordResponse{
			Seq: item.Seq,
			Event: containerRuntimeEventResponse{
				ID:           item.Event.ID,
				ResourceType: item.Event.ResourceType,
				ResourceID:   item.Event.ResourceID,
				EventType:    item.Event.EventType,
				Severity:     item.Event.Severity,
				OccurredAt:   item.Event.OccurredAt,
				Attributes:   mapStringValue(item.Event.Attributes),
			},
		})
	}
	return containerRuntimeEventsHistoryResponse{
		ResourceID: history.ResourceID,
		Context: containerRuntimeEventStreamContextResponse{
			Runtime: history.Context.Runtime,
		},
		Items: items,
	}
}

// 当输入为空时返回 nil。
func mapStringValue(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
