package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

func sameWorkingDirectory(left string, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

// toProjectListItemWithManagedRoot 将聚合信息映射为项目列表项，并在提供运行时摘要时补充容器数量。
// 结果包含项目标识、名称、来源、工作目录、声明服务数，以及最近刷新和漂移状态。
func toProjectListItemWithManagedRoot(
	aggregate projectstore.ProjectAggregate,
	managedRootDirectory string,
	runtimeSummary *moduleapi.ContainerProjectRuntimeSummary,
	runtimeErr error,
) generated.ProjectListItem {
	serviceCount := 0
	if aggregate.Snapshot != nil {
		serviceCount = aggregate.Snapshot.DeclaredServiceCount
	}
	counts := buildProjectContainerCounts(runtimeSummary)
	return generated.ProjectListItem{
		Id:                         mustGeneratedID(aggregate.Project.ID),
		DisplayName:                aggregate.Project.DisplayName,
		CanonicalProjectName:       aggregate.Project.CanonicalProjectName,
		CanonicalProjectNameSource: generated.ProjectCanonicalNameSource(aggregate.Project.CanonicalProjectNameSource),
		SourceKind:                 generated.ProjectSourceKind(aggregate.Project.SourceKind),
		LifecycleReviewStatus:      generated.ProjectLifecycleReviewStatus(nonEmptyString(aggregate.Project.LifecycleReviewStatus, projectcontract.LifecycleReviewStatusReviewRequired.String())),
		SourceMetadata:             buildListSourceMetadataWithManagedRoot(aggregate, managedRootDirectory),
		ActivityAuthority:          generated.ProjectActivityAuthority(resolveActivityAuthority(aggregate)),
		HostScope:                  generated.ProjectHostScope(aggregate.Project.HostScope),
		OwnershipMode:              generated.ProjectOwnershipMode(aggregate.Project.OwnershipMode),
		WorkingDirectory:           aggregate.Project.WorkingDirectory,
		RuntimeStatus:              deriveProjectRuntimeStatus(runtimeSummary, runtimeErr),
		ServiceCount:               serviceCount,
		ContainerCounts:            counts,
		DriftStatus:                generated.ProjectDriftStatus(aggregate.Project.DriftStatus),
	}
}

// toProjectDetailResponse 将项目聚合数据转换为详情响应。
//
// toProjectDetailResponse 将项目聚合转换为详情响应，并在提供运行时汇总时填充容器运行与停止数量。
// 当聚合包含快照时，会写入服务数；刷新错误信息和配置哈希仅在存在时写入响应。
func toProjectDetailResponse(
	aggregate projectstore.ProjectAggregate,
	runtimeSummary *moduleapi.ContainerProjectRuntimeSummary,
	runtimeErr error,
) generated.ProjectDetailResponse {
	return toProjectDetailResponseWithManagedRoot(aggregate, "", runtimeSummary, runtimeErr)
}

func toProjectDetailResponseWithManagedRoot(
	aggregate projectstore.ProjectAggregate,
	managedRootDirectory string,
	runtimeSummary *moduleapi.ContainerProjectRuntimeSummary,
	runtimeErr error,
) generated.ProjectDetailResponse {
	counts := buildProjectContainerCounts(runtimeSummary)
	item := generated.ProjectDetailResponse{
		CanonicalProjectName:       aggregate.Project.CanonicalProjectName,
		CanonicalProjectNameSource: generated.ProjectCanonicalNameSource(aggregate.Project.CanonicalProjectNameSource),
		LifecycleReviewStatus:      generated.ProjectLifecycleReviewStatus(nonEmptyString(aggregate.Project.LifecycleReviewStatus, projectcontract.LifecycleReviewStatusReviewRequired.String())),
		LifecycleConfiguration:     toGeneratedProjectLifecycleConfiguration(aggregate),
		ComposeFiles:               toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())),
		ContainerCounts:            counts,
		DisplayName:                aggregate.Project.DisplayName,
		DriftStatus:                generated.ProjectDriftStatus(aggregate.Project.DriftStatus),
		EnvFiles:                   toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindEnv.String())),
		HostScope:                  generated.ProjectHostScope(aggregate.Project.HostScope),
		Id:                         mustGeneratedID(aggregate.Project.ID),
		LastDriftCheckedAt:         aggregate.Project.LastDriftCheckedAt,
		OwnershipMode:              generated.ProjectOwnershipMode(aggregate.Project.OwnershipMode),
		RuntimeStatus:              deriveProjectRuntimeStatus(runtimeSummary, runtimeErr),
		SourceKind:                 generated.ProjectSourceKind(aggregate.Project.SourceKind),
		SourceMetadata:             buildDetailSourceMetadataWithManagedRoot(aggregate, managedRootDirectory),
		ActivityAuthority:          generated.ProjectActivityAuthority(resolveActivityAuthority(aggregate)),
		WorkingDirectory:           aggregate.Project.WorkingDirectory,
	}
	if aggregate.Project.LastObservedConfigHash != "" {
		item.LastObservedConfigHash = stringPointer(aggregate.Project.LastObservedConfigHash)
	}
	if aggregate.Snapshot != nil {
		item.ServiceCount = aggregate.Snapshot.DeclaredServiceCount
	}
	return item
}

func toProjectLifecycleConfigurationResponse(
	aggregate projectstore.ProjectAggregate,
) generated.ProjectLifecycleConfigurationResponse {
	config := lifecycleConfigurationFromAggregate(aggregate)
	return generated.ProjectLifecycleConfigurationResponse{
		ProjectId:              mustGeneratedID(aggregate.Project.ID),
		LifecycleReviewStatus:  generated.ProjectLifecycleReviewStatus(nonEmptyString(aggregate.Project.LifecycleReviewStatus, projectcontract.LifecycleReviewStatusReviewRequired.String())),
		WorkingDirectory:       aggregate.Project.WorkingDirectory,
		CanonicalProjectName:   aggregate.Project.CanonicalProjectName,
		ComposeFiles:           toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())),
		LifecycleConfiguration: toGeneratedLifecycleConfiguration(config),
	}
}

func toGeneratedProjectLifecycleConfiguration(
	aggregate projectstore.ProjectAggregate,
) generated.ProjectLifecycleConfiguration {
	return toGeneratedLifecycleConfiguration(lifecycleConfigurationFromAggregate(aggregate))
}

func toGeneratedLifecycleConfiguration(config LifecycleConfiguration) generated.ProjectLifecycleConfiguration {
	return generated.ProjectLifecycleConfiguration{
		StrategyKind:             generated.ProjectLifecycleStrategyKind(config.StrategyKind),
		Profiles:                 append([]string(nil), config.Standard.Profiles...),
		DownBeforeRedeploy:       config.Standard.DownBeforeRedeploy,
		PullBeforeRedeploy:       config.Standard.PullBeforeRedeploy,
		BuildBeforeUp:            config.Standard.BuildBeforeUp,
		ForceRecreate:            config.Standard.ForceRecreate,
		RemoveOrphans:            config.Standard.RemoveOrphans,
		WaitAfterUp:              config.Standard.WaitAfterUp,
		WaitTimeoutSeconds:       config.Standard.WaitTimeoutSeconds,
		RenewAnonVolumes:         config.Standard.RenewAnonVolumes,
		PruneImagesAfterRedeploy: config.Standard.PruneImagesAfterRedeploy,
		GeneratedCommands:        toGeneratedLifecycleCommands(config),
	}
}

func toGeneratedLifecycleCommands(config LifecycleConfiguration) struct {
	Redeploy generated.ProjectLifecycleGeneratedCommand `json:"redeploy"`
	Restart  generated.ProjectLifecycleGeneratedCommand `json:"restart"`
	Stop     generated.ProjectLifecycleGeneratedCommand `json:"stop"`
	Up       generated.ProjectLifecycleGeneratedCommand `json:"up"`
} {
	return struct {
		Redeploy generated.ProjectLifecycleGeneratedCommand `json:"redeploy"`
		Restart  generated.ProjectLifecycleGeneratedCommand `json:"restart"`
		Stop     generated.ProjectLifecycleGeneratedCommand `json:"stop"`
		Up       generated.ProjectLifecycleGeneratedCommand `json:"up"`
	}{
		Redeploy: buildGeneratedLifecycleCommand(config, "redeploy"),
		Restart:  buildGeneratedLifecycleCommand(config, "restart"),
		Stop:     buildGeneratedLifecycleCommand(config, "stop"),
		Up:       buildGeneratedLifecycleCommand(config, "up"),
	}
}

func buildGeneratedLifecycleCommand(
	config LifecycleConfiguration,
	action string,
) generated.ProjectLifecycleGeneratedCommand {
	steps := buildLifecycleCommandSteps(config, action)
	displayParts := make([]string, 0, len(steps))
	for _, item := range steps {
		displayParts = append(displayParts, item.DisplayCommand)
	}
	return generated.ProjectLifecycleGeneratedCommand{
		Action:         generated.ProjectLifecycleGeneratedCommandAction(action),
		Steps:          steps,
		DisplayCommand: strings.Join(displayParts, "\n"),
	}
}

func buildLifecycleCommandSteps(
	config LifecycleConfiguration,
	action string,
) []generated.ProjectLifecycleCommandStep {
	base := buildLifecycleBaseArgv(config)
	switch action {
	case "up":
		return []generated.ProjectLifecycleCommandStep{buildLifecycleCommandStep("up", buildLifecycleUpArgv(base, config.Standard))}
	case "stop":
		return []generated.ProjectLifecycleCommandStep{buildLifecycleCommandStep("stop", append(base, "stop"))}
	case "restart":
		return []generated.ProjectLifecycleCommandStep{buildLifecycleCommandStep("restart", append(base, "restart"))}
	case "redeploy":
		steps := make([]generated.ProjectLifecycleCommandStep, 0, lifecycleRedeployStepCap)
		if config.Standard.DownBeforeRedeploy {
			steps = append(steps, buildLifecycleCommandStep("down", append(append([]string(nil), base...), "down")))
		}
		if config.Standard.PullBeforeRedeploy {
			steps = append(steps, buildLifecycleCommandStep("pull", append(append([]string(nil), base...), "pull")))
		}
		steps = append(steps, buildLifecycleCommandStep("up", buildLifecycleUpArgv(base, config.Standard)))
		if config.Standard.PruneImagesAfterRedeploy {
			steps = append(steps, buildLifecycleCommandStep("prune", []string{"docker", "image", "prune", "-f"}))
		}
		return steps
	default:
		return []generated.ProjectLifecycleCommandStep{buildLifecycleCommandStep("up", buildLifecycleUpArgv(base, config.Standard))}
	}
}

func buildLifecycleBaseArgv(config LifecycleConfiguration) []string {
	base := []string{"docker", "compose"}
	for _, file := range config.ComposeFiles {
		base = append(base, "-f", file)
	}
	for _, profile := range config.Standard.Profiles {
		base = append(base, "--profile", profile)
	}
	if strings.TrimSpace(config.ProjectName) != "" {
		base = append(base, "-p", config.ProjectName)
	}
	return base
}

func buildLifecycleUpArgv(base []string, standard LifecycleStandardConfig) []string {
	args := append(append([]string(nil), base...), "up", "-d")
	if standard.BuildBeforeUp {
		args = append(args, "--build")
	}
	if standard.ForceRecreate {
		args = append(args, "--force-recreate")
	}
	if standard.RemoveOrphans {
		args = append(args, "--remove-orphans")
	}
	if standard.RenewAnonVolumes {
		args = append(args, "--renew-anon-volumes")
	}
	if standard.WaitAfterUp {
		args = append(args, "--wait")
		args = append(args, "--wait-timeout", fmt.Sprintf("%d", standard.WaitTimeoutSeconds))
	}
	return args
}

func buildLifecycleCommandStep(kind string, argv []string) generated.ProjectLifecycleCommandStep {
	return generated.ProjectLifecycleCommandStep{
		Kind:           generated.ProjectLifecycleCommandStepKind(kind),
		Argv:           append([]string(nil), argv...),
		DisplayCommand: strings.Join(argv, " "),
	}
}

// toGeneratedFiles 将存储的文件记录转换为生成的文件项列表。
func toGeneratedFiles(files []projectstore.ProjectFile) []generated.ProjectFileItem {
	items := make([]generated.ProjectFileItem, 0, len(files))
	for _, item := range files {
		items = append(items, generated.ProjectFileItem{
			Id:               mustGeneratedID(item.ID),
			Kind:             generated.ProjectFileKind(item.Kind),
			Role:             generated.ProjectFileRole(item.Role),
			AbsolutePath:     item.AbsolutePath,
			DisplayPath:      item.DisplayPath,
			OrderIndex:       item.OrderIndex,
			LastObservedHash: optionalString(item.LastObservedHash),
		})
	}
	return items
}

// 将 compose 投影文件转换为生成的项目文件项列表。
func toGeneratedFilesFromCompose(files []projectcompose.FileProjection) []generated.ProjectFileItem {
	items := make([]generated.ProjectFileItem, 0, len(files))
	for index, item := range files {
		hash := item.Hash
		items = append(items, generated.ProjectFileItem{
			Id:               int64(index + 1),
			Kind:             generated.ProjectFileKind(item.Kind),
			Role:             generated.ProjectFileRole(item.Role),
			AbsolutePath:     item.AbsolutePath,
			DisplayPath:      item.DisplayPath,
			OrderIndex:       item.OrderIndex,
			LastObservedHash: &hash,
		})
	}
	return items
}

// toStoreFiles 将 compose 和 env 文件投影转换为存储层文件记录。
func toStoreFiles(composeFiles []projectcompose.FileProjection, envFiles []projectcompose.FileProjection) []projectstore.ProjectFile {
	items := make([]projectstore.ProjectFile, 0, len(composeFiles)+len(envFiles))
	for _, item := range append(append([]projectcompose.FileProjection(nil), composeFiles...), envFiles...) {
		items = append(items, projectstore.ProjectFile{
			Kind:             item.Kind,
			Role:             item.Role,
			AbsolutePath:     item.AbsolutePath,
			DisplayPath:      item.DisplayPath,
			OrderIndex:       item.OrderIndex,
			LastObservedHash: hashString(string(item.Content)),
		})
	}
	return items
}

// 返回的文件先按 OrderIndex 升序排列，OrderIndex 相同时按 ID 升序排列。
func filterFiles(files []projectstore.ProjectFile, kind string) []projectstore.ProjectFile {
	items := make([]projectstore.ProjectFile, 0)
	for _, item := range files {
		if item.Kind == kind {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(left, right int) bool {
		if items[left].OrderIndex == items[right].OrderIndex {
			return items[left].ID < items[right].ID
		}
		return items[left].OrderIndex < items[right].OrderIndex
	})
	return items
}

// collectFilesByKind 返回指定类型的文件绝对路径列表。
func collectFilesByKind(files []projectstore.ProjectFile, kind string) []string {
	filtered := filterFiles(files, kind)
	paths := make([]string, 0, len(filtered))
	for _, item := range filtered {
		paths = append(paths, item.AbsolutePath)
	}
	return paths
}

func (s *Service) loadFromAggregate(aggregate projectstore.ProjectAggregate) (projectcompose.Result, error) {
	return projectcompose.Load(projectcompose.Input{
		WorkingDirectory: aggregate.Project.WorkingDirectory,
		ComposeFiles:     collectFilesByKind(aggregate.Files, projectcontract.FileKindCompose.String()),
		EnvFiles:         collectFilesByKind(aggregate.Files, projectcontract.FileKindEnv.String()),
	})
}

// normalizeSnapshotJSON 将输入内容规范化为 JSON 表示。
// 输入为空时返回 "{}"；当解析或重新编码失败时，返回原始内容。
func normalizeSnapshotJSON(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	var generic any
	if err := yamlJSONRoundTrip(raw, &generic); err != nil {
		return raw
	}
	encoded, err := json.Marshal(generic)
	if err != nil {
		return raw
	}
	return encoded
}

// yamlJSONRoundTrip 将 JSON 数据解析到目标值。
//
// @param raw 要解析的数据。
// @param target 接收解析结果的目标值。
// @returns 解析过程中返回的错误。
func yamlJSONRoundTrip(raw []byte, target any) error {
	return json.Unmarshal(raw, target)
}

// digestServiceNames 计算服务名称集合的稳定摘要。
// digestServiceNames 对服务名按字典序排序后计算摘要。
// 返回排序后的名称序列对应的 SHA-256 十六进制字符串。
func digestServiceNames(names []string) string {
	normalized := append([]string(nil), names...)
	sort.Strings(normalized)
	hasher := sha256.New()
	for _, item := range normalized {
		mustWriteDigestFragment(hasher, []byte(item))
		mustWriteDigestFragment(hasher, []byte{0})
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// mustWriteDigestFragment 将摘要片段写入给定写入器。
// 写入失败时会 panic。
func mustWriteDigestFragment(writer io.Writer, value []byte) {
	if _, err := writer.Write(value); err != nil {
		panic(fmt.Sprintf("project digest writer failed: %v", err))
	}
}

// buildRefreshProjectInput 组装项目刷新持久化输入，包含刷新状态、快照、文件与操作者信息。
// buildRefreshProjectInput 构建用于刷新项目存储记录的输入。
// 它写入刷新状态、刷新时间、配置哈希、归一化后的 compose 快照和声明服务摘要，并保留操作者信息。
func buildRefreshProjectInput(
	projectID uint64,
	parseResult projectcompose.Result,
	now time.Time,
	actorID *uint64,
) projectstore.RefreshProjectInput {
	return projectstore.RefreshProjectInput{
		ProjectID:              projectID,
		LastObservedConfigHash: parseResult.ConfigHash,
		LastDriftCheckedAt:     &now,
		DriftStatus:            projectcontract.DriftStatusClean.String(),
		Files:                  toStoreFiles(parseResult.ComposeFiles, parseResult.EnvFiles),
		Snapshot: &projectstore.Snapshot{
			ProjectID:              projectID,
			ConfigHash:             parseResult.ConfigHash,
			NormalizedComposeJSON:  normalizeSnapshotJSON(parseResult.NormalizedComposeJSON),
			DeclaredServiceCount:   len(parseResult.ServiceNames),
			DeclaredServicesDigest: digestServiceNames(parseResult.ServiceNames),
			RefreshedAt:            now,
		},
		ActorID: actorID,
	}
}

// displayNameOrCanonical 返回修剪后的显示名称或规范名称。
//
// 当显示名称存在且非空时，返回其去除首尾空白后的值；否则返回规范名称。
func displayNameOrCanonical(displayName *string, canonical string) string {
	if displayName != nil && strings.TrimSpace(*displayName) != "" {
		return strings.TrimSpace(*displayName)
	}
	return canonical
}

// hashString 返回归一化文本块的 SHA-256 十六进制摘要。
func hashString(value string) string {
	sum := sha256.Sum256([]byte(normalizeTextBlock(value)))
	return hex.EncodeToString(sum[:])
}

// normalizeTextBlock 规范化文本块的换行、行尾空白和整体边界，并在非空时补充结尾换行符。
func normalizeTextBlock(value string) string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRight(line, " \t")
	}
	joined := strings.TrimSpace(strings.Join(lines, "\n"))
	if joined == "" {
		return ""
	}
	return joined + "\n"
}

// nonEmptyString 在 primary 为空白时返回 fallback。
//
// @return primary 经修剪后非空时返回其原值；否则返回 fallback。
func nonEmptyString(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

// stringPointer 返回一个指向非空白字符串的指针。
//
// 如果输入经修剪后为空，则返回 nil。
func stringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// optionalString 将字符串包装为可选字符串指针。
func optionalString(value string) *string {
	return stringPointer(value)
}

// stringValue 返回指针指向的字符串；当指针为 nil 时返回空字符串。
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// mapManagedSourceCatalogStatus 将托管根状态映射为来源目录状态。
//
// @param status 托管根状态字符串。
// @returns 目录状态值；当状态为 ready 时返回 "ready"，其余情况返回 "blocked"。
func mapManagedSourceCatalogStatus(status string) string {
	switch strings.TrimSpace(status) {
	case projectcontract.ManagedRootStatusReady.String():
		return "ready"
	case projectcontract.ManagedRootStatusUnconfigured.String(), projectcontract.ManagedRootStatusInvalid.String():
		return "blocked"
	default:
		return "blocked"
	}
}

// managedRootStatusReasonKey 将托管根状态映射为状态原因键。
// 返回与给定托管根状态对应的原因键；当状态为就绪时返回 nil。
func managedRootStatusReasonKey(status string) *string {
	switch strings.TrimSpace(status) {
	case projectcontract.ManagedRootStatusReady.String():
		return nil
	case projectcontract.ManagedRootStatusUnconfigured.String():
		return stringPointer("project.createSource.statusReason.managedUnconfigured")
	case projectcontract.ManagedRootStatusInvalid.String():
		return stringPointer("project.createSource.statusReason.managedInvalid")
	default:
		return stringPointer("project.createSource.statusReason.managedUnknown")
	}
}

// toGeneratedSourceMetadata 将源元数据映射为生成的项目来源元数据。
//
// 仅在至少有一个已知字段可映射时返回结果；否则返回 nil。
func toGeneratedSourceMetadata(metadata map[string]string) *generated.ProjectSourceMetadata {
	if len(metadata) == 0 {
		return nil
	}
	result := generated.ProjectSourceMetadata{}
	assignSourceMetadataField(metadata, "managed_root_key", &result.ManagedRootKey)
	assignSourceMetadataField(metadata, "managed_relative_directory", &result.ManagedRelativeDirectory)
	assignSourceMetadataField(metadata, "managed_compose_file_name", &result.ManagedComposeFileName)
	assignSourceMetadataField(metadata, "managed_env_file_name", &result.ManagedEnvFileName)
	assignSourceMetadataField(metadata, "git_repository_url", &result.GitRepositoryUrl)
	assignSourceMetadataField(metadata, "git_reference", &result.GitReference)
	assignSourceMetadataField(metadata, "git_compose_subpath", &result.GitComposeSubpath)
	assignSourceMetadataField(metadata, "template_key", &result.TemplateKey)
	assignSourceMetadataField(metadata, "template_version", &result.TemplateVersion)
	assignSourceMetadataField(metadata, "template_instance_name", &result.TemplateInstanceName)
	if result == (generated.ProjectSourceMetadata{}) {
		return nil
	}
	return &result
}

// assignSourceMetadataField 将来源元数据中的指定值去除首尾空白后写入目标指针。
// 当对应值为空时保持目标不变。
func assignSourceMetadataField(metadata map[string]string, key string, target **string) {
	value := strings.TrimSpace(metadata[key])
	if value == "" {
		return
	}
	*target = &value
}

// buildListSourceMetadataWithManagedRoot 为项目列表构建来源元数据。
// 当来源类型为受托管根或远程主机时返回对应的来源元数据；其他来源类型返回 nil。
func buildListSourceMetadataWithManagedRoot(aggregate projectstore.ProjectAggregate, managedRootDirectory string) *generated.ProjectSourceMetadata {
	switch strings.TrimSpace(aggregate.Project.SourceKind) {
	case projectcontract.SourceKindManaged.String():
		return buildManagedSourceMetadata(aggregate, managedRootDirectory)
	case projectcontract.SourceKindRemoteHost.String():
		return buildRemoteHostSourceMetadata(aggregate)
	default:
		return nil
	}
}

// buildDetailSourceMetadataWithManagedRoot 返回项目详情来源元数据。
// 如果没有可映射的来源信息，则返回 nil。
func buildDetailSourceMetadataWithManagedRoot(aggregate projectstore.ProjectAggregate, managedRootDirectory string) *generated.ProjectSourceMetadata {
	switch strings.TrimSpace(aggregate.Project.SourceKind) {
	case projectcontract.SourceKindManaged.String():
		return buildManagedSourceMetadata(aggregate, managedRootDirectory)
	case projectcontract.SourceKindRemoteHost.String():
		return buildRemoteHostSourceMetadata(aggregate)
	default:
		return nil
	}
}

// buildManagedSourceMetadata 生成托管项目的来源元数据。
// 结果包含托管根标识、相对目录，以及已登记的 Compose 和环境文件名。
func buildManagedSourceMetadata(aggregate projectstore.ProjectAggregate, managedRootDirectory string) *generated.ProjectSourceMetadata {
	composeFiles := filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())
	envFiles := filterFiles(aggregate.Files, projectcontract.FileKindEnv.String())
	metadata := map[string]string{
		"managed_root_key": projectcontract.ProjectManagedRootConfig.String(),
	}
	if relativePath := deriveManagedRelativeDirectory(managedRootDirectory, aggregate.Project.WorkingDirectory); relativePath != "" {
		metadata["managed_relative_directory"] = relativePath
	}
	if len(composeFiles) > 0 {
		metadata["managed_compose_file_name"] = filepath.Base(composeFiles[0].AbsolutePath)
	}
	if len(envFiles) > 0 {
		metadata["managed_env_file_name"] = filepath.Base(envFiles[0].AbsolutePath)
	}
	return toGeneratedSourceMetadata(metadata)
}

// buildRemoteHostSourceMetadata 构建远程主机来源元数据。
// 元数据包含活动权威和汇总范围。
func buildRemoteHostSourceMetadata(aggregate projectstore.ProjectAggregate) *generated.ProjectSourceMetadata {
	activityAuthority := string(resolveActivityAuthority(aggregate))
	rollupScope := "planned-remote-summary"
	return &generated.ProjectSourceMetadata{
		ActivityAuthority:   &activityAuthority,
		ActivityRollupScope: &rollupScope,
	}
}

// resolveActivityAuthority 根据项目主机范围确定活动执行方式。
// 当项目的 HostScope 为 remote 时返回 `ProjectActivityAuthorityBackendPlanned`，否则返回 `ProjectActivityAuthorityFrontendFanout`。
func resolveActivityAuthority(aggregate projectstore.ProjectAggregate) ActivityAuthority {
	if strings.TrimSpace(aggregate.Project.HostScope) == projectcontract.HostScopeRemote.String() {
		return ProjectActivityAuthorityBackendPlanned
	}
	return ProjectActivityAuthorityFrontendFanout
}

// deriveManagedRelativeDirectory 从工作目录推导托管相对目录。
// 当存在可用 managed root 时返回其相对路径；否则回退到清理后的路径基名。
func deriveManagedRelativeDirectory(managedRootDirectory string, workingDirectory string) string {
	cleaned := filepath.Clean(strings.TrimSpace(workingDirectory))
	if cleaned == "" || cleaned == "." || cleaned == string(filepath.Separator) {
		return ""
	}
	root := filepath.Clean(strings.TrimSpace(managedRootDirectory))
	if !hasUsableManagedRoot(root) {
		return filepath.Base(cleaned)
	}
	relative, err := filepath.Rel(root, cleaned)
	if err == nil && isUsableManagedRelativePath(relative) {
		return filepath.ToSlash(relative)
	}
	return filepath.Base(cleaned)
}

func hasUsableManagedRoot(root string) bool {
	return root != "" && root != "." && root != string(filepath.Separator)
}

func isUsableManagedRelativePath(relative string) bool {
	if relative == "" || relative == "." || relative == ".." {
		return false
	}
	return !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func (s *Service) readyManagedRootDirectory(ctx context.Context) string {
	if s == nil {
		return ""
	}
	managedRoot, err := s.ManagedRoot(ctx)
	if err != nil || managedRoot.Status != projectcontract.ManagedRootStatusReady.String() || managedRoot.ConfiguredRootDirectory == nil {
		return ""
	}
	return filepath.Clean(strings.TrimSpace(*managedRoot.ConfiguredRootDirectory))
}

// uniqueStrings 返回去重后的字符串切片，保留首次出现的顺序。
func uniqueStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

// normalizeListLimit 将列表限制值规范到默认值或最大值范围内。
//
// @returns 规范后的列表限制值：当输入小于等于 0 时返回默认值，超过最大值时返回最大值，否则返回原值。
func normalizeListLimit(value int) int {
	switch {
	case value <= 0:
		return defaultProjectListLimit
	case value > maxProjectListLimit:
		return maxProjectListLimit
	default:
		return value
	}
}

// mustGeneratedID 将生成的无符号 ID 转换为 int64。
// 当值为 0 或超出 int64 可表示范围时会 panic。
func mustGeneratedID(value uint64) int64 {
	if value == 0 || value > math.MaxInt64 {
		panic("project generated id out of range")
	}
	return int64(value)
}

// maxInt 返回 value 与 minimum 中较大的值。
func maxInt(value int, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

// mapStoreError 将存储层错误映射为本包使用的错误。
// 已知的无效输入、未找到、冲突和文件不存在错误会转换为对应的本地哨兵错误；其他错误原样返回。
