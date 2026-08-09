// Package contract 定义 runtime-target 稳定的导航、权限和事件标识。
package contract

import "graft/server/internal/event"

// 稳定的模块契约标识。
const (
	// ModuleID 是 runtime-target 模块的稳定编译期标识。
	ModuleID                   = "runtime-target"
	MenuTitle                  = "menu.runtimeTargets.title"
	MenuPath                   = "/infrastructure/runtime-targets"
	ViewPermission             = "runtime_target.view"
	ManagePermission           = "runtime_target.manage"
	AssignmentManagePermission = "runtime_target.assignment.manage"
	RefreshPermission          = "runtime_target.refresh"
	// SummaryTopic 是目标级资源快照使用的实时主题。
	SummaryTopic          = "runtime-target.summary.list"
	AssignmentsRoute      = "/runtime-targets/:id/assignments"
	AssignmentDeleteRoute = "/runtime-targets/:id/assignments/:userId/delete"
)

// AgentCertificateRevocationEventType 是撤销已签发 Agent 证书的 durable event 类型。
const AgentCertificateRevocationEventType event.Type = "runtime-target.agent-certificate-revocation.v1"

// AgentCertificateRevocationEvent 是 Runtime Target 写入 Outbox 的非秘密撤销事实。
type AgentCertificateRevocationEvent struct {
	IdentityID        string `json:"identity_id"`
	TargetID          int64  `json:"target_id"`
	AgentID           string `json:"agent_id"`
	Generation        int64  `json:"generation"`
	CertificateSerial string `json:"certificate_serial"`
	Reason            string `json:"reason"`
}
