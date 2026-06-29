package audit

import (
	"strings"
	"time"

	auditstore "graft/server/modules/audit/store"
)

// normalizeAuditUTCTime 将时间值转换为 UTC。
// 当输入为空或为零值时，返回 nil；否则返回指向 UTC 时间的新指针。
func normalizeAuditUTCTime(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

// normalizeAuditTrimmedUpper 去除首尾空白后将字符串转为大写。
func normalizeAuditTrimmedUpper(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

// normalizeAuditEnum 对字符串枚举值做归一化并限制在允许集合内。
// normalizeAuditEnum 将值归一化后限制在允许集合中；不在集合内时返回 fallback。
// allowed 参数指定可接受的枚举值集合。
func normalizeAuditEnum[T ~string](value T, fallback T, normalize func(string) string, allowed ...T) T {
	normalized := T(normalize(string(value)))
	for _, candidate := range allowed {
		if normalized == candidate {
			return candidate
		}
	}
	return fallback
}

// normalizeAuditSlice 对切片中的每个值执行归一化，并丢弃零值结果。
// normalizeAuditSlice 对切片元素逐项归一化并过滤零值。
// 当输入为空或所有项归一化后都为零值时，返回 nil。
func normalizeAuditSlice[T comparable](values []T, normalize func(T) T) []T {
	if len(values) == 0 {
		return nil
	}

	var zero T
	normalized := make([]T, 0, len(values))
	for _, value := range values {
		value = normalize(value)
		if value == zero {
			continue
		}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

// normalizeAuditCreatedFrom 将审计创建时间起始值规范化为 UTC。
// normalizeAuditCreatedFrom 将创建起始时间归一到 UTC；输入为 nil 或零值时返回 nil。
func normalizeAuditCreatedFrom(value *time.Time) *time.Time {
	return normalizeAuditUTCTime(value)
}

// normalizeAuditCreatedTo 将审计创建时间上限规范为 UTC，并在输入为空或零值时返回 nil。
func normalizeAuditCreatedTo(value *time.Time) *time.Time {
	return normalizeAuditUTCTime(value)
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
// normalizeAuditSource 将审计来源归一为允许的枚举值，仅保留 Request、SecurityEvent 和 DomainEvent。
// 无法识别的输入返回空值。
func normalizeAuditSource(source auditstore.AuditSource) auditstore.AuditSource {
	return normalizeAuditEnum(
		source,
		"",
		normalizeAuditTrimmedUpper,
		auditstore.AuditSourceRequest,
		auditstore.AuditSourceSecurityEvent,
		auditstore.AuditSourceDomainEvent,
	)
}

// normalizeAuditBusinessCategory 规范化审计业务分类值。
// normalizeAuditBusinessCategory 规范化审计业务分类，仅保留已知分类。
// 无法识别的值返回空字符串。
func normalizeAuditBusinessCategory(category auditstore.AuditBusinessCategory) auditstore.AuditBusinessCategory {
	return normalizeAuditEnum(
		category,
		"",
		strings.TrimSpace,
		auditstore.AuditBusinessCategoryFailedOperations,
		auditstore.AuditBusinessCategoryHighRiskOperations,
		auditstore.AuditBusinessCategorySensitiveOperations,
		auditstore.AuditBusinessCategoryAuthFailures,
		auditstore.AuditBusinessCategoryPermissionDenials,
		auditstore.AuditBusinessCategoryRBACChanges,
		auditstore.AuditBusinessCategoryCriticalSecurity,
	)
}

// normalizeAuditVisibilityScope 将可见性范围归一化为允许的取值，并在无法识别时返回默认值。
// normalizeAuditVisibilityScope 将可见性范围归一化为允许值。
//
// 仅保留 `All` 和 `HiddenOnly`；其他输入返回 `AuditVisibilityScopeDefault`。
func normalizeAuditVisibilityScope(scope auditstore.AuditVisibilityScope) auditstore.AuditVisibilityScope {
	return normalizeAuditEnum(
		scope,
		auditstore.AuditVisibilityScopeDefault,
		strings.TrimSpace,
		auditstore.AuditVisibilityScopeAll,
		auditstore.AuditVisibilityScopeHiddenOnly,
	)
}

// normalizeAuditVisibilityStrategy 规范化审计可见性策略，仅保留允许的策略值。
// 无法识别的输入返回空值。
func normalizeAuditVisibilityStrategy(strategy auditstore.AuditVisibilityStrategy) auditstore.AuditVisibilityStrategy {
	return normalizeAuditEnum(
		strategy,
		"",
		strings.TrimSpace,
		auditstore.AuditVisibilityStrategyVisible,
		auditstore.AuditVisibilityStrategyHidden,
		auditstore.AuditVisibilityStrategyIgnore,
	)
}

// normalizeMutableAuditVisibilityStrategy 将可见性策略归一化为仅允许可写的可见或隐藏值。
// 输入可识别时返回 `Visible` 或 `Hidden`，其余情况返回空字符串。
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

// normalizeAuditStringFilters 去除字符串筛选值两侧空白并丢弃空项；当输入为空或处理后没有有效值时返回 nil。
func normalizeAuditStringFilters(values []string) []string {
	return normalizeAuditSlice(values, strings.TrimSpace)
}

// normalizeAuditTimePreset 将审计时间预设规范化为受支持的值。
// normalizeAuditTimePreset 将时间预设归一到允许的值。
//
// @return 归一化后的时间预设；仅保留 `Last24Hours`、`Last7Days` 和 `Last30Days`，无法识别时返回空值。
func normalizeAuditTimePreset(value auditstore.AuditTimePreset) auditstore.AuditTimePreset {
	return normalizeAuditEnum(
		value,
		"",
		strings.TrimSpace,
		auditstore.AuditTimePresetLast24Hours,
		auditstore.AuditTimePresetLast7Days,
		auditstore.AuditTimePresetLast30Days,
	)
}

// normalizeAuditOverviewTimePreset 规范化审计概览时间预设。
// normalizeAuditOverviewTimePreset 将时间预设归一为概览允许的值。
//
// 当输入可识别时，保留 `Last7Days` 或 `Last30Days`；否则返回 `Last24Hours`。
func normalizeAuditOverviewTimePreset(value auditstore.AuditTimePreset) auditstore.AuditTimePreset {
	return normalizeAuditEnum(
		value,
		auditstore.AuditTimePresetLast24Hours,
		strings.TrimSpace,
		auditstore.AuditTimePresetLast7Days,
		auditstore.AuditTimePresetLast30Days,
	)
}

// normalizeAuditResult 将审计结果规范为受支持的枚举值。
// 仅保留 Success、Failed、Denied 和 Error；无法识别的值返回空字符串。
func normalizeAuditResult(result auditstore.AuditResult) auditstore.AuditResult {
	return normalizeAuditEnum(
		result,
		"",
		normalizeAuditTrimmedUpper,
		auditstore.AuditResultSuccess,
		auditstore.AuditResultFailed,
		auditstore.AuditResultDenied,
		auditstore.AuditResultError,
	)
}

// normalizeAuditResults 规范化审计结果列表并过滤无效项。
// normalizeAuditResults 归一化审计结果列表并过滤无效项；输入为空或全部无效时返回 nil。
// 结果中会丢弃无法识别的值和零值项。
func normalizeAuditResults(results []auditstore.AuditResult) []auditstore.AuditResult {
	return normalizeAuditSlice(results, normalizeAuditResult)
}

// normalizeAuditRiskLevel 将风险等级归一化为允许的值。
// 仅保留 `Low`、`Medium`、`High` 和 `Critical`；无法识别时返回空字符串。
func normalizeAuditRiskLevel(level auditstore.AuditRiskLevel) auditstore.AuditRiskLevel {
	return normalizeAuditEnum(
		level,
		"",
		normalizeAuditTrimmedUpper,
		auditstore.AuditRiskLevelLow,
		auditstore.AuditRiskLevelMedium,
		auditstore.AuditRiskLevelHigh,
		auditstore.AuditRiskLevelCritical,
	)
}

// normalizeAuditRiskLevels 规范化审计风险等级列表，并丢弃无法识别的项。
// normalizeAuditRiskLevels 规范化风险等级列表并丢弃无法识别的项。
// 当输入为空或所有项都无法识别时，返回 nil。
func normalizeAuditRiskLevels(levels []auditstore.AuditRiskLevel) []auditstore.AuditRiskLevel {
	return normalizeAuditSlice(levels, normalizeAuditRiskLevel)
}
