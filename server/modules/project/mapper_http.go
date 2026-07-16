package project

import (
	"path/filepath"
	"strconv"
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

// toRuntimeImportMembers 将运行时导入成员转换为 API 响应中的成员列表。
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

// toCreationMethodCatalogResponse 将创建方式可用性转换为 OpenAPI 响应。
func toCreationMethodCatalogResponse(result CreationMethodCatalogResult) generated.ProjectCreationMethodCatalogResponse {
	return generated.ProjectCreationMethodCatalogResponse{
		Items: append([]generated.ProjectCreationMethod(nil), result.Items...),
	}
}

// toDiscoveryCandidatesResponse 将发现候选结果转换为 OpenAPI 响应，并保留候选项及结果级可选字段。
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
			sourceType := generated.ProjectSourceKind(item.SourceType)
			candidate.SourceType = &sourceType
		}
		if item.StatusReason != nil {
			candidate.StatusReason = item.StatusReason
		}
		items = append(items, candidate)
	}
	response := generated.ProjectDiscoveryCandidatesResponse{
		SourceType:            generated.ProjectSourceKind(result.SourceType),
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

// toRuntimeImportInspectResponse 将运行时导入检查结果转换为 OpenAPI 响应。
func toRuntimeImportInspectResponse(result RuntimeImportInspectResult) generated.ProjectImportRuntimeInspectResponse {
	return generated.ProjectImportRuntimeInspectResponse{
		InspectionId:               result.InspectionID,
		ExpiresAt:                  result.ExpiresAt,
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
		LifecycleConfiguration:     toGeneratedLifecycleConfigurationRequest(result.LifecycleConfiguration),
	}
}

// toGeneratedLifecycleConfigurationRequest 将内部标准生命周期配置转换为 OpenAPI 生命周期配置请求。
// 返回的请求使用标准策略，并包含配置的配置文件、策略选项和附加参数。
func toGeneratedLifecycleConfigurationRequest(config LifecycleStandardConfig) generated.ProjectLifecycleConfigurationRequest {
	additionalArgs := append([]string{}, config.AdditionalArgs...)
	return generated.ProjectLifecycleConfigurationRequest{
		StrategyKind:             generated.ProjectLifecycleStrategyKindStandard,
		Profiles:                 append([]string{}, config.Profiles...),
		DownBeforeRedeploy:       config.DownBeforeRedeploy,
		PullBeforeRedeploy:       config.PullBeforeRedeploy,
		BuildBeforeUp:            config.BuildBeforeUp,
		ForceRecreate:            config.ForceRecreate,
		RemoveOrphans:            config.RemoveOrphans,
		WaitAfterUp:              config.WaitAfterUp,
		WaitTimeoutSeconds:       config.WaitTimeoutSeconds,
		RenewAnonVolumes:         config.RenewAnonVolumes,
		PruneImagesAfterRedeploy: config.PruneImagesAfterRedeploy,
		AdditionalArgs:           &additionalArgs,
	}
}

// lifecycleStandardConfigFromGenerated 将生成的生命周期配置请求转换为标准配置；策略类型不是 standard 时返回错误。
func lifecycleStandardConfigFromGenerated(config generated.ProjectLifecycleConfigurationRequest) (LifecycleStandardConfig, error) {
	if config.StrategyKind != generated.ProjectLifecycleStrategyKindStandard {
		return LifecycleStandardConfig{}, errProjectInvalidArgument
	}
	return toLifecycleConfigurationRequest(config), nil
}

// toRuntimeImportNetworkResources 将运行时导入网络资源转换为 OpenAPI 网络资源，并复制容器和服务名称切片。
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
			AbsolutePath:     item.AbsolutePath,
			DisplayPath:      item.DisplayPath,
			Kind:             generated.ProjectFileKind(item.Kind),
			LastObservedHash: item.LastObservedHash,
			OrderIndex:       item.OrderIndex,
			Role:             generated.ProjectFileRole(item.Role),
		})
	}
	return files
}

// toConfigurationMetadataResponse 将配置元数据结果转换为 OpenAPI 的项目配置元数据响应。
// 当诊断摘要非空时，会复制后作为可选字段返回。
func toConfigurationMetadataResponse(result ConfigurationMetadataResult) generated.ProjectConfigurationMetadataResponse {
	response := generated.ProjectConfigurationMetadataResponse{
		ProjectId:     mustGeneratedID(result.ProjectID),
		ComposeFiles:  result.ComposeFiles,
		EnvFiles:      result.EnvFiles,
		OwnershipMode: generated.ProjectOwnershipMode(result.OwnershipMode),
		DriftStatus:   generated.ProjectDriftStatus(result.DriftStatus),
	}
	if len(result.DiagnosticsSummary) > 0 {
		summary := append([]string(nil), result.DiagnosticsSummary...)
		response.DiagnosticsSummary = &summary
	}
	return response
}

// toConfigurationPreviewResponse 将配置预览结果转换为项目配置预览响应。
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
func toProjectWorkspaceFilesResponse(result workspaceFilesResult) generated.ProjectFilesResponse {
	items := make([]generated.ProjectFileTreeItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, generated.ProjectFileTreeItem{
			Name:            item.Name,
			RelativePath:    item.RelativePath,
			NodeType:        generated.ProjectFileTreeNodeType(item.NodeType),
			FileKind:        generated.ProjectWorkspaceFileKind(item.FileKind),
			Readable:        item.Readable,
			Editable:        item.Editable,
			LanguageHint:    item.LanguageHint,
			SizeBytes:       item.SizeBytes,
			HiddenByDefault: item.HiddenByDefault,
			HasChildren:     item.HasChildren,
			ProjectNote:     optionalString(item.ProjectNote),
			Tooltip:         optionalString(item.Tooltip),
			TooltipSource:   optionalTooltipSource(item.TooltipSource),
		})
	}
	response := generated.ProjectFilesResponse{
		ProjectId:     mustGeneratedID(result.ProjectID),
		RootPath:      result.RootPath,
		CurrentPath:   result.CurrentPath,
		HasMoreHidden: result.HasMoreHidden,
		Items:         items,
	}
	if result.ParentPath != nil {
		response.ParentPath = result.ParentPath
	}
	return response
}

func toProjectWorkspaceFileContentResponse(result workspaceFileContentResult) generated.ProjectFileContentResponse {
	return generated.ProjectFileContentResponse{
		ProjectId:    mustGeneratedID(result.ProjectID),
		RelativePath: result.RelativePath,
		FileKind:     generated.ProjectWorkspaceFileKind(result.FileKind),
		LanguageHint: result.LanguageHint,
		Readable:     result.Readable,
		Editable:     result.Editable,
		Encoding:     generated.ProjectFileContentResponseEncoding(result.Encoding),
		Content:      result.Content,
		SizeBytes:    result.SizeBytes,
	}
}

// toProjectWorkspaceFileSaveResponse 将工作区文件保存结果转换为项目文件保存响应。
// 生成包含项目 ID、相对路径、保存时间、内容哈希和文件大小的响应。
func toProjectWorkspaceFileSaveResponse(result workspaceFileSaveResult) generated.ProjectFileSaveResponse {
	return generated.ProjectFileSaveResponse{
		ProjectId:    mustGeneratedID(result.ProjectID),
		RelativePath: result.RelativePath,
		SavedAt:      result.SavedAt,
		ContentHash:  result.ContentHash,
		SizeBytes:    result.SizeBytes,
	}
}

// 当声明服务数大于等于 0 时包含声明服务数；当消息键、消息或守卫结果存在时一并写入响应。
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

// toLifecycleConfigurationRequest 将生成的生命周期配置请求转换为标准生命周期配置，并复制切片字段以避免共享底层存储。
func toLifecycleConfigurationRequest(request generated.ProjectLifecycleConfigurationRequest) LifecycleStandardConfig {
	additionalArgs := []string{}
	if request.AdditionalArgs != nil {
		additionalArgs = append(additionalArgs, (*request.AdditionalArgs)...)
	}
	return LifecycleStandardConfig{
		Profiles:                 append([]string(nil), request.Profiles...),
		DownBeforeRedeploy:       request.DownBeforeRedeploy,
		PullBeforeRedeploy:       request.PullBeforeRedeploy,
		BuildBeforeUp:            request.BuildBeforeUp,
		ForceRecreate:            request.ForceRecreate,
		RemoveOrphans:            request.RemoveOrphans,
		WaitAfterUp:              request.WaitAfterUp,
		WaitTimeoutSeconds:       request.WaitTimeoutSeconds,
		RenewAnonVolumes:         request.RenewAnonVolumes,
		PruneImagesAfterRedeploy: request.PruneImagesAfterRedeploy,
		AdditionalArgs:           additionalArgs,
	}
}

// toActionResponse 将动作结果转换为项目动作响应，并包含可选的消息和守卫结果。
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

// toTaskReceiptResponse 从操作结果的守卫结果中提取正整数任务 ID，构建待处理任务回执；未找到时仅返回待处理状态。
func toTaskReceiptResponse(result ActionResult) generated.TaskReceipt {
	for _, guard := range result.GuardResults {
		if guard.Code != "task_id" || guard.Detail == nil {
			continue
		}
		id, err := strconv.ParseInt(*guard.Detail, 10, 64)
		if err == nil && id > 0 {
			return generated.TaskReceipt{TaskId: id, Status: generated.TaskStatusPending}
		}
	}
	return generated.TaskReceipt{Status: generated.TaskStatusPending}
}

// toBatchActionResponse 将批量操作结果转换为 OpenAPI 批量操作响应。
// 返回包含各状态计数及逐项目操作结果的响应。
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

// toManagedRootResponse 将托管根目录信息转换为 OpenAPI 响应，并在可用时附带配置目录和状态原因。
func toManagedRootResponse(info ManagedRootInfo) generated.ProjectManagedRootResponse {
	response := generated.ProjectManagedRootResponse{
		SourceType:            generated.ProjectSourceKind(info.SourceType),
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

// toManagedCreateValidateResponse 将托管项目创建校验结果转换为项目创建校验响应，并包含可选的环境文件、来源元数据和告警。
func toManagedCreateValidateResponse(result ManagedProjectCreateValidationResult) generated.ProjectCreateValidateResponse {
	response := generated.ProjectCreateValidateResponse{
		ManagedRoot:             toManagedRootResponse(result.ManagedRoot),
		SourceType:              generated.ProjectSourceKind(result.SourceType),
		DisplayName:             result.DisplayName,
		ComposeProjectName:      result.ComposeProjectName,
		ApplicationName:         result.ApplicationName,
		OwnershipMode:           generated.ProjectOwnershipMode(result.OwnershipMode),
		WorkspacePath:           result.WorkspacePath,
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

// toManagedCreateResponse 将托管项目创建结果转换为项目创建响应，包含创建结果、配置快照及可选环境文件、来源元数据和告警。
func toManagedCreateResponse(result ManagedProjectCreateResult) generated.ProjectCreateResponse {
	response := generated.ProjectCreateResponse{
		ManagedRoot:             toManagedRootResponse(result.Validation.ManagedRoot),
		SourceType:              generated.ProjectSourceKind(result.SourceType),
		ApplicationId:           result.ApplicationID,
		DisplayName:             result.Validation.DisplayName,
		ComposeProjectName:      result.Validation.ComposeProjectName,
		ApplicationName:         result.Validation.ApplicationName,
		OwnershipMode:           generated.ProjectOwnershipMode(result.Validation.OwnershipMode),
		WorkspacePath:           result.Validation.WorkspacePath,
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

// toManagedCreateRequest 将工作区条目校验请求转换为托管项目创建请求；运行时目标 ID 或工作区条目无效时返回错误。
func toManagedCreateRequest(request generated.PostProjectCreateValidateJSONRequestBody) (ManagedProjectCreateRequest, error) {
	runtimeTargetID, err := runtimeTargetIDFromGenerated(request.RuntimeTargetId)
	if err != nil {
		return ManagedProjectCreateRequest{}, err
	}
	return managedCreateRequestFromEntries(managedCreateEntriesHTTPParts{displayName: request.DisplayName, runtimeTargetID: runtimeTargetID, applicationName: stringPointer(request.ApplicationName), workspaceEntries: request.WorkspaceEntries, composeFilePath: request.ComposeFilePath, lifecycle: request.LifecycleConfiguration, reuseExistingWorkspace: request.ReuseExistingWorkspace != nil && *request.ReuseExistingWorkspace})
}

type managedCreateEntriesHTTPParts struct {
	displayName            string
	runtimeTargetID        uint64
	applicationName        *string
	workspaceEntries       []generated.ProjectWorkspaceEntry
	composeFilePath        string
	lifecycle              *generated.ProjectLifecycleConfigurationRequest
	reuseExistingWorkspace bool
}

// managedCreateRequestFromEntries 将工作区条目转换为托管项目创建请求，并提取 Compose 文件内容。
// 当 Compose 文件路径无效、工作区条目无效、Compose 文件缺失或生命周期配置无效时返回错误。
func managedCreateRequestFromEntries(parts managedCreateEntriesHTTPParts) (ManagedProjectCreateRequest, error) {
	composePath, err := normalizeManagedWorkspacePath(parts.composeFilePath)
	if err != nil {
		return ManagedProjectCreateRequest{}, err
	}
	entries := make([]ManagedWorkspaceEntry, 0, len(parts.workspaceEntries))
	var composeContent string
	foundCompose := false
	for _, entry := range parts.workspaceEntries {
		mapped, err := managedWorkspaceEntryFromGenerated(entry)
		if err != nil {
			return ManagedProjectCreateRequest{}, err
		}
		entries = append(entries, mapped)
		if mapped.NodeType == "file" && mapped.Path == composePath && mapped.Content != nil {
			composeContent = *mapped.Content
			foundCompose = true
		}
	}
	if !foundCompose {
		return ManagedProjectCreateRequest{}, errProjectInvalidArgument
	}
	request := ManagedProjectCreateRequest{
		DisplayName: parts.displayName, RuntimeTargetID: parts.runtimeTargetID, ApplicationName: parts.applicationName,
		ReuseExistingWorkspace: parts.reuseExistingWorkspace,
		ComposeFileName:        filepath.Base(composePath), ComposeFileContent: composeContent, ComposeFilePath: composePath,
		WorkspaceEntries: entries,
	}
	if parts.lifecycle != nil {
		config, err := lifecycleStandardConfigFromGenerated(*parts.lifecycle)
		if err != nil {
			return ManagedProjectCreateRequest{}, err
		}
		request.LifecycleConfig = &config
	}
	return request, nil
}

// managedWorkspaceEntryFromGenerated 将生成的工作区条目转换为内部表示；文件必须包含内容，目录必须省略内容。
func managedWorkspaceEntryFromGenerated(entry generated.ProjectWorkspaceEntry) (ManagedWorkspaceEntry, error) {
	path, err := normalizeManagedWorkspacePath(entry.Path)
	if err != nil {
		return ManagedWorkspaceEntry{}, err
	}
	nodeType := string(entry.NodeType)
	if nodeType != "file" && nodeType != "directory" {
		return ManagedWorkspaceEntry{}, errProjectInvalidArgument
	}
	if nodeType == "file" && entry.Content == nil {
		return ManagedWorkspaceEntry{}, errProjectInvalidArgument
	}
	if nodeType == "directory" && entry.Content != nil {
		return ManagedWorkspaceEntry{}, errProjectInvalidArgument
	}
	return ManagedWorkspaceEntry{Path: path, NodeType: nodeType, Content: entry.Content}, nil
}

// toManagedCreateExecuteRequest 将执行请求转换为内部托管项目创建请求，并返回参数校验错误。
func toManagedCreateExecuteRequest(request generated.PostProjectCreateJSONRequestBody) (ManagedProjectCreateRequest, error) {
	runtimeTargetID, err := runtimeTargetIDFromGenerated(request.RuntimeTargetId)
	if err != nil {
		return ManagedProjectCreateRequest{}, err
	}
	return managedCreateRequestFromEntries(managedCreateEntriesHTTPParts{displayName: request.DisplayName, runtimeTargetID: runtimeTargetID, applicationName: stringPointer(request.ApplicationName), workspaceEntries: request.WorkspaceEntries, composeFilePath: request.ComposeFilePath, lifecycle: request.LifecycleConfiguration, reuseExistingWorkspace: request.ReuseExistingWorkspace != nil && *request.ReuseExistingWorkspace})
}

func toApplicationNameAvailabilityResponse(result ApplicationNameAvailabilityResult) generated.ProjectApplicationNameAvailabilityResponse {
	response := generated.ProjectApplicationNameAvailabilityResponse{
		Status:            generated.ProjectApplicationNameAvailabilityResponseStatus(result.Status),
		WorkspacePath:     result.WorkspacePath,
		WorkspaceNonEmpty: result.WorkspaceNonEmpty,
	}
	if result.ComposeFilePath != nil {
		response.ComposeFilePath = result.ComposeFilePath
	}
	if len(result.WorkspaceEntries) > 0 {
		entries := make([]generated.ProjectWorkspaceEntry, 0, len(result.WorkspaceEntries))
		for _, entry := range result.WorkspaceEntries {
			entries = append(entries, generated.ProjectWorkspaceEntry{Path: entry.Path, NodeType: generated.ProjectWorkspaceEntryNodeType(entry.NodeType), Content: entry.Content})
		}
		response.WorkspaceEntries = &entries
	}
	return response
}

// runtimeTargetIDFromGenerated 校验并转换运行时目标标识；标识小于 1 时返回参数错误。
func runtimeTargetIDFromGenerated(value int64) (uint64, error) {
	if value < 1 {
		return 0, errProjectInvalidArgument
	}
	return uint64(value), nil
}

// toProjectOverviewServiceItem 将服务投影和运行时资源摘要转换为项目概览服务项。
// 返回服务概览项及其是否处于健康状态的标志。
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
// items 为空时返回 nil，否则返回其底层数据拷贝的指针。
func optionalStringSlice(items []string) *[]string {
	if len(items) == 0 {
		return nil
	}
	copyItems := append([]string(nil), items...)
	return &copyItems
}

func optionalTooltipSource(value string) *generated.ProjectFileTreeItemTooltipSource {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	source := generated.ProjectFileTreeItemTooltipSource(trimmed)
	return &source
}
