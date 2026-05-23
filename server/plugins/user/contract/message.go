package contract

// MenuMessageKey identifies a stable user-plugin menu title message key.
type MenuMessageKey string

// String returns the canonical menu message key value.
func (k MenuMessageKey) String() string {
	return string(k)
}

const (
	// UserListMenuTitle identifies the localized title for the user list menu.
	UserListMenuTitle MenuMessageKey = "menu.access_control.users.title"
)
