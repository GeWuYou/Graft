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

func validateEffectiveConfig(job JobDefinition, configJSON string) error {
	effectiveConfig, err := effectiveConfigJSON(job.DefaultConfig, configJSON)
	if err != nil {
		return err
	}
	return ValidateConfigJSON(job.ConfigSchema, effectiveConfig)
}

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

func jobActionResult(execution actionExecution, result cronx.JobRunResult, effectiveConfig string) JobActionResult {
	return JobActionResult{
		ActionKey:       strings.TrimSpace(execution.action.Key),
		TaskKey:         execution.definition.TaskKey,
		JobKey:          execution.definition.JobKey,
		Result:          result,
		EffectiveConfig: effectiveConfig,
	}
}

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

func hasRequiredJobDefinitionFields(definition JobDefinition) bool {
	return strings.TrimSpace(definition.JobKey) != "" &&
		strings.TrimSpace(definition.ModuleKey) != "" &&
		strings.TrimSpace(string(definition.Category)) != "" &&
		strings.TrimSpace(definition.Title) != "" &&
		strings.TrimSpace(definition.DefaultCron) != ""
}

func validateCronExpression(expression string) error {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(expression); err != nil {
		return fmt.Errorf("%w: invalid cron expression", ErrTaskValidation)
	}
	return nil
}

func isJSONObject(value string) bool {
	var decoded map[string]any
	return json.Unmarshal([]byte(strings.TrimSpace(value)), &decoded) == nil && decoded != nil
}

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
