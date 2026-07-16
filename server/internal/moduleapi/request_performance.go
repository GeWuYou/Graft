package moduleapi

import (
	"context"
	"errors"
	"time"
)

// RequestPerformanceMinuteBucketSize 是有界请求性能读取唯一支持的时间桶宽度。
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

// RequestPerformanceReader 暴露有界的只读请求性能投影。
type RequestPerformanceReader interface {
	ReadRequestPerformance(context.Context, RequestPerformanceQuery) (RequestPerformanceSummary, error)
}

// RequestPerformanceSummary 包含一个有界时间窗口内的请求质量聚合结果。
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

// RequestPerformanceStatusGroups 按 HTTP 状态码类别聚合响应计数。
type RequestPerformanceStatusGroups struct {
	TwoXX   int64
	ThreeXX int64
	FourXX  int64
	FiveXX  int64
}

// RequestPerformanceMinuteBucket 描述一个经过零填充的请求活动分钟桶。
type RequestPerformanceMinuteBucket struct {
	Start            time.Time
	TotalRequests    int64
	ServerErrorCount int64
	P95LatencyMS     int64
}

// RequestPerformanceTopRoutes 聚合三种相互独立的路由排名。
type RequestPerformanceTopRoutes struct {
	ByTraffic      []RequestPerformanceRoute
	ByServerErrors []RequestPerformanceRoute
	ByP95Latency   []RequestPerformanceRoute
}

// RequestPerformanceRoute 是按路由聚合的请求质量结果。
type RequestPerformanceRoute struct {
	Method           string
	Route            string
	TotalRequests    int64
	ServerErrorCount int64
	P95LatencyMS     int64
}
