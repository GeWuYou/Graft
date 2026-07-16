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

func sameWorkspacePath(left string, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

// toProjectListItemWithManagedRoot 将项目聚合数据转换为项目列表项，并在可用时补充运行时容器摘要。
func toProjectListItemWithManagedRoot(
	aggregate projectstore.ApplicationAggregate,
	managedRootDirectory string,
	runtimeSummary *moduleapi.ContainerProjectRuntimeSummary,
	runtimeErr error,
) generated.ApplicationListItem {
	serviceCount := 0
	if aggregate.Snapshot != nil {
		serviceCount = aggregate.Snapshot.DeclaredServiceCount
	}
	counts := buildProjectContainerCounts(runtimeSummary)
	return generated.ApplicationListItem{
		ApplicationId:            aggregate.Application.ApplicationID,
		ApplicationType:          generated.ApplicationType(nonEmptyString(aggregate.Application.ApplicationType, projectcontract.ApplicationTypeCompose.String())),
		DisplayName:              aggregate.Application.DisplayName,
		ComposeProjectName:       nonEmptyString(aggregate.Application.ComposeProjectName, aggregate.Application.ComposeProjectName),
		ComposeProjectNameSource: generated.ApplicationComposeProjectNameSource(nonEmptyString(aggregate.Application.ComposeProjectNameSource, aggregate.Application.ComposeProjectNameSource)),
		ApplicationName:          aggregate.Application.ApplicationName,
		SourceType:               generated.ApplicationSourceType(aggregate.Application.SourceType),
		LifecycleReviewStatus:    generated.ApplicationLifecycleReviewStatus(nonEmptyString(aggregate.Application.LifecycleReviewStatus, projectcontract.LifecycleReviewStatusReviewRequired.String())),
		SourceMetadata:           buildListSourceMetadataWithManagedRoot(aggregate, managedRootDirectory),
		ActivityAuthority:        generated.ApplicationActivityAuthority(resolveActivityAuthority()),
		OwnershipMode:            generated.ApplicationOwnershipMode(aggregate.Application.OwnershipMode),
		WorkspacePath:            aggregate.Application.WorkspacePath,
		RuntimeStatus:            deriveProjectRuntimeStatus(runtimeSummary, runtimeErr),
		ServiceCount:             serviceCount,
		ContainerCounts:          counts,
		DriftStatus:              generated.ApplicationDriftStatus(aggregate.Application.DriftStatus),
	}
}

// toProjectDetailResponse 将项目聚合数据转换为详情响应。
//
// toProjectDetailResponse 将项目聚合转换为详情响应，并在提供运行时汇总时填充容器运行与停止数量。
// 当聚合包含快照时，会写入服务数；刷新错误信息和配置哈希仅在存在时写入响应。
func toProjectDetailResponse(
	aggregate projectstore.ApplicationAggregate,
	runtimeSummary *moduleapi.ContainerProjectRuntimeSummary,
	runtimeErr error,
) generated.ApplicationDetailResponse {
	return toProjectDetailResponseWithManagedRoot(aggregate, "", runtimeSummary, runtimeErr)
}

// toProjectDetailResponseWithManagedRoot 将项目聚合数据转换为详情响应，并包含生命周期配置、文件、运行时状态及托管根目录来源元数据。
func toProjectDetailResponseWithManagedRoot(
	aggregate projectstore.ApplicationAggregate,
	managedRootDirectory string,
	runtimeSummary *moduleapi.ContainerProjectRuntimeSummary,
	runtimeErr error,
) generated.ApplicationDetailResponse {
	counts := buildProjectContainerCounts(runtimeSummary)
	item := generated.ApplicationDetailResponse{
		ApplicationType:          generated.ApplicationType(nonEmptyString(aggregate.Application.ApplicationType, projectcontract.ApplicationTypeCompose.String())),
		ComposeProjectName:       nonEmptyString(aggregate.Application.ComposeProjectName, aggregate.Application.ComposeProjectName),
		ComposeProjectNameSource: generated.ApplicationComposeProjectNameSource(nonEmptyString(aggregate.Application.ComposeProjectNameSource, aggregate.Application.ComposeProjectNameSource)),
		LifecycleReviewStatus:    generated.ApplicationLifecycleReviewStatus(nonEmptyString(aggregate.Application.LifecycleReviewStatus, projectcontract.LifecycleReviewStatusReviewRequired.String())),
		LifecycleConfiguration:   toGeneratedProjectLifecycleConfiguration(aggregate),
		ComposeFiles:             toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())),
		ContainerCounts:          counts,
		DisplayName:              aggregate.Application.DisplayName,
		DriftStatus:              generated.ApplicationDriftStatus(aggregate.Application.DriftStatus),
		EnvFiles:                 toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindEnv.String())),
		ApplicationId:            aggregate.Application.ApplicationID,
		LastDriftCheckedAt:       aggregate.Application.LastDriftCheckedAt,
		OwnershipMode:            generated.ApplicationOwnershipMode(aggregate.Application.OwnershipMode),
		RuntimeStatus:            deriveProjectRuntimeStatus(runtimeSummary, runtimeErr),
		SourceType:               generated.ApplicationSourceType(aggregate.Application.SourceType),
		SourceMetadata:           buildDetailSourceMetadataWithManagedRoot(aggregate, managedRootDirectory),
		ActivityAuthority:        generated.ApplicationActivityAuthority(resolveActivityAuthority()),
		WorkspacePath:            aggregate.Application.WorkspacePath,
		ApplicationName:          aggregate.Application.ApplicationName,
	}
	if aggregate.Application.LastObservedConfigHash != "" {
		item.LastObservedConfigHash = stringPointer(aggregate.Application.LastObservedConfigHash)
	}
	if aggregate.Snapshot != nil {
		item.ServiceCount = aggregate.Snapshot.DeclaredServiceCount
	}
	return item
}

func toProjectLifecycleConfigurationResponse(
	aggregate projectstore.ApplicationAggregate,
) generated.ApplicationLifecycleConfigurationResponse {
	config := lifecycleConfigurationFromAggregate(aggregate)
	return generated.ApplicationLifecycleConfigurationResponse{
		ApplicationId:          aggregate.Application.ApplicationID,
		LifecycleReviewStatus:  generated.ApplicationLifecycleReviewStatus(nonEmptyString(aggregate.Application.LifecycleReviewStatus, projectcontract.LifecycleReviewStatusReviewRequired.String())),
		WorkspacePath:          aggregate.Application.WorkspacePath,
		ComposeProjectName:     aggregate.Application.ComposeProjectName,
		ComposeFiles:           toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())),
		LifecycleConfiguration: toGeneratedLifecycleConfiguration(config),
	}
}

func toGeneratedProjectLifecycleConfiguration(
	aggregate projectstore.ApplicationAggregate,
) generated.ApplicationLifecycleConfiguration {
	return toGeneratedLifecycleConfiguration(lifecycleConfigurationFromAggregate(aggregate))
}

// toGeneratedLifecycleConfiguration 将生命周期配置转换为生成的 API 表示形式，包括标准选项、附加参数和生成的命令。
func toGeneratedLifecycleConfiguration(config LifecycleConfiguration) generated.ApplicationLifecycleConfiguration {
	return generated.ApplicationLifecycleConfiguration{
		StrategyKind:             generated.ApplicationLifecycleStrategyKind(config.StrategyKind),
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
		AdditionalArgs:           append([]string{}, config.Standard.AdditionalArgs...),
		GeneratedCommands:        toGeneratedLifecycleCommands(config),
	}
}

func toGeneratedLifecycleCommands(config LifecycleConfiguration) struct {
	Redeploy generated.ApplicationLifecycleGeneratedCommand `json:"redeploy"`
	Restart  generated.ApplicationLifecycleGeneratedCommand `json:"restart"`
	Stop     generated.ApplicationLifecycleGeneratedCommand `json:"stop"`
	Up       generated.ApplicationLifecycleGeneratedCommand `json:"up"`
} {
	return struct {
		Redeploy generated.ApplicationLifecycleGeneratedCommand `json:"redeploy"`
		Restart  generated.ApplicationLifecycleGeneratedCommand `json:"restart"`
		Stop     generated.ApplicationLifecycleGeneratedCommand `json:"stop"`
		Up       generated.ApplicationLifecycleGeneratedCommand `json:"up"`
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
) generated.ApplicationLifecycleGeneratedCommand {
	steps := buildLifecycleCommandSteps(config, action)
	displayParts := make([]string, 0, len(steps))
	for _, item := range steps {
		displayParts = append(displayParts, item.DisplayCommand)
	}
	return generated.ApplicationLifecycleGeneratedCommand{
		Action:         generated.ApplicationLifecycleGeneratedCommandAction(action),
		Steps:          steps,
		DisplayCommand: strings.Join(displayParts, "\n"),
	}
}

func buildLifecycleCommandSteps(
	config LifecycleConfiguration,
	action string,
) []generated.ApplicationLifecycleCommandStep {
	base := buildLifecycleBaseArgv(config)
	switch action {
	case "up":
		return []generated.ApplicationLifecycleCommandStep{buildLifecycleCommandStep("up", buildLifecycleUpArgv(base, config.Standard))}
	case "stop":
		return []generated.ApplicationLifecycleCommandStep{buildLifecycleCommandStep("stop", append(base, "stop"))}
	case "restart":
		return []generated.ApplicationLifecycleCommandStep{buildLifecycleCommandStep("restart", append(base, "restart"))}
	case "redeploy":
		steps := make([]generated.ApplicationLifecycleCommandStep, 0, lifecycleRedeployStepCap)
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
		return []generated.ApplicationLifecycleCommandStep{buildLifecycleCommandStep("up", buildLifecycleUpArgv(base, config.Standard))}
	}
}

// buildLifecycleBaseArgv 构建包含 Compose 文件、配置文件和项目名称的 Docker Compose 基础命令参数。
func buildLifecycleBaseArgv(config LifecycleConfiguration) []string {
	base := []string{"docker", "compose"}
	for _, file := range config.ComposeFiles {
		base = append(base, "-f", file)
	}
	for _, profile := range config.Standard.Profiles {
		base = append(base, "--profile", profile)
	}
	if strings.TrimSpace(config.ApplicationName) != "" {
		base = append(base, "-p", config.ApplicationName)
	}
	return base
}

// buildLifecycleUpArgv 构建用于后台启动 Compose 服务的命令参数列表，并包含配置的启动选项及附加参数。
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
	args = append(args, standard.AdditionalArgs...)
	return args
}

func buildLifecycleCommandStep(kind string, argv []string) generated.ApplicationLifecycleCommandStep {
	return generated.ApplicationLifecycleCommandStep{
		Kind:           generated.ApplicationLifecycleCommandStepKind(kind),
		Argv:           append([]string(nil), argv...),
		DisplayCommand: strings.Join(argv, " "),
	}
}

// toGeneratedFiles 将存储的文件记录转换为生成的文件项列表。
func toGeneratedFiles(files []projectstore.ApplicationFile) []generated.ApplicationFileItem {
	items := make([]generated.ApplicationFileItem, 0, len(files))
	for _, item := range files {
		items = append(items, generated.ApplicationFileItem{
			Id:               mustGeneratedID(item.ID),
			Kind:             generated.ApplicationFileKind(item.Kind),
			Role:             generated.ApplicationFileRole(item.Role),
			AbsolutePath:     item.AbsolutePath,
			DisplayPath:      item.DisplayPath,
			OrderIndex:       item.OrderIndex,
			LastObservedHash: optionalString(item.LastObservedHash),
		})
	}
	return items
}

// toGeneratedFilesFromCompose 将 Compose 投影文件转换为生成的项目文件项列表。
func toGeneratedFilesFromCompose(files []projectcompose.FileProjection) []generated.ApplicationFileItem {
	items := make([]generated.ApplicationFileItem, 0, len(files))
	for index, item := range files {
		hash := item.Hash
		items = append(items, generated.ApplicationFileItem{
			Id:               int64(index + 1),
			Kind:             generated.ApplicationFileKind(item.Kind),
			Role:             generated.ApplicationFileRole(item.Role),
			AbsolutePath:     item.AbsolutePath,
			DisplayPath:      item.DisplayPath,
			OrderIndex:       item.OrderIndex,
			LastObservedHash: &hash,
		})
	}
	return items
}

// toStoreFiles 将 compose 和 env 文件投影转换为存储层文件记录。
func toStoreFiles(composeFiles []projectcompose.FileProjection, envFiles []projectcompose.FileProjection) []projectstore.ApplicationFile {
	items := make([]projectstore.ApplicationFile, 0, len(composeFiles)+len(envFiles))
	for _, item := range append(append([]projectcompose.FileProjection(nil), composeFiles...), envFiles...) {
		items = append(items, projectstore.ApplicationFile{
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

// filterFiles 按文件类型筛选记录，并按 OrderIndex、ID 稳定排序，以保持 API 文件列表与 Compose 参数顺序一致。
func filterFiles(files []projectstore.ApplicationFile, kind string) []projectstore.ApplicationFile {
	items := make([]projectstore.ApplicationFile, 0)
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
func collectFilesByKind(files []projectstore.ApplicationFile, kind string) []string {
	filtered := filterFiles(files, kind)
	paths := make([]string, 0, len(filtered))
	for _, item := range filtered {
		paths = append(paths, item.AbsolutePath)
	}
	return paths
}

func (s *Service) loadFromAggregate(aggregate projectstore.ApplicationAggregate) (projectcompose.Result, error) {
	// Compose 解析只消费聚合中的静态文件投影，不读取运行时容器状态，保证刷新结果可重现。
	return projectcompose.Load(projectcompose.Input{
		WorkspacePath: aggregate.Application.WorkspacePath,
		ComposeFiles:  collectFilesByKind(aggregate.Files, projectcontract.FileKindCompose.String()),
		EnvFiles:      collectFilesByKind(aggregate.Files, projectcontract.FileKindEnv.String()),
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

// yamlJSONRoundTrip 将 JSON 数据解析到目标值，并返回解析错误。
func yamlJSONRoundTrip(raw []byte, target any) error {
	return json.Unmarshal(raw, target)
}

// digestServiceNames 按字典序排列服务名后计算稳定的 SHA-256 十六进制摘要。
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

// buildRefreshApplicationInput 构建项目刷新持久化输入，记录配置文件、归一化快照、服务摘要、干净的漂移状态、配置哈希、刷新时间和操作者。
func buildRefreshApplicationInput(
	projectID uint64,
	parseResult projectcompose.Result,
	now time.Time,
	actorID *uint64,
) projectstore.RefreshApplicationInput {
	return projectstore.RefreshApplicationInput{
		ApplicationRecordID:    projectID,
		LastObservedConfigHash: parseResult.ConfigHash,
		LastDriftCheckedAt:     &now,
		DriftStatus:            projectcontract.DriftStatusClean.String(),
		Files:                  toStoreFiles(parseResult.ComposeFiles, parseResult.EnvFiles),
		Snapshot: &projectstore.Snapshot{
			ApplicationRecordID:    projectID,
			ConfigHash:             parseResult.ConfigHash,
			NormalizedComposeJSON:  normalizeSnapshotJSON(parseResult.NormalizedComposeJSON),
			DeclaredServiceCount:   len(parseResult.ServiceNames),
			DeclaredServicesDigest: digestServiceNames(parseResult.ServiceNames),
			RefreshedAt:            now,
		},
		ActorID: actorID,
	}
}

// hashString 返回输入文本归一化后的 SHA-256 十六进制摘要。
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

// nonEmptyString 返回去除首尾空白后的 primary；若其为空则返回 fallback。
func nonEmptyString(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

// stringPointer 返回去除首尾空白后的字符串指针；输入为空白时返回 nil。
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

// mapManagedCreationMethodAvailability 将托管根状态映射为创建方式可用性。
func mapManagedCreationMethodAvailability(status string) string {
	switch strings.TrimSpace(status) {
	case projectcontract.ManagedRootStatusReady.String():
		return "ready"
	case projectcontract.ManagedRootStatusUnconfigured.String(), projectcontract.ManagedRootStatusInvalid.String():
		return "blocked"
	default:
		return "blocked"
	}
}

// managedRootCreationBlockedReason 将托管根状态映射为稳定的创建方式阻塞原因代码。
// 就绪状态返回 nil；未配置、无效或未知状态分别返回对应的原因代码指针。
func managedRootCreationBlockedReason(status string) *string {
	switch strings.TrimSpace(status) {
	case projectcontract.ManagedRootStatusReady.String():
		return nil
	case projectcontract.ManagedRootStatusUnconfigured.String():
		return stringPointer("managed_root_unconfigured")
	case projectcontract.ManagedRootStatusInvalid.String():
		return stringPointer("managed_root_invalid")
	default:
		return stringPointer("managed_root_unknown")
	}
}

// toGeneratedSourceMetadata 将已知来源元数据字段转换为生成的项目来源元数据；没有可映射字段时返回 nil。
func toGeneratedSourceMetadata(metadata map[string]string) *generated.ApplicationSourceMetadata {
	if len(metadata) == 0 {
		return nil
	}
	result := generated.ApplicationSourceMetadata{}
	assignSourceMetadataField(metadata, "managed_root_key", &result.ManagedRootKey)
	assignSourceMetadataField(metadata, "managed_relative_directory", &result.ManagedRelativeDirectory)
	assignSourceMetadataField(metadata, "managed_compose_file_name", &result.ManagedComposeFileName)
	assignSourceMetadataField(metadata, "managed_env_file_name", &result.ManagedEnvFileName)
	assignSourceMetadataField(metadata, "template_key", &result.TemplateKey)
	assignSourceMetadataField(metadata, "template_version", &result.TemplateVersion)
	assignSourceMetadataField(metadata, "template_instance_name", &result.TemplateInstanceName)
	if result == (generated.ApplicationSourceMetadata{}) {
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

// buildListSourceMetadataWithManagedRoot 构建项目列表来源元数据，优先使用已存储字段，并在托管来源缺少字段时根据托管根目录补充。
func buildListSourceMetadataWithManagedRoot(aggregate projectstore.ApplicationAggregate, managedRootDirectory string) *generated.ApplicationSourceMetadata {
	if metadata := toGeneratedSourceMetadata(aggregate.Application.SourceMetadata); metadata != nil {
		return metadata
	}
	switch strings.TrimSpace(aggregate.Application.SourceType) {
	case projectcontract.SourceTypeManaged.String():
		return buildManagedSourceMetadata(aggregate, managedRootDirectory)
	default:
		return nil
	}
}

// buildDetailSourceMetadataWithManagedRoot 构建项目详情来源元数据，优先使用已存储来源信息，并为托管来源补充托管根目录信息；无法生成时返回 nil。
func buildDetailSourceMetadataWithManagedRoot(aggregate projectstore.ApplicationAggregate, managedRootDirectory string) *generated.ApplicationSourceMetadata {
	if metadata := toGeneratedSourceMetadata(aggregate.Application.SourceMetadata); metadata != nil {
		return metadata
	}
	switch strings.TrimSpace(aggregate.Application.SourceType) {
	case projectcontract.SourceTypeManaged.String():
		return buildManagedSourceMetadata(aggregate, managedRootDirectory)
	default:
		return nil
	}
}

// buildManagedSourceMetadata 为托管项目构建来源元数据，包含托管根标识、相对目录及已登记的 Compose 和环境文件名。
func buildManagedSourceMetadata(aggregate projectstore.ApplicationAggregate, managedRootDirectory string) *generated.ApplicationSourceMetadata {
	composeFiles := filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())
	envFiles := filterFiles(aggregate.Files, projectcontract.FileKindEnv.String())
	metadata := map[string]string{
		"managed_root_key": projectcontract.ApplicationRootDirectoryConfig.String(),
	}
	if relativePath := deriveManagedRelativeDirectory(managedRootDirectory, aggregate.Application.WorkspacePath); relativePath != "" {
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

// resolveActivityAuthority 返回项目活动使用的权威路径，固定为容器所有者前端扇出路径。
func resolveActivityAuthority() ActivityAuthority {
	return ApplicationActivityAuthorityFrontendFanout
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
// 规范后的值在输入小于等于 0 时取默认值，超过最大值时取最大值，否则保留原值。
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
