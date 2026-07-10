package storeent

import (
	"context"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/modules/user/ent/enttest"
	userstore "graft/server/modules/user/store"
)

func TestUserRepositoryGetByUsernameReturnsOnlyActiveProfile(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:user-storeent-get-by-username?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close user ent client: %v", err)
		}
	})

	if _, err := client.User.Create().SetUsername("active-user").SetDisplay("Active User").SetStatus("enabled").Save(context.Background()); err != nil {
		t.Fatalf("create active user: %v", err)
	}
	if _, err := client.User.Create().SetUsername("deleted-user").SetDisplay("Deleted User").SetStatus("enabled").SetDeletedAt(1).Save(context.Background()); err != nil {
		t.Fatalf("create deleted user: %v", err)
	}

	repo, err := newUserRepository(client)
	if err != nil {
		t.Fatalf("newUserRepository() error = %v", err)
	}

	active, err := repo.GetByUsername(context.Background(), "active-user")
	if err != nil {
		t.Fatalf("GetByUsername(active-user) error = %v", err)
	}
	if active.Username != "active-user" {
		t.Fatalf("GetByUsername(active-user) = %#v", active)
	}

	_, err = repo.GetByUsername(context.Background(), "deleted-user")
	if !errors.Is(err, userstore.ErrUserNotFound) {
		t.Fatalf("GetByUsername(deleted-user) error = %v, want ErrUserNotFound", err)
	}
}
