package contract

// MenuMessageKey 标识 user 模块稳定的菜单标题消息键。
type MenuMessageKey string

// String 返回 canonical 菜单消息键值。
func (k MenuMessageKey) String() string {
	return string(k)
}

const (
	// UserListMenuTitle identifies the localized title for the user list menu.
	UserListMenuTitle MenuMessageKey = "menu.security.users.title"
)
