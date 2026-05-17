package contract

// MenuMessageKey identifies a stable rbac-plugin menu title message key.
type MenuMessageKey string

// String returns the canonical menu message key value.
func (k MenuMessageKey) String() string {
	return string(k)
}

const (
	// RoleListMenuTitle identifies the localized title for the role list menu.
	RoleListMenuTitle MenuMessageKey = "menu.role_list.title"
)
