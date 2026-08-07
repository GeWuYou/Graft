package app

import (
	"errors"
	"fmt"

	capabilityruntime "graft/server/internal/capability"
	"graft/server/internal/container"
	capabilitycontract "graft/server/internal/contract/capability"
	"graft/server/internal/httpx"
	"graft/server/internal/logger"
	productmcp "graft/server/internal/mcp"
	"graft/server/internal/moduleapi"
	"graft/server/internal/moduleruntime"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
)

func (r *Runtime) registerCoreAuthenticatedRoutes() error {
	authService, authorizer, err := r.resolveLogExplorerAuth()
	if errors.Is(err, container.ErrServiceNotRegistered) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve log explorer auth service: %w", err)
	}
	return r.registerCoreAuthenticatedRoutesWith(authService, authorizer)
}

func (r *Runtime) registerCoreAuthenticatedRoutesWith(authService moduleapi.AuthService, authorizer moduleapi.Authorizer) error {
	if err := r.registerAccessLogExplorerWithAuth(authService, authorizer); err != nil {
		return err
	}
	if err := r.registerAppLogExplorerWithAuth(authService, authorizer); err != nil {
		return err
	}
	if err := r.registerModuleRuntimeWithAuth(authService, authorizer); err != nil {
		return err
	}
	if err := r.registerCoreDashboardWidgets(); err != nil {
		return err
	}
	if err := r.registerDashboardWithAuth(authService, authorizer); err != nil {
		return err
	}
	if err := r.registerCapabilityRoutes(authService, authorizer); err != nil {
		return fmt.Errorf("register capability routes: %w", err)
	}
	if err := r.registerRealtimeSubscriptionRoutes(); err != nil {
		return err
	}
	if err := r.registerMCPRuntime(authorizer); err != nil {
		return err
	}

	return nil
}

func (r *Runtime) registerCapabilityRoutes(authService moduleapi.AuthService, authorizer moduleapi.Authorizer) error {
	return capabilityruntime.RegisterRoutes(r.server.Engine().Group("/api"), r.services, r.capabilityCoordinator, httpx.RequirePermission(r.i18n, authService, authorizer, capabilitycontract.ReadPermission))
}

func (r *Runtime) registerMCPRuntime(authorizer moduleapi.Authorizer) error {
	if r == nil || r.config == nil || !r.config.MCP.Enabled {
		return nil
	}
	if r.server == nil {
		return errors.New("runtime server is unavailable")
	}

	personalTokens, err := resolveRuntimeService[moduleapi.PersonalAccessTokenService](
		r.services,
		(*moduleapi.PersonalAccessTokenService)(nil),
		"personal API token service",
		"personal API token service",
	)
	if err != nil {
		return fmt.Errorf("resolve MCP personal API token service: %w", err)
	}

	mcpRuntime, err := productmcp.Register(productmcp.HTTPRegistration{
		Engine:                 r.server.Engine(),
		OpenAPISpec:            OpenAPIDocsBundle(),
		I18n:                   r.i18n,
		PersonalTokenService:   personalTokens,
		Authorizer:             authorizer,
		SecurityAuditPublisher: httpx.NewSecurityAuditPublisher(r.eventBus, r.logger, "mcp"),
		ConfirmationTokenTTL:   r.config.MCP.ConfirmationTokenTTL,
		Limits: productmcp.RuntimeLimits{
			SessionTimeout:        r.config.MCP.SessionTimeout,
			RequestTimeout:        r.config.MCP.RequestTimeout,
			MaxRequestBytes:       r.config.MCP.MaxRequestBytes,
			MaxSessions:           r.config.MCP.MaxSessions,
			MaxConcurrentRequests: r.config.MCP.MaxConcurrentRequests,
		},
		Logger: r.logger,
	})
	if err != nil {
		return fmt.Errorf("register MCP runtime: %w", err)
	}
	r.mcpRuntime = mcpRuntime
	return nil
}

func (r *Runtime) registerModuleRuntimeWithAuth(
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
	if err := moduleruntime.Register(
		moduleruntime.Registration{
			I18n:               r.i18n,
			MenuRegistry:       r.menuRegistry,
			PermissionRegistry: r.permissionRegistry,
			EventBus:           r.eventBus,
			Config:             r.config,
			Specs:              r.moduleRuntimeSpecs(),
		},
		r.server.Engine().Group("/api"),
		authService,
		authorizer,
	); err != nil {
		return fmt.Errorf("register module runtime routes: %w", err)
	}

	return nil
}

func (r *Runtime) registerAccessLogExplorerWithAuth(
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
	savedViews, err := r.resolveSavedViewService()
	if err != nil {
		return fmt.Errorf("resolve access-log saved-view service: %w", err)
	}
	if err := httpx.RegisterAccessLogExplorer(
		httpx.AccessLogExplorerRegistration{
			I18n:               r.i18n,
			MenuRegistry:       r.menuRegistry,
			PermissionRegistry: r.permissionRegistry,
			EventBus:           r.eventBus,
		},
		r.server.Engine().Group("/api"),
		r.server.AccessLogRepository(),
		authService,
		authorizer,
		savedViews,
	); err != nil {
		return fmt.Errorf("register access-log explorer: %w", err)
	}

	return nil
}

func (r *Runtime) registerAppLogExplorerWithAuth(
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
	if r.appLogRepository == nil {
		return nil
	}

	savedViews, err := r.resolveSavedViewService()
	if err != nil {
		return fmt.Errorf("resolve app-log saved-view service: %w", err)
	}
	if err := logger.RegisterAppLogExplorer(
		logger.AppLogExplorerRegistration{
			I18n:               r.i18n,
			MenuRegistry:       r.menuRegistry,
			PermissionRegistry: r.permissionRegistry,
			EventBus:           r.eventBus,
		},
		r.server.Engine().Group("/api"),
		r.appLogRepository,
		authService,
		authorizer,
		savedViews,
	); err != nil {
		return fmt.Errorf("register app-log explorer: %w", err)
	}

	return nil
}

func (r *Runtime) resolveSavedViewService() (moduleapi.SavedViewService, error) {
	return resolveRuntimeService[moduleapi.SavedViewService](r.services, (*moduleapi.SavedViewService)(nil), "saved-view service", "saved-view service")
}

func (r *Runtime) resolveLogExplorerAuth() (moduleapi.AuthService, moduleapi.Authorizer, error) {
	authService, err := r.resolveLogExplorerAuthService()
	if err != nil {
		return nil, nil, err
	}

	authorizer, err := r.resolveLogExplorerAuthorizer()
	if err != nil {
		return nil, nil, err
	}

	return authService, authorizer, nil
}

func (r *Runtime) resolveLogExplorerAuthService() (moduleapi.AuthService, error) {
	return resolveRuntimeService[moduleapi.AuthService](r.services, (*moduleapi.AuthService)(nil), "", "")
}

func (r *Runtime) resolveLogExplorerAuthorizer() (moduleapi.Authorizer, error) {
	return resolveRuntimeService[moduleapi.Authorizer](
		r.services,
		(*moduleapi.Authorizer)(nil),
		"access-log authorizer",
		"access-log authorizer",
	)
}

func (r *Runtime) registerRealtimeSubscriptionRoutes() error {
	if r == nil || r.server == nil {
		return errors.New("runtime server is unavailable")
	}

	authService, authorizer, err := r.resolveLogExplorerAuth()
	if err != nil {
		return fmt.Errorf("resolve realtime route auth: %w", err)
	}

	group := r.server.Engine().Group("/api")
	group.Use(httpx.RequestIDMiddleware())
	group.Use(httpx.RequirePermission(r.i18n, authService, authorizer, ""))

	return realtime.RegisterSubscriptionRoutes(group, realtime.HTTPRegistration{
		I18n:     r.i18n,
		Registry: r.realtimeTopicIssuers,
	})
}

func (r *Runtime) injectedRealtimeTicketService() (realtimeauth.Service, error) {
	if r == nil || r.services == nil {
		return nil, nil
	}

	return resolveRuntimeService[realtimeauth.Service](
		r.services,
		(*realtimeauth.Service)(nil),
		"realtime ticket service",
		"realtime ticket service",
	)
}

// 若对应标签为空，则直接返回底层错误信息。
func resolveRuntimeService[T any](
	services *container.Container,
	key any,
	resolveLabel string,
	unexpectedTypeLabel string,
) (T, error) {
	var zero T

	resolved, err := services.Resolve(key)
	if err != nil {
		if resolveLabel == "" {
			return zero, err
		}
		return zero, fmt.Errorf("resolve %s: %w", resolveLabel, err)
	}

	service, ok := resolved.(T)
	if !ok {
		if unexpectedTypeLabel == "" {
			return zero, fmt.Errorf("unexpected type %T", resolved)
		}
		return zero, fmt.Errorf("resolve %s: unexpected type %T", unexpectedTypeLabel, resolved)
	}

	return service, nil
}
