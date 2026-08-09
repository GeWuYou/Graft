package credentialvault

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"graft/server/internal/event"
	"graft/server/internal/moduleapi"
	runtimetargetcontract "graft/server/modules/runtime-target/contract"
)

const agentCertificateRevocationHandlerID = "credential-vault.agent-certificate-revocation"

// agentCertificateRevocationHandler 将 Runtime Target 的撤销事实交给 Vault issuer。
// Outbox 负责重试和重启恢复；处理器只使用事件稳定幂等键，不创建本地队列或后台协程。
type agentCertificateRevocationHandler struct {
	issuer moduleapi.AgentCertificateIssuer
}

func (h agentCertificateRevocationHandler) ID() string { return agentCertificateRevocationHandlerID }

func (h agentCertificateRevocationHandler) Types() []event.Type {
	return []event.Type{runtimetargetcontract.AgentCertificateRevocationEventType}
}

func (h agentCertificateRevocationHandler) Handle(ctx context.Context, incoming event.Event) error {
	if incoming.Version != 1 {
		return errors.New("unsupported agent certificate revocation event version")
	}
	var payload runtimetargetcontract.AgentCertificateRevocationEvent
	if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
		return err
	}
	serial := strings.TrimSpace(payload.CertificateSerial)
	if serial == "" {
		return nil
	}
	if h.issuer == nil {
		return errors.New("agent certificate issuer is unavailable")
	}
	idempotencyKey := strings.TrimSpace(incoming.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = strings.TrimSpace(incoming.ID)
	}
	if idempotencyKey == "" {
		return errors.New("agent certificate revocation event idempotency key is required")
	}
	return h.issuer.RevokeCertificate(ctx, moduleapi.AgentCertificateRevocation{
		IdentityID: strings.TrimSpace(payload.IdentityID), TargetID: payload.TargetID,
		AgentID: strings.TrimSpace(payload.AgentID), Generation: payload.Generation,
		CertificateSerial: serial, Reason: strings.TrimSpace(payload.Reason), IdempotencyKey: idempotencyKey,
	})
}

var _ event.Handler = agentCertificateRevocationHandler{}
