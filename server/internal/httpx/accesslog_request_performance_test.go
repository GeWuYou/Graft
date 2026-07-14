package httpx

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestAccessLogRepositoryReadRequestPerformance(t *testing.T) {
	repository := newSQLiteAccessLogRepository(t)
	reader, ok := repository.(moduleapi.RequestPerformanceReader)
	if !ok {
		t.Fatal("expected access log repository to expose request performance reader")
	}
	base := time.Date(2026, 7, 14, 8, 0, 0, 0, time.UTC)
	seedRequestPerformanceAccessLogs(t, repository, base)

	summary, err := reader.ReadRequestPerformance(context.Background(), moduleapi.RequestPerformanceQuery{
		WindowStart: base,
		WindowEnd:   base.Add(3 * time.Minute),
		BucketSize:  moduleapi.RequestPerformanceMinuteBucketSize,
	})
	if err != nil {
		t.Fatalf("read request performance: %v", err)
	}
	assertRequestPerformanceSummary(t, summary)
	assertRequestPerformanceBuckets(t, summary.Buckets)
	assertRequestPerformanceTopRoutes(t, summary.TopRoutes)
}

func assertRequestPerformanceSummary(t *testing.T, summary moduleapi.RequestPerformanceSummary) {
	t.Helper()
	if summary.TotalRequests != 6 || summary.ServerErrorCount != 2 || summary.SlowRequestCount != 2 {
		t.Fatalf("unexpected totals: %#v", summary)
	}
	if summary.P50LatencyMS != 100 || summary.P95LatencyMS != 1500 {
		t.Fatalf("unexpected percentiles: p50=%d p95=%d", summary.P50LatencyMS, summary.P95LatencyMS)
	}
	if summary.StatusGroups != (moduleapi.RequestPerformanceStatusGroups{TwoXX: 2, ThreeXX: 1, FourXX: 1, FiveXX: 2}) {
		t.Fatalf("unexpected status groups: %#v", summary.StatusGroups)
	}
}

func assertRequestPerformanceBuckets(t *testing.T, buckets []moduleapi.RequestPerformanceMinuteBucket) {
	t.Helper()
	if len(buckets) != 3 {
		t.Fatalf("expected three buckets, got %#v", buckets)
	}
	if buckets[0].TotalRequests != 3 || buckets[0].ServerErrorCount != 1 || buckets[0].P95LatencyMS != 1200 {
		t.Fatalf("unexpected first bucket: %#v", buckets[0])
	}
	if buckets[1].TotalRequests != 3 || buckets[1].ServerErrorCount != 1 || buckets[1].P95LatencyMS != 1500 {
		t.Fatalf("unexpected second bucket: %#v", buckets[1])
	}
	if buckets[2].TotalRequests != 0 || buckets[2].P95LatencyMS != 0 {
		t.Fatalf("expected zero-filled final bucket, got %#v", buckets[2])
	}
}

func assertRequestPerformanceTopRoutes(t *testing.T, routes moduleapi.RequestPerformanceTopRoutes) {
	t.Helper()
	if len(routes.ByTraffic) != 4 || routes.ByTraffic[0].Method != "GET" || routes.ByTraffic[0].Route != "/api/reports" || routes.ByTraffic[0].TotalRequests != 2 {
		t.Fatalf("unexpected traffic routes: %#v", routes.ByTraffic)
	}
	if len(routes.ByServerErrors) != 2 || routes.ByServerErrors[0].Route != "/api/reports" || routes.ByServerErrors[0].ServerErrorCount != 1 {
		t.Fatalf("unexpected server error routes: %#v", routes.ByServerErrors)
	}
	if len(routes.ByP95Latency) != 4 || routes.ByP95Latency[0].Route != "/api/reports" || routes.ByP95Latency[0].P95LatencyMS != 1500 {
		t.Fatalf("unexpected latency routes: %#v", routes.ByP95Latency)
	}
}

func TestAccessLogRepositoryReadRequestPerformanceRejectsInvalidQuery(t *testing.T) {
	repository := newSQLiteAccessLogRepository(t)
	reader := repository.(moduleapi.RequestPerformanceReader)
	base := time.Date(2026, 7, 14, 8, 0, 0, 0, time.UTC)
	_, err := reader.ReadRequestPerformance(context.Background(), moduleapi.RequestPerformanceQuery{
		WindowStart: base,
		WindowEnd:   base.Add(time.Minute),
		BucketSize:  5 * time.Minute,
	})
	if err != moduleapi.ErrRequestPerformanceInvalidQuery {
		t.Fatalf("expected invalid query error, got %v", err)
	}
}

func TestRequestPerformanceUsesCanonicalUnmatchedRouteAndBoundedLatencySamples(t *testing.T) {
	base := time.Date(2026, 7, 14, 8, 0, 0, 0, time.UTC)
	query := moduleapi.RequestPerformanceQuery{
		WindowStart: base,
		WindowEnd:   base.Add(time.Minute),
		BucketSize:  moduleapi.RequestPerformanceMinuteBucketSize,
	}
	collector := newRequestPerformanceCollector(query)
	collector.add(base.Add(time.Second), "GET", requestPerformanceUnmatchedRoute, 200, 100, query.BucketSize)
	for index := 0; index < requestPerformanceLatencySampleLimit+10; index++ {
		collector.add(base.Add(2*time.Second), "GET", "/api/users", 200, int64(index), query.BucketSize)
	}

	summary := collector.summaryWithRankings()
	if len(collector.latencies.values) != requestPerformanceLatencySampleLimit {
		t.Fatalf("expected bounded overall latency samples, got %d", len(collector.latencies.values))
	}
	if len(collector.routes[requestPerformanceRouteKey{method: "GET", route: "/api/users"}].latencies.values) != requestPerformanceLatencySampleLimit {
		t.Fatalf("expected bounded route latency samples")
	}
	if len(summary.TopRoutes.ByP95Latency) != 2 || summary.TopRoutes.ByP95Latency[0].Route != "/api/users" {
		t.Fatalf("expected route rankings to include canonical routes, got %#v", summary.TopRoutes.ByP95Latency)
	}
	if summary.TopRoutes.ByP95Latency[1].Route != requestPerformanceUnmatchedRoute {
		t.Fatalf("expected unmatched route marker, got %#v", summary.TopRoutes.ByP95Latency)
	}
	if got := requestPerformanceRouteValue(sql.NullString{}); got != requestPerformanceUnmatchedRoute {
		t.Fatalf("expected NULL route marker %q, got %q", requestPerformanceUnmatchedRoute, got)
	}
}

func seedRequestPerformanceAccessLogs(t *testing.T, repository AccessLogRepository, base time.Time) {
	t.Helper()
	inputs := []CreateAccessLogInput{
		{RequestID: "in-window-1", Method: "GET", Path: "/api/users", Route: "/api/users", StatusCode: 200, DurationMS: 100, OccurredAt: base.Add(10 * time.Second)},
		{RequestID: "in-window-2", Method: "GET", Path: "/api/users", Route: "/api/users", StatusCode: 500, DurationMS: 1200, OccurredAt: base.Add(20 * time.Second)},
		{RequestID: "in-window-3", Method: "POST", Path: "/api/users", Route: "/api/users", StatusCode: 201, DurationMS: 200, OccurredAt: base.Add(30 * time.Second)},
		{RequestID: "in-window-4", Method: "GET", Path: "/api/reports", Route: "/api/reports", StatusCode: 502, DurationMS: 1500, OccurredAt: base.Add(time.Minute + 5*time.Second)},
		{RequestID: "in-window-5", Method: "GET", Path: "/api/reports", Route: "/api/reports", StatusCode: 404, DurationMS: 20, OccurredAt: base.Add(time.Minute + 10*time.Second)},
		{RequestID: "in-window-6", Method: "GET", Path: "/healthz", Route: "/healthz", StatusCode: 302, DurationMS: 50, OccurredAt: base.Add(time.Minute + 15*time.Second)},
		{RequestID: "outside-window", Method: "GET", Path: "/api/ignored", Route: "/api/ignored", StatusCode: 500, DurationMS: 9999, OccurredAt: base.Add(3 * time.Minute)},
	}
	if _, err := repository.CreateAccessLogs(context.Background(), inputs); err != nil {
		t.Fatalf("seed request performance access logs: %v", err)
	}
}
