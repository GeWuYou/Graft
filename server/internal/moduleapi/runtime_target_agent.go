package moduleapi

import (
	"context"
	"time"
)

// AgentEnrollmentAuthority 由 Runtime Target 实现，负责 Agent 与目标的业务绑定及世代生命周期。
// 该接口不签发证书，也不接触私钥、投递令牌或其他秘密材料；这些职责分别属于 Credential Vault 和部署交付边界。
type AgentEnrollmentAuthority interface {
	// CreateEnrollment 创建待激活的身份世代；返回结果不得包含任何秘密材料。
	CreateEnrollment(ctx context.Context, request AgentEnrollmentRequest) (AgentEnrollment, error)
	// ActivateGeneration 激活已完成外部材料投递且与证书绑定的身份世代。
	ActivateGeneration(ctx context.Context, activation AgentEnrollmentActivation) error
	// RotateGeneration 原子停用当前活动旧世代并创建新待激活世代；无需等待旧证书到期。
	RotateGeneration(ctx context.Context, request AgentEnrollmentRotationRequest) (AgentEnrollment, error)
	// RevokeGeneration 撤销指定身份世代；重复调用必须保持幂等。
	RevokeGeneration(ctx context.Context, revocation AgentEnrollmentRevocation) error
}

// AgentEnrollmentRequest 描述不携带秘密的 Agent 绑定请求。
type AgentEnrollmentRequest struct {
	TargetID          int64
	AgentID           string
	ProviderID        string
	BuilderScope      string
	CapabilityProfile string
	CapabilityVersion string
	ImageDigest       string
	AgentVersion      string
	EnrollmentRef     string
	TrustBundle       TrustBundleReference
	ExpiresAt         time.Time
}

// AgentEnrollment 仅包含已登记身份世代的非秘密信息。
type AgentEnrollment struct {
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

// AgentEnrollmentActivation 确认外部投递材料与证书绑定后的指定身份世代激活。
type AgentEnrollmentActivation struct {
	IdentityID           string
	TargetID             int64
	AgentID              string
	Generation           int64
	CertificateIssuer    string
	CertificateSerial    string
	PublicKeyFingerprint string
}

// AgentEnrollmentRotationRequest 请求原子停用当前活动旧世代并创建新待激活世代。
type AgentEnrollmentRotationRequest struct {
	IdentityID        string
	TargetID          int64
	AgentID           string
	ProviderID        string
	BuilderScope      string
	CapabilityProfile string
	CapabilityVersion string
	EnrollmentRef     string
	TrustBundle       TrustBundleReference
	ExpiresAt         time.Time
	Reason            string
}

// AgentEnrollmentRevocation 撤销一个身份世代；重复撤销必须幂等。
type AgentEnrollmentRevocation struct {
	IdentityID string
	TargetID   int64
	AgentID    string
	Generation int64
	Reason     string
}

// AgentCertificateIssuer 由 Credential Vault 实现，只负责经 Runtime Target 授权后的 PKI 操作。
// 调用方必须先完成登记、投递与 CSR 绑定校验；Vault 不得借由证书签发建立或激活业务归属。
type AgentCertificateIssuer interface {
	// IssueCSR 为已验证的 CSR 签发证书；相同 IssuanceKey 必须支持外部副作用的查询或幂等协调。
	IssueCSR(ctx context.Context, request AgentCertificateIssuanceRequest) (IssuedAgentCertificate, error)
	// ReadTrustBundle 返回不透明信任束引用，不得将 PEM 或私钥材料带入模块 API。
	ReadTrustBundle(ctx context.Context, request TrustBundleRequest) (TrustBundleReference, error)
	// RevokeCertificate 撤销已签发证书；重复调用必须保持幂等。
	RevokeCertificate(ctx context.Context, revocation AgentCertificateRevocation) error
}

// AgentCertificateIssuanceRequest 描述已获授权的 CSR 签发请求。
// CSRDER 是公开密钥证明而非私钥；IssuanceKey 用于协调 Vault 外部副作用的重试与恢复。
type AgentCertificateIssuanceRequest struct {
	IdentityID   string
	TargetID     int64
	AgentID      string
	Generation   int64
	IssuanceKey  string
	SPIFFEURI    string
	CSRDER       []byte
}

// IssuedAgentCertificate 是 Vault 返回的非私钥签发结果。
// CertificateChainDER 只能通过受控 bootstrap TLS 返回给对应 Agent，不得持久化为模块业务状态。
type IssuedAgentCertificate struct {
	IssuanceKey          string
	CertificateSerial    string
	CertificateChainDER  [][]byte
	PublicKeyFingerprint string
	ExpiresAt            time.Time
	TrustBundle          TrustBundleReference
}

// AgentCertificateRevocation 描述一个已签发 Agent 证书的幂等撤销请求。
type AgentCertificateRevocation struct {
	IdentityID        string
	TargetID          int64
	AgentID           string
	Generation        int64
	CertificateSerial string
	Reason            string
	IdempotencyKey    string
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

// RuntimeTargetLedgerSnapshot 是仅供内部 ledger 与 agent-ledger-snapshot 映射使用的 canonical DTO。
// CapabilityVersion、AffinityKey、Available、IssuedAt、BuilderScope、ProviderID 属于内部字段，
// 不等同于 OpenAPI agent-telemetry-report-request；对外快照仍受 additionalProperties: false 约束。
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

// RuntimeTargetTelemetryReport 是内部 canonical 回执 DTO，不是 agent-telemetry-report-request 的逐字段镜像。
// ExpiresAt、Available、Diagnostic 仅服务于回执校验与受限诊断，不作为该 OpenAPI 请求字段的额外扩展。
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
