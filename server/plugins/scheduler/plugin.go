package scheduler

import (
	"fmt"

	"graft/server/internal/plugin"
	schedulercore "graft/server/internal/scheduler"
)

const (
	moduleID = "scheduler"
)

// Plugin 是当前 MVP 阶段的最小调度插件。
//
// 该插件只负责在所有插件完成 Register 后，把 `cron registry` 中已声明的
// 任务装配到运行时调度器，并在 Boot / Shutdown 阶段统一完成“运行启动、
// 收敛关闭”。若 Boot 阶段任务装配或启动失败，插件不会进入可运行状态；
// Shutdown 会把运行时停止错误上抛给调用方，便于宿主决定是否继续整体退出流程。
type Plugin struct {
	runtime schedulercore.Runtime
}

// NewPlugin 创建最小调度插件。
func NewPlugin() *Plugin {
	return &Plugin{}
}

// Register 保持为空，由 Boot 在所有插件完成声明后统一装配任务。
func (p *Plugin) Register(_ *plugin.Context) error {
	return nil
}

// Boot 在所有插件 Register 完成后装配并启动最小调度器。
func (p *Plugin) Boot(ctx *plugin.Context) error {
	if ctx == nil || ctx.CronRegistry == nil {
		return fmt.Errorf("scheduler boot context is required")
	}

	runtime := p.runtime
	if runtime == nil {
		runtime = schedulercore.New(ctx.Logger)
	}

	for _, job := range ctx.CronRegistry.Items() {
		if err := runtime.RegisterJob(job); err != nil {
			return fmt.Errorf("register scheduler job %s: %w", job.Name, err)
		}
	}

	if err := runtime.Start(ctx.LifecycleContext); err != nil {
		return fmt.Errorf("start scheduler runtime: %w", err)
	}

	p.runtime = runtime
	return nil
}

// Shutdown 停止当前调度器并等待在途任务收敛。
func (p *Plugin) Shutdown(ctx *plugin.Context) error {
	if p.runtime == nil {
		return nil
	}

	if ctx == nil || ctx.LifecycleContext == nil {
		return fmt.Errorf("scheduler shutdown lifecycle context is required")
	}

	return p.runtime.Stop(ctx.LifecycleContext)
}
