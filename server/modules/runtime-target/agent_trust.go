package runtimetarget

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"graft/server/internal/event"
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
	// runtimeTargetAgentEnrollmentProviderID 是当前唯一已实现交付路径的稳定提供方标识。
	runtimeTargetAgentEnrollmentProviderID = "docker"
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
		StatusCode:   agentTrustAuditStatusCode(success),
		Success:      success,
		Message:      strings.TrimSpace(message),
		Metadata:     metadata,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

// agentTrustAuditStatusCode 将 Agent 信任生命周期审计结果映射为一致的 HTTP 状态码。
func agentTrustAuditStatusCode(success bool) int {
	if success {
		return http.StatusOK
	}
	return http.StatusUnprocessableEntity
}

type runtimeTargetAgentBindingReader struct {
	repository *store.SQLRepository
}

// runtimeTargetAgentEnrollmentAuthority 保持 Agent 业务归属与证书签发事实分离。
// Vault 和交付边界只能提供非秘密证据，不能藉由这些证据建立或激活 Runtime Target 绑定。
type runtimeTargetAgentEnrollmentAuthority struct {
	repository *store.SQLRepository
	events     event.TransactionalPublisher
	now        func() time.Time
}

func newRuntimeTargetAgentEnrollmentAuthority(repository *store.SQLRepository, publishers ...event.TransactionalPublisher) moduleapi.AgentEnrollmentAuthority {
	var publisher event.TransactionalPublisher
	if len(publishers) > 0 {
		publisher = publishers[0]
	}
	return runtimeTargetAgentEnrollmentAuthority{repository: repository, events: publisher, now: time.Now}
}

func (a runtimeTargetAgentEnrollmentAuthority) CreateEnrollment(ctx context.Context, request moduleapi.AgentEnrollmentRequest) (moduleapi.AgentEnrollment, error) {
	if a.repository == nil || !validAgentEnrollmentRequest(request, a.currentTime()) {
		return moduleapi.AgentEnrollment{}, errors.New("runtime target agent enrollment request is invalid")
	}
	identity := agentTrustIdentityFromEnrollmentRequest(request)
	// 此预检只提供稳定的重复登记反馈；持久化事务仍是并发创建时身份与世代的最终约束。
	if _, err := a.repository.ReadCurrentAgentTrustGeneration(ctx, identity.TargetID, identity.AgentID); err == nil {
		return moduleapi.AgentEnrollment{}, errors.New("runtime target agent enrollment already exists")
	} else if !errors.Is(err, store.ErrAgentTrustNotFound) {
		return moduleapi.AgentEnrollment{}, err
	}
	created, err := a.createPendingGeneration(ctx, identity, 1, request.EnrollmentRef, request.TrustBundle, request.ExpiresAt)
	if err != nil {
		return moduleapi.AgentEnrollment{}, err
	}
	return agentEnrollmentFromGeneration(created), nil
}

func (a runtimeTargetAgentEnrollmentAuthority) ActivateGeneration(ctx context.Context, activation moduleapi.AgentEnrollmentActivation) error {
	if a.repository == nil || !validAgentEnrollmentActivation(activation) {
		return errors.New("runtime target agent enrollment activation is invalid")
	}
	current, err := a.repository.ReadCurrentAgentTrustGeneration(ctx, activation.TargetID, activation.AgentID)
	if err != nil {
		return err
	}
	if current.Identity.IdentityID != strings.TrimSpace(activation.IdentityID) || current.Generation != activation.Generation || current.Status != string(moduleapi.RuntimeTargetAgentStatusPending) {
		return errors.New("runtime target agent enrollment activation does not match a pending generation")
	}
	return a.repository.ActivateAgentTrustGeneration(ctx, activation.TargetID, activation.AgentID, activation.Generation, strings.TrimSpace(activation.CertificateIssuer), strings.TrimSpace(activation.CertificateSerial), strings.TrimSpace(activation.PublicKeyFingerprint), 0, a.currentTime())
}

func (a runtimeTargetAgentEnrollmentAuthority) RotateGeneration(ctx context.Context, request moduleapi.AgentEnrollmentRotationRequest) (moduleapi.AgentEnrollment, error) {
	if a.repository == nil || !validAgentEnrollmentRotationRequest(request, a.currentTime()) {
		return moduleapi.AgentEnrollment{}, errors.New("runtime target agent enrollment rotation request is invalid")
	}
	current, err := a.repository.ReadCurrentAgentTrustGeneration(ctx, request.TargetID, request.AgentID)
	if err != nil {
		return moduleapi.AgentEnrollment{}, err
	}
	if current.Identity.IdentityID != strings.TrimSpace(request.IdentityID) || current.Status != string(moduleapi.RuntimeTargetAgentStatusActive) || !sameEnrollmentScope(current.Identity, request) {
		return moduleapi.AgentEnrollment{}, errors.New("runtime target agent enrollment rotation does not match the active identity")
	}
	created, err := a.createPendingGeneration(ctx, current.Identity, current.Generation+1, request.EnrollmentRef, request.TrustBundle, request.ExpiresAt)
	if err != nil {
		return moduleapi.AgentEnrollment{}, err
	}
	return agentEnrollmentFromGeneration(created), nil
}

func (a runtimeTargetAgentEnrollmentAuthority) RevokeGeneration(ctx context.Context, revocation moduleapi.AgentEnrollmentRevocation) error {
	if a.repository == nil || !validAgentEnrollmentRevocation(revocation) {
		return errors.New("runtime target agent enrollment revocation is invalid")
	}
	if a.events == nil {
		return errors.New("runtime target agent certificate revocation publisher is unavailable")
	}
	current, err := a.repository.ReadCurrentAgentTrustGeneration(ctx, revocation.TargetID, revocation.AgentID)
	if err != nil {
		return err
	}
	if current.Identity.IdentityID != strings.TrimSpace(revocation.IdentityID) {
		return errors.New("runtime target agent enrollment revocation does not match the identity")
	}
	now := a.currentTime()
	return a.repository.RunInTransaction(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		serial, changed, err := a.repository.RevokeAgentTrustGenerationTx(txCtx, tx, revocation.TargetID, revocation.AgentID, revocation.Generation, strings.TrimSpace(revocation.Reason), 0, now)
		if err != nil || !changed || a.events == nil {
			return err
		}
		payload, err := json.Marshal(contract.AgentCertificateRevocationEvent{IdentityID: strings.TrimSpace(revocation.IdentityID), TargetID: revocation.TargetID, AgentID: strings.TrimSpace(revocation.AgentID), Generation: revocation.Generation, CertificateSerial: strings.TrimSpace(serial), Reason: strings.TrimSpace(revocation.Reason)})
		if err != nil {
			return fmt.Errorf("encode agent certificate revocation event: %w", err)
		}
		id := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s:%d", strings.TrimSpace(revocation.IdentityID), revocation.TargetID, strings.TrimSpace(revocation.AgentID), revocation.Generation)))
		_, err = a.events.PublishTx(txCtx, tx, event.Event{ID: hex.EncodeToString(id[:]), Type: contract.AgentCertificateRevocationEventType, Version: 1, Source: contract.ModuleID, Payload: payload, IdempotencyKey: hex.EncodeToString(id[:])}, event.PublishOptions{Delivery: event.DeliveryDurable})
		return err
	})
}

func (a runtimeTargetAgentEnrollmentAuthority) createPendingGeneration(ctx context.Context, identity store.AgentTrustIdentity, generation int64, enrollmentRef string, trustBundle moduleapi.TrustBundleReference, expiresAt time.Time) (store.AgentTrustGeneration, error) {
	return a.repository.CreatePendingAgentTrustGeneration(ctx, identity, store.AgentTrustGeneration{
		Generation:         generation,
		EnrollmentRef:      strings.TrimSpace(enrollmentRef),
		TrustBundleRef:     strings.TrimSpace(trustBundle.Reference),
		TrustBundleVersion: strings.TrimSpace(trustBundle.Version),
		ExpiresAt:          expiresAt.UTC(),
	})
}

func (a runtimeTargetAgentEnrollmentAuthority) currentTime() time.Time {
	if a.now == nil {
		return time.Now().UTC()
	}
	return a.now().UTC()
}

func agentTrustIdentityFromEnrollmentRequest(request moduleapi.AgentEnrollmentRequest) store.AgentTrustIdentity {
	return store.AgentTrustIdentity{
		IdentityID:        stableAgentIdentityID(request.TargetID, request.AgentID),
		TargetID:          request.TargetID,
		AgentID:           strings.TrimSpace(request.AgentID),
		ProviderID:        strings.TrimSpace(request.ProviderID),
		BuilderScope:      strings.TrimSpace(request.BuilderScope),
		CapabilityProfile: strings.TrimSpace(request.CapabilityProfile),
		CapabilityVersion: strings.TrimSpace(request.CapabilityVersion),
		ImageDigest:       strings.TrimSpace(request.ImageDigest),
		AgentVersion:      strings.TrimSpace(request.AgentVersion),
	}
}

func stableAgentIdentityID(targetID int64, agentID string) string {
	return fmt.Sprintf("runtime-target:%d:agent:%s", targetID, strings.TrimSpace(agentID))
}

func agentEnrollmentFromGeneration(generation store.AgentTrustGeneration) moduleapi.AgentEnrollment {
	return moduleapi.AgentEnrollment{
		IdentityID:           generation.Identity.IdentityID,
		TargetID:             generation.Identity.TargetID,
		AgentID:              generation.Identity.AgentID,
		ProviderID:           generation.Identity.ProviderID,
		BuilderScope:         generation.Identity.BuilderScope,
		CapabilityProfile:    generation.Identity.CapabilityProfile,
		CapabilityVersion:    generation.Identity.CapabilityVersion,
		Generation:           generation.Generation,
		EnrollmentRef:        generation.EnrollmentRef,
		ExpiresAt:            generation.ExpiresAt,
		TrustBundleVersion:   generation.TrustBundleVersion,
		CertificateSerial:    generation.CertificateSerial,
		PublicKeyFingerprint: generation.PublicKeyFingerprint,
		Status:               moduleapi.RuntimeTargetAgentStatus(generation.Status),
	}
}

func validAgentEnrollmentRequest(request moduleapi.AgentEnrollmentRequest, now time.Time) bool {
	return validAgentEnrollmentScope(request.TargetID, request.AgentID, request.ProviderID, request.BuilderScope, request.CapabilityProfile, request.CapabilityVersion) && validAgentEnrollmentAttestation(request.EnrollmentRef, request.TrustBundle, request.ExpiresAt, now) && validAgentPackageAttestation(request.ImageDigest, request.AgentVersion)
}

func validAgentEnrollmentActivation(activation moduleapi.AgentEnrollmentActivation) bool {
	return strings.TrimSpace(activation.IdentityID) != "" && activation.TargetID > 0 && strings.TrimSpace(activation.AgentID) != "" && activation.Generation > 0 && strings.TrimSpace(activation.CertificateIssuer) != "" && strings.TrimSpace(activation.CertificateSerial) != "" && strings.TrimSpace(activation.PublicKeyFingerprint) != ""
}

func validAgentEnrollmentRotationRequest(request moduleapi.AgentEnrollmentRotationRequest, now time.Time) bool {
	return strings.TrimSpace(request.IdentityID) != "" && validAgentEnrollmentScope(request.TargetID, request.AgentID, request.ProviderID, request.BuilderScope, request.CapabilityProfile, request.CapabilityVersion) && validAgentEnrollmentAttestation(request.EnrollmentRef, request.TrustBundle, request.ExpiresAt, now) && strings.TrimSpace(request.Reason) != ""
}

func validAgentEnrollmentScope(targetID int64, agentID, providerID, builderScope, capabilityProfile, capabilityVersion string) bool {
	return targetID > 0 && validAgentSPIFFEPathSegment(agentID) && strings.TrimSpace(providerID) == runtimeTargetAgentEnrollmentProviderID && strings.TrimSpace(builderScope) != "" && strings.TrimSpace(capabilityProfile) != "" && strings.TrimSpace(capabilityVersion) != ""
}

func validAgentSPIFFEPathSegment(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, character := range value {
		if !validAgentSPIFFEPathCharacter(character) {
			return false
		}
	}
	return true
}

func validAgentSPIFFEPathCharacter(character rune) bool {
	return (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '.' || character == '-' || character == '_'
}

func validAgentEnrollmentAttestation(enrollmentRef string, trustBundle moduleapi.TrustBundleReference, expiresAt, now time.Time) bool {
	return strings.TrimSpace(enrollmentRef) != "" && strings.TrimSpace(trustBundle.Reference) != "" && strings.TrimSpace(trustBundle.Version) != "" && trustBundle.ExpiresAt.After(now) && expiresAt.After(now)
}

// validAgentPackageAttestation 将 Docker Agent 绑定到可审计的不可变 OCI image，而不是可变标签或空版本。
func validAgentPackageAttestation(imageDigest, agentVersion string) bool {
	digest := strings.TrimSpace(imageDigest)
	if !strings.HasPrefix(digest, "sha256:") || len(digest) != len("sha256:")+64 || strings.TrimSpace(agentVersion) == "" {
		return false
	}
	for _, value := range digest[len("sha256:"):] {
		if (value < '0' || value > '9') && (value < 'a' || value > 'f') {
			return false
		}
	}
	return true
}

func validAgentEnrollmentRevocation(revocation moduleapi.AgentEnrollmentRevocation) bool {
	return strings.TrimSpace(revocation.IdentityID) != "" && revocation.TargetID > 0 && strings.TrimSpace(revocation.AgentID) != "" && revocation.Generation > 0 && strings.TrimSpace(revocation.Reason) != ""
}

func sameEnrollmentScope(identity store.AgentTrustIdentity, request moduleapi.AgentEnrollmentRotationRequest) bool {
	return identity.TargetID == request.TargetID && identity.AgentID == strings.TrimSpace(request.AgentID) && identity.ProviderID == strings.TrimSpace(request.ProviderID) && identity.BuilderScope == strings.TrimSpace(request.BuilderScope) && identity.CapabilityProfile == strings.TrimSpace(request.CapabilityProfile) && identity.CapabilityVersion == strings.TrimSpace(request.CapabilityVersion)
}

func (r runtimeTargetAgentBindingReader) ReadAgentBinding(ctx context.Context, targetID int64, agentID string) (moduleapi.RuntimeTargetAgentBinding, error) {
	if r.repository == nil {
		return moduleapi.RuntimeTargetAgentBinding{}, errors.New("runtime target agent trust repository is unavailable")
	}
	generation, err := r.repository.ReadCurrentAgentTrustGeneration(ctx, targetID, agentID)
	if err != nil {
		return moduleapi.RuntimeTargetAgentBinding{}, fmt.Errorf("read runtime target agent binding for target %d agent %q: %w", targetID, agentID, err)
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
