package credentialvault

import (
	"context"
	"errors"

	"graft/server/internal/moduleapi"
)

// ErrAgentCertificateIssuerUnavailable 表示当前进程没有已配置的 Vault PKI adapter。
// 调用方必须将其视为 fail-closed 的服务不可用状态，不能以本地凭据替代。
var ErrAgentCertificateIssuerUnavailable = errors.New("agent certificate issuer is unavailable")

// VaultPKIAdapter 是 Credential Vault 拥有的真实 Vault PKI 集成 adapter seam。
// adapter 只能返回非秘密的 moduleapi DTO；私钥、PEM 和 enrollment secret 必须留在 Vault 或部署交付
// 通道中，不能由本模块 materialize。
type VaultPKIAdapter interface {
	moduleapi.AgentCertificateIssuer
}

// unavailableAgentCertificateIssuer 使已启用但尚未实现的 Vault 集成显式 fail-closed。
// 它刻意不含持久化或 fallback 行为，避免部分配置的部署在本地签发或伪造 Agent 证书。
type unavailableAgentCertificateIssuer struct{}

// IssueCSR 在真实 Vault PKI adapter 注册前拒绝签发。
func (unavailableAgentCertificateIssuer) IssueCSR(context.Context, moduleapi.AgentCertificateIssuanceRequest) (moduleapi.IssuedAgentCertificate, error) {
	return moduleapi.IssuedAgentCertificate{}, ErrAgentCertificateIssuerUnavailable
}

// ReadTrustBundle 在真实 Vault PKI adapter 注册前拒绝信任束读取。
func (unavailableAgentCertificateIssuer) ReadTrustBundle(context.Context, moduleapi.TrustBundleRequest) (moduleapi.TrustBundleReference, error) {
	return moduleapi.TrustBundleReference{}, ErrAgentCertificateIssuerUnavailable
}

// RevokeCertificate 在真实 Vault PKI adapter 注册前拒绝证书撤销。
func (unavailableAgentCertificateIssuer) RevokeCertificate(context.Context, moduleapi.AgentCertificateRevocation) error {
	return ErrAgentCertificateIssuerUnavailable
}

var _ VaultPKIAdapter = unavailableAgentCertificateIssuer{}
