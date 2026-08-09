package moduleapi

import (
	"context"
	"time"
)

// AgentDeliveryAuthority 由 Runtime Target 实现，负责一次性引导材料交付的业务授权与证据记录。
//
// 调用方必须在既有认证边界中构造 DeliveryActor；该接口不解析凭据、不签发证书，也不激活 Agent 世代。
type AgentDeliveryAuthority interface {
	// CreateDeliveryGrant 为已有待激活世代创建仅可交付一次的引导材料授权。
	CreateDeliveryGrant(context.Context, AgentDeliveryGrantRequest) (AgentDeliveryGrant, error)
	// HandoffDeliveryGrant 向被冻结的已认证部署主体释放一次性引导材料。
	HandoffDeliveryGrant(context.Context, DeliveryActor, AgentDeliveryHandoffRequest) (AgentDeliveryHandoffMaterial, error)
	// RecordDeliveryReceipt 记录部署主体提交的 Docker 投递证据；证据不构成 Agent 激活授权。
	RecordDeliveryReceipt(context.Context, DeliveryActor, AgentDeliveryReceiptRequest) (AgentDeliveryReceipt, error)
}

// AgentDeliveryGrantRequest 描述与既有待激活世代绑定的 Docker 投递授权条件。
// ExpectedAutomationID 是创建时冻结的已有认证主体，而不是来自后续 handoff 或 receipt 请求的声明。
type AgentDeliveryGrantRequest struct {
	TargetID              int64
	AgentID               string
	Generation            int64
	ExpectedAutomationID  string
	DockerInstallationRef string
	ExpiresAt             time.Time
}

// AgentDeliveryGrant 是不包含引导材料的投递授权投影。
type AgentDeliveryGrant struct {
	GrantID               string
	TargetID              int64
	AgentID               string
	Generation            int64
	ExpectedAutomationID  string
	DockerInstallationRef string
	ExpiresAt             time.Time
}

// AgentDeliveryHandoffRequest 标识一个不可重放的既有认证主体交接。
type AgentDeliveryHandoffRequest struct {
	GrantID string
}

// AgentDeliveryHandoffMaterial 是仅可由内部交付调用方接收的一次性引导材料。
// BootstrapToken 不得被记录、持久化、回显到 Operator HTTP 或放入其它 moduleapi DTO。
type AgentDeliveryHandoffMaterial struct {
	GrantID        string
	HandoffID      string
	BootstrapToken string
	ExpiresAt      time.Time
}

// AgentDeliveryReceiptRequest 是不携带 automation identity 或秘密值的 Docker 投递证据。
// Automation identity 始终由 DeliveryActor 注入，DockerSecretRef 只能是不透明的非秘密引用。
type AgentDeliveryReceiptRequest struct {
	GrantID               string
	ReceiptID             string
	ProtocolVersion       string
	HandoffID             string
	AssertedDeliveredAt   time.Time
	DockerInstallationRef string
	DockerSecretRef       string
	PayloadFingerprint    string
}

// AgentDeliveryReceipt 是 Runtime Target 接受的投递证据投影。
type AgentDeliveryReceipt struct {
	GrantID               string
	ReceiptID             string
	ProtocolVersion       string
	AutomationID          string
	HandoffID             string
	AssertedDeliveredAt   time.Time
	AcceptedAt            time.Time
	DockerInstallationRef string
	DockerSecretRef       string
	PayloadFingerprint    string
	Replay                bool
}
