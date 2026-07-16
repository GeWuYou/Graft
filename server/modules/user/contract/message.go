package contract

// MenuMessageKey 标识 user 模块稳定的菜单标题消息键。
type MenuMessageKey string

// String 返回 canonical 菜单消息键值。
func (k MenuMessageKey) String() string {
	return string(k)
}

const (
	// UserListMenuTitle 标识用户列表菜单的本地化标题键。
	UserListMenuTitle MenuMessageKey = "menu.security.users.title"
)
