package project

import (
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	projectcontract "graft/server/modules/project/contract"
)

// toProjectListResponse 将 ListResult 映射为 ProjectListResponse。
func toProjectListResponse(result ListResult) generated.ProjectListResponse {
	return generated.ProjectListResponse{
		Items:  result.Items,
		Limit:  result.Limit,
		Offset: result.Offset,
		Total:  result.Total,
	}
}

// 当配置哈希或声明的服务名存在时，会附带归一化预览摘要。
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

// toConfigurationMetadataResponse 将配置元数据结果转换为 OpenAPI 的项目配置元数据响应。
// 当诊断摘要非空时，会复制后作为可选字段返回。
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

// toConfigurationPreviewResponse 将配置预览结果转换为项目配置预览响应。
//
// ProjectId 通过 mustGeneratedID 转换，其余字段按原样复制。
func toConfigurationPreviewResponse(result ConfigurationPreviewResult) generated.ProjectConfigurationPreviewResponse {
	return generated.ProjectConfigurationPreviewResponse{
		ProjectId:             mustGeneratedID(result.ProjectID),
		CanonicalProjectName:  result.CanonicalProjectName,
		ConfigHash:            result.ConfigHash,
		NormalizedComposeYaml: result.NormalizedComposeYAML,
		RefreshedAt:           result.RefreshedAt,
	}
}

// toConfigurationFileResponse 返回配置文件响应，包含文件标识、类型、路径、内容和下载名称，并固定为 UTF-8 编码且只读。
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

// toConfigurationDiffRequest 将配置差异请求转换为内部的 ConfigurationDraft。
// 它复制 ComposeFileContent，并在 EnvFileContent 存在时生成其独立副本。
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

// toConfigurationDiffResponse 将配置差异结果转换为项目配置差异响应。
// 返回的响应包含项目 ID、项目名、所有权模式、当前和 प्रस्ताव定配置哈希、变更标记以及差异文件列表；当存在警告时，会一并返回复制后的警告列表。
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

// toConfigurationValidateRequest 将配置校验请求转换为内部的 ConfigurationDraft。
//
// 它会复制组合文件内容，并在请求包含环境文件内容时创建新的字符串指针。
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

// toConfigurationValidateResponse 将配置校验结果转换为项目配置校验响应。
// 返回包含项目 ID、规范化项目名、所有权模式、建议配置哈希、规范化 Compose YAML 和声明的服务名称的响应；当存在警告时，还会附加警告列表。
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

// toDeployRequest 将部署请求转换为配置草稿，并在存在时复制环境文件内容。
//
// 返回包含请求中的 `ComposeFileContent` 和可选 `EnvFileContent` 的 `ConfigurationDraft`。
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

// toDeployResponse 将部署结果映射为项目部署响应，保留可选消息、守卫结果和声明服务数等字段。
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

// toActionResponse 将动作结果转换为项目动作响应，并在需要时包含消息键、消息和守卫结果。
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

// toManagedRootResponse 将托管根信息转换为项目托管根响应。
//
// 当可配置根目录或状态原因存在时，会将其一并写入响应。
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

// toManagedCreateValidateResponse 将托管项目创建校验结果映射为创建校验响应。
// 它会保留托管根信息、显示名、规范名、所有权模式、工作目录以及 compose 文件相关路径，并在有内容时附带环境文件字段和警告列表。
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

// toManagedCreateResponse 将托管项目创建结果转换为创建响应。
// 返回创建后的项目信息、托管根状态、快照摘要以及可选的环境文件信息和警告。
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

// toManagedCreateRequest 将项目创建校验请求转换为内部创建请求。
// 它复制显示名称、规范项目名、相对目录、Compose 文件名，并在提供环境文件名时创建独立副本。
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

// toManagedCreateExecuteRequest 将项目创建执行请求转换为内部创建请求。
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

// optionalStringSlice 在切片非空时返回其拷贝指针。
//
// @param items 要包装的字符串切片。
// @returns 切片为空时返回 nil；否则返回一个包含原始内容拷贝的字符串切片指针。
func optionalStringSlice(items []string) *[]string {
	if len(items) == 0 {
		return nil
	}
	copyItems := append([]string(nil), items...)
	return &copyItems
}
