package contract

// PermissionCode 标识 platform-network 模块稳定的权限码。
type PermissionCode string

// String 返回权限码的 wire-format 表示。
func (c PermissionCode) String() string { return string(c) }

const (
	// NetworkReadPermission 允许读取平台出站网络策略和已净化诊断结果。
	NetworkReadPermission PermissionCode = "platform-network.read"
	// NetworkWritePermission 允许更新平台出站网络策略。
	NetworkWritePermission PermissionCode = "platform-network.write"
	// NetworkDiagnosePermission 允许执行注册 target 的出站网络诊断。
	NetworkDiagnosePermission PermissionCode = "platform-network.diagnose"
	// NetworkManageTargetsPermission 允许管理受 SSRF 策略保护的自定义 target。
	NetworkManageTargetsPermission PermissionCode = "platform-network.targets.manage"
	// NetworkExitIPReadPermission 保留给未来仅实时、可审计的完整出口 IP 披露；历史报告永远不会返回完整 IP。
	NetworkExitIPReadPermission PermissionCode = "platform-network.exit-ip.read"
)
