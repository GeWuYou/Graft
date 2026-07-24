package update

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/moduleapi"
)

type updateAuthorizerStub struct{ err error }

func (s updateAuthorizerStub) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, _ string) error {
	return s.err
}

func TestMayViewComposeCandidatesRequiresUpdateManagePermission(t *testing.T) {
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})

	if !mayViewComposeCandidates(ctx, updateAuthorizerStub{}) {
		t.Fatal("expected manage-authorized request to view compose candidates")
	}
	if mayViewComposeCandidates(ctx, updateAuthorizerStub{err: errors.New("denied")}) {
		t.Fatal("expected denied request to hide compose candidates")
	}
	if mayViewComposeCandidates(context.Background(), updateAuthorizerStub{}) {
		t.Fatal("expected request without auth context to hide compose candidates")
	}
}

func TestStatusForUpdateViewerRedactsEveryNonManageResponse(t *testing.T) {
	status := Status{Profile: InstallationProfile{ComposeCandidates: []ComposeRootCandidate{{CandidateKey: "compose-a", Root: "/srv/graft"}}}}
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})

	if got := statusForUpdateViewer(ctx, updateAuthorizerStub{}, status); len(got.Profile.ComposeCandidates) != 1 {
		t.Fatalf("expected manage response to preserve candidates, got %#v", got.Profile.ComposeCandidates)
	}
	if got := statusForUpdateViewer(ctx, updateAuthorizerStub{err: errors.New("denied")}, status); len(got.Profile.ComposeCandidates) != 0 {
		t.Fatalf("expected non-manage response to redact candidates, got %#v", got.Profile.ComposeCandidates)
	}
}
