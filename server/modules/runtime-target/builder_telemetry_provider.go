package runtimetarget

import (
	"context"

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

func (p controlPlaneBuilderTelemetryProvider) ListBuilderTelemetry(_ context.Context, _ []int64) ([]moduleapi.BuilderTelemetrySnapshot, error) {
	return []moduleapi.BuilderTelemetrySnapshot{}, nil
}

func (p controlPlaneBuilderTelemetryProvider) ConformBuilderTelemetry(_ context.Context, _ []int64) (bool, error) {
	return false, nil
}
