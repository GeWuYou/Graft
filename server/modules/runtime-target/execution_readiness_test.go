package runtimetarget

import (
	"context"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
	"graft/server/modules/runtime-target/store"
)

func TestLocalExecutionReadinessIsIndependentOfAgentEnrollment(t *testing.T) {
	module := &Module{executionMode: config.ExecutionModeLocal, executionAdmission: executionAdmissionStub{}}
	target := store.Target{ID: 1, Provider: "docker", ConnectionKind: "unix_socket", EndpointLabel: localDockerEndpoint, Availability: true}
	readiness := module.executionReadiness(context.Background(), target)
	if !readiness.Ready || readiness.Mode != "local" || readiness.DiagnosticCode != "execution_ready" || len(readiness.Capabilities) != 4 {
		t.Fatalf("readiness=%#v", readiness)
	}
	target.Availability = false
	readiness = module.executionReadiness(context.Background(), target)
	if readiness.Ready || readiness.DiagnosticCode != "docker_unavailable" {
		t.Fatalf("unavailable=%#v", readiness)
	}
	target.Availability = true
	target.EndpointLabel = "unix:///other.sock"
	readiness = module.executionReadiness(context.Background(), target)
	if readiness.Ready || readiness.DiagnosticCode != "execution_mode_mismatch" {
		t.Fatalf("mismatch=%#v", readiness)
	}
}

type executionAdmissionStub struct{}

func (executionAdmissionStub) ExecutionAdmission() moduleapi.TaskExecutionAdmission {
	return moduleapi.TaskExecutionAdmission{Mode: "local", Ready: true, DiagnosticCode: "execution_ready"}
}
