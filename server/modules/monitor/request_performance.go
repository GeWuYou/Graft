package monitor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	generated "graft/server/internal/contract/openapi/generated"
	monitoropenapi "graft/server/internal/contract/openapi/monitor"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/moduleapi"
	monitorcontract "graft/server/modules/monitor/contract"
)

const requestPerformanceRateScale = 100

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
			handler.logRequestPerformanceError(ginCtx, err)
			httpx.AbortLocalizedError(ginCtx, handler.localizer(), http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
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

func parseGeneratedRequestPerformanceRange(raw *monitoropenapi.GetMonitorRequestPerformanceParamsRange) monitorcontract.TrendRange {
	if raw == nil {
		return monitorcontract.TrendRange10Minutes
	}
	return parseTrendRange(string(*raw))
}

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

func requestPerformanceStatusGroups(groups moduleapi.RequestPerformanceStatusGroups, total int64) []generated.RequestPerformanceStatusGroup {
	return []generated.RequestPerformanceStatusGroup{
		{StatusGroup: generated.RequestPerformanceStatusGroupStatusGroupN2xx, RequestCount: groups.TwoXX, RequestRate: percentage(groups.TwoXX, total)},
		{StatusGroup: generated.RequestPerformanceStatusGroupStatusGroupN3xx, RequestCount: groups.ThreeXX, RequestRate: percentage(groups.ThreeXX, total)},
		{StatusGroup: generated.RequestPerformanceStatusGroupStatusGroupN4xx, RequestCount: groups.FourXX, RequestRate: percentage(groups.FourXX, total)},
		{StatusGroup: generated.RequestPerformanceStatusGroupStatusGroupN5xx, RequestCount: groups.FiveXX, RequestRate: percentage(groups.FiveXX, total)},
	}
}

func requestPerformanceTopRoutesResponse(items moduleapi.RequestPerformanceTopRoutes) generated.RequestPerformanceTopRoutes {
	return generated.RequestPerformanceTopRoutes{
		Traffic:    requestPerformanceRoutes(items.ByTraffic),
		Errors5xx:  requestPerformanceRoutes(items.ByServerErrors),
		P95Latency: requestPerformanceRoutes(items.ByP95Latency),
	}
}

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

func requestsPerSecond(total int64, seconds float64) float64 {
	if total <= 0 || seconds <= 0 {
		return 0
	}
	return float64(total) / seconds
}

func percentage(value int64, total int64) float64 {
	if value <= 0 || total <= 0 {
		return 0
	}
	return float64(value) * requestPerformanceRateScale / float64(total)
}

func (h *monitorServerHandler) logRequestPerformanceError(ginCtx *gin.Context, err error) {
	if h == nil || h.ctx == nil || h.ctx.Logger == nil {
		return
	}
	logsafe.Error(h.ctx.Logger, "build monitor request performance failed",
		zap.String("module", h.moduleName),
		zap.String("requestId", httpx.EnsureRequestID(ginCtx)),
		zap.Error(err),
	)
}
