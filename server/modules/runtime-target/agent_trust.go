package runtimetarget

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
	contract "graft/server/modules/runtime-target/contract"
	store "graft/server/modules/runtime-target/store"
)

const (
	// AgentTrustAuditActionRegistration 记录操作者请求创建 Agent 登记世代的事实。
	AgentTrustAuditActionRegistration = "runtime_target.agent_registration"
	// AgentTrustAuditActionRotation 记录操作者请求轮换 Agent 信任世代的事实。
	AgentTrustAuditActionRotation = "runtime_target.agent_trust_rotation"
	// AgentTrustAuditActionRevocation 记录操作者撤销或重置 Agent 信任世代的事实。
	AgentTrustAuditActionRevocation = "runtime_target.agent_revocation"
)

// AgentTrustOperatorAuthorizer 是 Runtime Target Agent 危险操作的服务层授权钩子。
// HTTP 路由在未来接入时可复用它，但 Vault 调用方不能绕过该边界直接把操作者输入写入仓储。
type AgentTrustOperatorAuthorizer interface {
	AuthorizeAgentTrustOperator(context.Context) error
}

type agentTrustOperatorAuthorizer struct {
	authorizer moduleapi.Authorizer
}

// NewAgentTrustOperatorAuthorizer 创建 Runtime Target 的 Agent 信任操作授权钩子。
func NewAgentTrustOperatorAuthorizer(authorizer moduleapi.Authorizer) AgentTrustOperatorAuthorizer {
	return agentTrustOperatorAuthorizer{authorizer: authorizer}
}

func (a agentTrustOperatorAuthorizer) AuthorizeAgentTrustOperator(ctx context.Context) error {
	if a.authorizer == nil {
		return errors.New("runtime target agent trust authorizer is unavailable")
	}
	auth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || auth.User == nil {
		return errors.New("runtime target agent trust operator is unauthenticated")
	}
	return a.authorizer.Authorize(ctx, auth, contract.ManagePermission)
}

// NewAgentTrustAuditEvent 创建不包含秘密材料的 Runtime Target Agent 生命周期审计事件。
func NewAgentTrustAuditEvent(action string, operator *moduleapi.CurrentUser, binding moduleapi.RuntimeTargetAgentBinding, success bool, message string) (moduleapi.AuditEvent, error) {
	if !validAgentTrustAuditAction(action) || binding.TargetID < 1 || strings.TrimSpace(binding.AgentID) == "" || binding.Generation < 1 {
		return moduleapi.AuditEvent{}, errors.New("runtime target agent trust audit input is invalid")
	}
	metadata := map[string]any{
		"agent_id":             binding.AgentID,
		"generation":           binding.Generation,
		"identity_id":          binding.IdentityID,
		"status":               binding.Status,
		"trust_bundle_version": binding.TrustBundleVersion,
	}
	return moduleapi.AuditEvent{
		Kind:         moduleapi.AuditEventKindDomain,
		Operator:     operator,
		Action:       action,
		ResourceType: "runtime_target_agent_generation",
		ResourceID:   strconv.FormatInt(binding.TargetID, 10) + ":" + binding.AgentID + ":" + strconv.FormatInt(binding.Generation, 10),
		ResourceName: binding.AgentID,
		StatusCode:   http.StatusOK,
		Success:      success,
		Message:      strings.TrimSpace(message),
		Metadata:     metadata,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

type runtimeTargetAgentBindingReader struct {
	repository *store.SQLRepository
}

func (r runtimeTargetAgentBindingReader) ReadAgentBinding(ctx context.Context, targetID int64, agentID string) (moduleapi.RuntimeTargetAgentBinding, error) {
	if r.repository == nil {
		return moduleapi.RuntimeTargetAgentBinding{}, errors.New("runtime target agent trust repository is unavailable")
	}
	generation, err := r.repository.ReadCurrentAgentTrustGeneration(ctx, targetID, agentID)
	if err != nil {
		return moduleapi.RuntimeTargetAgentBinding{}, err
	}
	return moduleapi.RuntimeTargetAgentBinding{
		IdentityID:           generation.Identity.IdentityID,
		TargetID:             generation.Identity.TargetID,
		AgentID:              generation.Identity.AgentID,
		ProviderID:           generation.Identity.ProviderID,
		BuilderScope:         generation.Identity.BuilderScope,
		CapabilityProfile:    generation.Identity.CapabilityProfile,
		CapabilityVersion:    generation.Identity.CapabilityVersion,
		Generation:           generation.Generation,
		CertificateSerial:    generation.CertificateSerial,
		PublicKeyFingerprint: generation.PublicKeyFingerprint,
		TrustBundleVersion:   generation.TrustBundleVersion,
		ExpiresAt:            generation.ExpiresAt,
		RevokedAt:            generation.RevokedAt,
		Status:               moduleapi.RuntimeTargetAgentStatus(generation.Status),
	}, nil
}

func validAgentTrustAuditAction(action string) bool {
	switch action {
	case AgentTrustAuditActionRegistration, AgentTrustAuditActionRotation, AgentTrustAuditActionRevocation:
		return true
	default:
		return false
	}
}
