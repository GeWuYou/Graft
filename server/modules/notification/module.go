package notification

import (
	"context"
	"errors"
	"fmt"

	"graft/server/internal/container"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	notificationcontract "graft/server/modules/notification/contract"
)

// Module is the Notification Center backend module.
type Module struct {
	service   *Service
	publisher *Publisher
}

// NewModule creates a Notification Center module instance.
func NewModule(service *Service, publisher *Publisher) *Module {
	return &Module{service: service, publisher: publisher}
}

// Register declares notification permissions and the cross-module publisher capability.
func (m *Module) Register(ctx *module.Context) error {
	if err := m.validateRegisterContext(ctx); err != nil {
		return err
	}
	m.publisher.setLogger(ctx.Logger)
	if err := registerNotificationMetadata(ctx); err != nil {
		return err
	}
	if err := m.bindRBACAccessService(ctx); err != nil {
		return err
	}
	if ctx.Router != nil {
		if err := m.registerRoutes(ctx); err != nil {
			return err
		}
	}
	return ctx.Services.RegisterSingleton((*moduleapi.NotificationPublisher)(nil), func(_ container.Resolver) (any, error) {
		return m.publisher, nil
	})
}

func (m *Module) validateRegisterContext(ctx *module.Context) error {
	if m == nil || m.service == nil || m.publisher == nil {
		return errors.New("notification module dependencies are unavailable")
	}
	if ctx == nil || ctx.Services == nil {
		return errors.New("notification register context is required")
	}
	return nil
}

func (m *Module) bindRBACAccessService(ctx *module.Context) error {
	rbacAccess, err := resolveRBACAccessService(ctx)
	if err != nil {
		return fmt.Errorf("resolve rbac access service: %w", err)
	}
	if err := m.publisher.setRBACAccessService(rbacAccess); err != nil {
		return fmt.Errorf("bind rbac access service: %w", err)
	}
	return nil
}

func (m *Module) bindSystemConfigResolver(ctx *module.Context) error {
	resolver, err := resolveSystemConfigResolver(ctx)
	if err != nil {
		return fmt.Errorf("resolve system config resolver: %w", err)
	}
	if err := m.publisher.setConfigResolver(systemConfigNotificationResolver{resolver: resolver}); err != nil {
		return fmt.Errorf("bind system config resolver: %w", err)
	}
	return nil
}

func (m *Module) registerRoutes(ctx *module.Context) error {
	authService, err := resolveAuthService(ctx)
	if err != nil {
		return err
	}
	authorizer, err := resolveAuthorizer(ctx)
	if err != nil {
		return err
	}

	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	registerNotificationRoutes(ctx, m.service, notificationGuards{
		view: httpx.RequirePermission(
			ctx.I18n,
			authService,
			authorizer,
			notificationcontract.NotificationViewPermission.String(),
			publisher,
		),
		read: httpx.RequirePermission(
			ctx.I18n,
			authService,
			authorizer,
			notificationcontract.NotificationReadPermission.String(),
			publisher,
		),
	})
	return nil
}

// Boot resolves cross-module capabilities that are registered after all modules finish Register.
func (m *Module) Boot(ctx *module.Context) error {
	return m.bindSystemConfigResolver(ctx)
}

// Shutdown currently has no runtime resources to release.
func (m *Module) Shutdown(_ *module.Context) error {
	return nil
}

func resolveAuthService(ctx *module.Context) (moduleapi.AuthService, error) {
	resolved, err := ctx.Services.Resolve((*moduleapi.AuthService)(nil))
	if err != nil {
		return nil, err
	}
	authService, ok := resolved.(moduleapi.AuthService)
	if !ok || authService == nil {
		return nil, errors.New("notification auth service has unexpected type")
	}
	return authService, nil
}

func resolveAuthorizer(ctx *module.Context) (moduleapi.Authorizer, error) {
	resolved, err := ctx.Services.Resolve((*moduleapi.Authorizer)(nil))
	if err != nil {
		return nil, err
	}
	authorizer, ok := resolved.(moduleapi.Authorizer)
	if !ok || authorizer == nil {
		return nil, errors.New("notification authorizer has unexpected type")
	}
	return authorizer, nil
}

func resolveRBACAccessService(ctx *module.Context) (moduleapi.RBACAccessService, error) {
	resolved, err := ctx.Services.Resolve((*moduleapi.RBACAccessService)(nil))
	if err != nil {
		return nil, err
	}
	rbacAccess, ok := resolved.(moduleapi.RBACAccessService)
	if !ok || rbacAccess == nil {
		return nil, errors.New("notification rbac access service has unexpected type")
	}
	return rbacAccess, nil
}

func resolveSystemConfigResolver(ctx *module.Context) (moduleapi.SystemConfigResolver, error) {
	if ctx == nil || ctx.Services == nil {
		return nil, errors.New("notification services are required")
	}
	resolved, err := ctx.Services.Resolve((*moduleapi.SystemConfigResolver)(nil))
	if err != nil {
		return nil, err
	}
	resolver, ok := resolved.(moduleapi.SystemConfigResolver)
	if !ok || resolver == nil {
		return nil, fmt.Errorf("notification system config resolver has unexpected type %T", resolved)
	}
	return resolver, nil
}

type systemConfigNotificationResolver struct {
	resolver moduleapi.SystemConfigResolver
}

func (r systemConfigNotificationResolver) Boolean(ctx context.Context, key string, fallback bool) bool {
	if r.resolver == nil {
		return fallback
	}
	return r.resolver.IsBooleanConfigEnabled(ctx, key, fallback)
}
