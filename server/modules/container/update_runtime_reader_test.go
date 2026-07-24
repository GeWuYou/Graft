package container

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
)

type updateRuntimeDetailRuntime struct {
	stubProjectReaderRuntime
	detail Detail
}

func (r updateRuntimeDetailRuntime) Detail(context.Context, Ref) (Detail, error) {
	return r.detail, nil
}

func TestDiscoverCurrentServerComposePreservesNestedConfigOrder(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "overrides", "web.yml")
	compose := filepath.Join(root, "compose.yml")
	rootOverride := filepath.Join(root, "compose.override.yml")
	external := filepath.Join(t.TempDir(), "outside.yml")
	runtime := updateRuntimeDetailRuntime{detail: Detail{Summary: Summary{Labels: map[string]string{
		composeWorkingDirLabel:  root,
		composeConfigFilesLabel: stringsJoinConfigFiles(compose, nested, rootOverride, external),
		composeProjectLabel:     "graft",
	}}}}
	reader := containerProjectRuntimeReader{service: &service{runtime: runtime, enabled: true}}

	candidates, err := reader.DiscoverCurrentServerCompose(context.Background())
	if err != nil {
		t.Fatalf("discover compose candidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected one compose candidate, got %d", len(candidates))
	}
	if want := []string{compose, nested, rootOverride}; !slices.Equal(candidates[0].ConfigFiles, want) {
		t.Fatalf("unexpected ordered config files: got %#v want %#v", candidates[0].ConfigFiles, want)
	}
}

func stringsJoinConfigFiles(values ...string) string {
	result := ""
	for index, value := range values {
		if index > 0 {
			result += ","
		}
		result += value
	}
	return result
}
