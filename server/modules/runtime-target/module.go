// Package runtimetarget owns persisted runtime connection identities and discovery facts.
package runtimetarget

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	containerdi "graft/server/internal/container"

	messagecontract "graft/server/internal/contract/message"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
	contract "graft/server/modules/runtime-target/contract"
	store "graft/server/modules/runtime-target/store"
)

const maxRuntimeTargetID = uint64(^uint64(0) >> 1)

// Module exposes runtime-target API routes and bounded Local Docker discovery.
type Module struct {
	repository      *store.SQLRepository
	summaries       *summaryCache
	authorizer      moduleapi.Authorizer
	realtimeTickets realtimeauth.Service
	topicIssuers    realtime.TopicIssuerRegistry
	collector       *runtimeTargetSummaryCollector
}

// NewModule constructs the runtime-target module.
func NewModule(repository *store.SQLRepository) *Module {
	return &Module{repository: repository, summaries: newSummaryCache()}
}

// Register declares runtime-target permissions, menu metadata, and API routes.
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.repository == nil {
		return errors.New("runtime target repository is unavailable")
	}
	if err := registerModuleMetadata(ctx, moduleID); err != nil {
		return err
	}
	auth, err := module.ResolveService[moduleapi.AuthService](ctx.Services, (*moduleapi.AuthService)(nil))
	if err != nil {
		return err
	}
	authorizer, err := module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
	if err != nil {
		return err
	}
	if err := m.configureRealtime(ctx, authorizer); err != nil {
		return err
	}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	if err := ctx.Services.RegisterSingleton((*moduleapi.RuntimeTargetReader)(nil), func(_ containerdi.Resolver) (any, error) { return runtimeTargetReader{repository: m.repository}, nil }); err != nil {
		return err
	}
	ctx.Router.GET("/runtime-targets", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.ViewPermission, publisher), m.handleList)
	ctx.Router.POST("/runtime-targets/discover-local", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.RefreshPermission, publisher), m.handleDiscoverLocal(ctx))
	ctx.Router.GET("/runtime-targets/:id", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.ViewPermission, publisher), m.handleDetail)
	ctx.Router.POST("/runtime-targets/:id/refresh", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.RefreshPermission, publisher), m.handleRefresh(ctx))
	return nil
}

func (m *Module) configureRealtime(ctx *module.Context, authorizer moduleapi.Authorizer) error {
	realtimeTickets, err := module.ResolveService[realtimeauth.Service](ctx.Services, (*realtimeauth.Service)(nil))
	if err != nil {
		return err
	}
	topicIssuers, err := module.ResolveService[realtime.TopicIssuerRegistry](ctx.Services, (*realtime.TopicIssuerRegistry)(nil))
	if err != nil {
		return err
	}
	hub := ctx.Realtime
	if hub == nil {
		hub, err = module.ResolveService[realtime.Hub](ctx.Services, (*realtime.Hub)(nil))
		if err != nil {
			return err
		}
	}
	m.authorizer = authorizer
	m.realtimeTickets = realtimeTickets
	m.topicIssuers = topicIssuers
	m.collector = newRuntimeTargetSummaryCollector(hub, m.collectRealtimeSummaries)
	return m.topicIssuers.Register(contract.SummaryTopic, m)
}

type runtimeTargetReader struct{ repository *store.SQLRepository }

func (r runtimeTargetReader) ReadDockerTarget(ctx context.Context, id *int64) (moduleapi.RuntimeTargetSummary, error) {
	if r.repository == nil {
		return moduleapi.RuntimeTargetSummary{}, store.ErrNotFound
	}
	if id != nil {
		if *id < 1 {
			return moduleapi.RuntimeTargetSummary{}, store.ErrNotFound
		}
		target, err := r.repository.Get(ctx, uint64(*id))
		if err != nil {
			return moduleapi.RuntimeTargetSummary{}, store.ErrNotFound
		}
		summary, ok := dockerTargetSummary(target)
		if !ok {
			return moduleapi.RuntimeTargetSummary{}, store.ErrNotFound
		}
		return summary, nil
	}
	items, err := r.repository.List(ctx)
	if err != nil {
		return moduleapi.RuntimeTargetSummary{}, err
	}
	for _, target := range items {
		if summary, ok := dockerTargetSummary(target); ok {
			return summary, nil
		}
	}
	return moduleapi.RuntimeTargetSummary{}, store.ErrNotFound
}

// ListDockerTargets exposes target identity only. Consumers must not receive endpoint or credential fields.
func (r runtimeTargetReader) ListDockerTargets(ctx context.Context) ([]moduleapi.RuntimeTargetSummary, error) {
	if r.repository == nil {
		return []moduleapi.RuntimeTargetSummary{}, nil
	}
	items, err := r.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]moduleapi.RuntimeTargetSummary, 0, len(items))
	for _, target := range items {
		if summary, ok := dockerTargetSummary(target); ok {
			results = append(results, summary)
		}
	}
	return results, nil
}

// dockerTargetSummary 将可表示标识符的 Docker 目标转换为运行时目标摘要。
// 如果目标不是 Docker 目标或其标识符超出可表示范围，则返回 false。
func dockerTargetSummary(target store.Target) (moduleapi.RuntimeTargetSummary, bool) {
	if target.ID > maxRuntimeTargetID || target.Provider != "docker" {
		return moduleapi.RuntimeTargetSummary{}, false
	}
	return moduleapi.RuntimeTargetSummary{ID: int64(target.ID), DisplayName: target.DisplayName, Provider: target.Provider}, true
}

// Boot records the currently usable local Docker endpoint without making application boot depend on Docker.
func (m *Module) Boot(ctx *module.Context) error {
	if m == nil || m.repository == nil || ctx == nil {
		return nil
	}
	if err := discoverLocalDocker(ctx.LifecycleContext, m.repository); err != nil {
		return err
	}
	if m.collector == nil {
		return nil
	}
	return m.collector.Start(ctx.LifecycleContext)
}

// Shutdown releases the module-owned realtime collector.
func (m *Module) Shutdown(ctx *module.Context) error {
	if m == nil || m.collector == nil {
		return nil
	}
	if ctx == nil || ctx.LifecycleContext == nil {
		return errors.New("runtime target shutdown lifecycle context is required")
	}
	return m.collector.Stop(ctx.LifecycleContext)
}

func (m *Module) handleList(c *gin.Context) {
	limit, offset, ok := runtimeTargetListWindow(c)
	if !ok {
		return
	}
	page, err := m.repository.ListPage(c.Request.Context(), limit, offset)
	if err != nil {
		httpx.AbortLocalizedError(c, nil, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		return
	}
	mapped := make([]generated.RuntimeTarget, 0, len(page.Items))
	for _, item := range page.Items {
		mapped = append(mapped, m.toHTTP(c.Request.Context(), item))
	}
	httpx.WriteSuccess(c, http.StatusOK, generated.RuntimeTargetListResponse{Items: mapped, Total: page.Total, Limit: limit, Offset: offset})
}

func (m *Module) handleDetail(c *gin.Context) {
	target, ok := m.readTarget(c)
	if !ok {
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, m.toHTTP(c.Request.Context(), target))
}

func (m *Module) handleRefresh(moduleCtx *module.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		target, ok := m.readTarget(c)
		if !ok {
			return
		}
		refreshed, err := refreshTarget(c.Request.Context(), m.repository, target.ID)
		m.publishRefreshAudit(c.Request.Context(), moduleCtx, target, err)
		if err != nil {
			httpx.AbortLocalizedError(c, moduleCtx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}
		m.summaries.invalidate(target.ID)
		httpx.WriteSuccess(c, http.StatusOK, m.toHTTP(c.Request.Context(), refreshed))
	}
}

func (m *Module) handleDiscoverLocal(moduleCtx *module.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := discoverLocalDocker(c.Request.Context(), m.repository); err != nil {
			httpx.AbortLocalizedError(c, moduleCtx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}
		target, err := m.repository.FindSystemLocalDocker(c.Request.Context())
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteSuccess[any](c, http.StatusOK, nil)
			return
		}
		if err != nil {
			httpx.AbortLocalizedError(c, moduleCtx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}
		m.summaries.invalidate(target.ID)
		m.publishRefreshAudit(c.Request.Context(), moduleCtx, target, nil)
		httpx.WriteSuccess(c, http.StatusOK, m.toHTTP(c.Request.Context(), target))
	}
}

// runtimeTargetListWindow parses and validates pagination parameters from the request.
// It returns the requested limit and offset, or aborts the request with a bad-request
// response when either parameter is invalid.
func runtimeTargetListWindow(c *gin.Context) (int, int, bool) {
	limit := 10
	offset := 0
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || (parsed != 10 && parsed != 20 && parsed != 50 && parsed != 100) {
			httpx.AbortLocalizedError(c, nil, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
			return 0, 0, false
		}
		limit = parsed
	}
	if raw := c.Query("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			httpx.AbortLocalizedError(c, nil, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
			return 0, 0, false
		}
		offset = parsed
	}
	return limit, offset, true
}

func (m *Module) readTarget(c *gin.Context) (store.Target, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		httpx.AbortLocalizedError(c, nil, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return store.Target{}, false
	}
	target, err := m.repository.Get(c.Request.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		httpx.AbortLocalizedError(c, nil, http.StatusNotFound, "common.not_found", nil)
		return store.Target{}, false
	}
	if err != nil {
		httpx.AbortLocalizedError(c, nil, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		return store.Target{}, false
	}
	return target, true
}

// toHTTP converts a stored runtime target to its HTTP response representation.
// It returns an empty response when the target ID cannot be represented safely.
func (m *Module) toHTTP(ctx context.Context, target store.Target) generated.RuntimeTarget {
	if target.ID > maxRuntimeTargetID {
		return generated.RuntimeTarget{}
	}
	summary := unavailableTargetSummary("Docker target is unavailable")
	if m != nil && m.summaries != nil {
		summary = m.summaries.get(ctx, target)
	}
	return generated.RuntimeTarget{Id: int64(target.ID), Provider: target.Provider, DisplayName: target.DisplayName, EndpointLabel: target.EndpointLabel, ConnectionKind: target.ConnectionKind, Capabilities: target.Capabilities, Availability: target.Availability, LastError: target.LastError, LastCheckedAt: target.CheckedAt, Summary: toHTTPSummary(summary)}
}

// toHTTPSummary converts a runtime target summary to its HTTP response representation.
func toHTTPSummary(summary targetRuntimeSummary) generated.RuntimeTargetSummary {
	return generated.RuntimeTargetSummary{
		Containers: generated.RuntimeTargetCountMetric{
			Available:         summary.Containers.Available,
			Total:             summary.Containers.Total,
			Running:           summary.Containers.Running,
			Stopped:           summary.Containers.Stopped,
			UnavailableReason: summary.Containers.UnavailableReason,
		},
		Images: generated.RuntimeTargetImageMetric{
			Available:         summary.Images.Available,
			Total:             summary.Images.Total,
			Used:              summary.Images.Used,
			Unused:            summary.Images.Unused,
			UnavailableReason: summary.Images.UnavailableReason,
		},
		Cpu: generated.RuntimeTargetUsageMetric{
			Available:         summary.CPU.Available,
			UsedBytes:         summary.CPU.UsedBytes,
			TotalBytes:        summary.CPU.TotalBytes,
			UsagePercent:      summary.CPU.UsagePercent,
			UnavailableReason: summary.CPU.UnavailableReason,
		},
		Memory: generated.RuntimeTargetUsageMetric{
			Available:         summary.Memory.Available,
			UsedBytes:         summary.Memory.UsedBytes,
			TotalBytes:        summary.Memory.TotalBytes,
			UsagePercent:      summary.Memory.UsagePercent,
			UnavailableReason: summary.Memory.UnavailableReason,
		},
		Disk: generated.RuntimeTargetUsageMetric{
			Available:         summary.Disk.Available,
			UsedBytes:         summary.Disk.UsedBytes,
			TotalBytes:        summary.Disk.TotalBytes,
			UsagePercent:      summary.Disk.UsagePercent,
			UnavailableReason: summary.Disk.UnavailableReason,
		},
	}
}

func (m *Module) publishRefreshAudit(ctx context.Context, moduleCtx *module.Context, target store.Target, refreshErr error) {
	if moduleCtx == nil || moduleCtx.EventBus == nil {
		return
	}
	event := moduleapi.AuditEvent{Kind: moduleapi.AuditEventKindDomain, Action: "runtime_target.refresh", ResourceType: "runtime_target", ResourceID: strconv.FormatUint(target.ID, 10), ResourceName: strings.TrimSpace(target.DisplayName), StatusCode: http.StatusOK, Success: refreshErr == nil, Metadata: map[string]any{"provider": target.Provider, "result": map[bool]string{true: "success", false: "failure"}[refreshErr == nil]}}
	if refreshErr != nil {
		event.StatusCode = http.StatusInternalServerError
		event.MessageKey = messagecontract.CommonInternalError.String()
	}
	_ = moduleCtx.EventBus.Publish(ctx, eventbus.Event{Name: string(moduleapi.AuditRecordEventName), Source: moduleID, Payload: event})
}
