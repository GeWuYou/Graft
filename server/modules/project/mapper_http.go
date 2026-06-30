package project

import generated "graft/server/internal/contract/openapi/generated"

func toProjectListResponse(result ListResult) generated.ProjectListResponse {
	return generated.ProjectListResponse{
		Items:  result.Items,
		Limit:  result.Limit,
		Offset: result.Offset,
		Total:  result.Total,
	}
}

func toImportValidateResponse(result ImportValidationResult) generated.ProjectImportValidateResponse {
	response := generated.ProjectImportValidateResponse{
		CanonicalProjectName:       result.CanonicalProjectName,
		CanonicalProjectNameSource: generated.ProjectCanonicalNameSource(result.CanonicalProjectNameSource),
		ComposeFiles:               result.ComposeFiles,
		Conflicts:                  append([]string(nil), result.Conflicts...),
		EnvFiles:                   result.EnvFiles,
		ServiceCount:               result.ServiceCount,
		Warnings:                   append([]string(nil), result.Warnings...),
		WorkingDirectory:           result.WorkingDirectory,
	}
	if result.ConfigHash != "" || len(result.DeclaredServiceNames) > 0 {
		response.NormalizedPreviewSummary = &struct {
			ConfigHash           *string   `json:"config_hash,omitempty"`
			DeclaredServiceNames *[]string `json:"declared_service_names,omitempty"`
		}{
			ConfigHash:           optionalString(result.ConfigHash),
			DeclaredServiceNames: optionalStringSlice(result.DeclaredServiceNames),
		}
	}
	return response
}

func toConfigurationMetadataResponse(result ConfigurationMetadataResult) generated.ProjectConfigurationMetadataResponse {
	response := generated.ProjectConfigurationMetadataResponse{
		ProjectId:         mustGeneratedID(result.ProjectID),
		ComposeFiles:      result.ComposeFiles,
		EnvFiles:          result.EnvFiles,
		OwnershipMode:     generated.ProjectOwnershipMode(result.OwnershipMode),
		DriftStatus:       generated.ProjectDriftStatus(result.DriftStatus),
		LastRefreshStatus: generated.ProjectRefreshStatus(result.LastRefreshStatus),
		LastRefreshAt:     result.LastRefreshAt,
	}
	if len(result.DiagnosticsSummary) > 0 {
		summary := append([]string(nil), result.DiagnosticsSummary...)
		response.DiagnosticsSummary = &summary
	}
	return response
}

func toConfigurationPreviewResponse(result ConfigurationPreviewResult) generated.ProjectConfigurationPreviewResponse {
	return generated.ProjectConfigurationPreviewResponse{
		ProjectId:             mustGeneratedID(result.ProjectID),
		CanonicalProjectName:  result.CanonicalProjectName,
		ConfigHash:            result.ConfigHash,
		NormalizedComposeYaml: result.NormalizedComposeYAML,
		RefreshedAt:           result.RefreshedAt,
	}
}

func toConfigurationFileResponse(result ConfigurationFileResult) generated.ProjectConfigurationFileResponse {
	return generated.ProjectConfigurationFileResponse{
		FileId:       mustGeneratedID(result.FileID),
		Kind:         generated.ProjectFileKind(result.Kind),
		Path:         result.Path,
		Content:      result.Content,
		Encoding:     generated.ProjectConfigurationFileResponseEncoding("utf-8"),
		ReadOnly:     true,
		DownloadName: result.DownloadName,
	}
}

func toActionResponse(result ActionResult) generated.ProjectActionResponse {
	response := generated.ProjectActionResponse{
		ProjectId: mustGeneratedID(result.ProjectID),
		Action:    result.Action,
		Result:    result.Result,
	}
	if result.MessageKey != nil {
		response.MessageKey = result.MessageKey
	}
	if result.Message != nil {
		response.Message = result.Message
	}
	if len(result.GuardResults) > 0 {
		items := append([]string(nil), result.GuardResults...)
		response.GuardResults = &items
	}
	return response
}

func optionalStringSlice(items []string) *[]string {
	if len(items) == 0 {
		return nil
	}
	copyItems := append([]string(nil), items...)
	return &copyItems
}
