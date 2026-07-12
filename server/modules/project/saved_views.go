package project

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
)

const projectListSavedViewSurface = "project.list"

var projectListSavedViewColumns = map[string]struct{}{
	"name": {}, "application_type": {}, "runtime_target": {}, "provider": {}, "source_kind": {}, "runtime_status": {}, "resources": {}, "drift_status": {}, "updated_at": {},
}

type projectListQueryState struct {
	Keyword         *string `json:"keyword"`
	ApplicationType *string `json:"application_type"`
	RuntimeTargetID *int64  `json:"runtime_target_id"`
	Provider        *string `json:"provider"`
	SourceKind      *string `json:"source_kind"`
	RuntimeStatus   *string `json:"runtime_status"`
	DriftStatus     *string `json:"drift_status"`
}

// savedViewRequest is the project-owned, consumer-specific state accepted by its saved-view routes.
type savedViewRequest struct {
	Name           string          `json:"name"`
	QueryState     json.RawMessage `json:"query_state"`
	PageSize       int             `json:"page_size"`
	VisibleColumns []string        `json:"visible_columns"`
}

func (s *Service) listSavedViews(ctx context.Context, ownerUserID uint64) ([]moduleapi.SavedView, error) {
	if s == nil || s.savedViews == nil || ownerUserID == 0 {
		return nil, errProjectInvalidArgument
	}
	return s.savedViews.List(ctx, ownerUserID, projectListSavedViewSurface)
}

func (s *Service) createSavedView(ctx context.Context, ownerUserID uint64, request savedViewRequest) (moduleapi.SavedView, error) {
	if s == nil || s.savedViews == nil || ownerUserID == 0 {
		return moduleapi.SavedView{}, errProjectInvalidArgument
	}
	if err := validateProjectListSavedView(request); err != nil {
		return moduleapi.SavedView{}, err
	}
	view, err := s.savedViews.Create(ctx, moduleapi.SavedViewCreateInput{OwnerUserID: ownerUserID, SurfaceKey: projectListSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns})
	return view, mapSavedViewError(err)
}

func (s *Service) updateSavedView(ctx context.Context, ownerUserID, id uint64, request savedViewRequest) (moduleapi.SavedView, error) {
	if s == nil || s.savedViews == nil || ownerUserID == 0 || id == 0 {
		return moduleapi.SavedView{}, errProjectInvalidArgument
	}
	if err := validateProjectListSavedView(request); err != nil {
		return moduleapi.SavedView{}, err
	}
	view, err := s.savedViews.Update(ctx, moduleapi.SavedViewUpdateInput{ID: id, OwnerUserID: ownerUserID, SurfaceKey: projectListSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns})
	return view, mapSavedViewError(err)
}

func (s *Service) deleteSavedView(ctx context.Context, ownerUserID, id uint64) error {
	if s == nil || s.savedViews == nil || ownerUserID == 0 || id == 0 {
		return errProjectInvalidArgument
	}
	return mapSavedViewError(s.savedViews.Delete(ctx, ownerUserID, projectListSavedViewSurface, id))
}

// validateProjectListSavedView validates the name, page size, query state, and visible columns of a project list saved-view request.
// It returns errProjectInvalidArgument when any field is invalid.
func validateProjectListSavedView(request savedViewRequest) error {
	if strings.TrimSpace(request.Name) == "" || request.PageSize < 1 || !json.Valid(request.QueryState) {
		return errProjectInvalidArgument
	}
	if err := validateProjectListQueryState(request.QueryState); err != nil {
		return err
	}
	return validateProjectListVisibleColumns(request.VisibleColumns)
}

// validateProjectListQueryState 校验项目列表已保存视图查询状态中的字段和值。
func validateProjectListQueryState(queryState json.RawMessage) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(queryState, &raw); err != nil || raw == nil {
		return errProjectInvalidArgument
	}
	if !validProjectListQueryStateFields(raw) {
		return errProjectInvalidArgument
	}
	var state projectListQueryState
	if err := json.Unmarshal(queryState, &state); err != nil {
		return errProjectInvalidArgument
	}
	return validateProjectListQueryStateValues(state)
}

func validProjectListQueryStateFields(raw map[string]json.RawMessage) bool {
	for key := range raw {
		switch key {
		case "keyword", "application_type", "runtime_target_id", "provider", "source_kind", "runtime_status", "drift_status":
		default:
			return false
		}
	}
	return true
}

func validateProjectListQueryStateValues(state projectListQueryState) error {
	if !validProjectListQueryStateStrings(state) {
		return errProjectInvalidArgument
	}
	if !validProjectListQueryStateTarget(state) {
		return errProjectInvalidArgument
	}
	if !validProjectListQueryStateEnums(state) {
		return errProjectInvalidArgument
	}
	return nil
}

func validProjectListQueryStateEnums(state projectListQueryState) bool {
	if !validProjectListStaticEnums(state) {
		return false
	}
	return validProjectListGeneratedEnums(state)
}

func validProjectListStaticEnums(state projectListQueryState) bool {
	return (state.ApplicationType == nil || *state.ApplicationType == "compose") && (state.Provider == nil || *state.Provider == "docker")
}

func validProjectListGeneratedEnums(state projectListQueryState) bool {
	if state.SourceKind != nil && !generated.ProjectSourceKind(*state.SourceKind).Valid() {
		return false
	}
	if state.RuntimeStatus != nil && !generated.ProjectRuntimeStatus(*state.RuntimeStatus).Valid() {
		return false
	}
	if state.DriftStatus != nil && !generated.ProjectDriftStatus(*state.DriftStatus).Valid() {
		return false
	}
	return true
}

func validProjectListQueryStateStrings(state projectListQueryState) bool {
	for _, value := range []*string{state.Keyword, state.ApplicationType, state.Provider, state.SourceKind, state.RuntimeStatus, state.DriftStatus} {
		if value != nil && strings.TrimSpace(*value) == "" {
			return false
		}
	}
	return true
}

func validProjectListQueryStateTarget(state projectListQueryState) bool {
	return state.RuntimeTargetID == nil || *state.RuntimeTargetID > 0
}

// validateProjectListVisibleColumns 验证项目列表已保存视图的可见列是否均受支持。
// 如果所有列均有效则返回 nil，否则返回 errProjectInvalidArgument。
func validateProjectListVisibleColumns(columns []string) error {
	for _, column := range columns {
		if _, ok := projectListSavedViewColumns[strings.TrimSpace(column)]; !ok {
			return errProjectInvalidArgument
		}
	}
	return nil
}

// mapSavedViewError 将已保存视图错误映射为项目域错误。
func mapSavedViewError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, moduleapi.ErrSavedViewConflict):
		return errProjectConflict
	case errors.Is(err, moduleapi.ErrSavedViewNotFound):
		return errProjectNotFound
	default:
		return err
	}
}

// projectSavedViewRequestFromGenerated converts a generated saved-view request into the internal request format.
// It serializes the query state and copies the visible columns. It returns an invalid-argument error if serialization fails.
func projectSavedViewRequestFromGenerated(request generated.ProjectSavedViewRequest) (savedViewRequest, error) {
	queryState, err := json.Marshal(request.QueryState)
	if err != nil {
		return savedViewRequest{}, errProjectInvalidArgument
	}
	return savedViewRequest{Name: request.Name, QueryState: queryState, PageSize: request.PageSize, VisibleColumns: append([]string(nil), request.VisibleColumns...)}, nil
}

// toGeneratedProjectSavedView 将已保存视图转换为生成的项目视图模型。
// 如果视图 ID 无效或查询状态不是合法 JSON，则返回参数错误。
func toGeneratedProjectSavedView(view moduleapi.SavedView) (generated.ProjectSavedView, error) {
	if view.ID == 0 || view.ID > uint64(^uint64(0)>>1) {
		return generated.ProjectSavedView{}, errProjectInvalidArgument
	}
	queryState := make(map[string]interface{})
	if err := json.Unmarshal(view.QueryState, &queryState); err != nil {
		return generated.ProjectSavedView{}, errProjectInvalidArgument
	}
	return generated.ProjectSavedView{Id: int64(view.ID), Name: view.Name, QueryState: queryState, PageSize: view.PageSize, VisibleColumns: append([]string(nil), view.VisibleColumns...), CreatedAt: view.CreatedAt, UpdatedAt: view.UpdatedAt}, nil
}
