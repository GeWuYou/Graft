package runtimetarget

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"time"

	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

const agentLedgerSnapshotTTL = 5 * time.Minute
const agentLedgerSnapshotRandomBytes = 32

// errAgentLedgerRejected 防止 Agent 通过失败细节探测绑定、证书或快照状态。
var errAgentLedgerRejected = errors.New("runtime target agent ledger request rejected")

// runtimeTargetAgentLedgerAuthority 只协调 mTLS 已验证身份与 Runtime Target 持久化账本。
type runtimeTargetAgentLedgerAuthority struct {
	repository *store.SQLRepository
	now        func() time.Time
	random     io.Reader
}

func newRuntimeTargetAgentLedgerAuthority(repository *store.SQLRepository) runtimeTargetAgentLedgerAuthority {
	return runtimeTargetAgentLedgerAuthority{repository: repository, now: time.Now, random: rand.Reader}
}

// IssueLedgerSnapshot 为当前活动 Docker Agent 签发与其证书绑定的一次性账本快照。
func (a runtimeTargetAgentLedgerAuthority) IssueLedgerSnapshot(ctx context.Context, identity moduleapi.AgentIdentity) (moduleapi.RuntimeTargetLedgerSnapshot, error) {
	if a.repository == nil || a.now == nil || a.random == nil {
		return moduleapi.RuntimeTargetLedgerSnapshot{}, errAgentLedgerRejected
	}
	now := a.now().UTC()
	active, err := a.repository.ReadActiveAgentTrustGenerationByCertificate(ctx, identity.TargetID, identity.AgentID, identity.CertificateSerial, identity.PublicKeyFingerprint, now)
	if err != nil {
		return moduleapi.RuntimeTargetLedgerSnapshot{}, errAgentLedgerRejected
	}
	identity.IdentityID, identity.Generation = active.Identity.IdentityID, active.Generation
	snapshotID, err := newAgentLedgerSnapshotID(a.random)
	if err != nil {
		return moduleapi.RuntimeTargetLedgerSnapshot{}, errAgentLedgerRejected
	}
	snapshot, err := a.repository.IssueAgentLedgerSnapshot(ctx, store.AgentLedgerIdentity{
		IdentityID: identity.IdentityID, TargetID: identity.TargetID, AgentID: identity.AgentID, Generation: identity.Generation,
		CertificateSerial: identity.CertificateSerial, PublicKeyFingerprint: identity.PublicKeyFingerprint,
	}, snapshotID, now, now.Add(agentLedgerSnapshotTTL))
	if err != nil {
		return moduleapi.RuntimeTargetLedgerSnapshot{}, errAgentLedgerRejected
	}
	return moduleapi.RuntimeTargetLedgerSnapshot{
		IdentityID: snapshot.IdentityID, TargetID: snapshot.TargetID, AgentID: snapshot.AgentID, Generation: snapshot.Generation,
		Sequence: snapshot.Sequence, SnapshotID: snapshot.SnapshotID, SnapshotDigest: snapshot.SnapshotDigest,
		BuilderScope: snapshot.BuilderScope, ProviderID: snapshot.ProviderID, CapabilityProfile: snapshot.CapabilityProfile,
		CapabilityVersion: snapshot.CapabilityVersion, AffinityKey: snapshot.AffinityKey, Available: snapshot.Available,
		Running: snapshot.Running, Queued: snapshot.Queued, AllocatableSlots: snapshot.AllocatableSlots,
		ObservedAt: snapshot.ObservedAt, ExpiresAt: snapshot.ExpiresAt, IssuedAt: snapshot.IssuedAt,
	}, nil
}

// SubmitTelemetryReport 持久化一个已签发快照的受限回执；完全相同的重试保持幂等。
func (a runtimeTargetAgentLedgerAuthority) SubmitTelemetryReport(ctx context.Context, report moduleapi.RuntimeTargetTelemetryReport) error {
	if a.repository == nil || a.now == nil {
		return errAgentLedgerRejected
	}
	active, err := a.repository.ReadActiveAgentTrustGenerationByCertificate(ctx, report.TargetID, report.AgentID, report.CertificateSerial, report.PublicKeyFingerprint, a.now().UTC())
	if err != nil || active.Generation != report.Generation || active.Identity.IdentityID != report.IdentityID {
		return errAgentLedgerRejected
	}
	if err := a.repository.RecordAgentTelemetryReceipt(ctx, store.AgentLedgerIdentity{
		IdentityID: report.IdentityID, TargetID: report.TargetID, AgentID: report.AgentID, Generation: report.Generation,
		CertificateSerial: report.CertificateSerial, PublicKeyFingerprint: report.PublicKeyFingerprint,
	}, store.AgentTelemetryReceiptInput{
		SnapshotID: report.SnapshotID, SnapshotDigest: report.SnapshotDigest, ObservedAt: report.ObservedAt,
		ExpiresAt: report.ExpiresAt, Available: report.Available, ImplementationVersion: report.ImplementationVersion,
		Diagnostic: report.Diagnostic,
	}, a.now().UTC()); err != nil {
		return errAgentLedgerRejected
	}
	return nil
}

func newAgentLedgerSnapshotID(random io.Reader) (string, error) {
	bytes := make([]byte, agentLedgerSnapshotRandomBytes)
	if _, err := io.ReadFull(random, bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
