package httpx

import (
	"database/sql"
	"strings"
	"testing"
	"time"
)

type scanAccessLogTestScanner struct {
	values []any
}

func (s scanAccessLogTestScanner) Scan(dest ...any) error {
	for index := range dest {
		switch target := dest[index].(type) {
		case *int64:
			*target = s.values[index].(int64)
		case *string:
			*target = s.values[index].(string)
		case *int:
			*target = s.values[index].(int)
		case *sql.NullString:
			*target = s.values[index].(sql.NullString)
		case *sql.NullInt64:
			*target = s.values[index].(sql.NullInt64)
		case *time.Time:
			*target = s.values[index].(time.Time)
		default:
			panic("unsupported scan target")
		}
	}
	return nil
}

func TestScanAccessLogRejectsNegativeID(t *testing.T) {
	now := time.Date(2026, 6, 29, 8, 0, 0, 0, time.UTC)
	_, err := scanAccessLog(scanAccessLogTestScanner{values: []any{
		int64(-1),
		"req-1",
		sql.NullString{},
		"GET",
		"/healthz",
		sql.NullString{},
		200,
		int64(1),
		sql.NullString{},
		sql.NullString{},
		sql.NullInt64{},
		sql.NullString{},
		sql.NullInt64{},
		sql.NullInt64{},
		now,
		now,
	}})
	if err == nil {
		t.Fatal("expected negative access log id error")
	}
	if !strings.Contains(err.Error(), "access log id must be non-negative") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScanAccessLogRejectsNegativeUserID(t *testing.T) {
	now := time.Date(2026, 6, 29, 8, 0, 0, 0, time.UTC)
	_, err := scanAccessLog(scanAccessLogTestScanner{values: []any{
		int64(1),
		"req-1",
		sql.NullString{},
		"GET",
		"/healthz",
		sql.NullString{},
		200,
		int64(1),
		sql.NullString{},
		sql.NullString{},
		sql.NullInt64{Int64: -7, Valid: true},
		sql.NullString{},
		sql.NullInt64{},
		sql.NullInt64{},
		now,
		now,
	}})
	if err == nil {
		t.Fatal("expected negative access log user id error")
	}
	if !strings.Contains(err.Error(), "access log user id must be non-negative") {
		t.Fatalf("unexpected error: %v", err)
	}
}
