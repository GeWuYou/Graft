package project

import (
	"context"
	"testing"

	projectstore "graft/server/modules/project/store"
)

type composeContextReferenceRepository struct {
	*stubProjectRepository
	contexts []projectstore.ComposeContext
}

func (r *composeContextReferenceRepository) ResolveComposeContexts(
	_ context.Context,
	contexts []projectstore.ComposeContext,
) ([]projectstore.ComposeApplicationReference, error) {
	r.contexts = append([]projectstore.ComposeContext(nil), contexts...)
	return []projectstore.ComposeApplicationReference{{
		ComposeContext: projectstore.ComposeContext{RuntimeTargetID: 7, ComposeProjectName: "gitea"},
		ApplicationID:  "app_gitea",
		DisplayName:    "Gitea",
	}}, nil
}

func TestResolveComposeContextReferencesUsesCanonicalIdentity(t *testing.T) {
	t.Parallel()

	repository := &composeContextReferenceRepository{stubProjectRepository: &stubProjectRepository{}}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	items, err := service.ResolveComposeContextReferences(context.Background(), []ComposeContextReferenceRequest{
		{RuntimeTargetID: 7, ComposeProjectName: " gitea "},
		{RuntimeTargetID: 7, ComposeProjectName: "gitea"},
	})
	if err != nil {
		t.Fatalf("resolve compose context references: %v", err)
	}
	if len(repository.contexts) != 1 || repository.contexts[0] != (projectstore.ComposeContext{RuntimeTargetID: 7, ComposeProjectName: "gitea"}) {
		t.Fatalf("unexpected repository contexts: %#v", repository.contexts)
	}
	if len(items) != 1 || items[0].ApplicationID != "app_gitea" || items[0].DisplayName != "Gitea" {
		t.Fatalf("unexpected resolved references: %#v", items)
	}
}
