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
func (*createUserRepository) ListPage(context.Context, userstore.UserListFilter) ([]userstore.User, int, error) {
	return nil, 0, nil
}
func (*createUserRepository) ListCandidates(context.Context, userstore.UserCandidateQuery) ([]userstore.User, int, error) {
	return nil, 0, nil
}
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

type listPageRepository struct {
	filter userstore.UserListFilter
}

func (r *listPageRepository) GetByID(context.Context, uint64) (userstore.User, error) {
	return userstore.User{}, nil
}
func (r *listPageRepository) GetByUsername(context.Context, string) (userstore.User, error) {
	return userstore.User{}, nil
}
func (r *listPageRepository) List(context.Context) ([]userstore.User, error) { return nil, nil }
func (r *listPageRepository) ListPage(_ context.Context, filter userstore.UserListFilter) ([]userstore.User, int, error) {
	r.filter = filter
	return []userstore.User{{ID: 7}}, 1, nil
}
func (r *listPageRepository) ListCandidates(context.Context, userstore.UserCandidateQuery) ([]userstore.User, int, error) {
	return nil, 0, nil
}
func (r *listPageRepository) ListSecuritySummaries(context.Context, uint64, int) ([]userstore.User, error) {
	return nil, nil
}
func (r *listPageRepository) Count(context.Context) (int, error) { return 0, nil }
func (r *listPageRepository) Create(context.Context, userstore.CreateUserInput) (userstore.User, error) {
	return userstore.User{}, nil
}
func (r *listPageRepository) Update(context.Context, userstore.UpdateUserInput) (userstore.User, error) {
	return userstore.User{}, nil
}
func (r *listPageRepository) SetStatus(context.Context, userstore.SetUserStatusInput) (userstore.User, error) {
	return userstore.User{}, nil
}
func (r *listPageRepository) Delete(context.Context, userstore.DeleteUserInput) error { return nil }

type roleFilterAccessService struct{ userIDs []uint64 }

func (s roleFilterAccessService) ListRoleNamesByUserID(context.Context, uint64) ([]string, error) {
	return nil, nil
}
func (s roleFilterAccessService) ListPermissionCodesByUserID(context.Context, uint64) ([]string, error) {
	return nil, nil
}
func (s roleFilterAccessService) ListUserIDsByPermissionCode(context.Context, string) ([]uint64, error) {
	return nil, nil
}
func (s roleFilterAccessService) ListUserIDsByRoleID(context.Context, uint64) ([]uint64, error) {
	return s.userIDs, nil
}
func (s roleFilterAccessService) ListRoleSummariesByUserIDs(context.Context, []uint64) (map[uint64][]moduleapi.RoleSummary, error) {
	return nil, nil
}

func TestListUsersPageNormalizesBoundsAndUsesRBACRoleFilter(t *testing.T) {
	repository := &listPageRepository{}
	roleID := uint64(4)
	service := userService{users: repository, rbac: roleFilterAccessService{userIDs: []uint64{3, 9}}}

	page, err := service.ListUsersPage(context.Background(), ListQuery{Keyword: "  admin ", Status: "enabled", RoleID: &roleID, Limit: 200, Offset: 4})
	if err != nil {
		t.Fatalf("ListUsersPage() error = %v", err)
	}
	if page.Total != 1 || page.Limit != maximumUserListLimit || page.Offset != 4 {
		t.Fatalf("ListUsersPage() = %#v", page)
	}
	if repository.filter.Keyword != "admin" || repository.filter.Status != "enabled" || repository.filter.Limit != maximumUserListLimit || repository.filter.Offset != 4 {
		t.Fatalf("repository filter = %#v", repository.filter)
	}
	if len(repository.filter.UserIDs) != 2 || repository.filter.UserIDs[0] != 3 || repository.filter.UserIDs[1] != 9 {
		t.Fatalf("role-filter user IDs = %#v", repository.filter.UserIDs)
	}
}

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
