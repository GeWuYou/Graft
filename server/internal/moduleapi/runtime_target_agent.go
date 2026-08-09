package moduleapi

import (
	"context"
	"time"
)

// MachineIdentityAuthority 负责 Agent 信任的签发与生命周期；秘密材料留在外部 Vault。
type MachineIdentityAuthority interface {
	// CreateEnrollment 创建待激活的身份世代；返回结果不得包含任何秘密材料。
	CreateEnrollment(ctx context.Context, request MachineEnrollmentRequest) (MachineEnrollment, error)
	// ActivateGeneration 激活已完成外部材料投递且与证书绑定的身份世代。
	ActivateGeneration(ctx context.Context, activation MachineIdentityActivation) error
	// RotateGeneration 在旧世代停用后创建新世代，并返回新的待激活登记结果。
	RotateGeneration(ctx context.Context, request MachineIdentityRotationRequest) (MachineEnrollment, error)
	// RevokeGeneration 撤销指定身份世代；重复调用必须保持幂等。
	RevokeGeneration(ctx context.Context, revocation MachineIdentityRevocation) error
	// ReadTrustBundle 返回不透明信任束引用，不得将 PEM 或密钥材料带入模块 API。
	ReadTrustBundle(ctx context.Context, request TrustBundleRequest) (TrustBundleReference, error)
}

// MachineEnrollmentRequest 描述不携带秘密的 Agent 绑定请求。
type MachineEnrollmentRequest struct {
	TargetID          int64
	AgentID           string
	ProviderID        string
	BuilderScope      string
	CapabilityProfile string
	CapabilityVersion string
	ImageDigest       string
	AgentVersion      string
}

// MachineEnrollment 仅包含已签发身份世代的非秘密信息。
type MachineEnrollment struct {
	IdentityID           string
	TargetID             int64
	AgentID              string
	ProviderID           string
	BuilderScope         string
	CapabilityProfile    string
	CapabilityVersion    string
	Generation           int64
	EnrollmentRef        string
	ExpiresAt            time.Time
	TrustBundleVersion   string
	CertificateSerial    string
	PublicKeyFingerprint string
	Status               RuntimeTargetAgentStatus
}

// RuntimeTargetAgentStatus 表示 Runtime Target Agent 身份世代的生命周期状态。
type RuntimeTargetAgentStatus string

const (
	// RuntimeTargetAgentStatusPending 表示登记已创建但尚未完成证书绑定激活。
	RuntimeTargetAgentStatusPending RuntimeTargetAgentStatus = "pending"
	// RuntimeTargetAgentStatusActive 表示当前身份世代已完成激活并可用于受控访问。
	RuntimeTargetAgentStatusActive RuntimeTargetAgentStatus = "active"
	// RuntimeTargetAgentStatusRevoked 表示身份世代已被撤销且不得继续使用。
	RuntimeTargetAgentStatusRevoked RuntimeTargetAgentStatus = "revoked"
	// RuntimeTargetAgentStatusRetired 表示身份世代已被更新世代替代且不再接受使用。
	RuntimeTargetAgentStatusRetired RuntimeTargetAgentStatus = "retired"
)

// MachineIdentityActivation 确认外部投递材料后的指定身份世代激活。
type MachineIdentityActivation struct {
	IdentityID           string
	TargetID             int64
	AgentID              string
	Generation           int64
	CertificateSerial    string
	PublicKeyFingerprint string
}

// MachineIdentityRotationRequest 请求在旧世代停用后创建新世代。
type MachineIdentityRotationRequest struct {
	IdentityID        string
	TargetID          int64
	AgentID           string
	ProviderID        string
	BuilderScope      string
	CapabilityProfile string
	CapabilityVersion string
	Reason            string
}

// MachineIdentityRevocation 撤销一个身份世代；重复撤销必须幂等。
type MachineIdentityRevocation struct {
	IdentityID string
	TargetID   int64
	AgentID    string
	Generation int64
	Reason     string
}

// TrustBundleRequest 选择 Agent 作用域对应的信任束。
type TrustBundleRequest struct {
	TargetID   int64
	ProviderID string
	Generation int64
}

// TrustBundleReference 是不透明的信任束引用，不包含 PEM 或密钥。
type TrustBundleReference struct {
	Reference string
	Version   string
	ExpiresAt time.Time
}

// RuntimeTargetAgentBindingReader 仅向调用方提供 Agent 绑定状态。
type RuntimeTargetAgentBindingReader interface {
	// ReadAgentBinding 读取指定 Runtime Target 与 Agent 的当前绑定快照。
	ReadAgentBinding(ctx context.Context, targetID int64, agentID string) (RuntimeTargetAgentBinding, error)
}

// RuntimeTargetAgentBinding 描述 Agent 与单个 Runtime Target 及世代的绑定。
type RuntimeTargetAgentBinding struct {
	IdentityID           string
	TargetID             int64
	AgentID              string
	ProviderID           string
	BuilderScope         string
	CapabilityProfile    string
	CapabilityVersion    string
	Generation           int64
	CertificateSerial    string
	PublicKeyFingerprint string
	TrustBundleVersion   string
	ExpiresAt            time.Time
	RevokedAt            *time.Time
	Status               RuntimeTargetAgentStatus
}

// RuntimeTargetAgentLedgerReader 向 Agent 提供受控的 Driver-controller ledger 快照及回执入口。
type RuntimeTargetAgentLedgerReader interface {
	// IssueLedgerSnapshot 为已验证身份签发一次性 canonical ledger 快照。
	IssueLedgerSnapshot(ctx context.Context, identity AgentIdentity) (RuntimeTargetLedgerSnapshot, error)
	// SubmitTelemetryReport 接收与已签发快照绑定的受限 Agent 遥测回执。
	SubmitTelemetryReport(ctx context.Context, report RuntimeTargetTelemetryReport) error
}

// AgentIdentity 表示从已验证 mTLS URI SAN 提取的身份。
type AgentIdentity struct {
	IdentityID string
	TargetID   int64
	AgentID    string
	Generation int64
}

// RuntimeTargetLedgerSnapshot 是一次性的 canonical ledger 快照；Agent 不能修改其中的值。
type RuntimeTargetLedgerSnapshot struct {
	IdentityID        string
	TargetID          int64
	AgentID           string
	Generation        int64
	Sequence          int64
	SnapshotID        string
	SnapshotDigest    string
	BuilderScope      string
	ProviderID        string
	CapabilityProfile string
	CapabilityVersion string
	AffinityKey       string
	Available         bool
	Running           int
	Queued            int
	AllocatableSlots  int
	ObservedAt        time.Time
	ExpiresAt         time.Time
	IssuedAt          time.Time
}

// RuntimeTargetTelemetryReport 回执已签发快照，并仅包含受限 Agent 诊断信息。
type RuntimeTargetTelemetryReport struct {
	IdentityID            string
	TargetID              int64
	AgentID               string
	Generation            int64
	SnapshotID            string
	SnapshotDigest        string
	ObservedAt            time.Time
	ExpiresAt             time.Time
	Available             bool
	ImplementationVersion string
	Diagnostic            string
}
