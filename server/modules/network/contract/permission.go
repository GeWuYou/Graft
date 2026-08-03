package contract

// PermissionCode 标识 platform-network 模块稳定的权限码。
type PermissionCode string

// String 返回权限码的 wire-format 表示。
func (c PermissionCode) String() string { return string(c) }

const (
	// NetworkReadPermission 允许读取平台出站网络策略和诊断结果。
	NetworkReadPermission PermissionCode = "platform-network.read"
	// NetworkWritePermission 允许更新平台出站网络策略。
	NetworkWritePermission PermissionCode = "platform-network.write"
	// NetworkDiagnosePermission 允许执行固定的出站网络诊断。
	NetworkDiagnosePermission PermissionCode = "platform-network.diagnose"
)
