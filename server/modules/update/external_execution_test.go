package update

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

//nolint:gocyclo // 测试显式覆盖每个材料绑定断言。
func TestUpdateControllerRecoveryMaterialBindsClaimAndStateVolume(t *testing.T) {
	operation := ComposeUpdateOperation{OperationID: "update-material-recovery", SourceVersion: "1.0.0", TargetVersion: "1.1.0"}
	service := NewRolloutService(nil, nil, nil, nil, nil)
	if err := service.prepareExternalLaunch(operation.OperationID, 9); err != nil {
		t.Fatal(err)
	}
	recovery := RunnerRecoveryInput{OperationID: operation.OperationID, RunnerID: "runner-recovery-material", SourceVersion: operation.SourceVersion, TargetVersion: operation.TargetVersion, Strategy: string(DeploymentStrategyBetaTracking)}
	if err := service.prepareRecoveryLaunch(operation.OperationID, 9, recovery, "ghcr.io/gewuyou/graft-compose-runner@sha256:"+strings.Repeat("a", 64), "recovery-material"); err != nil {
		t.Fatal(err)
	}
	taskInput, err := updateControllerTaskInput(operation)
	if err != nil {
		t.Fatal(err)
	}
	material, err := service.ResolveExternalExecutionMaterial(context.Background(), moduleapi.ExternalExecutionMaterialRequest{TaskID: 1, ExecutorType: composeUpdateLaunchExecutor, RuntimeTargetID: 9, OperationID: composeUpdateLaunchOperation, Input: taskInput})
	if err != nil {
		t.Fatal(err)
	}
	var payload updateControllerMaterial
	if err := json.Unmarshal(material.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Recovery || payload.RecoveryClaimID != "recovery-material" || payload.OperationID != operation.OperationID || payload.StateVolume == "" || payload.InputBase64 != "" || payload.ComposeRoot != "" || payload.DockerSocket != "" {
		t.Fatalf("recovery material leaked or lost binding: %#v", payload)
	}
	decoded, err := base64.RawStdEncoding.DecodeString(payload.RecoveryStateBase64)
	if err != nil {
		t.Fatal(err)
	}
	var got RunnerRecoveryInput
	if err := json.Unmarshal(decoded, &got); err != nil || got.OperationID != recovery.OperationID || got.RunnerID != recovery.RunnerID {
		t.Fatalf("recovery state binding = %#v, %v", got, err)
	}
}

func TestUpdateControllerRecoveryMaterialReusesNormalLaunchSlot(t *testing.T) {
	service := NewRolloutService(nil, nil, nil, nil, nil)
	if err := service.prepareExternalLaunch("update-material-slot", 11); err != nil {
		t.Fatal(err)
	}
	input := RunnerRecoveryInput{OperationID: "update-material-slot", RunnerID: "runner-slot", SourceVersion: "1.0.0", TargetVersion: "1.1.0", Strategy: string(DeploymentStrategyBetaTracking)}
	if err := service.prepareRecoveryLaunch(input.OperationID, 11, input, "image", "recovery-slot"); err != nil {
		t.Fatalf("existing launch slot must be reusable: %v", err)
	}
	if err := service.prepareRecoveryLaunch(input.OperationID, 12, input, "image", "recovery-slot"); err == nil {
		t.Fatal("recovery slot must remain bound to its runtime target")
	}
}

func TestUpdateControllerMaterialWaitsForPreparedHandoff(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "compose.yml"), []byte("services: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	operation := ComposeUpdateOperation{OperationID: "update-material-wait", SourceVersion: "1.0.0", TargetVersion: "1.1.0"}
	input := fixtureRunnerInput(root)
	input.OperationID = operation.OperationID
	input.SourceVersion = operation.SourceVersion
	input.TargetVersion = operation.TargetVersion
	input.BackupArtifactRoot = "/var/lib/graft/backups/" + operation.OperationID
	service := NewRolloutService(nil, nil, nil, nil, nil)
	if err := service.prepareExternalLaunch(operation.OperationID, 7); err != nil {
		t.Fatal(err)
	}
	taskInput, err := updateControllerTaskInput(operation)
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan moduleapi.ExternalExecutionMaterial, 1)
	errorsCh := make(chan error, 1)
	go func() {
		material, resolveErr := service.ResolveExternalExecutionMaterial(context.Background(), moduleapi.ExternalExecutionMaterialRequest{
			TaskID: input.TaskID, ExecutorType: composeUpdateLaunchExecutor, RuntimeTargetID: 7,
			OperationID: composeUpdateLaunchOperation, Input: taskInput,
		})
		result <- material
		errorsCh <- resolveErr
	}()
	select {
	case err := <-errorsCh:
		t.Fatalf("material resolved before backup handoff: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	if err := service.completeExternalLaunch(operation.OperationID, 7, input); err != nil {
		t.Fatal(err)
	}
	material := <-result
	if err := <-errorsCh; err != nil {
		t.Fatal(err)
	}
	var payload updateControllerMaterial
	if material.Protocol != composeUpdateLaunchMaterialProtocol || json.Unmarshal(material.Payload, &payload) != nil || payload.OperationID != operation.OperationID || payload.ComposeRoot != root {
		t.Fatalf("unexpected transient material: %#v %#v", material, payload)
	}
}
