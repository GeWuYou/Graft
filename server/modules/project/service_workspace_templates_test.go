package project

import (
	"context"
	"errors"
	"testing"

	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

func TestValidateTemplateDefinitionAcceptsSnakeCaseWorkspaceEntries(t *testing.T) {
	t.Parallel()
	definition, err := (&Service{}).validateTemplateDefinition(projectcontract.DeploymentAdapterKindCompose, []byte(`{
  "compose_file_path": "compose.yaml",
  "workspace_entries": [
    {"path": ".env", "node_type": "file", "content": ""},
    {"path": "compose.yaml", "node_type": "file", "content": "services:\n  app:\n    image: nginx:alpine\n"}
  ],
  "lifecycle_configuration": {}
}`))
	if err != nil {
		t.Fatalf("validate snake_case template definition: %v", err)
	}
	if len(definition.WorkspaceEntries) != 2 || definition.WorkspaceEntries[0].NodeType != "file" || definition.WorkspaceEntries[0].Content == nil || *definition.WorkspaceEntries[0].Content != "" {
		t.Fatalf("unexpected parsed workspace entries: %#v", definition.WorkspaceEntries)
	}
}

func TestValidateTemplateDefinitionRejectsInvalidCatalogDocumentation(t *testing.T) {
	t.Parallel()
	_, err := (&Service{}).validateTemplateDefinition(projectcontract.DeploymentAdapterKindCompose, []byte(`{
  "compose_file_path": "compose.yaml",
  "workspace_entries": [
    {"path": "compose.yaml", "node_type": "file", "content": "services: {}\n"}
  ],
  "catalog_documentation": {
    "variables": [
      {"name": "invalid-name", "required": true, "description": "invalid"}
    ]
  }
}`))
	if !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("invalid catalog documentation error = %v, want invalid argument", err)
	}
}

func TestNormalizeTemplateDraftRejectsInvalidCategory(t *testing.T) {
	t.Parallel()
	_, err := (&Service{}).normalizeTemplateDraft(ApplicationTemplateDraftRequest{
		DisplayName:             "Template",
		Category:                "unsupported",
		DeploymentAdapterKind:   projectcontract.DeploymentAdapterKindCompose,
		DefinitionSchemaVersion: 1,
	})
	if !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("invalid category error = %v, want invalid argument", err)
	}
}

func TestValidateTemplateDefinitionRejectsUnnormalizedCatalogVariableName(t *testing.T) {
	t.Parallel()

	_, err := (&Service{}).validateTemplateDefinition(projectcontract.DeploymentAdapterKindCompose, []byte(`{
  "compose_file_path": "compose.yaml",
  "workspace_entries": [
    {"path": "compose.yaml", "node_type": "file", "content": "services: {}\n"}
  ],
  "catalog_documentation": {
    "variables": [
      {"name": " APP_PORT ", "required": true, "description": "port"}
    ]
  }
}`))
	if !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("unnormalized catalog variable name error = %v, want invalid argument", err)
	}
}

func TestListPublishedApplicationTemplatesRejectsUnsafePagination(t *testing.T) {
	t.Parallel()

	for _, query := range []projectstore.TemplateCatalogQuery{
		{Page: 0, PageSize: 24},
		{Page: 1, PageSize: 0},
		{Page: 1, PageSize: 101},
		{Page: int(^uint(0) >> 1), PageSize: 100},
	} {
		_, err := (&Service{}).ListPublishedApplicationTemplates(context.Background(), projectcontract.DeploymentAdapterKindCompose, query)
		if !errors.Is(err, errProjectInvalidArgument) {
			t.Fatalf("pagination %#v error = %v, want invalid argument", query, err)
		}
	}
}

func TestNormalizeManagedWorkspaceEntriesAcceptsArbitraryTextAndEmptyDirectory(t *testing.T) {
	compose := "services: {}\n"
	content := "arbitrary text\n"
	entries, err := normalizeManagedWorkspaceEntries([]ManagedWorkspaceEntry{
		{Path: "nested", NodeType: "directory"},
		{Path: "nested/readme", NodeType: "file", Content: &content},
		{Path: "compose.yaml", NodeType: "file", Content: &compose},
	}, "compose.yaml")
	if err != nil {
		t.Fatalf("normalize entries: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("entry count = %d, want 3", len(entries))
	}
}

func TestNormalizeManagedWorkspaceEntriesRejectsFileAncestorRegardlessOfOrder(t *testing.T) {
	compose := "services: {}\n"
	for _, entries := range [][]ManagedWorkspaceEntry{
		{{Path: "config/child", NodeType: "file", Content: &compose}, {Path: "config", NodeType: "file", Content: &compose}, {Path: "compose.yaml", NodeType: "file", Content: &compose}},
		{{Path: "config", NodeType: "file", Content: &compose}, {Path: "config/child", NodeType: "file", Content: &compose}, {Path: "compose.yaml", NodeType: "file", Content: &compose}},
	} {
		if _, err := normalizeManagedWorkspaceEntries(entries, "compose.yaml"); !errors.Is(err, errProjectInvalidArgument) {
			t.Fatalf("expected file ancestor conflict, got %v", err)
		}
	}
}
