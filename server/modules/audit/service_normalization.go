package audit

import (
	"strings"
	"time"

	auditstore "graft/server/modules/audit/store"
)

// normalizeAuditCreatedFrom 将审计创建时间起始值规范化为 UTC。
// 当输入为 nil 或零值时间时返回 nil。
func normalizeAuditCreatedFrom(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

// normalizeAuditCreatedTo 将审计创建时间上限规范为 UTC，并在输入为空或零值时返回 nil。
func normalizeAuditCreatedTo(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

// normalizeAuditSorts 规范化审计排序表达式列表。
// 它会去除首尾空白、忽略无效项，并按字段去重后返回标准化的排序键。
// @returns 标准化后的排序表达式列表；当输入切片长度为 0 时返回 nil。
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

// ParseAuditSortExpressionForBinding 验证审计排序表达式并提取字段与排序方向。
// @return 第一个返回值为排序字段，第二个返回值为排序方向，第三个返回值在表达式有效时为 `true`，否则为 `false`。
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

// normalizeAuditSource 将审计来源归一到受支持的枚举值。
// 仅保留 Request、SecurityEvent 和 DomainEvent；无法识别的输入返回空值。
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
// normalizeAuditBusinessCategory 规范化审计业务分类值。
// 返回匹配的已知业务分类；无法识别时返回空字符串。
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

// normalizeAuditVisibilityScope 将可见性范围归一化为允许的取值，并在无法识别时返回默认值。
// 仅接受 `All` 和 `HiddenOnly`，其他输入返回 `AuditVisibilityScopeDefault`。
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
// normalizeAuditVisibilityStrategy 规范化审计可见性策略，并返回允许的策略值；无法识别时返回空值。
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
// normalizeMutableAuditVisibilityStrategy 将可见性策略归一化为仅允许可写的可见或隐藏值。
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

// normalizeAuditTimePreset 将审计时间预设规范化为受支持的值。
// 返回已识别的时间预设；未识别时返回空值。
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

// normalizeAuditOverviewTimePreset 规范化审计概览时间预设。
// 当输入为可识别值时，保留 `Last7Days` 或 `Last30Days`；否则返回 `Last24Hours`。
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

// normalizeAuditResult 将审计结果规范为受支持的枚举值。
// 仅保留 Success、Failed、Denied 和 Error；无法识别的值返回空字符串。
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

// normalizeAuditResults 规范化审计结果列表并过滤无效项。
// 返回规范化后的结果列表；当输入为空或所有项都无效时返回 nil。
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

// normalizeAuditRiskLevel 将风险等级规范化为受支持的值。
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

// normalizeAuditRiskLevels 规范化审计风险等级列表，并丢弃无法识别的项。
// 当输入为空或所有项都无法识别时，返回 nil。
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
