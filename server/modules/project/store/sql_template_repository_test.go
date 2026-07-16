package store

import (
	"context"
	"errors"
	"testing"
)

func TestTemplateRepositoryPublishesImmutableDraftAndDerivesNextDraft(t *testing.T) {
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
	derived, err := repository.DeriveTemplateDraft(ctx, DeriveTemplateDraftInput{TemplateID: created.Template.ID, SourceVersionID: created.Version.ID, VersionID: "tplv_01ARZ3NDEKTSV4RRFFQ69G5FAX", ActorID: nil})
	if err != nil {
		t.Fatalf("derive published version: %v", err)
	}
	if derived.Version.Status != "draft" || derived.Version.VersionNumber != 2 {
		t.Fatalf("unexpected derived draft: %#v", derived.Version)
	}
	if _, err = repository.DeriveTemplateDraft(ctx, DeriveTemplateDraftInput{TemplateID: created.Template.ID, SourceVersionID: created.Version.ID, VersionID: "tplv_01ARZ3NDEKTSV4RRFFQ69G5FAY"}); !errors.Is(err, ErrTemplateConflict) {
		t.Fatalf("expected one-draft conflict, got %v", err)
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
