package runtimetarget

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
	"graft/server/modules/runtime-target/store"
)

func TestLocalExecutionReadinessIsIndependentOfAgentEnrollment(t *testing.T) {
	module := &Module{executionMode: config.ExecutionModeLocal, executionAdmission: executionAdmissionStub{}}
	target := store.Target{ID: 1, Provider: "docker", ConnectionKind: "unix_socket", EndpointLabel: localDockerEndpoint, Availability: true}
	readiness := module.executionReadiness(context.Background(), target, true)
	if !readiness.Ready || readiness.Mode != "local" || readiness.DiagnosticCode != "execution_ready" || len(readiness.Capabilities) != 4 {
		t.Fatalf("readiness=%#v", readiness)
	}
	target.Availability = false
	readiness = module.executionReadiness(context.Background(), target, false)
	if readiness.Ready || readiness.DiagnosticCode != "docker_unavailable" {
		t.Fatalf("unavailable=%#v", readiness)
	}
	target.Availability = true
	target.EndpointLabel = "unix:///other.sock"
	readiness = module.executionReadiness(context.Background(), target, true)
	if readiness.Ready || readiness.DiagnosticCode != "execution_mode_mismatch" {
		t.Fatalf("mismatch=%#v", readiness)
	}
}

func TestLocalHTTPReadinessUsesTheCurrentHealthSnapshot(t *testing.T) {
	for _, healthy := range []bool{false, true} {
		module := &Module{executionMode: config.ExecutionModeLocal, executionAdmission: executionAdmissionStub{}, summaries: newSummaryCache()}
		target := store.Target{ID: 1, Provider: "docker", ConnectionKind: "unix_socket", EndpointLabel: localDockerEndpoint, Availability: !healthy}
		module.summaries.entries[target.ID] = summaryCacheEntry{summary: targetRuntimeSummary{Healthy: healthy, CheckedAt: time.Now().UTC()}, expiresAt: time.Now().Add(time.Minute)}
		detail := module.toHTTP(t.Context(), target)
		summary := module.toHTTPSummary(t.Context(), target)
		if detail.Execution.Ready != healthy || summary.Execution.Ready != healthy {
			t.Fatalf("persisted=%t current=%t detail=%#v summary=%#v", target.Availability, healthy, detail.Execution, summary.Execution)
		}
		if !healthy && (detail.Execution.DiagnosticCode != "docker_unavailable" || summary.Execution.DiagnosticCode != "docker_unavailable") {
			t.Fatalf("unavailable snapshot was not reflected: %#v", detail.Execution)
		}
	}
}

func TestAgentReadinessAdvertisesOnlyItsBoundReadyCapabilities(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	columns := strings.Split("id,identity_id,target_id,agent_id,provider,builder_scope,profile,version,digest,agent_version,generation_id,generation,enrollment_ref,bundle_ref,bundle_version,issuer,serial,fingerprint,expires_at,status,activated_at,retired_at,revoked_at,reason", ",")
	rows := sqlmock.NewRows(columns).AddRow(1, "identity-1", 1, "agent-1", "docker", "scope-1", "oci-build", "docker/v1",
		"sha256:"+strings.Repeat("a", 64), "fixture", 12, 1, "enrollment", "bundle", "bundle-1", "issuer", "serial", "fingerprint",
		time.Now().Add(time.Hour), "active", nil, nil, nil, "")
	mock.ExpectQuery("SELECT .* FROM runtime_target_agent_identities i INNER JOIN runtime_target_agent_generations g").
		WithArgs(int64(1), sqlmock.AnyArg()).WillReturnRows(rows)
	mock.ExpectQuery("SELECT identity_id, generation_id, provider_id, capabilities, capability_version FROM runtime_target_agent_capability_bindings").
		WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"identity_id", "generation_id", "provider_id", "capabilities", "capability_version"}).
		AddRow(1, 12, "docker", `["oci-build"]`, "docker/v1"))
	module := &Module{executionMode: config.ExecutionModeAgent, executionAdmission: fixedAdmissionReader{moduleapi.TaskExecutionAdmission{Mode: "agent", Ready: true}}, repository: store.NewSQLRepository(db)}
	target := store.Target{ID: 1, Provider: "docker", Availability: true}
	readiness := module.executionReadiness(t.Context(), target, true)
	if !readiness.Ready || len(readiness.Capabilities) != 1 || readiness.Capabilities[0] != "oci-build" {
		t.Fatalf("Agent binding was expanded into unsupported capabilities: %#v", readiness)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type fixedAdmissionReader struct {
	value moduleapi.TaskExecutionAdmission
}

func (s fixedAdmissionReader) ExecutionAdmission() moduleapi.TaskExecutionAdmission { return s.value }

func TestLocalReadinessKeepsOwnershipLossAheadOfDockerHealth(t *testing.T) {
	module := &Module{executionMode: config.ExecutionModeLocal,
		executionAdmission: fixedAdmissionReader{moduleapi.TaskExecutionAdmission{Mode: "local", DiagnosticCode: "execution_ownership_lost"}},
		summaries:          newSummaryCache()}
	target := store.Target{ID: 1, Provider: "docker", Availability: true}
	module.summaries.entries[target.ID] = summaryCacheEntry{summary: targetRuntimeSummary{Healthy: false}, expiresAt: time.Now().Add(time.Minute)}
	response := module.toHTTPSummary(t.Context(), target)
	if response.Execution.Ready || response.Execution.DiagnosticCode != "execution_ownership_lost" {
		t.Fatalf("ownership loss was replaced by another diagnostic: %#v", response.Execution)
	}
}

type executionAdmissionStub struct{}

func (executionAdmissionStub) ExecutionAdmission() moduleapi.TaskExecutionAdmission {
	return moduleapi.TaskExecutionAdmission{Mode: "local", Ready: true, DiagnosticCode: "execution_ready"}
}
