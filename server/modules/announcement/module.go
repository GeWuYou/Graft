package announcement

import (
	"errors"
	"fmt"

	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	announcementcontract "graft/server/modules/announcement/contract"
)

// Module 拥有公告管理接口和当前用户公告读取接口的生命周期。
type Module struct {
	service *Service
}

// NewModule 创建公告模块实例；公告模块不自行启动后台运行资源。
func NewModule(service *Service) *Module {
	return &Module{service: service}
}

// Register 声明公告模块的消息、权限、菜单和路由；此阶段不启动后台行为。
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil {
		return errors.New("announcement module service is unavailable")
	}
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	if err := registerAnnouncementPermissions(ctx.PermissionRegistry, moduleID); err != nil {
		return err
	}
	if err := registerAnnouncementMenu(ctx.MenuRegistry, moduleID); err != nil {
		return err
	}
	if ctx.Router == nil {
		return nil
	}
	return m.registerRoutes(ctx)
}

func (m *Module) registerRoutes(ctx *module.Context) error {
	authService, err := resolveAuthService(ctx)
	if err != nil {
		return fmt.Errorf("resolve auth service: %w", err)
	}
	authorizer, err := resolveAuthorizer(ctx)
	if err != nil {
		return fmt.Errorf("resolve authorizer: %w", err)
	}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	return registerAnnouncementRoutes(ctx, m.service, announcementGuards{
		authenticated: httpx.RequirePermission(ctx.I18n, authService, nil, "", publisher),
		read:          httpx.RequirePermission(ctx.I18n, authService, authorizer, announcementcontract.AnnouncementReadPermission.String(), publisher),
		create:        httpx.RequirePermission(ctx.I18n, authService, authorizer, announcementcontract.AnnouncementCreatePermission.String(), publisher),
		update:        httpx.RequirePermission(ctx.I18n, authService, authorizer, announcementcontract.AnnouncementUpdatePermission.String(), publisher),
		publish:       httpx.RequirePermission(ctx.I18n, authService, authorizer, announcementcontract.AnnouncementPublishPermission.String(), publisher),
		delete:        httpx.RequirePermission(ctx.I18n, authService, authorizer, announcementcontract.AnnouncementDeletePermission.String(), publisher),
	})
}

// Boot 当前没有需要启动的模块运行时资源。
func (m *Module) Boot(_ *module.Context) error {
	return nil
}

// Shutdown 当前没有需要释放的模块运行时资源。
func (m *Module) Shutdown(_ *module.Context) error {
	return nil
}

func resolveAuthService(ctx *module.Context) (moduleapi.AuthService, error) {
	return module.ResolveService[moduleapi.AuthService](ctx.Services, (*moduleapi.AuthService)(nil))
}

func resolveAuthorizer(ctx *module.Context) (moduleapi.Authorizer, error) {
	return module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
}
