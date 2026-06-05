package scheduler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/cronx"
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

func (r *stopContextRecorderRuntime) ListTasks(context.Context) ([]schedulercore.TaskSnapshot, error) {
	return nil, nil
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

type schedulerAPIRuntime struct {
	stopContextRecorderRuntime
	tasks          []schedulercore.TaskSnapshot
	createInputs   []schedulercore.TaskMutation
	createResult   schedulercore.TaskSnapshot
	createErr      error
	updateInputs   []schedulercore.TaskMutation
	updateKeys     []string
	updateResult   schedulercore.TaskSnapshot
	updateErr      error
	deleteKeys     []string
	deleteErr      error
	setEnabledKeys []string
	setEnabledVals []bool
	setResult      schedulercore.TaskSnapshot
	setErr         error
	runOnceKeys    []string
	runOnceResult  schedulercore.TaskRun
	runOnceErr     error
	getRunID       uint64
	getRunResult   schedulercore.TaskRun
	getRunErr      error
}

func (r *schedulerAPIRuntime) ListTasks(context.Context) ([]schedulercore.TaskSnapshot, error) {
	return r.tasks, nil
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
			Key:      key,
			Type:     cronx.TaskTypeHTTP,
			Title:    key,
			Schedule: "*/5 * * * * *",
			Enabled:  enabled,
		}
	}
	r.setResult.Enabled = enabled
	return r.setResult, nil
}

func (r *schedulerAPIRuntime) RunOnce(_ context.Context, key string) (schedulercore.TaskRun, error) {
	r.runOnceKeys = append(r.runOnceKeys, key)
	if r.runOnceErr != nil {
		return schedulercore.TaskRun{}, r.runOnceErr
	}
	if r.runOnceResult.ID == 0 {
		r.runOnceResult = schedulercore.TaskRun{
			ID:          17,
			TaskKey:     key,
			TaskName:    key,
			TaskType:    cronx.TaskTypeHTTP,
			TriggerType: schedulercore.TriggerTypeManual,
			Status:      schedulercore.RunStatusSuccess,
			StartedAt:   time.Now().UTC(),
			CreatedAt:   time.Now().UTC(),
		}
	}
	return r.runOnceResult, nil
}

func (r *schedulerAPIRuntime) GetRun(_ context.Context, id uint64) (schedulercore.TaskRun, error) {
	r.getRunID = id
	return r.getRunResult, r.getRunErr
}

func taskSnapshotFromMutation(command schedulercore.TaskMutation) schedulercore.TaskSnapshot {
	return schedulercore.TaskSnapshot{
		Key:         command.TaskKey,
		Type:        command.TaskType,
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
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE scheduled_tasks (
		id integer PRIMARY KEY AUTOINCREMENT,
		task_key text NOT NULL UNIQUE,
		task_type text NOT NULL,
		title text NOT NULL DEFAULT '',
		description text NOT NULL DEFAULT '',
		cron_expression text NOT NULL,
		enabled boolean NOT NULL DEFAULT true,
		builtin boolean NOT NULL DEFAULT false,
		config_json text NOT NULL DEFAULT '{}',
		created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at datetime NULL
	);
	CREATE TABLE scheduler_task_runs (
		id integer PRIMARY KEY AUTOINCREMENT,
		task_key text NOT NULL,
		task_name text NOT NULL DEFAULT '',
		owner text NOT NULL DEFAULT '',
		module text NOT NULL DEFAULT '',
		task_type text NOT NULL DEFAULT 'cron',
		trigger_type text NOT NULL,
		status text NOT NULL,
		error text NOT NULL DEFAULT '',
		result_summary text NOT NULL DEFAULT '',
		error_message text NOT NULL DEFAULT '',
		started_at datetime NOT NULL,
		finished_at datetime NULL,
		duration_ms integer NULL,
		created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		panic(err)
	}
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
	}, engine
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

func TestScheduledTaskListRouteReturnsRuntimeTasks(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	moduleInstance := NewModule()
	moduleInstance.runtime = &schedulerAPIRuntime{
		tasks: []schedulercore.TaskSnapshot{
			{
				Key:                   "audit.retention.cleanup",
				Name:                  "audit-retention-cleanup",
				Owner:                 "audit",
				Module:                "audit",
				Type:                  cronx.TaskTypeSystem,
				DisplayMessageKey:     "audit.retention.title",
				DescriptionMessageKey: "audit.retention.description",
				Schedule:              "0 0 * * * *",
				Enabled:               true,
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

	request := httptest.NewRequest(http.MethodGet, "/api/scheduled-tasks", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	payload := decodeScheduledTaskListPayload(t, recorder.Body.Bytes())
	if !payload.Success || payload.Data.Total != 1 || len(payload.Data.Items) != 1 {
		t.Fatalf("unexpected scheduled task list payload: %#v", payload)
	}
	item := payload.Data.Items[0]
	if item.Key != "audit.retention.cleanup" ||
		item.TaskType != "system" ||
		item.ScheduleType != "cron" ||
		item.DisplayNameKey != "audit.retention.title" ||
		item.Module != "audit" ||
		!item.Enabled ||
		item.Status != "idle" ||
		item.Running {
		t.Fatalf("unexpected scheduled task item: %#v", item)
	}
}

func TestScheduledTaskCreateRouteCreatesHTTPTask(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	runtimeRecorder := &schedulerAPIRuntime{}
	moduleInstance := NewModule()
	moduleInstance.runtime = runtimeRecorder
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	recorder := performSchedulerRequest(engine, http.MethodPost, "/api/scheduled-tasks", `{
		"task_key": "webhook.health",
		"task_type": "http",
		"title": "Webhook health",
		"description": "Checks webhook health",
		"cron_expression": "*/5 * * * * *",
		"enabled": true,
		"config": {
			"method": "GET",
			"url": "https://example.com/health",
			"timeout_seconds": 10
		}
	}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if len(runtimeRecorder.createInputs) != 1 {
		t.Fatalf("expected one create call, got %d", len(runtimeRecorder.createInputs))
	}
	input := runtimeRecorder.createInputs[0]
	if input.TaskKey != "webhook.health" || input.TaskType != cronx.TaskTypeHTTP || !input.EnabledSet || !input.Enabled {
		t.Fatalf("unexpected create input: %#v", input)
	}
	if !strings.Contains(input.ConfigJSON, `"url":"https://example.com/health"`) {
		t.Fatalf("expected HTTP config JSON to include URL, got %s", input.ConfigJSON)
	}
}

func TestScheduledTaskCreateRouteRejectsNonHTTPTaskType(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	runtimeRecorder := &schedulerAPIRuntime{}
	moduleInstance := NewModule()
	moduleInstance.runtime = runtimeRecorder
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	recorder := performSchedulerRequest(engine, http.MethodPost, "/api/scheduled-tasks", `{
		"task_key": "system.task",
		"task_type": "system",
		"title": "System task",
		"cron_expression": "*/5 * * * * *",
		"enabled": true,
		"config": {
			"method": "GET",
			"url": "https://example.com/health"
		}
	}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if len(runtimeRecorder.createInputs) != 0 {
		t.Fatalf("expected non-http task type to skip runtime create, got %d calls", len(runtimeRecorder.createInputs))
	}
}

func TestScheduledTaskUpdateSystemTaskAllowsCronAndEnabledOnly(t *testing.T) {
	ctx, engine := newModuleTestContextWithEngine()
	ctx.CronRegistry.Register(cronx.Job{
		Name:           "builtin-cleanup",
		Module:         "scheduler",
		Schedule:       "*/5 * * * * *",
		DefaultEnabled: true,
		Run:            func(context.Context) error { return nil },
	})
	moduleInstance := NewModule()
	registerAndBootSchedulerModule(t, ctx, moduleInstance)
	defer func() {
		_ = moduleInstance.Shutdown(ctx)
	}()

	recorder := performSchedulerRequest(engine, http.MethodPut, "/api/scheduled-tasks/builtin-cleanup", `{
		"cron_expression": "*/10 * * * * *",
		"enabled": false
	}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200 for cron/enabled update, got %d with body %s", recorder.Code, recorder.Body.String())
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
}

func TestScheduledTaskRunDetailReturnsResultAndErrorFields(t *testing.T) {
	startedAt := time.Now().UTC().Add(-time.Second)
	finishedAt := time.Now().UTC()
	duration := int64(1000)
	ctx, engine := newModuleTestContextWithEngine()
	moduleInstance := NewModule()
	moduleInstance.runtime = &schedulerAPIRuntime{
		getRunResult: schedulercore.TaskRun{
			ID:          42,
			TaskKey:     "webhook.health",
			TaskName:    "Webhook health",
			Owner:       "scheduler",
			Module:      "scheduler",
			TaskType:    cronx.TaskTypeHTTP,
			TriggerType: schedulercore.TriggerTypeManual,
			Status:      schedulercore.RunStatusFailed,
			Error:       "http status 500",
			Result:      "HTTP 500 failed",
			StartedAt:   startedAt,
			FinishedAt:  &finishedAt,
			DurationMS:  &duration,
			CreatedAt:   startedAt,
		},
	}
	registerAndBootSchedulerModule(t, ctx, moduleInstance)

	recorder := performSchedulerRequest(engine, http.MethodGet, "/api/scheduled-tasks/runs/42", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	payload := decodeScheduledTaskRunPayload(t, recorder.Body.Bytes())
	if payload.Data.ID != 42 || payload.Data.ResultSummary == nil || *payload.Data.ResultSummary != "HTTP 500 failed" || payload.Data.ErrorSummary != "http status 500" {
		t.Fatalf("unexpected run detail payload: %#v", payload.Data)
	}
}

type scheduledTaskListPayload struct {
	Success bool `json:"success"`
	Data    struct {
		Total int `json:"total"`
		Items []struct {
			Key            string `json:"key"`
			TaskType       string `json:"task_type"`
			ScheduleType   string `json:"schedule_type"`
			DisplayNameKey string `json:"display_name_key"`
			Module         string `json:"module"`
			Enabled        bool   `json:"enabled"`
			Status         string `json:"status"`
			Running        bool   `json:"running"`
		} `json:"items"`
	} `json:"data"`
}

func decodeScheduledTaskListPayload(t *testing.T, body []byte) scheduledTaskListPayload {
	t.Helper()

	var payload scheduledTaskListPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return payload
}

type scheduledTaskRunPayload struct {
	Success bool `json:"success"`
	Data    struct {
		ID            uint64  `json:"id"`
		ErrorSummary  string  `json:"error_summary"`
		ResultSummary *string `json:"result_summary"`
	} `json:"data"`
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
