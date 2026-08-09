package credentialvault

import (
	"fmt"

	"graft/server/internal/config"
	"graft/server/internal/module"
	credentialvaultcontract "graft/server/modules/credential-vault/contract"
)

// NewModuleSpec 声明 Credential Vault 编译期模块及其非秘密部署配置 seam。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID: credentialvaultcontract.ModuleID,
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			runtimeConfig, err := module.ResolveService[*config.Config](ctx.Services, (*config.Config)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve runtime config: %w", err)
			}
			return NewModule(runtimeConfig.CredentialVault, nil), nil
		}),
	}
}
