package auth

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/moduleapi"
)

func TestCredentialManagementExposesStablePasswordPolicyErrors(t *testing.T) {
	t.Parallel()

	service, _, user := newRuntimeRegressionService(t, false)
	for _, test := range []struct {
		name string
		call func() error
		want error
	}{
		{
			name: "provision policy violation",
			call: func() error {
				return service.ProvisionPasswordCredential(context.Background(), user.ID, "short1", false)
			},
			want: moduleapi.ErrPasswordPolicyViolation,
		},
		{
			name: "reset reuse forbidden",
			call: func() error {
				return service.ResetPassword(context.Background(), user.ID, defaultAdminPassword)
			},
			want: moduleapi.ErrPasswordReuseForbidden,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); !errors.Is(err, test.want) {
				t.Fatalf("credential management error = %v, want %v", err, test.want)
			}
		})
	}
}
