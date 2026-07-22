package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/statex"
	monitorcontract "graft/server/modules/monitor/contract"
)

func TestBuildDependencyHistoryReadsSelectedRedisWindow(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	key := dependencyHistoryStorageKey("", resolveHostName(), "unstarted", monitorcontract.DependencyKindPostgreSQL)
	point := generated.ServerStatusDependencyHistoryPoint{ObservedAt: observedAt.Add(-15 * time.Minute)}
	availabilityPercent := float32(98.5)
	point.AvailabilityPercent = &availabilityPercent
	payload, err := json.Marshal(point)
	if err != nil {
		t.Fatalf("marshal dependency history point: %v", err)
	}
	store := &dependencyHistoryStoreStub{samplesByKey: map[string][]statex.TimeSeriesSample{
		key: {{ObservedAt: point.ObservedAt, Payload: payload}},
	}}

	history := buildDependencyHistory(
		context.Background(),
		nil,
		&Module{trendStore: store},
		observedAt,
		monitorcontract.TrendRange30Minutes,
		monitorcontract.DependencyKindPostgreSQL,
	)

	if history.Status != generated.ServerStatusDependencyHistoryStatus(monitorcontract.DependencyHistoryStatusAvailable) {
		t.Fatalf("expected available history, got %q", history.Status)
	}
	if len(history.Points) != 1 || history.Points[0].AvailabilityPercent == nil || *history.Points[0].AvailabilityPercent != availabilityPercent {
		t.Fatalf("unexpected history points: %#v", history.Points)
	}
	if len(store.queries) != 1 {
		t.Fatalf("expected one history query, got %d", len(store.queries))
	}
	query := store.queries[0]
	if query.key != key || !query.query.StartAt.Equal(observedAt.Add(-monitorcontract.TrendRange30Minutes.Duration())) || !query.query.EndAt.Equal(observedAt) {
		t.Fatalf("unexpected dependency history query: %#v", query)
	}
}

func TestBuildDependencyHistoryReportsUnavailableWithoutFailingCurrentSnapshot(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	history := buildDependencyHistory(
		context.Background(),
		nil,
		&Module{trendStore: &dependencyHistoryStoreStub{rangeErr: errors.New("redis read failed")}},
		observedAt,
		monitorcontract.TrendRange10Minutes,
		monitorcontract.DependencyKindRedis,
	)

	if history.Status != generated.ServerStatusDependencyHistoryStatus(monitorcontract.DependencyHistoryStatusUnavailable) {
		t.Fatalf("expected unavailable history, got %q", history.Status)
	}
	if history.UnavailableReason == nil || *history.UnavailableReason != generated.ServerStatusDependencyHistoryUnavailableReason(monitorcontract.DependencyHistoryUnavailableReadFailed) {
		t.Fatalf("unexpected unavailable reason: %#v", history.UnavailableReason)
	}
	if len(history.Points) != 0 {
		t.Fatalf("expected no history points, got %#v", history.Points)
	}
}

func TestDependencyHistoryCancellationIsNotTreatedAsDependencyFailure(t *testing.T) {
	if !isExpectedDependencyHistoryCancellation(fmt.Errorf("write failed: %w", context.Canceled)) {
		t.Fatal("expected wrapped cancellation to be recognized")
	}
	if !isExpectedDependencyHistoryCancellation(context.DeadlineExceeded) {
		t.Fatal("expected deadline exceeded to be recognized")
	}
	if isExpectedDependencyHistoryCancellation(errors.New("redis unavailable")) {
		t.Fatal("unexpectedly recognized redis failure as cancellation")
	}
}

func TestBuildDependencyHistoryReportsRedisNotConfigured(t *testing.T) {
	t.Parallel()

	history := buildDependencyHistory(
		context.Background(),
		nil,
		&Module{},
		time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC),
		monitorcontract.TrendRange10Minutes,
		monitorcontract.DependencyKindRedis,
	)

	if history.Status != generated.ServerStatusDependencyHistoryStatus(monitorcontract.DependencyHistoryStatusUnavailable) {
		t.Fatalf("expected unavailable history, got %q", history.Status)
	}
	if history.UnavailableReason == nil || *history.UnavailableReason != generated.ServerStatusDependencyHistoryUnavailableReason(monitorcontract.DependencyHistoryUnavailableRedisNotConfigured) {
		t.Fatalf("unexpected unavailable reason: %#v", history.UnavailableReason)
	}
	if len(history.Points) != 0 {
		t.Fatalf("expected no history points, got %#v", history.Points)
	}
}

func TestBuildDependencyHistoryReportsPartialWhenSamplesCannotBeDecoded(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	point := generated.ServerStatusDependencyHistoryPoint{ObservedAt: observedAt.Add(-time.Minute)}
	payload, err := json.Marshal(point)
	if err != nil {
		t.Fatalf("marshal dependency history point: %v", err)
	}
	key := dependencyHistoryStorageKey("", resolveHostName(), "unstarted", monitorcontract.DependencyKindRedis)
	history := buildDependencyHistory(
		context.Background(),
		nil,
		&Module{trendStore: &dependencyHistoryStoreStub{samplesByKey: map[string][]statex.TimeSeriesSample{
			key: {
				{ObservedAt: point.ObservedAt, Payload: payload},
				{ObservedAt: observedAt.Add(-2 * time.Minute), Payload: []byte("not-json")},
			},
		}}},
		observedAt,
		monitorcontract.TrendRange10Minutes,
		monitorcontract.DependencyKindRedis,
	)

	if history.Status != generated.ServerStatusDependencyHistoryStatus(monitorcontract.DependencyHistoryStatusPartial) {
		t.Fatalf("expected partial history, got %q", history.Status)
	}
	if len(history.Points) != 1 {
		t.Fatalf("expected one decoded history point, got %#v", history.Points)
	}
}

func TestDependencyHistoryStorageKeySeparatesDeploymentInstances(t *testing.T) {
	t.Parallel()

	first := dependencyHistoryStorageKey("foo.bar", "host", "deployment-one", monitorcontract.DependencyKindRedis)
	second := dependencyHistoryStorageKey("foo-bar", "host", "deployment-two", monitorcontract.DependencyKindRedis)
	if first == second {
		t.Fatalf("expected deployment history keys to remain distinct, got %q", first)
	}
	if dependencyHistoryStorageKey("app", "host", "deployment-one", monitorcontract.DependencyKindRedis) == dependencyHistoryStorageKey("app", "host", "deployment-two", monitorcontract.DependencyKindRedis) {
		t.Fatal("expected distinct deployment IDs to produce distinct history keys")
	}
	if dependencyHistoryStorageKey("foo.bar", "host", "deployment", monitorcontract.DependencyKindRedis) == dependencyHistoryStorageKey("foo-bar", "host", "deployment", monitorcontract.DependencyKindRedis) {
		t.Fatal("expected distinct raw app names to produce distinct history keys")
	}
}

type dependencyHistoryQuery struct {
	key   string
	query statex.TimeSeriesQuery
}

type dependencyHistoryStoreStub struct {
	samplesByKey map[string][]statex.TimeSeriesSample
	queries      []dependencyHistoryQuery
	rangeErr     error
}

func (s *dependencyHistoryStoreStub) Append(context.Context, string, statex.TimeSeriesSample, statex.RetentionPolicy) error {
	return nil
}

func (s *dependencyHistoryStoreStub) Range(_ context.Context, key string, query statex.TimeSeriesQuery) ([]statex.TimeSeriesSample, error) {
	s.queries = append(s.queries, dependencyHistoryQuery{key: key, query: query})
	if s.rangeErr != nil {
		return nil, s.rangeErr
	}
	return s.samplesByKey[key], nil
}
