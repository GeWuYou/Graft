package moduleapi

import (
	"errors"
	"testing"
)

func TestValidateDeliveryActor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		actor   DeliveryActor
		wantErr error
	}{
		{
			name:  "accepts authenticated deployment principal",
			actor: DeliveryActor{ID: "github-actions-prod", Type: "service"},
		},
		{
			name:    "rejects missing ID",
			actor:   DeliveryActor{ID: " \t", Type: "service"},
			wantErr: ErrDeliveryActorInvalid,
		},
		{
			name:    "rejects missing type",
			actor:   DeliveryActor{ID: "github-actions-prod", Type: "\n"},
			wantErr: ErrDeliveryActorInvalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateDeliveryActor(testCase.actor)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("ValidateDeliveryActor(%#v) error = %v, want %v", testCase.actor, err, testCase.wantErr)
			}
		})
	}
}
