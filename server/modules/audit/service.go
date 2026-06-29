package audit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/drilldown"
	auditstore "graft/server/modules/audit/store"
)

const (
	defaultPage                     = 1
	defaultPageSize                 = 20
	maxPageSize                     = 200
	auditSortPartCount              = 2
	auditVisibilityGlobalDefaultKey = "global"
)

var (
	// ErrNilAuditRepository indicates the service was built without the module-owned repository.
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
	Visibility       auditstore.AuditVisibilityStrategy
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
	VisibilityScope     auditstore.AuditVisibilityScope
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

// DetailResult contains one immutable audit log evidence record.
type DetailResult = auditstore.AuditLog

// OverviewResult contains the read model for the audit overview page.
type OverviewResult = auditstore.AuditOverview

// IncidentResult contains the audit-owned incident drilldown read model.
type IncidentResult = auditstore.AuditIncident

// VisibilityPolicyResult contains the audit-owned visibility policy snapshot.
type VisibilityPolicyResult = auditstore.AuditVisibilityPolicySnapshot

// Service writes and queries audit records through the module-owned repository boundary.
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
		Visibility:       normalizeAuditVisibilityStrategy(input.Visibility),
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
		VisibilityScope:     normalizeAuditVisibilityScope(effectiveQuery.VisibilityScope),
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

// Detail returns one immutable audit record by id.
func (s *Service) Detail(ctx context.Context, id uint64) (DetailResult, error) {
	if s == nil || s.repo == nil {
		return DetailResult{}, ErrAuditServiceUnavailable
	}
	if id == 0 {
		return DetailResult{}, auditstore.ErrAuditLogNotFound
	}

	record, err := s.repo.ReadAuditLog(ctx, id)
	if err != nil {
		return DetailResult{}, fmt.Errorf("read audit log detail: %w", err)
	}
	return record, nil
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

// DeleteBefore deletes audit records older than an explicit audit-owned retention cutoff.
func (s *Service) DeleteBefore(ctx context.Context, createdBefore time.Time) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, ErrAuditServiceUnavailable
	}
	if createdBefore.IsZero() {
		return 0, errors.New("audit log cleanup cutoff is required")
	}

	deleted, err := s.repo.DeleteAuditLogsBefore(ctx, createdBefore.UTC())
	if err != nil {
		return 0, fmt.Errorf("delete audit logs before cutoff: %w", err)
	}

	return deleted, nil
}

// VisibilityPolicy returns the current audit-owned visibility policy snapshot.
func (s *Service) VisibilityPolicy(ctx context.Context) (VisibilityPolicyResult, error) {
	if s == nil || s.repo == nil {
		return VisibilityPolicyResult{}, ErrAuditServiceUnavailable
	}

	defaultValue, err := s.repo.GetAuditVisibilityDefault(ctx, auditVisibilityGlobalDefaultKey)
	if err != nil {
		return VisibilityPolicyResult{}, fmt.Errorf("read audit visibility default: %w", err)
	}
	overrides, err := s.repo.ListAuditVisibilityOverrides(ctx)
	if err != nil {
		return VisibilityPolicyResult{}, fmt.Errorf("list audit visibility overrides: %w", err)
	}

	return auditstore.AuditVisibilityPolicySnapshot{
		Default:   defaultValue,
		Overrides: overrides,
		Catalog:   buildAuditEventCatalog(defaultValue.Strategy, overrides),
	}, nil
}

// UpdateVisibilityDefault updates the global audit visibility strategy.
func (s *Service) UpdateVisibilityDefault(
	ctx context.Context,
	strategy auditstore.AuditVisibilityStrategy,
	userID *uint64,
	username string,
) (auditstore.AuditVisibilityDefault, error) {
	if s == nil || s.repo == nil {
		return auditstore.AuditVisibilityDefault{}, ErrAuditServiceUnavailable
	}

	normalized := normalizeMutableAuditVisibilityStrategy(strategy)
	if normalized == "" {
		return auditstore.AuditVisibilityDefault{}, fmt.Errorf("%w: audit visibility default strategy is required", auditstore.ErrAuditValidation)
	}

	updated, err := s.repo.UpsertAuditVisibilityDefault(
		ctx,
		auditVisibilityGlobalDefaultKey,
		normalized,
		userID,
		strings.TrimSpace(username),
	)
	if err != nil {
		return auditstore.AuditVisibilityDefault{}, fmt.Errorf("upsert audit visibility default: %w", err)
	}
	return updated, nil
}

// UpdateVisibilityOverride updates one audit-owned source+action visibility override.
func (s *Service) UpdateVisibilityOverride(
	ctx context.Context,
	input auditstore.UpsertAuditVisibilityOverrideInput,
) (auditstore.AuditVisibilityOverride, error) {
	if s == nil || s.repo == nil {
		return auditstore.AuditVisibilityOverride{}, ErrAuditServiceUnavailable
	}

	normalizedSource := normalizeAuditSource(input.Source)
	if normalizedSource == "" {
		return auditstore.AuditVisibilityOverride{}, fmt.Errorf("%w: audit visibility override source is required", auditstore.ErrAuditValidation)
	}
	normalizedActionKey := strings.TrimSpace(input.ActionKey)
	if normalizedActionKey == "" {
		return auditstore.AuditVisibilityOverride{}, fmt.Errorf("%w: audit visibility override action key is required", auditstore.ErrAuditValidation)
	}
	normalizedStrategy := normalizeAuditVisibilityStrategy(input.Strategy)
	if normalizedStrategy == "" {
		return auditstore.AuditVisibilityOverride{}, fmt.Errorf("%w: audit visibility override strategy is required", auditstore.ErrAuditValidation)
	}

	updated, err := s.repo.UpsertAuditVisibilityOverride(
		ctx,
		auditstore.UpsertAuditVisibilityOverrideInput{
			Source:      normalizedSource,
			ActionKey:   normalizedActionKey,
			Strategy:    normalizedStrategy,
			Description: strings.TrimSpace(input.Description),
			Actor: auditstore.AuditVisibilityActor{
				UserID:   input.Actor.UserID,
				Username: strings.TrimSpace(input.Actor.Username),
			},
		},
	)
	if err != nil {
		return auditstore.AuditVisibilityOverride{}, fmt.Errorf("upsert audit visibility override: %w", err)
	}
	return updated, nil
}

// DeleteVisibilityOverride removes one audit-owned source+action visibility override.
func (s *Service) DeleteVisibilityOverride(ctx context.Context, source auditstore.AuditSource, actionKey string) error {
	if s == nil || s.repo == nil {
		return ErrAuditServiceUnavailable
	}

	normalizedSource := normalizeAuditSource(source)
	if normalizedSource == "" {
		return fmt.Errorf("%w: audit visibility override source is required", auditstore.ErrAuditValidation)
	}
	normalizedActionKey := strings.TrimSpace(actionKey)
	if normalizedActionKey == "" {
		return fmt.Errorf("%w: audit visibility override action key is required", auditstore.ErrAuditValidation)
	}
	if err := s.repo.DeleteAuditVisibilityOverride(ctx, normalizedSource, normalizedActionKey); err != nil {
		return fmt.Errorf("delete audit visibility override: %w", err)
	}
	return nil
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

	strategy, err := s.resolveCandidateVisibilityStrategy(ctx, candidate)
	if err != nil {
		return auditstore.AuditLog{}, false, err
	}
	if strategy == auditstore.AuditVisibilityStrategyIgnore {
		return auditstore.AuditLog{}, false, nil
	}
	candidate.Visibility = strategy

	record, err := s.Record(ctx, RecordInput{
		ActorUserID:      candidate.ActorUserID,
		ActorUsername:    candidate.ActorUsername,
		ActorDisplayName: candidate.ActorDisplayName,
		Action:           normalizeCandidateAction(candidate),
		Visibility:       candidate.Visibility,
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
