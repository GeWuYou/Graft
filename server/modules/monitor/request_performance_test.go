package monitor

import (
	"context"
	"testing"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	monitoropenapi "graft/server/internal/contract/openapi/monitor"
	"graft/server/internal/moduleapi"
	monitorcontract "graft/server/modules/monitor/contract"
)

func TestBuildRequestPerformanceResponseMapsReaderSnapshot(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.July, 14, 8, 30, 15, 0, time.UTC)
	reader := &requestPerformanceReaderStub{summary: moduleapi.RequestPerformanceSummary{
		TotalRequests:    120,
		ServerErrorCount: 6,
		SlowRequestCount: 9,
		P50LatencyMS:     20,
		P95LatencyMS:     150,
		StatusGroups: moduleapi.RequestPerformanceStatusGroups{
			TwoXX:  100,
			FourXX: 14,
			FiveXX: 6,
		},
		Buckets: []moduleapi.RequestPerformanceMinuteBucket{{
			Start:            observedAt.Add(-time.Minute).Truncate(time.Minute),
			TotalRequests:    12,
			ServerErrorCount: 3,
			P95LatencyMS:     200,
		}},
		TopRoutes: moduleapi.RequestPerformanceTopRoutes{
			ByTraffic: []moduleapi.RequestPerformanceRoute{{
				Method:           "GET",
				Route:            "/api/healthz",
				TotalRequests:    30,
				ServerErrorCount: 2,
				P95LatencyMS:     90,
			}},
		},
	}}

	response, err := buildRequestPerformanceResponse(context.Background(), reader, monitorcontract.TrendRange10Minutes, observedAt)
	if err != nil {
		t.Fatalf("build request performance response: %v", err)
	}
	assertRequestPerformanceReaderQuery(t, reader.query, observedAt)
	assertRequestPerformanceResponse(t, response, observedAt)
}

func assertRequestPerformanceReaderQuery(t *testing.T, query moduleapi.RequestPerformanceQuery, observedAt time.Time) {
	t.Helper()
	if query.BucketSize != moduleapi.RequestPerformanceMinuteBucketSize || query.WindowEnd != observedAt || query.WindowStart != observedAt.Add(-10*time.Minute) {
		t.Fatalf("unexpected reader query: %#v", query)
	}
}

func assertRequestPerformanceResponse(t *testing.T, response generated.RequestPerformanceResponse, observedAt time.Time) {
	t.Helper()
	if response.Range != "10m" || response.ObservedAt != observedAt {
		t.Fatalf("unexpected response window: %#v", response)
	}
	if response.Summary.TotalRequests != 120 || response.Summary.RequestsPerSecond != 0.2 || response.Summary.Error5xxRate != 5 || response.Summary.P95LatencyMs != 150 {
		t.Fatalf("unexpected summary: %#v", response.Summary)
	}
	if len(response.MinuteBuckets) != 1 || response.MinuteBuckets[0].RequestsPerSecond != 0.2 || response.MinuteBuckets[0].Error5xxRate != 25 {
		t.Fatalf("unexpected minute buckets: %#v", response.MinuteBuckets)
	}
	if len(response.StatusGroups) != 4 || response.StatusGroups[0].RequestRate != float64(100)*100/120 || response.StatusGroups[3].RequestRate != 5 {
		t.Fatalf("unexpected status groups: %#v", response.StatusGroups)
	}
	if len(response.TopRoutes.Traffic) != 1 || response.TopRoutes.Traffic[0].Error5xxRate != float64(2)*100/30 {
		t.Fatalf("unexpected top routes: %#v", response.TopRoutes)
	}
}

func TestGetMonitorRequestPerformanceRejectsUnknownRange(t *testing.T) {
	t.Parallel()

	invalidRange := monitoropenapi.GetMonitorRequestPerformanceParamsRange("2h")
	handler := &monitorServerHandler{}
	if err := handler.GetMonitorRequestPerformance(context.Background(), monitoropenapi.GetMonitorRequestPerformanceParams{Range: &invalidRange}); err == nil {
		t.Fatal("expected unknown range to be rejected")
	}
}

type requestPerformanceReaderStub struct {
	query   moduleapi.RequestPerformanceQuery
	summary moduleapi.RequestPerformanceSummary
	err     error
}

func (s *requestPerformanceReaderStub) ReadRequestPerformance(_ context.Context, query moduleapi.RequestPerformanceQuery) (moduleapi.RequestPerformanceSummary, error) {
	s.query = query
	return s.summary, s.err
}
