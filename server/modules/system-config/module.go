package systemconfig

import (
	"errors"
	"fmt"

	"graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	schedulercore "graft/server/internal/scheduler"
)

const moduleID = "system-config"

// Module 拥有系统配置用户覆盖值和 HTTP 管理接口的生命周期。
type Module struct {
	service *Service
}

// NewModule 创建系统配置模块实例；配置定义由各业务模块注册，覆盖值由本模块持久化。
func NewModule(service *Service) (*Module, error) {
	if service == nil {
		return nil, errors.New("system config service is unavailable")
	}
	return &Module{service: service}, nil
}

// Register 声明系统配置权限、菜单、消息和管理路由；此阶段不读取或写入覆盖值。
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil {
		return errors.New("system config module service is unavailable")
	}
	userService, err := requiredUserService(ctx.Services)
	if err != nil {
		return fmt.Errorf("resolve user service: %w", err)
	}
	m.service.setUserService(userService)
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	if err := registerSystemConfigPermissions(ctx.PermissionRegistry, moduleID); err != nil {
		return err
	}
	if err := registerSystemConfigMenu(ctx.MenuRegistry, moduleID); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*schedulercore.DefaultConfigResolver)(nil), func(_ container.Resolver) (any, error) {
		return m.service, nil
	}); err != nil {
		return fmt.Errorf("register system-config default resolver: %w", err)
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.SystemConfigResolver)(nil), func(_ container.Resolver) (any, error) {
		return m.service, nil
	}); err != nil {
		return fmt.Errorf("register system-config resolver: %w", err)
	}
	return registerSystemConfigRoutes(ctx, moduleID, m.service)
}

// requiredUserService 从提供的依赖注入解析器中解析并返回用户服务。
func requiredUserService(resolver container.Resolver) (moduleapi.UserService, error) {
	return module.ResolveService[moduleapi.UserService](resolver, (*moduleapi.UserService)(nil))
}

// Boot 不启动额外后台机制；共享快照缓存由 cachex 管理，服务只在请求路径按需装载。
func (m *Module) Boot(ctx *module.Context) error {
	if m == nil || m.service == nil {
		return errors.New("system config module service is unavailable")
	}
	if ctx == nil {
		return errors.New("system config module context is unavailable")
	}
	return nil
}

// Shutdown 当前没有系统配置模块自有的后台资源需要释放。
func (m *Module) Shutdown(_ *module.Context) error {
	return nil
}
