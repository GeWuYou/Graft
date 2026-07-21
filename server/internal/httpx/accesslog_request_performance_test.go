package httpx

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
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
	assertRequestPerformanceDistributions(t, summary)
	assertRequestPerformanceInstances(t, summary)
}

func assertRequestPerformanceSummary(t *testing.T, summary moduleapi.RequestPerformanceSummary) {
	t.Helper()
	if summary.TotalRequests != 6 || summary.ServerErrorCount != 2 || summary.SlowRequestCount != 2 {
		t.Fatalf("unexpected totals: %#v", summary)
	}
	assertRequestPerformanceLatencySummary(t, summary)
	assertRequestPerformanceByteSummary(t, summary)
	assertRequestPerformanceStatusSummary(t, summary)
}

func assertRequestPerformanceLatencySummary(t *testing.T, summary moduleapi.RequestPerformanceSummary) {
	t.Helper()
	if summary.P50LatencyMS != 100 || summary.P95LatencyMS != 1500 {
		t.Fatalf("unexpected percentiles: p50=%d p95=%d", summary.P50LatencyMS, summary.P95LatencyMS)
	}
	if summary.AverageLatencyMS != 511.6666666666667 || summary.P99LatencyMS != 1500 || summary.MaxLatencyMS != 1500 {
		t.Fatalf("unexpected extended latency summary: %#v", summary)
	}
}

func assertRequestPerformanceByteSummary(t *testing.T, summary moduleapi.RequestPerformanceSummary) {
	t.Helper()
	if summary.RequestBytes.MeasuredCount != 5 || summary.RequestBytes.TotalBytes != 118272 || summary.RequestBytes.AverageBytes != 23654.4 {
		t.Fatalf("unexpected request bytes: %#v", summary.RequestBytes)
	}
	if summary.ResponseBytes.MeasuredCount != 5 || summary.ResponseBytes.TotalBytes != 1283100 || summary.ResponseBytes.AverageBytes != 256620 {
		t.Fatalf("unexpected response bytes: %#v", summary.ResponseBytes)
	}
}

func assertRequestPerformanceStatusSummary(t *testing.T, summary moduleapi.RequestPerformanceSummary) {
	t.Helper()
	if summary.StatusGroups != (moduleapi.RequestPerformanceStatusGroups{TwoXX: 2, ThreeXX: 1, FourXX: 1, FiveXX: 2}) {
		t.Fatalf("unexpected status groups: %#v", summary.StatusGroups)
	}
	expectedStatusCodes := []moduleapi.RequestPerformanceStatusCodeCount{{StatusCode: 200, Count: 1}, {StatusCode: 201, Count: 1}, {StatusCode: 302, Count: 1}, {StatusCode: 404, Count: 1}, {StatusCode: 500, Count: 1}, {StatusCode: 502, Count: 1}}
	if !reflect.DeepEqual(summary.StatusCodes, expectedStatusCodes) {
		t.Fatalf("unexpected status codes: %#v", summary.StatusCodes)
	}
}

func assertRequestPerformanceBuckets(t *testing.T, buckets []moduleapi.RequestPerformanceMinuteBucket) {
	t.Helper()
	if len(buckets) != 3 {
		t.Fatalf("expected three buckets, got %#v", buckets)
	}
	assertRequestPerformanceBucket(t, buckets[0], moduleapi.RequestPerformanceMinuteBucket{TotalRequests: 3, ServerErrorCount: 1, P95LatencyMS: 1200, P99LatencyMS: 1200, RequestBytes: 107008, ResponseBytes: 132124})
	assertRequestPerformanceBucket(t, buckets[1], moduleapi.RequestPerformanceMinuteBucket{TotalRequests: 3, ServerErrorCount: 1, P95LatencyMS: 1500, P99LatencyMS: 1500, RequestBytes: 11264, ResponseBytes: 1150976})
	if buckets[2].TotalRequests != 0 || buckets[2].P95LatencyMS != 0 {
		t.Fatalf("expected zero-filled final bucket, got %#v", buckets[2])
	}
}

func assertRequestPerformanceBucket(t *testing.T, bucket, expected moduleapi.RequestPerformanceMinuteBucket) {
	t.Helper()
	if bucket.TotalRequests != expected.TotalRequests || bucket.ServerErrorCount != expected.ServerErrorCount || bucket.P95LatencyMS != expected.P95LatencyMS || bucket.P99LatencyMS != expected.P99LatencyMS || bucket.RequestBytes != expected.RequestBytes || bucket.ResponseBytes != expected.ResponseBytes {
		t.Fatalf("unexpected bucket: %#v", bucket)
	}
}

func assertRequestPerformanceDistributions(t *testing.T, summary moduleapi.RequestPerformanceSummary) {
	t.Helper()
	if got := requestPerformanceHistogramCounts(summary.LatencyHistogram); !reflect.DeepEqual(got, []int64{0, 0, 0, 1, 1, 4}) {
		t.Fatalf("unexpected latency histogram: %#v", summary.LatencyHistogram)
	}
	if got := requestPerformanceHistogramCounts(summary.RequestSizeHistogram); !reflect.DeepEqual(got, []int64{1, 2, 1, 1, 0, 0}) {
		t.Fatalf("unexpected request size histogram: %#v", summary.RequestSizeHistogram)
	}
	if got := requestPerformanceHistogramCounts(summary.ResponseSizeHistogram); !reflect.DeepEqual(got, []int64{1, 1, 0, 2, 1, 0}) {
		t.Fatalf("unexpected response size histogram: %#v", summary.ResponseSizeHistogram)
	}
}

func assertRequestPerformanceInstances(t *testing.T, summary moduleapi.RequestPerformanceSummary) {
	t.Helper()
	if len(summary.SlowestRequests) != 5 || summary.SlowestRequests[0].RequestID != "in-window-4" || summary.SlowestRequests[1].RequestID != "in-window-2" {
		t.Fatalf("unexpected slowest requests: %#v", summary.SlowestRequests)
	}
	if len(summary.LargestRequests) != 5 || summary.LargestRequests[0].RequestID != "in-window-3" || summary.LargestRequests[4].RequestID != "in-window-1" {
		t.Fatalf("unexpected largest requests: %#v", summary.LargestRequests)
	}
	if len(summary.LargestResponses) != 5 || summary.LargestResponses[0].RequestID != "in-window-4" || summary.LargestResponses[4].RequestID != "in-window-1" {
		t.Fatalf("unexpected largest responses: %#v", summary.LargestResponses)
	}
}

func requestPerformanceHistogramCounts(buckets []moduleapi.RequestPerformanceHistogramBucket) []int64 {
	counts := make([]int64, len(buckets))
	for index := range buckets {
		counts[index] = buckets[index].Count
	}
	return counts
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
	collector.add(requestPerformanceRow{requestID: "unmatched", occurredAt: base.Add(time.Second), method: "GET", path: "/missing", route: requestPerformanceUnmatchedRoute, statusCode: 200, durationMS: 100}, query.BucketSize)
	for index := 0; index < requestPerformanceLatencySampleLimit+10; index++ {
		collector.add(requestPerformanceRow{requestID: fmt.Sprintf("sample-%04d", index), occurredAt: base.Add(2 * time.Second), method: "GET", path: "/api/users", route: "/api/users", statusCode: 200, durationMS: int64(index)}, query.BucketSize)
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

func TestRequestPerformanceTopInstancesUseDeterministicTieOrder(t *testing.T) {
	base := time.Date(2026, 7, 14, 8, 0, 0, 0, time.UTC)
	instances := []moduleapi.RequestPerformanceRequestInstance{
		{RequestID: "z", OccurredAt: base, DurationMS: 100},
		{RequestID: "b", OccurredAt: base.Add(time.Second), DurationMS: 100},
		{RequestID: "a", OccurredAt: base.Add(time.Second), DurationMS: 100},
		{RequestID: "high", OccurredAt: base.Add(-time.Second), DurationMS: 200},
		{RequestID: "low", OccurredAt: base.Add(2 * time.Second), DurationMS: 50},
		{RequestID: "trimmed", OccurredAt: base.Add(3 * time.Second), DurationMS: 25},
	}

	var ranked []moduleapi.RequestPerformanceRequestInstance
	for _, instance := range instances {
		ranked = requestPerformanceInsertTopInstance(ranked, instance, requestPerformanceInstanceMetricDuration)
	}
	got := make([]string, len(ranked))
	for index := range ranked {
		got[index] = ranked[index].RequestID
	}
	if expected := []string{"high", "a", "b", "z", "low"}; !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected deterministic instance order: got %v want %v", got, expected)
	}
}

func seedRequestPerformanceAccessLogs(t *testing.T, repository AccessLogRepository, base time.Time) {
	t.Helper()
	inputs := []CreateAccessLogInput{
		{RequestID: "in-window-1", Method: "GET", Path: "/api/users?page=1", Route: "/api/users", StatusCode: 200, DurationMS: 100, RequestSize: requestPerformanceInt64Pointer(512), ResponseSize: requestPerformanceInt64Pointer(512), OccurredAt: base.Add(10 * time.Second)},
		{RequestID: "in-window-2", Method: "GET", Path: "/api/users?page=2", Route: "/api/users", StatusCode: 500, DurationMS: 1200, RequestSize: requestPerformanceInt64Pointer(4096), ResponseSize: requestPerformanceInt64Pointer(4096), OccurredAt: base.Add(20 * time.Second)},
		{RequestID: "in-window-3", Method: "POST", Path: "/api/users", Route: "/api/users", StatusCode: 201, DurationMS: 200, RequestSize: requestPerformanceInt64Pointer(100 * 1024), ResponseSize: requestPerformanceInt64Pointer(127516), OccurredAt: base.Add(30 * time.Second)},
		{RequestID: "in-window-4", Method: "GET", Path: "/api/reports", Route: "/api/reports", StatusCode: 502, DurationMS: 1500, RequestSize: requestPerformanceInt64Pointer(10 * 1024), ResponseSize: requestPerformanceInt64Pointer(1024 * 1024), OccurredAt: base.Add(time.Minute + 5*time.Second)},
		{RequestID: "in-window-5", Method: "GET", Path: "/api/reports", Route: "/api/reports", StatusCode: 404, DurationMS: 20, RequestSize: requestPerformanceInt64Pointer(1024), ResponseSize: requestPerformanceInt64Pointer(102400), OccurredAt: base.Add(time.Minute + 10*time.Second)},
		{RequestID: "in-window-6", Method: "GET", Path: "/healthz", Route: "/healthz", StatusCode: 302, DurationMS: 50, OccurredAt: base.Add(time.Minute + 15*time.Second)},
		{RequestID: "websocket", Method: "GET", Path: "/ws", Route: "/ws", ConnectionType: AccessLogConnectionTypeWebSocket, StatusCode: 101, DurationMS: 750000, OccurredAt: base.Add(time.Minute + 20*time.Second)},
		{RequestID: "outside-window", Method: "GET", Path: "/api/ignored", Route: "/api/ignored", StatusCode: 500, DurationMS: 9999, OccurredAt: base.Add(3 * time.Minute)},
	}
	if _, err := repository.CreateAccessLogs(context.Background(), inputs); err != nil {
		t.Fatalf("seed request performance access logs: %v", err)
	}
}

func requestPerformanceInt64Pointer(value int64) *int64 {
	return &value
}
