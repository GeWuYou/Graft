// 浏览器验收确定性 fixture：向隔离的 PostgreSQL 环境写入 Application 任务
// 历史，供 task 浏览器套件在真实 API 上渲染。仅在设置
// GRAFT_BROWSER_SEED_DATABASE_URL 时执行；本文件是测试专用代码。
package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// browserFixtureOwnerType/ID 必须与 project 模块的 browser_seed_test.go 保持一致：
// owner_type 使用 Application 模块注册的 Task 授权类型，owner_id 使用同一公开应用标识。
const (
	browserFixtureOwnerType = "application"
	browserFixtureOwnerID   = "app_APPBR0WSERFXTR000000000000"
)

// browserFixtureTasks 是 task 浏览器套件消费的确定性任务历史。
var browserFixtureTasks = []struct {
	TaskType       string
	Status         string
	FailureCode    string
	FailureMessage string
	StageStatus    string
}{
	{TaskType: "application.compose.up.v1", Status: "success", StageStatus: "success"},
	{TaskType: "application.compose.up.v1", Status: "failed", FailureCode: "compose.up.failed", FailureMessage: "browser fixture compose up failed deterministically", StageStatus: "failed"},
}

// TestSeedBrowserFixture 写入确定性任务与阶段行；未提供种子数据库地址时跳过，
// 保证常规 `go test` 与 CI 不依赖任何真实数据库。
func TestSeedBrowserFixture(t *testing.T) {
	databaseURL := os.Getenv("GRAFT_BROWSER_SEED_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("GRAFT_BROWSER_SEED_DATABASE_URL is required for the browser acceptance fixture seed")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open browser fixture database: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping browser fixture database: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	for index, fixture := range browserFixtureTasks {
		var stageKey = "deploy"
		input, err := json.Marshal(map[string]any{})
		if err != nil {
			t.Fatalf("marshal browser fixture stage input: %v", err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("begin browser fixture task tx: %v", err)
		}
		var taskID int64
		err = tx.QueryRowContext(ctx, `
			INSERT INTO tasks
				(task_type, owner_type, owner_id, status, input_json, metadata_json, plan_json, state_json,
				 created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb, $8::jsonb, $9, $9)
			RETURNING id`,
			fixture.TaskType, browserFixtureOwnerType, browserFixtureOwnerID, fixture.Status,
			string(input), "{}", browserFixturePlanJSON(), "{}", now).Scan(&taskID)
		if err != nil {
			_ = tx.Rollback()
			t.Fatalf("seed browser fixture task %d: %v", index, err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO task_stages
				(task_id, stage_key, sequence, executor_type, status, attempt, max_attempts,
				 retry_backoff_ms, input_json, recovery_policy, result_json, failure_code, failure_message,
				 started_at, finished_at, created_at, updated_at)
			VALUES ($1, $2, 1, 'local.compose', $3, 1, 1, 0, $4::jsonb, 'retry_if_idempotent', '{}'::jsonb,
					NULLIF($5, ''), NULLIF($6, ''), $7, $7, $7, $7)`,
			taskID, stageKey, fixture.StageStatus, string(input),
			fixture.FailureCode, fixture.FailureMessage, now); err != nil {
			_ = tx.Rollback()
			t.Fatalf("seed browser fixture stage for task %d: %v", index, err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("commit browser fixture task %d: %v", index, err)
		}
	}
}

// browserFixturePlanJSON 冻结与任务阶段行一致的计划 JSON，保持计划与阶段事实对齐。
func browserFixturePlanJSON() string {
	plan := map[string]any{
		"Stages": []map[string]any{{
			"Key":            "deploy",
			"ExecutorType":   "local.compose",
			"MaxAttempts":    1,
			"Backoff":        0,
			"RecoveryPolicy": "retry_if_idempotent",
		}},
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

// TestBrowserFixtureTaskHistoryIsDeterministic 校验 fixture 任务事实在无需数据库时
// 保持确定性：owner 常量与公开应用标识契约一致，任务类型与阶段状态合法。
func TestBrowserFixtureTaskHistoryIsDeterministic(t *testing.T) {
	if browserFixtureOwnerType != "application" {
		t.Fatalf("fixture owner type = %q, want the Application module task owner type", browserFixtureOwnerType)
	}
	if len(browserFixtureOwnerID) != len("app_")+26 {
		t.Fatalf("fixture owner id %q must be a valid application id length", browserFixtureOwnerID)
	}
	for _, fixture := range browserFixtureTasks {
		if fixture.StageStatus != "success" && fixture.StageStatus != "failed" {
			t.Fatalf("fixture stage status %q must be a terminal deterministic state", fixture.StageStatus)
		}
		if (fixture.FailureCode == "") != (fixture.Status != "failed") {
			t.Fatalf("fixture task %q must pair failure metadata with the failed status", fixture.TaskType)
		}
	}
	if !json.Valid([]byte(browserFixturePlanJSON())) {
		t.Fatal("frozen browser fixture plan JSON must stay valid")
	}
}
