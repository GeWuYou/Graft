// Package cronx 存放模块声明的定时任务元数据，供后续调度器装配使用。
package cronx

import (
	"context"
	"errors"
	"strings"
)

// Job 描述一个待注册的定时任务。
type Job struct {
	// Name 是历史内部标识；为空时沿用 Key。
	Name string
	// Key 是 Job Definition 的稳定标识。
	Key string
	// Owner 标记任务所有者，优先使用 module 名称；为空时沿用 Module。
	Owner string
	// Title 是 Job Definition 的默认展示标题。
	Title string
	// Description 是 Job Definition 的默认说明。
	Description string
	// DisplayMessageKey 是任务名称的稳定 i18n key。
	DisplayMessageKey string
	// DescriptionMessageKey 是任务说明的稳定 i18n key。
	DescriptionMessageKey string
	// ParamsSchema 是 Job Definition 的参数 schema JSON。
	ParamsSchema string
	// DefaultParams 是 Job Definition 的默认参数 JSON。
	DefaultParams string
	// Schedule 保存默认 cron 表达式语义，当前阶段仅做声明透传。
	Schedule string
	// DefaultEnabled 表示任务默认是否随运行时启用。MVP 不提供动态启停能力。
	DefaultEnabled bool
	// Module 标记任务来源模块，方便在启动失败或停机清理时定位责任边界。
	Module string
	// Handler 是调度器实际调用的执行入口，paramsJSON 来自 Scheduled Task。
	Handler func(ctx context.Context, paramsJSON string) error
	// Run 是历史无参数执行入口；新 Job 应优先使用 Handler。
	//
	// 模块应在 Register 阶段显式提供该函数，而不是在 Boot 阶段隐式拼装
	// 或依赖全局单例回填执行体。
	Run func(ctx context.Context) error
}

// RuntimeKey returns the visible stable task key.
func (j Job) RuntimeKey() string {
	if key := strings.TrimSpace(j.Key); key != "" {
		return key
	}
	return strings.TrimSpace(j.Name)
}

// RuntimeOwner returns the visible owner/module key.
func (j Job) RuntimeOwner() string {
	if owner := strings.TrimSpace(j.Owner); owner != "" {
		return owner
	}
	return strings.TrimSpace(j.Module)
}

// RuntimeTitle returns the default display title.
func (j Job) RuntimeTitle() string {
	if title := strings.TrimSpace(j.Title); title != "" {
		return title
	}
	return j.RuntimeKey()
}

// RuntimeDescription returns the default description.
func (j Job) RuntimeDescription() string {
	if description := strings.TrimSpace(j.Description); description != "" {
		return description
	}
	return j.RuntimeTitle()
}

// RuntimeDefaultParams returns stable JSON for default job parameters.
func (j Job) RuntimeDefaultParams() string {
	if params := strings.TrimSpace(j.DefaultParams); params != "" {
		return params
	}
	return "{}"
}

// RuntimeParamsSchema returns stable JSON for the job parameter schema.
func (j Job) RuntimeParamsSchema() string {
	if schema := strings.TrimSpace(j.ParamsSchema); schema != "" {
		return schema
	}
	return "{}"
}

// Invoke executes the registered handler with the given Scheduled Task parameters.
func (j Job) Invoke(ctx context.Context, paramsJSON string) error {
	if j.Handler != nil {
		return j.Handler(ctx, paramsJSON)
	}
	if j.Run != nil {
		return j.Run(ctx)
	}
	return errors.New("job handler is required")
}

// Registry 按注册顺序保存任务声明，供后续调度器接线阶段消费。
type Registry struct {
	items []Job
}

// NewRegistry 创建一个空的定时任务注册表。
func NewRegistry() *Registry {
	return &Registry{items: make([]Job, 0)}
}

// Register 按调用顺序向注册表追加一个定时任务声明。
//
// 当前仅收集元数据，不在这里解析 cron 表达式；真正的调度校验应由运行时装配层负责。
func (r *Registry) Register(item Job) {
	r.items = append(r.items, item)
}

// Items 返回当前已注册任务集合的副本，避免外部篡改内部切片。
func (r *Registry) Items() []Job {
	items := make([]Job, len(r.items))
	copy(items, r.items)
	return items
}

// Validate 校验任务声明是否满足当前最小调度契约。
func (j Job) Validate() error {
	if strings.TrimSpace(j.Name) == "" && strings.TrimSpace(j.Key) == "" {
		return errors.New("job name is required")
	}
	if strings.TrimSpace(j.Schedule) == "" {
		return errors.New("job schedule is required")
	}
	if j.Handler == nil && j.Run == nil {
		return errors.New("job handler is required")
	}

	return nil
}
