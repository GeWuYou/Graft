package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func (r *stopContextRecorderRuntime) ListRuns(context.Context, schedulercore.RunListQuery) (schedulercore.RunListResult, error) {
	return schedulercore.RunListResult{}, nil
}

func (r *stopContextRecorderRuntime) RunOnce(context.Context, string) (schedulercore.TaskRun, error) {
	return schedulercore.TaskRun{}, nil
}

type schedulerAPIRuntime struct {
	stopContextRecorderRuntime
	tasks []schedulercore.TaskSnapshot
}

func (r *schedulerAPIRuntime) ListTasks(context.Context) ([]schedulercore.TaskSnapshot, error) {
	return r.tasks, nil
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

func newModuleTestContext() *module.Context {
	ctx, _ := newModuleTestContextWithEngine()
	return ctx
}

func newModuleTestContextWithEngine() (*module.Context, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE scheduler_task_runs (
		id integer PRIMARY KEY AUTOINCREMENT,
		task_key text NOT NULL,
		task_name text NOT NULL DEFAULT '',
		owner text NOT NULL DEFAULT '',
		module text NOT NULL DEFAULT '',
		task_type text NOT NULL DEFAULT 'cron',
		trigger_type text NOT NULL,
		status text NOT NULL,
		error text NOT NULL DEFAULT '',
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
		return allowAllAuthorizer{}, nil
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
				Type:                  cronx.TaskTypeCron,
				DisplayMessageKey:     "audit.retention.title",
				DescriptionMessageKey: "audit.retention.description",
				Schedule:              "0 0 * * * *",
				DefaultEnabled:        true,
			},
		},
	}

	if err := moduleInstance.Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/scheduled-tasks", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
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
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Success || payload.Data.Total != 1 || len(payload.Data.Items) != 1 {
		t.Fatalf("unexpected scheduled task list payload: %#v", payload)
	}
	item := payload.Data.Items[0]
	if item.Key != "audit.retention.cleanup" ||
		item.TaskType != "cron" ||
		item.ScheduleType != "cron" ||
		item.DisplayNameKey != "audit.retention.title" ||
		item.Module != "audit" ||
		!item.Enabled ||
		item.Status != "idle" ||
		item.Running {
		t.Fatalf("unexpected scheduled task item: %#v", item)
	}
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
		Name:     "heartbeat",
		Schedule: "*/1 * * * * *",
		Module:   "test",
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
