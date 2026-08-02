package runtimetarget

import (
	"encoding/json"
	"math"
	"testing"

	"graft/server/internal/moduleapi"
)

func TestValidRuntimeTargetSavedViewRejectsUnknownQueryStateKey(t *testing.T) {
	queryState, err := json.Marshal(map[string]any{"unknown": "value"})
	if err != nil {
		t.Fatalf("marshal query state: %v", err)
	}
	input := runtimeTargetSavedViewInput{Name: "My view", QueryState: queryState, PageSize: 10, VisibleColumns: []string{"workloads", "cpu", "memory", "storage"}}
	if validRuntimeTargetSavedView(input) {
		t.Fatal("unknown query state key was accepted")
	}
}

func TestRuntimeTargetSavedViewResponseRejectsInt64OverflowID(t *testing.T) {
	_, err := runtimeTargetSavedViewResponse(moduleapi.SavedView{ID: uint64(math.MaxInt64) + 1})
	if err == nil {
		t.Fatal("overflow saved view ID was accepted")
	}
}
