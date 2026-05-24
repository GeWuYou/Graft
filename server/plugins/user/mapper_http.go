package user

import (
	"strings"
	"time"

	openapicontract "graft/server/internal/contract/openapi"
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

func toCreateUserCommand(request openapicontract.PostUsersJSONRequestBody, actorID uint64) CreateUserCommand {
	return CreateUserCommand{
		Username: request.Username,
		Display:  request.Display,
		Password: request.Password,
		ActorID:  actorID,
	}
}

func toUpdateUserCommand(request openapicontract.PostUserUpdateJSONRequestBody, userID uint64, actorID uint64) UpdateUserCommand {
	return UpdateUserCommand{
		ID:       userID,
		Username: request.Username,
		Display:  request.Display,
		ActorID:  actorID,
	}
}

func toUpdateUserStatusCommand(request openapicontract.PostUserStatusJSONRequestBody, userID uint64, actorID uint64) (UpdateUserStatusCommand, bool) {
	status, ok := toCanonicalManagedUserStatus(request.Status)
	if !ok {
		return UpdateUserStatusCommand{}, false
	}

	return UpdateUserStatusCommand{
		ID:      userID,
		Status:  status,
		ActorID: actorID,
	}, true
}

func toCanonicalManagedUserStatus(status openapicontract.PostUserStatusJSONBodyStatus) (string, bool) {
	switch status {
	case openapicontract.PostUserStatusJSONBodyStatusEnabled:
		return usercontract.UserStatusEnabled, true
	case openapicontract.PostUserStatusJSONBodyStatusDisabled:
		return usercontract.UserStatusDisabled, true
	default:
		return "", false
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
