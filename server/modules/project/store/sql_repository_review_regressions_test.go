package store

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestComposeProjectsUpsertSQLPlaceholderCountMatchesColumns(t *testing.T) {
	t.Parallel()

	query := applicationsUpsertSQL()
	columnCount := countDelimitedItems(extractBetween(query, "INSERT INTO applications (", ") VALUES"))
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

func TestNormalizeListQuerySortDefaultsAndRejectsUnknownValues(t *testing.T) {
	t.Parallel()

	defaulted, err := normalizeListQuery(ListQuery{})
	if err != nil {
		t.Fatalf("normalize default list query: %v", err)
	}
	if defaulted.Sort != ApplicationListSortCreatedAtDesc {
		t.Fatalf("expected default sort %q, got %q", ApplicationListSortCreatedAtDesc, defaulted.Sort)
	}
	for _, sortExpression := range []string{ApplicationListSortCreatedAtAsc, ApplicationListSortCreatedAtDesc} {
		normalized, normalizeErr := normalizeListQuery(ListQuery{Sort: sortExpression})
		if normalizeErr != nil || normalized.Sort != sortExpression {
			t.Fatalf("expected valid sort %q, got %#v, err=%v", sortExpression, normalized, normalizeErr)
		}
	}
	if _, err := normalizeListQuery(ListQuery{Sort: "updated_at:desc"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid sort to return ErrInvalidInput, got %v", err)
	}
}

func TestSQLRepositoryListSortsByCreatedAtAndStableID(t *testing.T) {
	t.Parallel()

	repo, db := newTestSQLRepository(t)
	createdAt := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	insertProjectRow(t, db, 1, "older", createdAt.Add(-time.Minute), 0)
	insertProjectRow(t, db, 2, "same-a", createdAt, 0)
	insertProjectRow(t, db, 3, "same-b", createdAt, 0)

	for _, testCase := range []struct {
		name string
		sort string
		want []uint64
	}{
		{name: "default descending", want: []uint64{3, 2, 1}},
		{name: "explicit descending", sort: ApplicationListSortCreatedAtDesc, want: []uint64{3, 2, 1}},
		{name: "ascending", sort: ApplicationListSortCreatedAtAsc, want: []uint64{1, 3, 2}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := repo.List(context.Background(), ListQuery{Limit: 10, Sort: testCase.sort})
			if err != nil {
				t.Fatalf("list applications: %v", err)
			}
			if len(result.Items) != len(testCase.want) {
				t.Fatalf("expected %d items, got %d", len(testCase.want), len(result.Items))
			}
			for index, wantID := range testCase.want {
				if got := result.Items[index].Application.ApplicationRecordID; got != wantID {
					t.Fatalf("item %d: expected id %d, got %d", index, wantID, got)
				}
			}
		})
	}
}

func TestBuildListOrderByUsesWhitelistedExpressions(t *testing.T) {
	t.Parallel()

	if got := buildListOrderBy(ApplicationListSortCreatedAtDesc); got != "created_at DESC, application_record_id DESC" {
		t.Fatalf("unexpected descending order clause %q", got)
	}
	if got := buildListOrderBy(ApplicationListSortCreatedAtAsc); got != "created_at ASC, application_record_id DESC" {
		t.Fatalf("unexpected ascending order clause %q", got)
	}
}

func TestLifecycleConfigJSONUsesStableSnakeCaseKeys(t *testing.T) {
	t.Parallel()

	encoded, err := encodeLifecycleConfigJSON(LifecycleConfig{
		Profiles:                 []string{"blue"},
		ManagedServiceNames:      []string{"api"},
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
		StopArgs:                 []string{"--timeout", "30"},
		RestartArgs:              []string{"--no-deps"},
		PullArgs:                 []string{"--include-deps"},
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
		"managed_service_names",
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
		"stop_args",
		"restart_args",
		"pull_args",
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
		"managed_service_names":["api","worker"],
		"down_before_redeploy":true,
		"pull_before_redeploy":true,
		"build_before_up":true,
		"force_recreate":true,
		"remove_orphans":false,
		"wait_after_up":true,
		"wait_timeout_seconds":180,
		"renew_anon_volumes":true,
		"prune_images_after_redeploy":true
		,"additional_args":["--progress","plain"],
		"stop_args":["--timeout","30"],
		"restart_args":["--no-deps"],
		"pull_args":["--include-deps"]
	}`))
	if err != nil {
		t.Fatalf("decode lifecycle config: %v", err)
	}
	expected := LifecycleConfig{
		Profiles: []string{"blue", "green"}, ManagedServiceNames: []string{"api", "worker"},
		DownBeforeRedeploy: true, PullBeforeRedeploy: true, BuildBeforeUp: true, ForceRecreate: true,
		RemoveOrphans: false, WaitAfterUp: true, WaitTimeoutSeconds: 180, RenewAnonVolumes: true,
		PruneImagesAfterRedeploy: true, AdditionalArgs: []string{"--progress", "plain"},
		StopArgs: []string{"--timeout", "30"}, RestartArgs: []string{"--no-deps"}, PullArgs: []string{"--include-deps"},
	}
	if !reflect.DeepEqual(config, expected) {
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
