package audit

import (
	"context"
	"strings"
	"testing"
	"time"

	auditstore "graft/server/modules/audit/store"
)

func TestLoadAuditRiskEventsWidgetBuildsExplicitAuditLogQueries(t *testing.T) {
	t.Parallel()

	service, err := NewService(&stubAuditRepository{
		overview: auditstore.AuditOverview{
			Summary: auditstore.OverviewSummary{
				FailedOperations: 5,
				HighRiskEvents:   3,
			},
			RiskGroups: []auditstore.OverviewRiskGroup{
				{Key: string(auditstore.AuditBusinessCategoryAuthFailures), Count: 2},
			},
			FailedAuth: []auditstore.OverviewItem{
				{
					Action:    "auth.login_failed",
					Message:   "auth failed",
					CreatedAt: time.Date(2026, 6, 12, 8, 0, 0, 0, time.UTC),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	payload, err := loadAuditRiskEventsWidget(context.Background(), service)
	if err != nil {
		t.Fatalf("load widget: %v", err)
	}

	items, ok := payload["items"].([]map[string]any)
	if !ok {
		t.Fatalf("expected alert items, got %#v", payload["items"])
	}
	if len(items) != 3 {
		t.Fatalf("expected three security entries, got %#v", items)
	}

	assertWidgetItem(t, items[0], "audit.high-risk", "dashboard.widget.auditRiskEvents.highRisk.action", "risk_levels=HIGH%2CCRITICAL")
	assertWidgetItem(t, items[1], "audit.failed-operations", "dashboard.widget.auditRiskEvents.failedOperations.action", "results=FAILED%2CDENIED%2CERROR")
	assertWidgetItem(t, items[2], "audit.failed-auth", "dashboard.widget.auditRiskEvents.authFailures.action", "business_category=auth_failures")

	for _, item := range items {
		location, _ := item["route_location"].(string)
		if strings.Contains(location, "scope=") {
			t.Fatalf("security dashboard links must use explicit filters, got %s", location)
		}
		if !strings.Contains(location, "preset=last_24h") {
			t.Fatalf("security dashboard links must keep the 24h window, got %s", location)
		}
	}
}

func assertWidgetItem(t *testing.T, item map[string]any, wantID string, wantActionKey string, wantQueryPart string) {
	t.Helper()

	if item["id"] != wantID {
		t.Fatalf("expected item %s, got %#v", wantID, item)
	}
	if item["action_label_key"] != wantActionKey {
		t.Fatalf("expected action key %s, got %#v", wantActionKey, item)
	}
	location, _ := item["route_location"].(string)
	if !strings.Contains(location, wantQueryPart) {
		t.Fatalf("expected %s in route location, got %s", wantQueryPart, location)
	}
}
