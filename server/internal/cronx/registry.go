// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package cronx 存放模块声明的定时任务元数据，供后续调度器装配使用。
package cronx

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// JobCategory is the stable execution category for a Job Definition.
type JobCategory string

// Supported Job Definition categories.
const (
	JobCategoryRetention    JobCategory = "retention"
	JobCategorySync         JobCategory = "sync"
	JobCategoryMaintenance  JobCategory = "maintenance"
	JobCategoryNotification JobCategory = "notification"
	JobCategoryReport       JobCategory = "report"
	JobCategoryWorkflow     JobCategory = "workflow"
	JobCategoryCustom       JobCategory = "custom"
)

// Job 描述一个待注册的定时任务。
type Job struct {
	// Name 是历史内部标识；为空时沿用 Key。
	Name string
	// Key 是 Job Definition 的稳定标识。
	Key string
	// ModuleKey 标记声明该 Job Definition 的模块。
	ModuleKey string
	// Category 是 Job Definition 的稳定分类。
	Category JobCategory
	// Title 是 Job Definition 的默认展示标题。
	Title string
	// TitleKey 是 Job Definition 标题的稳定 i18n key。
	TitleKey string
	// ShortTitle 是列表等紧凑场景使用的默认短标题。
	ShortTitle string
	// ShortTitleKey 是 Job Definition 短标题的稳定 i18n key。
	ShortTitleKey string
	// Description 是 Job Definition 的默认说明。
	Description string
	// DescriptionKey 是 Job Definition 说明的稳定 i18n key。
	DescriptionKey string
	// ConfigSchema 是 scheduler 接受的 Job Definition JSON Schema 子集。
	ConfigSchema string
	// DefaultConfig 是与每个任务 config_json 合并的默认 JSON object。
	DefaultConfig string
	// DefaultConfigKey 指向拥有 DefaultConfig 真相的非敏感 object 型 system-config definition。
	DefaultConfigKey string
	// Actions are backend-defined one-shot operations available for this Job Definition.
	Actions []JobAction
	// Schedule 保存默认 cron 表达式语义，当前阶段仅做声明透传。
	Schedule string
	// DefaultEnabled 表示任务默认是否随运行时启用。MVP 不提供动态启停能力。
	DefaultEnabled bool
	// Module 标记任务来源模块，方便在启动失败或停机清理时定位责任边界。
	//
	// Deprecated: use ModuleKey for new Job Definition declarations.
	Module string
	// Handler is the scheduler execution entrypoint. configJSON is effective_config.
	Handler func(ctx context.Context, configJSON string) (JobRunResult, error)
	// Run is the no-config execution fallback for simple internal jobs.
	//
	// 模块应在 Register 阶段显式提供该函数，而不是在 Boot 阶段隐式拼装
	// 或依赖全局单例回填执行体。
	Run func(ctx context.Context) error
}

// JobAction describes one backend-defined operation available for a Job Definition.
type JobAction struct {
	Key            string
	TitleKey       string
	Title          string
	DescriptionKey string
	Description    string
	Handler        func(ctx context.Context, configJSON string) (JobRunResult, error)
}

// JobRunResult is the structured outcome a scheduler job should persist.
type JobRunResult struct {
	Summary          string         `json:"summary,omitempty"`
	Stage            string         `json:"stage,omitempty"`
	AffectedResource string         `json:"affected_resource,omitempty"`
	Metrics          map[string]any `json:"metrics,omitempty"`
	Details          map[string]any `json:"details,omitempty"`
	Warnings         []string       `json:"warnings,omitempty"`
}

// RuntimeKey returns the visible stable task key.
func (j Job) RuntimeKey() string {
	if key := strings.TrimSpace(j.Key); key != "" {
		return key
	}
	return strings.TrimSpace(j.Name)
}

// RuntimeModuleKey returns the Job Definition module key.
func (j Job) RuntimeModuleKey() string {
	if moduleKey := strings.TrimSpace(j.ModuleKey); moduleKey != "" {
		return moduleKey
	}
	return strings.TrimSpace(j.Module)
}

// RuntimeCategory returns the stable category for this Job Definition.
func (j Job) RuntimeCategory() JobCategory {
	if strings.TrimSpace(string(j.Category)) == "" {
		return JobCategoryCustom
	}
	return j.Category
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

// RuntimeTitleKey returns the stable i18n key for the job title.
func (j Job) RuntimeTitleKey() string {
	return strings.TrimSpace(j.TitleKey)
}

// RuntimeShortTitle returns the compact default display title.
func (j Job) RuntimeShortTitle() string {
	if title := strings.TrimSpace(j.ShortTitle); title != "" {
		return title
	}
	return j.RuntimeTitle()
}

// RuntimeShortTitleKey returns the stable i18n key for the compact title.
func (j Job) RuntimeShortTitleKey() string {
	return strings.TrimSpace(j.ShortTitleKey)
}

// RuntimeDescriptionKey returns the stable i18n key for the job description.
func (j Job) RuntimeDescriptionKey() string {
	return strings.TrimSpace(j.DescriptionKey)
}

// RuntimeDefaultConfig returns stable JSON for default job config.
func (j Job) RuntimeDefaultConfig() string {
	if config := strings.TrimSpace(j.DefaultConfig); config != "" {
		return config
	}
	return "{}"
}

// RuntimeConfigSchema returns stable JSON for the job config schema.
func (j Job) RuntimeConfigSchema() string {
	if schema := strings.TrimSpace(j.ConfigSchema); schema != "" {
		return schema
	}
	return "{}"
}

// Invoke executes the registered handler with the given effective Scheduled Task config.
func (j Job) Invoke(ctx context.Context, configJSON string) (JobRunResult, error) {
	if j.Handler != nil {
		return j.Handler(ctx, configJSON)
	}
	if j.Run != nil {
		err := j.Run(ctx)
		if err != nil {
			return JobRunResult{
				Summary: err.Error(),
				Stage:   "failed",
			}, err
		}
		return JobRunResult{
			Summary: "completed",
			Stage:   "completed",
		}, nil
	}
	return JobRunResult{}, errors.New("job handler is required")
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
	if strings.TrimSpace(j.RuntimeModuleKey()) == "" {
		return errors.New("job module key is required")
	}
	if strings.TrimSpace(j.Schedule) == "" {
		return errors.New("job schedule is required")
	}
	if !isValidJobCategory(j.Category) {
		return fmt.Errorf("job category %q is unsupported", j.Category)
	}
	if j.Handler == nil && j.Run == nil {
		return errors.New("job handler is required")
	}

	return nil
}

func isValidJobCategory(category JobCategory) bool {
	switch category {
	case "",
		JobCategoryRetention,
		JobCategorySync,
		JobCategoryMaintenance,
		JobCategoryNotification,
		JobCategoryReport,
		JobCategoryWorkflow,
		JobCategoryCustom:
		return true
	default:
		return false
	}
}
