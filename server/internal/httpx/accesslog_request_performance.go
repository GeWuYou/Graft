package httpx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"graft/server/internal/moduleapi"
)

const (
	requestPerformanceSlowThresholdMS = int64(1000)
	requestPerformanceTopRouteLimit   = 5
	requestPerformanceP50Percentile   = 0.50
	requestPerformanceP95Percentile   = 0.95
	requestPerformanceWindowStartArg  = 1
	requestPerformanceWindowEndArg    = 2
)

type requestPerformanceRouteAggregate struct {
	method           string
	route            string
	totalRequests    int64
	serverErrorCount int64
	latencies        []int64
}

type requestPerformanceRouteKey struct {
	method string
	route  string
}

type requestPerformanceBucketAggregate struct {
	totalRequests    int64
	serverErrorCount int64
	latencies        []int64
}

type requestPerformanceCollector struct {
	summary   moduleapi.RequestPerformanceSummary
	buckets   map[time.Time]*requestPerformanceBucketAggregate
	routes    map[requestPerformanceRouteKey]*requestPerformanceRouteAggregate
	latencies []int64
}

// ReadRequestPerformance reads a bounded aggregate from canonical access-log facts.
func (r *accessLogRepository) ReadRequestPerformance(
	ctx context.Context,
	query moduleapi.RequestPerformanceQuery,
) (moduleapi.RequestPerformanceSummary, error) {
	if r == nil || r.db == nil {
		return moduleapi.RequestPerformanceSummary{}, errors.New("access log repository is unavailable")
	}
	if err := validateRequestPerformanceQuery(query); err != nil {
		return moduleapi.RequestPerformanceSummary{}, err
	}

	windowStart := query.WindowStart.UTC()
	windowEnd := query.WindowEnd.UTC()
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT occurred_at, method, route, status_code, duration_ms
		FROM access_logs
		WHERE occurred_at >= %s AND occurred_at < %s`, r.placeholder(requestPerformanceWindowStartArg), r.placeholder(requestPerformanceWindowEndArg)), windowStart, windowEnd)
	if err != nil {
		return moduleapi.RequestPerformanceSummary{}, fmt.Errorf("query request performance access logs: %w", err)
	}

	summary, readErr := collectRequestPerformanceRows(rows, query)
	closeErr := rows.Close()
	if readErr != nil {
		return moduleapi.RequestPerformanceSummary{}, readErr
	}
	if closeErr != nil {
		return moduleapi.RequestPerformanceSummary{}, fmt.Errorf("close request performance access log rows: %w", closeErr)
	}
	return summary, nil
}

func collectRequestPerformanceRows(rows *sql.Rows, query moduleapi.RequestPerformanceQuery) (moduleapi.RequestPerformanceSummary, error) {
	collector := newRequestPerformanceCollector(query)
	for rows.Next() {
		var occurredAt time.Time
		var method string
		var route sql.NullString
		var statusCode int
		var durationMS int64
		if err := rows.Scan(&occurredAt, &method, &route, &statusCode, &durationMS); err != nil {
			return moduleapi.RequestPerformanceSummary{}, fmt.Errorf("scan request performance access log: %w", err)
		}
		collector.add(occurredAt, method, route.String, statusCode, durationMS, query.BucketSize)
	}
	if err := rows.Err(); err != nil {
		return moduleapi.RequestPerformanceSummary{}, fmt.Errorf("iterate request performance access logs: %w", err)
	}
	return collector.summaryWithRankings(), nil
}

func newRequestPerformanceCollector(query moduleapi.RequestPerformanceQuery) *requestPerformanceCollector {
	summary := newRequestPerformanceSummary(query.WindowStart.UTC(), query.WindowEnd.UTC(), query.BucketSize)
	buckets := make(map[time.Time]*requestPerformanceBucketAggregate, len(summary.Buckets))
	for index := range summary.Buckets {
		buckets[summary.Buckets[index].Start] = &requestPerformanceBucketAggregate{}
	}
	return &requestPerformanceCollector{
		summary: summary,
		buckets: buckets,
		routes:  make(map[requestPerformanceRouteKey]*requestPerformanceRouteAggregate),
	}
}

func (c *requestPerformanceCollector) add(occurredAt time.Time, method, route string, statusCode int, durationMS int64, bucketSize time.Duration) {
	bucket := c.buckets[occurredAt.UTC().Truncate(bucketSize)]
	if bucket == nil {
		return
	}
	routeAggregate := c.route(method, route)
	c.summary.TotalRequests++
	bucket.totalRequests++
	routeAggregate.totalRequests++
	incrementRequestPerformanceStatusGroup(&c.summary.StatusGroups, statusCode)
	if isRequestPerformanceServerError(statusCode) {
		c.summary.ServerErrorCount++
		bucket.serverErrorCount++
		routeAggregate.serverErrorCount++
	}
	if durationMS >= requestPerformanceSlowThresholdMS {
		c.summary.SlowRequestCount++
	}
	c.latencies = append(c.latencies, durationMS)
	bucket.latencies = append(bucket.latencies, durationMS)
	routeAggregate.latencies = append(routeAggregate.latencies, durationMS)
}

func (c *requestPerformanceCollector) route(method, route string) *requestPerformanceRouteAggregate {
	key := requestPerformanceRouteKey{method: method, route: route}
	if aggregate := c.routes[key]; aggregate != nil {
		return aggregate
	}
	aggregate := &requestPerformanceRouteAggregate{method: method, route: route}
	c.routes[key] = aggregate
	return aggregate
}

func (c *requestPerformanceCollector) summaryWithRankings() moduleapi.RequestPerformanceSummary {
	c.summary.P50LatencyMS = requestPerformancePercentile(c.latencies, requestPerformanceP50Percentile)
	c.summary.P95LatencyMS = requestPerformancePercentile(c.latencies, requestPerformanceP95Percentile)
	for index := range c.summary.Buckets {
		bucket := c.buckets[c.summary.Buckets[index].Start]
		c.summary.Buckets[index].TotalRequests = bucket.totalRequests
		c.summary.Buckets[index].ServerErrorCount = bucket.serverErrorCount
		c.summary.Buckets[index].P95LatencyMS = requestPerformancePercentile(bucket.latencies, requestPerformanceP95Percentile)
	}
	c.summary.TopRoutes = requestPerformanceTopRoutes(c.routes)
	return c.summary
}

func validateRequestPerformanceQuery(query moduleapi.RequestPerformanceQuery) error {
	if query.WindowStart.IsZero() || query.WindowEnd.IsZero() || !query.WindowStart.Before(query.WindowEnd) || query.BucketSize != moduleapi.RequestPerformanceMinuteBucketSize {
		return moduleapi.ErrRequestPerformanceInvalidQuery
	}
	return nil
}

func newRequestPerformanceSummary(start, end time.Time, bucketSize time.Duration) moduleapi.RequestPerformanceSummary {
	firstBucket := start.Truncate(bucketSize)
	bucketCount := int(end.Sub(firstBucket) / bucketSize)
	if end.After(firstBucket.Add(time.Duration(bucketCount) * bucketSize)) {
		bucketCount++
	}
	buckets := make([]moduleapi.RequestPerformanceMinuteBucket, 0, bucketCount)
	for bucketStart := firstBucket; bucketStart.Before(end); bucketStart = bucketStart.Add(bucketSize) {
		buckets = append(buckets, moduleapi.RequestPerformanceMinuteBucket{Start: bucketStart})
	}
	return moduleapi.RequestPerformanceSummary{WindowStart: start, WindowEnd: end, Buckets: buckets}
}

func incrementRequestPerformanceStatusGroup(groups *moduleapi.RequestPerformanceStatusGroups, statusCode int) {
	switch {
	case statusCode >= 200 && statusCode <= 299:
		groups.TwoXX++
	case statusCode >= 300 && statusCode <= 399:
		groups.ThreeXX++
	case statusCode >= accessLogStatus4xxMin && statusCode <= accessLogStatus4xxMax:
		groups.FourXX++
	case isRequestPerformanceServerError(statusCode):
		groups.FiveXX++
	}
}

func isRequestPerformanceServerError(statusCode int) bool {
	return statusCode >= accessLogStatus5xxMin && statusCode <= accessLogStatus5xxMax
}

func requestPerformancePercentile(values []int64, percentile float64) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(left, right int) bool { return sorted[left] < sorted[right] })
	index := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	return sorted[index]
}

func requestPerformanceTopRoutes(routes map[requestPerformanceRouteKey]*requestPerformanceRouteAggregate) moduleapi.RequestPerformanceTopRoutes {
	aggregates := make([]*requestPerformanceRouteAggregate, 0, len(routes))
	for _, aggregate := range routes {
		aggregates = append(aggregates, aggregate)
	}

	return moduleapi.RequestPerformanceTopRoutes{
		ByTraffic:      requestPerformanceRankRoutes(aggregates, compareRequestPerformanceTraffic),
		ByServerErrors: requestPerformanceRankRoutes(onlyRequestPerformanceErrorRoutes(aggregates), compareRequestPerformanceServerErrors),
		ByP95Latency:   requestPerformanceRankRoutes(aggregates, compareRequestPerformanceP95Latency),
	}
}

func onlyRequestPerformanceErrorRoutes(routes []*requestPerformanceRouteAggregate) []*requestPerformanceRouteAggregate {
	filtered := make([]*requestPerformanceRouteAggregate, 0, len(routes))
	for _, route := range routes {
		if route.serverErrorCount > 0 {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

func requestPerformanceRankRoutes(routes []*requestPerformanceRouteAggregate, compare func(*requestPerformanceRouteAggregate, *requestPerformanceRouteAggregate) bool) []moduleapi.RequestPerformanceRoute {
	sorted := append([]*requestPerformanceRouteAggregate(nil), routes...)
	sort.Slice(sorted, func(left, right int) bool { return compare(sorted[left], sorted[right]) })
	if len(sorted) > requestPerformanceTopRouteLimit {
		sorted = sorted[:requestPerformanceTopRouteLimit]
	}
	result := make([]moduleapi.RequestPerformanceRoute, 0, len(sorted))
	for _, route := range sorted {
		result = append(result, moduleapi.RequestPerformanceRoute{
			Method:           route.method,
			Route:            route.route,
			TotalRequests:    route.totalRequests,
			ServerErrorCount: route.serverErrorCount,
			P95LatencyMS:     requestPerformancePercentile(route.latencies, requestPerformanceP95Percentile),
		})
	}
	return result
}

func compareRequestPerformanceTraffic(left, right *requestPerformanceRouteAggregate) bool {
	if left.totalRequests != right.totalRequests {
		return left.totalRequests > right.totalRequests
	}
	return requestPerformanceRouteIdentity(left) < requestPerformanceRouteIdentity(right)
}

func compareRequestPerformanceServerErrors(left, right *requestPerformanceRouteAggregate) bool {
	if left.serverErrorCount != right.serverErrorCount {
		return left.serverErrorCount > right.serverErrorCount
	}
	if left.totalRequests != right.totalRequests {
		return left.totalRequests > right.totalRequests
	}
	return requestPerformanceRouteIdentity(left) < requestPerformanceRouteIdentity(right)
}

func compareRequestPerformanceP95Latency(left, right *requestPerformanceRouteAggregate) bool {
	leftP95 := requestPerformancePercentile(left.latencies, requestPerformanceP95Percentile)
	rightP95 := requestPerformancePercentile(right.latencies, requestPerformanceP95Percentile)
	if leftP95 != rightP95 {
		return leftP95 > rightP95
	}
	if left.totalRequests != right.totalRequests {
		return left.totalRequests > right.totalRequests
	}
	return requestPerformanceRouteIdentity(left) < requestPerformanceRouteIdentity(right)
}

func requestPerformanceRouteIdentity(route *requestPerformanceRouteAggregate) string {
	return route.method + "\x00" + route.route
}

var _ moduleapi.RequestPerformanceReader = (*accessLogRepository)(nil)
