package savedview

import (
	"errors"

	containerdi "graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

// Module 注册通用保存视图服务；该模块只提供跨模块服务边界，不拥有菜单或 HTTP 路由。
type Module struct{ service *Service }

// NewModule 使用给定服务构造保存视图模块；依赖由编译期模块构建器提供。
func NewModule(service *Service) *Module { return &Module{service: service} }

// Register 仅注册不感知消费者筛选语义的保存视图服务，并在依赖缺失时拒绝装配。
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil || ctx == nil || ctx.Services == nil {
		return errors.New("saved view module service is unavailable")
	}
	return ctx.Services.RegisterSingleton((*moduleapi.SavedViewService)(nil), func(containerdi.Resolver) (any, error) {
		return m.service, nil
	})
}

// Boot 当前没有保存视图模块自有的后台工作。
func (*Module) Boot(*module.Context) error { return nil }

// Shutdown 当前没有需要由模块单独释放的持久资源；数据库连接池由平台负责关闭。
func (*Module) Shutdown(*module.Context) error { return nil }
