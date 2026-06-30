package project

import (
	"time"

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

func toConfigurationDiffRequest(request generated.ProjectConfigurationDiffRequest) ConfigurationDraft {
	var envFileContent *string
	if request.EnvFileContent != nil {
		value := *request.EnvFileContent
		envFileContent = &value
	}
	return ConfigurationDraft{
		ComposeFileContent: request.ComposeFileContent,
		EnvFileContent:     envFileContent,
	}
}

func toConfigurationDiffResponse(result ConfigurationDiffResult) generated.ProjectConfigurationDiffResponse {
	files := make([]generated.ProjectConfigurationDiffFile, 0, len(result.Files))
	for _, item := range result.Files {
		files = append(files, generated.ProjectConfigurationDiffFile{
			Kind:            generated.ProjectFileKind(item.Kind),
			Path:            item.Path,
			Changed:         item.Changed,
			CurrentHash:     item.CurrentHash,
			ProposedHash:    item.ProposedHash,
			CurrentContent:  item.CurrentContent,
			ProposedContent: item.ProposedContent,
		})
	}
	response := generated.ProjectConfigurationDiffResponse{
		ProjectId:            mustGeneratedID(result.ProjectID),
		CanonicalProjectName: result.CanonicalProjectName,
		OwnershipMode:        generated.ProjectOwnershipMode(result.OwnershipMode),
		CurrentConfigHash:    result.CurrentConfigHash,
		ProposedConfigHash:   result.ProposedConfigHash,
		HasChanges:           result.HasChanges,
		Files:                files,
	}
	if len(result.Warnings) > 0 {
		warnings := append([]string(nil), result.Warnings...)
		response.Warnings = &warnings
	}
	return response
}

func toConfigurationValidateRequest(request generated.ProjectConfigurationValidateRequest) ConfigurationDraft {
	var envFileContent *string
	if request.EnvFileContent != nil {
		value := *request.EnvFileContent
		envFileContent = &value
	}
	return ConfigurationDraft{
		ComposeFileContent: request.ComposeFileContent,
		EnvFileContent:     envFileContent,
	}
}

func toConfigurationValidateResponse(result ConfigurationValidateResult) generated.ProjectConfigurationValidateResponse {
	response := generated.ProjectConfigurationValidateResponse{
		ProjectId:             mustGeneratedID(result.ProjectID),
		CanonicalProjectName:  result.CanonicalProjectName,
		OwnershipMode:         generated.ProjectOwnershipMode(result.OwnershipMode),
		ProposedConfigHash:    result.ProposedConfigHash,
		NormalizedComposeYaml: result.NormalizedComposeYAML,
		DeclaredServiceNames:  append([]string(nil), result.DeclaredServiceNames...),
	}
	if len(result.Warnings) > 0 {
		warnings := append([]string(nil), result.Warnings...)
		response.Warnings = &warnings
	}
	return response
}

func toDeployRequest(request generated.ProjectDeployRequest) ConfigurationDraft {
	var envFileContent *string
	if request.EnvFileContent != nil {
		value := *request.EnvFileContent
		envFileContent = &value
	}
	return ConfigurationDraft{
		ComposeFileContent: request.ComposeFileContent,
		EnvFileContent:     envFileContent,
	}
}

func toDeployResponse(result DeployResult) generated.ProjectDeployResponse {
	response := generated.ProjectDeployResponse{
		ProjectId:            mustGeneratedID(result.ProjectID),
		Action:               generated.ProjectDeployResponseAction(result.Action),
		Result:               generated.ProjectDeployResponseResult(result.Result),
		CanonicalProjectName: result.CanonicalProjectName,
		OwnershipMode:        generated.ProjectOwnershipMode(result.OwnershipMode),
		ConfigHash:           result.ConfigHash,
		RefreshedAt:          result.RefreshedAt,
	}
	if result.DeclaredServiceCount >= 0 {
		count := result.DeclaredServiceCount
		response.DeclaredServiceCount = &count
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

func toManagedCreateResponse(result ManagedProjectCreateResult) generated.ProjectCreateResponse {
	response := generated.ProjectCreateResponse{
		ManagedRoot:             toManagedRootResponse(result.Validation.ManagedRoot),
		ProjectId:               mustGeneratedID(result.ProjectID),
		DisplayName:             result.Validation.DisplayName,
		CanonicalProjectName:    result.Validation.CanonicalProjectName,
		OwnershipMode:           generated.ProjectOwnershipMode(result.Validation.OwnershipMode),
		WorkingDirectory:        result.Validation.WorkingDirectory,
		ComposeFileName:         result.Validation.ComposeFileName,
		ComposeFileAbsolutePath: result.Validation.ComposeFileAbsolutePath,
		Action:                  generated.ProjectCreateResponseAction("create"),
		Result:                  generated.ProjectCreateResponseResult("created"),
		MessageKey:              optionalString(projectcontract.ProjectImported.String()),
		Message:                 optionalString(projectcontract.ProjectImported.String()),
		SnapshotSummary: struct {
			ConfigHash           string    `json:"config_hash"`
			DeclaredServiceCount *int      `json:"declared_service_count,omitempty"`
			RefreshedAt          time.Time `json:"refreshed_at"`
		}{
			ConfigHash:  result.ConfigHash,
			RefreshedAt: result.RefreshedAt,
		},
	}
	if result.DeclaredServiceCount >= 0 {
		count := result.DeclaredServiceCount
		response.SnapshotSummary.DeclaredServiceCount = &count
	}
	if result.Validation.EnvFileName != nil {
		response.EnvFileName = result.Validation.EnvFileName
	}
	if result.Validation.EnvFileAbsolutePath != nil {
		response.EnvFileAbsolutePath = result.Validation.EnvFileAbsolutePath
	}
	if len(result.Validation.Warnings) > 0 {
		warnings := append([]string(nil), result.Validation.Warnings...)
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

func toManagedCreateExecuteRequest(request generated.PostProjectCreateJSONRequestBody) ManagedProjectCreateRequest {
	var envFileName *string
	if request.EnvFileName != nil {
		value := *request.EnvFileName
		envFileName = &value
	}
	var envFileContent *string
	if request.EnvFileContent != nil {
		value := *request.EnvFileContent
		envFileContent = &value
	}
	return ManagedProjectCreateRequest{
		DisplayName:              request.DisplayName,
		CanonicalProjectName:     request.CanonicalProjectName,
		RelativeProjectDirectory: request.RelativeProjectDirectory,
		ComposeFileName:          request.ComposeFileName,
		ComposeFileContent:       request.ComposeFileContent,
		EnvFileName:              envFileName,
		EnvFileContent:           envFileContent,
	}
}

func optionalStringSlice(items []string) *[]string {
	if len(items) == 0 {
		return nil
	}
	copyItems := append([]string(nil), items...)
	return &copyItems
}
