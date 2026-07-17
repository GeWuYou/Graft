package project

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/contract/openapi/generated"
	"graft/server/internal/httpx"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

type templateDraftHTTPBody struct {
	DisplayName             string                    `json:"display_name"`
	Description             string                    `json:"description"`
	Category                string                    `json:"category"`
	DeploymentAdapterKind   string                    `json:"deployment_adapter_kind"`
	DefinitionSchemaVersion int                       `json:"definition_schema_version"`
	Definition              jsonRawTemplateDefinition `json:"definition"`
}

// jsonRawTemplateDefinition 在 HTTP 边界保留由部署适配器拥有的原始 JSON。
type jsonRawTemplateDefinition []byte

func (value *jsonRawTemplateDefinition) UnmarshalJSON(raw []byte) error {
	*value = append((*value)[:0], raw...)
	return nil
}
func (value jsonRawTemplateDefinition) MarshalJSON() ([]byte, error) { return value, nil }

type cloneTemplateHTTPBody struct {
	DisplayName string `json:"display_name"`
}

func (r routeRuntime) handlePublishedTemplates(ginCtx *gin.Context) {
	kind := projectcontract.DeploymentAdapterKind(strings.TrimSpace(ginCtx.Query("deployment_adapter_kind")))
	if kind == "" {
		kind = projectcontract.DeploymentAdapterKindCompose
	}
	page, pageSize, err := templateCatalogPagination(ginCtx)
	if err != nil {
		r.writeInvalidArgumentError(ginCtx)
		return
	}
	result, err := r.service.ListPublishedApplicationTemplates(ginCtx.Request.Context(), kind, projectstore.TemplateCatalogQuery{Search: ginCtx.Query("q"), Category: ginCtx.Query("category"), Sort: ginCtx.Query("sort"), Page: page, PageSize: pageSize})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, generated.ApplicationTemplateCatalogListResponse{Items: templateCatalogItemsHTTP(result.Items), Page: page, PageSize: pageSize, HasMore: result.HasMore})
}

func (r routeRuntime) handlePublishedTemplateDetail(ginCtx *gin.Context) {
	item, err := r.service.GetPublishedApplicationTemplate(ginCtx.Request.Context(), ginCtx.Param("templateId"))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, templateAggregateHTTP(item))
}

func (r routeRuntime) handlePublishedTemplateVersion(ginCtx *gin.Context) {
	item, err := r.service.GetPublishedApplicationTemplateVersion(ginCtx.Request.Context(), ginCtx.Param("templateVersionId"))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, templateAggregateHTTP(item))
}

func (r routeRuntime) handleManagedTemplates(ginCtx *gin.Context) {
	items, err := r.service.ListApplicationTemplates(ginCtx.Request.Context(), true)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, generated.ApplicationTemplateListResponse{Items: templateAggregatesHTTP(items)})
}
func (r routeRuntime) handleTemplateDetail(ginCtx *gin.Context) {
	item, err := r.service.GetApplicationTemplate(ginCtx.Request.Context(), ginCtx.Param("templateId"))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, templateAggregateHTTP(item))
}
func (r routeRuntime) handleCreateTemplateDraft(ginCtx *gin.Context) {
	var body templateDraftHTTPBody
	if !bindJSON(ginCtx, r.ctx, &body) {
		return
	}
	item, err := r.service.CreateApplicationTemplateDraft(ginCtx.Request.Context(), templateDraftRequestFromHTTP(body), currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusCreated, templateAggregateHTTP(item))
}
func (r routeRuntime) handleUpdateTemplateDraft(ginCtx *gin.Context) {
	var body templateDraftHTTPBody
	if !bindJSON(ginCtx, r.ctx, &body) {
		return
	}
	item, err := r.service.UpdateApplicationTemplateDraft(ginCtx.Request.Context(), ginCtx.Param("templateId"), templateDraftRequestFromHTTP(body), currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, templateAggregateHTTP(item))
}
func (r routeRuntime) handleCloneTemplate(ginCtx *gin.Context) {
	var body cloneTemplateHTTPBody
	if !bindJSON(ginCtx, r.ctx, &body) {
		return
	}
	item, err := r.service.CloneApplicationTemplate(ginCtx.Request.Context(), ginCtx.Param("templateId"), body.DisplayName, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusCreated, templateAggregateHTTP(item))
}
func (r routeRuntime) handlePublishTemplateDraft(ginCtx *gin.Context) {
	item, err := r.service.PublishApplicationTemplateDraft(ginCtx.Request.Context(), ginCtx.Param("templateId"), currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, templateAggregateHTTP(item))
}
func (r routeRuntime) handleArchiveTemplate(ginCtx *gin.Context) {
	if err := r.service.ArchiveApplicationTemplate(ginCtx.Request.Context(), ginCtx.Param("templateId"), currentUserIDPointer(ginCtx)); err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	ginCtx.Status(http.StatusNoContent)
}
func (r routeRuntime) handleWithdrawTemplate(ginCtx *gin.Context) {
	item, err := r.service.WithdrawApplicationTemplate(ginCtx.Request.Context(), ginCtx.Param("templateId"), currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, templateAggregateHTTP(item))
}
func (r routeRuntime) handleDeleteTemplate(ginCtx *gin.Context) {
	if err := r.service.DeleteApplicationTemplate(ginCtx.Request.Context(), ginCtx.Param("templateId"), currentUserIDPointer(ginCtx)); err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	ginCtx.Status(http.StatusNoContent)
}
func templateDraftRequestFromHTTP(body templateDraftHTTPBody) ApplicationTemplateDraftRequest {
	return ApplicationTemplateDraftRequest{DisplayName: body.DisplayName, Description: body.Description, Category: projectcontract.ApplicationTemplateCategory(body.Category), DeploymentAdapterKind: projectcontract.DeploymentAdapterKind(body.DeploymentAdapterKind), DefinitionSchemaVersion: body.DefinitionSchemaVersion, DefinitionJSON: append([]byte(nil), body.Definition...)}
}

func templateCatalogPagination(ginCtx *gin.Context) (int, int, error) {
	page, pageSize := 1, 24
	var err error
	if raw := ginCtx.Query("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page < 1 {
			return 0, 0, errProjectInvalidArgument
		}
	}
	if raw := ginCtx.Query("page_size"); raw != "" {
		pageSize, err = strconv.Atoi(raw)
		if err != nil || pageSize < 1 || pageSize > 100 {
			return 0, 0, errProjectInvalidArgument
		}
	}
	return page, pageSize, nil
}
func templateAggregatesHTTP(items []projectstore.ApplicationTemplateAggregate) []generated.ApplicationTemplateResponse {
	result := make([]generated.ApplicationTemplateResponse, 0, len(items))
	for _, item := range items {
		result = append(result, templateAggregateHTTP(item))
	}
	return result
}
func templateAggregateHTTP(item projectstore.ApplicationTemplateAggregate) generated.ApplicationTemplateResponse {
	return generated.ApplicationTemplateResponse{
		TemplateId:            item.Template.ID,
		DisplayName:           item.Template.DisplayName,
		Description:           item.Template.Description,
		Category:              generated.ApplicationTemplateCategory(item.Template.Category),
		DeploymentAdapterKind: generated.ApplicationTemplateResponseDeploymentAdapterKind(item.Template.DeploymentAdapterKind),
		UpdatedAt:             item.Template.UpdatedAt,
		ArchivedAt:            item.Template.ArchivedAt,
		Version: generated.ApplicationTemplateVersion{
			TemplateVersionId:       item.Version.ID,
			VersionNumber:           item.Version.VersionNumber,
			Status:                  generated.ApplicationTemplateVersionStatus(item.Version.Status),
			DefinitionSchemaVersion: item.Version.DefinitionSchemaVersion,
			Definition:              templateDefinitionHTTP(item.Version.DefinitionJSON),
			PublishedAt:             item.Version.PublishedAt,
			PublishedBy:             templateUserIDHTTP(item.Version.PublishedBy),
			WithdrawnAt:             item.Version.WithdrawnAt,
			WithdrawnBy:             templateUserIDHTTP(item.Version.WithdrawnBy),
		},
	}
}

func templateCatalogItemsHTTP(items []projectstore.ApplicationTemplateCatalogItem) []generated.ApplicationTemplateCatalogItem {
	result := make([]generated.ApplicationTemplateCatalogItem, 0, len(items))
	for _, item := range items {
		response := generated.ApplicationTemplateCatalogItem{
			TemplateId:            item.TemplateID,
			DisplayName:           item.DisplayName,
			Description:           item.Description,
			Category:              generated.ApplicationTemplateCategory(item.Category),
			DeploymentAdapterKind: generated.ApplicationTemplateCatalogItemDeploymentAdapterKind(item.DeploymentAdapterKind),
			UpdatedAt:             item.UpdatedAt,
		}
		response.Version.TemplateVersionId = item.TemplateVersionID
		response.Version.VersionNumber = item.VersionNumber
		response.Version.PublishedAt = item.PublishedAt
		result = append(result, response)
	}
	return result
}

func templateDefinitionHTTP(raw []byte) map[string]interface{} {
	var definition map[string]interface{}
	if err := json.Unmarshal(raw, &definition); err != nil || definition == nil {
		return map[string]interface{}{}
	}
	return definition
}

func templateUserIDHTTP(value *uint64) *int64 {
	const maxInt64 = uint64(1<<63 - 1)
	if value == nil || *value > maxInt64 {
		return nil
	}
	result := int64(*value)
	return &result
}
