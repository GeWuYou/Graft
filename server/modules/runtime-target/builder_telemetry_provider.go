package runtimetarget

import (
	"context"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

// controlPlaneBuilderTelemetryReader 将 Build 可见读取边界与 provider 自有来源隔离。
// 它只暴露已脱敏的 Builder 范围调度事实，绝不暴露 provider 连接。
type controlPlaneBuilderTelemetryReader struct {
	provider moduleapi.BuilderTelemetryProvider
}

func (r controlPlaneBuilderTelemetryReader) ListBuilderTelemetry(ctx context.Context, targetIDs []int64) ([]moduleapi.BuilderTelemetrySnapshot, error) {
	if r.provider == nil {
		return []moduleapi.BuilderTelemetrySnapshot{}, nil
	}
	return r.provider.ListBuilderTelemetry(ctx, targetIDs)
}

func (r controlPlaneBuilderTelemetryReader) ConformBuilderTelemetry(ctx context.Context, targetIDs []int64) (bool, error) {
	if r.provider == nil {
		return false, nil
	}
	return r.provider.ConformBuilderTelemetry(ctx, targetIDs)
}

// controlPlaneBuilderTelemetryProvider 读取持久化的 Builder Agent/控制平面上报。
// 它刻意不依赖 Docker 客户端、主机指标读取器、Task 读取器或 UI/Monitor。
type controlPlaneBuilderTelemetryProvider struct {
	repository *store.SQLRepository
}

func (p controlPlaneBuilderTelemetryProvider) ListBuilderTelemetry(ctx context.Context, targetIDs []int64) ([]moduleapi.BuilderTelemetrySnapshot, error) {
	if p.repository == nil || len(targetIDs) == 0 {
		return []moduleapi.BuilderTelemetrySnapshot{}, nil
	}
	ledgers, err := p.repository.ListActiveDockerAgentLedgerSnapshots(ctx, targetIDs, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("read Docker builder ledger telemetry: %w", err)
	}
	result := make([]moduleapi.BuilderTelemetrySnapshot, 0, len(ledgers))
	for _, ledger := range ledgers {
		result = append(result, moduleapi.BuilderTelemetrySnapshot{
			TargetID: ledger.TargetID, BuilderScope: ledger.BuilderScope, ProviderID: ledger.ProviderID,
			CapabilityProfile: ledger.CapabilityProfile, CapabilityVersion: ledger.CapabilityVersion,
			AffinityKey: ledger.AffinityKey, Available: ledger.Available, Running: ledger.Running,
			Queued: ledger.Queued, AllocatableSlots: ledger.AllocatableSlots, ObservedAt: ledger.ObservedAt,
			ExpiresAt: ledger.ExpiresAt, SourceRef: "ledger:" + ledger.SnapshotID,
			Provenance: "runtime-target-controlled-execution-ledger", Integrity: "sha256:" + ledger.SnapshotDigest,
		})
	}
	return result, nil
}

func (p controlPlaneBuilderTelemetryProvider) ConformBuilderTelemetry(ctx context.Context, targetIDs []int64) (bool, error) {
	if len(targetIDs) == 0 {
		return false, nil
	}
	snapshots, err := p.ListBuilderTelemetry(ctx, targetIDs)
	if err != nil {
		return false, err
	}
	byTarget := make(map[int64]moduleapi.BuilderTelemetrySnapshot, len(snapshots))
	for _, snapshot := range snapshots {
		byTarget[snapshot.TargetID] = snapshot
	}
	now := time.Now().UTC()
	for _, targetID := range targetIDs {
		snapshot, found := byTarget[targetID]
		if !found || snapshot.ProviderID != "docker" || !strings.HasPrefix(snapshot.SourceRef, "ledger:") || snapshot.Provenance != "runtime-target-controlled-execution-ledger" || !snapshot.DynamicPlacementConformantAt(now) {
			return false, nil
		}
	}
	return true, nil
}
