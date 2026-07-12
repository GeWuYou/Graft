package user

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/moduleapi"
	userstore "graft/server/modules/user/store"
)

type identityProviderRepository struct {
	getByID       func(context.Context, uint64) (userstore.User, error)
	getByUsername func(context.Context, string) (userstore.User, error)
	create        func(context.Context, userstore.CreateUserInput) (userstore.User, error)
	listCalls     int
}

func (r *identityProviderRepository) GetByID(ctx context.Context, id uint64) (userstore.User, error) {
	return r.getByID(ctx, id)
}

func (r *identityProviderRepository) GetByUsername(ctx context.Context, username string) (userstore.User, error) {
	return r.getByUsername(ctx, username)
}

func (r *identityProviderRepository) List(context.Context) ([]userstore.User, error) {
	r.listCalls++
	return nil, errors.New("List must not be used for username lookup")
}

func (*identityProviderRepository) ListSecuritySummaries(context.Context, uint64, int) ([]userstore.User, error) {
	return nil, nil
}

func (r *identityProviderRepository) Count(context.Context) (int, error) { return 0, nil }

func (r *identityProviderRepository) Create(ctx context.Context, input userstore.CreateUserInput) (userstore.User, error) {
	return r.create(ctx, input)
}

func (*identityProviderRepository) Update(context.Context, userstore.UpdateUserInput) (userstore.User, error) {
	return userstore.User{}, nil
}

func (*identityProviderRepository) SetStatus(context.Context, userstore.SetUserStatusInput) (userstore.User, error) {
	return userstore.User{}, nil
}

func (*identityProviderRepository) Delete(context.Context, userstore.DeleteUserInput) error {
	return nil
}

func TestLookupUserByUsernameUsesDirectRepositoryLookup(t *testing.T) {
	repo := &identityProviderRepository{
		getByID: func(context.Context, uint64) (userstore.User, error) { return userstore.User{}, nil },
		getByUsername: func(_ context.Context, username string) (userstore.User, error) {
			if username != "admin" {
				t.Fatalf("username = %q, want admin", username)
			}
			return userstore.User{ID: 11, Username: username, Display: "Administrator"}, nil
		},
		create: func(context.Context, userstore.CreateUserInput) (userstore.User, error) { return userstore.User{}, nil },
	}

	current, err := (userIdentityProvider{users: repo}).LookupUserByUsername(context.Background(), " admin ")
	if err != nil {
		t.Fatalf("LookupUserByUsername() error = %v", err)
	}
	if current.ID != 11 || current.Username != "admin" {
		t.Fatalf("LookupUserByUsername() = %#v", current)
	}
	if repo.listCalls != 0 {
		t.Fatalf("List() calls = %d, want 0", repo.listCalls)
	}
}

func TestEnsureDefaultAdminProfileReturnsConcurrentCreateWinner(t *testing.T) {
	lookupCalls := 0
	repo := &identityProviderRepository{
		getByID: func(context.Context, uint64) (userstore.User, error) { return userstore.User{}, nil },
		getByUsername: func(_ context.Context, username string) (userstore.User, error) {
			lookupCalls++
			if username != defaultAdminUsername {
				t.Fatalf("username = %q, want %q", username, defaultAdminUsername)
			}
			if lookupCalls == 1 {
				return userstore.User{}, userstore.ErrUserNotFound
			}
			return userstore.User{ID: 7, Username: defaultAdminUsername, Display: defaultAdminDisplay}, nil
		},
		create: func(_ context.Context, input userstore.CreateUserInput) (userstore.User, error) {
			if input.Username != defaultAdminUsername {
				t.Fatalf("create username = %q", input.Username)
			}
			return userstore.User{}, userstore.ErrUsernameConflict
		},
	}

	current, err := (userIdentityProvider{users: repo}).EnsureDefaultAdminProfile(context.Background())
	if err != nil {
		t.Fatalf("EnsureDefaultAdminProfile() error = %v", err)
	}
	if current.ID != 7 || lookupCalls != 2 {
		t.Fatalf("EnsureDefaultAdminProfile() = %#v, lookup calls = %d", current, lookupCalls)
	}
}

func TestGetCurrentUserByIDTranslatesMissingUserToModuleAPIError(t *testing.T) {
	repo := &identityProviderRepository{
		getByID: func(context.Context, uint64) (userstore.User, error) {
			return userstore.User{}, userstore.ErrUserNotFound
		},
		getByUsername: func(context.Context, string) (userstore.User, error) { return userstore.User{}, nil },
		create:        func(context.Context, userstore.CreateUserInput) (userstore.User, error) { return userstore.User{}, nil },
	}

	_, err := (userIdentityProvider{users: repo}).GetCurrentUserByID(context.Background(), 123)
	if !errors.Is(err, moduleapi.ErrUserNotFound) {
		t.Fatalf("GetCurrentUserByID() error = %v, want moduleapi.ErrUserNotFound", err)
	}
}
