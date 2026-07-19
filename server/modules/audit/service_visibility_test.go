package audit

import (
	"testing"

	auditstore "graft/server/modules/audit/store"
)

func TestAuditEventCatalogIncludesDockerVolumeBatchRemove(t *testing.T) {
	t.Parallel()

	items := buildAuditEventCatalog(auditstore.AuditVisibilityStrategyVisible, nil)
	for _, item := range items {
		if item.Source != auditstore.AuditSourceDomainEvent || item.ActionKey != "ops.container.volume.remove.batch" {
			continue
		}
		if item.Description != "Docker volume batch-remove dangerous action event." {
			t.Fatalf("unexpected batch volume catalog description %q", item.Description)
		}
		if item.DefaultStrategy != auditstore.AuditVisibilityStrategyVisible || item.EffectiveStrategy != auditstore.AuditVisibilityStrategyVisible {
			t.Fatalf("unexpected batch volume catalog strategy: %#v", item)
		}
		return
	}
	t.Fatal("expected Docker volume batch-remove audit catalog seed")
}
