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

// ActiveRequestReader 读取当前进程内仍在处理的普通 HTTP 请求数。
// 实现应识别调用上下文所属请求，使指标读取方能够排除自身而无需依赖具体路由。
type ActiveRequestReader interface {
	ReadActiveRequests(context.Context) int64
}

// RequestPerformanceSummary 包含一个有界时间窗口内的请求质量聚合结果。
type RequestPerformanceSummary struct {
	WindowStart           time.Time
	WindowEnd             time.Time
	TotalRequests         int64
	ServerErrorCount      int64
	SlowRequestCount      int64
	AverageLatencyMS      float64
	P50LatencyMS          int64
	P95LatencyMS          int64
	P99LatencyMS          int64
	MaxLatencyMS          int64
	RequestBytes          RequestPerformanceByteSummary
	ResponseBytes         RequestPerformanceByteSummary
	StatusGroups          RequestPerformanceStatusGroups
	StatusCodes           []RequestPerformanceStatusCodeCount
	LatencyHistogram      []RequestPerformanceHistogramBucket
	RequestSizeHistogram  []RequestPerformanceHistogramBucket
	ResponseSizeHistogram []RequestPerformanceHistogramBucket
	Buckets               []RequestPerformanceMinuteBucket
	TopRoutes             RequestPerformanceTopRoutes
	SlowestRequests       []RequestPerformanceRequestInstance
	LargestRequests       []RequestPerformanceRequestInstance
	LargestResponses      []RequestPerformanceRequestInstance
}

// RequestPerformanceByteSummary 描述已采集大小的请求样本及其字节聚合；MeasuredCount 不包含未知大小记录。
type RequestPerformanceByteSummary struct {
	MeasuredCount int64
	TotalBytes    int64
	AverageBytes  float64
}

// RequestPerformanceStatusGroups 按 HTTP 状态码类别聚合响应计数。
type RequestPerformanceStatusGroups struct {
	TwoXX   int64
	ThreeXX int64
	FourXX  int64
	FiveXX  int64
}

// RequestPerformanceStatusCodeCount 描述一个精确 HTTP 状态码的响应计数。
type RequestPerformanceStatusCodeCount struct {
	StatusCode int
	Count      int64
}

// RequestPerformanceHistogramBucket 描述左闭右开的数值区间；UpperBound 为空表示无上界。
type RequestPerformanceHistogramBucket struct {
	LowerBound int64
	UpperBound *int64
	Count      int64
}

// RequestPerformanceMinuteBucket 描述一个经过零填充的请求活动分钟桶。
type RequestPerformanceMinuteBucket struct {
	Start            time.Time
	TotalRequests    int64
	ServerErrorCount int64
	P95LatencyMS     int64
	P99LatencyMS     int64
	RequestBytes     int64
	ResponseBytes    int64
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

// RequestPerformanceRequestInstance 是用于请求性能榜单下钻的单次已完成 HTTP 请求事实。
type RequestPerformanceRequestInstance struct {
	RequestID    string
	OccurredAt   time.Time
	Method       string
	Path         string
	Route        string
	StatusCode   int
	DurationMS   int64
	RequestSize  *int64
	ResponseSize *int64
}
