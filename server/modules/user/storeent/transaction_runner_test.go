package storeent

import (
	"context"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/modules/user/ent/enttest"
	userstore "graft/server/modules/user/store"
)

func TestUserTransactionRunnerRollsBackFailedProfileWrite(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:user-storeent-transaction?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close user ent client: %v", err)
		}
	})

	repo, err := newUserRepository(client)
	if err != nil {
		t.Fatalf("newUserRepository() error = %v", err)
	}
	callbackErr := errors.New("stop profile lifecycle")
	err = repo.RunInTransaction(context.Background(), func(ctx context.Context, profiles userstore.UserRepository) error {
		if _, err := profiles.Create(ctx, userstore.CreateUserInput{Username: "rolled-back-user", Display: "Rolled Back", Status: "enabled"}); err != nil {
			return err
		}
		return callbackErr
	})
	if !errors.Is(err, callbackErr) {
		t.Fatalf("RunInTransaction() error = %v, want callback error", err)
	}
	if _, err := repo.GetByUsername(context.Background(), "rolled-back-user"); !errors.Is(err, userstore.ErrUserNotFound) {
		t.Fatalf("GetByUsername() error = %v, want ErrUserNotFound after rollback", err)
	}
}
