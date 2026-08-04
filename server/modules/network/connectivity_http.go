package network

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
)

// connectivityTargetResponse 是 registry 描述的 HTTP 投影，不携带 URL、凭据或可执行网络配置。
type connectivityTargetResponse struct {
	ID         string   `json:"id"`
	ModuleID   string   `json:"module_id"`
	Category   string   `json:"category"`
	TitleKey   string   `json:"title_key"`
	ProbeKinds []string `json:"probe_kinds"`
	Features   []string `json:"features"`
}

type connectivityCheckResponse struct {
	CheckID   int64     `json:"check_id"`
	TargetID  string    `json:"target_id"`
	Status    string    `json:"status"`
	LatencyMS int64     `json:"latency_ms"`
	CheckedAt time.Time `json:"checked_at"`
}

type connectivityAggregateResponse struct {
	LastRunAt        *time.Time `json:"last_run_at,omitempty"`
	TargetCount      int        `json:"target_count"`
	HealthyCount     int        `json:"healthy_count"`
	DegradedCount    int        `json:"degraded_count"`
	FailedCount      int        `json:"failed_count"`
	AverageLatencyMS int64      `json:"average_latency_ms"`
	WorstTargetID    string     `json:"worst_target_id,omitempty"`
	WorstLatencyMS   int64      `json:"worst_latency_ms"`
}

type connectivityProbeResponse struct {
	Kind       string    `json:"kind"`
	Status     string    `json:"status"`
	DurationMS int64     `json:"duration_ms"`
	Summary    string    `json:"summary,omitempty"`
	ErrorCode  string    `json:"error_code,omitempty"`
	OccurredAt time.Time `json:"occurred_at,omitempty"`
}

type connectivityRouteResponse struct {
	MatchedStrategy string `json:"matched_strategy"`
	Decision        string `json:"decision"`
	Reason          string `json:"reason"`
}

type connectivityExitIPResponse struct {
	Masked    string `json:"masked"`
	Available bool   `json:"available"`
}

type connectivityReportResponse struct {
	SchemaVersion  int                         `json:"schema_version"`
	TargetID       string                      `json:"target_id"`
	Status         string                      `json:"status"`
	CheckedAt      time.Time                   `json:"checked_at"`
	TotalLatencyMS int64                       `json:"total_latency_ms"`
	Probes         []connectivityProbeResponse `json:"probes"`
	Route          *connectivityRouteResponse  `json:"route,omitempty"`
	ExitIP         *connectivityExitIPResponse `json:"exit_ip,omitempty"`
}

func (r routeRuntime) handleConnectivityTargets(ginCtx *gin.Context) {
	targets, err := r.service.ConnectivityTargets()
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	items := make([]connectivityTargetResponse, 0, len(targets))
	for _, target := range targets {
		items = append(items, toConnectivityTargetResponse(target))
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, gin.H{"items": items})
}

func (r routeRuntime) handleConnectivityLatest(ginCtx *gin.Context) {
	items, err := r.service.ConnectivityLatest(ginCtx.Request.Context())
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, gin.H{"items": toConnectivityCheckResponses(items)})
}

func (r routeRuntime) handleConnectivityAggregate(ginCtx *gin.Context) {
	value, err := r.service.ConnectivityAggregate(ginCtx.Request.Context())
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toConnectivityAggregateResponse(value))
}

func (r routeRuntime) handleConnectivityRun(ginCtx *gin.Context) {
	targetID := moduleapi.ConnectivityTargetID(strings.TrimSpace(ginCtx.Param("targetId")))
	check, report, err := r.service.RunConnectivity(ginCtx.Request.Context(), targetID)
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, gin.H{"check": toConnectivityCheckResponse(check), "report": toConnectivityReportResponse(report)})
}

func (r routeRuntime) handleConnectivityHistory(ginCtx *gin.Context) {
	targetID := moduleapi.ConnectivityTargetID(strings.TrimSpace(ginCtx.Param("targetId")))
	limit, ok := connectivityHistoryLimit(ginCtx)
	if !ok {
		r.badRequest(ginCtx)
		return
	}
	items, err := r.service.ConnectivityHistory(ginCtx.Request.Context(), targetID, limit)
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, gin.H{"target_id": targetID, "items": toConnectivityCheckResponses(items)})
}

func (r routeRuntime) handleConnectivityReport(ginCtx *gin.Context) {
	r.writeConnectivityReport(ginCtx, false)
}
func (r routeRuntime) handleConnectivityTrace(ginCtx *gin.Context) {
	r.writeConnectivityReport(ginCtx, true)
}
func (r routeRuntime) handleConnectivityExport(ginCtx *gin.Context) {
	report, ok := r.loadConnectivityReport(ginCtx)
	if !ok {
		return
	}
	ginCtx.Header("Content-Disposition", "attachment; filename=connectivity-report.json")
	httpx.WriteSuccess(ginCtx, http.StatusOK, toConnectivityReportResponse(report))
}

func (r routeRuntime) writeConnectivityReport(ginCtx *gin.Context, traceOnly bool) {
	report, ok := r.loadConnectivityReport(ginCtx)
	if !ok {
		return
	}
	response := toConnectivityReportResponse(report)
	if traceOnly {
		httpx.WriteSuccess(ginCtx, http.StatusOK, gin.H{"target_id": response.TargetID, "check_id": ginCtx.Param("checkId"), "probes": response.Probes})
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, response)
}

func (r routeRuntime) loadConnectivityReport(ginCtx *gin.Context) (moduleapi.ConnectivityReport, bool) {
	checkID, err := strconv.ParseInt(ginCtx.Param("checkId"), 10, 64)
	if err != nil || checkID < 1 {
		r.badRequest(ginCtx)
		return moduleapi.ConnectivityReport{}, false
	}
	report, err := r.service.ConnectivityReport(ginCtx.Request.Context(), moduleapi.ConnectivityTargetID(strings.TrimSpace(ginCtx.Param("targetId"))), checkID)
	if err != nil {
		r.writeError(ginCtx, err)
		return moduleapi.ConnectivityReport{}, false
	}
	return report, true
}

func connectivityHistoryLimit(ginCtx *gin.Context) (int, bool) {
	raw := strings.TrimSpace(ginCtx.DefaultQuery("limit", "20"))
	limit, err := strconv.Atoi(raw)
	return limit, err == nil && limit >= 1 && limit <= maxConnectivityHistoryLimit
}

func toConnectivityTargetResponse(value moduleapi.ConnectivityTargetDescriptor) connectivityTargetResponse {
	probes := make([]string, 0, len(value.Capabilities.ProbeKinds))
	for _, item := range value.Capabilities.ProbeKinds {
		probes = append(probes, string(item))
	}
	features := make([]string, 0, len(value.Capabilities.Features))
	for _, item := range value.Capabilities.Features {
		features = append(features, string(item))
	}
	return connectivityTargetResponse{ID: string(value.ID), ModuleID: value.ModuleID, Category: value.Category, TitleKey: value.TitleKey, ProbeKinds: probes, Features: features}
}
func toConnectivityCheckResponses(values []ConnectivityCheck) []connectivityCheckResponse {
	items := make([]connectivityCheckResponse, 0, len(values))
	for _, item := range values {
		items = append(items, toConnectivityCheckResponse(item))
	}
	return items
}
func toConnectivityCheckResponse(value ConnectivityCheck) connectivityCheckResponse {
	return connectivityCheckResponse{CheckID: value.ID, TargetID: string(value.TargetID), Status: string(value.Status), LatencyMS: value.Latency.Milliseconds(), CheckedAt: value.CheckedAt.UTC()}
}
func toConnectivityAggregateResponse(value ConnectivityAggregate) connectivityAggregateResponse {
	return connectivityAggregateResponse{LastRunAt: value.LastRunAt, TargetCount: value.TargetCount, HealthyCount: value.HealthyCount, DegradedCount: value.DegradedCount, FailedCount: value.FailedCount, AverageLatencyMS: value.AverageLatency.Milliseconds(), WorstTargetID: string(value.WorstTargetID), WorstLatencyMS: value.WorstLatency.Milliseconds()}
}
func toConnectivityReportResponse(value moduleapi.ConnectivityReport) connectivityReportResponse {
	probes := make([]connectivityProbeResponse, 0, len(value.Probes))
	for _, probe := range value.Probes {
		probes = append(probes, connectivityProbeResponse{Kind: string(probe.Kind), Status: string(probe.Status), DurationMS: probe.Duration.Milliseconds(), Summary: probe.Summary, ErrorCode: probe.ErrorCode, OccurredAt: probe.OccurredAt.UTC()})
	}
	response := connectivityReportResponse{SchemaVersion: value.SchemaVersion, TargetID: string(value.TargetID), Status: string(value.Status), CheckedAt: value.CheckedAt.UTC(), TotalLatencyMS: value.TotalLatency.Milliseconds(), Probes: probes}
	if value.Route != nil {
		response.Route = &connectivityRouteResponse{MatchedStrategy: value.Route.MatchedStrategy, Decision: value.Route.Decision, Reason: value.Route.Reason}
	}
	if value.ExitIP != nil {
		response.ExitIP = &connectivityExitIPResponse{Masked: value.ExitIP.Masked, Available: value.ExitIP.Available}
	}
	return response
}
