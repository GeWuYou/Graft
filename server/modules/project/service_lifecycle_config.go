package project

import (
	"context"
	"strings"

	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

func defaultLifecycleStandardConfig() LifecycleStandardConfig {
	return LifecycleStandardConfig{
		Profiles:                 []string{},
		DownBeforeRedeploy:       true,
		PullBeforeRedeploy:       false,
		BuildBeforeUp:            false,
		ForceRecreate:            false,
		WaitAfterUp:              false,
		PruneImagesAfterRedeploy: false,
	}
}

func lifecycleSeedForImportedProject() projectstore.LifecycleConfig {
	return toStoreLifecycleConfig(defaultLifecycleStandardConfig())
}

func lifecycleSeedForManagedProject() projectstore.LifecycleConfig {
	return toStoreLifecycleConfig(defaultLifecycleStandardConfig())
}

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
			WaitAfterUp:              aggregate.Project.LifecycleConfig.WaitAfterUp,
			PruneImagesAfterRedeploy: aggregate.Project.LifecycleConfig.PruneImagesAfterRedeploy,
		},
	}
}

func toStoreLifecycleConfig(config LifecycleStandardConfig) projectstore.LifecycleConfig {
	return projectstore.LifecycleConfig{
		Profiles:                 append([]string(nil), config.Profiles...),
		DownBeforeRedeploy:       config.DownBeforeRedeploy,
		PullBeforeRedeploy:       config.PullBeforeRedeploy,
		BuildBeforeUp:            config.BuildBeforeUp,
		ForceRecreate:            config.ForceRecreate,
		WaitAfterUp:              config.WaitAfterUp,
		PruneImagesAfterRedeploy: config.PruneImagesAfterRedeploy,
	}
}

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
	return LifecycleStandardConfig{
		Profiles:                 normalizedProfiles,
		DownBeforeRedeploy:       config.DownBeforeRedeploy,
		PullBeforeRedeploy:       config.PullBeforeRedeploy,
		BuildBeforeUp:            config.BuildBeforeUp,
		ForceRecreate:            config.ForceRecreate,
		WaitAfterUp:              config.WaitAfterUp,
		PruneImagesAfterRedeploy: config.PruneImagesAfterRedeploy,
	}, nil
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
