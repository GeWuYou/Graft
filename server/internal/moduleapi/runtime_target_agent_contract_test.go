package moduleapi_test

import (
	"reflect"
	"strings"
	"testing"

	"graft/server/internal/moduleapi"
)

func TestRuntimeTargetAgentContractsKeepSecretMaterialOutsideModuleAPI(t *testing.T) {
	for _, contract := range []any{
		moduleapi.MachineEnrollmentRequest{},
		moduleapi.MachineEnrollment{},
		moduleapi.MachineIdentityActivation{},
		moduleapi.MachineIdentityRotationRequest{},
		moduleapi.MachineIdentityRevocation{},
		moduleapi.TrustBundleRequest{},
		moduleapi.TrustBundleReference{},
		moduleapi.AgentIdentity{},
		moduleapi.RuntimeTargetAgentBinding{},
		moduleapi.RuntimeTargetLedgerSnapshot{},
		moduleapi.RuntimeTargetTelemetryReport{},
	} {
		typeOfContract := reflect.TypeOf(contract)
		for field := range typeOfContract.NumField() {
			name := typeOfContract.Field(field).Name
			lowerName := strings.ToLower(name)
			for _, fragment := range []string{"private", "secret", "credential", "certificatepem", "signature", "password", "token"} {
				if strings.Contains(lowerName, fragment) {
					t.Errorf("%s leaks prohibited secret or legacy trust field %q", typeOfContract.Name(), name)
				}
			}
		}
	}
}

func TestRuntimeTargetAgentStatusContractsUseNamedLifecycleType(t *testing.T) {
	statusType := reflect.TypeOf(moduleapi.RuntimeTargetAgentStatus(""))
	for _, contract := range []any{
		moduleapi.MachineEnrollment{},
		moduleapi.RuntimeTargetAgentBinding{},
	} {
		field, found := reflect.TypeOf(contract).FieldByName("Status")
		if !found {
			t.Fatalf("%s lacks status field", reflect.TypeOf(contract).Name())
		}
		if field.Type != statusType {
			t.Fatalf("%s status has type %s, want %s", reflect.TypeOf(contract).Name(), field.Type, statusType)
		}
	}

	wantStatuses := map[moduleapi.RuntimeTargetAgentStatus]string{
		moduleapi.RuntimeTargetAgentStatusPending: "pending",
		moduleapi.RuntimeTargetAgentStatusActive:  "active",
		moduleapi.RuntimeTargetAgentStatusRevoked: "revoked",
		moduleapi.RuntimeTargetAgentStatusRetired: "retired",
	}
	for status, want := range wantStatuses {
		if string(status) != want {
			t.Errorf("status constant = %q, want %q", status, want)
		}
	}
}

func TestRuntimeTargetAgentLedgerContractsBindReportsToIssuedSnapshot(t *testing.T) {
	snapshotType := reflect.TypeOf(moduleapi.RuntimeTargetLedgerSnapshot{})
	reportType := reflect.TypeOf(moduleapi.RuntimeTargetTelemetryReport{})
	for field, wantKind := range map[string]reflect.Kind{
		"TargetID":       reflect.Int64,
		"AgentID":        reflect.String,
		"Generation":     reflect.Int64,
		"Sequence":       reflect.Int64,
		"SnapshotID":     reflect.String,
		"SnapshotDigest": reflect.String,
	} {
		fieldType, found := snapshotType.FieldByName(field)
		if !found {
			t.Errorf("ledger snapshot lacks binding field %q", field)
			continue
		}
		if fieldType.Type.Kind() != wantKind {
			t.Errorf("ledger snapshot field %q has kind %s, want %s", field, fieldType.Type.Kind(), wantKind)
		}
	}
	for field, wantKind := range map[string]reflect.Kind{
		"TargetID":       reflect.Int64,
		"AgentID":        reflect.String,
		"Generation":     reflect.Int64,
		"SnapshotID":     reflect.String,
		"SnapshotDigest": reflect.String,
	} {
		fieldType, found := reportType.FieldByName(field)
		if !found {
			t.Errorf("telemetry report lacks binding field %q", field)
			continue
		}
		if fieldType.Type.Kind() != wantKind {
			t.Errorf("telemetry report field %q has kind %s, want %s", field, fieldType.Type.Kind(), wantKind)
		}
	}
}
