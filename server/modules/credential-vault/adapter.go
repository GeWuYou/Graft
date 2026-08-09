package credentialvault

import (
	"context"
	"errors"

	"graft/server/internal/moduleapi"
)

// ErrMachineIdentityAuthorityUnavailable 表示当前进程没有已配置的 Vault PKI adapter。
// 调用方必须将其视为 fail-closed 的服务不可用状态，不能以本地凭据替代。
var ErrMachineIdentityAuthorityUnavailable = errors.New("machine identity authority is unavailable")

// VaultPKIAdapter 是 Credential Vault 拥有的真实 Vault PKI 集成 adapter seam。
// adapter 只能返回非秘密的 moduleapi DTO；私钥、PEM 和 enrollment secret 必须留在 Vault 或部署交付
// 通道中，不能由本模块 materialize。
type VaultPKIAdapter interface {
	moduleapi.MachineIdentityAuthority
}

// unavailableMachineIdentityAuthority 使已启用但尚未实现的 Vault 集成显式 fail-closed。
// 它刻意不含持久化或 fallback 行为，避免部分配置的部署创建本地信任的 Agent identity。
type unavailableMachineIdentityAuthority struct{}

// CreateEnrollment 在真实 Vault PKI adapter 注册前拒绝 enrollment。
func (unavailableMachineIdentityAuthority) CreateEnrollment(context.Context, moduleapi.MachineEnrollmentRequest) (moduleapi.MachineEnrollment, error) {
	return moduleapi.MachineEnrollment{}, ErrMachineIdentityAuthorityUnavailable
}

// ActivateGeneration 在真实 Vault PKI adapter 注册前拒绝 activation。
func (unavailableMachineIdentityAuthority) ActivateGeneration(context.Context, moduleapi.MachineIdentityActivation) error {
	return ErrMachineIdentityAuthorityUnavailable
}

// RotateGeneration 在真实 Vault PKI adapter 注册前拒绝 rotation。
func (unavailableMachineIdentityAuthority) RotateGeneration(context.Context, moduleapi.MachineIdentityRotationRequest) (moduleapi.MachineEnrollment, error) {
	return moduleapi.MachineEnrollment{}, ErrMachineIdentityAuthorityUnavailable
}

// RevokeGeneration 在真实 Vault PKI adapter 注册前拒绝 revocation。
func (unavailableMachineIdentityAuthority) RevokeGeneration(context.Context, moduleapi.MachineIdentityRevocation) error {
	return ErrMachineIdentityAuthorityUnavailable
}

// ReadTrustBundle 在真实 Vault PKI adapter 注册前拒绝 trust bundle 读取。
func (unavailableMachineIdentityAuthority) ReadTrustBundle(context.Context, moduleapi.TrustBundleRequest) (moduleapi.TrustBundleReference, error) {
	return moduleapi.TrustBundleReference{}, ErrMachineIdentityAuthorityUnavailable
}

var _ VaultPKIAdapter = unavailableMachineIdentityAuthority{}
