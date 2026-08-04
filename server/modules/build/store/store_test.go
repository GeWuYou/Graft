package store

import (
	"reflect"
	"testing"
	"time"
)

func TestJobListFiltersPreserveExactFilterArgumentOrder(t *testing.T) {
	repository := "example/app"
	tag := "v1"
	applicationID := uint64(42)
	after := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	before := after.Add(24 * time.Hour)

	where, args := jobListFilters(ListQuery{ApplicationID: &applicationID, ImageRepository: &repository, ImageTag: &tag, CreatedAfter: &after, CreatedBefore: &before})
	wantWhere := []string{"1 = 1", "j.application_id = $1", "j.image_repository = $2", "j.image_tag = $3", "j.created_at >= $4", "j.created_at <= $5"}
	if !reflect.DeepEqual(where, wantWhere) {
		t.Fatalf("unexpected where clauses: %#v", where)
	}
	wantArgs := []any{applicationID, repository, tag, after, before}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected filter arguments: %#v", args)
	}
}
