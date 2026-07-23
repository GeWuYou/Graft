package update

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/buildinfo"
)

type releaseProviderFunc func(context.Context) ([]Release, error)

func (f releaseProviderFunc) List(ctx context.Context) ([]Release, error) {
	return f(ctx)
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
