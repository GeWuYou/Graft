package healthopenapi

import "testing"

func TestGetHealthzStatusKeepsGeneratedEnum(t *testing.T) {
	t.Parallel()

	if !Ok.Valid() {
		t.Fatalf("expected generated health status enum to remain valid")
	}
}
