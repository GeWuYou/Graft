package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	generated "graft/server/internal/contract/openapi/generated"
	monitoropenapi "graft/server/internal/contract/openapi/monitor"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	monitorcontract "graft/server/modules/monitor/contract"
)

func TestBuildRequestPerformanceResponseMapsReaderSnapshot(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.July, 14, 8, 30, 15, 0, time.UTC)
	reader := &requestPerformanceReaderStub{summary: moduleapi.RequestPerformanceSummary{
		WindowStart:      observedAt.Add(-10 * time.Minute),
		WindowEnd:        observedAt,
		TotalRequests:    120,
		ServerErrorCount: 6,
		SlowRequestCount: 9,
		P50LatencyMS:     20,
		P95LatencyMS:     150,
		P99LatencyMS:     300,
		MaxLatencyMS:     800,
		AverageLatencyMS: 42.5,
		RequestBytes:     moduleapi.RequestPerformanceByteSummary{MeasuredCount: 100, TotalBytes: 60000, AverageBytes: 600},
		ResponseBytes:    moduleapi.RequestPerformanceByteSummary{MeasuredCount: 110, TotalBytes: 120000, AverageBytes: 120000.0 / 110},
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
			P99LatencyMS:     250,
			RequestBytes:     6000,
			ResponseBytes:    12000,
		}},
		StatusCodes:           []moduleapi.RequestPerformanceStatusCodeCount{{StatusCode: 200, Count: 100}, {StatusCode: 500, Count: 6}},
		LatencyHistogram:      []moduleapi.RequestPerformanceHistogramBucket{{LowerBound: 0, UpperBound: int64Pointer(5), Count: 12}, {LowerBound: 100, Count: 3}},
		RequestSizeHistogram:  []moduleapi.RequestPerformanceHistogramBucket{{LowerBound: 0, UpperBound: int64Pointer(1024), Count: 80}},
		ResponseSizeHistogram: []moduleapi.RequestPerformanceHistogramBucket{{LowerBound: 1024, Count: 50}},
		SlowestRequests:       []moduleapi.RequestPerformanceRequestInstance{{RequestID: "slow", OccurredAt: observedAt.Add(-time.Second), Method: "GET", Path: "/slow?x=1", Route: "/slow", StatusCode: 200, DurationMS: 800}},
		LargestRequests:       []moduleapi.RequestPerformanceRequestInstance{{RequestID: "large-request", OccurredAt: observedAt.Add(-2 * time.Second), Method: "POST", Path: "/upload", Route: "/upload", StatusCode: 201, DurationMS: 50, RequestSize: int64Pointer(4096)}},
		LargestResponses:      []moduleapi.RequestPerformanceRequestInstance{{RequestID: "large-response", OccurredAt: observedAt.Add(-3 * time.Second), Method: "GET", Path: "/download", Route: "/download", StatusCode: 200, DurationMS: 60, ResponseSize: int64Pointer(8192)}},
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

	response, err := buildRequestPerformanceResponse(context.Background(), reader, activeRequestReaderStub(3), monitorcontract.TrendRange10Minutes, observedAt)
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
	assertRequestPerformanceWindow(t, response, observedAt)
	assertRequestPerformanceSummary(t, response.Summary)
	assertRequestPerformanceSeries(t, response)
	assertRequestPerformanceBreakdowns(t, response)
	assertRequestPerformanceInstances(t, response)
}

func assertRequestPerformanceWindow(t *testing.T, response generated.RequestPerformanceResponse, observedAt time.Time) {
	t.Helper()
	if response.Range != "10m" || response.ObservedAt != observedAt || response.WindowStart != observedAt.Add(-10*time.Minute) || response.WindowEnd != observedAt {
		t.Fatalf("unexpected response window: %#v", response)
	}
}

func assertRequestPerformanceSummary(t *testing.T, summary generated.RequestPerformanceSummary) {
	t.Helper()
	if summary.TotalRequests != 120 || summary.RequestsPerSecond != 0.2 || summary.Error5xxRate != 5 || summary.P95LatencyMs != 150 || summary.P99LatencyMs != 300 || summary.MaxLatencyMs != 800 || summary.AverageLatencyMs != 42.5 || summary.ActiveRequests != 3 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if summary.RequestBytes.AverageBytes == nil || *summary.RequestBytes.AverageBytes != 600 || summary.RequestBytes.BytesPerSecond == nil || *summary.RequestBytes.BytesPerSecond != 100 {
		t.Fatalf("unexpected byte summary: %#v", summary.RequestBytes)
	}
}

func assertRequestPerformanceSeries(t *testing.T, response generated.RequestPerformanceResponse) {
	t.Helper()
	if len(response.MinuteBuckets) != 1 || response.MinuteBuckets[0].RequestsPerSecond != 0.2 || response.MinuteBuckets[0].Error5xxRate != 25 || response.MinuteBuckets[0].P99LatencyMs != 250 || response.MinuteBuckets[0].RequestBytesPerSecond != 100 || response.MinuteBuckets[0].ResponseBytesPerSecond != 200 {
		t.Fatalf("unexpected minute buckets: %#v", response.MinuteBuckets)
	}
	if len(response.StatusGroups) != 4 || response.StatusGroups[0].RequestRate != float64(100)*100/120 || response.StatusGroups[3].RequestRate != 5 {
		t.Fatalf("unexpected status groups: %#v", response.StatusGroups)
	}
}

func assertRequestPerformanceBreakdowns(t *testing.T, response generated.RequestPerformanceResponse) {
	t.Helper()
	if len(response.TopRoutes.Traffic) != 1 || response.TopRoutes.Traffic[0].Error5xxRate != float64(2)*100/30 {
		t.Fatalf("unexpected top routes: %#v", response.TopRoutes)
	}
	if len(response.StatusCodes) != 2 || response.StatusCodes[1].StatusCode != 500 || response.StatusCodes[1].RequestRate != 5 {
		t.Fatalf("unexpected status codes: %#v", response.StatusCodes)
	}
	if len(response.LatencyDistribution) != 2 || response.LatencyDistribution[0].SampleRate != 10 || response.LatencyDistribution[1].UpperBound != nil {
		t.Fatalf("unexpected latency distribution: %#v", response.LatencyDistribution)
	}
}

func assertRequestPerformanceInstances(t *testing.T, response generated.RequestPerformanceResponse) {
	t.Helper()
	if len(response.SlowestRequests) != 1 || response.SlowestRequests[0].RequestId != "slow" || len(response.LargestRequests) != 1 || response.LargestRequests[0].RequestSizeBytes == nil || len(response.LargestResponses) != 1 || response.LargestResponses[0].ResponseSizeBytes == nil {
		t.Fatalf("unexpected request instances: %#v %#v %#v", response.SlowestRequests, response.LargestRequests, response.LargestResponses)
	}
}

func TestBuildRequestPerformanceResponseKeepsUnavailableBytesNullableAndArraysEmpty(t *testing.T) {
	observedAt := time.Date(2026, time.July, 14, 8, 30, 15, 0, time.UTC)
	reader := &requestPerformanceReaderStub{summary: moduleapi.RequestPerformanceSummary{WindowStart: observedAt.Add(-10 * time.Minute), WindowEnd: observedAt}}
	response, err := buildRequestPerformanceResponse(context.Background(), reader, activeRequestReaderStub(0), monitorcontract.TrendRange10Minutes, observedAt)
	if err != nil {
		t.Fatalf("build empty request performance response: %v", err)
	}
	if response.Summary.RequestBytes.AverageBytes != nil || response.Summary.RequestBytes.BytesPerSecond != nil || response.Summary.ResponseBytes.AverageBytes != nil || response.Summary.ResponseBytes.BytesPerSecond != nil {
		t.Fatalf("expected unavailable byte metrics to remain nullable: %#v", response.Summary)
	}
	if response.StatusCodes == nil || response.LatencyDistribution == nil || response.RequestSizeDistribution == nil || response.ResponseSizeDistribution == nil || response.SlowestRequests == nil || response.LargestRequests == nil || response.LargestResponses == nil {
		t.Fatalf("expected empty arrays instead of null: %#v", response)
	}
}

func TestBuildRequestPerformanceResponseExcludesHandlerRequestFromActiveCount(t *testing.T) {
	server := httpx.NewServer(zap.NewNop())
	observedAt := time.Date(2026, time.July, 14, 8, 30, 15, 0, time.UTC)
	reader := &requestPerformanceReaderStub{summary: moduleapi.RequestPerformanceSummary{WindowStart: observedAt.Add(-10 * time.Minute), WindowEnd: observedAt}}
	activeCount := make(chan int64, 1)
	server.Engine().GET("/request-performance", func(ctx *gin.Context) {
		response, err := buildRequestPerformanceResponse(ctx.Request.Context(), reader, server.ActiveRequestReader(), monitorcontract.TrendRange10Minutes, observedAt)
		if err != nil {
			t.Errorf("build handler request performance response: %v", err)
			activeCount <- -1
			return
		}
		activeCount <- response.Summary.ActiveRequests
		ctx.Status(http.StatusNoContent)
	})
	server.Engine().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/request-performance", nil))
	if got := <-activeCount; got != 0 {
		t.Fatalf("expected handler request to exclude itself, got %d", got)
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

type activeRequestReaderStub int64

func (s activeRequestReaderStub) ReadActiveRequests(context.Context) int64 {
	return int64(s)
}

func int64Pointer(value int64) *int64 {
	return &value
}

func (s *requestPerformanceReaderStub) ReadRequestPerformance(_ context.Context, query moduleapi.RequestPerformanceQuery) (moduleapi.RequestPerformanceSummary, error) {
	s.query = query
	return s.summary, s.err
}
