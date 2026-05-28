package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	auditstore "graft/server/plugins/audit/store"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 200
)

var (
	// ErrNilAuditRepository indicates the service was built without the plugin-owned repository.
	ErrNilAuditRepository = errors.New("audit repository is required")
	// ErrAuditServiceUnavailable indicates the service or its repository dependency is unavailable at runtime.
	ErrAuditServiceUnavailable = errors.New("audit service is unavailable")
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
	Result       auditstore.AuditResult
	RiskLevel    auditstore.AuditRiskLevel
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
		return nil, ErrNilAuditRepository
	}

	return &Service{repo: repo}, nil
}

// Record writes one audit record after normalizing stable fields and redacting sensitive data.
func (s *Service) Record(ctx context.Context, input RecordInput) (auditstore.AuditLog, error) {
	if s == nil || s.repo == nil {
		return auditstore.AuditLog{}, ErrAuditServiceUnavailable
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
		return ListResult{}, ErrAuditServiceUnavailable
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
		Result:       normalizeAuditResult(query.Result),
		RiskLevel:    normalizeAuditRiskLevel(query.RiskLevel),
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

func normalizeAuditResult(result auditstore.AuditResult) auditstore.AuditResult {
	switch auditstore.AuditResult(strings.ToUpper(strings.TrimSpace(string(result)))) {
	case auditstore.AuditResultSuccess:
		return auditstore.AuditResultSuccess
	case auditstore.AuditResultFailed:
		return auditstore.AuditResultFailed
	case auditstore.AuditResultDenied:
		return auditstore.AuditResultDenied
	case auditstore.AuditResultError:
		return auditstore.AuditResultError
	default:
		return ""
	}
}

func normalizeAuditRiskLevel(level auditstore.AuditRiskLevel) auditstore.AuditRiskLevel {
	switch auditstore.AuditRiskLevel(strings.ToUpper(strings.TrimSpace(string(level)))) {
	case auditstore.AuditRiskLevelLow:
		return auditstore.AuditRiskLevelLow
	case auditstore.AuditRiskLevelMedium:
		return auditstore.AuditRiskLevelMedium
	case auditstore.AuditRiskLevelHigh:
		return auditstore.AuditRiskLevelHigh
	case auditstore.AuditRiskLevelCritical:
		return auditstore.AuditRiskLevelCritical
	default:
		return ""
	}
}

// Overview returns the aggregated overview payload for the selected window.
func (s *Service) Overview(ctx context.Context, window auditstore.OverviewWindow) (OverviewResult, error) {
	if s == nil || s.repo == nil {
		return OverviewResult{}, ErrAuditServiceUnavailable
	}

	return s.repo.ReadAuditOverview(ctx, window)
}

// RecordCandidate writes one normalized candidate after policy evaluation approves it.
func (s *Service) RecordCandidate(ctx context.Context, candidate auditstore.AuditCandidate) (auditstore.AuditLog, bool, error) {
	if s == nil || s.repo == nil {
		return auditstore.AuditLog{}, false, ErrAuditServiceUnavailable
	}

	evaluator, err := NewPolicyEvaluator(s.repo)
	if err != nil {
		return auditstore.AuditLog{}, false, err
	}

	decision, err := evaluator.Evaluate(ctx, candidate)
	if err != nil {
		return auditstore.AuditLog{}, false, err
	}
	if !decision.Matched || !decision.Allowed {
		return auditstore.AuditLog{}, false, nil
	}

	record, err := s.Record(ctx, RecordInput{
		ActorUserID:      candidate.ActorUserID,
		ActorUsername:    candidate.ActorUsername,
		ActorDisplayName: candidate.ActorDisplayName,
		Action:           normalizeCandidateAction(candidate),
		ResourceType:     candidate.ResourceType,
		ResourceID:       candidate.ResourceID,
		ResourceName:     candidate.ResourceName,
		Success:          candidate.Success,
		RequestID:        candidate.RequestID,
		IP:               candidate.IP,
		UserAgent:        candidate.UserAgent,
		Message:          candidate.Message,
		Metadata:         candidateMetadata(candidate, decision),
		CreatedAt:        candidate.CreatedAt,
	})
	if err != nil {
		return auditstore.AuditLog{}, false, err
	}

	return record, true, nil
}

func normalizeCandidateAction(candidate auditstore.AuditCandidate) string {
	if eventType := strings.TrimSpace(candidate.EventType); eventType != "" {
		return eventType
	}

	return strings.TrimSpace(candidate.Action)
}

func candidateMetadata(candidate auditstore.AuditCandidate, decision auditstore.AuditPolicyDecision) any {
	metadata := decodeCandidateMetadata(candidate.Metadata)
	metadata["audit_source"] = candidate.Source
	metadata["request_method"] = strings.TrimSpace(candidate.RequestMethod)
	metadata["request_path"] = strings.TrimSpace(candidate.RequestPath)
	metadata["status_code"] = candidate.StatusCode
	if traceID := strings.TrimSpace(candidate.TraceID); traceID != "" {
		metadata["trace_id"] = traceID
	}
	if sessionID := strings.TrimSpace(candidate.SessionID); sessionID != "" {
		metadata["session_id"] = sessionID
	}
	if eventType := strings.TrimSpace(candidate.EventType); eventType != "" {
		metadata["event_type"] = eventType
	}
	if targetType := strings.TrimSpace(candidate.TargetType); targetType != "" {
		metadata["target_type"] = targetType
	}
	if decision.Rule != nil {
		metadata["policy_rule_id"] = decision.Rule.ID
		metadata["policy_rule_name"] = decision.Rule.Name
		metadata["policy_effect"] = decision.Rule.Effect
	}
	return metadata
}

func decodeCandidateMetadata(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil || metadata == nil {
		return map[string]any{}
	}

	return metadata
}

func classifyCandidateRiskLevel(candidate auditstore.AuditCandidate) auditstore.AuditRiskLevel {
	record := auditstore.AuditLog{
		Action:       normalizeCandidateAction(candidate),
		Success:      candidate.Success,
		ResourceType: candidate.ResourceType,
	}
	record.Metadata = mustMarshalMetadata(candidateMetadata(candidate, auditstore.AuditPolicyDecision{}))
	record.RequestPath = candidate.RequestPath
	record.StatusCode = candidate.StatusCode
	record.Result = classifyCandidateResult(record, decodeCandidateMetadata(record.Metadata))
	return classifyCandidateAuditRiskLevel(record)
}

func mustMarshalMetadata(value any) json.RawMessage {
	payload, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage([]byte("{}"))
	}
	return payload
}

func classifyCandidateResult(record auditstore.AuditLog, metadata map[string]any) auditstore.AuditResult {
	if record.Success {
		return auditstore.AuditResultSuccess
	}

	statusCode := record.StatusCode
	if statusCode == 0 {
		if raw, ok := metadata["status_code"]; ok {
			switch typed := raw.(type) {
			case float64:
				statusCode = int(typed)
			case int:
				statusCode = typed
			}
		}
	}
	if statusCode == http.StatusForbidden {
		return auditstore.AuditResultDenied
	}

	if errorKind, _ := metadata["error_kind"].(string); statusCode >= http.StatusInternalServerError || strings.TrimSpace(errorKind) == "system" {
		return auditstore.AuditResultError
	}
	if errorText, _ := metadata["error"].(string); strings.TrimSpace(errorText) != "" {
		return auditstore.AuditResultError
	}

	return auditstore.AuditResultFailed
}

func classifyCandidateAuditRiskLevel(record auditstore.AuditLog) auditstore.AuditRiskLevel {
	action := strings.ToLower(strings.TrimSpace(record.Action))

	if record.Result == auditstore.AuditResultError || record.Result == auditstore.AuditResultDenied {
		return auditstore.AuditRiskLevelCritical
	}
	if containsAny(action, []string{"reset_password", "update_permission", "update_role", "assign_role", "token_revoke"}) {
		return auditstore.AuditRiskLevelCritical
	}
	if record.Result == auditstore.AuditResultFailed || containsAny(action, []string{"delete", "reset", "grant", "assign", "revoke", "remove", "replace", "update_role", "update_permission"}) {
		return auditstore.AuditRiskLevelHigh
	}
	if containsAny(action, []string{"login_failed", "login", "permission", "role", "auth"}) {
		return auditstore.AuditRiskLevelMedium
	}

	return auditstore.AuditRiskLevelLow
}

func containsAny(source string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(source, keyword) {
			return true
		}
	}
	return false
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
