package runtimetarget

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

const dockerBuilderAgentTelemetryTTL = time.Minute

// DockerBuilderAgent 是 Docker provider 的受控执行账本适配器。
// 它只接受自身 queue/running 计数的变更，并从该账本生成签名遥测；不会读取 Docker stats、Task JSON 或主机指标。
type DockerBuilderAgent struct {
	mu           sync.Mutex
	registration moduleapi.BuilderTelemetryAgentRegistration
	privateKey   ed25519.PrivateKey
	repository   *store.SQLRepository
	now          func() time.Time
	slots        int
	queued       int
	running      int
	sequence     int64
}

// NewDockerBuilderAgent 创建绑定单一 Runtime Target、Builder scope 和能力版本的 Agent 账本。
func NewDockerBuilderAgent(registration moduleapi.BuilderTelemetryAgentRegistration, privateKey ed25519.PrivateKey, slots int) (*DockerBuilderAgent, error) {
	if registration.TargetID < 1 || strings.TrimSpace(registration.AgentID) == "" || registration.ProviderID != "docker" || strings.TrimSpace(registration.BuilderScope) == "" || strings.TrimSpace(registration.CapabilityProfile) == "" || strings.TrimSpace(registration.CapabilityVersion) == "" || len(privateKey) != ed25519.PrivateKeySize || slots < 1 {
		return nil, errors.New("docker builder agent binding is invalid")
	}
	return &DockerBuilderAgent{registration: registration, privateKey: append(ed25519.PrivateKey(nil), privateKey...), slots: slots, now: func() time.Time { return time.Now().UTC() }}, nil
}

// NewDurableDockerBuilderAgent 创建使用 Runtime Target 持久化执行账本的 Docker Builder Agent。
// 代理绑定和容量预算只在初始化时写入；已有排队、运行和遥测序列不会因进程重启而重置。
func NewDurableDockerBuilderAgent(ctx context.Context, registration moduleapi.BuilderTelemetryAgentRegistration, privateKey ed25519.PrivateKey, slots int, repository *store.SQLRepository) (*DockerBuilderAgent, error) {
	if repository == nil {
		return nil, errors.New("docker builder agent repository is unavailable")
	}
	agent, err := NewDockerBuilderAgent(registration, privateKey, slots)
	if err != nil {
		return nil, err
	}
	if err := repository.EnsureBuilderAgentLedger(ctx, registration.TargetID, registration.AgentID, slots); err != nil {
		return nil, err
	}
	agent.repository = repository
	return agent, nil
}

// QueueBuild 将一个待执行 Build 记入 Agent 自有队列账本。
func (a *DockerBuilderAgent) QueueBuild() error {
	return a.QueueBuildContext(context.Background())
}

// QueueBuildContext 将一个待执行 Build 写入持久化或内存账本。
func (a *DockerBuilderAgent) QueueBuildContext(ctx context.Context) error {
	if a == nil {
		return errors.New("docker builder agent is unavailable")
	}
	if a.repository != nil {
		return a.repository.QueueBuilderAgentBuild(ctx, a.registration.TargetID, a.registration.AgentID)
	}
	a.mu.Lock()
	a.queued++
	a.mu.Unlock()
	return nil
}

// StartBuild 将一个排队 Build 转为受控执行，并拒绝超过 Agent slot budget 的启动。
func (a *DockerBuilderAgent) StartBuild() error {
	return a.StartBuildContext(context.Background())
}

// StartBuildContext 原子地将一个排队 Build 转为运行 Build。
func (a *DockerBuilderAgent) StartBuildContext(ctx context.Context) error {
	if a == nil {
		return errors.New("docker builder agent is unavailable")
	}
	if a.repository != nil {
		return a.repository.StartBuilderAgentBuild(ctx, a.registration.TargetID, a.registration.AgentID)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.queued < 1 || a.running >= a.slots {
		return errors.New("docker builder agent has no allocatable slot")
	}
	a.queued--
	a.running++
	return nil
}

// FinishBuild 结算一个由 Agent 账本确认的运行 Build。
func (a *DockerBuilderAgent) FinishBuild() error {
	return a.FinishBuildContext(context.Background())
}

// FinishBuildContext 结算一个运行 Build。
func (a *DockerBuilderAgent) FinishBuildContext(ctx context.Context) error {
	if a == nil {
		return errors.New("docker builder agent is unavailable")
	}
	if a.repository != nil {
		return a.repository.FinishBuilderAgentBuild(ctx, a.registration.TargetID, a.registration.AgentID)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.running < 1 {
		return errors.New("docker builder agent has no running build")
	}
	a.running--
	return nil
}

// Snapshot 返回当前受控执行账本状态。
func (a *DockerBuilderAgent) Snapshot(ctx context.Context) (store.BuilderAgentLedgerState, error) {
	if a == nil {
		return store.BuilderAgentLedgerState{}, errors.New("docker builder agent is unavailable")
	}
	if a.repository != nil {
		return a.repository.SnapshotBuilderAgentLedger(ctx, a.registration.TargetID, a.registration.AgentID)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return store.BuilderAgentLedgerState{TargetID: a.registration.TargetID, AgentID: a.registration.AgentID, SlotBudget: a.slots, Queued: a.queued, Running: a.running, TelemetrySequence: a.sequence}, nil
}

// PublishTelemetry 从当前受控账本生成并签名一条报告，再交给 Runtime Target 控制平面校验。
func (a *DockerBuilderAgent) PublishTelemetry(ctx context.Context, controlPlane moduleapi.RuntimeTargetBuilderTelemetryControlPlane) error {
	if a == nil || controlPlane == nil {
		return errors.New("docker builder agent telemetry sink is unavailable")
	}
	var state store.BuilderAgentLedgerState
	var now time.Time
	if a.repository != nil {
		var err error
		state, err = a.repository.AdvanceBuilderAgentTelemetry(ctx, a.registration.TargetID, a.registration.AgentID)
		if err != nil {
			return err
		}
		now = a.currentTime()
	} else {
		a.mu.Lock()
		a.sequence++
		now = a.currentTime()
		state = store.BuilderAgentLedgerState{TargetID: a.registration.TargetID, AgentID: a.registration.AgentID, SlotBudget: a.slots, Queued: a.queued, Running: a.running, TelemetrySequence: a.sequence}
		a.mu.Unlock()
	}
	report := moduleapi.BuilderTelemetryReport{
		AgentID: a.registration.AgentID, TargetID: a.registration.TargetID, Sequence: state.TelemetrySequence,
		BuilderScope: a.registration.BuilderScope, ProviderID: a.registration.ProviderID,
		CapabilityProfile: a.registration.CapabilityProfile, CapabilityVersion: a.registration.CapabilityVersion,
		Available: true, Running: state.Running, Queued: state.Queued, AllocatableSlots: state.SlotBudget - state.Running - state.Queued,
		ObservedAt: now, ExpiresAt: now.Add(dockerBuilderAgentTelemetryTTL),
		SourceRef: fmt.Sprintf("docker-agent-ledger:%s", a.registration.AgentID), Provenance: "docker-builder-agent-ledger",
		UnsupportedDimensions: []string{"cache_state"},
	}
	if report.AllocatableSlots < 0 {
		report.AllocatableSlots = 0
	}
	payload, err := canonicalBuilderTelemetryReport(report, now)
	if err != nil {
		return err
	}
	report.Signature = ed25519.Sign(a.privateKey, payload)
	return controlPlane.SubmitBuilderTelemetry(ctx, report)
}

func (a *DockerBuilderAgent) currentTime() time.Time {
	if a.now == nil {
		return time.Now().UTC()
	}
	return a.now().UTC()
}
