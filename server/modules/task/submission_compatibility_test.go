package task

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
	taskstore "graft/server/modules/task/store"
)

func TestAgentSubmissionReplaysPersistedPreUpgradeReceipt(t *testing.T) {
	// 该夹具由 346854ff5 的真实 buildTaskRecord 生成，避免用当前序列化规则重算历史指纹。
	encoded, err := os.ReadFile("testdata/legacy-agent-submission-346854ff5.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Input  moduleapi.SubmitTaskInput `json:"input"`
		Task   taskmodel.Task            `json:"task"`
		Stages []taskmodel.Stage         `json:"stages"`
	}
	if err := json.Unmarshal(encoded, &fixture); err != nil {
		t.Fatal(err)
	}
	runtime, repository := newRuntimeForTest(t)
	stored, _, _, err := repository.Create(context.Background(), taskstore.CreateInput{Task: fixture.Task, Stages: fixture.Stages})
	if err != nil {
		t.Fatal(err)
	}
	fixture.Input.Plan.Stages[0].ExternalExecution.ResourceKeys = []string{"compose:legacy"}
	receipt, err := runtime.Submit(context.Background(), fixture.Input)
	if err != nil || receipt.TaskID != stored.ID || receipt.Status != stored.Status {
		t.Fatalf("pre-upgrade Agent receipt replay = %#v, %v", receipt, err)
	}
	loaded := mustTask(t, repository, stored.ID)
	if !bytes.Equal(loaded.Plan, stored.Plan) || !sameSubmissionFingerprint(loaded.SubmissionFingerprint, fixture.Task.SubmissionFingerprint) {
		t.Fatal("receipt replay rewrote the frozen pre-upgrade execution plan")
	}
}
