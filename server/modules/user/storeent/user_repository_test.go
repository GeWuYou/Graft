package storeent

import (
	"context"
	"errors"
	"strconv"
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
	deletedRecord, err := client.User.Create().SetUsername("deleted-user").SetDisplay("Deleted User").SetStatus("enabled").SetDeletedAt(1).Save(context.Background())
	if err != nil {
		t.Fatalf("create deleted user: %v", err)
	}
	deletedUserID, err := strconv.ParseUint(strconv.Itoa(deletedRecord.ID), 10, 64)
	if err != nil {
		t.Fatalf("convert deleted user id: %v", err)
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

	deletionState, err := repo.GetDeletionState(context.Background(), deletedUserID)
	if err != nil {
		t.Fatalf("GetDeletionState(deleted-user) error = %v", err)
	}
	if !deletionState.Deleted || deletionState.Username != "deleted-user" {
		t.Fatalf("GetDeletionState(deleted-user) = %#v", deletionState)
	}
	if _, err := repo.GetByID(context.Background(), deletedUserID); !errors.Is(err, userstore.ErrUserNotFound) {
		t.Fatalf("GetByID(deleted-user) error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepositoryListPageFiltersCountsAndOrdersByID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:user-storeent-list-page?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close user ent client: %v", err)
		}
	})
	for _, record := range []struct {
		username string
		display  string
		status   string
	}{
		{"alpha", "Alpha User", "enabled"},
		{"bravo", "Bravo User", "disabled"},
		{"charlie", "Alpha Operator", "enabled"},
	} {
		if _, err := client.User.Create().SetUsername(record.username).SetDisplay(record.display).SetStatus(record.status).Save(context.Background()); err != nil {
			t.Fatalf("create %s: %v", record.username, err)
		}
	}
	repo, err := newUserRepository(client)
	if err != nil {
		t.Fatalf("newUserRepository() error = %v", err)
	}

	users, total, err := repo.ListPage(context.Background(), userstore.UserListFilter{Keyword: "alpha", Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("ListPage() error = %v", err)
	}
	if total != 2 {
		t.Fatalf("ListPage() total = %d, want 2", total)
	}
	if len(users) != 1 || users[0].Username != "charlie" {
		t.Fatalf("ListPage() users = %#v, want second ID-ordered alpha result", users)
	}

	users, total, err = repo.ListPage(context.Background(), userstore.UserListFilter{Status: "disabled", UserIDs: []uint64{1, 2}, Limit: 20})
	if err != nil {
		t.Fatalf("ListPage(status and IDs) error = %v", err)
	}
	if total != 1 || len(users) != 1 || users[0].Username != "bravo" {
		t.Fatalf("ListPage(status and IDs) = %#v, total %d; want bravo and total 1", users, total)
	}
}

func TestUserRepositoryListSummariesByIDsReturnsOnlyRequestedActiveUsers(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:user-storeent-list-summaries?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close user ent client: %v", err)
		}
	})
	first, err := client.User.Create().SetUsername("first").SetDisplay("First User").SetStatus("enabled").Save(context.Background())
	if err != nil {
		t.Fatalf("create first user: %v", err)
	}
	second, err := client.User.Create().SetUsername("second").SetDisplay("Second User").SetStatus("disabled").Save(context.Background())
	if err != nil {
		t.Fatalf("create second user: %v", err)
	}
	deleted, err := client.User.Create().SetUsername("deleted").SetDisplay("Deleted User").SetStatus("enabled").SetDeletedAt(1).Save(context.Background())
	if err != nil {
		t.Fatalf("create deleted user: %v", err)
	}
	repo, err := newUserRepository(client)
	if err != nil {
		t.Fatalf("newUserRepository() error = %v", err)
	}

	items, err := repo.ListSummariesByIDs(context.Background(), []uint64{
		toStoreID(second.ID),
		toStoreID(deleted.ID),
		toStoreID(first.ID),
	})
	if err != nil {
		t.Fatalf("ListSummariesByIDs() error = %v", err)
	}
	if len(items) != 2 ||
		items[0].Username != "first" ||
		items[1].Username != "second" ||
		items[1].Status != "disabled" {
		t.Fatalf("ListSummariesByIDs() = %#v", items)
	}
}
