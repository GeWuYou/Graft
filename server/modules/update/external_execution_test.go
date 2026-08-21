package update

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

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
