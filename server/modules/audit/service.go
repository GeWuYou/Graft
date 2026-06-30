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
	repo, err := s.repository()
	if err != nil {
		return auditstore.AuditLog{}, err
	}

	createInput, err := normalizeAuditRecordInput(input)
	if err != nil {
		return auditstore.AuditLog{}, err
	}

	return repo.CreateAuditLog(ctx, createInput)
}

// List returns a bounded page of audit records.
func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	repo, err := s.repository()
	if err != nil {
		return ListResult{}, err
	}
	page, pageSize := normalizeAuditPagination(query)

	resolvedScope, effectiveQuery, err := s.resolveScope(ctx, query)
	if err != nil {
		return ListResult{}, fmt.Errorf("resolve audit list scope: %w", err)
	}

	result, err := repo.ListAuditLogs(ctx, normalizedAuditListQuery(effectiveQuery, page, pageSize))
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
	repo, err := s.repository()
	if err != nil {
		return DetailResult{}, err
	}
	if id == 0 {
		return DetailResult{}, auditstore.ErrAuditLogNotFound
	}

	record, err := repo.ReadAuditLog(ctx, id)
	if err != nil {
		return DetailResult{}, fmt.Errorf("read audit log detail: %w", err)
	}
	return record, nil
}

// Overview returns the aggregated overview payload for the selected window.
func (s *Service) Overview(ctx context.Context, preset auditstore.AuditTimePreset) (OverviewResult, error) {
	repo, err := s.repository()
	if err != nil {
		return OverviewResult{}, err
	}

	return repo.ReadAuditOverview(ctx, normalizeAuditOverviewTimePreset(preset))
}

// Incident returns the audit-owned incident drilldown for one stable seed event.
func (s *Service) Incident(ctx context.Context, eventID uint64) (IncidentResult, error) {
	repo, err := s.repository()
	if err != nil {
		return IncidentResult{}, err
	}
	if eventID == 0 {
		return IncidentResult{}, errors.New("audit incident event id is required")
	}

	return repo.ReadIncident(ctx, eventID)
}

// DeleteBefore deletes audit records older than an explicit audit-owned retention cutoff.
func (s *Service) DeleteBefore(ctx context.Context, createdBefore time.Time) (int64, error) {
	repo, err := s.repository()
	if err != nil {
		return 0, err
	}
	if createdBefore.IsZero() {
		return 0, errors.New("audit log cleanup cutoff is required")
	}

	deleted, err := repo.DeleteAuditLogsBefore(ctx, createdBefore.UTC())
	if err != nil {
		return 0, fmt.Errorf("delete audit logs before cutoff: %w", err)
	}

	return deleted, nil
}

// VisibilityPolicy returns the current audit-owned visibility policy snapshot.
func (s *Service) VisibilityPolicy(ctx context.Context) (VisibilityPolicyResult, error) {
	repo, err := s.repository()
	if err != nil {
		return VisibilityPolicyResult{}, err
	}

	return s.readAuditVisibilityPolicy(ctx, repo)
}

// UpdateVisibilityDefault updates the global audit visibility strategy.
func (s *Service) UpdateVisibilityDefault(
	ctx context.Context,
	strategy auditstore.AuditVisibilityStrategy,
	userID *uint64,
	username string,
) (auditstore.AuditVisibilityDefault, error) {
	repo, err := s.repository()
	if err != nil {
		return auditstore.AuditVisibilityDefault{}, err
	}

	normalized := normalizeMutableAuditVisibilityStrategy(strategy)
	if normalized == "" {
		return auditstore.AuditVisibilityDefault{}, fmt.Errorf("%w: audit visibility default strategy is required", auditstore.ErrAuditValidation)
	}

	updated, err := repo.UpsertAuditVisibilityDefault(
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
	repo, err := s.repository()
	if err != nil {
		return auditstore.AuditVisibilityOverride{}, err
	}

	normalizedSource, normalizedActionKey, err := normalizeVisibilityOverrideRef(input.Source, input.ActionKey)
	if err != nil {
		return auditstore.AuditVisibilityOverride{}, err
	}
	normalizedStrategy := normalizeAuditVisibilityStrategy(input.Strategy)
	if normalizedStrategy == "" {
		return auditstore.AuditVisibilityOverride{}, fmt.Errorf("%w: audit visibility override strategy is required", auditstore.ErrAuditValidation)
	}

	updated, err := repo.UpsertAuditVisibilityOverride(
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
	repo, err := s.repository()
	if err != nil {
		return err
	}

	normalizedSource, normalizedActionKey, err := normalizeVisibilityOverrideRef(source, actionKey)
	if err != nil {
		return err
	}
	if err := repo.DeleteAuditVisibilityOverride(ctx, normalizedSource, normalizedActionKey); err != nil {
		return fmt.Errorf("delete audit visibility override: %w", err)
	}
	return nil
}

// RecordCandidate writes one normalized candidate after policy evaluation approves it.
func (s *Service) RecordCandidate(ctx context.Context, candidate auditstore.AuditCandidate) (auditstore.AuditLog, bool, error) {
	repo, err := s.repository()
	if err != nil {
		return auditstore.AuditLog{}, false, err
	}

	evaluator, err := NewPolicyEvaluator(repo)
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
