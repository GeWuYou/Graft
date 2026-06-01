package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"graft/server/internal/drilldown"
	auditstore "graft/server/plugins/audit/store"
)

const (
	defaultPage        = 1
	defaultPageSize    = 20
	maxPageSize        = 200
	auditSortPartCount = 2
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
	Page                int
	PageSize            int
	Scope               string
	ActorUserID         *uint64
	Keyword             string
	Actor               string
	Action              string
	ActionPrefix        string
	ActionPrefixes      []string
	ActionKeywords      []string
	TimePreset          auditstore.AuditTimePreset
	Source              auditstore.AuditSource
	BusinessCategory    auditstore.AuditBusinessCategory
	ResourceType        string
	ResourceTypes       []string
	ResourceID          string
	ResourceName        string
	RequestPathPrefixes []string
	Success             *bool
	SessionID           string
	RequestID           string
	Result              auditstore.AuditResult
	Results             []auditstore.AuditResult
	RiskLevel           auditstore.AuditRiskLevel
	RiskLevels          []auditstore.AuditRiskLevel
	CreatedFrom         *time.Time
	CreatedTo           *time.Time
	Sorts               []string
}

// ListResult contains one page of audit records plus the total count.
type ListResult struct {
	Items              []auditstore.AuditLog
	Total              int
	Page               int
	PageSize           int
	AppliedScope       *drilldown.AppliedScope
	ScopeProjection    *drilldown.ScopeProjection
	ConvertibleFilters *drilldown.ConvertibleFilters
}

// OverviewResult contains the read model for the audit overview page.
type OverviewResult = auditstore.AuditOverview

// IncidentResult contains the audit-owned incident drilldown read model.
type IncidentResult = auditstore.AuditIncident

// Service writes and queries audit records through the plugin-owned repository boundary.
type Service struct {
	repo      auditstore.AuditRepository
	drilldown *drilldown.Service[ListQuery, ListQuery]
}

// NewService creates the audit service.
func NewService(repo auditstore.AuditRepository) (*Service, error) {
	if repo == nil {
		return nil, ErrNilAuditRepository
	}

	return &Service{repo: repo}, nil
}

// NewServiceWithDrilldown creates the audit service with an optional drilldown scope resolver.
func NewServiceWithDrilldown(
	repo auditstore.AuditRepository,
	drilldownService *drilldown.Service[ListQuery, ListQuery],
) (*Service, error) {
	service, err := NewService(repo)
	if err != nil {
		return nil, err
	}
	service.drilldown = drilldownService
	return service, nil
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

	resolvedScope, effectiveQuery, err := s.resolveScope(ctx, query)
	if err != nil {
		return ListResult{}, fmt.Errorf("resolve audit list scope: %w", err)
	}

	result, err := s.repo.ListAuditLogs(ctx, auditstore.ListAuditLogsQuery{
		ActorUserID:         effectiveQuery.ActorUserID,
		Keyword:             strings.TrimSpace(effectiveQuery.Keyword),
		Actor:               strings.TrimSpace(effectiveQuery.Actor),
		Action:              strings.TrimSpace(effectiveQuery.Action),
		ActionPrefix:        strings.TrimSpace(effectiveQuery.ActionPrefix),
		ActionPrefixes:      normalizeAuditStringFilters(effectiveQuery.ActionPrefixes),
		ActionKeywords:      normalizeAuditStringFilters(effectiveQuery.ActionKeywords),
		TimePreset:          normalizeAuditTimePreset(effectiveQuery.TimePreset),
		Source:              normalizeAuditSource(effectiveQuery.Source),
		BusinessCategory:    normalizeAuditBusinessCategory(effectiveQuery.BusinessCategory),
		ResourceType:        strings.TrimSpace(effectiveQuery.ResourceType),
		ResourceTypes:       normalizeAuditStringFilters(effectiveQuery.ResourceTypes),
		ResourceID:          strings.TrimSpace(effectiveQuery.ResourceID),
		ResourceName:        strings.TrimSpace(effectiveQuery.ResourceName),
		RequestPathPrefixes: normalizeAuditStringFilters(effectiveQuery.RequestPathPrefixes),
		Success:             effectiveQuery.Success,
		SessionID:           strings.TrimSpace(effectiveQuery.SessionID),
		RequestID:           strings.TrimSpace(effectiveQuery.RequestID),
		Result:              normalizeAuditResult(effectiveQuery.Result),
		Results:             normalizeAuditResults(effectiveQuery.Results),
		RiskLevel:           normalizeAuditRiskLevel(effectiveQuery.RiskLevel),
		RiskLevels:          normalizeAuditRiskLevels(effectiveQuery.RiskLevels),
		CreatedFrom:         normalizeAuditCreatedFrom(effectiveQuery.CreatedFrom),
		CreatedTo:           normalizeAuditCreatedTo(effectiveQuery.CreatedTo),
		Sorts:               normalizeAuditSorts(effectiveQuery.Sorts),
		Limit:               pageSize,
		Offset:              (page - 1) * pageSize,
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("list audit logs: %w", err)
	}

	listResult := ListResult{
		Items:    result.Items,
		Total:    result.Total,
		Page:     page,
		PageSize: pageSize,
	}
	if resolvedScope != nil {
		listResult.AppliedScope = &resolvedScope.Applied
		listResult.ScopeProjection = &resolvedScope.Projection
		convertible := resolvedScope.ConvertibleFilters
		listResult.ConvertibleFilters = &convertible
	}
	return listResult, nil
}

func (s *Service) resolveScope(
	ctx context.Context,
	query ListQuery,
) (*drilldown.ResolvedScope[ListQuery], ListQuery, error) {
	if s == nil || s.drilldown == nil || strings.TrimSpace(query.Scope) == "" {
		return nil, query, nil
	}

	resolved, err := s.drilldown.ResolveScope(ctx, "audit", "audit_logs", query.Scope, query)
	if err != nil {
		return nil, query, err
	}

	effectiveQuery := query
	effectiveQuery.Scope = ""
	effectiveQuery.ActionKeywords = mergeListQueryStringField(effectiveQuery.ActionKeywords, resolved.QueryPatch.ActionKeywords)
	if resolved.QueryPatch.BusinessCategory != "" {
		effectiveQuery.BusinessCategory = resolved.QueryPatch.BusinessCategory
	}
	return &resolved, effectiveQuery, nil
}

func mergeListQueryStringField(base []string, patch []string) []string {
	if len(patch) == 0 {
		return base
	}
	if len(base) == 0 {
		return append([]string(nil), patch...)
	}

	merged := append([]string(nil), base...)
	for _, value := range patch {
		exists := false
		for _, current := range merged {
			if current == value {
				exists = true
				break
			}
		}
		if !exists {
			merged = append(merged, value)
		}
	}
	return merged
}

func normalizeAuditCreatedFrom(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

func normalizeAuditCreatedTo(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

func normalizeAuditSorts(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		field, order, ok := ParseAuditSortExpressionForBinding(value)
		if !ok {
			continue
		}
		if _, exists := seen[field]; exists {
			continue
		}
		seen[field] = struct{}{}
		key := field + ":" + order
		normalized = append(normalized, key)
	}
	return normalized
}

// ParseAuditSortExpressionForBinding validates the stable `field:dir` audit sort contract.
func ParseAuditSortExpressionForBinding(value string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != auditSortPartCount {
		return "", "", false
	}
	field := strings.TrimSpace(parts[0])
	order := strings.ToLower(strings.TrimSpace(parts[1]))
	if field != "created_at" {
		return "", "", false
	}
	if order != "asc" && order != "desc" {
		return "", "", false
	}
	return field, order, true
}

func normalizeAuditSource(source auditstore.AuditSource) auditstore.AuditSource {
	switch auditstore.AuditSource(strings.ToUpper(strings.TrimSpace(string(source)))) {
	case auditstore.AuditSourceRequest:
		return auditstore.AuditSourceRequest
	case auditstore.AuditSourceSecurityEvent:
		return auditstore.AuditSourceSecurityEvent
	case auditstore.AuditSourceDomainEvent:
		return auditstore.AuditSourceDomainEvent
	default:
		return ""
	}
}

func normalizeAuditBusinessCategory(category auditstore.AuditBusinessCategory) auditstore.AuditBusinessCategory {
	switch auditstore.AuditBusinessCategory(strings.TrimSpace(string(category))) {
	case auditstore.AuditBusinessCategoryFailedOperations:
		return auditstore.AuditBusinessCategoryFailedOperations
	case auditstore.AuditBusinessCategoryHighRiskOperations:
		return auditstore.AuditBusinessCategoryHighRiskOperations
	case auditstore.AuditBusinessCategorySensitiveOperations:
		return auditstore.AuditBusinessCategorySensitiveOperations
	case auditstore.AuditBusinessCategoryAuthFailures:
		return auditstore.AuditBusinessCategoryAuthFailures
	case auditstore.AuditBusinessCategoryPermissionDenials:
		return auditstore.AuditBusinessCategoryPermissionDenials
	case auditstore.AuditBusinessCategoryRBACChanges:
		return auditstore.AuditBusinessCategoryRBACChanges
	case auditstore.AuditBusinessCategoryCriticalSecurity:
		return auditstore.AuditBusinessCategoryCriticalSecurity
	default:
		return ""
	}
}

func normalizeAuditStringFilters(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeAuditTimePreset(value auditstore.AuditTimePreset) auditstore.AuditTimePreset {
	switch auditstore.AuditTimePreset(strings.TrimSpace(string(value))) {
	case auditstore.AuditTimePresetLast24Hours:
		return auditstore.AuditTimePresetLast24Hours
	case auditstore.AuditTimePresetLast7Days:
		return auditstore.AuditTimePresetLast7Days
	case auditstore.AuditTimePresetLast30Days:
		return auditstore.AuditTimePresetLast30Days
	default:
		return ""
	}
}

func normalizeAuditOverviewTimePreset(value auditstore.AuditTimePreset) auditstore.AuditTimePreset {
	switch auditstore.AuditTimePreset(strings.TrimSpace(string(value))) {
	case auditstore.AuditTimePresetLast7Days:
		return auditstore.AuditTimePresetLast7Days
	case auditstore.AuditTimePresetLast30Days:
		return auditstore.AuditTimePresetLast30Days
	default:
		return auditstore.AuditTimePresetLast24Hours
	}
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

func normalizeAuditResults(results []auditstore.AuditResult) []auditstore.AuditResult {
	if len(results) == 0 {
		return nil
	}

	normalized := make([]auditstore.AuditResult, 0, len(results))
	for _, result := range results {
		value := normalizeAuditResult(result)
		if value == "" {
			continue
		}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
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

func normalizeAuditRiskLevels(levels []auditstore.AuditRiskLevel) []auditstore.AuditRiskLevel {
	if len(levels) == 0 {
		return nil
	}

	normalized := make([]auditstore.AuditRiskLevel, 0, len(levels))
	for _, level := range levels {
		value := normalizeAuditRiskLevel(level)
		if value == "" {
			continue
		}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

// Overview returns the aggregated overview payload for the selected window.
func (s *Service) Overview(ctx context.Context, preset auditstore.AuditTimePreset) (OverviewResult, error) {
	if s == nil || s.repo == nil {
		return OverviewResult{}, ErrAuditServiceUnavailable
	}

	return s.repo.ReadAuditOverview(ctx, normalizeAuditOverviewTimePreset(preset))
}

// Incident returns the audit-owned incident drilldown for one stable seed event.
func (s *Service) Incident(ctx context.Context, eventID uint64) (IncidentResult, error) {
	if s == nil || s.repo == nil {
		return IncidentResult{}, ErrAuditServiceUnavailable
	}
	if eventID == 0 {
		return IncidentResult{}, errors.New("audit incident event id is required")
	}

	return s.repo.ReadIncident(ctx, eventID)
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
	resolved := resolveCandidateMetadataFields(candidate, metadata)

	applyCanonicalCandidateMetadata(metadata, candidate, resolved)
	if sessionID := firstNonEmptyTrimmed(strings.TrimSpace(candidate.SessionID), stringValue(metadata["sessionId"]), stringValue(metadata["session_id"])); sessionID != "" {
		metadata["sessionId"] = sessionID
		metadata["session_id"] = sessionID
	}
	if decision.Rule != nil {
		metadata["policy_rule_id"] = decision.Rule.ID
		metadata["policy_rule_name"] = decision.Rule.Name
		metadata["policy_effect"] = decision.Rule.Effect
	}
	applyLegacyCandidateMetadataAliases(metadata, resolved)
	return metadata
}

type resolvedCandidateMetadata struct {
	requestMethod string
	requestPath   string
	requestID     string
	traceID       string
	eventType     string
	targetType    string
	targetID      string
	status        int
	actorID       string
	actorType     string
}

func resolveCandidateMetadataFields(candidate auditstore.AuditCandidate, metadata map[string]any) resolvedCandidateMetadata {
	actorID := ""
	if candidate.ActorUserID != nil {
		actorID = strconv.FormatUint(*candidate.ActorUserID, 10)
	}

	resolved := resolvedCandidateMetadata{
		requestMethod: firstNonEmptyTrimmed(strings.TrimSpace(candidate.RequestMethod), stringValue(metadata["method"]), stringValue(metadata["request_method"])),
		requestPath:   firstNonEmptyTrimmed(strings.TrimSpace(candidate.RequestPath), stringValue(metadata["route"]), stringValue(metadata["path"]), stringValue(metadata["request_path"])),
		requestID:     firstNonEmptyTrimmed(strings.TrimSpace(candidate.RequestID), stringValue(metadata["requestId"]), stringValue(metadata["request_id"])),
		eventType:     firstNonEmptyTrimmed(strings.TrimSpace(candidate.EventType), stringValue(metadata["eventType"]), stringValue(metadata["event_type"])),
		targetType:    firstNonEmptyTrimmed(strings.TrimSpace(candidate.TargetType), stringValue(metadata["targetType"]), stringValue(metadata["target_type"])),
		targetID:      firstNonEmptyTrimmed(strings.TrimSpace(candidate.ResourceID), stringValue(metadata["targetId"]), stringValue(metadata["target_id"])),
		status:        firstNonZeroInt(candidate.StatusCode, intValue(metadata["status"]), intValue(metadata["status_code"])),
		actorID:       firstNonEmptyTrimmed(actorID, stringValue(metadata["actorId"]), stringValue(metadata["actor_id"])),
		actorType:     firstNonEmptyTrimmed(stringValue(metadata["actorType"]), stringValue(metadata["actor_type"])),
	}
	resolved.traceID = firstNonEmptyTrimmed(strings.TrimSpace(candidate.TraceID), stringValue(metadata["traceId"]), stringValue(metadata["trace_id"]), resolved.requestID)
	if resolved.actorType == "" && resolved.actorID != "" {
		resolved.actorType = "user"
	}

	return resolved
}

func applyCanonicalCandidateMetadata(metadata map[string]any, candidate auditstore.AuditCandidate, resolved resolvedCandidateMetadata) {
	metadata["auditSource"] = string(candidate.Source)
	metadata["requestId"] = resolved.requestID
	metadata["traceId"] = resolved.traceID
	metadata["method"] = resolved.requestMethod
	metadata["path"] = resolved.requestPath
	metadata["route"] = resolved.requestPath
	metadata["status"] = resolved.status
	assignOptionalMetadataString(metadata, "actorId", resolved.actorID)
	assignOptionalMetadataString(metadata, "actorType", resolved.actorType)
	assignOptionalMetadataString(metadata, "eventType", resolved.eventType)
	assignOptionalMetadataString(metadata, "targetType", resolved.targetType)
	assignOptionalMetadataString(metadata, "targetId", resolved.targetID)
}

func applyLegacyCandidateMetadataAliases(metadata map[string]any, resolved resolvedCandidateMetadata) {
	metadata["audit_source"] = metadata["auditSource"]
	metadata["request_method"] = metadata["method"]
	metadata["request_path"] = metadata["path"]
	metadata["status_code"] = metadata["status"]
	metadata["trace_id"] = metadata["traceId"]
	assignOptionalMetadataString(metadata, "event_type", resolved.eventType)
	assignOptionalMetadataString(metadata, "target_type", resolved.targetType)
	assignOptionalMetadataString(metadata, "target_id", resolved.targetID)
}

func assignOptionalMetadataString(metadata map[string]any, key string, value string) {
	if value != "" {
		metadata[key] = value
	}
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

func stringValue(value any) string {
	typed, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(typed)
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func firstNonZeroInt(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
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
