package store

import (
	"encoding/json"
	"errors"
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

func TestBuildListWhereEscapesKeywordLikeMetacharacters(t *testing.T) {
	t.Parallel()

	where, args := buildListWhere(ListQuery{Keyword: `50%_\`})
	if len(where) != 2 || !strings.Contains(where[1], "ESCAPE '\\'") {
		t.Fatalf("expected LIKE escape clause, got %#v", where)
	}
	if len(args) != 3 {
		t.Fatalf("expected one keyword argument per LIKE term, got %#v", args)
	}
	for _, arg := range args {
		if got, ok := arg.(string); !ok || got != `%50\%\_\\%` {
			t.Fatalf("expected escaped LIKE pattern, got %#v", arg)
		}
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
		RemoveOrphans:            true,
		WaitAfterUp:              true,
		WaitTimeoutSeconds:       180,
		RenewAnonVolumes:         true,
		PruneImagesAfterRedeploy: true,
		AdditionalArgs:           []string{"--progress", "plain"},
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
		"remove_orphans",
		"wait_after_up",
		"wait_timeout_seconds",
		"renew_anon_volumes",
		"prune_images_after_redeploy",
		"additional_args",
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
		"remove_orphans":false,
		"wait_after_up":true,
		"wait_timeout_seconds":180,
		"renew_anon_volumes":true,
		"prune_images_after_redeploy":true
		,"additional_args":["--progress","plain"]
	}`))
	if err != nil {
		t.Fatalf("decode lifecycle config: %v", err)
	}
	if len(config.Profiles) != 2 || !config.DownBeforeRedeploy || !config.PullBeforeRedeploy || !config.BuildBeforeUp || !config.ForceRecreate || config.RemoveOrphans || !config.WaitAfterUp || config.WaitTimeoutSeconds != 180 || !config.RenewAnonVolumes || !config.PruneImagesAfterRedeploy || len(config.AdditionalArgs) != 2 || config.AdditionalArgs[0] != "--progress" || config.AdditionalArgs[1] != "plain" {
		t.Fatalf("expected snake_case payload to round-trip, got %#v", config)
	}
}

func TestDecodeLifecycleConfigJSONAppliesDefaultsForLegacyEmptyObject(t *testing.T) {
	t.Parallel()

	config, err := decodeLifecycleConfigJSON([]byte(`{}`))
	if err != nil {
		t.Fatalf("decode legacy empty lifecycle config: %v", err)
	}
	if len(config.Profiles) != 0 || config.DownBeforeRedeploy || config.PullBeforeRedeploy || config.BuildBeforeUp || config.ForceRecreate || !config.RemoveOrphans || config.WaitAfterUp || config.WaitTimeoutSeconds != defaultLifecycleWaitTimeoutSeconds || config.RenewAnonVolumes || config.PruneImagesAfterRedeploy || len(config.AdditionalArgs) != 0 {
		t.Fatalf("expected legacy defaults, got %#v", config)
	}
}

func TestSourceMetadataRejectsLineBreaksInValues(t *testing.T) {
	t.Parallel()

	for name, metadata := range map[string]map[string]string{
		"carriage return": {"source": "runtime\rcompose"},
		"line feed":       {"source": "runtime\ncompose"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := encodeSourceMetadataJSON(metadata); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected encode to reject line break metadata value, got %v", err)
			}

			raw, err := json.Marshal(metadata)
			if err != nil {
				t.Fatalf("marshal metadata: %v", err)
			}
			if _, err := decodeSourceMetadataJSON(raw); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected decode to reject line break metadata value, got %v", err)
			}
		})
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
