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
	contract "graft/server/modules/runtime-target/contract"
	store "graft/server/modules/runtime-target/store"
)

const maxRuntimeTargetID = uint64(^uint64(0) >> 1)

// Module exposes runtime-target API routes and bounded Local Docker discovery.
type Module struct{ repository *store.SQLRepository }

// NewModule constructs the runtime-target module.
func NewModule(repository *store.SQLRepository) *Module { return &Module{repository: repository} }

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
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	if err := ctx.Services.RegisterSingleton((*moduleapi.RuntimeTargetReader)(nil), func(_ containerdi.Resolver) (any, error) { return runtimeTargetReader{repository: m.repository}, nil }); err != nil {
		return err
	}
	ctx.Router.GET("/runtime-targets", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.ViewPermission, publisher), m.handleList)
	ctx.Router.GET("/runtime-targets/:id", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.ViewPermission, publisher), m.handleDetail)
	ctx.Router.POST("/runtime-targets/:id/refresh", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.RefreshPermission, publisher), m.handleRefresh(ctx))
	return nil
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
	return discoverLocalDocker(ctx.LifecycleContext, m.repository)
}

// Shutdown releases no resources because discovery is request-bounded.
func (m *Module) Shutdown(*module.Context) error { return nil }

func (m *Module) handleList(c *gin.Context) {
	items, err := m.repository.List(c.Request.Context())
	if err != nil {
		httpx.AbortLocalizedError(c, nil, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		return
	}
	mapped := make([]generated.RuntimeTarget, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, toHTTP(item))
	}
	httpx.WriteSuccess(c, http.StatusOK, generated.RuntimeTargetListResponse{Items: mapped})
}

func (m *Module) handleDetail(c *gin.Context) {
	target, ok := m.readTarget(c)
	if !ok {
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, toHTTP(target))
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
		httpx.WriteSuccess(c, http.StatusOK, toHTTP(refreshed))
	}
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
func toHTTP(target store.Target) generated.RuntimeTarget {
	if target.ID > maxRuntimeTargetID {
		return generated.RuntimeTarget{}
	}
	return generated.RuntimeTarget{Id: int64(target.ID), Provider: target.Provider, DisplayName: target.DisplayName, EndpointLabel: target.EndpointLabel, ConnectionKind: target.ConnectionKind, Capabilities: target.Capabilities, Availability: target.Availability, LastError: target.LastError, LastCheckedAt: target.CheckedAt}
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
