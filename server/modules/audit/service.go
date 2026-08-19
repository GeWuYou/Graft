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
	maxAuditVisibilityOverrideBatch = 100
	maxAuditLogDeleteBatch           = 100
)

var (
	// ErrNilAuditRepository 表示服务构造时缺少模块自有 repository。
	ErrNilAuditRepository = errors.New("audit repository is required")
	// ErrAuditServiceUnavailable 表示服务或其 repository 依赖在运行时不可用。
	ErrAuditServiceUnavailable = errors.New("audit service is unavailable")
)

// RecordInput 描述服务边界接收的一条审计记录写入请求。
type RecordInput struct {
	IdempotencyKey   string
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

// ListQuery 描述服务层使用的审计记录分页与筛选条件。
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

// ListResult 包含一页审计记录及匹配条件的总数。
type ListResult struct {
	Items              []auditstore.AuditLog
	Total              int
	Page               int
	PageSize           int
	AppliedScope       *drilldown.AppliedScope
	ScopeProjection    *drilldown.ScopeProjection
	ConvertibleFilters *drilldown.ConvertibleFilters
}

// DetailResult 包含一条不可变的审计日志证据记录。
type DetailResult = auditstore.AuditLog

// OverviewResult 包含审计概览页面使用的读模型。
type OverviewResult = auditstore.AuditOverview

// IncidentResult 包含由审计模块拥有的事件下钻读模型。
type IncidentResult = auditstore.AuditIncident

// VisibilityPolicyResult 包含由审计模块拥有的可见性策略快照。
type VisibilityPolicyResult = auditstore.AuditVisibilityPolicySnapshot

// Service 通过模块拥有的仓储边界写入和查询审计记录。
type Service struct {
	repo      auditstore.AuditRepository
	drilldown *drilldown.Service[ListQuery, ListQuery]
}

// NewService 创建审计服务。
func NewService(repo auditstore.AuditRepository) (*Service, error) {
	if repo == nil {
		return nil, ErrNilAuditRepository
	}

	return &Service{repo: repo}, nil
}

// NewServiceWithDrilldown 创建带可选下钻范围解析器的审计服务。
// 下钻解析器只决定读取范围，不改变审计记录的写入事实。
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

// Record 规范化稳定字段并脱敏后写入一条审计记录。
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

// List 返回一页受上限约束的审计记录。
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

// Detail 按 ID 返回一条不可变的审计记录。
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

// Overview 返回所选时间窗口的聚合概览数据。
func (s *Service) Overview(ctx context.Context, preset auditstore.AuditTimePreset) (OverviewResult, error) {
	repo, err := s.repository()
	if err != nil {
		return OverviewResult{}, err
	}

	return repo.ReadAuditOverview(ctx, normalizeAuditOverviewTimePreset(preset))
}

// Incident 返回一个稳定种子事件对应的审计模块事件下钻数据。
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

// DeleteBefore 删除早于审计模块显式保留期限边界的审计记录。
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

// DeleteByIDs 原子删除一批审计记录，并要求仓储为该操作写入不可手工删除凭证。
func (s *Service) DeleteByIDs(ctx context.Context, ids []uint64, input auditstore.AuditLogDeletionInput) (int64, error) {
	repo, err := s.repository()
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 || len(ids) > maxAuditLogDeleteBatch {
		return 0, fmt.Errorf("%w: audit log delete batch must contain 1-%d ids", auditstore.ErrAuditValidation, maxAuditLogDeleteBatch)
	}
	seen := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			return 0, fmt.Errorf("%w: audit log id is required", auditstore.ErrAuditValidation)
		}
		if _, ok := seen[id]; ok {
			return 0, fmt.Errorf("%w: duplicate audit log id", auditstore.ErrAuditValidation)
		}
		seen[id] = struct{}{}
	}
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.IdempotencyKey == "" {
		return 0, fmt.Errorf("%w: audit log deletion idempotency key is required", auditstore.ErrAuditValidation)
	}
	if input.DeletedAt.IsZero() {
		input.DeletedAt = time.Now().UTC()
	}
	deleter, ok := repo.(interface {
		DeleteAuditLogsByIDs(context.Context, []uint64, auditstore.AuditLogDeletionInput) (int64, error)
	})
	if !ok {
		return 0, ErrAuditServiceUnavailable
	}
	return deleter.DeleteAuditLogsByIDs(ctx, ids, input)
}

// VisibilityPolicy 返回当前由审计模块拥有的可见性策略快照。
func (s *Service) VisibilityPolicy(ctx context.Context) (VisibilityPolicyResult, error) {
	repo, err := s.repository()
	if err != nil {
		return VisibilityPolicyResult{}, err
	}

	return s.readAuditVisibilityPolicy(ctx, repo)
}

// UpdateVisibilityDefault 更新全局审计可见性策略。
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

	normalized := normalizeAuditVisibilityStrategy(strategy)
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

// UpdateVisibilityOverride 更新一个由审计模块拥有的来源和动作可见性覆盖项。
func (s *Service) UpdateVisibilityOverride(
	ctx context.Context,
	input auditstore.UpsertAuditVisibilityOverrideInput,
) (auditstore.AuditVisibilityOverride, error) {
	repo, err := s.repository()
	if err != nil {
		return auditstore.AuditVisibilityOverride{}, err
	}

	normalizedInput, err := normalizeVisibilityOverrideInput(input)
	if err != nil {
		return auditstore.AuditVisibilityOverride{}, err
	}

	updated, err := repo.UpsertAuditVisibilityOverride(
		ctx,
		normalizedInput,
	)
	if err != nil {
		return auditstore.AuditVisibilityOverride{}, fmt.Errorf("upsert audit visibility override: %w", err)
	}
	return updated, nil
}

// UpdateVisibilityOverrides 原子更新一组审计可见性覆盖项，并保持响应顺序与请求顺序一致。
func (s *Service) UpdateVisibilityOverrides(
	ctx context.Context,
	inputs []auditstore.UpsertAuditVisibilityOverrideInput,
) ([]auditstore.AuditVisibilityOverride, error) {
	repo, err := s.repository()
	if err != nil {
		return nil, err
	}
	if len(inputs) == 0 || len(inputs) > maxAuditVisibilityOverrideBatch {
		return nil, fmt.Errorf("%w: audit visibility override batch must contain 1 to %d items", auditstore.ErrAuditValidation, maxAuditVisibilityOverrideBatch)
	}

	normalized := make([]auditstore.UpsertAuditVisibilityOverrideInput, 0, len(inputs))
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		item, normalizeErr := normalizeVisibilityOverrideInput(input)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		key := string(item.Source) + "\x00" + item.ActionKey
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("%w: duplicate audit visibility override", auditstore.ErrAuditValidation)
		}
		seen[key] = struct{}{}
		normalized = append(normalized, item)
	}

	updated, err := repo.UpsertAuditVisibilityOverrides(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("upsert audit visibility overrides: %w", err)
	}
	return updated, nil
}

// DeleteVisibilityOverride 删除一条审计模块拥有的来源加动作可见性覆盖规则。
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

// RecordCandidate 在策略评估通过后写入一条已规范化的候选审计记录。
func (s *Service) RecordCandidate(ctx context.Context, candidate auditstore.AuditCandidate) (auditstore.AuditLog, bool, error) {
	repo, err := s.repository()
	if err != nil {
		return auditstore.AuditLog{}, false, err
	}

	evaluator, err := NewPolicyEvaluator(repo)
	if err != nil {
		return auditstore.AuditLog{}, false, err
	}

	recordingPolicy, err := s.resolveCandidateRecordingPolicy(ctx, evaluator, candidate)
	if err != nil {
		return auditstore.AuditLog{}, false, err
	}
	if !recordingPolicy.shouldRecord {
		return auditstore.AuditLog{}, false, nil
	}
	candidate.Visibility = recordingPolicy.strategy

	record, err := s.Record(ctx, RecordInput{
		IdempotencyKey:   candidate.IdempotencyKey,
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
		Metadata:         candidateMetadata(candidate, recordingPolicy.decision),
		CreatedAt:        candidate.CreatedAt,
	})
	if err != nil {
		return auditstore.AuditLog{}, false, err
	}

	return record, true, nil
}
