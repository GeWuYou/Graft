package backup

import (
	"errors"

	"graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

// Module 注册 backup owner 的跨模块 capability；菜单、HTTP 读取面和恢复操作留待对应阶段实现。
type Module struct{ service *Service }

// NewModule 使用已构造的备份服务创建模块。
func NewModule(service *Service) *Module { return &Module{service: service} }

// Register 仅发布窄 BackupService，避免让 Update 依赖 backup 的存储实现。
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil || ctx == nil || ctx.Services == nil {
		return errors.New("platform-backup module service is unavailable")
	}
	return ctx.Services.RegisterSingleton((*moduleapi.BackupService)(nil), func(container.Resolver) (any, error) {
		return m.service, nil
	})
}

// Boot 当前没有常驻备份执行器或清理任务。
func (*Module) Boot(*module.Context) error { return nil }

// Shutdown 当前没有模块自有的可关闭资源。
func (*Module) Shutdown(*module.Context) error { return nil }
