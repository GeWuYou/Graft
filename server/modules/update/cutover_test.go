package update

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
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

func TestCutoverV1PurgesHistoricalEventsWithoutCurrentSnapshotIdempotently(t *testing.T) {
	root := t.TempDir()
	historical := filepath.Join(root, "events", "old-operation", "00000000000000000001.json")
	if err := os.MkdirAll(filepath.Dir(historical), 0o750); err != nil {
		t.Fatalf("create historical events: %v", err)
	}
	if err := os.WriteFile(historical, []byte(`{"revision":1}`), 0o600); err != nil {
		t.Fatalf("write historical event: %v", err)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		if err := CutoverV1(context.Background(), root, (*sql.DB)(nil), nil, nil); err != nil {
			t.Fatalf("cutover attempt %d: %v", attempt, err)
		}
		if _, err := os.Stat(filepath.Join(root, "events")); !os.IsNotExist(err) {
			t.Fatalf("legacy events root still exists after attempt %d: %v", attempt, err)
		}
	}
}
