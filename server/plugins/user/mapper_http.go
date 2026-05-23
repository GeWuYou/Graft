package user

import (
	"strings"
	"time"

	usercontract "graft/server/plugins/user/contract"
	userstore "graft/server/plugins/user/store"
)

func normalizeUserStatus(status string) string {
	switch strings.TrimSpace(status) {
	case usercontract.UserStatusDisabled:
		return usercontract.UserStatusDisabled
	default:
		return usercontract.UserStatusEnabled
	}
}

func toUserListItem(user userstore.User) userListItem {
	return userListItem{
		ID:        user.ID,
		Username:  user.Username,
		Display:   user.Display,
		Status:    normalizeUserStatus(user.Status),
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
