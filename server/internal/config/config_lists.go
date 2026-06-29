package config

import "strings"

// parseLocaleList 将逗号分隔的地域字符串解析为规范化的地域值。
// 结果已去除首尾空白、空项和重复值。
func parseLocaleList(raw string) []string {
	items, _ := normalizeLocaleList(strings.Split(raw, ","))
	return items
}

// parseCommaSeparatedList parses a comma-separated string into a normalized slice, trimming whitespace, removing empty values, and deduplicating.
func parseCommaSeparatedList(raw string) []string {
	return normalizeStringList(strings.Split(raw, ","))
}

// parseModuleList parses a comma-separated string into a normalized list of module identifiers.
func parseModuleList(raw string) []string {
	items, _ := normalizeModuleList(strings.Split(raw, ","))
	return items
}

// normalizeStringList removes empty strings and deduplicates the input list while preserving the order of first occurrence.
func normalizeStringList(items []string) []string {
	normalized := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, raw := range items {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}
