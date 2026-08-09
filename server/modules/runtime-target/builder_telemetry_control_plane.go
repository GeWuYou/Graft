package runtimetarget

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

// controlPlaneBuilderTelemetryIngress 是 Runtime Target 对已绑定 Builder Agent 的签名报告入口。
// 它在写入账本前校验目标绑定、公钥签名和遥测窗口，不从 Docker、主机或 Task 投影补齐事实。
type controlPlaneBuilderTelemetryIngress struct {
	repository *store.SQLRepository
}

func (p controlPlaneBuilderTelemetryIngress) ProvisionBuilderTelemetryAgent(_ context.Context, _ moduleapi.BuilderTelemetryAgentRegistration) error {
	return store.ErrLegacyAgentTrustDisabled
}

//nolint:cyclop // 报告准入必须在签名、身份、窗口和序列 fence 的同一边界内完成。
func (p controlPlaneBuilderTelemetryIngress) SubmitBuilderTelemetry(_ context.Context, _ moduleapi.BuilderTelemetryReport) error {
	return store.ErrLegacyAgentTrustDisabled
}

func canonicalBuilderTelemetryReport(report moduleapi.BuilderTelemetryReport, now time.Time) ([]byte, error) {
	if !validBuilderTelemetryReport(report, now) {
		return nil, errors.New("builder telemetry report is invalid")
	}
	return json.Marshal(struct {
		AgentID               string   `json:"agent_id"`
		TargetID              int64    `json:"target_id"`
		Sequence              int64    `json:"sequence"`
		BuilderScope          string   `json:"builder_scope"`
		ProviderID            string   `json:"provider_id"`
		CapabilityProfile     string   `json:"capability_profile"`
		CapabilityVersion     string   `json:"capability_version"`
		AffinityKey           string   `json:"affinity_key"`
		Available             bool     `json:"available"`
		Running               int      `json:"running"`
		Queued                int      `json:"queued"`
		AllocatableSlots      int      `json:"allocatable_slots"`
		ObservedAt            string   `json:"observed_at"`
		ExpiresAt             string   `json:"expires_at"`
		SourceRef             string   `json:"source_ref"`
		Provenance            string   `json:"provenance"`
		UnsupportedDimensions []string `json:"unsupported_dimensions"`
	}{
		AgentID: report.AgentID, TargetID: report.TargetID, Sequence: report.Sequence, BuilderScope: report.BuilderScope, ProviderID: report.ProviderID,
		CapabilityProfile: report.CapabilityProfile, CapabilityVersion: report.CapabilityVersion, AffinityKey: report.AffinityKey, Available: report.Available,
		Running: report.Running, Queued: report.Queued, AllocatableSlots: report.AllocatableSlots,
		ObservedAt: report.ObservedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		ExpiresAt:  report.ExpiresAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), SourceRef: report.SourceRef,
		Provenance: report.Provenance, UnsupportedDimensions: report.UnsupportedDimensions,
	})
}

func validBuilderTelemetryReport(report moduleapi.BuilderTelemetryReport, now time.Time) bool {
	return builderTelemetryReportIdentityValid(report) && builderTelemetryReportCapacityValid(report) && builderTelemetryReportWindowValid(report, now) && builderTelemetryReportEvidenceValid(report)
}

func builderTelemetryReportIdentityValid(report moduleapi.BuilderTelemetryReport) bool {
	return report.TargetID > 0 && report.Sequence > 0 && strings.TrimSpace(report.AgentID) != "" && strings.TrimSpace(report.BuilderScope) != "" && report.ProviderID == "docker" && strings.TrimSpace(report.CapabilityProfile) != "" && strings.TrimSpace(report.CapabilityVersion) != ""
}

func builderTelemetryReportCapacityValid(report moduleapi.BuilderTelemetryReport) bool {
	return report.Running >= 0 && report.Queued >= 0 && report.AllocatableSlots >= 0
}

func builderTelemetryReportWindowValid(report moduleapi.BuilderTelemetryReport, now time.Time) bool {
	const maxClockSkew = 2 * time.Minute
	const maxReportLifetime = 5 * time.Minute
	return !report.ObservedAt.IsZero() && report.ExpiresAt.After(report.ObservedAt) && !report.ObservedAt.Before(now.Add(-maxClockSkew)) && !report.ObservedAt.After(now.Add(maxClockSkew)) && !report.ExpiresAt.After(report.ObservedAt.Add(maxReportLifetime))
}

func builderTelemetryReportEvidenceValid(report moduleapi.BuilderTelemetryReport) bool {
	return safeBuilderTelemetryEvidence(report.SourceRef) && report.Provenance == "docker-builder-agent-ledger"
}

func safeBuilderTelemetryEvidence(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && !strings.ContainsAny(trimmed, "\r\n@") && !strings.Contains(trimmed, "://")
}
