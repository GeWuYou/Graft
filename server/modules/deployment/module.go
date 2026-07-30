package deployment

import (
	"errors"
	"fmt"
	"os"

	containerdi "graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

// Module 发布唯一的 Deployment Runtime capability；它不拥有路由、持久化或后台任务。
type Module struct{ runtime moduleapi.DeploymentRuntime }

// NewModule 创建 Deployment Runtime 模块。
func NewModule() *Module { return &Module{} }

// Register 解析 Docker 原始事实并发布不可变的 Deployment Runtime capability。
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || ctx == nil || ctx.Services == nil {
		return errors.New("deployment module context is unavailable")
	}
	provider, err := module.ResolveService[moduleapi.DockerFactsProvider](ctx.Services, (*moduleapi.DockerFactsProvider)(nil))
	if err != nil {
		return fmt.Errorf("resolve Docker facts provider: %w", err)
	}
	m.runtime = NewRuntime(os.LookupEnv, provider)
	if err := ctx.Services.RegisterSingleton((*moduleapi.DeploymentRuntime)(nil), func(containerdi.Resolver) (any, error) { return m.runtime, nil }); err != nil {
		return fmt.Errorf("register deployment runtime: %w", err)
	}
	return nil
}

// Boot 不启动后台任务；Deployment Runtime 在消费时读取当前运行时事实。
func (*Module) Boot(*module.Context) error { return nil }

// Shutdown 不持有外部连接或可释放资源。
func (*Module) Shutdown(*module.Context) error { return nil }
