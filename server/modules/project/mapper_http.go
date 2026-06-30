package project

import (
	generated "graft/server/internal/contract/openapi/generated"
	projectcontract "graft/server/modules/project/contract"
)

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

func toManagedRootResponse(info ManagedRootInfo) generated.ProjectManagedRootResponse {
	response := generated.ProjectManagedRootResponse{
		Status:                generated.ProjectManagedRootStatus(info.Status),
		ConfigKey:             info.ConfigKey,
		OwnershipMode:         generated.ProjectOwnershipMode(info.OwnershipMode),
		CreatePermission:      info.CreatePermission,
		SupportsManagedCreate: info.SupportsManagedCreate,
	}
	if info.ConfiguredRootDirectory != nil {
		response.ConfiguredRootDirectory = info.ConfiguredRootDirectory
	}
	if info.StatusReason != nil {
		response.StatusReason = info.StatusReason
	}
	return response
}

func toManagedCreateValidateResponse(result ManagedProjectCreateValidationResult) generated.ProjectCreateValidateResponse {
	response := generated.ProjectCreateValidateResponse{
		ManagedRoot:             toManagedRootResponse(result.ManagedRoot),
		DisplayName:             result.DisplayName,
		CanonicalProjectName:    result.CanonicalProjectName,
		OwnershipMode:           generated.ProjectOwnershipMode(result.OwnershipMode),
		WorkingDirectory:        result.WorkingDirectory,
		ComposeFileName:         result.ComposeFileName,
		ComposeFileAbsolutePath: result.ComposeFileAbsolutePath,
	}
	if result.EnvFileName != nil {
		response.EnvFileName = result.EnvFileName
	}
	if result.EnvFileAbsolutePath != nil {
		response.EnvFileAbsolutePath = result.EnvFileAbsolutePath
	}
	if len(result.Warnings) > 0 {
		warnings := append([]string(nil), result.Warnings...)
		response.Warnings = &warnings
	}
	return response
}

func toManagedCreateResponse(result ManagedProjectCreateValidationResult) generated.ProjectCreateResponse {
	response := generated.ProjectCreateResponse{
		ManagedRoot:          toManagedRootResponse(result.ManagedRoot),
		DisplayName:          result.DisplayName,
		CanonicalProjectName: result.CanonicalProjectName,
		OwnershipMode:        generated.ProjectOwnershipMode(result.OwnershipMode),
		WorkingDirectory:     result.WorkingDirectory,
		ComposeFileName:      result.ComposeFileName,
		Action:               generated.ProjectCreateResponseAction("create"),
		Result:               generated.ProjectCreateResponseResultAccepted,
		MessageKey:           optionalString(projectcontract.ProjectManagedCreateAccepted.String()),
		Message:              optionalString(projectcontract.ProjectManagedCreateAccepted.String()),
	}
	if result.EnvFileName != nil {
		response.EnvFileName = result.EnvFileName
	}
	if result.EnvFileAbsolutePath != nil {
		response.EnvFileAbsolutePath = result.EnvFileAbsolutePath
	}
	if len(result.Warnings) > 0 {
		warnings := append([]string(nil), result.Warnings...)
		response.Warnings = &warnings
	}
	return response
}

func toManagedCreateRequest(request generated.PostProjectCreateValidateJSONRequestBody) ManagedProjectCreateRequest {
	var envFileName *string
	if request.EnvFileName != nil {
		value := *request.EnvFileName
		envFileName = &value
	}
	return ManagedProjectCreateRequest{
		DisplayName:              request.DisplayName,
		CanonicalProjectName:     request.CanonicalProjectName,
		RelativeProjectDirectory: request.RelativeProjectDirectory,
		ComposeFileName:          request.ComposeFileName,
		EnvFileName:              envFileName,
	}
}

func optionalStringSlice(items []string) *[]string {
	if len(items) == 0 {
		return nil
	}
	copyItems := append([]string(nil), items...)
	return &copyItems
}
