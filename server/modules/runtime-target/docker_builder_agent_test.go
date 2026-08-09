package runtimetarget

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

type telemetryReportSink struct {
	report moduleapi.BuilderTelemetryReport
}

func (s *telemetryReportSink) SubmitBuilderTelemetry(_ context.Context, report moduleapi.BuilderTelemetryReport) error {
	s.report = report
	return nil
}

func (s *telemetryReportSink) ProvisionBuilderTelemetryAgent(context.Context, moduleapi.BuilderTelemetryAgentRegistration) error {
	return nil
}

//nolint:gocyclo,cyclop // 状态转移、签名发布和遥测断言共同覆盖 Agent 的最小 conformance seam。
func TestDockerBuilderAgentTelemetryReflectsControlledLedgerState(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate agent key: %v", err)
	}
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	agent, err := NewDockerBuilderAgent(moduleapi.BuilderTelemetryAgentRegistration{TargetID: 7, AgentID: "agent:7", ProviderID: "docker", BuilderScope: "builder-agent:7", CapabilityProfile: "oci-build", CapabilityVersion: "v1", PublicKey: publicKey, Enabled: true}, privateKey, 2)
	if err != nil {
		t.Fatalf("new Docker builder agent: %v", err)
	}
	agent.now = func() time.Time { return now }
	sink := &telemetryReportSink{}
	if err := agent.QueueBuildContext(context.Background()); err != nil {
		t.Fatalf("queue build: %v", err)
	}
	if err := agent.PublishTelemetry(context.Background(), sink); err != nil {
		t.Fatalf("publish queued telemetry: %v", err)
	}
	if sink.report.Queued != 1 || sink.report.Running != 0 || sink.report.AllocatableSlots != 1 || sink.report.Sequence != 1 {
		t.Fatalf("queued telemetry = %#v", sink.report)
	}
	if err := agent.StartBuildContext(context.Background()); err != nil {
		t.Fatalf("start build: %v", err)
	}
	if err := agent.PublishTelemetry(context.Background(), sink); err != nil {
		t.Fatalf("publish running telemetry: %v", err)
	}
	if sink.report.Queued != 0 || sink.report.Running != 1 || sink.report.AllocatableSlots != 1 || sink.report.Sequence != 2 {
		t.Fatalf("running telemetry = %#v", sink.report)
	}
	if err := agent.FinishBuildContext(context.Background()); err != nil {
		t.Fatalf("finish build: %v", err)
	}
	if err := agent.PublishTelemetry(context.Background(), sink); err != nil {
		t.Fatalf("publish idle telemetry: %v", err)
	}
	if sink.report.Running != 0 || sink.report.AllocatableSlots != 2 || sink.report.Sequence != 3 {
		t.Fatalf("idle telemetry = %#v", sink.report)
	}
}

func TestDockerBuilderAgentCannotPublishThroughLegacyRuntimeTargetIngress(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate agent key: %v", err)
	}
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	registration := moduleapi.BuilderTelemetryAgentRegistration{TargetID: 7, AgentID: "agent:7", ProviderID: "docker", BuilderScope: "builder-agent:7", CapabilityProfile: "oci-build", CapabilityVersion: "v1", PublicKey: publicKey, Enabled: true}
	ingress := controlPlaneBuilderTelemetryIngress{}
	agent, err := NewDockerBuilderAgent(registration, privateKey, 2)
	if err != nil {
		t.Fatalf("new Docker builder agent: %v", err)
	}
	agent.now = func() time.Time { return now }
	if err := agent.QueueBuildContext(context.Background()); err != nil {
		t.Fatalf("queue build: %v", err)
	}
	if err := agent.StartBuildContext(context.Background()); err != nil {
		t.Fatalf("start build: %v", err)
	}
	if err := agent.PublishTelemetry(context.Background(), ingress); !errors.Is(err, store.ErrLegacyAgentTrustDisabled) {
		t.Fatalf("legacy ingress error = %v, want %v", err, store.ErrLegacyAgentTrustDisabled)
	}
}

//nolint:gocyclo // 重启、容量拒绝与释放后的恢复必须在同一 durable ledger 场景中验证。
func TestDurableDockerBuilderAgentSurvivesRestartAndEnforcesSlots(t *testing.T) {
	db := openBuilderTelemetryTestDB(t)
	repository := store.NewSQLRepository(db)
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate agent key: %v", err)
	}
	registration := moduleapi.BuilderTelemetryAgentRegistration{TargetID: 7, AgentID: "agent:durable", ProviderID: "docker", BuilderScope: "builder-agent:durable", CapabilityProfile: "oci-build", CapabilityVersion: "v1", PublicKey: publicKey, Enabled: true}
	first, err := NewDurableDockerBuilderAgent(context.Background(), registration, privateKey, 1, repository)
	if err != nil {
		t.Fatalf("new durable agent: %v", err)
	}
	if err := first.QueueBuildContext(context.Background()); err != nil {
		t.Fatalf("queue first build: %v", err)
	}
	if err := first.StartBuildContext(context.Background()); err != nil {
		t.Fatalf("start first build: %v", err)
	}
	if err := first.QueueBuildContext(context.Background()); err != nil {
		t.Fatalf("queue second build: %v", err)
	}
	if err := first.StartBuildContext(context.Background()); !errors.Is(err, store.ErrBuilderLedgerRejected) {
		t.Fatalf("over-capacity start error = %v, want %v", err, store.ErrBuilderLedgerRejected)
	}

	second, err := NewDurableDockerBuilderAgent(context.Background(), registration, privateKey, 1, repository)
	if err != nil {
		t.Fatalf("recreate durable agent: %v", err)
	}
	state, err := second.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot after restart: %v", err)
	}
	if state.Queued != 1 || state.Running != 1 || state.SlotBudget != 1 {
		t.Fatalf("restart state = %+v", state)
	}
	if err := second.FinishBuildContext(context.Background()); err != nil {
		t.Fatalf("finish first build: %v", err)
	}
	if err := second.StartBuildContext(context.Background()); err != nil {
		t.Fatalf("start queued build after release: %v", err)
	}
	state, err = second.Snapshot(context.Background())
	if err != nil || state.Queued != 0 || state.Running != 1 {
		t.Fatalf("state after release/start = %+v, err=%v", state, err)
	}
}
