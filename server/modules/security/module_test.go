package security

import (
	"testing"

	"graft/server/internal/moduleapi"
)

func TestParseOverviewPreset(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  moduleapi.AuditOverviewPreset
		ok    bool
	}{
		{name: "default", input: "", want: moduleapi.AuditOverviewPresetLast24Hours, ok: true},
		{name: "24 hours", input: "last_24h", want: moduleapi.AuditOverviewPresetLast24Hours, ok: true},
		{name: "7 days", input: "last_7d", want: moduleapi.AuditOverviewPresetLast7Days, ok: true},
		{name: "30 days", input: "last_30d", want: moduleapi.AuditOverviewPresetLast30Days, ok: true},
		{name: "invalid", input: "all", ok: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseOverviewPreset(tc.input)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("parseOverviewPreset(%q) = (%q, %t), want (%q, %t)", tc.input, got, ok, tc.want, tc.ok)
			}
		})
	}
}
