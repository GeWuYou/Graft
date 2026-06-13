// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/dashboard"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	schedulercore "graft/server/internal/scheduler"
	schedulercontract "graft/server/modules/scheduler/contract"
)

type stopContextRecorderRuntime struct {
	registeredJobs []cronx.Job
	startCtx       context.Context
	stopCtx        context.Context
}

func (r *stopContextRecorderRuntime) RegisterJob(job cronx.Job) error {
	r.registeredJobs = append(r.registeredJobs, job)
	return nil
}

func (r *stopContextRecorderRuntime) RemoveJob(_ string) error { return nil }

func (r *stopContextRecorderRuntime) Start(ctx context.Context) error {
	r.startCtx = ctx
	return nil
}

func (r *stopContextRecorderRuntime) Stop(ctx context.Context) error {
	r.stopCtx = ctx
	return nil
}

func (r *stopContextRecorderRuntime) ListJobDefinitions(context.Context) ([]schedulercore.JobDefinitionSnapshot, error) {
	return nil, nil
}

func (r *stopContextRecorderRuntime) GetJobDefinition(context.Context, string) (schedulercore.JobDefinitionSnapshot, error) {
	return schedulercore.JobDefinitionSnapshot{}, nil
}

func (r *stopContextRecorderRuntime) ListTasks(context.Context, schedulercore.TaskListQuery) (schedulercore.TaskListResult, error) {
	return schedulercore.TaskListResult{}, nil
}

func (r *stopContextRecorderRuntime) GetTask(context.Context, string) (schedulercore.TaskSnapshot, error) {
	return schedulercore.TaskSnapshot{}, nil
}

func (r *stopContextRecorderRuntime) SeedBuiltinJobs(_ context.Context, jobs []cronx.Job) error {
	for _, job := range jobs {
		if err := r.RegisterJob(job); err != nil {
			return err
		}
	}
	return nil
}

func (r *stopContextRecorderRuntime) CreateTask(context.Context, schedulercore.TaskMutation) (schedulercore.TaskSnapshot, error) {
	return schedulercore.TaskSnapshot{}, nil
}

func (r *stopContextRecorderRuntime) UpdateTask(context.Context, string, schedulercore.TaskMutation) (schedulercore.TaskSnapshot, error) {
	return schedulercore.TaskSnapshot{}, nil
}

func (r *stopContextRecorderRuntime) DeleteTask(context.Context, string) error {
	return nil
}

func (r *stopContextRecorderRuntime) SetTaskEnabled(context.Context, string, bool) (schedulercore.TaskSnapshot, error) {
	return schedulercore.TaskSnapshot{}, nil
}

func (r *stopContextRecorderRuntime) ListRuns(context.Context, schedulercore.RunListQuery) (schedulercore.RunListResult, error) {
	return schedulercore.RunListResult{}, nil
}

func (r *stopContextRecorderRuntime) GetRun(context.Context, uint64) (schedulercore.TaskRun, error) {
	return schedulercore.TaskRun{}, nil
}

func (r *stopContextRecorderRuntime) RunOnce(context.Context, string) (schedulercore.TaskRun, error) {
	return schedulercore.TaskRun{}, nil
}

func (r *stopContextRecorderRuntime) RunOnceWithTrigger(context.Context, string, schedulercore.RunTrigger) (schedulercore.TaskRun, error) {
	return schedulercore.TaskRun{}, nil
}

func (r *stopContextRecorderRuntime) RunAction(context.Context, string, string, string) (schedulercore.JobActionResult, error) {
	return schedulercore.JobActionResult{}, nil
}

type schedulerAPIRuntime struct {
	stopContextRecorderRuntime
	jobDefinitions  []schedulercore.JobDefinitionSnapshot
	tasks           []schedulercore.TaskSnapshot
	listTaskQueries []schedulercore.TaskListQuery
	createInputs    []schedulercore.TaskMutation
	createResult    schedulercore.TaskSnapshot
	createErr       error
	updateInputs    []schedulercore.TaskMutation
	updateKeys      []string
	updateResult    schedulercore.TaskSnapshot
	updateErr       error
	deleteKeys      []string
	deleteErr       error
	setEnabledKeys  []string
	setEnabledVals  []bool
	setResult       schedulercore.TaskSnapshot
	setErr          error
	runOnceKeys     []string
	runOnceTriggers []schedulercore.RunTrigger
	runOnceResult   schedulercore.TaskRun
	runOnceErr      error
	actionTaskKeys  []string
	actionKeys      []string
	actionConfigs   []string
	actionResult    schedulercore.JobActionResult
	actionErr       error
	getRunID        uint64
	getRunResult    schedulercore.TaskRun
	getRunErr       error
}

func (r *schedulerAPIRuntime) ListJobDefinitions(context.Context) ([]schedulercore.JobDefinitionSnapshot, error) {
	return r.jobDefinitions, nil
}

func (r *schedulerAPIRuntime) GetJobDefinition(_ context.Context, key string) (schedulercore.JobDefinitionSnapshot, error) {
	for _, definition := range r.jobDefinitions {
		if definition.JobKey == key {
			return definition, nil
		}
	}
	return schedulercore.JobDefinitionSnapshot{}, schedulercore.ErrJobDefinitionNotFound
}

func (r *schedulerAPIRuntime) ListTasks(_ context.Context, query schedulercore.TaskListQuery) (schedulercore.TaskListResult, error) {
	r.listTaskQueries = append(r.listTaskQueries, query)
	items := r.tasks
	total := len(items)
	if query.Limit > 0 {
		start := min(max(query.Offset, 0), total)
		end := min(start+query.Limit, total)
		items = items[start:end]
	}
	return schedulercore.TaskListResult{Items: items, Total: total}, nil
}

func (r *schedulerAPIRuntime) CreateTask(_ context.Context, command schedulercore.TaskMutation) (schedulercore.TaskSnapshot, error) {
	r.createInputs = append(r.createInputs, command)
	if r.createErr != nil {
		return schedulercore.TaskSnapshot{}, r.createErr
	}
	if r.createResult.Key == "" {
		r.createResult = taskSnapshotFromMutation(command)
	}
	return r.createResult, nil
}

func (r *schedulerAPIRuntime) UpdateTask(_ context.Context, key string, command schedulercore.TaskMutation) (schedulercore.TaskSnapshot, error) {
	r.updateKeys = append(r.updateKeys, key)
	r.updateInputs = append(r.updateInputs, command)
	if r.updateErr != nil {
		return schedulercore.TaskSnapshot{}, r.updateErr
	}
	if r.updateResult.Key == "" {
		r.updateResult = taskSnapshotFromMutation(command)
		r.updateResult.Key = key
	}
	return r.updateResult, nil
}

func (r *schedulerAPIRuntime) SetTaskEnabled(_ context.Context, key string, enabled bool) (schedulercore.TaskSnapshot, error) {
	r.setEnabledKeys = append(r.setEnabledKeys, key)
	r.setEnabledVals = append(r.setEnabledVals, enabled)
	if r.setErr != nil {
		return schedulercore.TaskSnapshot{}, r.setErr
	}
	if r.setResult.Key == "" {
		r.setResult = schedulercore.TaskSnapshot{
			Key:        key,
			JobKey:     "scheduler.test-job",
			Title:      key,
			Schedule:   "*/5 * * * * *",
			Enabled:    enabled,
			ConfigJSON: "{}",
		}
	}
	r.setResult.Enabled = enabled
	return r.setResult, nil
}

func (r *schedulerAPIRuntime) RunOnce(_ context.Context, key string) (schedulercore.TaskRun, error) {
	return r.RunOnceWithTrigger(context.Background(), key, schedulercore.RunTrigger{Type: schedulercore.TriggerTypeManual})
}

func (r *schedulerAPIRuntime) RunOnceWithTrigger(_ context.Context, key string, trigger schedulercore.RunTrigger) (schedulercore.TaskRun, error) {
	r.runOnceKeys = append(r.runOnceKeys, key)
	r.runOnceTriggers = append(r.runOnceTriggers, trigger)
	if r.runOnceErr != nil {
		return schedulercore.TaskRun{}, r.runOnceErr
	}
	if r.runOnceResult.ID == 0 {
		r.runOnceResult = schedulercore.TaskRun{
			ID:            17,
			TaskKey:       key,
			JobKey:        "scheduler.test-job",
			TaskTitle:     key,
			JobTitle:      "Scheduler test job",
			JobShortTitle: "Test Job",
			JobCategory:   "custom",
			ModuleKey:     "scheduler",
			TriggerType:   schedulercore.TriggerTypeManual,
			Status:        schedulercore.RunStatusSuccess,
			StartedAt:     time.Now().UTC(),
			CreatedAt:     time.Now().UTC(),
		}
	}
	return r.runOnceResult, nil
}

func (r *schedulerAPIRuntime) RunAction(_ context.Context, taskKey string, actionKey string, configJSON string) (schedulercore.JobActionResult, error) {
	r.actionTaskKeys = append(r.actionTaskKeys, taskKey)
	r.actionKeys = append(r.actionKeys, actionKey)
	r.actionConfigs = append(r.actionConfigs, configJSON)
	if r.actionErr != nil {
		return schedulercore.JobActionResult{}, r.actionErr
	}
	if r.actionResult.ActionKey == "" {
		r.actionResult = schedulercore.JobActionResult{
			ActionKey:       actionKey,
			TaskKey:         taskKey,
			JobKey:          "scheduler.test-job",
			EffectiveConfig: `{"batchSize":25}`,
		}
		r.actionResult.Result.Summary = "previewed"
		r.actionResult.Result.Stage = "completed"
		r.actionResult.Result.Metrics = map[string]any{"estimatedDeleteCount": int64(128)}
	}
	return r.actionResult, nil
}

func (r *schedulerAPIRuntime) GetRun(_ context.Context, id uint64) (schedulercore.TaskRun, error) {
	r.getRunID = id
	return r.getRunResult, r.getRunErr
}

func taskSnapshotFromMutation(command schedulercore.TaskMutation) schedulercore.TaskSnapshot {
	return schedulercore.TaskSnapshot{
		Key:         command.TaskKey,
		JobKey:      command.JobKey,
		Title:       command.Title,
		Description: command.Description,
		Schedule:    command.CronExpression,
		Enabled:     command.Enabled,
		ConfigJSON:  command.ConfigJSON,
	}
}

type testAuthService struct{}

func (testAuthService) CurrentUser(context.Context) (*moduleapi.CurrentUser, error) {
	return &moduleapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"}, nil
}

func (testAuthService) ParseAccessToken(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
	return &moduleapi.AccessTokenClaims{
		UserID:       7,
		SessionID:    "session-1",
		TokenVersion: 1,
		IssuedAt:     time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(time.Minute),
	}, nil
}

type allowAllAuthorizer struct{}

func (allowAllAuthorizer) Authorize(context.Context, moduleapi.RequestAuthContext, string) error {
	return nil
}

type recordingAuthorizer struct {
	permissions []string
}

func (a *recordingAuthorizer) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, permission string) error {
	a.permissions = append(a.permissions, permission)
	return nil
}

const schedulerModuleTestSchema = `CREATE TABLE scheduled_tasks (
	id integer PRIMARY KEY AUTOINCREMENT,
	task_key text NOT NULL,
	job_key text NOT NULL DEFAULT '',
	title_key text NOT NULL DEFAULT '',
	title text NOT NULL CHECK (trim(title) <> ''),
	description_key text NOT NULL DEFAULT '',
	description text NOT NULL DEFAULT '',
	cron_expression text NOT NULL,
	enabled boolean NOT NULL DEFAULT true,
	builtin boolean NOT NULL DEFAULT false,
	config_json text NOT NULL DEFAULT '{}',
	config_source text NOT NULL DEFAULT 'system',
	created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at integer NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX scheduled_tasks_task_key_live_key
	ON scheduled_tasks (task_key)
	WHERE deleted_at = 0;
CREATE TABLE scheduler_job_definitions (
	id integer PRIMARY KEY AUTOINCREMENT,
	job_key text NOT NULL,
	module_key text NOT NULL DEFAULT '',
	category text NOT NULL DEFAULT 'custom',
	title_key text NOT NULL DEFAULT '',
	title text NOT NULL DEFAULT '',
	short_title_key text NOT NULL DEFAULT '',
	short_title text NOT NULL DEFAULT '',
	description_key text NOT NULL DEFAULT '',
	description text NOT NULL DEFAULT '',
	config_schema text NOT NULL DEFAULT '{}',
	default_config text NOT NULL DEFAULT '{}',
	default_cron text NOT NULL DEFAULT '',
	default_enabled boolean NOT NULL DEFAULT false,
	enabled boolean NOT NULL DEFAULT true,
	created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at integer NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX scheduler_job_definitions_job_key_live_key
	ON scheduler_job_definitions (job_key)
	WHERE deleted_at = 0;
CREATE TABLE scheduler_task_runs (
	id integer PRIMARY KEY AUTOINCREMENT,
	task_key text NOT NULL,
	job_key text NOT NULL DEFAULT '',
	task_title text NOT NULL DEFAULT '',
	task_title_key text NOT NULL DEFAULT '',
	job_title text NOT NULL DEFAULT '',
	job_title_key text NOT NULL DEFAULT '',
	job_short_title text NOT NULL DEFAULT '',
	job_short_title_key text NOT NULL DEFAULT '',
	job_category text NOT NULL DEFAULT 'custom',
	module_key text NOT NULL DEFAULT '',
	task_builtin boolean NOT NULL DEFAULT false,
	trigger_type text NOT NULL,
	status text NOT NULL,
	result_summary text NOT NULL DEFAULT '',
	result_json text NOT NULL DEFAULT '{}',
	error_message text NOT NULL DEFAULT '',
	started_at datetime NOT NULL,
	finished_at datetime NULL,
	duration_ms integer NULL,
	created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
)`

func newModuleTestContext() *module.Context {
	ctx, _ := newModuleTestContextWithEngine()
	return ctx
}

func newModuleTestContextWithEngine() (*module.Context, *gin.Engine) {
	return newModuleTestContextWithEngineAndAuthorizer(allowAllAuthorizer{})
}

func newModuleTestContextWithEngineAndAuthorizer(authorizer moduleapi.Authorizer) (*module.Context, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	db := newSchedulerModuleTestDB()
	services := container.New()
	if err := services.RegisterSingleton((*sql.DB)(nil), func(container.Resolver) (any, error) {
		return db, nil
	}); err != nil {
		panic(err)
	}
	if err := services.RegisterSingleton((*moduleapi.AuthService)(nil), func(container.Resolver) (any, error) {
		return testAuthService{}, nil
	}); err != nil {
		panic(err)
	}
	if err := services.RegisterSingleton((*moduleapi.Authorizer)(nil), func(container.Resolver) (any, error) {
		return authorizer, nil
	}); err != nil {
		panic(err)
	}

	return &module.Context{
		Logger:             zap.NewNop(),
		Config:             &config.Config{},
		I18n:               i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}),
		EventBus:           eventbus.New(zap.NewNop()),
		Router:             engine.Group("/api"),
		Services:           services,
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
		DashboardRegistry:  dashboard.NewRegistry(),
	}, engine
}

func newSchedulerModuleTestDB() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schedulerModuleTestSchema); err != nil {
		panic(err)
	}
	return db
}

func (r *schedulerAPIRuntime) DeleteTask(_ context.Context, key string) error {
	r.deleteKeys = append(r.deleteKeys, key)
	return r.deleteErr
}

func registerAndBootSchedulerModule(t *testing.T, ctx *module.Context, moduleInstance *Module) {
	t.Helper()
	if err := moduleInstance.Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}
	ctx.LifecycleContext = context.Background()
	if err := moduleInstance.Boot(ctx); err != nil {
		t.Fatalf("boot module: %v", err)
	}
	t.Cleanup(func() {
		_ = moduleInstance.Shutdown(ctx)
	})
}

func performSchedulerRequest(engine *gin.Engine, method string, path string, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Authorization", "Bearer token")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}

// TestRegisterExposesRuntimeService 验证 scheduler 模块会把运行时能力注册到服务容器。
func TestRegisterExposesRuntimeService(t *testing.T) {
	ctx := newModuleTestContext()
	moduleInstance := NewModule()

	if err := moduleInstance.Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}

	resolved, err := ctx.Services.Resolve((*schedulercore.Runtime)(nil))
	if err != nil {
		t.Fatalf("resolve runtime service: %v", err)
	}
	if _, ok := resolved.(schedulercore.Runtime); !ok {
		t.Fatalf("expected scheduler runtime service, got %T", resolved)
	}
}

func TestRegisterMessagesIncludesRunFailureNotificationKeys(t *testing.T) {
	localizer := i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}})

	if err := registerMessages(localizer); err != nil {
		t.Fatalf("register scheduler messages: %v", err)
	}

	assertRegisteredSchedulerMessage(t, localizer, i18n.LocaleZHCN, schedulercontract.ScheduledTaskRunFailedNotificationTitle.String(), "定时任务失败")
	assertRegisteredSchedulerMessage(t, localizer, i18n.LocaleZHCN, schedulercontract.ScheduledTaskRunFailedNotificationMessage.String(), "定时任务执行失败。")
	assertRegisteredSchedulerMessage(t, localizer, i18n.LocaleENUS, schedulercontract.ScheduledTaskRunFailedNotificationTitle.String(), "Scheduled Task Failed")
	assertRegisteredSchedulerMessage(t, localizer, i18n.LocaleENUS, schedulercontract.ScheduledTaskRunFailedNotificationMessage.String(), "Scheduled task failed.")
	assertRegisteredSchedulerMessage(t, localizer, i18n.LocaleZHCN, schedulercontract.ScheduledTaskRunSucceededNotificationTitle.String(), "定时任务成功")
	assertRegisteredSchedulerMessage(t, localizer, i18n.LocaleZHCN, schedulercontract.ScheduledTaskRunSucceededNotificationMessage.String(), "手动定时任务执行成功。")
	assertRegisteredSchedulerMessage(t, localizer, i18n.LocaleENUS, schedulercontract.ScheduledTaskRunSucceededNotificationTitle.String(), "Scheduled Task Succeeded")
	assertRegisteredSchedulerMessage(t, localizer, i18n.LocaleENUS, schedulercontract.ScheduledTaskRunSucceededNotificationMessage.String(), "Manual scheduled task succeeded.")
}

func TestSchedulerRunSuccessNotifierPublishesManualSuccessToTriggerUser(t *testing.T) {
	publisher := &schedulerNotificationPublisherRecorder{}
	notifier := schedulerRunSuccessNotifier{publisher: publisher, logger: zap.NewNop()}
	finishedAt := time.Date(2026, 6, 11, 9, 30, 0, 0, time.UTC)

	notifier.NotifyRunSucceeded(context.Background(), schedulercore.TaskRun{
		ID:          99,
		TaskKey:     "webhook.health",
		JobKey:      "scheduler.webhook-health",
		TaskTitle:   "Webhook Health",
		JobTitleKey: "scheduler.job.webhookHealth.title",
		TaskBuiltin: false,
		Status:      schedulercore.RunStatusSuccess,
		Result:      "deleted 3 rows",
		FinishedAt:  &finishedAt,
		CreatedAt:   finishedAt.Add(-time.Second),
	}, schedulercore.RunTrigger{Type: schedulercore.TriggerTypeManual, TriggerUserID: 42})

	if len(publisher.inputs) != 1 {
		t.Fatalf("expected one success notification, got %d", len(publisher.inputs))
	}
	input := publisher.inputs[0]
	assertSchedulerRunSuccessDisplay(t, input)
	assertSchedulerRunSuccessRouting(t, input)
	assertSchedulerRunPayload(t, input.Navigation.Payload, uint64(99), "webhook.health", "scheduler.webhook-health")
	assertSchedulerRunSuccessMetadata(t, input.Metadata, expectedSchedulerRunSuccessMetadata{
		runID:         99,
		taskName:      "Webhook Health",
		jobKey:        "scheduler.webhook-health",
		jobTitleKey:   "scheduler.job.webhookHealth.title",
		triggerType:   "manual",
		resultSummary: "deleted 3 rows",
	})
}

func TestSchedulerRunSuccessNotifierPublishesBuiltinTaskInstanceMetadata(t *testing.T) {
	publisher := &schedulerNotificationPublisherRecorder{}
	notifier := schedulerRunSuccessNotifier{publisher: publisher, logger: zap.NewNop()}
	finishedAt := time.Date(2026, 6, 11, 9, 30, 0, 0, time.UTC)

	notifier.NotifyRunSucceeded(context.Background(), schedulercore.TaskRun{
		ID:           99,
		TaskKey:      "httpx.access-log-retention-cleanup",
		JobKey:       "httpx.access-log-retention-cleanup",
		TaskTitle:    "Access log retention cleanup",
		TaskTitleKey: "scheduler.job.accessLogRetentionCleanup.title",
		JobTitleKey:  "scheduler.job.accessLogRetentionCleanup.title",
		TaskBuiltin:  true,
		Status:       schedulercore.RunStatusSuccess,
		Result:       "deleted 0 rows",
		FinishedAt:   &finishedAt,
		CreatedAt:    finishedAt.Add(-time.Second),
	}, schedulercore.RunTrigger{Type: schedulercore.TriggerTypeManual, TriggerUserID: 42})

	if len(publisher.inputs) != 1 {
		t.Fatalf("expected one builtin success notification, got %d", len(publisher.inputs))
	}
	assertSchedulerRunSuccessMetadata(t, publisher.inputs[0].Metadata, expectedSchedulerRunSuccessMetadata{
		runID:         99,
		taskTitleKey:  "scheduler.job.accessLogRetentionCleanup.title",
		taskName:      "Access log retention cleanup",
		taskBuiltin:   true,
		jobKey:        "httpx.access-log-retention-cleanup",
		jobTitleKey:   "scheduler.job.accessLogRetentionCleanup.title",
		triggerType:   "manual",
		resultSummary: "deleted 0 rows",
	})
}

func TestSchedulerRunSuccessNotifierSkipsMissingTriggerUser(t *testing.T) {
	publisher := &schedulerNotificationPublisherRecorder{}
	notifier := schedulerRunSuccessNotifier{publisher: publisher, logger: zap.NewNop()}

	notifier.NotifyRunSucceeded(context.Background(), schedulercore.TaskRun{
		ID:        99,
		TaskKey:   "webhook.health",
		JobKey:    "scheduler.webhook-health",
		Status:    schedulercore.RunStatusSuccess,
		CreatedAt: time.Now().UTC(),
	}, schedulercore.RunTrigger{Type: schedulercore.TriggerTypeManual})

	if len(publisher.inputs) != 0 {
		t.Fatalf("expected missing trigger user to skip notification, got %#v", publisher.inputs)
	}
}

func TestSchedulerRunSuccessNotifierLogsPublishFailureWithoutReturning(t *testing.T) {
	publisher := &schedulerNotificationPublisherRecorder{err: errors.New("publish failed")}
	notifier := schedulerRunSuccessNotifier{publisher: publisher, logger: zap.NewNop()}

	notifier.NotifyRunSucceeded(context.Background(), schedulercore.TaskRun{
		ID:        100,
		TaskKey:   "webhook.health",
		JobKey:    "scheduler.webhook-health",
		Status:    schedulercore.RunStatusSuccess,
		CreatedAt: time.Now().UTC(),
	}, schedulercore.RunTrigger{Type: schedulercore.TriggerTypeManual, TriggerUserID: 42})

	if len(publisher.inputs) != 1 {
		t.Fatalf("expected publish attempt despite publisher error, got %d", len(publisher.inputs))
	}
}

func TestSchedulerRunFailureNotifierStillPublishesPermissionTarget(t *testing.T) {
	publisher := &schedulerNotificationPublisherRecorder{}
	notifier := schedulerRunFailureNotifier{publisher: publisher, logger: zap.NewNop()}

	notifier.NotifyRunFailed(context.Background(), schedulercore.TaskRun{
		ID:           101,
		TaskKey:      "webhook.health",
		JobKey:       "scheduler.webhook-health",
		TaskTitle:    "Webhook Health",
		TriggerType:  schedulercore.TriggerTypeManual,
		Status:       schedulercore.RunStatusFailed,
		ErrorMessage: "boom",
		CreatedAt:    time.Now().UTC(),
	})

	if len(publisher.inputs) != 1 {
		t.Fatalf("expected one failure notification, got %d", len(publisher.inputs))
	}
	input := publisher.inputs[0]
	if input.EventType != "task_failed" ||
		input.Severity != "error" ||
		input.Category != "TASK" ||
		input.Navigation.Kind != "SCHEDULER_RUN" ||
		input.Target.Type != "PERMISSION" ||
		input.Target.Ref != schedulercontract.ScheduledTaskReadPermission.String() ||
		input.DedupeKey != "scheduler:run_failed:101" {
		t.Fatalf("unexpected failure notification input: %#v", input)
	}
}

func TestRegisterRegistersSchedulerTaskAttentionDashboardWidget(t *testing.T) {
	ctx := newModuleTestContext()
	moduleInstance := NewModule()

	if err := moduleInstance.Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}

	widget, ok := ctx.DashboardRegistry.Get(schedulerTaskAttentionWidgetID)
	if !ok {
		t.Fatalf("expected scheduler task attention dashboard widget to be registered")
	}
	if widget.Type != dashboard.WidgetTypeStatGroup {
		t.Fatalf("expected stat-group widget, got %q", widget.Type)
	}
	if widget.RouteLocation != schedulercontract.ScheduledTaskMenuPath {
		t.Fatalf("expected scheduler route %q, got %q", schedulercontract.ScheduledTaskMenuPath, widget.RouteLocation)
	}
	if len(widget.RequiredPermissions) != 1 || widget.RequiredPermissions[0] != schedulercontract.ScheduledTaskReadPermission.String() {
		t.Fatalf("unexpected required permissions: %#v", widget.RequiredPermissions)
	}

	quickLinks := ctx.DashboardRegistry.QuickLinks()
	if len(quickLinks) != 1 {
		t.Fatalf("expected scheduler quick link, got %#v", quickLinks)
	}
	link := quickLinks[0]
	if link.ID != schedulerTaskQuickLinkID ||
		link.ModuleKey != moduleID ||
		link.TitleKey != schedulercontract.ScheduledTaskMenuTitle.String() ||
		link.RouteLocation != schedulercontract.ScheduledTaskMenuPath ||
		link.Order != schedulerTaskQuickLinkOrder {
		t.Fatalf("unexpected scheduler quick link: %#v", link)
	}
	if len(link.RequiredPermissions) != 1 || link.RequiredPermissions[0] != schedulercontract.ScheduledTaskReadPermission.String() {
		t.Fatalf("unexpected scheduler quick link permissions: %#v", link.RequiredPermissions)
	}
}

func assertRegisteredSchedulerMessage(t *testing.T, localizer *i18n.Service, locale i18n.LocaleTag, key string, expected string) {
	t.Helper()

	matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key))
	if len(matches) != 1 {
		t.Fatalf("expected one scheduler message for %s %q, got %#v", locale, key, matches)
	}
	if matches[0].Text != expected {
		t.Fatalf("expected scheduler message %q for %s %q, got %#v", expected, locale, key, matches[0])
	}
}

type schedulerNotificationPublisherRecorder struct {
	inputs []moduleapi.PublishNotificationInput
	err    error
}

func (r *schedulerNotificationPublisherRecorder) Publish(_ context.Context, input moduleapi.PublishNotificationInput) (moduleapi.PublishNotificationResult, error) {
	r.inputs = append(r.inputs, input)
	return moduleapi.PublishNotificationResult{}, r.err
}

func assertSchedulerRunSuccessDisplay(t *testing.T, input moduleapi.PublishNotificationInput) {
	t.Helper()
	if input.TitleKey != "notification.title.scheduler.runSucceeded" ||
		input.MessageKey != "notification.message.scheduler.runSucceeded" ||
		input.CategoryKey != "notification.category.task" ||
		input.SourceKey != "notification.source.scheduler" ||
		input.LevelKey != "notification.level.info" ||
		input.EventTypeKey != "notification.event.taskSucceeded" ||
		input.ResourceTypeKey != "notification.resourceType.scheduledTaskRun" ||
		input.ActionLabelKey != "notification.action.openRunRecord" {
		t.Fatalf("unexpected message keys: %#v", input)
	}
	if input.Title != "Webhook Health" ||
		input.Message != "Completed successfully." ||
		input.ActionLabel != "Open scheduled task run" {
		t.Fatalf("unexpected fallback copy: %#v", input)
	}
}

func assertSchedulerRunSuccessRouting(t *testing.T, input moduleapi.PublishNotificationInput) {
	t.Helper()
	if input.EventType != "task_succeeded" ||
		input.Severity != "info" ||
		input.Category != "TASK" ||
		input.SourceModule != moduleID ||
		input.ResourceType != "scheduled_task_run" ||
		input.ResourceID != "99" ||
		input.Navigation.Kind != "SCHEDULER_RUN" ||
		input.Target.Type != "USER" ||
		input.Target.Ref != "42" ||
		input.DedupeKey != "scheduler:run_succeeded:99" {
		t.Fatalf("unexpected success notification input: %#v", input)
	}
}

func assertSchedulerRunPayload(t *testing.T, payload json.RawMessage, runID uint64, taskKey string, jobKey string) {
	t.Helper()
	var decoded struct {
		RunID   uint64 `json:"run_id"`
		TaskKey string `json:"task_key"`
		JobKey  string `json:"job_key"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode scheduler run payload: %v", err)
	}
	if decoded.RunID != runID || decoded.TaskKey != taskKey || decoded.JobKey != jobKey {
		t.Fatalf("unexpected scheduler run payload: %#v", decoded)
	}
}

type expectedSchedulerRunSuccessMetadata struct {
	runID         uint64
	taskTitleKey  string
	taskName      string
	taskBuiltin   bool
	jobKey        string
	jobTitleKey   string
	triggerType   string
	resultSummary string
}

func assertSchedulerRunSuccessMetadata(t *testing.T, payload json.RawMessage, expected expectedSchedulerRunSuccessMetadata) {
	t.Helper()
	var decoded struct {
		RunID         uint64 `json:"runId"`
		TaskTitleKey  string `json:"taskTitleKey"`
		TaskName      string `json:"taskName"`
		TaskTitle     string `json:"taskTitle"`
		TaskBuiltin   bool   `json:"taskBuiltin"`
		JobKey        string `json:"jobKey"`
		JobTitleKey   string `json:"jobTitleKey"`
		TriggerType   string `json:"triggerType"`
		ResultSummary string `json:"resultSummary"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode scheduler success metadata: %v", err)
	}
	if decoded.RunID != expected.runID ||
		decoded.TaskTitleKey != expected.taskTitleKey ||
		decoded.TaskBuiltin != expected.taskBuiltin ||
		decoded.JobKey != expected.jobKey ||
		decoded.JobTitleKey != expected.jobTitleKey ||
		decoded.TriggerType != expected.triggerType ||
		decoded.ResultSummary != expected.resultSummary {
		t.Fatalf("unexpected scheduler success metadata: %#v", decoded)
	}
	if decoded.TaskName != expected.taskName || decoded.TaskTitle != expected.taskName {
		t.Fatalf("unexpected scheduler success task title metadata: %#v", decoded)
	}
}

func TestSchedulerTaskAttentionDashboardWidgetLoadsAttentionPayload(t *testing.T) {
	finishedAt := time.Now().UTC()
	payload, err := loadSchedulerTaskAttentionWidget(context.Background(), &schedulerAPIRuntime{
		tasks: []schedulercore.TaskSnapshot{
			{
				Key:     "failed",
				Enabled: true,
				LastRun: &schedulercore.TaskRun{
					Status:     schedulercore.RunStatusFailed,
					FinishedAt: &finishedAt,
				},
			},
			{Key: "running", Enabled: true, Running: true},
			{Key: "disabled", Enabled: false},
		},
	})
	if err != nil {
		t.Fatalf("load scheduler task attention widget: %v", err)
	}
	items, ok := payload["items"].([]map[string]any)
	if !ok {
		t.Fatalf("expected stat-group items payload, got %#v", payload["items"])
	}
	if len(items) != 3 {
		t.Fatalf("expected three scheduler attention stats, got %d", len(items))
	}
	for _, item := range items {
		if item["value"] != "1" {
			t.Fatalf("expected each attention stat to count one task, got %#v", items)
		}
		if item["route_location"] != schedulercontract.ScheduledTaskMenuPath {
			t.Fatalf("expected scheduler route on stat item, got %#v", item)
		}
	}
}

func TestSchedulerTaskAttentionDashboardWidgetPaginatesAllTasks(t *testing.T) {
	finishedAt := time.Now().UTC()
	tasks := make([]schedulercore.TaskSnapshot, schedulerTaskAttentionListLimit+1)
	for index := range tasks {
		tasks[index] = schedulercore.TaskSnapshot{Key: "task-" + strconv.Itoa(index), Enabled: true}
	}
	tasks[schedulerTaskAttentionListLimit] = schedulercore.TaskSnapshot{
		Key:     "failed-after-first-page",
		Enabled: true,
		LastRun: &schedulercore.TaskRun{
			Status:     schedulercore.RunStatusFailed,
			FinishedAt: &finishedAt,
		},
	}
	runtime := &schedulerAPIRuntime{tasks: tasks}

	payload, err := loadSchedulerTaskAttentionWidget(context.Background(), runtime)
	if err != nil {
		t.Fatalf("load scheduler task attention widget: %v", err)
	}

	if len(runtime.listTaskQueries) != 2 {
		t.Fatalf("expected two paginated task list queries, got %#v", runtime.listTaskQueries)
	}
	if runtime.listTaskQueries[0].Offset != 0 || runtime.listTaskQueries[1].Offset != schedulerTaskAttentionListLimit {
		t.Fatalf("unexpected task list pagination offsets: %#v", runtime.listTaskQueries)
	}
	items, ok := payload["items"].([]map[string]any)
	if !ok {
		t.Fatalf("expected stat-group items payload, got %#v", payload["items"])
	}
	if items[0]["value"] != "1" {
		t.Fatalf("expected failed count from second page, got %#v", items)
	}
}

func TestSchedulerTaskAttentionStateUsesNormalPriorityWhenHidden(t *testing.T) {
	state, priority := schedulerAttentionState(schedulerAttentionCounters{})

	if state != dashboard.WidgetStateHidden {
		t.Fatalf("expected hidden state for empty attention counters, got %q", state)
	}
	if priority != dashboard.WidgetPriorityNormal {
		t.Fatalf("expected hidden attention widget to use normal priority, got %q", priority)
	}
}

func TestScheduledTaskListRouteReturnsRuntimeTasks(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	moduleInstance := NewModule()
	moduleInstance.runtime = &schedulerAPIRuntime{
		tasks: []schedulercore.TaskSnapshot{
			{
				ID:             1,
				Key:            "audit.retention.cleanup",
				JobKey:         "audit.audit-log-retention-cleanup",
				TitleKey:       "scheduledTask.auditLogRetention.title",
				Title:          "Audit retention cleanup",
				DescriptionKey: "scheduledTask.auditLogRetention.description",
				Schedule:       "0 0 * * * *",
				Enabled:        true,
				ConfigJSON:     "{}",
				ConfigSource:   "system",
				JobDefinition: &schedulercore.JobDefinitionSnapshot{
					JobKey:        "audit.audit-log-retention-cleanup",
					ModuleKey:     "audit",
					Category:      "retention",
					TitleKey:      "scheduledTask.auditLogRetention.title",
					Title:         "Audit retention cleanup",
					ShortTitleKey: "scheduler.job.shortTitle.auditLog",
					ShortTitle:    "Audit Log",
				},
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
		},
	}

	if err := moduleInstance.Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}
	ctx.LifecycleContext = context.Background()
	if err := moduleInstance.Boot(ctx); err != nil {
		t.Fatalf("boot module: %v", err)
	}
	t.Cleanup(func() {
		_ = moduleInstance.Shutdown(ctx)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/scheduled-tasks?limit=1&offset=0", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	payload := decodeScheduledTaskListPayload(t, recorder.Body.Bytes())
	if !payload.Success || payload.Data.Total != 1 || payload.Data.Limit != 1 || payload.Data.Offset != 0 || len(payload.Data.Items) != 1 {
		t.Fatalf("unexpected scheduled task list payload: %#v", payload)
	}
	assertScheduledTaskListItem(t, payload.Data.Items[0])
}

func TestScheduledTaskJobDefinitionsRouteReturnsRuntimeJobDefinitions(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	moduleInstance := NewModule()
	moduleInstance.runtime = &schedulerAPIRuntime{
		jobDefinitions: []schedulercore.JobDefinitionSnapshot{
			{
				JobKey:         "audit.audit-log-retention-cleanup",
				ModuleKey:      "audit",
				Category:       "retention",
				TitleKey:       "scheduledTask.auditLogRetention.title",
				Title:          "Retention Cleanup",
				ShortTitleKey:  "scheduler.job.shortTitle.auditLog",
				ShortTitle:     "Audit Log",
				DescriptionKey: "scheduledTask.auditLogRetention.description",
				Description:    "Clean audit logs",
				ConfigSchema:   `{"type":"object"}`,
				DefaultConfig:  `{"retention_days":30}`,
				DefaultCron:    "0 0 * * * *",
				Enabled:        true,
				Actions: []schedulercore.JobActionSnapshot{
					{
						Key:            "dryRun",
						TitleKey:       "scheduledTask.action.dryRun.title",
						Title:          "试运行",
						DescriptionKey: "scheduledTask.action.dryRun.description",
						Description:    "预览本次执行结果",
					},
				},
			},
		},
	}
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	recorder := performSchedulerRequest(engine, http.MethodGet, "/api/scheduled-tasks/job-definitions", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	payload := decodeScheduledTaskJobDefinitionListPayload(t, recorder.Body.Bytes())
	if !payload.Success || payload.Data.Total != 1 || len(payload.Data.Items) != 1 {
		t.Fatalf("unexpected job definition list payload: %#v", payload)
	}
	assertScheduledTaskJobDefinitionListItem(t, payload.Data.Items[0])
}

func TestScheduledTaskJobDefinitionDetailRouteReturnsRuntimeJobDefinition(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	moduleInstance := NewModule()
	moduleInstance.runtime = &schedulerAPIRuntime{
		jobDefinitions: []schedulercore.JobDefinitionSnapshot{
			{
				JobKey:         "audit.audit-log-retention-cleanup",
				ModuleKey:      "audit",
				Category:       "retention",
				TitleKey:       "scheduledTask.auditLogRetention.title",
				Title:          "Retention Cleanup",
				ShortTitleKey:  "scheduler.job.shortTitle.auditLog",
				ShortTitle:     "Audit Log",
				DescriptionKey: "scheduledTask.auditLogRetention.description",
				Description:    "Clean audit logs",
				ConfigSchema:   `{"type":"object","properties":{"batchSize":{"type":"integer","x-i18n":{"titleKey":"scheduledTask.auditLogRetention.config.batchSize.title"}}},"additionalProperties":false}`,
				DefaultConfig:  `{"batchSize":1000}`,
				DefaultCron:    "0 0 * * * *",
				Enabled:        true,
			},
		},
	}
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	recorder := performSchedulerRequest(engine, http.MethodGet, "/api/scheduled-tasks/job-definitions/audit.audit-log-retention-cleanup", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	payload := decodeScheduledTaskJobDefinitionDetailPayload(t, recorder.Body.Bytes())
	if !payload.Success ||
		payload.Data.JobKey != "audit.audit-log-retention-cleanup" ||
		payload.Data.ConfigSchema == "" ||
		payload.Data.DefaultConfig != `{"batchSize":1000}` {
		t.Fatalf("unexpected job definition detail payload: %#v", payload)
	}
}

func TestScheduledTaskCreateRouteCreatesJobTask(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	runtimeRecorder := &schedulerAPIRuntime{}
	moduleInstance := NewModule()
	moduleInstance.runtime = runtimeRecorder
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	recorder := performSchedulerRequest(engine, http.MethodPost, "/api/scheduled-tasks", `{
		"task_key": "audit.retention.nightly",
		"job_key": "audit.audit-log-retention-cleanup",
		"title": "Nightly audit cleanup",
		"description": "Cleans audit logs",
		"cron_expression": "*/5 * * * * *",
		"enabled": true,
		"config_json": "{\"retention_days\":30}"
	}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if len(runtimeRecorder.createInputs) != 1 {
		t.Fatalf("expected one create call, got %d", len(runtimeRecorder.createInputs))
	}
	input := runtimeRecorder.createInputs[0]
	if input.TaskKey != "audit.retention.nightly" ||
		input.JobKey != "audit.audit-log-retention-cleanup" ||
		input.ConfigJSON != `{"retention_days":30}` ||
		!input.EnabledSet ||
		!input.Enabled {
		t.Fatalf("unexpected create input: %#v", input)
	}
}

func TestScheduledTaskCreateRouteRejectsMissingJobKey(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	runtimeRecorder := &schedulerAPIRuntime{}
	moduleInstance := NewModule()
	moduleInstance.runtime = runtimeRecorder
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	recorder := performSchedulerRequest(engine, http.MethodPost, "/api/scheduled-tasks", `{
		"task_key": "audit.retention.nightly",
		"job_key": "",
		"title": "Nightly audit cleanup",
		"cron_expression": "*/5 * * * * *",
		"enabled": true
	}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if len(runtimeRecorder.createInputs) != 0 {
		t.Fatalf("expected missing job key to skip runtime create, got %d calls", len(runtimeRecorder.createInputs))
	}
}

func TestScheduledTaskCreateRouteReturnsFieldErrorsForConflicts(t *testing.T) {
	for _, tc := range []struct {
		name          string
		err           error
		expectedField string
	}{
		{name: "task key", err: schedulercore.ErrTaskKeyConflict, expectedField: "task_key"},
		{name: "title", err: schedulercore.ErrTaskTitleConflict, expectedField: "title"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, engine := newModuleTestContextWithEngine()
			moduleInstance := NewModule()
			moduleInstance.runtime = &schedulerAPIRuntime{createErr: tc.err}
			registerAndBootSchedulerModule(t, ctx, moduleInstance)

			recorder := performSchedulerRequest(engine, http.MethodPost, "/api/scheduled-tasks", `{
				"task_key": "audit.retention.nightly",
				"job_key": "audit.audit-log-retention-cleanup",
				"title": "Nightly audit cleanup",
				"cron_expression": "*/5 * * * * *",
				"enabled": true
			}`)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d with body %s", recorder.Code, recorder.Body.String())
			}
			payload := decodeSchedulerErrorPayload(t, recorder.Body.Bytes())
			if payload.Data.Field != tc.expectedField {
				t.Fatalf("expected field %q, got %#v", tc.expectedField, payload)
			}
		})
	}
}

func TestScheduledTaskUpdateSystemTaskAllowsCronAndEnabledOnly(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	ctx.CronRegistry.Register(cronx.Job{
		Name:           "builtin-cleanup",
		Module:         "scheduler",
		Schedule:       "*/5 * * * * *",
		DefaultConfig:  `{"retentionDays":30,"batchSize":1000}`,
		ConfigSchema:   `{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":3650},"batchSize":{"type":"integer","minimum":1,"maximum":10000}},"additionalProperties":false}`,
		DefaultEnabled: true,
		Run:            func(context.Context) error { return nil },
	})
	moduleInstance := NewModule()
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	recorder := performSchedulerRequest(engine, http.MethodPut, "/api/scheduled-tasks/builtin-cleanup", `{
		"cron_expression": "*/10 * * * * *",
		"enabled": false,
		"config_json": "{\"retentionDays\":90,\"batchSize\":500}"
	}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200 for cron/enabled/config update, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	recorder = performSchedulerRequest(engine, http.MethodPut, "/api/scheduled-tasks/builtin-cleanup", `{
		"title": "Changed title"
	}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for builtin title update, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestScheduledTaskDeleteBuiltinRejectsBadRequest(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	moduleInstance := NewModule()
	moduleInstance.runtime = &schedulerAPIRuntime{deleteErr: schedulercore.ErrTaskImmutable}
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	recorder := performSchedulerRequest(engine, http.MethodDelete, "/api/scheduled-tasks/system.cleanup", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestScheduledTaskEnableDisableRoutesUseEnablePermissionAndRuntime(t *testing.T) {
	authorizer := &recordingAuthorizer{}
	ctx, engine := newModuleTestContextWithEngineAndAuthorizer(authorizer)
	runtimeRecorder := &schedulerAPIRuntime{}
	moduleInstance := NewModule()
	moduleInstance.runtime = runtimeRecorder
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	enable := performSchedulerRequest(engine, http.MethodPost, "/api/scheduled-tasks/webhook.health/enable", "")
	disable := performSchedulerRequest(engine, http.MethodPost, "/api/scheduled-tasks/webhook.health/disable", "")
	if enable.Code != http.StatusOK || disable.Code != http.StatusOK {
		t.Fatalf("expected enable/disable status 200, got %d/%d", enable.Code, disable.Code)
	}
	if len(runtimeRecorder.setEnabledVals) != 2 || !runtimeRecorder.setEnabledVals[0] || runtimeRecorder.setEnabledVals[1] {
		t.Fatalf("unexpected enabled calls: %#v", runtimeRecorder.setEnabledVals)
	}
	if len(authorizer.permissions) < 2 {
		t.Fatalf("expected permission checks, got %#v", authorizer.permissions)
	}
	lastTwo := authorizer.permissions[len(authorizer.permissions)-2:]
	if lastTwo[0] != schedulercontract.ScheduledTaskEnablePermission.String() ||
		lastTwo[1] != schedulercontract.ScheduledTaskEnablePermission.String() {
		t.Fatalf("expected enable permission for enable/disable, got %#v", authorizer.permissions)
	}
}

func TestScheduledTaskManualRunUsesSlashRunRoute(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	runtimeRecorder := &schedulerAPIRuntime{}
	moduleInstance := NewModule()
	moduleInstance.runtime = runtimeRecorder
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	wrongPath := performSchedulerRequest(engine, http.MethodPost, "/api/scheduled-tasks/webhook.health:run", "")
	if wrongPath.Code == http.StatusOK {
		t.Fatalf("expected colon run route to stay unregistered, got body %s", wrongPath.Body.String())
	}
	if len(runtimeRecorder.runOnceKeys) != 0 {
		t.Fatalf("expected colon run path not to invoke runtime, got %#v", runtimeRecorder.runOnceKeys)
	}

	recorder := performSchedulerRequest(engine, http.MethodPost, "/api/scheduled-tasks/webhook.health/run", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected slash run route status 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if len(runtimeRecorder.runOnceKeys) != 1 || runtimeRecorder.runOnceKeys[0] != "webhook.health" {
		t.Fatalf("unexpected run once keys: %#v", runtimeRecorder.runOnceKeys)
	}
	if len(runtimeRecorder.runOnceTriggers) != 1 ||
		runtimeRecorder.runOnceTriggers[0].Type != schedulercore.TriggerTypeManual ||
		runtimeRecorder.runOnceTriggers[0].TriggerUserID != 7 {
		t.Fatalf("unexpected run once triggers: %#v", runtimeRecorder.runOnceTriggers)
	}
}

func TestSchedulerManualTriggerUserIDReadsRequestAuthContext(t *testing.T) {
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/api/scheduled-tasks/webhook.health/run", nil)
	request = request.WithContext(moduleapi.WithRequestAuthContext(request.Context(), moduleapi.RequestAuthContext{
		User: &moduleapi.CurrentUser{ID: 42, Username: "alice"},
	}))
	ginCtx.Request = request

	if got := schedulerManualTriggerUserID(ginCtx); got != 42 {
		t.Fatalf("expected current user id 42, got %d", got)
	}

	missingRecorder := httptest.NewRecorder()
	missingCtx, _ := gin.CreateTestContext(missingRecorder)
	missingCtx.Request = httptest.NewRequest(http.MethodPost, "/api/scheduled-tasks/webhook.health/run", nil)
	if got := schedulerManualTriggerUserID(missingCtx); got != 0 {
		t.Fatalf("expected missing current user to produce zero trigger user id, got %d", got)
	}
}

func TestScheduledTaskActionRouteRunsDryRunAction(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	runtimeRecorder := &schedulerAPIRuntime{}
	moduleInstance := NewModule()
	moduleInstance.runtime = runtimeRecorder
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	recorder := performSchedulerRequest(engine, http.MethodPost, "/api/scheduled-tasks/webhook.health/actions/dryRun", `{
		"config_json": {"batchSize": 25}
	}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected action route status 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if len(runtimeRecorder.actionTaskKeys) != 1 ||
		runtimeRecorder.actionTaskKeys[0] != "webhook.health" ||
		runtimeRecorder.actionKeys[0] != "dryRun" ||
		runtimeRecorder.actionConfigs[0] != `{"batchSize":25}` {
		t.Fatalf("unexpected action runtime calls: task=%#v action=%#v config=%#v", runtimeRecorder.actionTaskKeys, runtimeRecorder.actionKeys, runtimeRecorder.actionConfigs)
	}
	payload := decodeScheduledTaskActionPayload(t, recorder.Body.Bytes())
	if payload.Data.ActionKey != "dryRun" ||
		payload.Data.Result.Summary != "previewed" ||
		payload.Data.ResultJSON == "" {
		t.Fatalf("unexpected action payload: %#v", payload.Data)
	}
}

func TestScheduledTaskActionRouteReturnsNotFoundForUnknownAction(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	moduleInstance := NewModule()
	moduleInstance.runtime = &schedulerAPIRuntime{actionErr: schedulercore.ErrJobActionNotFound}
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	recorder := performSchedulerRequest(engine, http.MethodPost, "/api/scheduled-tasks/webhook.health/actions/missing", `{
		"config_json": {}
	}`)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func TestScheduledTaskRunDetailReturnsResultAndErrorFields(t *testing.T) {
	startedAt := time.Now().UTC().Add(-time.Second)
	finishedAt := time.Now().UTC()
	duration := int64(1000)
	ctx, engine := newModuleTestContextWithEngine()
	moduleInstance := NewModule()
	moduleInstance.runtime = &schedulerAPIRuntime{
		getRunResult: schedulercore.TaskRun{
			ID:            42,
			TaskKey:       "webhook.health",
			JobKey:        "scheduler.webhook-health",
			TaskTitle:     "Webhook health",
			JobTitle:      "Webhook health",
			JobShortTitle: "Webhook",
			JobCategory:   "custom",
			ModuleKey:     "scheduler",
			TriggerType:   schedulercore.TriggerTypeManual,
			Status:        schedulercore.RunStatusFailed,
			ErrorMessage:  "http status 500",
			Result:        "HTTP 500 failed",
			StartedAt:     startedAt,
			FinishedAt:    &finishedAt,
			DurationMS:    &duration,
			CreatedAt:     startedAt,
		},
	}
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	recorder := performSchedulerRequest(engine, http.MethodGet, "/api/scheduled-tasks/runs/42", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	payload := decodeScheduledTaskRunPayload(t, recorder.Body.Bytes())
	if payload.Data.ID != 42 || payload.Data.ResultSummary == nil || *payload.Data.ResultSummary != "HTTP 500 failed" || payload.Data.ErrorMessage != "http status 500" {
		t.Fatalf("unexpected run detail payload: %#v", payload.Data)
	}
}

type scheduledTaskListPayload struct {
	Success bool `json:"success"`
	Data    struct {
		Total  int                            `json:"total"`
		Limit  int                            `json:"limit"`
		Offset int                            `json:"offset"`
		Items  []scheduledTaskListItemPayload `json:"items"`
	} `json:"data"`
}

type schedulerErrorPayload struct {
	Success bool `json:"success"`
	Data    struct {
		Field string `json:"field"`
	} `json:"data"`
}

func decodeSchedulerErrorPayload(t *testing.T, body []byte) schedulerErrorPayload {
	t.Helper()
	var payload schedulerErrorPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode scheduler error payload: %v", err)
	}
	return payload
}

type scheduledTaskListItemPayload struct {
	TaskKey  string `json:"task_key"`
	JobKey   string `json:"job_key"`
	TitleKey string `json:"title_key"`
	Job      struct {
		ModuleKey     string `json:"module_key"`
		Category      string `json:"category"`
		ShortTitleKey string `json:"short_title_key"`
	} `json:"job"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
	Running bool   `json:"running"`
}

func assertScheduledTaskListItem(t *testing.T, item scheduledTaskListItemPayload) {
	t.Helper()

	if item.TaskKey != "audit.retention.cleanup" ||
		item.JobKey != "audit.audit-log-retention-cleanup" ||
		item.TitleKey != "scheduledTask.auditLogRetention.title" ||
		item.Job.ModuleKey != "audit" ||
		item.Job.Category != "retention" ||
		item.Job.ShortTitleKey != "scheduler.job.shortTitle.auditLog" ||
		!item.Enabled ||
		item.Status != "idle" ||
		item.Running {
		t.Fatalf("unexpected scheduled task item: %#v", item)
	}
}

func decodeScheduledTaskListPayload(t *testing.T, body []byte) scheduledTaskListPayload {
	t.Helper()

	var payload scheduledTaskListPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return payload
}

type scheduledTaskJobDefinitionListPayload struct {
	Success bool `json:"success"`
	Data    struct {
		Total int `json:"total"`
		Items []struct {
			JobKey        string `json:"job_key"`
			ModuleKey     string `json:"module_key"`
			Category      string `json:"category"`
			TitleKey      string `json:"title_key"`
			ShortTitleKey string `json:"short_title_key"`
			DefaultCron   string `json:"default_cron"`
			DefaultConfig string `json:"default_config"`
			Actions       []struct {
				Key            string `json:"key"`
				TitleKey       string `json:"title_key"`
				Title          string `json:"title"`
				DescriptionKey string `json:"description_key"`
				Description    string `json:"description"`
			} `json:"actions"`
		} `json:"items"`
	} `json:"data"`
}

func decodeScheduledTaskJobDefinitionListPayload(t *testing.T, body []byte) scheduledTaskJobDefinitionListPayload {
	t.Helper()

	var payload scheduledTaskJobDefinitionListPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return payload
}

func assertScheduledTaskJobDefinitionListItem(
	t *testing.T,
	item struct {
		JobKey        string `json:"job_key"`
		ModuleKey     string `json:"module_key"`
		Category      string `json:"category"`
		TitleKey      string `json:"title_key"`
		ShortTitleKey string `json:"short_title_key"`
		DefaultCron   string `json:"default_cron"`
		DefaultConfig string `json:"default_config"`
		Actions       []struct {
			Key            string `json:"key"`
			TitleKey       string `json:"title_key"`
			Title          string `json:"title"`
			DescriptionKey string `json:"description_key"`
			Description    string `json:"description"`
		} `json:"actions"`
	},
) {
	t.Helper()

	if item.JobKey != "audit.audit-log-retention-cleanup" ||
		item.ModuleKey != "audit" ||
		item.Category != "retention" ||
		item.TitleKey != "scheduledTask.auditLogRetention.title" ||
		item.ShortTitleKey != "scheduler.job.shortTitle.auditLog" ||
		item.DefaultCron != "0 0 * * * *" ||
		item.DefaultConfig != `{"retention_days":30}` {
		t.Fatalf("unexpected job definition item: %#v", item)
	}
	if len(item.Actions) != 1 ||
		item.Actions[0].Key != "dryRun" ||
		item.Actions[0].TitleKey != "scheduledTask.action.dryRun.title" ||
		item.Actions[0].DescriptionKey != "scheduledTask.action.dryRun.description" {
		t.Fatalf("unexpected job definition actions: %#v", item.Actions)
	}
}

type scheduledTaskJobDefinitionDetailPayload struct {
	Success bool `json:"success"`
	Data    struct {
		JobKey        string `json:"job_key"`
		ConfigSchema  string `json:"config_schema"`
		DefaultConfig string `json:"default_config"`
	} `json:"data"`
}

func decodeScheduledTaskJobDefinitionDetailPayload(t *testing.T, body []byte) scheduledTaskJobDefinitionDetailPayload {
	t.Helper()

	var payload scheduledTaskJobDefinitionDetailPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return payload
}

type scheduledTaskRunPayload struct {
	Success bool `json:"success"`
	Data    struct {
		ID            uint64  `json:"id"`
		ErrorMessage  string  `json:"error_message"`
		ResultSummary *string `json:"result_summary"`
	} `json:"data"`
}

type scheduledTaskActionPayload struct {
	Success bool `json:"success"`
	Data    struct {
		ActionKey  string `json:"action_key"`
		ResultJSON string `json:"result_json"`
		Result     struct {
			Summary string `json:"summary"`
			Stage   string `json:"stage"`
		} `json:"result"`
	} `json:"data"`
}

func decodeScheduledTaskActionPayload(t *testing.T, body []byte) scheduledTaskActionPayload {
	t.Helper()

	var payload scheduledTaskActionPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return payload
}

func decodeScheduledTaskRunPayload(t *testing.T, body []byte) scheduledTaskRunPayload {
	t.Helper()

	var payload scheduledTaskRunPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return payload
}

// TestBootRejectsInvalidJobs 验证 scheduler 模块会在 Boot 阶段拒绝非法任务声明。
func TestBootRejectsInvalidJobs(t *testing.T) {
	ctx := newModuleTestContext()
	ctx.CronRegistry.Register(cronx.Job{Name: "invalid", Schedule: "*/1 * * * * *"})

	moduleInstance := NewModule()
	if err := moduleInstance.Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}

	err := moduleInstance.Boot(&module.Context{
		LifecycleContext: context.Background(),
		CronRegistry:     ctx.CronRegistry,
		Logger:           ctx.Logger,
		Services:         ctx.Services,
	})
	if err == nil {
		t.Fatal("expected invalid job boot to fail")
	}
}

// TestBootRegistersJobsAddedAfterRegister 验证 scheduler 模块会在 Boot 阶段读取最终 registry，
// 而不是在 Register 阶段提前快照。
func TestBootRegistersJobsAddedAfterRegister(t *testing.T) {
	ctx := newModuleTestContext()
	ctx.CronRegistry.Register(cronx.Job{
		Name:     "first",
		Schedule: "*/1 * * * * *",
		Run:      func(context.Context) error { return nil },
	})

	lifecycleCtx := context.Background()
	runtimeRecorder := &stopContextRecorderRuntime{}
	moduleInstance := NewModule()
	moduleInstance.runtime = runtimeRecorder

	if err := moduleInstance.Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}

	ctx.CronRegistry.Register(cronx.Job{
		Name:     "second",
		Schedule: "*/1 * * * * *",
		Run:      func(context.Context) error { return nil },
	})

	if err := moduleInstance.Boot(&module.Context{
		LifecycleContext: lifecycleCtx,
		Logger:           ctx.Logger,
		CronRegistry:     ctx.CronRegistry,
		Services:         ctx.Services,
	}); err != nil {
		t.Fatalf("boot module: %v", err)
	}

	if len(runtimeRecorder.registeredJobs) != 2 {
		t.Fatalf("expected 2 registered jobs, got %d", len(runtimeRecorder.registeredJobs))
	}
	if runtimeRecorder.registeredJobs[0].Name != "first" || runtimeRecorder.registeredJobs[1].Name != "second" {
		t.Fatalf("expected boot to register final registry snapshot, got %q then %q", runtimeRecorder.registeredJobs[0].Name, runtimeRecorder.registeredJobs[1].Name)
	}
	if runtimeRecorder.startCtx != lifecycleCtx {
		t.Fatal("expected boot to pass lifecycle context into scheduler runtime start")
	}
}

// TestBootRunsRegisteredJobs 验证 scheduler 模块会在 Boot 后驱动 registry 中的任务执行。
func TestBootRunsRegisteredJobs(t *testing.T) {
	ctx := newModuleTestContext()
	triggered := make(chan struct{}, 1)
	lifecycleCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx.CronRegistry.Register(cronx.Job{
		Name:           "heartbeat",
		Schedule:       "*/1 * * * * *",
		DefaultEnabled: true,
		Module:         "test",
		Run: func(context.Context) error {
			select {
			case triggered <- struct{}{}:
			default:
			}
			return nil
		},
	})

	moduleInstance := NewModule()
	if err := moduleInstance.Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}
	ctx.LifecycleContext = lifecycleCtx
	if err := moduleInstance.Boot(ctx); err != nil {
		t.Fatalf("boot module: %v", err)
	}
	defer func() {
		_ = moduleInstance.Shutdown(ctx)
	}()

	select {
	case <-triggered:
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("expected scheduled job to run after boot")
	}
}

// TestShutdownUsesLifecycleContext 验证 scheduler 模块会把生命周期关闭上下文
// 传递给底层 runtime，而不是回退到脱离宿主约束的全新 Background。
func TestShutdownUsesLifecycleContext(t *testing.T) {
	lifecycleCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runtimeRecorder := &stopContextRecorderRuntime{}
	moduleInstance := NewModule()
	moduleInstance.runtime = runtimeRecorder

	if err := moduleInstance.Shutdown(&module.Context{LifecycleContext: lifecycleCtx}); err != nil {
		t.Fatalf("shutdown module: %v", err)
	}
	if runtimeRecorder.stopCtx != lifecycleCtx {
		t.Fatal("expected scheduler shutdown to forward lifecycle context")
	}
}
