package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"graft/server/internal/cronx"
)

// jobDefinitionFromJob 从 cronx.Job 构建 JobDefinition，并填充运行时元数据、调度、配置和动作信息。
func jobDefinitionFromJob(job cronx.Job, now time.Time) JobDefinition {
	return JobDefinition{
		JobKey:         job.RuntimeKey(),
		ModuleKey:      job.RuntimeModuleKey(),
		Category:       job.RuntimeCategory(),
		TitleKey:       job.RuntimeTitleKey(),
		Title:          job.RuntimeTitle(),
		ShortTitleKey:  job.RuntimeShortTitleKey(),
		ShortTitle:     job.RuntimeShortTitle(),
		DescriptionKey: job.RuntimeDescriptionKey(),
		Description:    job.RuntimeDescription(),
		ConfigSchema:   job.RuntimeConfigSchema(),
		DefaultConfig:  job.RuntimeDefaultConfig(),
		DefaultCron:    strings.TrimSpace(job.Schedule),
		DefaultEnabled: job.DefaultEnabled,
		Enabled:        true,
		Actions:        jobActionsFromJob(job),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (r *CronRuntime) jobDefinitionFromJob(ctx context.Context, job cronx.Job) (JobDefinition, error) {
	definition := jobDefinitionFromJob(job, r.now())
	defaultConfig, err := r.resolveJobDefaultConfig(ctx, job)
	if err != nil {
		return JobDefinition{}, err
	}
	definition.DefaultConfig = defaultConfig
	return definition, nil
}

// builtinTaskDefinition 从作业生成内置任务定义，并填充运行时元数据、调度、默认配置和时间戳。
func builtinTaskDefinition(job cronx.Job, now time.Time) TaskDefinition {
	return TaskDefinition{
		TaskKey:        job.RuntimeKey(),
		JobKey:         job.RuntimeKey(),
		TitleKey:       job.RuntimeTitleKey(),
		Title:          job.RuntimeTitle(),
		DescriptionKey: job.RuntimeDescriptionKey(),
		Description:    job.RuntimeDescription(),
		CronExpression: strings.TrimSpace(job.Schedule),
		Enabled:        job.DefaultEnabled,
		Builtin:        true,
		ConfigJSON:     job.RuntimeDefaultConfig(),
		ConfigSource:   taskConfigSourceSystem,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// mutationToDefinition 将任务变更转换为任务定义，并校验定义与有效配置。
// 其中 `ConfigJSON` 为空白时会规范化为 `{}`。
// 返回校验后的任务定义；如果定义或配置无效，则返回错误。
func mutationToDefinition(command TaskMutation, job JobDefinition, now time.Time) (TaskDefinition, error) {
	configJSON := strings.TrimSpace(command.ConfigJSON)
	if configJSON == "" {
		configJSON = "{}"
	}
	definition := TaskDefinition{
		TaskKey:        strings.TrimSpace(command.TaskKey),
		JobKey:         strings.TrimSpace(command.JobKey),
		Title:          strings.TrimSpace(command.Title),
		Description:    strings.TrimSpace(command.Description),
		CronExpression: strings.TrimSpace(command.CronExpression),
		Enabled:        command.Enabled,
		Builtin:        false,
		ConfigJSON:     configJSON,
		ConfigSource:   taskConfigSourceUser,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := validateDefinition(definition); err != nil {
		return TaskDefinition{}, err
	}
	if err := validateEffectiveConfig(job, definition.ConfigJSON); err != nil {
		return TaskDefinition{}, err
	}
	return definition, nil
}

func (r *CronRuntime) validateTaskConfig(ctx context.Context, definition TaskDefinition) error {
	job, err := r.requireKnownJob(ctx, definition.JobKey)
	if err != nil {
		return err
	}
	taskConfig, err := r.taskConfigForEffective(definition)
	if err != nil {
		return err
	}
	return validateEffectiveConfig(job, taskConfig)
}

// validateEffectiveConfig 验证作业配置的有效 JSON 配置。
// 
// @param job 作业定义。
// @param configJSON 待验证的任务配置 JSON。
// @returns 配置有效时返回 nil；否则返回错误。
func validateEffectiveConfig(job JobDefinition, configJSON string) error {
	effectiveConfig, err := effectiveConfigJSON(job.DefaultConfig, configJSON)
	if err != nil {
		return err
	}
	return ValidateConfigJSON(job.ConfigSchema, effectiveConfig)
}

// actionEffectiveConfigJSON 合并作业、任务和请求的配置，生成动作的有效配置。
// 返回合并后的配置 JSON，或在配置合并失败时返回错误。
func actionEffectiveConfigJSON(job JobDefinition, taskConfig string, requestConfig string) (string, error) {
	return mergeConfigJSONObjects(job.DefaultConfig, taskConfig, requestConfig)
}

func (r *CronRuntime) resolveActionExecution(ctx context.Context, taskKey string, actionKey string) (actionExecution, error) {
	if r.tasks == nil {
		return actionExecution{}, fmt.Errorf("scheduler task repository is unavailable")
	}
	definition, err := r.tasks.GetTask(ctx, taskKey)
	if err != nil {
		return actionExecution{}, err
	}
	if err := validateDefinition(definition); err != nil {
		return actionExecution{}, err
	}
	jobDefinition, err := r.requireKnownJob(ctx, definition.JobKey)
	if err != nil {
		return actionExecution{}, err
	}
	action, ok := findJobAction(jobDefinition.Actions, actionKey)
	if !ok {
		return actionExecution{}, ErrJobActionNotFound
	}
	job, ok := r.findJob(definition.JobKey)
	if !ok {
		return actionExecution{}, ErrJobDefinitionNotFound
	}
	return actionExecution{
		definition:    definition,
		jobDefinition: jobDefinition,
		action:        action,
		job:           job,
	}, nil
}

// jobActionResult 构建动作执行结果记录。
func jobActionResult(execution actionExecution, result cronx.JobRunResult, effectiveConfig string) JobActionResult {
	return JobActionResult{
		ActionKey:       strings.TrimSpace(execution.action.Key),
		TaskKey:         execution.definition.TaskKey,
		JobKey:          execution.definition.JobKey,
		Result:          result,
		EffectiveConfig: effectiveConfig,
	}
}

// 并要求每个动作都具有有效的键和处理器。
func validateJob(job cronx.Job) error {
	if err := job.Validate(); err != nil {
		return err
	}
	if err := validateCronExpression(job.Schedule); err != nil {
		return err
	}
	if !isJSONObject(job.RuntimeDefaultConfig()) {
		return fmt.Errorf("%w: invalid default config", ErrTaskValidation)
	}
	if !isJSONObject(job.RuntimeConfigSchema()) {
		return fmt.Errorf("%w: invalid config schema", ErrTaskValidation)
	}
	for _, action := range job.Actions {
		if strings.TrimSpace(action.Key) == "" {
			return fmt.Errorf("%w: invalid job action key", ErrTaskValidation)
		}
		if action.Handler == nil {
			return fmt.Errorf("%w: job action handler is required", ErrTaskValidation)
		}
	}
	return nil
}

// validateDefinition 校验任务定义的必填字段、保留任务键、Cron 表达式和配置 JSON。
//
// 当任务键、作业键、Cron 表达式或标题为空，任务键属于保留键，Cron 表达式无效，
// 或配置不是 JSON 对象时，返回校验错误。
func validateDefinition(definition TaskDefinition) error {
	if strings.TrimSpace(definition.TaskKey) == "" ||
		strings.TrimSpace(definition.JobKey) == "" ||
		strings.TrimSpace(definition.CronExpression) == "" ||
		strings.TrimSpace(definition.Title) == "" {
		return ErrTaskValidation
	}
	if _, reserved := reservedTaskKeys[strings.TrimSpace(definition.TaskKey)]; reserved {
		return fmt.Errorf("%w: reserved task key", ErrTaskValidation)
	}
	if err := validateCronExpression(definition.CronExpression); err != nil {
		return err
	}
	if !isJSONObject(definition.ConfigJSON) {
		return fmt.Errorf("%w: invalid config json", ErrTaskValidation)
	}
	return nil
}

// validateJobDefinition 校验作业定义是否完整且其默认配置与配置模式有效。
//
// 它要求必要字段存在，DefaultCron 为有效的 cron 表达式，ConfigSchema 和 DefaultConfig 都是 JSON 对象，
// 并验证配置模式以及默认配置是否符合该模式。
//
// @return 返回校验过程中遇到的错误；校验通过时返回 nil。
func validateJobDefinition(definition JobDefinition) error {
	if !hasRequiredJobDefinitionFields(definition) {
		return ErrTaskValidation
	}
	if err := validateCronExpression(definition.DefaultCron); err != nil {
		return err
	}
	if !isJSONObject(definition.ConfigSchema) || !isJSONObject(definition.DefaultConfig) {
		return fmt.Errorf("%w: invalid job definition json", ErrTaskValidation)
	}
	if err := ValidateConfigSchema(definition.ConfigSchema); err != nil {
		return err
	}
	if err := ValidateConfigJSON(definition.ConfigSchema, definition.DefaultConfig); err != nil {
		return err
	}
	return nil
}

// hasRequiredJobDefinitionFields 检查作业定义是否包含必填字段。
// 当 JobKey、ModuleKey、Category、Title 和 DefaultCron 均在去除首尾空白后非空时，返回 true；否则返回 false。
func hasRequiredJobDefinitionFields(definition JobDefinition) bool {
	return strings.TrimSpace(definition.JobKey) != "" &&
		strings.TrimSpace(definition.ModuleKey) != "" &&
		strings.TrimSpace(string(definition.Category)) != "" &&
		strings.TrimSpace(definition.Title) != "" &&
		strings.TrimSpace(definition.DefaultCron) != ""
}

// validateCronExpression 验证 cron 表达式是否有效。
// @returns 表达式有效时返回 nil；否则返回带有 ErrTaskValidation 的错误。
func validateCronExpression(expression string) error {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(expression); err != nil {
		return fmt.Errorf("%w: invalid cron expression", ErrTaskValidation)
	}
	return nil
}

// isJSONObject 在 JSON 文本解析为非空对象时返回 true。
//
// @returns 如果字符串可解析为 JSON 对象且对象非 nil，则返回 true；否则返回 false。
func isJSONObject(value string) bool {
	var decoded map[string]any
	return json.Unmarshal([]byte(strings.TrimSpace(value)), &decoded) == nil && decoded != nil
}

// sameJSONObject 比较两个 JSON 字符串表示的对象是否相同。
// 
// @returns 两个输入都能解析为非空 JSON 对象且内容相等时返回 `true`，否则返回 `false`。
func sameJSONObject(left string, right string) bool {
	var leftDecoded map[string]any
	var rightDecoded map[string]any
	if json.Unmarshal([]byte(strings.TrimSpace(left)), &leftDecoded) != nil {
		return false
	}
	if json.Unmarshal([]byte(strings.TrimSpace(right)), &rightDecoded) != nil {
		return false
	}
	if leftDecoded == nil || rightDecoded == nil {
		return false
	}
	return reflect.DeepEqual(leftDecoded, rightDecoded)
}
