// Package runtimetarget owns persisted runtime connection identities and discovery facts.
package runtimetarget

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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

const runtimeTargetListSummaryConcurrency = 4

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
	if err := ctx.Services.RegisterSingleton((*moduleapi.ComposeRuntimeTargetReader)(nil), func(_ containerdi.Resolver) (any, error) { return runtimeTargetReader{repository: m.repository}, nil }); err != nil {
		return err
	}
	ctx.Router.GET("/runtime-targets", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.ViewPermission, publisher), m.handleList)
	ctx.Router.POST("/runtime-targets/discover-local-docker", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.RefreshPermission, publisher), m.handleDiscoverLocal(ctx))
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

type runtimeTargetReader struct {
	repository              *store.SQLRepository
	composeProjectNameProbe dockerComposeProjectNameProbe
}

func (r runtimeTargetReader) ReadDockerTarget(ctx context.Context, id *int64) (moduleapi.RuntimeTargetSummary, error) {
	if r.repository == nil {
		return moduleapi.RuntimeTargetSummary{}, store.ErrNotFound
	}
	if id != nil {
		if *id < 1 {
			return moduleapi.RuntimeTargetSummary{}, store.ErrNotFound
		}
		target, err := r.repository.Get(ctx, uint64(*id))
		if err != nil || target.Provider != "docker" || target.ID > maxRuntimeTargetID {
			return moduleapi.RuntimeTargetSummary{}, store.ErrNotFound
		}
		return moduleapi.RuntimeTargetSummary{ID: int64(target.ID), DisplayName: target.DisplayName, Provider: target.Provider}, nil
	}
	target, err := r.repository.FindSystemLocalDocker(ctx)
	if err != nil || target.ID > maxRuntimeTargetID {
		return moduleapi.RuntimeTargetSummary{}, store.ErrNotFound
	}
	return moduleapi.RuntimeTargetSummary{ID: int64(target.ID), DisplayName: target.DisplayName, Provider: target.Provider}, nil
}

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
		if target.Provider == "docker" && target.ID <= maxRuntimeTargetID {
			results = append(results, moduleapi.RuntimeTargetSummary{ID: int64(target.ID), DisplayName: target.DisplayName, Provider: target.Provider})
		}
	}
	return results, nil
}

func (r runtimeTargetReader) ReadComposeTarget(ctx context.Context, id *int64) (moduleapi.ComposeRuntimeTargetSummary, error) {
	if r.repository == nil {
		return moduleapi.ComposeRuntimeTargetSummary{}, store.ErrNotFound
	}
	if id != nil {
		if *id < 1 {
			return moduleapi.ComposeRuntimeTargetSummary{}, store.ErrNotFound
		}
		target, err := r.repository.Get(ctx, uint64(*id))
		if err != nil {
			return moduleapi.ComposeRuntimeTargetSummary{}, store.ErrNotFound
		}
		summary, ok := composeTargetSummary(target)
		if !ok {
			return moduleapi.ComposeRuntimeTargetSummary{}, store.ErrNotFound
		}
		return summary, nil
	}
	items, err := r.repository.List(ctx)
	if err != nil {
		return moduleapi.ComposeRuntimeTargetSummary{}, err
	}
	for _, target := range items {
		if summary, ok := composeTargetSummary(target); ok {
			return summary, nil
		}
	}
	return moduleapi.ComposeRuntimeTargetSummary{}, store.ErrNotFound
}

// ListComposeTargets exposes only targets that can execute Compose and access a managed workspace.
func (r runtimeTargetReader) ListComposeTargets(ctx context.Context) ([]moduleapi.ComposeRuntimeTargetSummary, error) {
	if r.repository == nil {
		return []moduleapi.ComposeRuntimeTargetSummary{}, nil
	}
	items, err := r.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]moduleapi.ComposeRuntimeTargetSummary, 0, len(items))
	for _, target := range items {
		if summary, ok := composeTargetSummary(target); ok {
			results = append(results, summary)
		}
	}
	return results, nil
}

// composeTargetSummary validates the provider-neutral capability contract while keeping
// endpoint and credential data inside the runtime-target module.
func composeTargetSummary(target store.Target) (moduleapi.ComposeRuntimeTargetSummary, bool) {
	if target.ID > maxRuntimeTargetID || !hasComposeCapabilities(target.Capabilities) {
		return moduleapi.ComposeRuntimeTargetSummary{}, false
	}
	return moduleapi.ComposeRuntimeTargetSummary{ID: int64(target.ID), DisplayName: target.DisplayName, Provider: target.Provider, Capabilities: append([]string(nil), target.Capabilities...), Available: target.Availability}, true
}

func hasComposeCapabilities(capabilities []string) bool {
	seen := make(map[string]bool, len(capabilities))
	for _, capability := range capabilities {
		seen[capability] = true
	}
	return seen["compose_execution"] && seen["workspace_access"]
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
	mapped := mapRuntimeTargetSummaries(c.Request.Context(), page.Items, runtimeTargetListSummaryConcurrency, func(ctx context.Context, item store.Target) generated.RuntimeTargetSummary {
		return m.toHTTPSummary(ctx, item)
	})
	httpx.WriteSuccess(c, http.StatusOK, generated.RuntimeTargetListResponse{Items: mapped, Total: page.Total, Limit: limit, Offset: offset})
}

func mapRuntimeTargetSummaries(ctx context.Context, items []store.Target, concurrency int, mapItem func(context.Context, store.Target) generated.RuntimeTargetSummary) []generated.RuntimeTargetSummary {
	if len(items) == 0 {
		return []generated.RuntimeTargetSummary{}
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(items) {
		concurrency = len(items)
	}
	mapped := make([]generated.RuntimeTargetSummary, len(items))
	jobs := make(chan int)
	var workers sync.WaitGroup
	workers.Add(concurrency)
	for worker := 0; worker < concurrency; worker++ {
		go func() {
			defer workers.Done()
			for index := range jobs {
				itemCtx, cancel := context.WithTimeout(ctx, runtimeTargetSummaryTimeout)
				mapped[index] = mapItem(itemCtx, items[index])
				cancel()
			}
		}()
	}
	for index := range items {
		jobs <- index
	}
	close(jobs)
	workers.Wait()
	return mapped
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

func (m *Module) snapshot(ctx context.Context, target store.Target) targetRuntimeSummary {
	if m == nil || m.summaries == nil {
		return unavailableTargetSummary("Runtime target is unavailable", time.Now().UTC())
	}
	return m.summaries.get(ctx, target)
}

func (m *Module) toHTTPSummary(ctx context.Context, target store.Target) generated.RuntimeTargetSummary {
	if target.ID > maxRuntimeTargetID {
		return generated.RuntimeTargetSummary{}
	}
	snapshot := m.snapshot(ctx, target)
	checkedAt := snapshot.CheckedAt
	response := generated.RuntimeTargetSummary{Id: int64(target.ID), DisplayName: target.DisplayName}
	response.Runtime.Provider = generated.RuntimeTargetSummaryRuntimeProviderDocker
	response.Runtime.Type = generated.RuntimeTargetSummaryRuntimeTypeContainerRuntime
	response.Runtime.Version = snapshot.Version
	response.Runtime.ApiVersion = snapshot.APIVersion
	response.Connection.Endpoint = target.EndpointLabel
	response.Connection.Kind = generated.RuntimeTargetSummaryConnectionKindUnixSocket
	response.Health.LastCheckedAt = &checkedAt
	response.Health.Diagnostic = snapshot.Diagnostic
	response.Health.Status = generated.RuntimeTargetSummaryHealthStatusUnavailable
	if snapshot.Healthy {
		response.Health.Status = generated.RuntimeTargetSummaryHealthStatusHealthy
	}
	response.Resources.Workloads = toHTTPCountMetric(snapshot.Workloads)
	response.Resources.Cpu = toHTTPUsageMetric(snapshot.CPU)
	response.Resources.Memory = toHTTPUsageMetric(snapshot.Memory)
	response.Resources.Storage = toHTTPUsageMetric(snapshot.Disk)
	return response
}

func (m *Module) toHTTP(ctx context.Context, target store.Target) generated.RuntimeTarget {
	if target.ID > maxRuntimeTargetID {
		return generated.RuntimeTarget{}
	}
	snapshot := m.snapshot(ctx, target)
	checkedAt := snapshot.CheckedAt
	response := generated.RuntimeTarget{Id: int64(target.ID), DisplayName: target.DisplayName}
	response.Runtime.Provider = generated.RuntimeTargetRuntimeProviderDocker
	response.Runtime.Type = generated.RuntimeTargetRuntimeTypeContainerRuntime
	response.Runtime.Version = snapshot.Version
	response.Runtime.ApiVersion = snapshot.APIVersion
	response.Connection.Endpoint = target.EndpointLabel
	response.Connection.Kind = generated.RuntimeTargetConnectionKindUnixSocket
	response.Health.LastCheckedAt = &checkedAt
	response.Health.Diagnostic = snapshot.Diagnostic
	response.Health.Status = generated.RuntimeTargetHealthStatusUnavailable
	if snapshot.Healthy {
		response.Health.Status = generated.RuntimeTargetHealthStatusHealthy
	}
	response.Resources.Workloads = toHTTPCountMetric(snapshot.Workloads)
	response.Resources.Cpu = toHTTPUsageMetric(snapshot.CPU)
	response.Resources.Memory = toHTTPUsageMetric(snapshot.Memory)
	response.Resources.Storage = toHTTPUsageMetric(snapshot.Disk)
	response.ProviderDetails.Provider = generated.RuntimeTargetProviderDetailsProviderDocker
	response.ProviderDetails.Docker.Images = toHTTPProviderCountMetric(snapshot.Images)
	response.ProviderDetails.Docker.Volumes = toHTTPProviderCountMetric(snapshot.Volumes)
	response.ProviderDetails.Docker.Networks = toHTTPProviderCountMetric(snapshot.Networks)
	return response
}

// toHTTPCountMetric 将目标计数指标转换为 HTTP 响应中的运行时目标计数指标。
func toHTTPCountMetric(metric targetCountMetric) generated.RuntimeTargetCountMetric {
	return generated.RuntimeTargetCountMetric{Available: metric.Available, Total: metric.Total, Active: metric.Active, UnavailableReason: metric.UnavailableReason}
}

// toHTTPProviderCountMetric converts an image metric to its HTTP response representation.
func toHTTPProviderCountMetric(metric targetImageMetric) generated.RuntimeTargetImageMetric {
	return generated.RuntimeTargetImageMetric{Available: metric.Available, Total: metric.Total, UnavailableReason: metric.UnavailableReason}
}

// toHTTPUsageMetric 将运行时使用量指标转换为 HTTP 响应中的使用量指标。
func toHTTPUsageMetric(metric targetUsageMetric) generated.RuntimeTargetUsageMetric {
	return generated.RuntimeTargetUsageMetric{Available: metric.Available, UsedBytes: metric.UsedBytes, TotalBytes: metric.TotalBytes, UsagePercent: metric.UsagePercent, UnavailableReason: metric.UnavailableReason}
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
