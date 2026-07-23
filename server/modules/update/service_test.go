package update

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/buildinfo"
)

func TestServiceRetainsSuccessfulCatalogWhenNextCheckFails(t *testing.T) {
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
