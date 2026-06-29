package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"graft/server/internal/cronx"
)

func (r *CronRuntime) resolveJobDefaultConfig(ctx context.Context, job cronx.Job) (string, error) {
	key := strings.TrimSpace(job.DefaultConfigKey)
	if key == "" {
		return job.RuntimeDefaultConfig(), nil
	}
	resolver := r.defaultConfigResolver()
	if resolver == nil {
		return job.RuntimeDefaultConfig(), nil
	}
	config, err := resolver.ResolveDefaultConfig(ctx, key)
	if err != nil {
		return "", fmt.Errorf("resolve scheduler job default config %s: %w", key, err)
	}
	config = strings.TrimSpace(config)
	if config == "" {
		config = "{}"
	}
	if !isJSONObject(config) {
		return "", fmt.Errorf("%w: invalid default config", ErrTaskValidation)
	}
	return config, nil
}

func (r *CronRuntime) defaultConfigResolver() DefaultConfigResolver {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultConfigs
}

func (r *CronRuntime) taskConfigForEffective(definition TaskDefinition) (string, error) {
	configJSON := strings.TrimSpace(definition.ConfigJSON)
	if configJSON == "" {
		configJSON = "{}"
	}
	if !definition.Builtin {
		return configJSON, nil
	}
	if definition.ConfigSource != taskConfigSourceUser {
		return "{}", nil
	}
	return configJSON, nil
}

// effectiveConfigJSON 合并默认配置和任务配置为最终配置 JSON。
// @returns 合并后的 JSON 字符串；当任一输入无效或编码失败时返回错误。
func effectiveConfigJSON(defaultConfig string, taskConfig string) (string, error) {
	return mergeConfigJSONObjects(defaultConfig, taskConfig)
}

// mergeConfigJSONObjects 合并多个 JSON 对象字符串并返回结果。
// 空字符串按 `{}` 处理，后面的键会覆盖前面的键。
func mergeConfigJSONObjects(items ...string) (string, error) {
	merged := make(map[string]any)
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			trimmed = "{}"
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
			return "", fmt.Errorf("%w: invalid config json", ErrTaskValidation)
		}
		for key, value := range decoded {
			merged[key] = value
		}
	}
	encoded, err := json.Marshal(merged)
	if err != nil {
		return "", fmt.Errorf("%w: encode effective config", ErrTaskValidation)
	}
	return string(encoded), nil
}

// encodeJobRunResult 将作业运行结果编码为 JSON，并返回摘要文本。
//
// 如果 result.Summary 为空，则使用 result.Stage 作为摘要；若 JSON 编码失败，则返回 "{}" 和摘要。
func encodeJobRunResult(result cronx.JobRunResult) (string, string) {
	if result.Summary == "" {
		result.Summary = result.Stage
	}
	encoded, err := json.Marshal(result)
	if err == nil {
		return string(encoded), result.Summary
	}

	fallbackEncoded, fallbackErr := json.Marshal(struct {
		Summary  string   `json:"summary,omitempty"`
		Stage    string   `json:"stage,omitempty"`
		Warnings []string `json:"warnings,omitempty"`
	}{
		Summary: result.Summary,
		Stage:   result.Stage,
		Warnings: []string{
			fmt.Sprintf("scheduler job run result serialization failed: %v", err),
		},
	})
	if fallbackErr != nil {
		zap.L().Warn("scheduler job run result marshal failed",
			zap.String("stage", result.Stage),
			zap.String("summary", result.Summary),
			zap.Error(err),
			zap.NamedError("fallbackError", fallbackErr),
		)
		return `{"summary":"scheduler job result serialization failed","warnings":["scheduler job result serialization failed"]}`, result.Summary
	}

	zap.L().Warn("scheduler job run result marshal failed",
		zap.String("stage", result.Stage),
		zap.String("summary", result.Summary),
		zap.Error(err),
	)
	return string(fallbackEncoded), result.Summary
}

// jobDefinitionSnapshot 将 JobDefinition 转换为 JobDefinitionSnapshot。
func jobDefinitionSnapshot(definition JobDefinition) JobDefinitionSnapshot {
	return JobDefinitionSnapshot(definition)
}

func (r *CronRuntime) enrichJobDefinition(ctx context.Context, definition JobDefinition) (JobDefinition, error) {
	if job, ok := r.findJob(definition.JobKey); ok {
		definition.Actions = jobActionsFromJob(job)
		defaultConfig, err := r.resolveJobDefaultConfig(ctx, job)
		if err != nil {
			return JobDefinition{}, err
		}
		definition.DefaultConfig = defaultConfig
	}
	return definition, nil
}

// jobActionsFromJob 将 job.Actions 转换为 JobActionSnapshot 切片，并去除各字段两端空白。
// 返回按原顺序生成的动作快照列表。
func jobActionsFromJob(job cronx.Job) []JobActionSnapshot {
	actions := make([]JobActionSnapshot, 0, len(job.Actions))
	for _, action := range job.Actions {
		actions = append(actions, JobActionSnapshot{
			Key:            strings.TrimSpace(action.Key),
			TitleKey:       strings.TrimSpace(action.TitleKey),
			Title:          strings.TrimSpace(action.Title),
			DescriptionKey: strings.TrimSpace(action.DescriptionKey),
			Description:    strings.TrimSpace(action.Description),
		})
	}
	return actions
}

// findJobAction 按键名从动作列表中查找匹配的作业动作。
//
// @param actions 待查找的作业动作列表。
// @param key 需要匹配的动作键名。
// @returns 找到的作业动作及 `true`；如果键名为空或未找到匹配项，则返回零值和 `false`。
func findJobAction(actions []JobActionSnapshot, key string) (JobActionSnapshot, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return JobActionSnapshot{}, false
	}
	for _, action := range actions {
		if strings.TrimSpace(action.Key) == key {
			return action, true
		}
	}
	return JobActionSnapshot{}, false
}

// invokeJobAction 执行指定作业动作，未匹配时回退到作业默认执行逻辑。
// 当匹配到的动作未提供处理函数时，返回带有 ErrTaskValidation 的错误。
// @returns 任务运行结果或执行错误。
func invokeJobAction(ctx context.Context, job cronx.Job, actionKey string, configJSON string) (cronx.JobRunResult, error) {
	actionKey = strings.TrimSpace(actionKey)
	for _, action := range job.Actions {
		if strings.TrimSpace(action.Key) == actionKey {
			if action.Handler == nil {
				return cronx.JobRunResult{}, fmt.Errorf("%w: job action handler is required", ErrTaskValidation)
			}
			return action.Handler(ctx, configJSON)
		}
	}
	return job.Invoke(ctx, configJSON)
}

func (r *CronRuntime) markRunning(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.running[key]; exists {
		return ErrTaskAlreadyRunning
	}
	r.running[key] = struct{}{}
	return nil
}

func (r *CronRuntime) markFinished(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete /* task-key */ (r.running, key)
}

// removeKey 从切片中移除第一个等于 key 的元素。
func removeKey(values []string, key string) []string {
	for index, value := range values {
		if value == key {
			return append(values[:index], values[index+1:]...)
		}
	}
	return values
}
