package rbac

import (
	"context"
	"strings"

	"graft/server/internal/container"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
)

// Plugin 是 MVP 阶段最小可用的 RBAC 授权插件。
//
// 该插件当前只暴露请求级授权能力，保持实现收敛，避免在完整管理链路落地
// 前把角色与权限管理接口过早塞进运行时。
type Plugin struct{}

// NewPlugin 创建最小 RBAC 插件。
func NewPlugin() *Plugin {
	return &Plugin{}
}

// Name 返回插件稳定标识。
func (p *Plugin) Name() string {
	return "rbac"
}

// Version 返回当前插件版本。
func (p *Plugin) Version() string {
	return "0.1.0"
}

// DependsOn 声明当前最小授权插件依赖用户插件已完成认证主体解析。
func (p *Plugin) DependsOn() []string {
	return []string{"user"}
}

// Register 注册跨插件可复用的授权服务。
//
// Register 阶段只做稳定能力暴露，不执行任何后台行为或耗时初始化。
func (p *Plugin) Register(ctx *plugin.Context) error {
	return ctx.Services.RegisterSingleton((*pluginapi.Authorizer)(nil), func(resolver container.Resolver) (any, error) {
		return authorizer{rbac: ctx.Stores.RBAC()}, nil
	})
}

// Boot 当前没有额外运行时行为需要启动。
func (p *Plugin) Boot(ctx *plugin.Context) error {
	return nil
}

// Shutdown 当前没有额外资源需要释放。
func (p *Plugin) Shutdown(ctx *plugin.Context) error {
	return nil
}

type authorizer struct {
	rbac store.RBACRepository
}

// Authorize 基于稳定 RBAC 仓储判断请求主体是否拥有指定权限。
func (a authorizer) Authorize(ctx context.Context, request pluginapi.RequestAuthContext, permission string) error {
	if request.User == nil || request.User.ID == 0 {
		return pluginapi.ErrUnauthenticated
	}
	if strings.TrimSpace(permission) == "" {
		return nil
	}

	permissions, err := a.rbac.ListPermissionsByUserID(ctx, request.User.ID)
	if err != nil {
		return err
	}
	for _, granted := range permissions {
		if granted.Code == permission {
			return nil
		}
	}

	return pluginapi.ErrPermissionDenied
}

var _ pluginapi.Authorizer = authorizer{}
