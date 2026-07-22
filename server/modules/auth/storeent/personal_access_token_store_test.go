package storeent

import (
	"math"
	"testing"
)

func TestPersonalAccessTokenEntIDRejectsValuesOutsideEntIntegerRange(t *testing.T) {
	t.Parallel()

	maxID := uint64(math.MaxInt)
	if got, err := personalAccessTokenEntID(maxID); err != nil || got != math.MaxInt {
		t.Fatalf("convert maximum ent id: got (%d, %v), want (%d, nil)", got, err, math.MaxInt)
	}

	if _, err := personalAccessTokenEntID(maxID + 1); err == nil {
		t.Fatal("expected id above the ent integer range to be rejected")
	}
}
