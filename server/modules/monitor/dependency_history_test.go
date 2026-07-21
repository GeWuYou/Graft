package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/statex"
	monitorcontract "graft/server/modules/monitor/contract"
)

func TestBuildDependencyHistoryReadsSelectedRedisWindow(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	key := dependencyHistoryStorageKey("app", resolveHostName(), monitorcontract.DependencyKindPostgreSQL)
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
