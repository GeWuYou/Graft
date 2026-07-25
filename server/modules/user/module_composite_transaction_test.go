package user

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"

	"graft/server/internal/moduleapi"
	authent "graft/server/modules/auth/ent"
	authcredential "graft/server/modules/auth/ent/authcredential"
	authrefreshsession "graft/server/modules/auth/ent/authrefreshsession"
	authstoreent "graft/server/modules/auth/storeent"
	userent "graft/server/modules/user/ent"
	userentuser "graft/server/modules/user/ent/user"
	userstore "graft/server/modules/user/store"
	userstoreent "graft/server/modules/user/storeent"
)

//nolint:gosec // Ent user IDs are constrained by the SQLite test schema.
func TestCompositeTransactionCommitsProfileAndCredential(t *testing.T) {
	service, _, users, credentials := newCompositeTransactionTestService(t)

	created, err := service.CreateUser(context.Background(), CreateUserCommand{
		Username: "atomic-create",
		Display:  "Atomic Create",
		Password: "Password1234",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if created.ID == 0 {
		t.Fatal("CreateUser() returned an empty user ID")
	}
	if _, err := users.User.Query().Where(userentuser.IDEQ(int(created.ID))).Only(context.Background()); err != nil {
		t.Fatalf("query committed user: %v", err)
	}
	credential, err := credentials.AuthCredential.Query().Where(authcredential.UserIDEQ(created.ID)).Only(context.Background())
	if err != nil {
		t.Fatalf("query committed credential: %v", err)
	}
	if credential.PasswordHash == nil || *credential.PasswordHash != "transaction-hash" {
		t.Fatalf("credential password hash = %v, want transaction-hash", credential.PasswordHash)
	}
}

func TestCompositeTransactionRollsBackProfileWhenCredentialWriteFails(t *testing.T) {
	service, db, users, credentials := newCompositeTransactionTestService(t)
	installAbortTrigger(t, db, "auth_credentials", "INSERT", "credential write failed")

	_, err := service.CreateUser(context.Background(), CreateUserCommand{
		Username: "credential-rollback",
		Display:  "Credential Rollback",
		Password: "Password1234",
	})
	if err == nil || !strings.Contains(err.Error(), "credential write failed") {
		t.Fatalf("CreateUser() error = %v, want credential write failure", err)
	}
	assertNoUserOrCredential(t, users, credentials, "credential-rollback")
}

//nolint:gosec // Fixtures use Ent's int ID while the module contract uses uint64.
func TestCompositeTransactionRollsBackDisableWhenSessionRevocationFails(t *testing.T) {
	service, db, users, credentials := newCompositeTransactionTestService(t)
	user := createCompositeTransactionUser(t, users, "disable-rollback")
	createCompositeTransactionSession(t, credentials, uint64(user.ID), "disable-session")
	installAbortTrigger(t, db, "auth_refresh_sessions", "UPDATE", "session revocation failed")

	_, err := service.SetUserStatus(context.Background(), UpdateUserStatusCommand{ID: uint64(user.ID), Status: "disabled"})
	if err == nil || !strings.Contains(err.Error(), "session revocation failed") {
		t.Fatalf("SetUserStatus() error = %v, want session revocation failure", err)
	}
	assertUserEnabledAndSessionActive(t, users, credentials, uint64(user.ID), "disable-session")
}

//nolint:gosec // Fixtures use Ent's int ID while the module contract uses uint64.
func TestCompositeTransactionRollsBackDeleteWhenSessionRevocationFails(t *testing.T) {
	service, db, users, credentials := newCompositeTransactionTestService(t)
	user := createCompositeTransactionUser(t, users, "delete-rollback")
	createCompositeTransactionSession(t, credentials, uint64(user.ID), "delete-session")
	installAbortTrigger(t, db, "auth_refresh_sessions", "UPDATE", "session revocation failed")

	err := service.DeleteUser(context.Background(), uint64(user.ID))
	if err == nil || !strings.Contains(err.Error(), "session revocation failed") {
		t.Fatalf("DeleteUser() error = %v, want session revocation failure", err)
	}
	assertUserEnabledAndSessionActive(t, users, credentials, uint64(user.ID), "delete-session")
}

//nolint:revive // The four returned test fixtures are consumed together by each scenario.
func newCompositeTransactionTestService(t *testing.T) (userService, *sql.DB, *userent.Client, *authent.Client) {
	t.Helper()
	dsn := fmt.Sprintf("file:user-auth-composite-%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	users := userent.NewClient(userent.Driver(entsql.OpenDB("sqlite3", db)))
	if err := users.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create user schema: %v", err)
	}
	credentials := authent.NewClient(authent.Driver(entsql.OpenDB("sqlite3", db)))
	if err := credentials.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create auth schema: %v", err)
	}
	repository, err := userstoreent.NewUserRepository(users, db)
	if err != nil {
		t.Fatalf("create user repository: %v", err)
	}
	transactions, ok := repository.(userstore.TransactionRunner)
	if !ok {
		t.Fatal("user repository does not implement TransactionRunner")
	}
	composites, ok := repository.(userstore.CompositeTransactionRunner)
	if !ok {
		t.Fatal("user repository does not implement CompositeTransactionRunner")
	}
	adapterFactory, err := authstoreent.NewTransactionAdapterFactory(func(context.Context, moduleapi.AuthCredentialProvisionInput) (string, time.Time, error) {
		return "transaction-hash", time.Now().UTC(), nil
	})
	if err != nil {
		t.Fatalf("create auth transaction adapter factory: %v", err)
	}

	return userService{
		users:        repository,
		transactions: transactions,
		composites:   composites,
		authTx:       adapterFactory,
		logger:       zap.NewNop(),
	}, db, users, credentials
}

func createCompositeTransactionUser(t *testing.T, users *userent.Client, username string) *userent.User {
	t.Helper()
	user, err := users.User.Create().SetUsername(username).SetDisplay(username).SetStatus("enabled").Save(context.Background())
	if err != nil {
		t.Fatalf("create user fixture: %v", err)
	}
	return user
}

func createCompositeTransactionSession(t *testing.T, credentials *authent.Client, userID uint64, tokenID string) {
	t.Helper()
	if _, err := credentials.AuthRefreshSession.Create().SetUserID(userID).SetTokenID(tokenID).SetExpiresAt(time.Now().Add(time.Hour)).Save(context.Background()); err != nil {
		t.Fatalf("create refresh session fixture: %v", err)
	}
}

//nolint:gosec // Every input is a test-controlled SQLite identifier or message.
func installAbortTrigger(t *testing.T, db *sql.DB, table, operation, message string) {
	t.Helper()
	statement := fmt.Sprintf("CREATE TRIGGER abort_%s_%s BEFORE %s ON %s BEGIN SELECT RAISE(ABORT, %q); END", table, strings.ToLower(operation), operation, table, message)
	if _, err := db.Exec(statement); err != nil {
		t.Fatalf("install %s trigger: %v", table, err)
	}
}

func assertNoUserOrCredential(t *testing.T, users *userent.Client, credentials *authent.Client, username string) {
	t.Helper()
	if count, err := users.User.Query().Where(userentuser.UsernameEQ(username)).Count(context.Background()); err != nil {
		t.Fatalf("count rolled-back user: %v", err)
	} else if count != 0 {
		t.Fatalf("rolled-back user count = %d, want 0", count)
	}
	if count, err := credentials.AuthCredential.Query().Count(context.Background()); err != nil {
		t.Fatalf("count rolled-back credentials: %v", err)
	} else if count != 0 {
		t.Fatalf("rolled-back credential count = %d, want 0", count)
	}
}

//nolint:gosec // Ent user IDs are constrained by the SQLite test schema.
func assertUserEnabledAndSessionActive(t *testing.T, users *userent.Client, credentials *authent.Client, userID uint64, tokenID string) {
	t.Helper()
	user, err := users.User.Get(context.Background(), int(userID))
	if err != nil {
		t.Fatalf("query rolled-back user: %v", err)
	}
	if user.Status != "enabled" || user.DeletedAt != 0 {
		t.Fatalf("user after rollback = status %q, deleted_at %d; want enabled and visible", user.Status, user.DeletedAt)
	}
	session, err := credentials.AuthRefreshSession.Query().Where(authrefreshsession.TokenIDEQ(tokenID)).Only(context.Background())
	if err != nil {
		t.Fatalf("query rolled-back session: %v", err)
	}
	if session.RevokedAt != nil {
		t.Fatalf("session RevokedAt = %v, want nil", session.RevokedAt)
	}
}
