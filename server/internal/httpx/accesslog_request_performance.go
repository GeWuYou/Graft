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
	requestPerformanceSlowThresholdMS    = int64(1000)
	requestPerformanceTopRouteLimit      = 5
	requestPerformanceP50Percentile      = 0.50
	requestPerformanceP95Percentile      = 0.95
	requestPerformanceP99Percentile      = 0.99
	requestPerformanceLatencySampleLimit = 1024
	requestPerformancePageSize           = 1024
	requestPerformanceTopInstanceLimit   = 5
	requestPerformanceWindowStartArg     = 1
	requestPerformanceWindowEndArg       = 2
	requestPerformanceConnectionTypeArg  = 3
	requestPerformanceCursorTimeArg      = 4
	requestPerformanceCursorIDArg        = 5
	requestPerformancePageSizeArg        = 6
	requestPerformanceUnmatchedRoute     = "<unmatched>"
)

type requestPerformanceRouteAggregate struct {
	method           string
	route            string
	totalRequests    int64
	serverErrorCount int64
	latencies        requestPerformanceLatencySummary
	p95LatencyMS     int64
}

type requestPerformanceRouteKey struct {
	method string
	route  string
}

type requestPerformanceBucketAggregate struct {
	totalRequests    int64
	serverErrorCount int64
	latencies        requestPerformanceLatencySummary
	requestBytes     int64
	responseBytes    int64
}

type requestPerformanceCollector struct {
	summary               moduleapi.RequestPerformanceSummary
	buckets               map[time.Time]*requestPerformanceBucketAggregate
	routes                map[requestPerformanceRouteKey]*requestPerformanceRouteAggregate
	latencies             requestPerformanceLatencySummary
	totalLatencyMS        int64
	statusCodes           map[int]int64
	latencyHistogram      requestPerformanceHistogram
	requestSizeHistogram  requestPerformanceHistogram
	responseSizeHistogram requestPerformanceHistogram
	slowestRequests       []moduleapi.RequestPerformanceRequestInstance
	largestRequests       []moduleapi.RequestPerformanceRequestInstance
	largestResponses      []moduleapi.RequestPerformanceRequestInstance
}

type requestPerformanceRow struct {
	id           int64
	requestID    string
	occurredAt   time.Time
	method       string
	path         string
	route        string
	statusCode   int
	durationMS   int64
	requestSize  *int64
	responseSize *int64
}

type requestPerformanceHistogram struct {
	bounds []int64
	counts []int64
}

// ReadRequestPerformance 从规范访问日志事实中读取有界的请求性能汇总。
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
	collector := newRequestPerformanceCollector(query)
	cursorOccurredAt := windowStart
	var cursorID int64
	for {
		rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT id, request_id, occurred_at, method, path, route, status_code, duration_ms, request_size, response_size
		FROM access_logs
		WHERE occurred_at >= %s AND occurred_at < %s AND connection_type = %s
			AND (occurred_at > %s OR (occurred_at = %s AND id > %s))
		ORDER BY occurred_at ASC, id ASC
		LIMIT %s`, r.placeholder(requestPerformanceWindowStartArg), r.placeholder(requestPerformanceWindowEndArg), r.placeholder(requestPerformanceConnectionTypeArg), r.placeholder(requestPerformanceCursorTimeArg), r.placeholder(requestPerformanceCursorTimeArg), r.placeholder(requestPerformanceCursorIDArg), r.placeholder(requestPerformancePageSizeArg)), windowStart, windowEnd, AccessLogConnectionTypeHTTP, cursorOccurredAt, cursorOccurredAt, cursorID, requestPerformancePageSize)
		if err != nil {
			return moduleapi.RequestPerformanceSummary{}, fmt.Errorf("query request performance access logs: %w", err)
		}

		page, readErr := collectRequestPerformanceRows(rows, collector, query)
		closeErr := rows.Close()
		if readErr != nil {
			return moduleapi.RequestPerformanceSummary{}, readErr
		}
		if closeErr != nil {
			return moduleapi.RequestPerformanceSummary{}, fmt.Errorf("close request performance access log rows: %w", closeErr)
		}
		if page.count == 0 {
			break
		}
		cursorOccurredAt = page.lastOccurredAt
		cursorID = page.lastID
	}
	return collector.summaryWithRankings(), nil
}

// collectRequestPerformanceRows 聚合数据库行中的请求性能数据；扫描失败或迭代失败时返回错误。
func collectRequestPerformanceRows(rows *sql.Rows, collector *requestPerformanceCollector, query moduleapi.RequestPerformanceQuery) (requestPerformancePage, error) {
	page := requestPerformancePage{}
	for rows.Next() {
		var row requestPerformanceRow
		var route sql.NullString
		var requestSize sql.NullInt64
		var responseSize sql.NullInt64
		if err := rows.Scan(&row.id, &row.requestID, &row.occurredAt, &row.method, &row.path, &route, &row.statusCode, &row.durationMS, &requestSize, &responseSize); err != nil {
			return requestPerformancePage{}, fmt.Errorf("scan request performance access log: %w", err)
		}
		row.route = requestPerformanceRouteValue(route)
		row.requestSize = requestPerformanceOptionalInt64(requestSize)
		row.responseSize = requestPerformanceOptionalInt64(responseSize)
		collector.add(row, query.BucketSize)
		page.count++
		page.lastOccurredAt = row.occurredAt
		page.lastID = row.id
	}
	if err := rows.Err(); err != nil {
		return requestPerformancePage{}, fmt.Errorf("iterate request performance access logs: %w", err)
	}
	return page, nil
}

type requestPerformancePage struct {
	count          int
	lastOccurredAt time.Time
	lastID         int64
}

// newRequestPerformanceCollector 创建按查询时间窗口和桶大小初始化的性能聚合器。
func newRequestPerformanceCollector(query moduleapi.RequestPerformanceQuery) *requestPerformanceCollector {
	summary := newRequestPerformanceSummary(query.WindowStart.UTC(), query.WindowEnd.UTC(), query.BucketSize)
	buckets := make(map[time.Time]*requestPerformanceBucketAggregate, len(summary.Buckets))
	for index := range summary.Buckets {
		buckets[summary.Buckets[index].Start] = &requestPerformanceBucketAggregate{}
	}
	return &requestPerformanceCollector{
		summary:               summary,
		buckets:               buckets,
		routes:                make(map[requestPerformanceRouteKey]*requestPerformanceRouteAggregate),
		statusCodes:           make(map[int]int64),
		latencyHistogram:      newRequestPerformanceHistogram([]int64{0, 5, 10, 20, 50, 100}),
		requestSizeHistogram:  newRequestPerformanceHistogram([]int64{0, 1024, 10 * 1024, 100 * 1024, 1024 * 1024, 10 * 1024 * 1024}),
		responseSizeHistogram: newRequestPerformanceHistogram([]int64{0, 1024, 10 * 1024, 100 * 1024, 1024 * 1024, 10 * 1024 * 1024}),
	}
}

func (c *requestPerformanceCollector) add(row requestPerformanceRow, bucketSize time.Duration) {
	bucket := c.buckets[row.occurredAt.UTC().Truncate(bucketSize)]
	if bucket == nil {
		return
	}
	routeAggregate := c.route(row.method, row.route)
	c.summary.TotalRequests++
	bucket.totalRequests++
	routeAggregate.totalRequests++
	c.statusCodes[row.statusCode]++
	incrementRequestPerformanceStatusGroup(&c.summary.StatusGroups, row.statusCode)
	if isRequestPerformanceServerError(row.statusCode) {
		c.summary.ServerErrorCount++
		bucket.serverErrorCount++
		routeAggregate.serverErrorCount++
	}
	if row.durationMS >= requestPerformanceSlowThresholdMS {
		c.summary.SlowRequestCount++
	}
	c.totalLatencyMS += row.durationMS
	if row.durationMS > c.summary.MaxLatencyMS {
		c.summary.MaxLatencyMS = row.durationMS
	}
	c.latencies.add(row.durationMS)
	bucket.latencies.add(row.durationMS)
	routeAggregate.latencies.add(row.durationMS)
	c.latencyHistogram.add(row.durationMS)
	instance := requestPerformanceRequestInstance(row)
	c.slowestRequests = requestPerformanceInsertTopInstance(c.slowestRequests, instance, requestPerformanceInstanceMetricDuration)
	if row.requestSize != nil {
		c.summary.RequestBytes.MeasuredCount++
		c.summary.RequestBytes.TotalBytes += *row.requestSize
		bucket.requestBytes += *row.requestSize
		c.requestSizeHistogram.add(*row.requestSize)
		c.largestRequests = requestPerformanceInsertTopInstance(c.largestRequests, instance, requestPerformanceInstanceMetricRequestSize)
	}
	if row.responseSize != nil {
		c.summary.ResponseBytes.MeasuredCount++
		c.summary.ResponseBytes.TotalBytes += *row.responseSize
		bucket.responseBytes += *row.responseSize
		c.responseSizeHistogram.add(*row.responseSize)
		c.largestResponses = requestPerformanceInsertTopInstance(c.largestResponses, instance, requestPerformanceInstanceMetricResponseSize)
	}
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
	if c.summary.TotalRequests > 0 {
		c.summary.AverageLatencyMS = float64(c.totalLatencyMS) / float64(c.summary.TotalRequests)
	}
	if c.summary.RequestBytes.MeasuredCount > 0 {
		c.summary.RequestBytes.AverageBytes = float64(c.summary.RequestBytes.TotalBytes) / float64(c.summary.RequestBytes.MeasuredCount)
	}
	if c.summary.ResponseBytes.MeasuredCount > 0 {
		c.summary.ResponseBytes.AverageBytes = float64(c.summary.ResponseBytes.TotalBytes) / float64(c.summary.ResponseBytes.MeasuredCount)
	}
	c.summary.P50LatencyMS = c.latencies.percentile(requestPerformanceP50Percentile)
	c.summary.P95LatencyMS = c.latencies.percentile(requestPerformanceP95Percentile)
	c.summary.P99LatencyMS = c.latencies.percentile(requestPerformanceP99Percentile)
	c.summary.StatusCodes = requestPerformanceSortedStatusCodes(c.statusCodes)
	c.summary.LatencyHistogram = c.latencyHistogram.result()
	c.summary.RequestSizeHistogram = c.requestSizeHistogram.result()
	c.summary.ResponseSizeHistogram = c.responseSizeHistogram.result()
	c.summary.SlowestRequests = c.slowestRequests
	c.summary.LargestRequests = c.largestRequests
	c.summary.LargestResponses = c.largestResponses
	for index := range c.summary.Buckets {
		bucket := c.buckets[c.summary.Buckets[index].Start]
		c.summary.Buckets[index].TotalRequests = bucket.totalRequests
		c.summary.Buckets[index].ServerErrorCount = bucket.serverErrorCount
		c.summary.Buckets[index].P95LatencyMS = bucket.latencies.percentile(requestPerformanceP95Percentile)
		c.summary.Buckets[index].P99LatencyMS = bucket.latencies.percentile(requestPerformanceP99Percentile)
		c.summary.Buckets[index].RequestBytes = bucket.requestBytes
		c.summary.Buckets[index].ResponseBytes = bucket.responseBytes
	}
	for _, route := range c.routes {
		route.p95LatencyMS = route.latencies.percentile(requestPerformanceP95Percentile)
	}
	c.summary.TopRoutes = requestPerformanceTopRoutes(c.routes)
	return c.summary
}

func requestPerformanceOptionalInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

func newRequestPerformanceHistogram(bounds []int64) requestPerformanceHistogram {
	return requestPerformanceHistogram{bounds: bounds, counts: make([]int64, len(bounds))}
}

func (h *requestPerformanceHistogram) add(value int64) {
	index := sort.Search(len(h.bounds), func(index int) bool { return h.bounds[index] > value }) - 1
	if index >= 0 {
		h.counts[index]++
	}
}

func (h requestPerformanceHistogram) result() []moduleapi.RequestPerformanceHistogramBucket {
	result := make([]moduleapi.RequestPerformanceHistogramBucket, 0, len(h.bounds))
	for index, lowerBound := range h.bounds {
		bucket := moduleapi.RequestPerformanceHistogramBucket{LowerBound: lowerBound, Count: h.counts[index]}
		if index+1 < len(h.bounds) {
			upperBound := h.bounds[index+1]
			bucket.UpperBound = &upperBound
		}
		result = append(result, bucket)
	}
	return result
}

func requestPerformanceSortedStatusCodes(counts map[int]int64) []moduleapi.RequestPerformanceStatusCodeCount {
	statusCodes := make([]int, 0, len(counts))
	for statusCode := range counts {
		statusCodes = append(statusCodes, statusCode)
	}
	sort.Ints(statusCodes)
	result := make([]moduleapi.RequestPerformanceStatusCodeCount, 0, len(statusCodes))
	for _, statusCode := range statusCodes {
		result = append(result, moduleapi.RequestPerformanceStatusCodeCount{StatusCode: statusCode, Count: counts[statusCode]})
	}
	return result
}

func requestPerformanceRequestInstance(row requestPerformanceRow) moduleapi.RequestPerformanceRequestInstance {
	return moduleapi.RequestPerformanceRequestInstance{
		RequestID:    row.requestID,
		OccurredAt:   row.occurredAt.UTC(),
		Method:       row.method,
		Path:         row.path,
		Route:        row.route,
		StatusCode:   row.statusCode,
		DurationMS:   row.durationMS,
		RequestSize:  row.requestSize,
		ResponseSize: row.responseSize,
	}
}

type requestPerformanceInstanceMetric int

const (
	requestPerformanceInstanceMetricDuration requestPerformanceInstanceMetric = iota
	requestPerformanceInstanceMetricRequestSize
	requestPerformanceInstanceMetricResponseSize
)

// requestPerformanceInsertTopInstance 只保留有界榜单，并按指标、发生时间和请求 ID 提供确定性顺序。
func requestPerformanceInsertTopInstance(
	instances []moduleapi.RequestPerformanceRequestInstance,
	instance moduleapi.RequestPerformanceRequestInstance,
	metric requestPerformanceInstanceMetric,
) []moduleapi.RequestPerformanceRequestInstance {
	instances = append(instances, instance)
	sort.Slice(instances, func(left, right int) bool {
		leftMetric := requestPerformanceInstanceMetricValue(instances[left], metric)
		rightMetric := requestPerformanceInstanceMetricValue(instances[right], metric)
		if leftMetric != rightMetric {
			return leftMetric > rightMetric
		}
		if !instances[left].OccurredAt.Equal(instances[right].OccurredAt) {
			return instances[left].OccurredAt.After(instances[right].OccurredAt)
		}
		return instances[left].RequestID < instances[right].RequestID
	})
	if len(instances) > requestPerformanceTopInstanceLimit {
		instances = instances[:requestPerformanceTopInstanceLimit]
	}
	return instances
}

func requestPerformanceInstanceMetricValue(instance moduleapi.RequestPerformanceRequestInstance, metric requestPerformanceInstanceMetric) int64 {
	switch metric {
	case requestPerformanceInstanceMetricRequestSize:
		return *instance.RequestSize
	case requestPerformanceInstanceMetricResponseSize:
		return *instance.ResponseSize
	default:
		return instance.DurationMS
	}
}

// validateRequestPerformanceQuery 校验请求性能查询的时间窗口和桶大小；当前仅接受规范的分钟桶。
func validateRequestPerformanceQuery(query moduleapi.RequestPerformanceQuery) error {
	if query.WindowStart.IsZero() || query.WindowEnd.IsZero() || !query.WindowStart.Before(query.WindowEnd) || query.BucketSize != moduleapi.RequestPerformanceMinuteBucketSize {
		return moduleapi.ErrRequestPerformanceInvalidQuery
	}
	return nil
}

// newRequestPerformanceSummary 根据时间窗口和桶大小创建请求性能汇总，并生成窗口内的时间桶。
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

// incrementRequestPerformanceStatusGroup 按 statusCode 将请求计入对应的 HTTP 状态码分组。
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

// isRequestPerformanceServerError 判断状态码是否处于 5xx 服务端错误范围。
func isRequestPerformanceServerError(statusCode int) bool {
	return statusCode >= accessLogStatus5xxMin && statusCode <= accessLogStatus5xxMax
}

// requestPerformanceLatencySummary 保存有界且确定性的延迟样本；样本池未满时保留精确分布，超出后使用稳定替换以控制内存。
type requestPerformanceLatencySummary struct {
	values []int64
	seen   int64
}

func (s *requestPerformanceLatencySummary) add(value int64) {
	s.seen++
	if len(s.values) < requestPerformanceLatencySampleLimit {
		s.values = append(s.values, value)
		return
	}
	// 确定性替换既限制内存，也避免相同查询因随机采样产生不同的 dashboard 结果。
	index := (s.seen*requestPerformanceReservoirMultiplier + requestPerformanceReservoirIncrement) % requestPerformanceLatencySampleLimit
	s.values[index] = value
}

const (
	requestPerformanceReservoirMultiplier = int64(7919)
	requestPerformanceReservoirIncrement  = int64(104729)
)

func (s requestPerformanceLatencySummary) percentile(percentile float64) int64 {
	if len(s.values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), s.values...)
	sort.Slice(sorted, func(left, right int) bool { return sorted[left] < sorted[right] })
	index := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	return sorted[index]
}

// requestPerformanceTopRoutes 按流量、服务端错误数和 P95 延迟生成路由排名。
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

// onlyRequestPerformanceErrorRoutes 过滤出至少发生过一次服务端错误的路由。
func onlyRequestPerformanceErrorRoutes(routes []*requestPerformanceRouteAggregate) []*requestPerformanceRouteAggregate {
	filtered := make([]*requestPerformanceRouteAggregate, 0, len(routes))
	for _, route := range routes {
		if route.serverErrorCount > 0 {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

// requestPerformanceRankRoutes按指定比较规则对路由聚合结果排序，生成最多包含前五名路由的性能结果。
// 返回按排序结果排列的路由信息，并包含每条路由的请求数、服务器错误数和P95延迟。
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
			P95LatencyMS:     route.p95LatencyMS,
		})
	}
	return result
}

// compareRequestPerformanceTraffic 按请求数降序比较路由聚合结果，并以路由标识作为确定性平局排序键。
func compareRequestPerformanceTraffic(left, right *requestPerformanceRouteAggregate) bool {
	if left.totalRequests != right.totalRequests {
		return left.totalRequests > right.totalRequests
	}
	return requestPerformanceRouteLess(left, right)
}

// compareRequestPerformanceServerErrors 按服务端错误数、请求总数和路由标识比较 left 是否应排在 right 之前。
func compareRequestPerformanceServerErrors(left, right *requestPerformanceRouteAggregate) bool {
	if left.serverErrorCount != right.serverErrorCount {
		return left.serverErrorCount > right.serverErrorCount
	}
	if left.totalRequests != right.totalRequests {
		return left.totalRequests > right.totalRequests
	}
	return requestPerformanceRouteLess(left, right)
}

// compareRequestPerformanceP95Latency 按 P95 延迟、请求总数及路由标识的优先级比较两个路由聚合结果，确定左侧是否应排在右侧之前。
func compareRequestPerformanceP95Latency(left, right *requestPerformanceRouteAggregate) bool {
	leftP95 := left.p95LatencyMS
	rightP95 := right.p95LatencyMS
	if leftP95 != rightP95 {
		return leftP95 > rightP95
	}
	if left.totalRequests != right.totalRequests {
		return left.totalRequests > right.totalRequests
	}
	return requestPerformanceRouteLess(left, right)
}

func requestPerformanceRouteLess(left, right *requestPerformanceRouteAggregate) bool {
	if left.method != right.method {
		return left.method < right.method
	}
	return left.route < right.route
}

func requestPerformanceRouteValue(route sql.NullString) string {
	if !route.Valid || route.String == "" {
		return requestPerformanceUnmatchedRoute
	}
	return route.String
}

var _ moduleapi.RequestPerformanceReader = (*accessLogRepository)(nil)
