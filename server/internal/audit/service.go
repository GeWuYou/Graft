package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	auditstore "graft/server/plugins/audit/store"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 200
)

const (
	defaultOverviewWindow = auditstore.OverviewWindow24Hours
)

// RecordInput describes one audit record write at the service boundary.
type RecordInput struct {
	ActorUserID      *uint64
	ActorUsername    string
	ActorDisplayName string
	Action           string
	ResourceType     string
	ResourceID       string
	ResourceName     string
	Success          bool
	RequestID        string
	IP               string
	UserAgent        string
	Message          string
	Metadata         any
	CreatedAt        time.Time
}

// ListQuery describes the service-layer read shape used by future API pagination/filtering.
type ListQuery struct {
	Page         int
	PageSize     int
	ActorUserID  *uint64
	Action       string
	ResourceType string
	ResourceID   string
	ResourceName string
	Success      *bool
	RequestID    string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
}

// ListResult contains one page of audit records plus the total count.
type ListResult struct {
	Items    []auditstore.AuditLog
	Total    int
	Page     int
	PageSize int
}

// OverviewResult contains the read model for the audit overview page.
type OverviewResult = auditstore.AuditOverview

// Service writes and queries audit records through the plugin-owned repository boundary.
type Service struct {
	repo auditstore.AuditRepository
}

// NewService creates the audit service.
func NewService(repo auditstore.AuditRepository) (*Service, error) {
	if repo == nil {
		return nil, errors.New("audit repository is required")
	}

	return &Service{repo: repo}, nil
}

// Record writes one audit record after normalizing stable fields and redacting sensitive data.
func (s *Service) Record(ctx context.Context, input RecordInput) (auditstore.AuditLog, error) {
	if s == nil || s.repo == nil {
		return auditstore.AuditLog{}, errors.New("audit service is unavailable")
	}

	action := strings.TrimSpace(input.Action)
	if action == "" {
		return auditstore.AuditLog{}, errors.New("audit action is required")
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}

	metadata, err := sanitizeMetadata(input.Metadata)
	if err != nil {
		return auditstore.AuditLog{}, err
	}

	return s.repo.CreateAuditLog(ctx, auditstore.CreateAuditLogInput{
		ActorUserID:      input.ActorUserID,
		ActorUsername:    strings.TrimSpace(input.ActorUsername),
		ActorDisplayName: strings.TrimSpace(input.ActorDisplayName),
		Action:           action,
		ResourceType:     strings.TrimSpace(input.ResourceType),
		ResourceID:       strings.TrimSpace(input.ResourceID),
		ResourceName:     strings.TrimSpace(input.ResourceName),
		Success:          input.Success,
		RequestID:        strings.TrimSpace(input.RequestID),
		IP:               strings.TrimSpace(input.IP),
		UserAgent:        strings.TrimSpace(input.UserAgent),
		Message:          sanitizeFreeText(strings.TrimSpace(input.Message)),
		Metadata:         metadata,
		CreatedAt:        input.CreatedAt.UTC(),
	})
}

// List returns a bounded page of audit records.
func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	if s == nil || s.repo == nil {
		return ListResult{}, errors.New("audit service is unavailable")
	}

	page := query.Page
	if page < 1 {
		page = defaultPage
	}
	pageSize := query.PageSize
	switch {
	case pageSize < 1:
		pageSize = defaultPageSize
	case pageSize > maxPageSize:
		pageSize = maxPageSize
	}

	result, err := s.repo.ListAuditLogs(ctx, auditstore.ListAuditLogsQuery{
		ActorUserID:  query.ActorUserID,
		Action:       strings.TrimSpace(query.Action),
		ResourceType: strings.TrimSpace(query.ResourceType),
		ResourceID:   strings.TrimSpace(query.ResourceID),
		ResourceName: strings.TrimSpace(query.ResourceName),
		Success:      query.Success,
		RequestID:    strings.TrimSpace(query.RequestID),
		CreatedFrom:  query.CreatedFrom,
		CreatedTo:    query.CreatedTo,
		Limit:        pageSize,
		Offset:       (page - 1) * pageSize,
	})
	if err != nil {
		return ListResult{}, err
	}

	return ListResult{
		Items:    result.Items,
		Total:    result.Total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Overview returns the aggregated overview payload for the selected window.
func (s *Service) Overview(ctx context.Context, window auditstore.OverviewWindow) (OverviewResult, error) {
	if s == nil || s.repo == nil {
		return OverviewResult{}, errors.New("audit service is unavailable")
	}

	switch strings.TrimSpace(string(window)) {
	case "", string(defaultOverviewWindow):
		window = defaultOverviewWindow
	case string(auditstore.OverviewWindow7Days):
		window = auditstore.OverviewWindow7Days
	case string(auditstore.OverviewWindow30Days):
		window = auditstore.OverviewWindow30Days
	default:
		window = defaultOverviewWindow
	}

	return s.repo.ReadAuditOverview(ctx, window)
}

func sanitizeMetadata(input any) (json.RawMessage, error) {
	if input == nil {
		return json.RawMessage([]byte("{}")), nil
	}

	payload, err := normalizeMetadataValue(input)
	if err != nil {
		return nil, fmt.Errorf("normalize metadata value: %w", err)
	}

	sanitized := sanitizeMetadataValue(payload)
	if sanitized == nil {
		sanitized = map[string]any{}
	}

	encoded, err := json.Marshal(sanitized)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata value: %w", err)
	}

	return json.RawMessage(encoded), nil
}

func normalizeMetadataValue(input any) (any, error) {
	switch typed := input.(type) {
	case json.RawMessage:
		if len(typed) == 0 {
			return map[string]any{}, nil
		}
		var decoded any
		if err := json.Unmarshal(typed, &decoded); err != nil {
			return nil, fmt.Errorf("unmarshal metadata raw message: %w", err)
		}
		return decoded, nil
	case []byte:
		if len(typed) == 0 {
			return map[string]any{}, nil
		}
		var decoded any
		if err := json.Unmarshal(typed, &decoded); err != nil {
			return nil, fmt.Errorf("unmarshal metadata bytes: %w", err)
		}
		return decoded, nil
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("marshal metadata input: %w", err)
		}
		var decoded any
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			return nil, fmt.Errorf("unmarshal metadata payload: %w", err)
		}
		return decoded, nil
	}
}
