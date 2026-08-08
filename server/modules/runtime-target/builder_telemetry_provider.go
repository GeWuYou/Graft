package runtimetarget

import (
	"context"
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
	now        func() time.Time
}

func (p controlPlaneBuilderTelemetryProvider) ListBuilderTelemetry(ctx context.Context, targetIDs []int64) ([]moduleapi.BuilderTelemetrySnapshot, error) {
	if p.repository == nil {
		return []moduleapi.BuilderTelemetrySnapshot{}, nil
	}
	observations, err := p.repository.ListLatestBuilderTelemetry(ctx, targetIDs)
	if err != nil {
		return nil, err
	}
	results := make([]moduleapi.BuilderTelemetrySnapshot, 0, len(observations))
	for _, observation := range observations {
		results = append(results, moduleapi.BuilderTelemetrySnapshot{
			TargetID: observation.TargetID, BuilderScope: observation.BuilderScope, ProviderID: observation.ProviderID,
			CapabilityProfile: observation.CapabilityProfile, CapabilityVersion: observation.CapabilityVersion,
			AffinityKey: observation.AffinityKey,
			Available:   observation.Available, Running: observation.Running, Queued: observation.Queued,
			AllocatableSlots: observation.AllocatableSlots, ObservedAt: observation.ObservedAt, ExpiresAt: observation.ExpiresAt,
			SourceRef: observation.SourceRef, Provenance: observation.Provenance, Integrity: observation.Integrity,
			UnsupportedDimensions: append([]string(nil), observation.UnsupportedDimensions...),
		})
	}
	return results, nil
}

func (p controlPlaneBuilderTelemetryProvider) ConformBuilderTelemetry(ctx context.Context, targetIDs []int64) (bool, error) {
	if len(targetIDs) == 0 {
		return false, nil
	}
	snapshots, err := p.ListBuilderTelemetry(ctx, targetIDs)
	if err != nil {
		return false, err
	}
	if len(snapshots) != len(targetIDs) {
		return false, nil
	}
	return builderTelemetrySnapshotsConform(targetIDs, snapshots, p.currentTime()), nil
}

func (p controlPlaneBuilderTelemetryProvider) currentTime() time.Time {
	if p.now == nil {
		return time.Now().UTC()
	}
	return p.now().UTC()
}

func builderTelemetrySnapshotsConform(targetIDs []int64, snapshots []moduleapi.BuilderTelemetrySnapshot, now time.Time) bool {
	seen := make(map[int64]struct{}, len(targetIDs))
	for _, snapshot := range snapshots {
		if !admitBuilderTelemetrySnapshot(seen, snapshot, now) {
			return false
		}
	}
	return allBuilderTelemetryTargetsSeen(targetIDs, seen)
}

func admitBuilderTelemetrySnapshot(seen map[int64]struct{}, snapshot moduleapi.BuilderTelemetrySnapshot, now time.Time) bool {
	if _, duplicate := seen[snapshot.TargetID]; duplicate || !snapshot.DynamicPlacementConformantAt(now) {
		return false
	}
	seen[snapshot.TargetID] = struct{}{}
	return true
}

func allBuilderTelemetryTargetsSeen(targetIDs []int64, seen map[int64]struct{}) bool {
	for _, targetID := range targetIDs {
		if targetID < 1 {
			return false
		}
		if _, ok := seen[targetID]; !ok {
			return false
		}
	}
	return true
}
