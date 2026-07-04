package store

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestComposeProjectsUpsertSQLPlaceholderCountMatchesColumns(t *testing.T) {
	t.Parallel()

	query := composeProjectsUpsertSQL()
	columnCount := countDelimitedItems(extractBetween(query, "INSERT INTO compose_projects (", ") VALUES"))
	placeholderCount := strings.Count(extractBetween(query, ") VALUES (", ")\n\t\tON CONFLICT"), "?")
	if columnCount == 0 || placeholderCount == 0 {
		t.Fatalf("expected non-empty insert SQL, got %q", query)
	}
	if columnCount != placeholderCount {
		t.Fatalf("expected placeholder count %d to match column count %d in query %q", placeholderCount, columnCount, query)
	}
}

func TestLifecycleConfigJSONUsesStableSnakeCaseKeys(t *testing.T) {
	t.Parallel()

	encoded, err := encodeLifecycleConfigJSON(LifecycleConfig{
		Profiles:                 []string{"blue"},
		DownBeforeRedeploy:       true,
		PullBeforeRedeploy:       true,
		BuildBeforeUp:            true,
		ForceRecreate:            true,
		WaitAfterUp:              true,
		PruneImagesAfterRedeploy: true,
	})
	if err != nil {
		t.Fatalf("encode lifecycle config: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("decode encoded lifecycle config: %v", err)
	}
	for _, key := range []string{
		"profiles",
		"down_before_redeploy",
		"pull_before_redeploy",
		"build_before_up",
		"force_recreate",
		"wait_after_up",
		"prune_images_after_redeploy",
	} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("expected encoded lifecycle config to contain %q, got %#v", key, raw)
		}
	}
}

func TestDecodeLifecycleConfigJSONAcceptsSnakeCasePayload(t *testing.T) {
	t.Parallel()

	config, err := decodeLifecycleConfigJSON([]byte(`{
		"profiles":["blue","green"],
		"down_before_redeploy":true,
		"pull_before_redeploy":true,
		"build_before_up":true,
		"force_recreate":true,
		"wait_after_up":true,
		"prune_images_after_redeploy":true
	}`))
	if err != nil {
		t.Fatalf("decode lifecycle config: %v", err)
	}
	if len(config.Profiles) != 2 || !config.DownBeforeRedeploy || !config.PullBeforeRedeploy || !config.BuildBeforeUp || !config.ForceRecreate || !config.WaitAfterUp || !config.PruneImagesAfterRedeploy {
		t.Fatalf("expected snake_case payload to round-trip, got %#v", config)
	}
}

func extractBetween(value string, start string, end string) string {
	startIndex := strings.Index(value, start)
	if startIndex < 0 {
		return ""
	}
	startIndex += len(start)
	endIndex := strings.Index(value[startIndex:], end)
	if endIndex < 0 {
		return ""
	}
	return value[startIndex : startIndex+endIndex]
}

func countDelimitedItems(value string) int {
	count := 0
	for _, item := range strings.Split(value, ",") {
		if strings.TrimSpace(item) != "" {
			count++
		}
	}
	return count
}
