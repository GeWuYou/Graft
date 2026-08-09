package moduleapi_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestRuntimeTargetAgentContractsKeepSecretMaterialOutsideModuleAPI(t *testing.T) {
	for _, contract := range []any{
		moduleapi.AgentEnrollmentRequest{},
		moduleapi.AgentEnrollment{},
		moduleapi.AgentEnrollmentActivation{},
		moduleapi.AgentEnrollmentRotationRequest{},
		moduleapi.AgentEnrollmentRevocation{},
		moduleapi.AgentCertificateIssuanceRequest{},
		moduleapi.IssuedAgentCertificate{},
		moduleapi.AgentCertificateRevocation{},
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
			for _, fragment := range []string{"private", "secret", "credential", "certificatepem", "pem", "signature", "password", "token"} {
				if strings.Contains(lowerName, fragment) {
					t.Errorf("%s leaks prohibited secret or legacy trust field %q", typeOfContract.Name(), name)
				}
			}
		}
	}
}

func TestAgentEnrollmentContractsCarryNonSecretPKIAttestationMetadata(t *testing.T) {
	requestType := reflect.TypeOf(moduleapi.AgentEnrollmentRequest{})
	rotationType := reflect.TypeOf(moduleapi.AgentEnrollmentRotationRequest{})
	activationType := reflect.TypeOf(moduleapi.AgentEnrollmentActivation{})
	trustBundleType := reflect.TypeOf(moduleapi.TrustBundleReference{})

	for _, contract := range []reflect.Type{requestType, rotationType} {
		field, found := contract.FieldByName("EnrollmentRef")
		if !found || field.Type.Kind() != reflect.String {
			t.Errorf("%s EnrollmentRef = %v, want string", contract.Name(), field.Type)
		}
		field, found = contract.FieldByName("TrustBundle")
		if !found || field.Type != trustBundleType {
			t.Errorf("%s TrustBundle = %v, want TrustBundleReference", contract.Name(), field.Type)
		}
		field, found = contract.FieldByName("ExpiresAt")
		if !found || field.Type != reflect.TypeOf(time.Time{}) {
			t.Errorf("%s ExpiresAt = %v, want time.Time", contract.Name(), field.Type)
		}
	}
	field, found := activationType.FieldByName("CertificateIssuer")
	if !found || field.Type.Kind() != reflect.String {
		t.Errorf("AgentEnrollmentActivation CertificateIssuer = %v, want string", field.Type)
	}
}

func TestRuntimeTargetAgentStatusContractsUseNamedLifecycleType(t *testing.T) {
	statusType := reflect.TypeOf(moduleapi.RuntimeTargetAgentStatus(""))
	for _, contract := range []any{
		moduleapi.AgentEnrollment{},
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
