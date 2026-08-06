package store

import (
	"testing"

	"graft/server/internal/moduleapi"
)

func TestNormalizeWorkspaceAcceptsSupportedSourceKinds(t *testing.T) {
	for _, sourceKind := range []string{
		moduleapi.WorkspaceSourceApplication,
		moduleapi.WorkspaceSourceGit,
		moduleapi.WorkspaceSourceArchive,
		moduleapi.WorkspaceSourceGenerated,
		moduleapi.WorkspaceSourceTargetLocal,
	} {
		workspace, err := normalizeWorkspace(moduleapi.BuildWorkspace{
			ID:              "workspace_test",
			Name:            "Application source",
			SourceKind:      sourceKind,
			SourceReference: "source:stable/main",
		})
		if err != nil {
			t.Fatalf("normalize %s: %v", sourceKind, err)
		}
		if workspace.RetentionPolicy != "workspace" {
			t.Fatalf("default retention policy = %q", workspace.RetentionPolicy)
		}
	}
}

func TestNormalizeWorkspaceRejectsUnsafeOrUnknownSource(t *testing.T) {
	cases := []moduleapi.BuildWorkspace{
		{ID: "workspace_test", Name: "source", SourceKind: "unknown", SourceReference: "ref"},
		{ID: "workspace_test", Name: "source", SourceKind: moduleapi.WorkspaceSourceGit, SourceReference: "ref\nsecret"},
		{ID: "workspace_test", Name: "", SourceKind: moduleapi.WorkspaceSourceGit, SourceReference: "ref"},
	}
	for index, value := range cases {
		if _, err := normalizeWorkspace(value); err == nil {
			t.Fatalf("case %d should be rejected", index)
		}
	}
}
