// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"context"
	"fmt"
	"strings"

	"graft/server/internal/drilldown"
	auditstore "graft/server/modules/audit/store"
)

type auditScopeResolver struct{}

func newAuditScopeResolver() auditScopeResolver {
	return auditScopeResolver{}
}

type scopeDefinition struct {
	category auditstore.AuditBusinessCategory
	labelKey string
}

func (r auditScopeResolver) Resolve(
	_ context.Context,
	metadata drilldown.ScopeMetadata,
	currentQuery ListQuery,
) (drilldown.ResolvedScope[ListQuery], error) {
	scope := strings.TrimSpace(metadata.Scope)
	definitions := map[string]scopeDefinition{
		string(auditstore.AuditBusinessCategoryFailedOperations):    {category: auditstore.AuditBusinessCategoryFailedOperations, labelKey: "audit.logList.businessCategory.failedOperations"},
		string(auditstore.AuditBusinessCategoryHighRiskOperations):  {category: auditstore.AuditBusinessCategoryHighRiskOperations, labelKey: "audit.logList.businessCategory.highRiskOperations"},
		string(auditstore.AuditBusinessCategorySensitiveOperations): {category: auditstore.AuditBusinessCategorySensitiveOperations, labelKey: "audit.logList.businessCategory.sensitiveOperations"},
		string(auditstore.AuditBusinessCategoryAuthFailures):        {category: auditstore.AuditBusinessCategoryAuthFailures, labelKey: "audit.logList.businessCategory.authFailures"},
		string(auditstore.AuditBusinessCategoryPermissionDenials):   {category: auditstore.AuditBusinessCategoryPermissionDenials, labelKey: "audit.logList.businessCategory.permissionDenials"},
		string(auditstore.AuditBusinessCategoryRBACChanges):         {category: auditstore.AuditBusinessCategoryRBACChanges, labelKey: "audit.logList.businessCategory.rbacChanges"},
		string(auditstore.AuditBusinessCategoryCriticalSecurity):    {category: auditstore.AuditBusinessCategoryCriticalSecurity, labelKey: "audit.logList.businessCategory.criticalSecurity"},
	}

	definition, ok := definitions[scope]
	if !ok {
		return drilldown.ResolvedScope[ListQuery]{}, drilldown.ErrScopeNotFound
	}

	if currentQuery.BusinessCategory != "" {
		return drilldown.ResolvedScope[ListQuery]{}, fmt.Errorf("%w: business_category", drilldown.ErrScopeConflict)
	}

	return drilldown.ResolvedScope[ListQuery]{
		Metadata: metadata,
		Applied: drilldown.AppliedScope{
			Module:      metadata.Module,
			Scope:       metadata.Scope,
			Name:        metadata.Name,
			Description: metadata.Description,
			OwnedFields: []string{"business_category"},
		},
		Projection: drilldown.ScopeProjection{
			Title:       metadata.Name,
			Description: metadata.Description,
			Items: []drilldown.ScopeProjectionItem{
				{
					Key:      "business_category",
					LabelKey: definition.labelKey,
					Kind:     "enum",
					Values:   []string{string(definition.category)},
					Locked:   true,
				},
			},
		},
		ConvertibleFilters: drilldown.ConvertibleFilters{
			Preset:           strings.TrimSpace(string(currentQuery.TimePreset)),
			BusinessCategory: string(definition.category),
		},
		QueryPatch: ListQuery{
			BusinessCategory: definition.category,
		},
	}, nil
}
