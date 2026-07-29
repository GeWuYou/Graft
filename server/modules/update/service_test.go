package update

import (
	"context"
	"errors"
	"slices"
	"testing"

	"graft/server/internal/buildinfo"
)

func TestServiceRetainsSuccessfulCatalogWhenNextCheckFails(t *testing.T) {
	t.Setenv(imageTagEnv, "latest")
	cache := &memoryDiscoveryCache{}
	provider := &stubReleaseProvider{releases: []Release{{Version: "1.1.0", Channel: "stable"}}}
	service := NewServiceWithCache(provider, cache)
	service.current = func() buildinfo.Info { return buildinfo.Info{Version: "1.0.0"} }
	if status := service.Check(context.Background()); status.Latest == nil || status.CacheStale {
		t.Fatalf("expected fresh candidate, got %#v", status)
	}
	provider.err = errors.New("github unavailable")
	status := service.Check(context.Background())
	if status.Latest == nil || status.Latest.Version != "1.1.0" || !status.CacheStale || status.CheckError != releaseDiscoveryFailedMessage {
		t.Fatalf("expected stale cached candidate after failure, got %#v", status)
	}
	if cache.snapshot.LastSuccessfulAt == nil || cache.snapshot.LastAttemptAt == nil {
		t.Fatalf("expected persisted attempt and successful timestamps: %#v", cache.snapshot)
	}
}

func TestStatusWithoutComposeCandidatesKeepsDiscoverySnapshotIntact(t *testing.T) {
	status := Status{Profile: InstallationProfile{ComposeCandidates: []ComposeRootCandidate{{CandidateKey: "compose-a", Root: "/srv/graft"}}}}

	redacted := status.withoutComposeCandidates()

	if len(redacted.Profile.ComposeCandidates) != 0 {
		t.Fatalf("expected read response candidates to be redacted, got %#v", redacted.Profile.ComposeCandidates)
	}
	if len(status.Profile.ComposeCandidates) != 1 || status.Profile.ComposeCandidates[0].Root != "/srv/graft" {
		t.Fatalf("expected source discovery snapshot to remain intact, got %#v", status.Profile.ComposeCandidates)
	}
}

func TestStatusDoesNotSelectReleaseForInvalidCurrentVersion(t *testing.T) {
	t.Setenv(imageTagEnv, "latest")
	service := NewService(nil)
	service.current = func() buildinfo.Info { return buildinfo.Info{Version: "development"} }
	service.catalog = []Release{{Version: "1.1.0", Channel: "stable"}}

	status := service.Status()

	if status.Latest != nil {
		t.Fatalf("invalid current version must not select a release: %#v", status.Latest)
	}
}

func TestStatusDerivesTrackingAndPinnedChoicesFromImageTag(t *testing.T) {
	tests := []struct {
		name       string
		tag        string
		wantMode   UpdateMode
		wantLatest string
		wantList   []string
	}{
		{name: "stable tracking", tag: "latest", wantMode: UpdateModeStableTracking, wantLatest: "1.2.0", wantList: []string{"1.1.0", "1.2.0"}},
		{name: "beta tracking", tag: "beta", wantMode: UpdateModeBetaTracking, wantLatest: "1.2.0-beta.2", wantList: []string{"1.1.0-beta.1", "1.2.0-beta.2"}},
		{name: "pinned stable", tag: "v1.0.0", wantMode: UpdateModePinnedStable, wantList: []string{"1.1.0", "1.2.0"}},
		{name: "pinned beta", tag: "v1.0.0-beta.1", wantMode: UpdateModePinnedBeta, wantList: []string{"1.1.0-beta.1", "1.2.0-beta.2"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(imageTagEnv, test.tag)
			service := NewService(nil)
			service.current = func() buildinfo.Info { return buildinfo.Info{Version: "1.0.0-beta.1"} }
			service.catalog = []Release{{Version: "1.1.0-beta.1", Channel: "beta"}, {Version: "1.2.0-beta.2", Channel: "beta"}, {Version: "1.1.0", Channel: "stable"}, {Version: "1.2.0", Channel: "stable"}}
			status := service.Status()
			if status.ImageTag != test.tag || status.UpdateMode != test.wantMode {
				t.Fatalf("strategy = (%q, %q), want (%q, %q)", status.ImageTag, status.UpdateMode, test.tag, test.wantMode)
			}
			if got := releaseVersions(status.AvailableReleases); !slices.Equal(got, test.wantList) {
				t.Fatalf("available releases = %#v, want %#v", got, test.wantList)
			}
			if got := versionOrEmpty(status.Latest); got != test.wantLatest {
				t.Fatalf("latest = %q, want %q", got, test.wantLatest)
			}
		})
	}
}

func releaseVersions(releases []Release) []string {
	versions := make([]string, 0, len(releases))
	for _, release := range releases {
		versions = append(versions, release.Version)
	}
	return versions
}

func versionOrEmpty(release *Release) string {
	if release == nil {
		return ""
	}
	return release.Version
}

type stubReleaseProvider struct {
	releases []Release
	err      error
}

func (s *stubReleaseProvider) List(context.Context) ([]Release, error) { return s.releases, s.err }

type memoryDiscoveryCache struct{ snapshot DiscoverySnapshot }

func (s *memoryDiscoveryCache) Load(context.Context) (DiscoverySnapshot, error) {
	return s.snapshot, nil
}
func (s *memoryDiscoveryCache) Save(_ context.Context, value DiscoverySnapshot) error {
	s.snapshot = value
	return nil
}
