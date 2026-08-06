package capability

import (
	"context"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

type provider func(context.Context) (moduleapi.CapabilityObservation, error)

func (p provider) Observe(ctx context.Context) (moduleapi.CapabilityObservation, error) {
	return p(ctx)
}

func TestCoordinatorRefreshAndStale(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	registry, err := NewRegistry([]Entry{{Descriptor: moduleapi.CapabilityDescriptor{Key: "docker", Category: moduleapi.CapabilityCategoryInfrastructure, Impact: moduleapi.CapabilityImpactFeature, StaleAfter: time.Minute}, Provider: provider(func(context.Context) (moduleapi.CapabilityObservation, error) {
		return moduleapi.CapabilityObservation{Status: moduleapi.CapabilityStatusHealthy}, nil
	})}})
	if err != nil {
		t.Fatal(err)
	}
	coordinator := NewCoordinator(registry)
	coordinator.now = func() time.Time { return now }
	if _, err := coordinator.Observe(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	got := coordinator.Snapshot()["docker"]
	if got.Status != moduleapi.CapabilityStatusUnknown || !got.Stale || got.ExpiresAt.IsZero() {
		t.Fatalf("expected expired observation projected to unknown, got %#v", got)
	}
	if got, ok := coordinator.Get("docker"); !ok || !got.Stale {
		t.Fatalf("expected stale lookup, got %#v, %v", got, ok)
	}
}

func TestRegistryRejectsInvalidEntries(t *testing.T) {
	_, err := NewRegistry([]Entry{{Descriptor: moduleapi.CapabilityDescriptor{Key: "x", Category: moduleapi.CapabilityCategoryInfrastructure, Impact: moduleapi.CapabilityImpactFeature}}, {Descriptor: moduleapi.CapabilityDescriptor{Key: "x", Category: moduleapi.CapabilityCategoryInfrastructure, Impact: moduleapi.CapabilityImpactFeature}, Provider: provider(func(context.Context) (moduleapi.CapabilityObservation, error) {
		return moduleapi.CapabilityObservation{}, nil
	})}})
	if err == nil {
		t.Fatal("expected duplicate/provider validation error")
	}
}

func TestRegistryRejectsInvalidDescriptorValues(t *testing.T) {
	provider := provider(func(context.Context) (moduleapi.CapabilityObservation, error) {
		return moduleapi.CapabilityObservation{Status: moduleapi.CapabilityStatusHealthy}, nil
	})
	for name, descriptor := range map[string]moduleapi.CapabilityDescriptor{
		"category": {Key: "x", Category: "unknown", Impact: moduleapi.CapabilityImpactFeature},
		"impact":   {Key: "x", Category: moduleapi.CapabilityCategoryInfrastructure, Impact: "unknown"},
		"ttl":      {Key: "x", Category: moduleapi.CapabilityCategoryInfrastructure, Impact: moduleapi.CapabilityImpactFeature, StaleAfter: -time.Second},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewRegistry([]Entry{{Descriptor: descriptor, Provider: provider}}); err == nil {
				t.Fatalf("expected invalid %s to be rejected", name)
			}
		})
	}
}

func TestCoordinatorRejectsInvalidProviderStatus(t *testing.T) {
	registry, err := NewRegistry([]Entry{{Descriptor: moduleapi.CapabilityDescriptor{Key: "x", Category: moduleapi.CapabilityCategoryInfrastructure, Impact: moduleapi.CapabilityImpactFeature}, Provider: provider(func(context.Context) (moduleapi.CapabilityObservation, error) {
		return moduleapi.CapabilityObservation{Status: "stale"}, nil
	})}})
	if err != nil {
		t.Fatal(err)
	}
	coordinator := NewCoordinator(registry)
	got, err := coordinator.Observe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got["x"].Status != moduleapi.CapabilityStatusUnavailable {
		t.Fatalf("expected invalid status to normalize to unavailable, got %#v", got["x"])
	}
}
