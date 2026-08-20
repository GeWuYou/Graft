package runtimetarget

import (
	"context"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

type assignmentUserSummaryReader struct {
	requestedIDs []uint64
	items        []moduleapi.UserAccountSummary
}

func (r *assignmentUserSummaryReader) ListUserSummariesByIDs(_ context.Context, userIDs []uint64) ([]moduleapi.UserAccountSummary, error) {
	r.requestedIDs = append([]uint64(nil), userIDs...)
	return r.items, nil
}

func TestAssignmentHTTPItemsEnrichAvailableUsersAndKeepMissingAssignments(t *testing.T) {
	reader := &assignmentUserSummaryReader{items: []moduleapi.UserAccountSummary{{
		ID: 7, Username: "operator", Display: "Operator", Status: "enabled",
	}}}
	module := &Module{userSummaries: reader}
	assignments := []store.UserAssignment{
		{TargetID: 3, UserID: 7, CreatedAt: time.Unix(10, 0).UTC(), CreatedBy: 1},
		{TargetID: 3, UserID: 9, CreatedAt: time.Unix(20, 0).UTC(), CreatedBy: 1},
	}

	summaries, err := module.assignmentUserSummaryMap(context.Background(), assignmentUserIDs(assignments))
	if err != nil {
		t.Fatalf("assignmentUserSummaryMap() error = %v", err)
	}
	items := assignmentHTTPItems(assignments, summaries)
	if len(reader.requestedIDs) != 2 || reader.requestedIDs[0] != 7 || reader.requestedIDs[1] != 9 {
		t.Fatalf("requested IDs = %#v", reader.requestedIDs)
	}
	if items[0].Username != "operator" || items[0].Display != "Operator" || items[0].Status != "enabled" {
		t.Fatalf("enriched assignment = %#v", items[0])
	}
	if items[1].UserID != 9 || items[1].Username != "" || items[1].Display != "" || items[1].Status != "" {
		t.Fatalf("missing-user assignment = %#v", items[1])
	}
}
