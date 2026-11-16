package dockerexecution

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRecoveryMaterialAcceptsOwnerSerializedProtocol(t *testing.T) {
	for name, payload := range recoveryOwnerPayloads(t) {
		if !validUpdateControllerMaterial(recoveryMaterialForPayload(payload), recoveryProtocolRequest()) {
			t.Fatalf("owner-serialized recovery %s was rejected", name)
		}
	}
}

func TestRecoveryMaterialRejectsChangedBindingsAndUnknownFields(t *testing.T) {
	base := recoveryOwnerPayloads(t)["with_state"]
	for _, testCase := range []struct {
		field string
		value json.RawMessage
	}{
		{"execution_mode", json.RawMessage(`"other"`)},
		{"execution_mode", json.RawMessage(`"agent"`)},
		{"launch_stage_id", json.RawMessage(`0`)},
		{"launch_attempt", json.RawMessage(`0`)},
		{"launch_attempt", json.RawMessage(`-1`)},
		{"state", json.RawMessage(`{"operation_id":"update-other","runner_id":"runner-recovery-42","execution_mode":"local"}`)},
		{"state", json.RawMessage(`{"operation_id":"update-42","runner_id":"runner-other","execution_mode":"local"}`)},
		{"unexpected", json.RawMessage(`true`)},
	} {
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(base, &payload); err != nil {
			t.Fatal(err)
		}
		payload[testCase.field] = testCase.value
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		if validUpdateControllerMaterial(recoveryMaterialForPayload(encoded), recoveryProtocolRequest()) {
			t.Fatalf("invalid recovery binding accepted for %s=%s", testCase.field, testCase.value)
		}
	}
}

func recoveryProtocolRequest() Request {
	return Request{Input: json.RawMessage(`{"operation_id":"update-42","source_version":"1.0.0","target_version":"1.1.0"}`)}
}

func recoveryMaterialForPayload(payload []byte) updateControllerMaterial {
	return updateControllerMaterial{ControllerReference: "ghcr.io/gewuyou/graft-compose-runner@sha256:" + strings.Repeat("a", 64), RecoveryStateBase64: base64.RawStdEncoding.EncodeToString(payload), RecoveryClaimID: "recovery-42", Recovery: true, StateVolume: "graft-update-state", OperationID: "update-42"}
}

func recoveryOwnerPayloads(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	// 夹具由 Update owner 的 RunnerRecoveryInput 与已验证状态存储序列化生成。
	contents, err := os.ReadFile("testdata/update-recovery-owner-868718f83.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Payloads map[string]json.RawMessage `json:"payloads"`
	}
	if err := json.Unmarshal(contents, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture.Payloads
}
