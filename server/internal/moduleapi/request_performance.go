package moduleapi

import (
	"context"
	"errors"
	"time"
)

// RequestPerformanceMinuteBucketSize is the only supported bucket width for
// bounded request-performance reads.
const RequestPerformanceMinuteBucketSize = time.Minute

// ErrRequestPerformanceInvalidQuery reports an unsupported request-performance
// time window or bucket width.
var ErrRequestPerformanceInvalidQuery = errors.New("request performance query is invalid")

// RequestPerformanceQuery bounds a read of persisted HTTP request facts.
type RequestPerformanceQuery struct {
	WindowStart time.Time
	WindowEnd   time.Time
	BucketSize  time.Duration
}

// RequestPerformanceReader exposes a bounded, read-only performance projection.
type RequestPerformanceReader interface {
	ReadRequestPerformance(context.Context, RequestPerformanceQuery) (RequestPerformanceSummary, error)
}

// RequestPerformanceSummary contains request quality aggregates for one bounded window.
type RequestPerformanceSummary struct {
	WindowStart      time.Time
	WindowEnd        time.Time
	TotalRequests    int64
	ServerErrorCount int64
	SlowRequestCount int64
	P50LatencyMS     int64
	P95LatencyMS     int64
	StatusGroups     RequestPerformanceStatusGroups
	Buckets          []RequestPerformanceMinuteBucket
	TopRoutes        RequestPerformanceTopRoutes
}

// RequestPerformanceStatusGroups groups response counts by HTTP status class.
type RequestPerformanceStatusGroups struct {
	TwoXX   int64
	ThreeXX int64
	FourXX  int64
	FiveXX  int64
}

// RequestPerformanceMinuteBucket describes one zero-filled minute of request activity.
type RequestPerformanceMinuteBucket struct {
	Start            time.Time
	TotalRequests    int64
	ServerErrorCount int64
	P95LatencyMS     int64
}

// RequestPerformanceTopRoutes groups the three independent route rankings.
type RequestPerformanceTopRoutes struct {
	ByTraffic      []RequestPerformanceRoute
	ByServerErrors []RequestPerformanceRoute
	ByP95Latency   []RequestPerformanceRoute
}

// RequestPerformanceRoute is a route-scoped request quality aggregate.
type RequestPerformanceRoute struct {
	Method           string
	Route            string
	TotalRequests    int64
	ServerErrorCount int64
	P95LatencyMS     int64
}
