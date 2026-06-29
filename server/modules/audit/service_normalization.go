package audit

import (
	"strings"
	"time"

	auditstore "graft/server/modules/audit/store"
)

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

// normalizeAuditBusinessCategory 规范化审计业务分类。
// 返回已知的业务分类值；未识别时返回空字符串。
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

// normalizeAuditVisibilityScope 将可见性范围归一化为允许的取值之一。
func normalizeAuditVisibilityScope(scope auditstore.AuditVisibilityScope) auditstore.AuditVisibilityScope {
	switch auditstore.AuditVisibilityScope(strings.TrimSpace(string(scope))) {
	case auditstore.AuditVisibilityScopeAll:
		return auditstore.AuditVisibilityScopeAll
	case auditstore.AuditVisibilityScopeHiddenOnly:
		return auditstore.AuditVisibilityScopeHiddenOnly
	default:
		return auditstore.AuditVisibilityScopeDefault
	}
}

// normalizeAuditVisibilityStrategy 归一化审计可见性策略。
// 返回允许的可见性策略值之一；如果输入无效，则返回空值。
func normalizeAuditVisibilityStrategy(strategy auditstore.AuditVisibilityStrategy) auditstore.AuditVisibilityStrategy {
	switch auditstore.AuditVisibilityStrategy(strings.TrimSpace(string(strategy))) {
	case auditstore.AuditVisibilityStrategyVisible:
		return auditstore.AuditVisibilityStrategyVisible
	case auditstore.AuditVisibilityStrategyHidden:
		return auditstore.AuditVisibilityStrategyHidden
	case auditstore.AuditVisibilityStrategyIgnore:
		return auditstore.AuditVisibilityStrategyIgnore
	default:
		return ""
	}
}

// normalizeMutableAuditVisibilityStrategy 规范化可修改的审计可见性策略。
// 仅保留可写入的可见和隐藏策略。
func normalizeMutableAuditVisibilityStrategy(
	strategy auditstore.AuditVisibilityStrategy,
) auditstore.AuditVisibilityStrategy {
	switch normalizeAuditVisibilityStrategy(strategy) {
	case auditstore.AuditVisibilityStrategyVisible:
		return auditstore.AuditVisibilityStrategyVisible
	case auditstore.AuditVisibilityStrategyHidden:
		return auditstore.AuditVisibilityStrategyHidden
	default:
		return ""
	}
}

// normalizeAuditStringFilters 去除字符串筛选值两侧空白并丢弃空项。
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
