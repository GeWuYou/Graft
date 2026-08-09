package credentialvault

import (
	"database/sql"
	"fmt"

	"graft/server/internal/config"
	"graft/server/internal/module"
	credentialvaultcontract "graft/server/modules/credential-vault/contract"
)

// NewModuleSpec 声明 Credential Vault 编译期模块及其非秘密部署配置 seam。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            credentialvaultcontract.ModuleID,
		MigrationPath: []string{"modules/credential-vault/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			runtimeConfig, err := module.ResolveService[*config.Config](ctx.Services, (*config.Config)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve runtime config: %w", err)
			}
			db, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			store, err := NewSQLIssuanceStateStore(db)
			if err != nil {
				return nil, fmt.Errorf("build issuance state store: %w", err)
			}
			adapter, err := NewVaultPKIClient(runtimeConfig.CredentialVault, store)
			if err != nil {
				if !runtimeConfig.CredentialVault.Enabled {
					return NewModule(runtimeConfig.CredentialVault, nil), nil
				}
				return nil, fmt.Errorf("build vault PKI client: %w", err)
			}
			return NewModule(runtimeConfig.CredentialVault, adapter), nil
		}),
	}
}
