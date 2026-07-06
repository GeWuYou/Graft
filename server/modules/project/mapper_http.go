package project

import (
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
)

// toProjectListResponse 将列表结果转换为项目列表响应，并复制分页信息和条目集合。
func toProjectListResponse(result ListResult) generated.ProjectListResponse {
	return generated.ProjectListResponse{
		Items:  result.Items,
		Limit:  result.Limit,
		Offset: result.Offset,
		Total:  result.Total,
	}
}

func toRuntimeImportCandidatesResponse(result RuntimeImportCandidatesResult) generated.ProjectImportRuntimeCandidatesResponse {
	items := make([]generated.ProjectImportRuntimeCandidate, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, generated.ProjectImportRuntimeCandidate{
			CandidateKey:           item.CandidateKey,
			CanonicalProjectName:   item.CanonicalProjectName,
			Status:                 generated.ProjectImportRuntimeCandidateStatus(item.Status),
			StatusReasonCodes:      append([]string(nil), item.StatusReasonCodes...),
			Importable:             item.Importable,
			RuntimeType:            item.RuntimeType,
			RuntimeVersion:         item.RuntimeVersion,
			WorkingDirectory:       item.WorkingDirectory,
			WorkingDirectorySource: generated.ProjectImportRuntimeWorkingDirectorySource(item.WorkingDirectorySource),
			ConfigFiles:            append([]string(nil), item.ConfigFiles...),
			ServiceNames:           append([]string(nil), item.ServiceNames...),
			ContainerCounts: generated.ProjectContainerCounts{
				Running:       item.ContainerCounts.Running,
				Stopped:       item.ContainerCounts.Stopped,
				Transitioning: 0,
				Issue:         0,
				Total:         item.ContainerCounts.Total,
			},
			Warnings: append([]string(nil), item.Warnings...),
		})
	}
	return generated.ProjectImportRuntimeCandidatesResponse{
		Items:  items,
		Total:  result.Total,
		Limit:  result.Limit,
		Offset: result.Offset,
		FilterCounts: generated.ProjectImportRuntimeCandidateFilterCounts{
			All:         result.FilterCounts.All,
			Ready:       result.FilterCounts.Ready,
			Imported:    result.FilterCounts.Imported,
			Unavailable: result.FilterCounts.Unavailable,
		},
	}
}

func toRuntimeImportMembers(items []RuntimeImportMember) []generated.ProjectImportRuntimeMember {
	members := make([]generated.ProjectImportRuntimeMember, 0, len(items))
	for _, item := range items {
		members = append(members, generated.ProjectImportRuntimeMember{
			ContainerId:   item.ContainerID,
			ContainerName: item.ContainerName,
			ServiceName:   item.ServiceName,
			State:         item.State,
		})
	}
	return members
}

// toSourceCatalogResponse 将源目录结果转换为源目录响应，并复制条目列表。
//
// @return Items 复制后的源目录条目列表。
func toSourceCatalogResponse(result SourceCatalogResult) generated.ProjectSourceCatalogResponse {
	return generated.ProjectSourceCatalogResponse{
		Items: append([]generated.ProjectSourceEntry(nil), result.Items...),
	}
}

// toDiscoveryCandidatesResponse 将内部发现候选结果转换为 OpenAPI 响应。
// 它会复制候选项及其切片字段，并在对应值存在时写入可选的来源信息、状态原因和结果级字段。
func toDiscoveryCandidatesResponse(result DiscoveryCandidatesResult) generated.ProjectDiscoveryCandidatesResponse {
	items := make([]generated.ProjectDiscoveryCandidate, 0, len(result.Items))
	for _, item := range result.Items {
		candidate := generated.ProjectDiscoveryCandidate{
			CandidateKey:               item.CandidateKey,
			CandidateKind:              generated.ProjectDiscoveryCandidateKind(item.CandidateKind),
			SourceKind:                 generated.ProjectSourceKind(item.SourceKind),
			DisplayName:                item.DisplayName,
			CanonicalProjectName:       item.CanonicalProjectName,
			CanonicalProjectNameSource: generated.ProjectCanonicalNameSource(item.CanonicalProjectNameSource),
			WorkingDirectory:           item.WorkingDirectory,
			OwnershipMode:              generated.ProjectOwnershipMode(item.OwnershipMode),
			HostScope:                  generated.ProjectHostScope(item.HostScope),
			Status:                     generated.ProjectDiscoveryCandidateStatus(item.Status),
			RecommendedAction:          generated.ProjectDiscoveryCandidateRecommendedAction(item.RecommendedAction),
			ComposeFiles:               append([]generated.ProjectFileItem(nil), item.ComposeFiles...),
			EnvFiles:                   append([]generated.ProjectFileItem(nil), item.EnvFiles...),
			DeclaredServiceNames:       append([]string(nil), item.DeclaredServiceNames...),
			ServiceCount:               item.ServiceCount,
			ConfigHash:                 item.ConfigHash,
			Warnings:                   append([]string(nil), item.Warnings...),
			Conflicts:                  append([]string(nil), item.Conflicts...),
		}
		if metadata := toGeneratedSourceMetadata(item.SourceMetadata); metadata != nil {
			candidate.SourceMetadata = metadata
		}
		if item.SourceType != "" {
			sourceType := generated.ProjectSourceEntryType(item.SourceType)
			candidate.SourceType = &sourceType
		}
		if item.StatusReason != nil {
			candidate.StatusReason = item.StatusReason
		}
		items = append(items, candidate)
	}
	response := generated.ProjectDiscoveryCandidatesResponse{
		SourceType:            generated.ProjectSourceEntryType(result.SourceType),
		SupportsScan:          result.SupportsScan,
		SupportsAutoDiscovery: result.SupportsAutoDiscovery,
		Items:                 items,
	}
	if result.AuthorityRoot != nil {
		response.AuthorityRoot = result.AuthorityRoot
	}
	if result.StatusReason != nil {
		response.StatusReason = result.StatusReason
	}
	return response
}

// toImportValidateResponse 将导入校验结果转换为 OpenAPI 响应，并在存在配置哈希或声明的服务名时附带归一化预览摘要。
// 该响应会复制冲突和告警列表，以避免与输入数据共享底层切片。
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

func toRuntimeImportInspectResponse(result RuntimeImportInspectResult) generated.ProjectImportRuntimeInspectResponse {
	return generated.ProjectImportRuntimeInspectResponse{
		InspectionId:               result.InspectionID,
		CandidateKey:               result.CandidateKey,
		ResolvedWorkingDirectory:   result.ResolvedWorkingDirectory,
		CanonicalProjectName:       result.CanonicalProjectName,
		CanonicalProjectNameSource: generated.ProjectCanonicalNameSource(result.CanonicalProjectNameSource),
		DisplayNameSuggested:       result.DisplayNameSuggested,
		ComposeFiles:               toGeneratedProjectFiles(result.ComposeFiles),
		EnvFiles:                   toGeneratedProjectFiles(result.EnvFiles),
		Services:                   append([]string(nil), result.ServiceNames...),
		Networks:                   toRuntimeImportNetworkResources(result.NetworkResources),
		Volumes:                    toRuntimeImportVolumeResources(result.VolumeResources),
		RuntimeMembers:             toRuntimeImportMembers(result.RuntimeMembers),
		ConfigHash:                 result.ConfigHash,
		Warnings:                   append([]string(nil), result.Warnings...),
		Conflicts:                  append([]string(nil), result.Conflicts...),
		ValidationStatus:           generated.ProjectImportRuntimeInspectResponseValidationStatus(result.ValidationStatus),
	}
}

func toRuntimeImportNetworkResources(items []RuntimeImportNetworkResource) []generated.ProjectImportRuntimeNetworkResource {
	resources := make([]generated.ProjectImportRuntimeNetworkResource, 0, len(items))
	for _, item := range items {
		resources = append(resources, generated.ProjectImportRuntimeNetworkResource{
			Name:           item.Name,
			Driver:         item.Driver,
			Scope:          item.Scope,
			Internal:       item.Internal,
			Containers:     append([]string(nil), item.Containers...),
			ContainerCount: item.ContainerCount,
			Services:       append([]string(nil), item.Services...),
			ServiceCount:   item.ServiceCount,
		})
	}
	return resources
}

func toRuntimeImportVolumeResources(items []RuntimeImportVolumeResource) []generated.ProjectImportRuntimeVolumeResource {
	resources := make([]generated.ProjectImportRuntimeVolumeResource, 0, len(items))
	for _, item := range items {
		resources = append(resources, generated.ProjectImportRuntimeVolumeResource{
			Name:           item.Name,
			Driver:         item.Driver,
			Anonymous:      item.Anonymous,
			MountTarget:    item.MountTarget,
			MountedBy:      append([]string(nil), item.MountedBy...),
			Containers:     append([]string(nil), item.Containers...),
			ContainerCount: item.ContainerCount,
		})
	}
	return resources
}

func toGeneratedProjectFiles(items []FileView) []generated.ProjectImportInspectFileItem {
	files := make([]generated.ProjectImportInspectFileItem, 0, len(items))
	for _, item := range items {
		files = append(files, generated.ProjectImportInspectFileItem{
			AbsolutePath:        item.AbsolutePath,
			DisplayPath:         item.DisplayPath,
			ExistsOnLastRefresh: item.ExistsOnLastRefresh,
			Kind:                generated.ProjectFileKind(item.Kind),
			LastObservedHash:    item.LastObservedHash,
			OrderIndex:          item.OrderIndex,
			Role:                generated.ProjectFileRole(item.Role),
		})
	}
	return files
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
		items := toGeneratedGuardResults(result.GuardResults)
		response.GuardResults = &items
	}
	return response
}

func toLifecycleConfigurationRequest(request generated.ProjectLifecycleConfigurationRequest) LifecycleStandardConfig {
	return LifecycleStandardConfig{
		Profiles:                 append([]string(nil), request.Profiles...),
		DownBeforeRedeploy:       request.DownBeforeRedeploy,
		PullBeforeRedeploy:       request.PullBeforeRedeploy,
		BuildBeforeUp:            request.BuildBeforeUp,
		ForceRecreate:            request.ForceRecreate,
		WaitAfterUp:              request.WaitAfterUp,
		PruneImagesAfterRedeploy: request.PruneImagesAfterRedeploy,
	}
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
		items := toGeneratedGuardResults(result.GuardResults)
		response.GuardResults = &items
	}
	return response
}

func toBatchActionResponse(result BatchActionResult) generated.ProjectBatchActionResponse {
	items := make([]generated.ProjectBatchActionItem, 0, len(result.Items))
	for _, item := range result.Items {
		mapped := generated.ProjectBatchActionItem{
			ProjectId: mustGeneratedID(item.ProjectID),
			Action:    generated.ProjectBatchActionItemAction(item.Action),
			Result:    generated.ProjectBatchActionItemResult(item.Result),
			Skipped:   item.Skipped,
		}
		if item.MessageKey != nil {
			mapped.MessageKey = item.MessageKey
		}
		if item.Message != nil {
			mapped.Message = item.Message
		}
		if len(item.GuardResults) > 0 {
			guards := toGeneratedGuardResults(item.GuardResults)
			mapped.GuardResults = &guards
		}
		items = append(items, mapped)
	}
	return generated.ProjectBatchActionResponse{
		TotalCount:     result.TotalCount,
		CompletedCount: result.CompletedCount,
		BlockedCount:   result.BlockedCount,
		SkippedCount:   result.SkippedCount,
		Items:          items,
	}
}

func toGeneratedGuardResults(items []GuardResult) []generated.ProjectGuardResult {
	result := make([]generated.ProjectGuardResult, 0, len(items))
	for _, item := range items {
		mapped := generated.ProjectGuardResult{Code: item.Code}
		if item.MessageKey != nil {
			mapped.MessageKey = item.MessageKey
		}
		if item.Detail != nil {
			mapped.Detail = item.Detail
		}
		result = append(result, mapped)
	}
	return result
}

// toManagedRootResponse 将托管根信息转换为项目托管根响应。
//
// toManagedRootResponse 将托管根目录信息映射为 OpenAPI 响应，并在可用时附加可配置根目录和状态原因。
func toManagedRootResponse(info ManagedRootInfo) generated.ProjectManagedRootResponse {
	response := generated.ProjectManagedRootResponse{
		SourceType:            generated.ProjectSourceEntryType(info.SourceType),
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
// toManagedCreateValidateResponse 将托管项目创建校验结果转换为生成的创建校验响应。
// 它包含托管根信息、源类型、显示名、规范项目名、所有权模式、工作目录以及 compose 文件信息，并在存在时附带环境文件路径、源元数据和警告列表。
func toManagedCreateValidateResponse(result ManagedProjectCreateValidationResult) generated.ProjectCreateValidateResponse {
	response := generated.ProjectCreateValidateResponse{
		ManagedRoot:             toManagedRootResponse(result.ManagedRoot),
		SourceType:              generated.ProjectSourceEntryType(result.SourceType),
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
	if metadata := toGeneratedSourceMetadata(result.SourceMetadata); metadata != nil {
		response.SourceMetadata = metadata
	}
	if len(result.Warnings) > 0 {
		warnings := append([]string(nil), result.Warnings...)
		response.Warnings = &warnings
	}
	return response
}

// toManagedCreateResponse 将托管项目创建结果转换为创建响应。
// toManagedCreateResponse 将托管项目创建结果转换为创建响应，包含托管根状态、项目信息、快照摘要以及可选的环境文件信息、源元数据和警告。
func toManagedCreateResponse(result ManagedProjectCreateResult) generated.ProjectCreateResponse {
	response := generated.ProjectCreateResponse{
		ManagedRoot:             toManagedRootResponse(result.Validation.ManagedRoot),
		SourceType:              generated.ProjectSourceEntryType(result.SourceType),
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
	if metadata := toGeneratedSourceMetadata(result.Validation.SourceMetadata); metadata != nil {
		response.SourceMetadata = metadata
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

func toProjectOverviewServiceItem(
	item projectcompose.ServiceProjection,
	runtime moduleapi.ContainerProjectServiceResourceSummary,
) (generated.ProjectOverviewServiceItem, bool) {
	status := projectOverviewServiceStatus(runtime)
	health := projectOverviewServiceHealth(runtime)
	return generated.ProjectOverviewServiceItem{
		ServiceName:                  item.ServiceName,
		Image:                        item.Image,
		Status:                       generated.ProjectOverviewServiceItemStatus(status),
		Health:                       generated.ProjectOverviewServiceItemHealth(health),
		ContainerCount:               runtime.ContainerCount,
		RunningCount:                 runtime.RunningCount,
		StoppedCount:                 runtime.StoppedCount,
		TransitioningCount:           runtime.TransitioningCount,
		IssueCount:                   runtime.IssueCount,
		HealthyContainerCount:        runtime.HealthyContainerCount,
		UnhealthyContainerCount:      runtime.UnhealthyContainerCount,
		StartingContainerCount:       runtime.StartingContainerCount,
		RestartCount:                 runtime.RestartCount,
		StatsAvailable:               runtime.StatsAvailable,
		StatsAvailableContainerCount: runtime.StatsAvailableContainerCount,
		CpuPercent:                   runtime.CPUPercent,
		MemoryUsageBytes:             runtime.MemoryUsageBytes,
		MemoryLimitBytes:             runtime.MemoryLimitBytes,
	}, projectOverviewServiceIsHealthy(runtime)
}

func projectOverviewServiceStatus(runtime moduleapi.ContainerProjectServiceResourceSummary) string {
	if runtime.UnhealthyContainerCount > 0 || runtime.TransitioningCount > 0 || runtime.IssueCount > 0 {
		return "degraded"
	}
	if runtime.RunningCount > 0 {
		return "running"
	}
	return "stopped"
}

func projectOverviewServiceHealth(runtime moduleapi.ContainerProjectServiceResourceSummary) string {
	if runtime.UnhealthyContainerCount > 0 || runtime.StartingContainerCount > 0 || runtime.TransitioningCount > 0 || runtime.IssueCount > 0 {
		return "attention"
	}
	if runtime.RunningCount > 0 {
		return "healthy"
	}
	return "unknown"
}

func projectOverviewServiceIsHealthy(runtime moduleapi.ContainerProjectServiceResourceSummary) bool {
	return strings.EqualFold(projectOverviewServiceHealth(runtime), "healthy")
}

func countDeclaredNetworks(items []projectcompose.ServiceProjection) int {
	names := make(map[string]struct{})
	for _, item := range items {
		for _, network := range item.DeclaredNetworks {
			network = strings.TrimSpace(network)
			if network == "" {
				continue
			}
			names[network] = struct{}{}
		}
	}
	return len(names)
}

func countDeclaredVolumes(items []projectcompose.ServiceProjection) int {
	names := make(map[string]struct{})
	for _, item := range items {
		for _, volume := range item.DeclaredVolumes {
			volume = strings.TrimSpace(volume)
			if volume == "" {
				continue
			}
			names[volume] = struct{}{}
		}
	}
	return len(names)
}

func optionalRFC3339Time(value string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return nil
	}
	return &parsed
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
