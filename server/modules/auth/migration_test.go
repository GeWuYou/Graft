package auth

import (
	"os"
	"strings"
	"testing"
)

func TestLegacyAuthDataForwardMigrationPreservesCredentialAndSessionState(t *testing.T) {
	t.Parallel()

	contents, err := os.ReadFile("migrations/202607100004_copy_legacy_auth_data.sql")
	if err != nil {
		t.Fatalf("read auth forward migration: %v", err)
	}
	sql := string(contents)

	for _, want := range []string{
		`INSERT INTO "auth_credentials"`,
		`FROM "users"`,
		`"password_hash"`,
		`"must_change_password"`,
		`"password_changed_at"`,
		`INSERT INTO "auth_refresh_sessions"`,
		`FROM "refresh_sessions"`,
		`"token_id"`,
		`"expires_at"`,
		`"revoked_at"`,
		`"replaced_by_token_id"`,
		`pg_get_serial_sequence('auth_credentials', 'id')`,
		`pg_get_serial_sequence('auth_refresh_sessions', 'id')`,
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("forward migration must preserve %q", want)
		}
	}

	if strings.Contains(sql, "ON CONFLICT") {
		t.Fatal("forward migration must not add a compatibility or dual-write path")
	}
}
