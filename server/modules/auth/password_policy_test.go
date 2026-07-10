package auth

import (
	"strings"
	"testing"
)

func TestPasswordPolicyCountsCharactersRatherThanUTF8Bytes(t *testing.T) {
	t.Parallel()

	policy := newPasswordPolicy()
	for _, password := range []string{
		strings.Repeat("A", minimumPasswordLength-1) + "1",
		strings.Repeat("密", minimumPasswordLength-1) + "1",
	} {
		if err := policy.ValidateNewPassword(password); err != nil {
			t.Fatalf("ValidateNewPassword(%q) = %v, want success", password, err)
		}
	}
}
