package project

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/httpx"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

type templateDraftHTTPBody struct {
	DisplayName             string                    `json:"display_name"`
	Description             string                    `json:"description"`
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
	items, err := r.service.ListPublishedApplicationTemplates(ginCtx.Request.Context(), kind)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, gin.H{"items": templateAggregatesHTTP(items)})
}

func (r routeRuntime) handleManagedTemplates(ginCtx *gin.Context) {
	items, err := r.service.ListApplicationTemplates(ginCtx.Request.Context(), true)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, gin.H{"items": templateAggregatesHTTP(items)})
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
	return ApplicationTemplateDraftRequest{DisplayName: body.DisplayName, Description: body.Description, DeploymentAdapterKind: projectcontract.DeploymentAdapterKind(body.DeploymentAdapterKind), DefinitionSchemaVersion: body.DefinitionSchemaVersion, DefinitionJSON: append([]byte(nil), body.Definition...)}
}
func templateAggregatesHTTP(items []projectstore.ApplicationTemplateAggregate) []gin.H {
	result := make([]gin.H, 0, len(items))
	for _, item := range items {
		result = append(result, templateAggregateHTTP(item))
	}
	return result
}
func templateAggregateHTTP(item projectstore.ApplicationTemplateAggregate) gin.H {
	return gin.H{"template_id": item.Template.ID, "display_name": item.Template.DisplayName, "description": item.Template.Description, "deployment_adapter_kind": item.Template.DeploymentAdapterKind, "archived_at": item.Template.ArchivedAt, "version": gin.H{"template_version_id": item.Version.ID, "version_number": item.Version.VersionNumber, "status": item.Version.Status, "definition_schema_version": item.Version.DefinitionSchemaVersion, "definition": jsonRawTemplateDefinition(item.Version.DefinitionJSON), "published_at": item.Version.PublishedAt, "published_by": item.Version.PublishedBy, "withdrawn_at": item.Version.WithdrawnAt, "withdrawn_by": item.Version.WithdrawnBy}}
}
