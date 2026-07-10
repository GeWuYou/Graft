package user

import (
	"context"
	"errors"
	"strings"

	"graft/server/internal/moduleapi"
	usercontract "graft/server/modules/user/contract"
	userstore "graft/server/modules/user/store"
)

// userIdentityProvider publishes user-profile facts to auth without exposing
// credentials, password state, sessions, or the user repository itself.
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

// currentUserFromStore converts a stored user into a current user profile containing identity and display information.
func currentUserFromStore(user userstore.User) moduleapi.CurrentUser {
	return moduleapi.CurrentUser{ID: user.ID, Username: user.Username, DisplayName: user.Display}
}

var _ moduleapi.UserIdentityProvider = userIdentityProvider{}
