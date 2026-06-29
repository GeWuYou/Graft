package config

import "strings"

// parseLocaleList 将逗号分隔的地域字符串解析为规范化的地域值。
// parseLocaleList 将逗号分隔的区域设置字符串解析为规范化列表。
// 结果会去除首尾空白、空项和重复值。
func parseLocaleList(raw string) []string {
	items, _ := normalizeLocaleList(strings.Split(raw, ","))
	return items
}

// parseCommaSeparatedList 将逗号分隔的字符串解析为规范化切片。
// 结果会去除首尾空白、跳过空项，并按首次出现顺序去重。
func parseCommaSeparatedList(raw string) []string {
	return normalizeStringList(strings.Split(raw, ","))
}

// parseModuleList 将逗号分隔的模块标识字符串解析为规范化列表。
func parseModuleList(raw string) []string {
	items, _ := normalizeModuleList(strings.Split(raw, ","))
	return items
}

// normalizeStringList 规范化字符串列表，去除首尾空白、空项，并按首次出现顺序去重。
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
