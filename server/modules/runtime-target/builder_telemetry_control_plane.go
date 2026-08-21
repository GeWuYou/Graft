package runtimetarget

import (
	"context"

	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

// controlPlaneBuilderTelemetryIngress 是 Runtime Target 对已绑定 Builder Agent 的签名报告入口。
// 它在写入账本前校验目标绑定、公钥签名和遥测窗口，不从 Docker、主机或 Task 投影补齐事实。
type controlPlaneBuilderTelemetryIngress struct{}

func (controlPlaneBuilderTelemetryIngress) ProvisionBuilderTelemetryAgent(_ context.Context, _ moduleapi.BuilderTelemetryAgentRegistration) error {
	return store.ErrLegacyAgentTrustDisabled
}

func (controlPlaneBuilderTelemetryIngress) SubmitBuilderTelemetry(_ context.Context, _ moduleapi.BuilderTelemetryReport) error {
	return store.ErrLegacyAgentTrustDisabled
}
