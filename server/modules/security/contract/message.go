package contract

// MessageKey 标识安全模块稳定的消息 key。
type MessageKey string

// String 返回规范化的消息 key。
func (k MessageKey) String() string { return string(k) }

const (
	// OverviewMenuTitle 标识安全概览菜单标题。
	OverviewMenuTitle MessageKey = "menu.security.overview.title"
	// OverviewReadDisplay 标识安全概览读取权限的展示名称。
	OverviewReadDisplay MessageKey = "security.permission.overview.read.display"
	// OverviewReadDescription 标识安全概览读取权限的说明。
	OverviewReadDescription MessageKey = "security.permission.overview.read.description"
)
