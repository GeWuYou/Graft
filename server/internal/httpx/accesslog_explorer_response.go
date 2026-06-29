package httpx

import "time"

type accessLogListResponse struct {
	Items    []accessLogDetailResponse `json:"items"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

type accessLogDetailResponse struct {
	ID           uint64  `json:"id"`
	RequestID    string  `json:"request_id"`
	TraceID      string  `json:"trace_id"`
	Method       string  `json:"method"`
	Path         string  `json:"path"`
	Route        string  `json:"route"`
	StatusCode   int     `json:"status_code"`
	DurationMS   int64   `json:"duration_ms"`
	ClientIP     string  `json:"client_ip"`
	UserAgent    string  `json:"user_agent"`
	UserID       *uint64 `json:"user_id,omitempty"`
	Username     string  `json:"username"`
	RequestSize  *int64  `json:"request_size,omitempty"`
	ResponseSize *int64  `json:"response_size,omitempty"`
	StartedAt    string  `json:"started_at"`
	OccurredAt   string  `json:"occurred_at"`
}

func toAccessLogListResponse(result AccessLogListResult) accessLogListResponse {
	return accessLogListResponse{
		Items:    toAccessLogDetailResponses(result.Items),
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}
}

func toAccessLogDetailResponses(records []AccessLog) []accessLogDetailResponse {
	items := make([]accessLogDetailResponse, 0, len(records))
	for _, record := range records {
		items = append(items, toAccessLogDetailResponse(record))
	}
	return items
}

func toAccessLogDetailResponse(record AccessLog) accessLogDetailResponse {
	return accessLogDetailResponse{
		ID:           record.ID,
		RequestID:    record.RequestID,
		TraceID:      record.TraceID,
		Method:       record.Method,
		Path:         record.Path,
		Route:        record.Route,
		StatusCode:   record.StatusCode,
		DurationMS:   record.DurationMS,
		ClientIP:     record.ClientIP,
		UserAgent:    record.UserAgent,
		UserID:       cloneUint64Pointer(record.UserID),
		Username:     record.Username,
		RequestSize:  cloneInt64Pointer(record.RequestSize),
		ResponseSize: cloneInt64Pointer(record.ResponseSize),
		StartedAt:    record.StartedAt.UTC().Format(time.RFC3339),
		OccurredAt:   record.OccurredAt.UTC().Format(time.RFC3339),
	}
}
