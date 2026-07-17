package store

import (
	"context"
	"errors"
	"testing"
)

func TestTemplateRepositoryWithdrawsPublishedVersionAndCreatesNextDraft(t *testing.T) {
	t.Parallel()
	repository, _ := newTestSQLRepository(t)
	ctx := context.Background()
	created, err := repository.CreateTemplateDraft(ctx, CreateTemplateDraftInput{TemplateID: "tpl_01ARZ3NDEKTSV4RRFFQ69G5FAV", VersionID: "tplv_01ARZ3NDEKTSV4RRFFQ69G5FAV", DisplayName: "Redis", DeploymentAdapterKind: "compose", DefinitionSchemaVersion: 1, DefinitionJSON: []byte(`{"compose_file_path":"compose.yaml","workspace_entries":[]}`)})
	if err != nil {
		t.Fatalf("create template draft: %v", err)
	}
	published, err := repository.PublishTemplateDraft(ctx, created.Template.ID, nil)
	if err != nil {
		t.Fatalf("publish template draft: %v", err)
	}
	if published.Version.Status != "published" || published.Version.PublishedAt == nil {
		t.Fatalf("unexpected published version: %#v", published.Version)
	}
	if _, err = repository.UpdateTemplateDraft(ctx, UpdateTemplateDraftInput{TemplateID: created.Template.ID, DisplayName: "Redis", DefinitionSchemaVersion: 1, DefinitionJSON: []byte(`{}`)}); !errors.Is(err, ErrTemplateDraftNotFound) {
		t.Fatalf("published version must not be editable, got %v", err)
	}
	withdrawn, err := repository.WithdrawTemplate(ctx, WithdrawTemplateInput{TemplateID: created.Template.ID, VersionID: "tplv_01ARZ3NDEKTSV4RRFFQ69G5FAX", ActorID: nil})
	if err != nil {
		t.Fatalf("withdraw published version: %v", err)
	}
	if withdrawn.Version.Status != "draft" || withdrawn.Version.VersionNumber != 2 || string(withdrawn.Version.DefinitionJSON) != string(created.Version.DefinitionJSON) {
		t.Fatalf("unexpected withdrawn draft: %#v", withdrawn.Version)
	}
	if _, err = repository.GetPublishedTemplateVersion(ctx, published.Version.ID); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("withdrawn version must not be instantiable, got %v", err)
	}
	if _, err = repository.WithdrawTemplate(ctx, WithdrawTemplateInput{TemplateID: created.Template.ID, VersionID: "tplv_01ARZ3NDEKTSV4RRFFQ69G5FAY"}); !errors.Is(err, ErrTemplateDraftNotFound) {
		t.Fatalf("expected no published version after withdraw, got %v", err)
	}
}

func TestTemplateRepositoryClonesCurrentDefinition(t *testing.T) {
	t.Parallel()
	repository, _ := newTestSQLRepository(t)
	ctx := context.Background()
	created, err := repository.CreateTemplateDraft(ctx, CreateTemplateDraftInput{TemplateID: "tpl_01ARZ3NDEKTSV4RRFFQ69G5FA1", VersionID: "tplv_01ARZ3NDEKTSV4RRFFQ69G5FA1", DisplayName: "Source", Description: "source definition", DeploymentAdapterKind: "compose", DefinitionSchemaVersion: 1, DefinitionJSON: []byte(`{"compose_file_path":"compose.yaml","workspace_entries":[]}`)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cloned, err := repository.CloneTemplate(ctx, CloneTemplateInput{SourceTemplateID: created.Template.ID, TemplateID: "tpl_01ARZ3NDEKTSV4RRFFQ69G5FA2", VersionID: "tplv_01ARZ3NDEKTSV4RRFFQ69G5FA2", DisplayName: "Clone"})
	if err != nil {
		t.Fatalf("clone: %v", err)
	}
	if cloned.Template.ID == created.Template.ID || cloned.Template.DisplayName != "Clone" || cloned.Template.Description != created.Template.Description || cloned.Version.Status != "draft" || string(cloned.Version.DefinitionJSON) != string(created.Version.DefinitionJSON) {
		t.Fatalf("unexpected clone: %#v", cloned)
	}
	if err = repository.ArchiveTemplate(ctx, created.Template.ID, nil); err != nil {
		t.Fatalf("archive source: %v", err)
	}
	archivedClone, err := repository.CloneTemplate(ctx, CloneTemplateInput{SourceTemplateID: created.Template.ID, TemplateID: "tpl_01ARZ3NDEKTSV4RRFFQ69G5FA3", VersionID: "tplv_01ARZ3NDEKTSV4RRFFQ69G5FA3", DisplayName: "Archived clone"})
	if err != nil || archivedClone.Template.ArchivedAt != nil || archivedClone.Version.Status != "draft" {
		t.Fatalf("archived source must clone to a live draft: clone=%#v err=%v", archivedClone, err)
	}
}

func TestTemplateRepositorySoftDeletesTemplate(t *testing.T) {
	t.Parallel()
	repository, _ := newTestSQLRepository(t)
	ctx := context.Background()
	created, err := repository.CreateTemplateDraft(ctx, CreateTemplateDraftInput{TemplateID: "tpl_01ARZ3NDEKTSV4RRFFQ69G5FA4", VersionID: "tplv_01ARZ3NDEKTSV4RRFFQ69G5FA4", DisplayName: "Delete source", DeploymentAdapterKind: "compose", DefinitionSchemaVersion: 1, DefinitionJSON: []byte(`{"compose_file_path":"compose.yaml","workspace_entries":[]}`)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err = repository.DeleteTemplate(ctx, created.Template.ID, nil); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err = repository.GetTemplate(ctx, created.Template.ID); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("deleted template must be hidden, got %v", err)
	}
	items, err := repository.ListTemplates(ctx, TemplateListQuery{IncludeArchived: true})
	if err != nil || len(items) != 0 {
		t.Fatalf("deleted template must be absent from management catalog: items=%#v err=%v", items, err)
	}
}

func TestTemplateRepositoryHidesArchivedTemplatesFromCreatorCatalog(t *testing.T) {
	t.Parallel()
	repository, _ := newTestSQLRepository(t)
	ctx := context.Background()
	created, err := repository.CreateTemplateDraft(ctx, CreateTemplateDraftInput{TemplateID: "tpl_01ARZ3NDEKTSV4RRFFQ69G5FAW", VersionID: "tplv_01ARZ3NDEKTSV4RRFFQ69G5FAW", DisplayName: "Gitea", DeploymentAdapterKind: "compose", DefinitionSchemaVersion: 1, DefinitionJSON: []byte(`{"compose_file_path":"compose.yaml","workspace_entries":[]}`)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err = repository.PublishTemplateDraft(ctx, created.Template.ID, nil); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if err = repository.ArchiveTemplate(ctx, created.Template.ID, nil); err != nil {
		t.Fatalf("archive: %v", err)
	}
	items, err := repository.ListTemplates(ctx, TemplateListQuery{DeploymentAdapterKind: "compose", PublishedOnly: true})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("archived templates must not enter creator catalog: %#v", items)
	}
}

func TestTemplateRepositoryManagedCatalogFallsBackToPublishedVersionWithoutDraft(t *testing.T) {
	t.Parallel()
	repository, _ := newTestSQLRepository(t)
	ctx := context.Background()
	created, err := repository.CreateTemplateDraft(ctx, CreateTemplateDraftInput{TemplateID: "tpl_01ARZ3NDEKTSV4RRFFQ69G5FAT", VersionID: "tplv_01ARZ3NDEKTSV4RRFFQ69G5FAT", DisplayName: "Postgres", DeploymentAdapterKind: "compose", DefinitionSchemaVersion: 1, DefinitionJSON: []byte(`{"compose_file_path":"compose.yaml"}`)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err = repository.PublishTemplateDraft(ctx, created.Template.ID, nil); err != nil {
		t.Fatalf("publish: %v", err)
	}
	items, err := repository.ListTemplates(ctx, TemplateListQuery{IncludeArchived: true})
	if err != nil {
		t.Fatalf("managed list must accept published-only templates: %v", err)
	}
	if len(items) != 1 || items[0].Version.Status != "published" || items[0].Version.ID != created.Version.ID {
		t.Fatalf("expected published fallback, got %#v", items)
	}
}

func TestTemplateRepositoryResolvesOnlyPublishedUnarchivedVersionsForCreation(t *testing.T) {
	t.Parallel()
	repository, _ := newTestSQLRepository(t)
	ctx := context.Background()
	created, err := repository.CreateTemplateDraft(ctx, CreateTemplateDraftInput{TemplateID: "tpl_01ARZ3NDEKTSV4RRFFQ69G5FAZ", VersionID: "tplv_01ARZ3NDEKTSV4RRFFQ69G5FAZ", DisplayName: "Grafana", DeploymentAdapterKind: "compose", DefinitionSchemaVersion: 1, DefinitionJSON: []byte(`{"compose_file_path":"compose.yaml","workspace_entries":[]}`)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err = repository.GetPublishedTemplateVersion(ctx, created.Version.ID); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("draft must not be instantiable, got %v", err)
	}
	published, err := repository.PublishTemplateDraft(ctx, created.Template.ID, nil)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	resolved, err := repository.GetPublishedTemplateVersion(ctx, published.Version.ID)
	if err != nil || resolved.Template.ID != created.Template.ID || resolved.Version.ID != published.Version.ID {
		t.Fatalf("resolve published version: item=%#v err=%v", resolved, err)
	}
	if err = repository.ArchiveTemplate(ctx, created.Template.ID, nil); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if _, err = repository.GetPublishedTemplateVersion(ctx, published.Version.ID); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("archived template must not be instantiable, got %v", err)
	}
}
