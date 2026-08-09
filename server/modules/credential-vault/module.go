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
// nil adapter 是有意的：启用配置时只暴露 unavailable authority；关闭配置时完全不暴露
// MachineIdentityAuthority capability。
func NewModule(configuration config.CredentialVaultConfig, adapter VaultPKIAdapter) *Module {
	return &Module{configuration: configuration, adapter: adapter}
}

// Register 仅在 Vault 集成启用时注册 canonical machine-identity capability。
// 在生产 adapter 接线前，已注册 provider 必须以 ErrMachineIdentityAuthorityUnavailable fail-closed。
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || ctx == nil || ctx.Services == nil {
		return errors.New("credential vault module context services are unavailable")
	}
	if !m.configuration.Enabled {
		return nil
	}
	authority := m.authority()
	return ctx.Services.RegisterSingleton((*moduleapi.MachineIdentityAuthority)(nil), func(containerdi.Resolver) (any, error) {
		return authority, nil
	})
}

// Boot 不启动 Vault client，因为当前 foundation 不包含生产 Vault 集成。
func (*Module) Boot(*module.Context) error { return nil }

// Shutdown 在当前 foundational implementation 中不拥有 live external client。
func (*Module) Shutdown(*module.Context) error { return nil }

func (m *Module) authority() moduleapi.MachineIdentityAuthority {
	if m.adapter != nil {
		return m.adapter
	}
	return unavailableMachineIdentityAuthority{}
}

var _ module.Module = (*Module)(nil)
