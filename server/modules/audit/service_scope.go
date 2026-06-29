package audit

import (
	"context"
	"strings"

	"graft/server/internal/drilldown"
)

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

// mergeListQueryStringField 合并两个字符串切片并去重，保留原有顺序。
// 返回合并后的字符串切片。base 先保留原顺序，patch 中已存在于结果中的值不会重复追加。
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
