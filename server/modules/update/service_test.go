package update

import (
	"context"
	"errors"
	"testing"
	"time"

	"graft/server/internal/buildinfo"
)

type releaseProviderFunc func(context.Context) ([]Release, error)

func (f releaseProviderFunc) List(ctx context.Context) ([]Release, error) {
	return f(ctx)
}

func TestServiceCheckRejectsInvalidCurrentVersionWithoutCallingProvider(t *testing.T) {
	providerCalled := false
	service := NewService(releaseProviderFunc(func(context.Context) ([]Release, error) {
		providerCalled = true
		return nil, nil
	}))
	service.current = func() buildinfo.Info { return buildinfo.Info{Version: "development"} }

	status := service.Check(context.Background())
	if providerCalled {
		t.Fatal("invalid current version must not call the release provider")
	}
	if status.CheckError != "current build is not a release semantic version" || status.Latest != nil || status.CheckedAt == nil {
		t.Fatalf("expected invalid version status without latest release, got %#v", status)
	}
}

func TestServiceCheckClearsLatestWhenNoNewReleaseExists(t *testing.T) {
	service := NewService(releaseProviderFunc(func(context.Context) ([]Release, error) {
		return []Release{{Version: "1.0.0", Channel: "stable"}, {Version: "0.9.0", Channel: "stable"}}, nil
	}))
	service.current = func() buildinfo.Info { return buildinfo.Info{Version: "1.0.0"} }

	status := service.Check(context.Background())
	if status.Latest != nil || status.CheckError != "" || status.CheckedAt == nil {
		t.Fatalf("expected checked status without a newer release, got %#v", status)
	}
}

func TestServiceStatusDefensivelyCopiesPointers(t *testing.T) {
	checkedAt := time.Date(2026, time.July, 23, 8, 0, 0, 0, time.UTC)
	service := NewService(nil)
	service.current = func() buildinfo.Info { return buildinfo.Info{Version: "1.0.0"} }
	service.profile = func() InstallationProfile { return InstallationProfile{} }
	service.latest = &Release{Version: "1.1.0", Channel: "stable"}
	service.checkedAt = &checkedAt

	status := service.Status()
	status.Latest.Version = "mutated"
	*status.CheckedAt = status.CheckedAt.Add(time.Hour)

	fresh := service.Status()
	if fresh.Latest == nil || fresh.Latest.Version != "1.1.0" {
		t.Fatalf("expected stored release to remain unchanged, got %#v", fresh.Latest)
	}
	if fresh.CheckedAt == nil || !fresh.CheckedAt.Equal(checkedAt) {
		t.Fatalf("expected stored timestamp to remain unchanged, got %#v", fresh.CheckedAt)
	}
}

func TestServiceCheckPreservesVerifiedReleaseWhenDiscoveryFails(t *testing.T) {
	provider := releaseProviderFunc(func(context.Context) ([]Release, error) {
		return nil, errors.New("GET https://releases.example.invalid/private: unexpected response")
	})
	service := NewService(provider)
	service.current = func() buildinfo.Info { return buildinfo.Info{Version: "1.0.0"} }
	service.latest = &Release{Version: "1.1.0", Channel: "stable"}

	status := service.Check(context.Background())
	if status.Latest == nil || status.Latest.Version != "1.1.0" {
		t.Fatalf("expected verified release to remain available, got %#v", status.Latest)
	}
	if status.CheckError != releaseDiscoveryFailedMessage {
		t.Fatalf("expected stable public error, got %q", status.CheckError)
	}
	if status.CheckError == "GET https://releases.example.invalid/private: unexpected response" {
		t.Fatal("provider error must not be exposed in status")
	}
}
