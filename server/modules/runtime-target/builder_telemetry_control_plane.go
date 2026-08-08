package runtimetarget

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

// controlPlaneBuilderTelemetryIngress 是 Runtime Target 对已绑定 Builder Agent 的签名报告入口。
// 它在写入账本前校验目标绑定、公钥签名和遥测窗口，不从 Docker、主机或 Task 投影补齐事实。
type controlPlaneBuilderTelemetryIngress struct {
	repository *store.SQLRepository
}

func (p controlPlaneBuilderTelemetryIngress) ProvisionBuilderTelemetryAgent(ctx context.Context, registration moduleapi.BuilderTelemetryAgentRegistration) error {
	if p.repository == nil {
		return errors.New("builder telemetry control plane is unavailable")
	}
	return p.repository.UpsertBuilderTelemetryAgent(ctx, store.BuilderTelemetryAgent{TargetID: registration.TargetID, AgentID: registration.AgentID, PublicKey: append([]byte(nil), registration.PublicKey...), Enabled: registration.Enabled})
}

func (p controlPlaneBuilderTelemetryIngress) SubmitBuilderTelemetry(ctx context.Context, report moduleapi.BuilderTelemetryReport) error {
	if p.repository == nil {
		return errors.New("builder telemetry control plane is unavailable")
	}
	if len(report.Signature) != ed25519.SignatureSize {
		return errors.New("builder telemetry signature is invalid")
	}
	payload, err := canonicalBuilderTelemetryReport(report)
	if err != nil {
		return err
	}
	agent, err := p.repository.GetBuilderTelemetryAgent(ctx, report.TargetID, report.AgentID)
	if err != nil {
		return fmt.Errorf("load builder telemetry agent: %w", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(agent.PublicKey), payload, report.Signature) {
		return errors.New("builder telemetry signature is invalid")
	}
	observation := builderTelemetryObservationFromReport(report, payload)
	if err := p.repository.RecordBuilderTelemetryObservation(ctx, observation); err != nil {
		return err
	}
	return nil
}

func canonicalBuilderTelemetryReport(report moduleapi.BuilderTelemetryReport) ([]byte, error) {
	if !validBuilderTelemetryReport(report) {
		return nil, errors.New("builder telemetry report is invalid")
	}
	return json.Marshal(struct {
		AgentID               string   `json:"agent_id"`
		TargetID              int64    `json:"target_id"`
		BuilderScope          string   `json:"builder_scope"`
		ProviderID            string   `json:"provider_id"`
		CapabilityProfile     string   `json:"capability_profile"`
		CapabilityVersion     string   `json:"capability_version"`
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
		AgentID: report.AgentID, TargetID: report.TargetID, BuilderScope: report.BuilderScope, ProviderID: report.ProviderID,
		CapabilityProfile: report.CapabilityProfile, CapabilityVersion: report.CapabilityVersion, Available: report.Available,
		Running: report.Running, Queued: report.Queued, AllocatableSlots: report.AllocatableSlots,
		ObservedAt: report.ObservedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		ExpiresAt:  report.ExpiresAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), SourceRef: report.SourceRef,
		Provenance: report.Provenance, UnsupportedDimensions: report.UnsupportedDimensions,
	})
}

func validBuilderTelemetryReport(report moduleapi.BuilderTelemetryReport) bool {
	return builderTelemetryReportIdentityValid(report) && builderTelemetryReportCapacityValid(report) && builderTelemetryReportWindowValid(report) && builderTelemetryReportEvidenceValid(report)
}

func builderTelemetryReportIdentityValid(report moduleapi.BuilderTelemetryReport) bool {
	return report.TargetID > 0 && strings.TrimSpace(report.AgentID) != "" && strings.TrimSpace(report.BuilderScope) != "" && strings.TrimSpace(report.ProviderID) != "" && strings.TrimSpace(report.CapabilityProfile) != "" && strings.TrimSpace(report.CapabilityVersion) != ""
}

func builderTelemetryReportCapacityValid(report moduleapi.BuilderTelemetryReport) bool {
	return report.Running >= 0 && report.Queued >= 0 && report.AllocatableSlots >= 0
}

func builderTelemetryReportWindowValid(report moduleapi.BuilderTelemetryReport) bool {
	return !report.ObservedAt.IsZero() && report.ExpiresAt.After(report.ObservedAt)
}

func builderTelemetryReportEvidenceValid(report moduleapi.BuilderTelemetryReport) bool {
	return safeBuilderTelemetryEvidence(report.SourceRef) && safeBuilderTelemetryEvidence(report.Provenance)
}

func safeBuilderTelemetryEvidence(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && !strings.ContainsAny(trimmed, "\r\n@") && !strings.Contains(trimmed, "://")
}

func builderTelemetryObservationFromReport(report moduleapi.BuilderTelemetryReport, payload []byte) store.BuilderTelemetryObservation {
	digest := sha256.Sum256(payload)
	return store.BuilderTelemetryObservation{TargetID: report.TargetID, BuilderScope: report.BuilderScope, ProviderID: report.ProviderID, CapabilityProfile: report.CapabilityProfile, CapabilityVersion: report.CapabilityVersion, Available: report.Available, Running: report.Running, Queued: report.Queued, AllocatableSlots: report.AllocatableSlots, ObservedAt: report.ObservedAt.UTC(), ExpiresAt: report.ExpiresAt.UTC(), SourceRef: report.SourceRef, Provenance: report.Provenance, Integrity: "sha256:" + hex.EncodeToString(digest[:]), UnsupportedDimensions: append([]string(nil), report.UnsupportedDimensions...)}
}
