package contract

// MessageKey 标识系统配置模块稳定的消息 key。
type MessageKey string

// String 返回规范化的消息 key。
func (k MessageKey) String() string {
	return string(k)
}

const (
	// SystemConfigMenuTitle 标识系统配置菜单标题。
	SystemConfigMenuTitle MessageKey = "menu.system_config.title"
	// SystemConfigNotFound 标识配置 key 不存在时的错误消息。
	SystemConfigNotFound MessageKey = "system_config.not_found"
	// SystemConfigInvalidRequest 标识配置请求无效时的错误消息。
	SystemConfigInvalidRequest MessageKey = "system_config.invalid_request"
)
