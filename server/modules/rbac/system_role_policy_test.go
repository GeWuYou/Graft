package rbac

import (
	"strings"
	"testing"

	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
)

func TestSystemRolePolicyHasOneExplicitDecisionPerPermission(t *testing.T) {
	items := policyTestItems()
	if err := ValidateSystemRolePolicy(items); err != nil {
		t.Fatalf("validate policy: %v", err)
	}

	seen := make(map[string]bool)
	for _, entry := range SystemRolePolicy() {
		if seen[entry.Code] {
			t.Fatalf("duplicate policy code %s", entry.Code)
		}
		seen[entry.Code] = true
		for role, scope := range entry.Grants {
			if scope == moduleapi.PermissionScopeOwned && !ownedScopePermission(entry.Code) {
				t.Fatalf("unapproved owned scope: %s=%s", role, entry.Code)
			}
		}
	}
}

func TestSystemRolePolicyRejectsCoverageAndCriticalReviewGaps(t *testing.T) {
	tests := []struct {
		name  string
		items []permission.Item
		want  string
	}{
		{name: "unregistered policy code", items: policyTestItems()[:len(policyTestItems())-1], want: "references unregistered permission"},
		{name: "uncovered registered code", items: append(policyTestItems(), permission.Item{Code: "example.read", Module: "example", Resource: "example", Action: "read", RiskLevel: permission.RiskLevelLow, RiskCategory: permission.RiskCategoryRead}), want: "does not cover"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateSystemRolePolicy(test.items)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestSystemRolePolicyCriticalEntriesHaveReviewMetadata(t *testing.T) {
	critical := map[string]bool{"container.shell": true, "role.permission.assign": true, "user.role.assign": true, "system-config.write": true}
	for _, entry := range SystemRolePolicy() {
		if !critical[entry.Code] {
			continue
		}
		if strings.TrimSpace(entry.RiskOwner) == "" || strings.TrimSpace(entry.ChangeReason) == "" {
			t.Fatalf("critical policy entry %s lacks review metadata", entry.Code)
		}
	}
}

func policyTestItems() []permission.Item {
	entries := SystemRolePolicy()
	items := make([]permission.Item, 0, len(entries))
	for _, entry := range entries {
		level := permission.RiskLevelLow
		if entry.Code == "container.shell" || entry.Code == "role.permission.assign" || entry.Code == "user.role.assign" || entry.Code == "system-config.write" {
			level = permission.RiskLevelCritical
		}
		items = append(items, permission.Item{Code: entry.Code, Module: "test", Resource: "test", Action: "read", RiskLevel: level, RiskCategory: permission.RiskCategoryRead})
	}
	return items
}
