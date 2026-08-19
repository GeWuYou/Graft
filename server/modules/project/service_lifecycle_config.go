package project

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

// defaultLifecycleStandardConfig 返回预设的标准生命周期配置。
func defaultLifecycleStandardConfig() LifecycleStandardConfig {
	return LifecycleStandardConfig{
		Profiles:                 []string{},
		ManagedServiceNames:      []string{},
		DownBeforeRedeploy:       true,
		PullBeforeRedeploy:       false,
		BuildBeforeUp:            false,
		ForceRecreate:            false,
		RemoveOrphans:            true,
		WaitAfterUp:              false,
		WaitTimeoutSeconds:       defaultLifecycleWaitTimeoutSeconds,
		RenewAnonVolumes:         false,
		PruneImagesAfterRedeploy: false,
		AdditionalArgs:           []string{},
		StopArgs:                 []string{},
		RestartArgs:              []string{},
		PullArgs:                 []string{},
	}
}

// lifecycleSeedForManagedApplication 返回受管项目在存储层使用的默认生命周期配置。
func lifecycleSeedForManagedApplication() projectstore.LifecycleConfig {
	return toStoreLifecycleConfig(defaultLifecycleStandardConfig())
}

// lifecycleConfigurationFromAggregate 从项目聚合体构建包含项目基本信息及标准生命周期配置的 LifecycleConfiguration。
func lifecycleConfigurationFromAggregate(aggregate projectstore.ApplicationAggregate) LifecycleConfiguration {
	return LifecycleConfiguration{
		StrategyKind: LifecycleStrategyKind(nonEmptyString(
			aggregate.Application.LifecycleStrategyKind,
			projectcontract.LifecycleStrategyKindStandard.String(),
		)),
		ReviewStatus: LifecycleReviewStatus(nonEmptyString(
			aggregate.Application.LifecycleReviewStatus,
			projectcontract.LifecycleReviewStatusReviewRequired.String(),
		)),
		WorkingDir:           aggregate.Application.WorkspacePath,
		ComposeFiles:         collectFilesByKind(aggregate.Files, projectcontract.FileKindCompose.String()),
		ApplicationName:      aggregate.Application.ComposeProjectName,
		DeclaredServiceCount: declaredServiceCount(aggregate),
		Standard: LifecycleStandardConfig{
			Profiles:                 append([]string(nil), aggregate.Application.LifecycleConfig.Profiles...),
			ManagedServiceNames:      append([]string(nil), aggregate.Application.LifecycleConfig.ManagedServiceNames...),
			DownBeforeRedeploy:       aggregate.Application.LifecycleConfig.DownBeforeRedeploy,
			PullBeforeRedeploy:       aggregate.Application.LifecycleConfig.PullBeforeRedeploy,
			BuildBeforeUp:            aggregate.Application.LifecycleConfig.BuildBeforeUp,
			ForceRecreate:            aggregate.Application.LifecycleConfig.ForceRecreate,
			RemoveOrphans:            aggregate.Application.LifecycleConfig.RemoveOrphans,
			WaitAfterUp:              aggregate.Application.LifecycleConfig.WaitAfterUp,
			WaitTimeoutSeconds:       aggregate.Application.LifecycleConfig.WaitTimeoutSeconds,
			RenewAnonVolumes:         aggregate.Application.LifecycleConfig.RenewAnonVolumes,
			PruneImagesAfterRedeploy: aggregate.Application.LifecycleConfig.PruneImagesAfterRedeploy,
			AdditionalArgs:           append([]string(nil), aggregate.Application.LifecycleConfig.AdditionalArgs...),
			StopArgs:                 append([]string(nil), aggregate.Application.LifecycleConfig.StopArgs...),
			RestartArgs:              append([]string(nil), aggregate.Application.LifecycleConfig.RestartArgs...),
			PullArgs:                 append([]string(nil), aggregate.Application.LifecycleConfig.PullArgs...),
		},
	}
}

func declaredServiceCount(aggregate projectstore.ApplicationAggregate) int {
	if aggregate.Snapshot == nil {
		return 0
	}
	return aggregate.Snapshot.DeclaredServiceCount
}

// toStoreLifecycleConfig 将标准生命周期配置转换为存储层表示。
func toStoreLifecycleConfig(config LifecycleStandardConfig) projectstore.LifecycleConfig {
	return projectstore.LifecycleConfig{
		Profiles:                 append([]string(nil), config.Profiles...),
		ManagedServiceNames:      append([]string(nil), config.ManagedServiceNames...),
		DownBeforeRedeploy:       config.DownBeforeRedeploy,
		PullBeforeRedeploy:       config.PullBeforeRedeploy,
		BuildBeforeUp:            config.BuildBeforeUp,
		ForceRecreate:            config.ForceRecreate,
		RemoveOrphans:            config.RemoveOrphans,
		WaitAfterUp:              config.WaitAfterUp,
		WaitTimeoutSeconds:       config.WaitTimeoutSeconds,
		RenewAnonVolumes:         config.RenewAnonVolumes,
		PruneImagesAfterRedeploy: config.PruneImagesAfterRedeploy,
		AdditionalArgs:           append([]string(nil), config.AdditionalArgs...),
		StopArgs:                 append([]string(nil), config.StopArgs...),
		RestartArgs:              append([]string(nil), config.RestartArgs...),
		PullArgs:                 append([]string(nil), config.PullArgs...),
	}
}

// normalizeLifecycleStandardConfig 裁剪并去重配置档案，应用默认等待超时并校验最终配置。
// normalizeLifecycleStandardConfig 规范化并校验标准生命周期配置，包括配置档案、附加参数和等待超时时间。
// 返回规范化后的配置；当配置包含无效配置档案、附加参数或超出允许范围的等待超时时间时，返回错误。
func normalizeLifecycleStandardConfig(config LifecycleStandardConfig) (LifecycleStandardConfig, error) {
	normalizedProfiles := make([]string, 0, len(config.Profiles))
	seen := make(map[string]struct{}, len(config.Profiles))
	for _, item := range config.Profiles {
		profile := strings.TrimSpace(item)
		if profile == "" {
			return LifecycleStandardConfig{}, errProjectInvalidArgument
		}
		if _, exists := seen[profile]; exists {
			continue
		}
		seen[profile] = struct{}{}
		normalizedProfiles = append(normalizedProfiles, profile)
	}
	normalizedAdditionalArgs, err := normalizeLifecycleAdditionalArgs(config.AdditionalArgs)
	if err != nil {
		return LifecycleStandardConfig{}, err
	}
	normalizedManagedServices, err := normalizeManagedServiceNames(config.ManagedServiceNames)
	if err != nil {
		return LifecycleStandardConfig{}, err
	}
	normalizedStopArgs, err := normalizeLifecycleActionArgs("stop", config.StopArgs)
	if err != nil {
		return LifecycleStandardConfig{}, err
	}
	normalizedRestartArgs, err := normalizeLifecycleActionArgs("restart", config.RestartArgs)
	if err != nil {
		return LifecycleStandardConfig{}, err
	}
	normalizedPullArgs, err := normalizeLifecycleActionArgs("pull", config.PullArgs)
	if err != nil {
		return LifecycleStandardConfig{}, err
	}
	return LifecycleStandardConfig{
		Profiles:                 normalizedProfiles,
		ManagedServiceNames:      normalizedManagedServices,
		DownBeforeRedeploy:       config.DownBeforeRedeploy,
		PullBeforeRedeploy:       config.PullBeforeRedeploy,
		BuildBeforeUp:            config.BuildBeforeUp,
		ForceRecreate:            config.ForceRecreate,
		RemoveOrphans:            config.RemoveOrphans,
		WaitAfterUp:              config.WaitAfterUp,
		WaitTimeoutSeconds:       normalizeLifecycleWaitTimeout(config.WaitTimeoutSeconds),
		RenewAnonVolumes:         config.RenewAnonVolumes,
		PruneImagesAfterRedeploy: config.PruneImagesAfterRedeploy,
		AdditionalArgs:           normalizedAdditionalArgs,
		StopArgs:                 normalizedStopArgs,
		RestartArgs:              normalizedRestartArgs,
		PullArgs:                 normalizedPullArgs,
	}, validateLifecycleWaitTimeout(config.WaitTimeoutSeconds)
}

// normalizeLifecycleWaitTimeout 将零值归一化为默认的生命周期等待超时时间。
func normalizeLifecycleWaitTimeout(value int) int {
	if value == 0 {
		return defaultLifecycleWaitTimeoutSeconds
	}
	return value
}

// validateLifecycleWaitTimeout 验证生命周期等待超时是否在允许的范围内。
// validateLifecycleWaitTimeout 验证生命周期等待超时时间是否在允许范围内；值为 0 时使用默认超时时间。
func validateLifecycleWaitTimeout(value int) error {
	timeout := normalizeLifecycleWaitTimeout(value)
	if timeout < minLifecycleWaitTimeoutSeconds || timeout > maxLifecycleWaitTimeoutSeconds {
		return errProjectInvalidArgument
	}
	return nil
}

// 它会裁剪参数首尾空白，并拒绝数量、长度、字符内容或权限覆盖相关约束不符合要求的参数。
func normalizeLifecycleAdditionalArgs(values []string) ([]string, error) {
	normalized, valid := projectcontract.NormalizeLifecycleAdditionalArgs(values)
	if !valid {
		return nil, errProjectInvalidArgument
	}
	for _, argument := range normalized {
		if isLifecycleAuthorityOverrideArg(argument) {
			return nil, errProjectInvalidArgument
		}
	}
	return normalized, nil
}

func normalizeLifecycleActionArgs(action string, values []string) ([]string, error) {
	normalized, valid := projectcontract.NormalizeLifecycleActionArgs(action, values)
	if !valid {
		return nil, errProjectInvalidArgument
	}
	return normalized, nil
}

func normalizeManagedServiceNames(values []string) ([]string, error) {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value)
		if name == "" {
			return nil, errProjectInvalidArgument
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	slices.Sort(normalized)
	return normalized, nil
}

func validateManagedServiceSelection(selected []string, declared []string, required bool) error {
	if len(selected) == 0 {
		if required {
			return errProjectInvalidArgument
		}
		return nil
	}
	declaredSet := make(map[string]struct{}, len(declared))
	for _, name := range declared {
		declaredSet[name] = struct{}{}
	}
	for _, name := range selected {
		if _, exists := declaredSet[name]; !exists {
			return errProjectInvalidArgument
		}
	}
	return nil
}

func declaredServiceNamesFromAggregate(aggregate projectstore.ApplicationAggregate) []string {
	if aggregate.Snapshot == nil || len(aggregate.Snapshot.NormalizedComposeJSON) == 0 {
		return nil
	}
	var document struct {
		Services map[string]json.RawMessage `json:"services"`
	}
	if json.Unmarshal(aggregate.Snapshot.NormalizedComposeJSON, &document) != nil {
		return nil
	}
	names := make([]string, 0, len(document.Services))
	for name := range document.Services {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// isLifecycleAuthorityOverrideArg 判断参数是否为禁止覆盖项目权威配置的选项。
func isLifecycleAuthorityOverrideArg(argument string) bool {
	forbidden := []string{
		"-f",
		"--file",
		"-p",
		"--project-name",
		"--project-directory",
		"--env-file",
		"--profile",
	}
	for _, prefix := range forbidden {
		if argument == prefix || strings.HasPrefix(argument, prefix+"=") {
			return true
		}
	}
	return argument == "--"
}

// UpdateLifecycleConfiguration 保存并确认一个项目的标准 Compose 生命周期配置。
func (s *Service) UpdateLifecycleConfiguration(
	ctx context.Context,
	projectID uint64,
	config LifecycleStandardConfig,
	actorID *uint64,
) (projectstore.ApplicationAggregate, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return projectstore.ApplicationAggregate{}, err
	}
	normalized, err := normalizeLifecycleStandardConfig(config)
	if err != nil {
		return projectstore.ApplicationAggregate{}, err
	}
	if len(normalized.ManagedServiceNames) > 0 {
		current, getErr := repository.Get(ctx, projectID)
		if getErr != nil {
			return projectstore.ApplicationAggregate{}, mapStoreError(getErr)
		}
		if err := validateManagedServiceSelection(normalized.ManagedServiceNames, declaredServiceNamesFromAggregate(current), false); err != nil {
			return projectstore.ApplicationAggregate{}, err
		}
	}
	aggregate, err := repository.UpdateLifecycleConfig(ctx, projectstore.UpdateLifecycleConfigInput{
		ApplicationRecordID:   projectID,
		LifecycleStrategyKind: projectcontract.LifecycleStrategyKindStandard.String(),
		LifecycleReviewStatus: projectcontract.LifecycleReviewStatusConfirmed.String(),
		LifecycleConfig:       toStoreLifecycleConfig(normalized),
		ActorID:               actorID,
	})
	if err != nil {
		return projectstore.ApplicationAggregate{}, mapStoreError(err)
	}
	return aggregate, nil
}

func lifecycleReviewGuard(aggregate projectstore.ApplicationAggregate) error {
	if strings.TrimSpace(aggregate.Application.LifecycleReviewStatus) != projectcontract.LifecycleReviewStatusConfirmed.String() {
		return errProjectLifecycleReview
	}
	return nil
}
