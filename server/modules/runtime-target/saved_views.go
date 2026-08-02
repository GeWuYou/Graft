package runtimetarget

import (
	"encoding/json"
	"errors"
	"strings"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

const runtimeTargetSavedViewSurface = "runtime-target.list"

var errRuntimeTargetInvalidSavedView = errors.New("invalid runtime target saved view")

type runtimeTargetSavedViewInput struct {
	Name           string
	QueryState     json.RawMessage
	PageSize       int
	VisibleColumns []string
	IsDefault      bool
}
type runtimeTargetSavedViewState struct {
	Keyword        *string  `json:"keyword"`
	Provider       *string  `json:"provider"`
	ConnectionKind *string  `json:"connection_kind"`
	Health         *string  `json:"health"`
	Sort           []string `json:"sort"`
}

func parseRuntimeTargetSavedViewInput(request generated.SavedViewRequest) (runtimeTargetSavedViewInput, error) {
	raw, err := json.Marshal(request.QueryState)
	if err != nil {
		return runtimeTargetSavedViewInput{}, err
	}
	input := runtimeTargetSavedViewInput{Name: request.Name, QueryState: raw, PageSize: request.PageSize, VisibleColumns: append([]string(nil), request.VisibleColumns...), IsDefault: request.IsDefault != nil && *request.IsDefault}
	if !validRuntimeTargetSavedView(input) {
		return runtimeTargetSavedViewInput{}, errRuntimeTargetInvalidSavedView
	}
	return input, nil
}

//nolint:gocognit,gocyclo,cyclop // 保存视图字段白名单需要与公开查询契约逐项对应。
func validRuntimeTargetSavedView(input runtimeTargetSavedViewInput) bool {
	if strings.TrimSpace(input.Name) == "" || input.PageSize != 10 && input.PageSize != 20 && input.PageSize != 50 && input.PageSize != 100 || !json.Valid(input.QueryState) {
		return false
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(input.QueryState, &raw) != nil || raw == nil {
		return false
	}
	for key := range raw {
		if key != "keyword" && key != "provider" && key != "connection_kind" && key != "health" && key != "sort" {
			return false
		}
	}
	var state runtimeTargetSavedViewState
	if json.Unmarshal(input.QueryState, &state) != nil {
		return false
	}
	if state.Keyword != nil && (strings.TrimSpace(*state.Keyword) == "" || len(*state.Keyword) > 128) {
		return false
	}
	if state.Provider != nil && *state.Provider != "docker" || state.ConnectionKind != nil && *state.ConnectionKind != "unix_socket" || state.Health != nil && *state.Health != "healthy" && *state.Health != "unavailable" || len(state.Sort) > 1 {
		return false
	}
	for _, sort := range state.Sort {
		if !validRuntimeTargetListQuery(storeListQueryForSort(sort)) {
			return false
		}
	}
	for _, column := range input.VisibleColumns {
		switch strings.TrimSpace(column) {
		case "displayName", "provider", "endpoint", "connectionKind", "health", "lastCheckedAt", "workloads", "cpu", "memory", "storage", "operation":
		default:
			return false
		}
	}
	return true
}

func storeListQueryForSort(sort string) store.ListQuery { return store.ListQuery{Sort: sort} }

func runtimeTargetSavedViewResponse(view moduleapi.SavedView) (generated.SavedView, error) {
	state := map[string]interface{}{}
	if view.ID == 0 || view.ID > uint64(^uint64(0)>>1) || json.Unmarshal(view.QueryState, &state) != nil {
		return generated.SavedView{}, errRuntimeTargetInvalidSavedView
	}
	return generated.SavedView{Id: int64(view.ID), Name: view.Name, QueryState: state, PageSize: view.PageSize, VisibleColumns: append([]string(nil), view.VisibleColumns...), IsDefault: view.IsDefault, CreatedAt: view.CreatedAt, UpdatedAt: view.UpdatedAt}, nil
}
