package moduleapi_test

import (
	"reflect"
	"testing"

	"graft/server/internal/moduleapi"
)

func TestRuntimeTargetAgentContractsKeepSecretMaterialOutsideModuleAPI(t *testing.T) {
	for _, contract := range []any{
		moduleapi.MachineEnrollment{},
		moduleapi.TrustBundleReference{},
		moduleapi.RuntimeTargetAgentBinding{},
		moduleapi.RuntimeTargetLedgerSnapshot{},
		moduleapi.RuntimeTargetTelemetryReport{},
	} {
		typeOfContract := reflect.TypeOf(contract)
		for field := range typeOfContract.NumField() {
			name := typeOfContract.Field(field).Name
			switch name {
			case "PrivateKey", "CertificatePEM", "CAPrivateMaterial", "EnrollmentSecret", "RegistryCredential", "Signature", "PublicKey":
				t.Fatalf("%s leaks prohibited secret or legacy trust field %q", typeOfContract.Name(), name)
			}
		}
	}
}

func TestRuntimeTargetAgentLedgerContractsBindReportsToIssuedSnapshot(t *testing.T) {
	snapshotType := reflect.TypeOf(moduleapi.RuntimeTargetLedgerSnapshot{})
	reportType := reflect.TypeOf(moduleapi.RuntimeTargetTelemetryReport{})
	for _, field := range []string{"TargetID", "AgentID", "Generation", "Sequence", "SnapshotID", "SnapshotDigest"} {
		if _, found := snapshotType.FieldByName(field); !found {
			t.Fatalf("ledger snapshot lacks binding field %q", field)
		}
	}
	for _, field := range []string{"TargetID", "AgentID", "Generation", "SnapshotID", "SnapshotDigest"} {
		if _, found := reportType.FieldByName(field); !found {
			t.Fatalf("telemetry report lacks binding field %q", field)
		}
	}
}
