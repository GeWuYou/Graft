package audit

import (
	"context"
	"fmt"
	"slices"
	"strings"

	auditstore "graft/server/modules/audit/store"
)

type auditEventCatalogSeed struct {
	Source         auditstore.AuditSource
	ActionKey      string
	DisplayName    string
	DescriptionKey string
	Description    string
	Category       string
}

// buildAuditEventCatalog 构建审计事件可见性目录，并合并全局默认策略与覆盖项。
// 结果按分类、来源和 actionKey 稳定排序。
func buildAuditEventCatalog(
	defaultStrategy auditstore.AuditVisibilityStrategy,
	overrides []auditstore.AuditVisibilityOverride,
) []auditstore.AuditEventCatalogItem {
	seeds := appendAuditEventCatalogSeeds()

	overrideMap := make(map[string]auditstore.AuditVisibilityOverride, len(overrides))
	for _, item := range overrides {
		overrideMap[string(item.Source)+"|"+strings.TrimSpace(item.ActionKey)] = item
	}

	items := make([]auditstore.AuditEventCatalogItem, 0, len(seeds)+len(overrides))
	seen := make(map[string]struct{}, len(seeds)+len(overrides))
	for _, seed := range seeds {
		appendAuditEventCatalogItem(&items, seen, overrideMap, defaultStrategy, seed)
	}
	for _, override := range overrides {
		appendAuditEventCatalogItem(&items, seen, overrideMap, defaultStrategy, auditEventCatalogSeed{
			Source:      override.Source,
			ActionKey:   override.ActionKey,
			DisplayName: override.ActionKey,
			Description: override.Description,
			Category:    "custom",
		})
	}

	slices.SortStableFunc(items, func(a, b auditstore.AuditEventCatalogItem) int {
		switch {
		case a.Category < b.Category:
			return -1
		case a.Category > b.Category:
			return 1
		case a.Source < b.Source:
			return -1
		case a.Source > b.Source:
			return 1
		default:
			return strings.Compare(a.ActionKey, b.ActionKey)
		}
	})
	return items
}

// appendAuditEventCatalogSeeds 返回审计可见性目录的内置种子项列表。
// 这些种子项用于构建默认的事件目录条目。
func appendAuditEventCatalogSeeds() []auditEventCatalogSeed {
	return []auditEventCatalogSeed{
		{Source: auditstore.AuditSourceSecurityEvent, ActionKey: "auth.token.expired", DisplayName: "auth.token.expired", DescriptionKey: "audit.visibilityCatalog.auth.tokenExpired.description", Description: "Access token expired security event.", Category: "auth"},
		{Source: auditstore.AuditSourceSecurityEvent, ActionKey: "auth.token.invalid", DisplayName: "auth.token.invalid", DescriptionKey: "audit.visibilityCatalog.auth.tokenInvalid.description", Description: "Access token invalid security event.", Category: "auth"},
		{Source: auditstore.AuditSourceSecurityEvent, ActionKey: "auth.token.missing", DisplayName: "auth.token.missing", DescriptionKey: "audit.visibilityCatalog.auth.tokenMissing.description", Description: "Access token missing security event.", Category: "auth"},
		{Source: auditstore.AuditSourceSecurityEvent, ActionKey: "auth.permission.denied", DisplayName: "auth.permission.denied", DescriptionKey: "audit.visibilityCatalog.auth.permissionDenied.description", Description: "Authorization denied security event.", Category: "auth"},
		{Source: auditstore.AuditSourceRequest, ActionKey: "POST /api/auth/login", DisplayName: "POST /api/auth/login", DescriptionKey: "audit.visibilityCatalog.auth.login.description", Description: "Login request audit record.", Category: "auth"},
		{Source: auditstore.AuditSourceRequest, ActionKey: "POST /api/auth/refresh", DisplayName: "POST /api/auth/refresh", DescriptionKey: "audit.visibilityCatalog.auth.refresh.description", Description: "Refresh-token rotation request audit record.", Category: "auth"},
		{Source: auditstore.AuditSourceRequest, ActionKey: "POST /api/auth/logout", DisplayName: "POST /api/auth/logout", DescriptionKey: "audit.visibilityCatalog.auth.logout.description", Description: "Logout request audit record.", Category: "auth"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "user.create", DisplayName: "user.create", DescriptionKey: "audit.visibilityCatalog.user.create.description", Description: "Managed-user creation event.", Category: "user"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "user.update", DisplayName: "user.update", DescriptionKey: "audit.visibilityCatalog.user.update.description", Description: "Managed-user update event.", Category: "user"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "user.status.update", DisplayName: "user.status.update", DescriptionKey: "audit.visibilityCatalog.user.statusUpdate.description", Description: "Managed-user status change event.", Category: "user"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "user.delete", DisplayName: "user.delete", DescriptionKey: "audit.visibilityCatalog.user.delete.description", Description: "Managed-user deletion event.", Category: "user"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "user.password.reset", DisplayName: "user.password.reset", DescriptionKey: "audit.visibilityCatalog.user.passwordReset.description", Description: "Managed-user password reset event.", Category: "user"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "rbac.role.create", DisplayName: "rbac.role.create", DescriptionKey: "audit.visibilityCatalog.rbac.role.create.description", Description: "RBAC role creation event.", Category: "rbac"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "rbac.role.update", DisplayName: "rbac.role.update", DescriptionKey: "audit.visibilityCatalog.rbac.role.update.description", Description: "RBAC role update event.", Category: "rbac"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "rbac.role.status.update", DisplayName: "rbac.role.status.update", DescriptionKey: "audit.visibilityCatalog.rbac.role.statusUpdate.description", Description: "RBAC role status change event.", Category: "rbac"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "rbac.role.delete", DisplayName: "rbac.role.delete", DescriptionKey: "audit.visibilityCatalog.rbac.role.delete.description", Description: "RBAC role deletion event.", Category: "rbac"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "rbac.role.permissions.replace", DisplayName: "rbac.role.permissions.replace", DescriptionKey: "audit.visibilityCatalog.rbac.role.permissionsReplace.description", Description: "RBAC role permission replacement event.", Category: "rbac"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "rbac.user.roles.replace", DisplayName: "rbac.user.roles.replace", DescriptionKey: "audit.visibilityCatalog.rbac.user.rolesReplace.description", Description: "RBAC user role replacement event.", Category: "rbac"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "ops.container.action.start", DisplayName: "ops.container.action.start", DescriptionKey: "audit.visibilityCatalog.container.action.start.description", Description: "Container start dangerous action event.", Category: "container"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "ops.container.action.stop", DisplayName: "ops.container.action.stop", DescriptionKey: "audit.visibilityCatalog.container.action.stop.description", Description: "Container stop dangerous action event.", Category: "container"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "ops.container.action.restart", DisplayName: "ops.container.action.restart", DescriptionKey: "audit.visibilityCatalog.container.action.restart.description", Description: "Container restart dangerous action event.", Category: "container"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "ops.container.action.remove", DisplayName: "ops.container.action.remove", DescriptionKey: "audit.visibilityCatalog.container.action.remove.description", Description: "Container remove dangerous action event.", Category: "container"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "ops.container.action.batch.start", DisplayName: "ops.container.action.batch.start", DescriptionKey: "audit.visibilityCatalog.container.action.batchStart.description", Description: "Container batch-start dangerous action event.", Category: "container"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "ops.container.action.batch.stop", DisplayName: "ops.container.action.batch.stop", DescriptionKey: "audit.visibilityCatalog.container.action.batchStop.description", Description: "Container batch-stop dangerous action event.", Category: "container"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "ops.container.action.batch.restart", DisplayName: "ops.container.action.batch.restart", DescriptionKey: "audit.visibilityCatalog.container.action.batchRestart.description", Description: "Container batch-restart dangerous action event.", Category: "container"},
		{Source: auditstore.AuditSourceDomainEvent, ActionKey: "ops.container.action.batch.remove", DisplayName: "ops.container.action.batch.remove", DescriptionKey: "audit.visibilityCatalog.container.action.batchRemove.description", Description: "Container batch-remove dangerous action event.", Category: "container"},
	}
}

// appendAuditEventCatalogItem 将一个审计事件目录项合并到结果列表中，并应用对应的可见性覆盖策略。
func appendAuditEventCatalogItem(
	items *[]auditstore.AuditEventCatalogItem,
	seen map[string]struct{},
	overrideMap map[string]auditstore.AuditVisibilityOverride,
	defaultStrategy auditstore.AuditVisibilityStrategy,
	seed auditEventCatalogSeed,
) {
	normalizedActionKey := strings.TrimSpace(seed.ActionKey)
	key := string(seed.Source) + "|" + normalizedActionKey
	if _, exists := seen[key]; exists {
		return
	}
	seen[key] = struct{}{}

	effectiveStrategy := defaultStrategy
	overridden := false
	if override, ok := overrideMap[key]; ok {
		effectiveStrategy = override.Strategy
		overridden = true
		if strings.TrimSpace(seed.Description) == "" {
			seed.Description = override.Description
		}
	}

	*items = append(*items, auditstore.AuditEventCatalogItem{
		Source:            seed.Source,
		ActionKey:         normalizedActionKey,
		DisplayName:       seed.DisplayName,
		Description:       seed.Description,
		Category:          seed.Category,
		DefaultStrategy:   defaultStrategy,
		EffectiveStrategy: effectiveStrategy,
		Overridden:        overridden,
	})
}

func (s *Service) resolveCandidateVisibilityStrategy(
	ctx context.Context,
	candidate auditstore.AuditCandidate,
) (auditstore.AuditVisibilityStrategy, error) {
	if strategy, ok, err := s.findCandidateVisibilityOverrideStrategy(ctx, candidate); err != nil {
		return "", err
	} else if ok {
		return strategy, nil
	}

	defaultValue, err := s.repo.GetAuditVisibilityDefault(ctx, auditVisibilityGlobalDefaultKey)
	if err != nil {
		return "", fmt.Errorf("read audit visibility default: %w", err)
	}

	if strategy := normalizeAuditVisibilityStrategy(defaultValue.Strategy); strategy != "" {
		return strategy, nil
	}
	return auditstore.AuditVisibilityStrategyVisible, nil
}

func (s *Service) findCandidateVisibilityOverrideStrategy(
	ctx context.Context,
	candidate auditstore.AuditCandidate,
) (auditstore.AuditVisibilityStrategy, bool, error) {
	normalizedSource := normalizeAuditSource(candidate.Source)
	actionKey := normalizeCandidateAction(candidate)
	if normalizedSource == "" || actionKey == "" {
		return "", false, nil
	}

	override, found, err := s.repo.FindAuditVisibilityOverride(ctx, normalizedSource, actionKey)
	if err != nil {
		return "", false, fmt.Errorf("find audit visibility override: %w", err)
	}
	if !found {
		return "", false, nil
	}

	strategy := normalizeAuditVisibilityStrategy(override.Strategy)
	if strategy == "" {
		return "", false, nil
	}
	return strategy, true, nil
}
