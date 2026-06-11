// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"graft/server/internal/cronx"
)

type runRepositoryRecorder struct {
	created []TaskRun
	updated []TaskRun
	latest  map[string]TaskRun
	nextID  uint64
}

func newRunRepositoryRecorder() *runRepositoryRecorder {
	return &runRepositoryRecorder{latest: make(map[string]TaskRun), nextID: 1}
}

func (r *runRepositoryRecorder) CreateRun(_ context.Context, run TaskRun) (TaskRun, error) {
	run.ID = r.nextID
	r.nextID++
	r.created = append(r.created, run)
	r.latest[run.TaskKey] = run
	return run, nil
}

func (r *runRepositoryRecorder) FinishRun(
	_ context.Context,
	command RunFinishCommand,
) (TaskRun, error) {
	for _, run := range r.created {
		if run.ID != command.ID {
			continue
		}

		run.Status = command.Status
		run.Error = command.ErrorMessage
		run.Result = command.ResultSummary
		run.ResultJSON = command.ResultJSON
		run.FinishedAt = &command.FinishedAt
		duration := int64(0)
		run.DurationMS = &duration
		r.updated = append(r.updated, run)
		r.latest[run.TaskKey] = run
		return run, nil
	}

	return TaskRun{}, errors.New("run not found")
}

func (r *runRepositoryRecorder) ListRuns(_ context.Context, query RunListQuery) (RunListResult, error) {
	items := make([]TaskRun, 0)
	for _, run := range r.updated {
		if run.TaskKey == query.TaskKey {
			items = append(items, run)
		}
	}
	return RunListResult{Items: items, Total: len(items)}, nil
}

func (r *runRepositoryRecorder) LatestRunByTask(_ context.Context, taskKey string) (TaskRun, bool, error) {
	run, ok := r.latest[taskKey]
	return run, ok, nil
}

func (r *runRepositoryRecorder) GetRun(_ context.Context, id uint64) (TaskRun, error) {
	for _, run := range r.updated {
		if run.ID == id {
			return run, nil
		}
	}
	return TaskRun{}, ErrTaskNotFound
}

type taskRepositoryRecorder struct {
	tasks     map[string]TaskDefinition
	listCalls int
	listErr   error
}

type defaultConfigResolverRecorder struct {
	values map[string]string
	err    error
}

func (r defaultConfigResolverRecorder) ResolveDefaultConfig(_ context.Context, key string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	value, ok := r.values[key]
	if !ok {
		return "", errors.New("default config not found")
	}
	return value, nil
}

func newTaskRepositoryRecorder() *taskRepositoryRecorder {
	return &taskRepositoryRecorder{tasks: make(map[string]TaskDefinition)}
}

func (r *taskRepositoryRecorder) SeedBuiltinTasks(_ context.Context, tasks []TaskDefinition) error {
	for _, task := range tasks {
		existing, exists := r.tasks[task.TaskKey]
		if exists {
			task.CronExpression = existing.CronExpression
			task.Enabled = existing.Enabled
		}
		if task.ConfigSource == "" {
			task.ConfigSource = taskConfigSourceSystem
		}
		task.ID = uint64(len(r.tasks) + 1)
		r.tasks[task.TaskKey] = task
	}
	return nil
}

func (r *taskRepositoryRecorder) CreateTask(_ context.Context, task TaskDefinition) (TaskDefinition, error) {
	task.ID = uint64(len(r.tasks) + 1)
	r.tasks[task.TaskKey] = task
	return task, nil
}

func (r *taskRepositoryRecorder) UpdateTask(_ context.Context, key string, patch TaskMutation) (TaskDefinition, error) {
	task, ok := r.tasks[key]
	if !ok {
		return TaskDefinition{}, ErrTaskNotFound
	}
	if patch.CronExpression != "" {
		task.CronExpression = patch.CronExpression
	}
	if patch.EnabledSet {
		task.Enabled = patch.Enabled
	}
	if patch.ConfigJSON != "" {
		task.ConfigJSON = patch.ConfigJSON
		task.ConfigSource = taskConfigSourceUser
	}
	r.tasks[key] = task
	return task, nil
}

func (r *taskRepositoryRecorder) DeleteTask(_ context.Context, key string) error {
	if _, ok := r.tasks[key]; !ok {
		return ErrTaskNotFound
	}
	delete(r.tasks, key)
	return nil
}

func (r *taskRepositoryRecorder) SetTaskEnabled(_ context.Context, key string, enabled bool) (TaskDefinition, error) {
	task, ok := r.tasks[key]
	if !ok {
		return TaskDefinition{}, ErrTaskNotFound
	}
	task.Enabled = enabled
	r.tasks[key] = task
	return task, nil
}

func (r *taskRepositoryRecorder) ListTasks(_ context.Context, query TaskListQuery) ([]TaskDefinition, int, error) {
	r.listCalls++
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	items := make([]TaskDefinition, 0, len(r.tasks))
	for _, task := range r.tasks {
		items = append(items, task)
	}
	total := len(items)
	if query.Limit > 0 {
		start := min(max(query.Offset, 0), total)
		end := min(start+query.Limit, total)
		items = items[start:end]
	}
	return items, total, nil
}

func (r *taskRepositoryRecorder) GetTask(_ context.Context, key string) (TaskDefinition, error) {
	task, ok := r.tasks[key]
	if !ok {
		return TaskDefinition{}, ErrTaskNotFound
	}
	return task, nil
}

func seedRuntimeJob(t *testing.T, runtime *CronRuntime, job cronx.Job) {
	t.Helper()
	if job.Module == "" && job.Owner == "" {
		job.Module = "test"
	}
	job.DefaultEnabled = true
	if err := runtime.SeedBuiltinJobs(context.Background(), []cronx.Job{job}); err != nil {
		t.Fatalf("seed job: %v", err)
	}
}

// TestRegisterJobRejectsInvalidDeclarations 验证调度器会拒绝缺失执行入口或非法表达式的任务声明。
func TestRegisterJobRejectsInvalidDeclarations(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())

	if err := runtime.RegisterJob(cronx.Job{Name: "", Schedule: "* * * * * *", Run: func(context.Context) error { return nil }}); err == nil {
		t.Fatal("expected empty job name to fail")
	}
	if err := runtime.RegisterJob(cronx.Job{Name: "cleanup", Schedule: "", Run: func(context.Context) error { return nil }}); err == nil {
		t.Fatal("expected empty schedule to fail")
	}
	if err := runtime.RegisterJob(cronx.Job{Name: "cleanup", Schedule: "* * * * * *"}); err == nil {
		t.Fatal("expected missing run function to fail")
	}
	if err := runtime.RegisterJob(cronx.Job{
		Name:     "cleanup",
		Schedule: "* * * * * *",
		Actions: []cronx.JobAction{
			{Key: "dryRun"},
		},
		Run: func(context.Context) error { return nil },
	}); err == nil {
		t.Fatal("expected missing action handler to fail")
	}
}

// TestRegisterJobRejectsDuplicateName 验证重复任务名会在注册阶段显式失败。
func TestRegisterJobRejectsDuplicateName(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	job := cronx.Job{
		Name:     "cleanup",
		Schedule: "*/1 * * * * *",
		Run:      func(context.Context) error { return nil },
	}

	if err := runtime.RegisterJob(job); err != nil {
		t.Fatalf("register first job: %v", err)
	}
	if err := runtime.RegisterJob(job); err == nil {
		t.Fatal("expected duplicate registration to fail")
	}
}

// TestValidateDefinitionRejectsReservedRouteKeys 验证任务 key 不会占用静态 API 路由片段。
func TestValidateDefinitionRejectsReservedRouteKeys(t *testing.T) {
	for _, key := range []string{"jobs", "runs"} {
		err := validateDefinition(TaskDefinition{
			TaskKey:        key,
			JobKey:         "scheduler.cleanup",
			ModuleKey:      "scheduler",
			Title:          "Cleanup",
			CronExpression: "*/5 * * * * *",
			ConfigJSON:     "{}",
		})
		if !errors.Is(err, ErrTaskValidation) {
			t.Fatalf("expected reserved key %q to fail validation, got %v", key, err)
		}
	}
}

// TestListTasksReturnsRuntimeJobSnapshots 验证运行时快照会保留任务声明中的展示与 owner 元数据。
func TestListTasksReturnsRuntimeJobSnapshots(t *testing.T) {
	repo := newRunRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)
	runtime.SetTaskRepository(newTaskRepositoryRecorder())

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:                  "audit.audit-log-retention-cleanup",
		Key:                   "audit.audit-log-retention-cleanup",
		Owner:                 "audit",
		DisplayMessageKey:     "scheduledTask.auditLogRetention.title",
		DescriptionMessageKey: "scheduledTask.auditLogRetention.description",
		Schedule:              "*/1 * * * * *",
		DefaultEnabled:        true,
		Module:                "audit",
		Run:                   func(context.Context) error { return nil },
	})

	result, err := runtime.ListTasks(context.Background(), TaskListQuery{})
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one task, got %#v", result)
	}
	item := result.Items[0]
	if item.Key != "audit.audit-log-retention-cleanup" ||
		item.JobKey != "audit.audit-log-retention-cleanup" ||
		item.ModuleKey != "audit" {
		t.Fatalf("unexpected task snapshot: %#v", item)
	}
	if item.DisplayMessageKey != "scheduledTask.auditLogRetention.title" || item.DescriptionMessageKey == "" {
		t.Fatalf("expected display metadata, got %#v", item)
	}
	if !item.Enabled {
		t.Fatal("expected runtime job to be default-enabled")
	}
}

func TestCreateTaskRejectsUnknownConfigField(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	taskRepo := newTaskRepositoryRecorder()
	runtime.SetTaskRepository(taskRepo)

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:         "schema-job",
		Schedule:     "*/1 * * * * *",
		ConfigSchema: `{"type":"object","properties":{"batchSize":{"type":"integer","minimum":1}},"additionalProperties":false}`,
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	})

	_, err := runtime.CreateTask(context.Background(), TaskMutation{
		TaskKey:        "custom",
		JobKey:         "schema-job",
		Title:          "Custom",
		CronExpression: "*/5 * * * * *",
		Enabled:        true,
		EnabledSet:     true,
		ConfigJSON:     `{"unknown":true}`,
	})
	var configErr ConfigValidationError
	if !errors.As(err, &configErr) || configErr.Field != "config_json.unknown" {
		t.Fatalf("expected field-addressable config error, got %v", err)
	}
	if _, ok := taskRepo.tasks["custom"]; ok {
		t.Fatal("expected invalid config to be rejected before task persistence")
	}
}

func TestCreateTaskReturnsEffectiveConfig(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	runtime.SetTaskRepository(newTaskRepositoryRecorder())

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:          "schema-job",
		Schedule:      "*/1 * * * * *",
		DefaultConfig: `{"batchSize":100,"retentionDays":30}`,
		ConfigSchema:  `{"type":"object","properties":{"batchSize":{"type":"integer","minimum":1},"retentionDays":{"type":"integer","minimum":1}},"additionalProperties":false}`,
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	})

	task, err := runtime.CreateTask(context.Background(), TaskMutation{
		TaskKey:        "custom",
		JobKey:         "schema-job",
		Title:          "Custom",
		CronExpression: "*/5 * * * * *",
		Enabled:        true,
		EnabledSet:     true,
		ConfigJSON:     `{"batchSize":25}`,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	effective := decodeRuntimeJSONObject(t, task.EffectiveConfig)
	if effective["batchSize"] != float64(25) || effective["retentionDays"] != float64(30) {
		t.Fatalf("unexpected effective config: %s", task.EffectiveConfig)
	}
}

func TestSeedBuiltinJobsUsesEffectiveDefaultConfig(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	taskRepo := newTaskRepositoryRecorder()
	runtime.SetTaskRepository(taskRepo)
	runtime.SetDefaultConfigResolver(defaultConfigResolverRecorder{
		values: map[string]string{
			"logger.app-log-retention-cleanup": `{"retentionDays":45,"batchSize":2000}`,
		},
	})

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:             "logger.app-log-retention-cleanup",
		Schedule:         "*/1 * * * * *",
		DefaultConfig:    `{"retentionDays":30,"batchSize":1000}`,
		DefaultConfigKey: "logger.app-log-retention-cleanup",
		ConfigSchema:     `{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":3650},"batchSize":{"type":"integer","minimum":1,"maximum":10000}},"additionalProperties":false}`,
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	})

	definitions, err := runtime.ListJobDefinitions(context.Background())
	if err != nil {
		t.Fatalf("list job definitions: %v", err)
	}
	if len(definitions) != 1 || definitions[0].DefaultConfig != `{"retentionDays":45,"batchSize":2000}` {
		t.Fatalf("expected effective job default config, got %#v", definitions)
	}
	if taskRepo.tasks["logger.app-log-retention-cleanup"].ConfigSource != taskConfigSourceSystem {
		t.Fatalf("expected builtin task to keep system config source, got %#v", taskRepo.tasks["logger.app-log-retention-cleanup"])
	}
	task, err := runtime.GetTask(context.Background(), "logger.app-log-retention-cleanup")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	effective := decodeRuntimeJSONObject(t, task.EffectiveConfig)
	if effective["retentionDays"] != float64(45) || effective["batchSize"] != float64(2000) {
		t.Fatalf("unexpected effective config: %s", task.EffectiveConfig)
	}
}

func TestSeedBuiltinJobsKeepsSystemConfigSourceForDefaultSnapshots(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	taskRepo := newTaskRepositoryRecorder()
	runtime.SetTaskRepository(taskRepo)
	runtime.SetDefaultConfigResolver(defaultConfigResolverRecorder{
		values: map[string]string{
			"audit.audit-log-retention-cleanup": `{"retentionDays":60,"batchSize":3000}`,
		},
	})

	job := cronx.Job{
		Name:             "audit.audit-log-retention-cleanup",
		Schedule:         "*/1 * * * * *",
		DefaultConfig:    `{"retentionDays":30,"batchSize":1000}`,
		DefaultConfigKey: "audit.audit-log-retention-cleanup",
		ConfigSchema:     `{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":3650},"batchSize":{"type":"integer","minimum":1,"maximum":10000}},"additionalProperties":false}`,
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	}
	taskRepo.tasks["audit.audit-log-retention-cleanup"] = TaskDefinition{
		TaskKey:        "audit.audit-log-retention-cleanup",
		JobKey:         "audit.audit-log-retention-cleanup",
		ModuleKey:      "audit",
		Title:          "Audit cleanup",
		CronExpression: "*/1 * * * * *",
		Enabled:        true,
		Builtin:        true,
		ConfigJSON:     `{"retentionDays":30,"batchSize":1000}`,
		ConfigSource:   taskConfigSourceSystem,
	}

	if err := runtime.SeedBuiltinJobs(context.Background(), []cronx.Job{job}); err != nil {
		t.Fatalf("seed builtin job: %v", err)
	}
	if taskRepo.tasks["audit.audit-log-retention-cleanup"].ConfigSource != taskConfigSourceSystem {
		t.Fatalf("expected system config source to survive reseed, got %#v", taskRepo.tasks["audit.audit-log-retention-cleanup"])
	}
}

func TestSeedBuiltinJobsReclassifiesHistoricalBuiltinOverride(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	taskRepo := newTaskRepositoryRecorder()
	runtime.SetTaskRepository(taskRepo)

	job := cronx.Job{
		Name:          "builtin-schema-job",
		Schedule:      "*/1 * * * * *",
		DefaultConfig: `{"retentionDays":30,"batchSize":1000}`,
		ConfigSchema:  `{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":3650},"batchSize":{"type":"integer","minimum":1,"maximum":10000}},"additionalProperties":false}`,
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	}
	taskRepo.tasks["builtin-schema-job"] = TaskDefinition{
		TaskKey:        "builtin-schema-job",
		JobKey:         "builtin-schema-job",
		ModuleKey:      "test",
		Title:          "builtin-schema-job",
		CronExpression: "*/1 * * * * *",
		Enabled:        true,
		Builtin:        true,
		ConfigJSON:     `{"retentionDays":90,"batchSize":500}`,
		ConfigSource:   taskConfigSourceSystem,
	}

	if err := runtime.SeedBuiltinJobs(context.Background(), []cronx.Job{job}); err != nil {
		t.Fatalf("reseed builtin job: %v", err)
	}
	task := taskRepo.tasks["builtin-schema-job"]
	config := decodeRuntimeJSONObject(t, task.ConfigJSON)
	if config["retentionDays"] != float64(90) || config["batchSize"] != float64(500) {
		t.Fatalf("expected historical override to survive reseed, got %s", task.ConfigJSON)
	}
	if task.ConfigSource != taskConfigSourceUser {
		t.Fatalf("expected historical override to be reclassified as user source, got %#v", task)
	}
}

func TestBuiltinExplicitTaskConfigTakesPrecedenceOverEffectiveDefault(t *testing.T) {
	repo := newRunRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)
	taskRepo := newTaskRepositoryRecorder()
	runtime.SetTaskRepository(taskRepo)
	runtime.SetDefaultConfigResolver(defaultConfigResolverRecorder{
		values: map[string]string{
			"httpx.access-log-retention-cleanup": `{"retentionDays":45,"batchSize":2000}`,
		},
	})

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:             "httpx.access-log-retention-cleanup",
		Schedule:         "*/1 * * * * *",
		DefaultConfig:    `{"retentionDays":30,"batchSize":1000}`,
		DefaultConfigKey: "httpx.access-log-retention-cleanup",
		ConfigSchema:     `{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":3650},"batchSize":{"type":"integer","minimum":1,"maximum":10000}},"additionalProperties":false}`,
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	})
	if _, err := runtime.UpdateTask(context.Background(), "httpx.access-log-retention-cleanup", TaskMutation{
		ConfigJSON: `{"retentionDays":90}`,
	}); err != nil {
		t.Fatalf("update builtin task config: %v", err)
	}

	task, err := runtime.GetTask(context.Background(), "httpx.access-log-retention-cleanup")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	effective := decodeRuntimeJSONObject(t, task.EffectiveConfig)
	if effective["retentionDays"] != float64(90) || effective["batchSize"] != float64(2000) {
		t.Fatalf("expected explicit task config to override effective default, got %s", task.EffectiveConfig)
	}
}

func TestUpdateTaskRejectsUnknownConfigBeforePersistence(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	taskRepo := newTaskRepositoryRecorder()
	runtime.SetTaskRepository(taskRepo)

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:         "schema-job",
		Schedule:     "*/1 * * * * *",
		ConfigSchema: `{"type":"object","properties":{"batchSize":{"type":"integer","minimum":1}},"additionalProperties":false}`,
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	})
	if _, err := runtime.CreateTask(context.Background(), TaskMutation{
		TaskKey:        "custom",
		JobKey:         "schema-job",
		Title:          "Custom",
		CronExpression: "*/5 * * * * *",
		Enabled:        true,
		EnabledSet:     true,
		ConfigJSON:     `{}`,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	_, err := runtime.UpdateTask(context.Background(), "custom", TaskMutation{
		ConfigJSON: `{"unknown":true}`,
	})
	var configErr ConfigValidationError
	if !errors.As(err, &configErr) || configErr.Field != "config_json.unknown" {
		t.Fatalf("expected field-addressable config error, got %v", err)
	}
	if taskRepo.tasks["custom"].ConfigJSON != "{}" {
		t.Fatalf("expected invalid config update not to persist, got %s", taskRepo.tasks["custom"].ConfigJSON)
	}
}

func TestUpdateBuiltinTaskAllowsSchemaBackedConfig(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	taskRepo := newTaskRepositoryRecorder()
	runtime.SetTaskRepository(taskRepo)

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:          "builtin-schema-job",
		Schedule:      "*/1 * * * * *",
		DefaultConfig: `{"retentionDays":30,"batchSize":1000}`,
		ConfigSchema:  `{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":3650},"batchSize":{"type":"integer","minimum":1,"maximum":10000}},"additionalProperties":false}`,
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	})

	updated, err := runtime.UpdateTask(context.Background(), "builtin-schema-job", TaskMutation{
		CronExpression: "*/10 * * * * *",
		Enabled:        false,
		EnabledSet:     true,
		ConfigJSON:     `{"retentionDays":90,"batchSize":500}`,
	})
	if err != nil {
		t.Fatalf("update builtin task config: %v", err)
	}
	if updated.Schedule != "*/10 * * * * *" || updated.Enabled {
		t.Fatalf("expected builtin cron/enabled update, got %#v", updated)
	}
	effective := decodeRuntimeJSONObject(t, updated.EffectiveConfig)
	if effective["retentionDays"] != float64(90) || effective["batchSize"] != float64(500) {
		t.Fatalf("unexpected effective config: %s", updated.EffectiveConfig)
	}
	if taskRepo.tasks["builtin-schema-job"].ConfigJSON != `{"retentionDays":90,"batchSize":500}` {
		t.Fatalf("expected builtin config to persist, got %s", taskRepo.tasks["builtin-schema-job"].ConfigJSON)
	}
	if taskRepo.tasks["builtin-schema-job"].ConfigSource != taskConfigSourceUser {
		t.Fatalf("expected explicit builtin config to mark user source, got %#v", taskRepo.tasks["builtin-schema-job"])
	}
}

func TestUpdateBuiltinTaskRejectsInvalidConfigBeforePersistence(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	taskRepo := newTaskRepositoryRecorder()
	runtime.SetTaskRepository(taskRepo)

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:          "builtin-schema-job",
		Schedule:      "*/1 * * * * *",
		DefaultConfig: `{"retentionDays":30,"batchSize":1000}`,
		ConfigSchema:  `{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":3650},"batchSize":{"type":"integer","minimum":1,"maximum":10000}},"additionalProperties":false}`,
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	})

	_, err := runtime.UpdateTask(context.Background(), "builtin-schema-job", TaskMutation{
		ConfigJSON: `{"retentionDays":0,"batchSize":500}`,
	})
	var configErr ConfigValidationError
	if !errors.As(err, &configErr) || configErr.Field != "config_json.retentionDays" {
		t.Fatalf("expected retentionDays config error, got %v", err)
	}
	if taskRepo.tasks["builtin-schema-job"].ConfigSource != taskConfigSourceSystem {
		t.Fatalf("expected invalid builtin config update not to persist, got %s", taskRepo.tasks["builtin-schema-job"].ConfigJSON)
	}
}

func TestSeedBuiltinJobsPreservesSchemaBackedConfigAndDropsStaleFields(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	taskRepo := newTaskRepositoryRecorder()
	runtime.SetTaskRepository(taskRepo)

	job := cronx.Job{
		Name:          "builtin-schema-job",
		Schedule:      "*/1 * * * * *",
		DefaultConfig: `{"retentionDays":30,"batchSize":1000}`,
		ConfigSchema:  `{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":3650},"batchSize":{"type":"integer","minimum":1,"maximum":10000}},"additionalProperties":false}`,
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	}
	seedRuntimeJob(t, runtime, job)
	taskRepo.tasks["builtin-schema-job"] = TaskDefinition{
		TaskKey:        "builtin-schema-job",
		JobKey:         "builtin-schema-job",
		ModuleKey:      "test",
		Title:          "builtin-schema-job",
		CronExpression: "*/1 * * * * *",
		Enabled:        true,
		Builtin:        true,
		ConfigJSON:     `{"retentionDays":90,"batchSize":500,"dryRun":true}`,
		ConfigSource:   taskConfigSourceUser,
	}

	if err := runtime.SeedBuiltinJobs(context.Background(), []cronx.Job{job}); err != nil {
		t.Fatalf("reseed builtin job: %v", err)
	}
	config := decodeRuntimeJSONObject(t, taskRepo.tasks["builtin-schema-job"].ConfigJSON)
	if config["retentionDays"] != float64(90) || config["batchSize"] != float64(500) {
		t.Fatalf("expected schema-backed config to survive reseed, got %s", taskRepo.tasks["builtin-schema-job"].ConfigJSON)
	}
	if _, ok := config["dryRun"]; ok {
		t.Fatalf("expected stale dryRun config to be dropped, got %s", taskRepo.tasks["builtin-schema-job"].ConfigJSON)
	}
}

// TestRunOncePersistsManualRunHistory 验证手动运行会写入运行历史并完成成功状态。
func TestRunOncePersistsManualRunHistory(t *testing.T) {
	repo := newRunRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	triggered := false

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:          "manual",
		Schedule:      "*/1 * * * * *",
		DefaultConfig: `{"batchSize":10}`,
		ConfigSchema:  `{"type":"object","properties":{"batchSize":{"type":"integer","minimum":1}},"additionalProperties":false}`,
		Handler: func(_ context.Context, configJSON string) (cronx.JobRunResult, error) {
			triggered = true
			effective := decodeRuntimeJSONObject(t, configJSON)
			if effective["batchSize"] != float64(10) {
				t.Fatalf("unexpected handler config: %s", configJSON)
			}
			return cronx.JobRunResult{
				Summary:          "deleted 3 audit logs",
				Stage:            "cleanup",
				AffectedResource: "audit_logs",
				Metrics:          map[string]any{"deletedCount": 3},
			}, nil
		},
	})

	run, err := runtime.RunOnce(context.Background(), "manual")
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !triggered {
		t.Fatal("expected manual run to execute job")
	}
	if run.TriggerType != TriggerTypeManual || run.Status != RunStatusSuccess {
		t.Fatalf("expected successful manual run, got %#v", run)
	}
	if len(repo.created) != 1 || len(repo.updated) != 1 {
		t.Fatalf("expected one persisted run lifecycle, got created=%d updated=%d", len(repo.created), len(repo.updated))
	}
	result := decodeRuntimeJSONObject(t, repo.updated[0].ResultJSON)
	if result["summary"] != "deleted 3 audit logs" ||
		result["stage"] != "cleanup" ||
		result["affected_resource"] != "audit_logs" {
		t.Fatalf("unexpected result json: %s", repo.updated[0].ResultJSON)
	}
	metrics, ok := result["metrics"].(map[string]any)
	if !ok || metrics["deletedCount"] != float64(3) {
		t.Fatalf("unexpected result metrics: %s", repo.updated[0].ResultJSON)
	}
}

func TestRunOnceNotifiesAfterPersistedFailure(t *testing.T) {
	repo := newRunRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	notifier := newRunFailureNotifierRecorder()
	runtime.SetRunFailureNotifier(notifier)

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:     "manual-failure",
		Schedule: "*/1 * * * * *",
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "failed"}, errors.New("boom")
		},
	})

	run, err := runtime.RunOnce(context.Background(), "manual-failure")
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected handler error, got %v", err)
	}
	if run.Status != RunStatusFailed {
		t.Fatalf("expected failed run, got %#v", run)
	}
	if len(repo.updated) != 1 || repo.updated[0].Status != RunStatusFailed {
		t.Fatalf("expected persisted failed run before notify, got %#v", repo.updated)
	}
	notified := notifier.wait(t)
	if notified.ID != repo.updated[0].ID {
		t.Fatalf("expected failed-run notification after persistence, got %#v", notified)
	}
}

func TestRunOnceWithTriggerNotifiesManualSuccess(t *testing.T) {
	repo := newRunRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	notifier := newRunSuccessNotifierRecorder()
	runtime.SetRunSuccessNotifier(notifier)

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:     "manual-success",
		Schedule: "*/1 * * * * *",
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	})

	run, err := runtime.RunOnceWithTrigger(context.Background(), "manual-success", RunTrigger{
		Type:          TriggerTypeManual,
		TriggerUserID: 42,
	})
	if err != nil {
		t.Fatalf("run once with trigger: %v", err)
	}
	if run.Status != RunStatusSuccess {
		t.Fatalf("expected successful run, got %#v", run)
	}

	notifiedRun, trigger := notifier.wait(t)
	if notifiedRun.ID != repo.updated[0].ID {
		t.Fatalf("expected successful-run notification after persistence, got %#v", notifiedRun)
	}
	if trigger.Type != TriggerTypeManual || trigger.TriggerUserID != 42 {
		t.Fatalf("expected manual trigger user to be preserved, got %#v", trigger)
	}
}

func TestCronSuccessDoesNotNotifyRunSuccess(t *testing.T) {
	repo := newRunRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	notifier := newRunSuccessNotifierRecorder()
	runtime.SetRunSuccessNotifier(notifier)

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:     "cron-success",
		Schedule: "*/1 * * * * *",
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	})

	run, err := runtime.runDefinition(context.Background(), TaskDefinition{
		TaskKey:        "cron-success",
		JobKey:         "cron-success",
		ModuleKey:      "test",
		Title:          "cron-success",
		CronExpression: "*/1 * * * * *",
		Enabled:        true,
		ConfigJSON:     "{}",
	}, RunTrigger{Type: TriggerTypeCron})
	if err != nil {
		t.Fatalf("run cron definition: %v", err)
	}
	if run.TriggerType != TriggerTypeCron || run.Status != RunStatusSuccess {
		t.Fatalf("expected successful cron run, got %#v", run)
	}
	notifier.assertNoNotification(t)
}

func TestManualSuccessWithEmptyUserDoesNotImplyBroadcast(t *testing.T) {
	repo := newRunRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	notifier := newRunSuccessNotifierRecorder()
	runtime.SetRunSuccessNotifier(notifier)

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:     "manual-empty-user",
		Schedule: "*/1 * * * * *",
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	})

	if _, err := runtime.RunOnceWithTrigger(context.Background(), "manual-empty-user", RunTrigger{Type: TriggerTypeManual}); err != nil {
		t.Fatalf("run once with empty user trigger: %v", err)
	}

	_, trigger := notifier.wait(t)
	if trigger.Type != TriggerTypeManual || trigger.TriggerUserID != 0 {
		t.Fatalf("expected empty manual trigger to stay userless, got %#v", trigger)
	}
}

type runFailureNotifierRecorder struct {
	done chan TaskRun
	once sync.Once
}

func newRunFailureNotifierRecorder() *runFailureNotifierRecorder {
	return &runFailureNotifierRecorder{done: make(chan TaskRun, 1)}
}

func (r *runFailureNotifierRecorder) NotifyRunFailed(_ context.Context, run TaskRun) {
	r.once.Do(func() {
		r.done <- run
	})
}

func (r *runFailureNotifierRecorder) wait(t *testing.T) TaskRun {
	t.Helper()

	select {
	case run := <-r.done:
		return run
	case <-time.After(time.Second):
		t.Fatal("expected failed-run notification")
		return TaskRun{}
	}
}

type runSuccessNotification struct {
	run     TaskRun
	trigger RunTrigger
}

type runSuccessNotifierRecorder struct {
	done chan runSuccessNotification
	once sync.Once
}

func newRunSuccessNotifierRecorder() *runSuccessNotifierRecorder {
	return &runSuccessNotifierRecorder{done: make(chan runSuccessNotification, 1)}
}

func (r *runSuccessNotifierRecorder) NotifyRunSucceeded(_ context.Context, run TaskRun, trigger RunTrigger) {
	r.once.Do(func() {
		r.done <- runSuccessNotification{run: run, trigger: trigger}
	})
}

func (r *runSuccessNotifierRecorder) wait(t *testing.T) (TaskRun, RunTrigger) {
	t.Helper()

	select {
	case notification := <-r.done:
		return notification.run, notification.trigger
	case <-time.After(time.Second):
		t.Fatal("expected successful-run notification")
		return TaskRun{}, RunTrigger{}
	}
}

func (r *runSuccessNotifierRecorder) assertNoNotification(t *testing.T) {
	t.Helper()

	select {
	case notification := <-r.done:
		t.Fatalf("expected no successful-run notification, got %#v", notification)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestRunOnceDoesNotBlockOnFailureNotifier(t *testing.T) {
	repo := newRunRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	runtime.SetRunFailureNotifier(blockingRunFailureNotifier{})

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:     "manual-blocked-notifier",
		Schedule: "*/1 * * * * *",
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "failed"}, errors.New("boom")
		},
	})

	done := make(chan error, 1)
	go func() {
		_, err := runtime.RunOnce(context.Background(), "manual-blocked-notifier")
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil || err.Error() != "boom" {
			t.Fatalf("expected handler error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("RunOnce blocked on failure notifier")
	}
}

func TestRunOnceIgnoresFailureNotifierPanic(t *testing.T) {
	repo := newRunRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	runtime.SetRunFailureNotifier(panicRunFailureNotifier{})

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:     "manual-panicking-notifier",
		Schedule: "*/1 * * * * *",
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: "failed"}, errors.New("boom")
		},
	})

	if _, err := runtime.RunOnce(context.Background(), "manual-panicking-notifier"); err == nil || err.Error() != "boom" {
		t.Fatalf("expected handler error, got %v", err)
	}
}

type blockingRunFailureNotifier struct{}

func (blockingRunFailureNotifier) NotifyRunFailed(ctx context.Context, _ TaskRun) {
	<-ctx.Done()
}

type panicRunFailureNotifier struct{}

func (panicRunFailureNotifier) NotifyRunFailed(context.Context, TaskRun) {
	panic("notifier failure")
}

func TestRunActionUsesActionHandlerAndSkipsHistory(t *testing.T) {
	repo := newRunRepositoryRecorder()
	taskRepo := newTaskRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)
	runtime.SetTaskRepository(taskRepo)
	var actionConfig string
	jobHandlerCalled := false

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:          "retention",
		Schedule:      "*/1 * * * * *",
		DefaultConfig: `{"batchSize":100}`,
		ConfigSchema:  `{"type":"object","properties":{"batchSize":{"type":"integer","minimum":1}},"additionalProperties":false}`,
		Actions: []cronx.JobAction{
			{
				Key: "dryRun",
				Handler: func(_ context.Context, configJSON string) (cronx.JobRunResult, error) {
					actionConfig = configJSON
					return cronx.JobRunResult{Summary: "estimated"}, nil
				},
			},
		},
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			jobHandlerCalled = true
			return cronx.JobRunResult{Summary: "deleted"}, nil
		},
	})
	taskBefore := taskRepo.tasks["retention"]
	taskBefore.ConfigJSON = `{"batchSize":50}`
	taskRepo.tasks["retention"] = taskBefore

	result, err := runtime.RunAction(context.Background(), "retention", "dryRun", `{"batchSize":25}`)
	if err != nil {
		t.Fatalf("run action: %v", err)
	}
	effective := decodeRuntimeJSONObject(t, result.EffectiveConfig)
	if effective["batchSize"] != float64(25) {
		t.Fatalf("unexpected effective action config: %s", result.EffectiveConfig)
	}
	actionEffective := decodeRuntimeJSONObject(t, actionConfig)
	if actionEffective["batchSize"] != float64(25) {
		t.Fatalf("unexpected action handler config: %s", actionConfig)
	}
	if jobHandlerCalled {
		t.Fatal("expected action handler to run without invoking the normal job handler")
	}
	if result.ActionKey != "dryRun" || result.TaskKey != "retention" || result.JobKey != "retention" || result.Result.Summary != "estimated" {
		t.Fatalf("unexpected action result: %#v", result)
	}
	if len(repo.created) != 0 || len(repo.updated) != 0 {
		t.Fatalf("expected action not to write run history, got created=%d updated=%d", len(repo.created), len(repo.updated))
	}
	taskAfter := taskRepo.tasks["retention"]
	if taskAfter.ConfigJSON != `{"batchSize":50}` || !taskAfter.Enabled {
		t.Fatalf("expected task unchanged after action, got %#v", taskAfter)
	}
}

func TestInvokeJobActionRejectsMissingActionHandler(t *testing.T) {
	jobHandlerCalled := false
	job := cronx.Job{
		Name: "retention",
		Actions: []cronx.JobAction{
			{Key: "validate-config"},
		},
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			jobHandlerCalled = true
			return cronx.JobRunResult{Summary: "validated"}, nil
		},
	}

	_, err := invokeJobAction(context.Background(), job, "validate-config", `{"batchSize":25}`)
	if !errors.Is(err, ErrTaskValidation) {
		t.Fatalf("expected missing action handler validation error, got %v", err)
	}
	if jobHandlerCalled {
		t.Fatal("expected missing action handler not to invoke the normal job handler")
	}
}

func TestRunActionMergesRequestSnapshot(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	taskRepo := newTaskRepositoryRecorder()
	runtime.SetTaskRepository(taskRepo)
	var actionConfig string

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:          "retention",
		Schedule:      "*/1 * * * * *",
		DefaultConfig: `{"batchSize":100}`,
		ConfigSchema:  `{"type":"object","properties":{"batchSize":{"type":"integer","minimum":1}},"additionalProperties":false}`,
		Actions: []cronx.JobAction{
			{
				Key: "validate-config",
				Handler: func(_ context.Context, configJSON string) (cronx.JobRunResult, error) {
					actionConfig = configJSON
					return cronx.JobRunResult{Summary: "validated"}, nil
				},
			},
		},
		Handler: func(_ context.Context, configJSON string) (cronx.JobRunResult, error) {
			return cronx.JobRunResult{Summary: configJSON}, nil
		},
	})

	result, err := runtime.RunAction(context.Background(), "retention", "validate-config", `{"batchSize":25}`)
	if err != nil {
		t.Fatalf("run action: %v", err)
	}
	effective := decodeRuntimeJSONObject(t, result.EffectiveConfig)
	if effective["batchSize"] != float64(25) {
		t.Fatalf("unexpected effective action config: %s", result.EffectiveConfig)
	}
	handlerEffective := decodeRuntimeJSONObject(t, actionConfig)
	if handlerEffective["batchSize"] != float64(25) {
		t.Fatalf("unexpected handler config: %s", actionConfig)
	}
}

func TestRunActionRejectsUnknownAction(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	seedRuntimeJob(t, runtime, cronx.Job{
		Name:     "retention",
		Schedule: "*/1 * * * * *",
		Actions: []cronx.JobAction{
			{
				Key: "dryRun",
				Handler: func(context.Context, string) (cronx.JobRunResult, error) {
					return cronx.JobRunResult{Summary: "dry run"}, nil
				},
			},
		},
		Run: func(context.Context) error { return nil },
	})

	_, err := runtime.RunAction(context.Background(), "retention", "missing", `{}`)
	if !errors.Is(err, ErrJobActionNotFound) {
		t.Fatalf("expected unknown action error, got %v", err)
	}
}

func TestRunActionValidatesMergedConfigBeforeExecution(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	called := false
	seedRuntimeJob(t, runtime, cronx.Job{
		Name:          "retention",
		Schedule:      "*/1 * * * * *",
		DefaultConfig: `{"batchSize":10}`,
		ConfigSchema:  `{"type":"object","properties":{"batchSize":{"type":"integer","minimum":1}},"additionalProperties":false}`,
		Actions: []cronx.JobAction{
			{
				Key: "dryRun",
				Handler: func(context.Context, string) (cronx.JobRunResult, error) {
					called = true
					return cronx.JobRunResult{Summary: "dry run"}, nil
				},
			},
		},
		Handler: func(context.Context, string) (cronx.JobRunResult, error) {
			called = true
			return cronx.JobRunResult{Summary: "ok"}, nil
		},
	})

	_, err := runtime.RunAction(context.Background(), "retention", "dryRun", `{"batchSize":0}`)
	var configErr ConfigValidationError
	if !errors.As(err, &configErr) || configErr.Field != "config_json.batchSize" {
		t.Fatalf("expected config validation error, got %v", err)
	}
	if called {
		t.Fatal("expected invalid merged config to skip handler execution")
	}
}

// TestRunOnceRejectsConcurrentSameTask 验证同一任务运行中再次手动触发会返回冲突式错误。
func TestRunOnceRejectsConcurrentSameTask(t *testing.T) {
	repo := newRunRepositoryRecorder()
	runtime := New(zap.NewNop(), repo)
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	started := make(chan struct{}, 1)
	release := make(chan struct{})

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:     "blocking",
		Schedule: "*/1 * * * * *",
		Run: func(context.Context) error {
			select {
			case started <- struct{}{}:
			default:
			}
			<-release
			return nil
		},
	})

	firstDone := make(chan error, 1)
	go func() {
		_, err := runtime.RunOnce(context.Background(), "blocking")
		firstDone <- err
	}()

	waitForSignal(t, started, time.Second, "expected first manual run to start")

	if _, err := runtime.RunOnce(context.Background(), "blocking"); !errors.Is(err, ErrTaskAlreadyRunning) {
		t.Fatalf("expected already-running conflict, got %v", err)
	}

	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first manual run failed: %v", err)
	}
}

// TestStartAndStopRunsRegisteredJob 验证最小调度器可以启动、执行一次任务并正常停止。
func TestStartAndStopRunsRegisteredJob(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	triggered := make(chan struct{}, 1)
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:     "heartbeat",
		Schedule: "*/1 * * * * *",
		Run: func(_ context.Context) error {
			select {
			case triggered <- struct{}{}:
			default:
			}
			return nil
		},
	})

	if err := runtime.Start(runCtx); err != nil {
		t.Fatalf("start runtime: %v", err)
	}
	defer func() {
		_ = runtime.Stop(context.Background())
	}()

	waitForSignal(t, triggered, 2500*time.Millisecond, "expected scheduled job to run")

	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatalf("stop runtime: %v", err)
	}
}

// TestStartRejectsCanceledContextBeforeRepositoryLoad 验证 Start 遇到已取消的生命周期
// 上下文时不会继续读取持久化任务，避免 pgx 把取消状态包装成数据库启动失败。
func TestStartRejectsCanceledContextBeforeRepositoryLoad(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	taskRepo := newTaskRepositoryRecorder()
	runtime.SetTaskRepository(taskRepo)

	runCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runtime.Start(runCtx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if taskRepo.listCalls != 0 {
		t.Fatalf("expected canceled start to skip repository load, got %d calls", taskRepo.listCalls)
	}
	if runtime.started || runtime.lifecycleCtx != nil || runtime.lifecycleCancel != nil {
		t.Fatal("expected canceled start to leave runtime unstarted")
	}
}

// TestStartKeepsRuntimeUnstartedWhenRepositoryLoadFails 验证持久化任务加载失败时不会
// 留下半初始化的 lifecycle 状态。
func TestStartKeepsRuntimeUnstartedWhenRepositoryLoadFails(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	taskRepo := newTaskRepositoryRecorder()
	taskRepo.listErr = errors.New("list failed")
	runtime.SetTaskRepository(taskRepo)

	err := runtime.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "list failed") {
		t.Fatalf("expected repository load error, got %v", err)
	}
	if taskRepo.listCalls != 1 {
		t.Fatalf("expected one repository load, got %d", taskRepo.listCalls)
	}
	if runtime.started || runtime.lifecycleCtx != nil || runtime.lifecycleCancel != nil {
		t.Fatal("expected failed start to leave runtime unstarted")
	}
}

func decodeRuntimeJSONObject(t *testing.T, value string) map[string]any {
	t.Helper()

	var decoded map[string]any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		t.Fatalf("decode json object %q: %v", value, err)
	}
	return decoded
}

// TestRemoveJobPreventsFutureExecution 验证移除任务后后续调度不会再次触发该任务。
func TestRemoveJobPreventsFutureExecution(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	triggered := make(chan struct{}, 2)
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:     "cleanup",
		Schedule: "*/1 * * * * *",
		Run: func(_ context.Context) error {
			triggered <- struct{}{}
			return nil
		},
	})
	if err := runtime.Start(runCtx); err != nil {
		t.Fatalf("start runtime: %v", err)
	}

	waitForSignal(t, triggered, 2500*time.Millisecond, "expected first scheduled execution")

	if err := runtime.RemoveJob("cleanup"); err != nil {
		t.Fatalf("remove job: %v", err)
	}

	assertNoSignal(t, triggered, 1200*time.Millisecond, "expected removed job not to run again")

	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatalf("stop runtime: %v", err)
	}
}

// TestStopHonorsContextCancellation 验证 Stop 会把外部取消信号作为稳定错误返回。
func TestStopHonorsContextCancellation(t *testing.T) {
	runtime := New(zap.NewNop())
	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	if err := runtime.Start(runCtx); err != nil {
		t.Fatalf("start runtime: %v", err)
	}

	stopCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runtime.Stop(stopCtx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

// TestStopCancelsJobLifecycleContext 验证显式 Stop 会取消运行中任务绑定的 lifecycle ctx。
func TestStopCancelsJobLifecycleContext(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	runCtx := context.Background()
	started := make(chan context.Context, 1)
	finished := make(chan struct{}, 1)

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:     "watch",
		Schedule: "*/1 * * * * *",
		Run: func(ctx context.Context) error {
			select {
			case started <- ctx:
			default:
			}
			<-ctx.Done()
			select {
			case finished <- struct{}{}:
			default:
			}
			return nil
		},
	})

	if err := runtime.Start(runCtx); err != nil {
		t.Fatalf("start runtime: %v", err)
	}

	jobCtx := waitForJobContext(t, started, 2500*time.Millisecond, "expected scheduled job to start")
	if jobCtx == nil {
		t.Fatal("expected job to receive lifecycle context")
	}
	if jobCtx.Err() != nil {
		t.Fatalf("expected job lifecycle context to be active, got %v", jobCtx.Err())
	}

	stopDone := make(chan error, 1)
	go func() {
		stopDone <- runtime.Stop(context.Background())
	}()

	waitForContextDone(jobCtx, t, time.Second, "expected stop to cancel job lifecycle context")

	waitForSignal(t, finished, time.Second, "expected job to observe lifecycle cancellation")
	waitForStopResult(t, stopDone, time.Second)
}

// TestStopWithNilContextWaitsForInFlightJob 验证 nil ctx 会等待当前在途任务自然结束。
func TestStopWithNilContextWaitsForInFlightJob(t *testing.T) {
	runtime := New(zap.NewNop(), newRunRepositoryRecorder())
	runtime.SetTaskRepository(newTaskRepositoryRecorder())
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	finished := make(chan struct{}, 1)
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runtime.cron.Schedule(runSoonSchedule{}, cron.FuncJob(func() {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		select {
		case finished <- struct{}{}:
		default:
		}
	}))

	seedRuntimeJob(t, runtime, cronx.Job{
		Name:     "blocking",
		Schedule: "0 0 0 1 1 *",
		Run: func(_ context.Context) error {
			select {
			case started <- struct{}{}:
			default:
			}
			<-release
			select {
			case finished <- struct{}{}:
			default:
			}
			return nil
		},
	})

	if err := runtime.Start(runCtx); err != nil {
		t.Fatalf("start runtime: %v", err)
	}

	waitForSignal(t, started, 2500*time.Millisecond, "expected scheduled job to start")

	stopDone := make(chan error, 1)
	var stopCtx context.Context
	go func() {
		stopDone <- runtime.Stop(stopCtx)
	}()

	assertNoStopResult(t, stopDone, 200*time.Millisecond)

	close(release)

	waitForSignal(t, finished, time.Second, "expected blocked job to finish after release")
	waitForStopResult(t, stopDone, time.Second)
}

type runSoonSchedule struct{}

func (runSoonSchedule) Next(time.Time) time.Time {
	return time.Now().Add(10 * time.Millisecond)
}

func waitForSignal(t *testing.T, signal <-chan struct{}, timeout time.Duration, failureMessage string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(timeout):
		t.Fatal(failureMessage)
	}
}

func waitForJobContext(t *testing.T, signal <-chan context.Context, timeout time.Duration, failureMessage string) context.Context {
	t.Helper()

	select {
	case ctx := <-signal:
		return ctx
	case <-time.After(timeout):
		t.Fatal(failureMessage)
		return nil
	}
}

func waitForContextDone(ctx context.Context, t *testing.T, timeout time.Duration, failureMessage string) {
	t.Helper()

	select {
	case <-ctx.Done():
	case <-time.After(timeout):
		t.Fatal(failureMessage)
	}
}

func assertNoSignal(t *testing.T, signal <-chan struct{}, timeout time.Duration, failureMessage string) {
	t.Helper()

	select {
	case <-signal:
		t.Fatal(failureMessage)
	case <-time.After(timeout):
	}
}

func assertNoStopResult(t *testing.T, stopDone <-chan error, timeout time.Duration) {
	t.Helper()

	select {
	case err := <-stopDone:
		t.Fatalf("expected Stop(nil) to wait for in-flight job, got early result %v", err)
	case <-time.After(timeout):
	}
}

func waitForStopResult(t *testing.T, stopDone <-chan error, timeout time.Duration) {
	t.Helper()

	select {
	case err := <-stopDone:
		if err != nil {
			t.Fatalf("stop runtime: %v", err)
		}
	case <-time.After(timeout):
		t.Fatal("expected Stop(nil) to return after in-flight job finished")
	}
}
