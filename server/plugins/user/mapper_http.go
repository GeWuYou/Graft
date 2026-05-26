package user

import (
	"fmt"
	"math"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	useropenapi "graft/server/internal/contract/openapi/user"
	usercontract "graft/server/plugins/user/contract"
	userstore "graft/server/plugins/user/store"
)

type userListResponse = generated.UserListResponse
type userListItem = generated.UserListItem

func normalizeUserStatus(status string) string {
	switch strings.TrimSpace(status) {
	case usercontract.UserStatusDisabled:
		return usercontract.UserStatusDisabled
	default:
		return usercontract.UserStatusEnabled
	}
}

func toCreateUserCommand(request useropenapi.PostUsersJSONRequestBody, actorID uint64) CreateUserCommand {
	return CreateUserCommand{
		Username: request.Username,
		Display:  request.Display,
		Password: request.Password,
		ActorID:  actorID,
	}
}

func toUpdateUserCommand(request useropenapi.PostUserUpdateJSONRequestBody, userID uint64, actorID uint64) UpdateUserCommand {
	return UpdateUserCommand{
		ID:       userID,
		Username: request.Username,
		Display:  request.Display,
		ActorID:  actorID,
	}
}

func toUpdateUserStatusCommand(request useropenapi.PostUserStatusJSONRequestBody, userID uint64, actorID uint64) (UpdateUserStatusCommand, bool) {
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

func toCanonicalManagedUserStatus(status useropenapi.PostUserStatusJSONBodyStatus) (string, bool) {
	switch status {
	case useropenapi.Enabled:
		return usercontract.UserStatusEnabled, true
	case useropenapi.Disabled:
		return usercontract.UserStatusDisabled, true
	default:
		return "", false
	}
}

func toUserListResponse(users []userstore.User) (userListResponse, error) {
	items := make([]userListItem, 0, len(users))
	for _, user := range users {
		item, err := toUserListItem(user)
		if err != nil {
			return userListResponse{}, err
		}
		items = append(items, item)
	}

	return userListResponse{Items: items}, nil
}

func toUserListItem(user userstore.User) (userListItem, error) {
	id, err := mustConvertGeneratedUserID(user.ID)
	if err != nil {
		return userListItem{}, err
	}

	return userListItem{
		Id:        id,
		Username:  user.Username,
		Display:   user.Display,
		Status:    normalizeUserStatus(user.Status),
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func toGeneratedSessionSummaries(items []sessionSummary) []generated.SessionSummary {
	summaries := make([]generated.SessionSummary, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, generated.SessionSummary{
			SessionId: item.SessionID,
			CreatedAt: item.CreatedAt,
			ExpiresAt: item.ExpiresAt,
			Current:   item.Current,
		})
	}

	return summaries
}

func mustConvertGeneratedUserID(id uint64) (int64, error) {
	if id > math.MaxInt64 {
		return 0, fmt.Errorf("user generated response user id exceeds int64: %d", id)
	}
	return int64(id), nil
}
