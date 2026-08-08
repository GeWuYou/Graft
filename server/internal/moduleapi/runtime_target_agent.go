package moduleapi

import (
	"context"
	"time"
)

// MachineIdentityAuthority 负责 Agent 信任的签发与生命周期；秘密材料留在外部 Vault。
type MachineIdentityAuthority interface {
	CreateEnrollment(context.Context, MachineEnrollmentRequest) (MachineEnrollment, error)
	ActivateGeneration(context.Context, MachineIdentityActivation) error
	RotateGeneration(context.Context, MachineIdentityRotationRequest) (MachineEnrollment, error)
	RevokeGeneration(context.Context, MachineIdentityRevocation) error
	ReadTrustBundle(context.Context, TrustBundleRequest) (TrustBundleReference, error)
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
	Status               string
}

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
	ReadAgentBinding(context.Context, int64, string) (RuntimeTargetAgentBinding, error)
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
	Status               string
}

// RuntimeTargetAgentLedgerReader 向 Agent 提供受控的 Driver-controller ledger 快照及回执入口。
type RuntimeTargetAgentLedgerReader interface {
	IssueLedgerSnapshot(context.Context, AgentIdentity) (RuntimeTargetLedgerSnapshot, error)
	SubmitTelemetryReport(context.Context, RuntimeTargetTelemetryReport) error
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
