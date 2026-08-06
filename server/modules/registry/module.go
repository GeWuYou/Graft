package registry

import (
	"errors"

	containerdi "graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

// Module 拥有外部 Registry Connection 与 Artifact Repository 权威，不声明镜像仓库服务、路由或后台运行时。
type Module struct{ service *Service }

// NewModule 创建 Registry 基础设施模块。
func NewModule(service *Service) *Module { return &Module{service: service} }

// Register 只注册面向 Build 的窄目的地解析能力。
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil || ctx == nil || ctx.Services == nil {
		return errors.New("registry module service is unavailable")
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.RegistryDestinationResolver)(nil), func(containerdi.Resolver) (any, error) {
		return m.service, nil
	}); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.RegistryPublicationResolver)(nil), func(containerdi.Resolver) (any, error) {
		return m.service, nil
	}); err != nil {
		return err
	}
	return ctx.Services.RegisterSingleton((*moduleapi.RegistryArtifactCopyResolver)(nil), func(containerdi.Resolver) (any, error) {
		return m.service, nil
	})
}

// Boot 不启动 Registry 自有后台进程。
func (*Module) Boot(*module.Context) error { return nil }

// Shutdown 不需要释放 Registry 自有长生命周期资源。
func (*Module) Shutdown(*module.Context) error { return nil }
