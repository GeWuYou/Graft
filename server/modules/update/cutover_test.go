package update

import (
	"context"
	"database/sql"
	"testing"
)

func TestLegacyCutoverPurgeMarkerIsStable(t *testing.T) {
	if LegacyCutoverPurgeMarker != "PLATFORM_UPDATE_LEGACY_CUTOVER" {
		t.Fatalf("marker = %q", LegacyCutoverPurgeMarker)
	}
	if err := CutoverV1(context.Background(), t.TempDir(), (*sql.DB)(nil), nil, nil); err != nil {
		t.Fatalf("empty state should succeed before dependency use: %v", err)
	}
}
