package contract

import "testing"

func TestDependencyObservabilityContractUsesOnlySupportedHistoryWindows(t *testing.T) {
	t.Parallel()

	windows := []TrendRange{TrendRange10Minutes, TrendRange30Minutes, TrendRange1Hour}
	for _, window := range windows {
		if !window.IsSupported() || window.Duration() <= 0 {
			t.Fatalf("expected supported positive duration for %q", window)
		}
	}

	for _, unsupported := range []TrendRange{"24h", "7d"} {
		if unsupported.IsSupported() {
			t.Fatalf("unsupported history window %q must be rejected by the contract", unsupported)
		}
	}
}

func TestDependencyHistoryContractValuesAreStable(t *testing.T) {
	t.Parallel()

	if DependencyKindPostgreSQL != "postgresql" || DependencyKindRedis != "redis" {
		t.Fatalf("unexpected dependency kinds: %q, %q", DependencyKindPostgreSQL, DependencyKindRedis)
	}
	if DependencyHistoryStatusAvailable != "available" || DependencyHistoryStatusPartial != "partial" || DependencyHistoryStatusUnavailable != "unavailable" {
		t.Fatalf("unexpected history statuses: %q, %q, %q", DependencyHistoryStatusAvailable, DependencyHistoryStatusPartial, DependencyHistoryStatusUnavailable)
	}
	if DependencyHistoryUnavailableRedisNotConfigured != "redis_not_configured" ||
		DependencyHistoryUnavailableReadFailed != "read_failed" {
		t.Fatalf("unexpected unavailable reasons: %q, %q", DependencyHistoryUnavailableRedisNotConfigured, DependencyHistoryUnavailableReadFailed)
	}
}
