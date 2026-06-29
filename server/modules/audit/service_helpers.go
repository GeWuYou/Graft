package audit

import (
	"context"
	"fmt"
	"strings"
	"time"

	auditstore "graft/server/modules/audit/store"
)

func (s *Service) repository() (auditstore.AuditRepository, error) {
	if s == nil || s.repo == nil {
		return nil, ErrAuditServiceUnavailable
	}
	return s.repo, nil
}

// normalizeAuditPagination 规范列表分页参数，返回有效的页码和每页数量。
// 当页码小于 1 时使用默认页码；当每页数量小于 1 时使用默认值，大于上限时截断为最大值。
func normalizeAuditPagination(query ListQuery) (page int, pageSize int) {
	page = query.Page
	if page < 1 {
		page = defaultPage
	}

	pageSize = query.PageSize
	switch {
	case pageSize < 1:
		pageSize = defaultPageSize
	case pageSize > maxPageSize:
		pageSize = maxPageSize
	}

	return page, pageSize
}

// normalizedAuditListQuery 将查询条件归一化为审计日志列表查询。
// 它会修剪字符串筛选条件、规范化枚举和列表型过滤项，并设置分页偏移与数量。
func normalizedAuditListQuery(query ListQuery, page int, pageSize int) auditstore.ListAuditLogsQuery {
	return auditstore.ListAuditLogsQuery{
		VisibilityScope:     normalizeAuditVisibilityScope(query.VisibilityScope),
		ActorUserID:         query.ActorUserID,
		Keyword:             strings.TrimSpace(query.Keyword),
		Actor:               strings.TrimSpace(query.Actor),
		Action:              strings.TrimSpace(query.Action),
		ActionPrefix:        strings.TrimSpace(query.ActionPrefix),
		ActionPrefixes:      normalizeAuditStringFilters(query.ActionPrefixes),
		ActionKeywords:      normalizeAuditStringFilters(query.ActionKeywords),
		TimePreset:          normalizeAuditTimePreset(query.TimePreset),
		Source:              normalizeAuditSource(query.Source),
		BusinessCategory:    normalizeAuditBusinessCategory(query.BusinessCategory),
		ResourceType:        strings.TrimSpace(query.ResourceType),
		ResourceTypes:       normalizeAuditStringFilters(query.ResourceTypes),
		ResourceID:          strings.TrimSpace(query.ResourceID),
		ResourceName:        strings.TrimSpace(query.ResourceName),
		RequestPathPrefixes: normalizeAuditStringFilters(query.RequestPathPrefixes),
		Success:             query.Success,
		SessionID:           strings.TrimSpace(query.SessionID),
		RequestID:           strings.TrimSpace(query.RequestID),
		Result:              normalizeAuditResult(query.Result),
		Results:             normalizeAuditResults(query.Results),
		RiskLevel:           normalizeAuditRiskLevel(query.RiskLevel),
		RiskLevels:          normalizeAuditRiskLevels(query.RiskLevels),
		CreatedFrom:         normalizeAuditCreatedFrom(query.CreatedFrom),
		CreatedTo:           normalizeAuditCreatedTo(query.CreatedTo),
		Sorts:               normalizeAuditSorts(query.Sorts),
		Limit:               pageSize,
		Offset:              (page - 1) * pageSize,
	}
}

// normalizeVisibilityOverrideRef 归一化并校验可见性覆盖引用。
// 它会规范化来源并裁剪动作键，且在任一字段缺失时返回校验错误。
func normalizeVisibilityOverrideRef(
	source auditstore.AuditSource,
	actionKey string,
) (auditstore.AuditSource, string, error) {
	normalizedSource := normalizeAuditSource(source)
	if normalizedSource == "" {
		return "", "", fmt.Errorf("%w: audit visibility override source is required", auditstore.ErrAuditValidation)
	}

	normalizedActionKey := strings.TrimSpace(actionKey)
	if normalizedActionKey == "" {
		return "", "", fmt.Errorf("%w: audit visibility override action key is required", auditstore.ErrAuditValidation)
	}

	return normalizedSource, normalizedActionKey, nil
}

// normalizeAuditRecordInput 将审计记录输入规范化为创建参数。
// 它会校验并裁剪操作名，清洗元数据，补充默认创建时间，并对文本字段做裁剪与内容清理。
// @param input 审计记录输入。
// @returns 规范化后的创建审计日志输入，以及在操作名缺失或元数据清洗失败时返回的错误。
func normalizeAuditRecordInput(input RecordInput) (auditstore.CreateAuditLogInput, error) {
	action := strings.TrimSpace(input.Action)
	if action == "" {
		return auditstore.CreateAuditLogInput{}, fmt.Errorf("audit action is required")
	}

	metadata, err := sanitizeMetadata(input.Metadata)
	if err != nil {
		return auditstore.CreateAuditLogInput{}, err
	}

	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return auditstore.CreateAuditLogInput{
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
		CreatedAt:        createdAt.UTC(),
	}, nil
}

func (s *Service) readAuditVisibilityPolicy(
	ctx context.Context,
	repo auditstore.AuditRepository,
) (VisibilityPolicyResult, error) {
	defaultValue, err := repo.GetAuditVisibilityDefault(ctx, auditVisibilityGlobalDefaultKey)
	if err != nil {
		return VisibilityPolicyResult{}, fmt.Errorf("read audit visibility default: %w", err)
	}
	overrides, err := repo.ListAuditVisibilityOverrides(ctx)
	if err != nil {
		return VisibilityPolicyResult{}, fmt.Errorf("list audit visibility overrides: %w", err)
	}

	return auditstore.AuditVisibilityPolicySnapshot{
		Default:   defaultValue,
		Overrides: overrides,
		Catalog:   buildAuditEventCatalog(defaultValue.Strategy, overrides),
	}, nil
}
