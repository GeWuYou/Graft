package user

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/moduleapi"
	userstore "graft/server/modules/user/store"
)

type createUserRepository struct {
	deleteErr error
	deleteIn  []userstore.DeleteUserInput
}

func (*createUserRepository) GetByID(context.Context, uint64) (userstore.User, error) {
	return userstore.User{}, nil
}
func (*createUserRepository) GetByUsername(context.Context, string) (userstore.User, error) {
	return userstore.User{}, nil
}
func (*createUserRepository) List(context.Context) ([]userstore.User, error) { return nil, nil }
func (*createUserRepository) Count(context.Context) (int, error)             { return 0, nil }
func (*createUserRepository) Update(context.Context, userstore.UpdateUserInput) (userstore.User, error) {
	return userstore.User{}, nil
}
func (*createUserRepository) SetStatus(context.Context, userstore.SetUserStatusInput) (userstore.User, error) {
	return userstore.User{}, nil
}
func (*createUserRepository) Create(context.Context, userstore.CreateUserInput) (userstore.User, error) {
	return userstore.User{ID: 19, Username: "new-user", Display: "New User"}, nil
}
func (r *createUserRepository) Delete(_ context.Context, input userstore.DeleteUserInput) error {
	r.deleteIn = append(r.deleteIn, input)
	return r.deleteErr
}

type failingCredentialManager struct{ provisionErr error }

func (m failingCredentialManager) ProvisionPasswordCredential(context.Context, uint64, string, bool) error {
	return m.provisionErr
}
func (failingCredentialManager) ResetPassword(context.Context, uint64, string) error { return nil }
func (failingCredentialManager) RevokeSessions(context.Context, uint64) error        { return nil }

var _ moduleapi.AuthCredentialManagementService = failingCredentialManager{}

func TestCreateUserRetainsProvisionAndRollbackFailures(t *testing.T) {
	provisionErr := errors.New("credential provisioning failed")
	rollbackErr := errors.New("profile rollback failed")
	repo := &createUserRepository{deleteErr: rollbackErr}
	service := userService{users: repo, credentials: failingCredentialManager{provisionErr: provisionErr}}

	_, err := service.CreateUser(context.Background(), CreateUserCommand{
		Username: "new-user",
		Display:  "New User",
		Password: "Password1234",
		ActorID:  5,
	})
	if !errors.Is(err, provisionErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("CreateUser() error = %v, want joined provisioning and rollback errors", err)
	}
	if len(repo.deleteIn) != 1 || repo.deleteIn[0].ID != 19 || repo.deleteIn[0].ActorID != 5 {
		t.Fatalf("Delete() inputs = %#v", repo.deleteIn)
	}
}
