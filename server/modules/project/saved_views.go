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
	"name": {}, "source_kind": {}, "runtime_status": {}, "resources": {}, "drift_status": {}, "updated_at": {},
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
	if err := validateProjectListSavedView(request); err != nil {
		return moduleapi.SavedView{}, err
	}
	view, err := s.savedViews.Create(ctx, moduleapi.SavedViewCreateInput{OwnerUserID: ownerUserID, SurfaceKey: projectListSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns})
	return view, mapSavedViewError(err)
}

func (s *Service) updateSavedView(ctx context.Context, ownerUserID, id uint64, request savedViewRequest) (moduleapi.SavedView, error) {
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

func validateProjectListSavedView(request savedViewRequest) error {
	if strings.TrimSpace(request.Name) == "" || request.PageSize < 1 || !json.Valid(request.QueryState) {
		return errProjectInvalidArgument
	}
	if err := validateProjectListQueryState(request.QueryState); err != nil {
		return err
	}
	return validateProjectListVisibleColumns(request.VisibleColumns)
}

func validateProjectListQueryState(queryState json.RawMessage) error {
	var state struct {
		SourceKind  *string `json:"source_kind"`
		DriftStatus *string `json:"drift_status"`
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(queryState, &raw); err != nil || raw == nil {
		return errProjectInvalidArgument
	}
	for key := range raw {
		if key != "source_kind" && key != "drift_status" {
			return errProjectInvalidArgument
		}
	}
	if err := json.Unmarshal(queryState, &state); err != nil {
		return errProjectInvalidArgument
	}
	for _, value := range []*string{state.SourceKind, state.DriftStatus} {
		if value != nil && strings.TrimSpace(*value) == "" {
			return errProjectInvalidArgument
		}
	}
	return nil
}

func validateProjectListVisibleColumns(columns []string) error {
	for _, column := range columns {
		if _, ok := projectListSavedViewColumns[strings.TrimSpace(column)]; !ok {
			return errProjectInvalidArgument
		}
	}
	return nil
}

func mapSavedViewError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, moduleapi.ErrSavedViewConflict):
		return errProjectConflict
	case errors.Is(err, moduleapi.ErrSavedViewNotFound):
		return errProjectNotFound
	default:
		return errProjectInvalidArgument
	}
}

func projectSavedViewRequestFromGenerated(request generated.ProjectSavedViewRequest) (savedViewRequest, error) {
	queryState, err := json.Marshal(request.QueryState)
	if err != nil {
		return savedViewRequest{}, errProjectInvalidArgument
	}
	return savedViewRequest{Name: request.Name, QueryState: queryState, PageSize: request.PageSize, VisibleColumns: append([]string(nil), request.VisibleColumns...)}, nil
}

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
