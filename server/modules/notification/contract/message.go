package contract

// MenuMessageKey 标识通知模块稳定的菜单标题消息 key。
type MenuMessageKey string

// String 返回规范化的菜单标题消息 key。
func (k MenuMessageKey) String() string {
	return string(k)
}

const (
	// NotificationMenuTitle 标识通知中心菜单的本地化标题。
	NotificationMenuTitle MenuMessageKey = "menu.notification.title"
)
