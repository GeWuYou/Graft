// Package cronx 存放模块声明的定时任务元数据，供后续调度器装配使用。
package cronx

import (
	"context"
	"errors"
	"strings"
)

// Job 描述一个待注册的定时任务。
type Job struct {
	// Name 是任务的稳定标识，便于日志、观测与后续幂等装配。
	Name string
	// Schedule 保存面向调度器的 cron 表达式语义，当前阶段仅做声明透传。
	Schedule string
	// Module 标记任务来源模块，方便在启动失败或停机清理时定位责任边界。
	Module string
	// Run 是调度器实际调用的执行入口。
	//
	// 模块应在 Register 阶段显式提供该函数，而不是在 Boot 阶段隐式拼装
	// 或依赖全局单例回填执行体。
	Run func(ctx context.Context) error
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
	if strings.TrimSpace(j.Name) == "" {
		return errors.New("job name is required")
	}
	if strings.TrimSpace(j.Schedule) == "" {
		return errors.New("job schedule is required")
	}
	if j.Run == nil {
		return errors.New("job run function is required")
	}

	return nil
}
