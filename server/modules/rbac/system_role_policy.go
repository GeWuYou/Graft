package rbac

import (
	"context"
	"fmt"
	"sort"
	"strings"

	capabilitycontract "graft/server/internal/contract/capability"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	rbacstore "graft/server/modules/rbac/store"
	usercontract "graft/server/modules/user/contract"
)

const (
	roleAdmin               = "admin"
	rolePlatformOperator    = "platform_operator"
	roleNoShellOperator     = "no_shell_operator"
	roleApplicationOperator = "application_operator"
	roleDeveloper           = "developer"
	roleViewer              = "viewer"
	roleMonitor             = "monitor"
	roleSecurityAuditor     = "security_auditor"
)

// SystemRolePolicyEntry 表示一个已注册权限的显式系统角色授权决策。
// 每个条目都显式授予 Admin；没有其他授权的条目仅限 Admin。
type SystemRolePolicyEntry struct {
	Code         string
	Grants       map[string]moduleapi.PermissionScope
	RiskOwner    string
	ChangeReason string
}

// SystemRolePolicy 返回 RBAC 拥有的已批准系统角色矩阵。
// 它明确列出每个权限，不从资源、动作或风险元数据推断授权。
func SystemRolePolicy() []SystemRolePolicyEntry {
	entries := []SystemRolePolicyEntry{}
	add := func(code string, grants map[string]moduleapi.PermissionScope) {
		if grants == nil {
			grants = make(map[string]moduleapi.PermissionScope)
		}
		grants[roleAdmin] = moduleapi.PermissionScopeAll
		entries = append(entries, SystemRolePolicyEntry{Code: code, Grants: grants})
	}
	add("announcement.read", allScope(roleViewer))
	add("announcement.create", nil)
	add("announcement.update", nil)
	add("announcement.publish", nil)
	add("announcement.delete", nil)
	add("audit.read", allScope(roleSecurityAuditor))
	add("audit.manage", nil)
	for _, entry := range systemRolePolicyGrants() {
		add(entry.code, entry.grants)
	}
	for _, code := range adminOnlySystemPermissions() {
		add(code, nil)
	}
	applyCriticalPolicyMetadata(entries)
	return entries
}

type policyGrant struct {
	code   string
	grants map[string]moduleapi.PermissionScope
}

func allScope(role string) map[string]moduleapi.PermissionScope {
	return map[string]moduleapi.PermissionScope{role: moduleapi.PermissionScopeAll}
}

func systemRolePolicyGrants() []policyGrant {
	return []policyGrant{
		{"container.view", operatorScope()}, {"container.detail", operatorScope()}, {"container.events", operatorScope()}, {"container.logs", operatorScope()}, {"container.start", operatorScope()}, {"container.stop", operatorScope()}, {"container.restart", operatorScope()}, {"container.image.pull", allScope(rolePlatformOperator)},
		{"access_log.read", allScope(roleMonitor)}, {"app_log.read", allScope(roleMonitor)}, {"monitor.server-status.read", map[string]moduleapi.PermissionScope{roleViewer: moduleapi.PermissionScopeAll, roleMonitor: moduleapi.PermissionScopeAll}}, {"security.overview.read", allScope(roleSecurityAuditor)},
		{"application.view", map[string]moduleapi.PermissionScope{roleApplicationOperator: moduleapi.PermissionScopeAll, roleDeveloper: moduleapi.PermissionScopeOwned}}, {"application.create", map[string]moduleapi.PermissionScope{roleDeveloper: moduleapi.PermissionScopeOwned}}, {"application.deploy", map[string]moduleapi.PermissionScope{roleApplicationOperator: moduleapi.PermissionScopeAll, roleDeveloper: moduleapi.PermissionScopeOwned}},
		{"application.lifecycle", allScope(roleApplicationOperator)}, {"application.import", allScope(roleApplicationOperator)}, {"application.refresh", allScope(roleApplicationOperator)}, {"application.discovery.view", allScope(roleApplicationOperator)},
	}
}

func operatorScope() map[string]moduleapi.PermissionScope {
	return map[string]moduleapi.PermissionScope{rolePlatformOperator: moduleapi.PermissionScopeAll, roleNoShellOperator: moduleapi.PermissionScopeAll}
}

func adminOnlySystemPermissions() []string {
	return []string{
		"container.shell", "container.environment", "container.remove", "container.volume.remove", "container.image.tag", "container.image.untag", "container.image.remove", "container.network.create", "container.network.remove", "app_log.delete", "modules.runtime.read", capabilitycontract.ReadPermission,
		"notification.view", "notification.read", "notification.manage", "platform-backup.read", "platform-backup.create", "platform-network.read", "platform-network.write", "platform-network.diagnose", "platform-network.targets.manage", "platform-network.exit-ip.read", "platform-update.read", "platform-update.check", "platform-update.manage",
		"build.read", "build.create", "build.cancel", "build.retry",
		"application.destroy", "application.creation-method.view", "application.template.manage", "application.template.publish",
		"role.read", "role.create", "role.update", "role.status.update", "role.delete", "role.permission.assign", "permission.read", "user.role.read", "user.role.assign", "runtime_target.view", "runtime_target.manage", "runtime_target.assignment.manage", "runtime_target.refresh", "scheduled-task.read", "scheduled-task.create", "scheduled-task.update", "scheduled-task.delete", "scheduled-task.run", "scheduled-task.enable", "system-config.read", "system-config.write", usercontract.UserReadPermission.String(), usercontract.UserCreatePermission.String(), usercontract.UserUpdatePermission.String(), usercontract.UserDisablePermission.String(), usercontract.UserSessionReadPermission.String(), usercontract.UserSessionRevokePermission.String(),
	}
}

func applyCriticalPolicyMetadata(entries []SystemRolePolicyEntry) {
	critical := map[string]bool{"container.shell": true, "role.permission.assign": true, "user.role.assign": true, "system-config.write": true}
	for index := range entries {
		if !critical[entries[index].Code] {
			continue
		}
		entries[index].RiskOwner = "platform-security"
		entries[index].ChangeReason = "Approved critical permission remains restricted to the Admin system role."
	}
}

// ValidateSystemRolePolicy 校验显式策略覆盖范围和受限支持的 owned scope。
func ValidateSystemRolePolicy(items []permission.Item) error {
	policy := SystemRolePolicy()
	byCode, err := indexSystemRolePolicy(policy)
	if err != nil {
		return err
	}
	registered, err := indexRegisteredPermissions(items, byCode)
	if err != nil {
		return err
	}
	for code, entry := range byCode {
		item, exists := registered[code]
		if !exists {
			return fmt.Errorf("system role policy references unregistered permission %s", code)
		}
		if item.RiskLevel == permission.RiskLevelCritical && (strings.TrimSpace(entry.RiskOwner) == "" || strings.TrimSpace(entry.ChangeReason) == "") {
			return fmt.Errorf("critical permission %s requires risk_owner and change_reason", code)
		}
	}
	return nil
}

func indexSystemRolePolicy(policy []SystemRolePolicyEntry) (map[string]SystemRolePolicyEntry, error) {
	byCode := make(map[string]SystemRolePolicyEntry, len(policy))
	for _, entry := range policy {
		if err := validateSystemRolePolicyEntry(entry, byCode); err != nil {
			return nil, err
		}
		byCode[entry.Code] = entry
	}
	return byCode, nil
}

func validateSystemRolePolicyEntry(entry SystemRolePolicyEntry, byCode map[string]SystemRolePolicyEntry) error {
	if strings.TrimSpace(entry.Code) == "" {
		return fmt.Errorf("system role policy contains an empty permission code")
	}
	if _, exists := byCode[entry.Code]; exists {
		return fmt.Errorf("system role policy duplicates %s", entry.Code)
	}
	for role, scope := range entry.Grants {
		if !knownSystemRole(role) || (scope != moduleapi.PermissionScopeAll && scope != moduleapi.PermissionScopeOwned) {
			return fmt.Errorf("system role policy has invalid grant %s=%s for %s", role, scope, entry.Code)
		}
		if scope == moduleapi.PermissionScopeOwned && !ownedScopePermission(entry.Code) {
			return fmt.Errorf("owned scope is not approved for %s", entry.Code)
		}
	}
	return nil
}

func indexRegisteredPermissions(items []permission.Item, policy map[string]SystemRolePolicyEntry) (map[string]permission.Item, error) {
	registered := make(map[string]permission.Item, len(items))
	for _, item := range items {
		if !item.Valid() {
			return nil, fmt.Errorf("registered permission %s has invalid metadata", item.Code)
		}
		if _, exists := registered[item.Code]; exists {
			return nil, fmt.Errorf("registered permission is duplicated: %s", item.Code)
		}
		if _, exists := policy[item.Code]; !exists {
			return nil, fmt.Errorf("system role policy does not cover registered permission %s", item.Code)
		}
		registered[item.Code] = item
	}
	return registered, nil
}

func knownSystemRole(role string) bool {
	return role == roleAdmin || role == rolePlatformOperator || role == roleNoShellOperator || role == roleApplicationOperator || role == roleDeveloper || role == roleViewer || role == roleMonitor || role == roleSecurityAuditor
}
func ownedScopePermission(code string) bool {
	return code == "application.view" || code == "application.create" || code == "application.deploy"
}

// SystemRolePolicyCodes 返回供迁移和一致性测试使用的稳定排序编码快照。
func SystemRolePolicyCodes() []string {
	entries := SystemRolePolicy()
	codes := make([]string, 0, len(entries))
	for _, entry := range entries {
		codes = append(codes, entry.Code)
	}
	sort.Strings(codes)
	return codes
}

// ValidateSystemRoleDatabase 对比已迁移的权限目录、系统角色绑定与策略。
// 它只读执行；不一致会阻止启动并要求通过显式迁移修复。
func ValidateSystemRoleDatabase(ctx context.Context, repository rbacstore.Repository, items []permission.Item) error {
	if repository == nil {
		return fmt.Errorf("rbac policy repository is unavailable")
	}
	if err := ValidateSystemRolePolicy(items); err != nil {
		return err
	}
	permissionIDs, err := validatePermissionCatalog(ctx, repository)
	if err != nil {
		return err
	}
	return validateSystemRoleBindings(ctx, repository, permissionIDs)
}

func validatePermissionCatalog(ctx context.Context, repository rbacstore.Repository) (map[string]uint64, error) {
	permissions, err := repository.ListPermissions(ctx, rbacstore.PermissionFilter{})
	if err != nil {
		return nil, fmt.Errorf("read permission catalog for policy validation: %w", err)
	}
	policyCodes := SystemRolePolicyCodes()
	databaseCodes := make([]string, 0, len(permissions))
	permissionIDs := make(map[string]uint64, len(permissions))
	for _, record := range permissions {
		databaseCodes = append(databaseCodes, record.Code)
		permissionIDs[record.Code] = record.ID
	}
	sort.Strings(databaseCodes)
	if strings.Join(databaseCodes, ",") != strings.Join(policyCodes, ",") {
		return nil, fmt.Errorf("rbac permission catalog does not match system role policy")
	}
	return permissionIDs, nil
}

func validateSystemRoleBindings(ctx context.Context, repository rbacstore.Repository, permissionIDs map[string]uint64) error {
	roles, err := repository.ListRoles(ctx, rbacstore.RoleFilter{})
	if err != nil {
		return fmt.Errorf("read system roles for policy validation: %w", err)
	}
	entries := SystemRolePolicy()
	policyByCode := make(map[string]SystemRolePolicyEntry, len(entries))
	for _, entry := range entries {
		policyByCode[entry.Code] = entry
	}
	seen := make(map[string]bool)
	for _, role := range roles {
		if role.Type != "system" {
			continue
		}
		roleKey, validationErr := validateSystemRoleBinding(ctx, repository, role, policyByCode, permissionIDs)
		if validationErr != nil {
			return validationErr
		}
		seen[roleKey] = true
	}
	for _, role := range []string{roleAdmin, rolePlatformOperator, roleNoShellOperator, roleApplicationOperator, roleDeveloper, roleViewer, roleMonitor, roleSecurityAuditor} {
		if !seen[role] {
			return fmt.Errorf("required system role %s is missing", role)
		}
	}
	return nil
}

func validateSystemRoleBinding(
	ctx context.Context,
	repository rbacstore.Repository,
	role rbacstore.Role,
	policy map[string]SystemRolePolicyEntry,
	permissionIDs map[string]uint64,
) (string, error) {
	roleKey := valueOrEmpty(role.BuiltinKey)
	if !knownSystemRole(roleKey) {
		return "", fmt.Errorf("unknown system role policy key %q", roleKey)
	}
	bindings, err := repository.ListRolePermissionBindings(ctx, role.ID)
	if err != nil {
		return "", fmt.Errorf("read system role bindings for %s: %w", roleKey, err)
	}
	expected := expectedRoleBindings(roleKey, policy, permissionIDs)
	actual := make(map[uint64]string, len(bindings))
	for _, binding := range bindings {
		actual[binding.PermissionID] = binding.Scope
	}
	if len(actual) != len(expected) {
		return "", fmt.Errorf("system role %s binding count does not match policy", roleKey)
	}
	for permissionID, scope := range expected {
		if actual[permissionID] != scope {
			return "", fmt.Errorf("system role %s binding parity mismatch", roleKey)
		}
	}
	return roleKey, nil
}

func expectedRoleBindings(role string, policy map[string]SystemRolePolicyEntry, ids map[string]uint64) map[uint64]string {
	bindings := make(map[uint64]string)
	for code, entry := range policy {
		id := ids[code]
		if role == roleAdmin {
			bindings[id] = string(moduleapi.PermissionScopeAll)
			continue
		}
		if scope, ok := entry.Grants[role]; ok {
			bindings[id] = string(scope)
		}
	}
	return bindings
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
