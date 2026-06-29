package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

func effectiveConfigJSON(defaultConfig string, taskConfig string) (string, error) {
	return mergeConfigJSONObjects(defaultConfig, taskConfig)
}

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

func encodeJobRunResult(result cronx.JobRunResult) (string, string) {
	if result.Summary == "" {
		result.Summary = result.Stage
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return "{}", result.Summary
	}
	return string(encoded), result.Summary
}

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
	delete(r.running, key)
}

func removeKey(values []string, key string) []string {
	for index, value := range values {
		if value == key {
			return append(values[:index], values[index+1:]...)
		}
	}
	return values
}
