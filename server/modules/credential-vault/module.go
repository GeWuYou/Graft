package credentialvault

import (
	"errors"

	"graft/server/internal/config"
	containerdi "graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

// Module 拥有 Vault PKI adapter 注册，但自身绝不存储或传输密码材料。
type Module struct {
	configuration config.CredentialVaultConfig
	adapter       VaultPKIAdapter
}

// NewModule 构造 Credential Vault 生命周期 owner。
// nil adapter 是有意的：启用配置时只暴露 unavailable issuer；关闭配置时完全不暴露
// AgentCertificateIssuer capability。
func NewModule(configuration config.CredentialVaultConfig, adapter VaultPKIAdapter) *Module {
	return &Module{configuration: configuration, adapter: adapter}
}

// Register 仅在 Vault 集成启用时注册 AgentCertificateIssuer。
// Runtime Target 的登记与激活不在本模块注册范围内；生产 adapter 接线前必须 fail-closed。
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || ctx == nil || ctx.Services == nil {
		return errors.New("credential vault module context services are unavailable")
	}
	if !m.configuration.Enabled {
		return nil
	}
	issuer := m.issuer()
	return ctx.Services.RegisterSingleton((*moduleapi.AgentCertificateIssuer)(nil), func(containerdi.Resolver) (any, error) {
		return issuer, nil
	})
}

// Boot 不启动 Vault client，因为当前 foundation 不包含生产 Vault 集成。
func (*Module) Boot(*module.Context) error { return nil }

// Shutdown 在当前 foundational implementation 中不拥有 live external client。
func (*Module) Shutdown(*module.Context) error { return nil }

func (m *Module) issuer() moduleapi.AgentCertificateIssuer {
	if m.adapter != nil {
		return m.adapter
	}
	return unavailableAgentCertificateIssuer{}
}

var _ module.Module = (*Module)(nil)
