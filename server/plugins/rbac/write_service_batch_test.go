package rbac

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/pluginapi"
	rbacstore "graft/server/plugins/rbac/store"
)

func TestManagementWriterAtomicBatchWriterUsesAtomicPath(t *testing.T) {
	testCases := []atomicBatchMutationTestCase{{mode: "replace"}, {mode: "add"}}

	for _, tc := range testCases {
		testManagementWriterAtomicBatchPath(t, tc)
	}
}

func TestManagementWriterRemoveRolesFromUsersUsesAtomicBatchWriter(t *testing.T) {
	var called bool
	repo := testRBACRepository{
		roles: []rbacstore.Role{
			{ID: 1, Name: "admin", Builtin: true, Status: rbacstore.RoleStatusEnabled},
			{ID: 3, Name: "editor", Status: rbacstore.RoleStatusEnabled},
		},
		roleByID: map[uint64]rbacstore.Role{
			1: {ID: 1, Name: "admin", Builtin: true, Status: rbacstore.RoleStatusEnabled},
			3: {ID: 3, Name: "editor", Status: rbacstore.RoleStatusEnabled},
		},
		removeUserRoles: func(context.Context, rbacstore.RemoveRolesFromUserInput) error {
			t.Fatal("per-user remove should not be used when atomic batch writer is available")
			return nil
		},
		removeUserRolesBatch: func(_ context.Context, input rbacstore.BatchUserRoleMutationInput) error {
			called = true
			if len(input.UserIDs) != 2 || input.UserIDs[0] != 11 || input.UserIDs[1] != 22 {
				t.Fatalf("unexpected user ids: %#v", input.UserIDs)
			}
			if len(input.RoleIDs) != 1 || input.RoleIDs[0] != 3 {
				t.Fatalf("unexpected role ids: %#v", input.RoleIDs)
			}
			return nil
		},
	}
	writer := managementWriter{
		users: testUserService{users: map[uint64]pluginapi.UserSummary{
			11: {ID: 11},
			22: {ID: 22},
		}},
		rbac: repo,
	}

	if err := writer.RemoveRolesFromUsers(context.Background(), rbacstore.BatchUserRoleMutationInput{
		UserIDs: []uint64{11, 22},
		RoleIDs: []uint64{3},
	}); err != nil {
		t.Fatalf("remove roles from users: %v", err)
	}
	if !called {
		t.Fatal("expected atomic batch remove to be called")
	}
}

func TestManagementWriterReplaceRolesForUsersPropagatesAtomicBatchError(t *testing.T) {
	wantErr := errors.New("boom")
	repo := testRBACRepository{
		roles: []rbacstore.Role{
			{ID: 3, Name: "editor", Status: rbacstore.RoleStatusEnabled},
		},
		roleByID: map[uint64]rbacstore.Role{
			3: {ID: 3, Name: "editor", Status: rbacstore.RoleStatusEnabled},
		},
		replaceUserRolesBatch: func(context.Context, rbacstore.BatchUserRoleMutationInput) error {
			return wantErr
		},
	}
	writer := managementWriter{
		users: testUserService{users: map[uint64]pluginapi.UserSummary{
			11: {ID: 11},
		}},
		rbac: repo,
	}

	err := writer.ReplaceRolesForUsers(context.Background(), rbacstore.BatchUserRoleMutationInput{
		UserIDs: []uint64{11},
		RoleIDs: []uint64{3},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestManagementWriterBatchMutationsRequireAtomicBatchWriter(t *testing.T) {
	repo := nonAtomicBatchRepo{
		Repository: testRBACRepository{
			roles: []rbacstore.Role{
				{ID: 3, Name: "editor", Status: rbacstore.RoleStatusEnabled},
			},
			roleByID: map[uint64]rbacstore.Role{
				3: {ID: 3, Name: "editor", Status: rbacstore.RoleStatusEnabled},
			},
		},
	}
	writer := managementWriter{
		users: testUserService{users: map[uint64]pluginapi.UserSummary{
			11: {ID: 11},
		}},
		rbac: repo,
	}
	input := rbacstore.BatchUserRoleMutationInput{
		UserIDs: []uint64{11},
		RoleIDs: []uint64{3},
	}

	for _, tc := range []struct {
		name string
		run  func(context.Context, managementWriter, rbacstore.BatchUserRoleMutationInput) error
	}{
		{
			name: "replace",
			run: func(ctx context.Context, writer managementWriter, input rbacstore.BatchUserRoleMutationInput) error {
				return writer.ReplaceRolesForUsers(ctx, input)
			},
		},
		{
			name: "add",
			run: func(ctx context.Context, writer managementWriter, input rbacstore.BatchUserRoleMutationInput) error {
				return writer.AddRolesToUsers(ctx, input)
			},
		},
		{
			name: "remove",
			run: func(ctx context.Context, writer managementWriter, input rbacstore.BatchUserRoleMutationInput) error {
				return writer.RemoveRolesFromUsers(ctx, input)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(context.Background(), writer, input)
			if !errors.Is(err, errAtomicBatchWriterMissing) {
				t.Fatalf("expected %v, got %v", errAtomicBatchWriterMissing, err)
			}
		})
	}
}

type atomicBatchMutationTestCase struct {
	mode string
}

func testManagementWriterAtomicBatchPath(t *testing.T, tc atomicBatchMutationTestCase) {
	t.Helper()

	t.Run(tc.mode, func(t *testing.T) {
		var called bool
		repo := testRBACRepository{
			roles: []rbacstore.Role{
				{ID: 3, Name: "editor", Status: rbacstore.RoleStatusEnabled},
			},
			roleByID: map[uint64]rbacstore.Role{
				3: {ID: 3, Name: "editor", Status: rbacstore.RoleStatusEnabled},
			},
		}
		writer := managementWriter{
			users: testUserService{users: map[uint64]pluginapi.UserSummary{
				11: {ID: 11},
				22: {ID: 22},
			}},
			rbac: configureAtomicBatchRepo(t, repo, tc.mode, &called),
		}

		if err := runAtomicBatchMutation(context.Background(), writer, tc.mode, rbacstore.BatchUserRoleMutationInput{
			UserIDs: []uint64{11, 22},
			RoleIDs: []uint64{3},
		}); err != nil {
			t.Fatalf("run atomic batch mutation: %v", err)
		}
		if !called {
			t.Fatal("expected atomic batch writer to be called")
		}
	})
}

func configureAtomicBatchRepo(t *testing.T, repo testRBACRepository, mode string, called *bool) testRBACRepository {
	t.Helper()

	switch mode {
	case "replace":
		repo.replaceUserRoles = func(context.Context, rbacstore.ReplaceRolesForUserInput) error {
			t.Fatal("per-user replace should not be used when atomic batch writer is available")
			return nil
		}
		repo.replaceUserRolesBatch = func(_ context.Context, input rbacstore.BatchUserRoleMutationInput) error {
			*called = true
			assertExpectedBatchInput(t, input)
			return nil
		}
	case "add":
		repo.addUserRoles = func(context.Context, rbacstore.AddRolesToUserInput) error {
			t.Fatal("per-user add should not be used when atomic batch writer is available")
			return nil
		}
		repo.addUserRolesBatch = func(_ context.Context, input rbacstore.BatchUserRoleMutationInput) error {
			*called = true
			assertExpectedBatchInput(t, input)
			return nil
		}
	default:
		t.Fatalf("unsupported atomic batch mode %q", mode)
	}

	return repo
}

func runAtomicBatchMutation(
	ctx context.Context,
	writer managementWriter,
	mode string,
	input rbacstore.BatchUserRoleMutationInput,
) error {
	switch mode {
	case "replace":
		return writer.ReplaceRolesForUsers(ctx, input)
	case "add":
		return writer.AddRolesToUsers(ctx, input)
	default:
		return errors.New("unsupported atomic batch mode")
	}
}

func assertExpectedBatchInput(t *testing.T, input rbacstore.BatchUserRoleMutationInput) {
	t.Helper()

	if len(input.UserIDs) != 2 || input.UserIDs[0] != 11 || input.UserIDs[1] != 22 {
		t.Fatalf("unexpected user ids: %#v", input.UserIDs)
	}
	if len(input.RoleIDs) != 1 || input.RoleIDs[0] != 3 {
		t.Fatalf("unexpected role ids: %#v", input.RoleIDs)
	}
}

type nonAtomicBatchRepo struct {
	rbacstore.Repository
}
