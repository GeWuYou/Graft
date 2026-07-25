package user

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"graft/server/internal/moduleapi"
	userstore "graft/server/modules/user/store"
)

type createUserRepository struct{}

func (*createUserRepository) GetByID(context.Context, uint64) (userstore.User, error) {
	return userstore.User{}, nil
}
func (*createUserRepository) GetByUsername(context.Context, string) (userstore.User, error) {
	return userstore.User{}, nil
}
func (*createUserRepository) List(context.Context) ([]userstore.User, error) { return nil, nil }
func (*createUserRepository) ListSecuritySummaries(context.Context, uint64, int) ([]userstore.User, error) {
	return nil, nil
}
func (*createUserRepository) Count(context.Context) (int, error) { return 0, nil }
func (*createUserRepository) Update(context.Context, userstore.UpdateUserInput) (userstore.User, error) {
	return userstore.User{}, nil
}
func (*createUserRepository) SetStatus(context.Context, userstore.SetUserStatusInput) (userstore.User, error) {
	return userstore.User{}, nil
}
func (*createUserRepository) Create(context.Context, userstore.CreateUserInput) (userstore.User, error) {
	return userstore.User{ID: 19, Username: "new-user", Display: "New User"}, nil
}
func (*createUserRepository) Delete(context.Context, userstore.DeleteUserInput) error {
	return nil
}

func (r *createUserRepository) RunInTransaction(ctx context.Context, callback func(context.Context, userstore.UserRepository) error) error {
	return callback(ctx, r)
}
func (r *createUserRepository) RunInCompositeTransaction(ctx context.Context, callback func(context.Context, userstore.UserRepository, *sql.Tx) error) error {
	return callback(ctx, r, nil)
}

type failingAuthTransactionFactory struct{ err error }

func (f failingAuthTransactionFactory) BindAuthTransaction(*sql.Tx) (moduleapi.AuthTransactionAdapter, error) {
	return failingAuthTransactionAdapter(f), nil
}

type failingAuthTransactionAdapter struct{ err error }

func (a failingAuthTransactionAdapter) ProvisionPasswordCredential(context.Context, moduleapi.AuthCredentialProvisionInput) error {
	return a.err
}
func (failingAuthTransactionAdapter) RevokeSessions(context.Context, uint64) error { return nil }

func TestCreateUserRollsBackWhenCredentialProvisionFails(t *testing.T) {
	provisionErr := errors.New("credential provisioning failed")
	repo := &createUserRepository{}
	service := userService{users: repo, transactions: repo, composites: repo, authTx: failingAuthTransactionFactory{err: provisionErr}}

	created, err := service.CreateUser(context.Background(), CreateUserCommand{
		Username: "new-user",
		Display:  "New User",
		Password: "Password1234",
		ActorID:  5,
	})
	if !errors.Is(err, provisionErr) {
		t.Fatalf("CreateUser() error = %v, want provisioning failure", err)
	}
	if created.ID != 0 {
		t.Fatalf("CreateUser() user = %#v, want zero result", created)
	}
}
