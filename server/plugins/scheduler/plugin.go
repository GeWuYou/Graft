package scheduler

import (
	"context"
	"fmt"

	"graft/server/internal/plugin"
	schedulercore "graft/server/internal/scheduler"
)

// Plugin 是当前 MVP 阶段的最小调度插件。
//
// 该插件只负责把 `cron registry` 中已声明的任务装配到运行时调度器，并在
// Register / Boot / Shutdown 阶段统一完成“声明收集、运行启动、收敛关闭”。
// 若 Register 阶段任务装配失败，插件不会进入可启动状态；Shutdown 会把
// 运行时停止错误上抛给调用方，便于宿主决定是否继续整体退出流程。
type Plugin struct {
	runtime schedulercore.Runtime
}

// NewPlugin 创建最小调度插件。
func NewPlugin() *Plugin {
	return &Plugin{}
}

// Name 返回插件稳定标识。
func (p *Plugin) Name() string {
	return "scheduler"
}

// Version 返回当前插件版本。
func (p *Plugin) Version() string {
	return "0.1.0"
}

// DependsOn 返回当前插件依赖列表。
func (p *Plugin) DependsOn() []string {
	return nil
}

// Register 根据当前 registry 快照装配全部任务声明。
func (p *Plugin) Register(ctx *plugin.Context) error {
	runtime := schedulercore.New(ctx.Logger)
	for _, job := range ctx.CronRegistry.Items() {
		if err := runtime.RegisterJob(job); err != nil {
			return fmt.Errorf("register scheduler job %s: %w", job.Name, err)
		}
	}

	p.runtime = runtime
	return nil
}

// Boot 启动已装配完成的最小调度器。
func (p *Plugin) Boot(ctx *plugin.Context) error {
	if p.runtime == nil {
		return nil
	}

	return p.runtime.Start()
}

// Shutdown 停止当前调度器并等待在途任务收敛。
func (p *Plugin) Shutdown(ctx *plugin.Context) error {
	if p.runtime == nil {
		return nil
	}

	stopCtx := context.Background()
	return p.runtime.Stop(stopCtx)
}
