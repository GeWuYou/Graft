package user

import (
	"context"
	"errors"
	"strings"

	"graft/server/internal/moduleapi"
	usercontract "graft/server/modules/user/contract"
	userstore "graft/server/modules/user/store"
)

// userIdentityProvider 向 auth 提供用户身份与展示信息，不暴露凭据、密码状态、会话或用户仓储本身。
type userIdentityProvider struct {
	users userstore.UserRepository
}

func (p userIdentityProvider) LookupUserByUsername(ctx context.Context, username string) (moduleapi.CurrentUser, error) {
	if p.users == nil {
		return moduleapi.CurrentUser{}, errors.New("user repository is unavailable")
	}
	username = strings.TrimSpace(username)
	user, err := p.users.GetByUsername(ctx, username)
	if err != nil {
		return moduleapi.CurrentUser{}, err
	}
	return currentUserFromStore(user), nil
}

func (p userIdentityProvider) GetCurrentUserByID(ctx context.Context, userID uint64) (moduleapi.CurrentUser, error) {
	if p.users == nil {
		return moduleapi.CurrentUser{}, errors.New("user repository is unavailable")
	}
	user, err := p.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			return moduleapi.CurrentUser{}, moduleapi.ErrUserNotFound
		}
		return moduleapi.CurrentUser{}, err
	}
	return currentUserFromStore(user), nil
}

func (p userIdentityProvider) EnsureDefaultAdminProfile(ctx context.Context) (moduleapi.CurrentUser, error) {
	current, err := p.LookupUserByUsername(ctx, defaultAdminUsername)
	if err == nil {
		return current, nil
	}
	if !errors.Is(err, userstore.ErrUserNotFound) {
		return moduleapi.CurrentUser{}, err
	}
	created, err := p.users.Create(ctx, userstore.CreateUserInput{
		Username: defaultAdminUsername,
		Display:  defaultAdminDisplay,
		Status:   usercontract.UserStatusEnabled,
	})
	if err != nil {
		if errors.Is(err, userstore.ErrUsernameConflict) {
			return p.LookupUserByUsername(ctx, defaultAdminUsername)
		}
		return moduleapi.CurrentUser{}, err
	}
	return currentUserFromStore(created), nil
}

// currentUserFromStore 将仓储用户转换为仅包含身份和展示信息的当前用户资料。
func currentUserFromStore(user userstore.User) moduleapi.CurrentUser {
	return moduleapi.CurrentUser{ID: user.ID, Username: user.Username, DisplayName: user.Display}
}

var _ moduleapi.UserIdentityProvider = userIdentityProvider{}
