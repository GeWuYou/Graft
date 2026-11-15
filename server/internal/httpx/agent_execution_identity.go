package httpx

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/moduleapi"
)

var errAgentExecutionIdentityRejected = errors.New("agent execution identity rejected")

func resolveAgentExecutionIdentity(bindings moduleapi.RuntimeTargetAgentBindingReader, logger *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		identity, ok := AgentMTLSIdentityFromGinContext(ctx)
		if !ok {
			abortAgentLedgerIdentity(ctx, logger)
			return
		}
		binding, err := bindings.ReadAgentBinding(ctx.Request.Context(), identity.TargetID, identity.AgentID)
		if err != nil || !agentCertificateMatchesBinding(identity, binding) ||
			(identity.IdentityID != "" && identity.IdentityID != binding.IdentityID) ||
			(identity.Generation != 0 && identity.Generation != binding.Generation) {
			abortAgentLedgerIdentity(ctx, logger)
			return
		}
		// URI SAN 只标识稳定主体；世代由与证书证据精确匹配的活动绑定派生。
		identity.IdentityID, identity.Generation = binding.IdentityID, binding.Generation
		ctx.Set(agentIdentityContextKey, identity)
		ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), agentIdentityContextValue{}, identity))
		ctx.Next()
	}
}

func agentCertificateMatchesBinding(identity AgentMTLSIdentity, binding moduleapi.RuntimeTargetAgentBinding) bool {
	return agentTrustBindingActive(binding) && identity.TargetID > 0 && identity.AgentID != "" &&
		binding.TargetID == identity.TargetID && binding.AgentID == identity.AgentID &&
		identity.CertificateSerial != "" && identity.PublicKeyFingerprint != "" &&
		binding.CertificateSerial == identity.CertificateSerial && binding.PublicKeyFingerprint == identity.PublicKeyFingerprint
}

func agentTrustBindingActive(binding moduleapi.RuntimeTargetAgentBinding) bool {
	return binding.Status == moduleapi.RuntimeTargetAgentStatusActive && binding.RevokedAt == nil && binding.ExpiresAt.After(time.Now()) &&
		strings.TrimSpace(binding.IdentityID) != "" && binding.Generation > 0
}
