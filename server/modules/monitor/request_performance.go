package monitor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	generated "graft/server/internal/contract/openapi/generated"
	monitoropenapi "graft/server/internal/contract/openapi/monitor"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/logger"
	"graft/server/internal/moduleapi"
	monitorcontract "graft/server/modules/monitor/contract"
)

const requestPerformanceRateScale = 100

// newRequestPerformanceHandler 创建请求性能监控的 HTTP 处理器。
// 它校验请求参数，构造性能响应，并在处理失败时写入本地化的客户端或服务端错误响应。
func newRequestPerformanceHandler(handler *monitorServerHandler) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		params := bindGeneratedRequestPerformanceParams(ginCtx)
		if err := handler.GetMonitorRequestPerformance(ginCtx.Request.Context(), params); err != nil {
			httpx.AbortLocalizedError(ginCtx, handler.localizer(), http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{"field": monitorcontract.RequestPerformanceRangeQueryKey})
			return
		}

		requestRange := parseGeneratedRequestPerformanceRange(params.Range)
		payload, err := buildRequestPerformanceResponse(ginCtx.Request.Context(), handler.requestPerformanceReader(), requestRange, time.Now().UTC())
		if err != nil {
			reported := handler.logRequestPerformanceError(ginCtx, err)
			httpx.AbortAppError(ginCtx, handler.localizer(), handler.runtimeLogger(), reported)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

func (h *monitorServerHandler) GetMonitorRequestPerformance(_ context.Context, params monitoropenapi.GetMonitorRequestPerformanceParams) error {
	if params.Range != nil && !params.Range.Valid() {
		return fmt.Errorf("invalid request-performance range %q", *params.Range)
	}
	return nil
}

func (h *monitorServerHandler) requestPerformanceReader() moduleapi.RequestPerformanceReader {
	if h == nil || h.instance == nil {
		return nil
	}
	return h.instance.requestPerformanceReader
}

func (h *monitorServerHandler) localizer() *i18n.Service {
	if h == nil || h.ctx == nil {
		return nil
	}
	return h.ctx.I18n
}

// bindGeneratedRequestPerformanceParams 从 Gin 请求上下文中提取并绑定请求性能接口的查询参数和请求头。
// 返回包含时间范围、请求 ID 和区域设置的请求参数。
func bindGeneratedRequestPerformanceParams(ginCtx *gin.Context) monitoropenapi.GetMonitorRequestPerformanceParams {
	params := monitoropenapi.GetMonitorRequestPerformanceParams{}
	if raw := strings.TrimSpace(ginCtx.Query(monitorcontract.RequestPerformanceRangeQueryKey)); raw != "" {
		value := monitoropenapi.GetMonitorRequestPerformanceParamsRange(raw)
		params.Range = &value
	}
	if raw := strings.TrimSpace(ginCtx.GetHeader(httpx.RequestIDHeader)); raw != "" {
		params.XRequestId = &raw
	}
	if raw := strings.TrimSpace(ginCtx.GetHeader(string(httpheader.Locale))); raw != "" {
		params.XGraftLocale = &raw
	}
	return params
}

// parseGeneratedRequestPerformanceRange 将可选请求范围转换为监控趋势范围；未提供范围时默认使用 10 分钟。
func parseGeneratedRequestPerformanceRange(raw *monitoropenapi.GetMonitorRequestPerformanceParamsRange) monitorcontract.TrendRange {
	if raw == nil {
		return monitorcontract.TrendRange10Minutes
	}
	return parseTrendRange(string(*raw))
}

// 读取请求性能聚合数据，并将其转换为生成的响应模型；读取器不可用或读取失败时返回错误。
func buildRequestPerformanceResponse(
	ctx context.Context,
	reader moduleapi.RequestPerformanceReader,
	requestRange monitorcontract.TrendRange,
	observedAt time.Time,
) (generated.RequestPerformanceResponse, error) {
	if reader == nil {
		return generated.RequestPerformanceResponse{}, errors.New("request performance reader is unavailable")
	}

	observedAt = observedAt.UTC()
	summary, err := reader.ReadRequestPerformance(ctx, moduleapi.RequestPerformanceQuery{
		WindowStart: observedAt.Add(-requestRange.Duration()),
		WindowEnd:   observedAt,
		BucketSize:  moduleapi.RequestPerformanceMinuteBucketSize,
	})
	if err != nil {
		return generated.RequestPerformanceResponse{}, fmt.Errorf("read request performance: %w", err)
	}

	durationSeconds := requestRange.Duration().Seconds()
	return generated.RequestPerformanceResponse{
		ObservedAt: observedAt,
		Range:      generated.RequestPerformanceResponseRange(requestRange.String()),
		Summary: generated.RequestPerformanceSummary{
			TotalRequests:     summary.TotalRequests,
			RequestsPerSecond: requestsPerSecond(summary.TotalRequests, durationSeconds),
			P50LatencyMs:      float64(summary.P50LatencyMS),
			P95LatencyMs:      float64(summary.P95LatencyMS),
			Error5xxCount:     summary.ServerErrorCount,
			Error5xxRate:      percentage(summary.ServerErrorCount, summary.TotalRequests),
			SlowRequestCount:  summary.SlowRequestCount,
		},
		MinuteBuckets: requestPerformanceMinuteBuckets(summary.Buckets),
		StatusGroups:  requestPerformanceStatusGroups(summary.StatusGroups, summary.TotalRequests),
		TopRoutes:     requestPerformanceTopRoutesResponse(summary.TopRoutes),
	}, nil
}

// requestPerformanceMinuteBuckets 将每分钟请求性能指标转换为响应模型中的分钟桶。
// 返回包含观测时间、请求量、请求速率、P95 延迟及 5xx 错误指标的分钟桶列表。
func requestPerformanceMinuteBuckets(items []moduleapi.RequestPerformanceMinuteBucket) []generated.RequestPerformanceMinuteBucket {
	result := make([]generated.RequestPerformanceMinuteBucket, 0, len(items))
	for _, item := range items {
		result = append(result, generated.RequestPerformanceMinuteBucket{
			ObservedAt:        item.Start.UTC(),
			TotalRequests:     item.TotalRequests,
			RequestsPerSecond: requestsPerSecond(item.TotalRequests, moduleapi.RequestPerformanceMinuteBucketSize.Seconds()),
			P95LatencyMs:      float64(item.P95LatencyMS),
			Error5xxCount:     item.ServerErrorCount,
			Error5xxRate:      percentage(item.ServerErrorCount, item.TotalRequests),
		})
	}
	return result
}

// requestPerformanceStatusGroups 将 HTTP 状态分组计数转换为包含百分比的响应条目。
func requestPerformanceStatusGroups(groups moduleapi.RequestPerformanceStatusGroups, total int64) []generated.RequestPerformanceStatusGroup {
	return []generated.RequestPerformanceStatusGroup{
		{StatusGroup: generated.RequestPerformanceStatusGroupStatusGroupN2xx, RequestCount: groups.TwoXX, RequestRate: percentage(groups.TwoXX, total)},
		{StatusGroup: generated.RequestPerformanceStatusGroupStatusGroupN3xx, RequestCount: groups.ThreeXX, RequestRate: percentage(groups.ThreeXX, total)},
		{StatusGroup: generated.RequestPerformanceStatusGroupStatusGroupN4xx, RequestCount: groups.FourXX, RequestRate: percentage(groups.FourXX, total)},
		{StatusGroup: generated.RequestPerformanceStatusGroupStatusGroupN5xx, RequestCount: groups.FiveXX, RequestRate: percentage(groups.FiveXX, total)},
	}
}

// requestPerformanceTopRoutesResponse 将请求性能 Top 路由数据转换为流量、5xx 错误和 P95 延迟三个响应分组。
func requestPerformanceTopRoutesResponse(items moduleapi.RequestPerformanceTopRoutes) generated.RequestPerformanceTopRoutes {
	return generated.RequestPerformanceTopRoutes{
		Traffic:    requestPerformanceRoutes(items.ByTraffic),
		Errors5xx:  requestPerformanceRoutes(items.ByServerErrors),
		P95Latency: requestPerformanceRoutes(items.ByP95Latency),
	}
}

// requestPerformanceRoutes 将请求性能路由指标转换为生成的响应路由列表。
func requestPerformanceRoutes(items []moduleapi.RequestPerformanceRoute) []generated.RequestPerformanceRoute {
	result := make([]generated.RequestPerformanceRoute, 0, len(items))
	for _, item := range items {
		result = append(result, generated.RequestPerformanceRoute{
			Method:        item.Method,
			Route:         item.Route,
			TotalRequests: item.TotalRequests,
			Error5xxCount: item.ServerErrorCount,
			Error5xxRate:  percentage(item.ServerErrorCount, item.TotalRequests),
			P95LatencyMs:  float64(item.P95LatencyMS),
		})
	}
	return result
}

// requestsPerSecond 计算指定时长内的请求速率。
// 当请求数或时长为零或负数时返回零。
func requestsPerSecond(total int64, seconds float64) float64 {
	if total <= 0 || seconds <= 0 {
		return 0
	}
	return float64(total) / seconds
}

// percentage 计算值占总数的百分比。
// 当值或总数小于等于零时，返回 0。
func percentage(value int64, total int64) float64 {
	if value <= 0 || total <= 0 {
		return 0
	}
	return float64(value) * requestPerformanceRateScale / float64(total)
}

func (h *monitorServerHandler) logRequestPerformanceError(ginCtx *gin.Context, err error) error {
	if h == nil || h.ctx == nil {
		return err
	}
	if h.ctx.AppLogger == nil {
		return err
	}
	return logger.ReportError(ginCtx.Request.Context(), h.ctx.AppLogger.Named("modules.monitor.request_performance"), "build monitor request performance failed", err,
		logger.StringField("module", h.moduleName),
		logger.StringField(logger.FieldOperation, "read_request_performance"),
	)
}
