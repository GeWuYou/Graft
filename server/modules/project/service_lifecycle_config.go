package project

import (
	"context"
	"strings"

	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

// defaultLifecycleStandardConfig 返回预设的标准生命周期配置。
func defaultLifecycleStandardConfig() LifecycleStandardConfig {
	return LifecycleStandardConfig{
		Profiles:                 []string{},
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
	}
}

// lifecycleSeedForManagedProject returns the default lifecycle configuration in store format.
func lifecycleSeedForManagedProject() projectstore.LifecycleConfig {
	return toStoreLifecycleConfig(defaultLifecycleStandardConfig())
}

// lifecycleConfigurationFromAggregate 将项目聚合体转换为生命周期配置，补充默认的策略类型和审核状态，并收集 Compose 文件。
// lifecycleConfigurationFromAggregate 从项目聚合体构建包含项目基本信息及标准生命周期配置的 LifecycleConfiguration。
func lifecycleConfigurationFromAggregate(aggregate projectstore.ProjectAggregate) LifecycleConfiguration {
	return LifecycleConfiguration{
		StrategyKind: LifecycleStrategyKind(nonEmptyString(
			aggregate.Project.LifecycleStrategyKind,
			projectcontract.LifecycleStrategyKindStandard.String(),
		)),
		ReviewStatus: LifecycleReviewStatus(nonEmptyString(
			aggregate.Project.LifecycleReviewStatus,
			projectcontract.LifecycleReviewStatusReviewRequired.String(),
		)),
		WorkingDir:   aggregate.Project.WorkingDirectory,
		ComposeFiles: collectFilesByKind(aggregate.Files, projectcontract.FileKindCompose.String()),
		ProjectName:  aggregate.Project.CanonicalProjectName,
		Standard: LifecycleStandardConfig{
			Profiles:                 append([]string(nil), aggregate.Project.LifecycleConfig.Profiles...),
			DownBeforeRedeploy:       aggregate.Project.LifecycleConfig.DownBeforeRedeploy,
			PullBeforeRedeploy:       aggregate.Project.LifecycleConfig.PullBeforeRedeploy,
			BuildBeforeUp:            aggregate.Project.LifecycleConfig.BuildBeforeUp,
			ForceRecreate:            aggregate.Project.LifecycleConfig.ForceRecreate,
			RemoveOrphans:            aggregate.Project.LifecycleConfig.RemoveOrphans,
			WaitAfterUp:              aggregate.Project.LifecycleConfig.WaitAfterUp,
			WaitTimeoutSeconds:       aggregate.Project.LifecycleConfig.WaitTimeoutSeconds,
			RenewAnonVolumes:         aggregate.Project.LifecycleConfig.RenewAnonVolumes,
			PruneImagesAfterRedeploy: aggregate.Project.LifecycleConfig.PruneImagesAfterRedeploy,
			AdditionalArgs:           append([]string(nil), aggregate.Project.LifecycleConfig.AdditionalArgs...),
		},
	}
}

// toStoreLifecycleConfig 将标准生命周期配置转换为存储层表示。
func toStoreLifecycleConfig(config LifecycleStandardConfig) projectstore.LifecycleConfig {
	return projectstore.LifecycleConfig{
		Profiles:                 append([]string(nil), config.Profiles...),
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
	}
}

// normalizeLifecycleStandardConfig trims and deduplicates profiles, applies the default wait timeout, and validates the resulting configuration.
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
	return LifecycleStandardConfig{
		Profiles:                 normalizedProfiles,
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
	const (
		maxArgs   = 32
		maxArgLen = 256
	)
	if len(values) > maxArgs {
		return nil, errProjectInvalidArgument
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		argument := strings.TrimSpace(value)
		if argument == "" || len(argument) > maxArgLen || strings.ContainsAny(argument, "\r\n\x00") || isLifecycleAuthorityOverrideArg(argument) {
			return nil, errProjectInvalidArgument
		}
		normalized = append(normalized, argument)
	}
	return normalized, nil
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

// UpdateLifecycleConfiguration saves one project's standard compose lifecycle configuration and confirms it.
func (s *Service) UpdateLifecycleConfiguration(
	ctx context.Context,
	projectID uint64,
	config LifecycleStandardConfig,
	actorID *uint64,
) (projectstore.ProjectAggregate, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return projectstore.ProjectAggregate{}, err
	}
	normalized, err := normalizeLifecycleStandardConfig(config)
	if err != nil {
		return projectstore.ProjectAggregate{}, err
	}
	aggregate, err := repository.UpdateLifecycleConfig(ctx, projectstore.UpdateLifecycleConfigInput{
		ProjectID:             projectID,
		LifecycleStrategyKind: projectcontract.LifecycleStrategyKindStandard.String(),
		LifecycleReviewStatus: projectcontract.LifecycleReviewStatusConfirmed.String(),
		LifecycleConfig:       toStoreLifecycleConfig(normalized),
		ActorID:               actorID,
	})
	if err != nil {
		return projectstore.ProjectAggregate{}, mapStoreError(err)
	}
	return aggregate, nil
}

func lifecycleReviewGuard(aggregate projectstore.ProjectAggregate) error {
	if strings.TrimSpace(aggregate.Project.LifecycleReviewStatus) != projectcontract.LifecycleReviewStatusConfirmed.String() {
		return errProjectLifecycleReview
	}
	return nil
}
