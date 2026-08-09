package runtimetarget

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

// errAgentDeliveryAuthorityRejected 不向内部调用方泄露 grant 是否存在或已交付。
var errAgentDeliveryAuthorityRejected = errors.New("runtime target agent delivery authority rejected")

const agentDeliveryTokenBytes = 32

// runtimeTargetAgentDeliveryAuthority 将 Runtime Target 的交付归属约束与 store 的原子状态转换组合起来。
type runtimeTargetAgentDeliveryAuthority struct {
	repository *store.SQLRepository
	pepper     *config.EnrollmentPepperProvider
	now        func() time.Time
	random     io.Reader
}

func newRuntimeTargetAgentDeliveryAuthority(repository *store.SQLRepository, pepper *config.EnrollmentPepperProvider) moduleapi.AgentDeliveryAuthority {
	return runtimeTargetAgentDeliveryAuthority{repository: repository, pepper: pepper, now: time.Now, random: rand.Reader}
}

func (a runtimeTargetAgentDeliveryAuthority) CreateDeliveryGrant(ctx context.Context, request moduleapi.AgentDeliveryGrantRequest) (moduleapi.AgentDeliveryGrant, error) {
	if a.repository == nil || !validAgentDeliveryGrantRequest(request, a.currentTime()) || len(a.enrollmentPepper()) == 0 {
		return moduleapi.AgentDeliveryGrant{}, errAgentDeliveryAuthorityRejected
	}

	generation, err := a.repository.ReadCurrentAgentTrustGeneration(ctx, request.TargetID, request.AgentID)
	if err != nil || generation.Generation != request.Generation || generation.Status != string(moduleapi.RuntimeTargetAgentStatusPending) || generation.Identity.ProviderID != runtimeTargetAgentEnrollmentProviderID {
		return moduleapi.AgentDeliveryGrant{}, errAgentDeliveryAuthorityRejected
	}
	grantID, err := a.randomValue()
	if err != nil {
		return moduleapi.AgentDeliveryGrant{}, errAgentDeliveryAuthorityRejected
	}
	pepper := a.enrollmentPepper()
	verifier := tokenVerifier(deriveBootstrapToken(pepper, grantID), pepper)
	created, err := a.repository.CreatePendingAgentDeliveryGrant(ctx, store.AgentDeliveryGrant{
		GenerationID:          generation.ID,
		GrantID:               grantID,
		TokenVerifier:         verifier,
		ExpectedAutomationID:  strings.TrimSpace(request.ExpectedAutomationID),
		DockerInstallationRef: strings.TrimSpace(request.DockerInstallationRef),
		ExpiresAt:             request.ExpiresAt.UTC(),
	}, a.currentTime())
	if err != nil {
		return moduleapi.AgentDeliveryGrant{}, normalizeAgentDeliveryAuthorityError(err)
	}

	result := agentDeliveryGrantFromStore(created)
	result.TargetID = generation.Identity.TargetID
	result.AgentID = generation.Identity.AgentID
	result.Generation = generation.Generation
	return result, nil
}

func (a runtimeTargetAgentDeliveryAuthority) HandoffDeliveryGrant(ctx context.Context, actor moduleapi.DeliveryActor, request moduleapi.AgentDeliveryHandoffRequest) (moduleapi.AgentDeliveryHandoffMaterial, error) {
	if a.repository == nil || moduleapi.ValidateDeliveryActor(actor) != nil || !validAgentDeliveryHandoffRequest(request) || len(a.enrollmentPepper()) == 0 {
		return moduleapi.AgentDeliveryHandoffMaterial{}, errAgentDeliveryAuthorityRejected
	}

	handoffID, err := a.randomValue()
	if err != nil {
		return moduleapi.AgentDeliveryHandoffMaterial{}, errAgentDeliveryAuthorityRejected
	}
	accepted, err := a.repository.AcceptAgentDeliveryHandoff(ctx, request.GrantID, actor.ID, handoffID, a.currentTime())
	if err != nil {
		return moduleapi.AgentDeliveryHandoffMaterial{}, normalizeAgentDeliveryAuthorityError(err)
	}
	return moduleapi.AgentDeliveryHandoffMaterial{
		GrantID:        accepted.GrantID,
		HandoffID:      accepted.HandoffID,
		BootstrapToken: deriveBootstrapToken(a.enrollmentPepper(), accepted.GrantID),
		ExpiresAt:      accepted.ExpiresAt,
	}, nil
}

func (a runtimeTargetAgentDeliveryAuthority) RecordDeliveryReceipt(ctx context.Context, actor moduleapi.DeliveryActor, request moduleapi.AgentDeliveryReceiptRequest) (moduleapi.AgentDeliveryReceipt, error) {
	if a.repository == nil || moduleapi.ValidateDeliveryActor(actor) != nil || !validAgentDeliveryReceiptRequest(request, a.currentTime()) {
		return moduleapi.AgentDeliveryReceipt{}, errAgentDeliveryAuthorityRejected
	}
	recorded, replay, err := a.repository.RecordAgentDeliveryReceipt(ctx, store.AgentDeliveryReceipt{
		GrantID:               strings.TrimSpace(request.GrantID),
		ReceiptID:             strings.TrimSpace(request.ReceiptID),
		ProtocolVersion:       strings.TrimSpace(request.ProtocolVersion),
		AutomationID:          strings.TrimSpace(actor.ID),
		HandoffID:             strings.TrimSpace(request.HandoffID),
		AssertedDeliveredAt:   request.AssertedDeliveredAt.UTC(),
		DockerInstallationRef: strings.TrimSpace(request.DockerInstallationRef),
		DockerSecretRef:       strings.TrimSpace(request.DockerSecretRef),
		PayloadFingerprint:    strings.TrimSpace(request.PayloadFingerprint),
	}, a.currentTime())
	if err != nil {
		return moduleapi.AgentDeliveryReceipt{}, normalizeAgentDeliveryAuthorityError(err)
	}
	return agentDeliveryReceiptFromStore(recorded, replay), nil
}

func (a runtimeTargetAgentDeliveryAuthority) enrollmentPepper() []byte {
	if a.pepper == nil {
		return nil
	}
	return a.pepper.Pepper()
}

func (a runtimeTargetAgentDeliveryAuthority) currentTime() time.Time {
	if a.now == nil {
		return time.Now().UTC()
	}
	return a.now().UTC()
}

func (a runtimeTargetAgentDeliveryAuthority) randomValue() (string, error) {
	reader := a.random
	if reader == nil {
		reader = rand.Reader
	}
	value := make([]byte, agentDeliveryTokenBytes)
	if _, err := io.ReadFull(reader, value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func tokenVerifier(token string, pepper []byte) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(token))
	_, _ = hash.Write(pepper)
	return hex.EncodeToString(hash.Sum(nil))
}

// deriveBootstrapToken 从随机 256 位 grant ID 与安装级 pepper 派生 256 位密码学伪随机引导材料。
// 该推导使 Runtime Target 能在重启后仅在成功固化一次性交接状态之后重建令牌，而无需持久化明文。
func deriveBootstrapToken(pepper []byte, grantID string) string {
	mac := hmac.New(sha256.New, pepper)
	_, _ = mac.Write([]byte(grantID))
	return hex.EncodeToString(mac.Sum(nil))
}

func validAgentDeliveryGrantRequest(request moduleapi.AgentDeliveryGrantRequest, now time.Time) bool {
	return request.TargetID > 0 && strings.TrimSpace(request.AgentID) != "" && request.Generation > 0 && strings.TrimSpace(request.ExpectedAutomationID) != "" && strings.TrimSpace(request.DockerInstallationRef) != "" && request.ExpiresAt.After(now)
}

func validAgentDeliveryHandoffRequest(request moduleapi.AgentDeliveryHandoffRequest) bool {
	return strings.TrimSpace(request.GrantID) != ""
}

func validAgentDeliveryReceiptRequest(request moduleapi.AgentDeliveryReceiptRequest, now time.Time) bool {
	return strings.TrimSpace(request.GrantID) != "" && strings.TrimSpace(request.ReceiptID) != "" && strings.TrimSpace(request.ProtocolVersion) == "graft.delivery-receipt.v1" && strings.TrimSpace(request.HandoffID) != "" && !request.AssertedDeliveredAt.IsZero() && !request.AssertedDeliveredAt.After(now) && strings.TrimSpace(request.DockerInstallationRef) != "" && strings.TrimSpace(request.DockerSecretRef) != "" && validLowerSHA256Hex(request.PayloadFingerprint)
}

func validLowerSHA256Hex(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func agentDeliveryGrantFromStore(grant store.AgentDeliveryGrant) moduleapi.AgentDeliveryGrant {
	return moduleapi.AgentDeliveryGrant{GrantID: grant.GrantID, ExpectedAutomationID: grant.ExpectedAutomationID, DockerInstallationRef: grant.DockerInstallationRef, ExpiresAt: grant.ExpiresAt}
}

func agentDeliveryReceiptFromStore(receipt store.AgentDeliveryReceipt, replay bool) moduleapi.AgentDeliveryReceipt {
	return moduleapi.AgentDeliveryReceipt{GrantID: receipt.GrantID, ReceiptID: receipt.ReceiptID, ProtocolVersion: receipt.ProtocolVersion, AutomationID: receipt.AutomationID, HandoffID: receipt.HandoffID, AssertedDeliveredAt: receipt.AssertedDeliveredAt, AcceptedAt: receipt.AcceptedAt, DockerInstallationRef: receipt.DockerInstallationRef, DockerSecretRef: receipt.DockerSecretRef, PayloadFingerprint: receipt.PayloadFingerprint, Replay: replay}
}

func normalizeAgentDeliveryAuthorityError(err error) error {
	if errors.Is(err, store.ErrAgentDeliveryRejected) || errors.Is(err, store.ErrAgentTrustNotFound) {
		return errAgentDeliveryAuthorityRejected
	}
	return err
}
