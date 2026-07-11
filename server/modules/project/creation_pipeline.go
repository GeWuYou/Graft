package project

import (
	"context"
	"strings"
	"time"

	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

// CreationCommand is the source-neutral handoff from a source adapter to
// the registry pipeline. Source adapters must resolve or materialize the workspace
// before calling it; this pipeline never runs compose lifecycle commands.
type CreationCommand struct {
	DisplayName                string
	CanonicalProjectName       string
	CanonicalProjectNameSource string
	SourceKind                 string
	HostScope                  string
	WorkingDirectory           string
	OwnershipMode              string
	SourceMetadata             map[string]string
	LifecycleConfig            LifecycleStandardConfig
	ParseResult                projectcompose.Result
	ActorID                    *uint64
}

// createProjectFromWorkspace persists the common project aggregate after a source
// has produced an actual, successfully parsed workspace.
func (s *Service) createProjectFromWorkspace(ctx context.Context, command CreationCommand) (projectstore.ProjectAggregate, time.Time, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return projectstore.ProjectAggregate{}, time.Time{}, err
	}
	if strings.TrimSpace(command.WorkingDirectory) == "" || strings.TrimSpace(command.ParseResult.ConfigHash) == "" {
		return projectstore.ProjectAggregate{}, time.Time{}, errProjectInvalidArgument
	}
	lifecycle, err := normalizeLifecycleStandardConfig(command.LifecycleConfig)
	if err != nil {
		return projectstore.ProjectAggregate{}, time.Time{}, err
	}
	now := time.Now().UTC()
	aggregate, err := repository.ImportProject(ctx, projectstore.ImportProjectInput{
		DisplayName:                strings.TrimSpace(command.DisplayName),
		CanonicalProjectName:       strings.TrimSpace(command.CanonicalProjectName),
		CanonicalProjectNameSource: strings.TrimSpace(command.CanonicalProjectNameSource),
		SourceKind:                 strings.TrimSpace(command.SourceKind),
		HostScope:                  strings.TrimSpace(command.HostScope),
		WorkingDirectory:           strings.TrimSpace(command.WorkingDirectory),
		OwnershipMode:              strings.TrimSpace(command.OwnershipMode),
		SourceMetadata:             command.SourceMetadata,
		LifecycleStrategyKind:      projectcontract.LifecycleStrategyKindStandard.String(),
		LifecycleReviewStatus:      projectcontract.LifecycleReviewStatusConfirmed.String(),
		LifecycleConfig:            toStoreLifecycleConfig(lifecycle),
		LastObservedConfigHash:     command.ParseResult.ConfigHash,
		LastDriftCheckedAt:         &now,
		DriftStatus:                projectcontract.DriftStatusClean.String(),
		Files:                      toStoreFiles(command.ParseResult.ComposeFiles, command.ParseResult.EnvFiles),
		Snapshot: &projectstore.Snapshot{
			ConfigHash:             command.ParseResult.ConfigHash,
			NormalizedComposeJSON:  normalizeSnapshotJSON(command.ParseResult.NormalizedComposeJSON),
			DeclaredServiceCount:   len(command.ParseResult.ServiceNames),
			DeclaredServicesDigest: digestServiceNames(command.ParseResult.ServiceNames),
			RefreshedAt:            now,
		},
		ActorID: command.ActorID,
	})
	if err != nil {
		return projectstore.ProjectAggregate{}, time.Time{}, mapStoreError(err)
	}
	return aggregate, now, nil
}

func defaultManagedLifecycleConfig(config *LifecycleStandardConfig) LifecycleStandardConfig {
	if config == nil {
		return lifecycleStandardConfigFromStore(lifecycleSeedForManagedProject())
	}
	return *config
}

func lifecycleStandardConfigFromStore(config projectstore.LifecycleConfig) LifecycleStandardConfig {
	return LifecycleStandardConfig{Profiles: append([]string(nil), config.Profiles...), DownBeforeRedeploy: config.DownBeforeRedeploy, PullBeforeRedeploy: config.PullBeforeRedeploy, BuildBeforeUp: config.BuildBeforeUp, ForceRecreate: config.ForceRecreate, RemoveOrphans: config.RemoveOrphans, WaitAfterUp: config.WaitAfterUp, WaitTimeoutSeconds: config.WaitTimeoutSeconds, RenewAnonVolumes: config.RenewAnonVolumes, PruneImagesAfterRedeploy: config.PruneImagesAfterRedeploy, AdditionalArgs: append([]string(nil), config.AdditionalArgs...)}
}

func managedCreationCommand(validation ManagedProjectCreateValidationResult, normalized normalizedManagedCreateRequest, parseResult projectcompose.Result, actorID *uint64) CreationCommand {
	return CreationCommand{DisplayName: normalized.DisplayName, CanonicalProjectName: normalized.CanonicalProjectName, CanonicalProjectNameSource: projectcontract.CanonicalProjectNameSourceOverride.String(), SourceKind: projectcontract.SourceKindManaged.String(), HostScope: projectcontract.HostScopeLocal.String(), WorkingDirectory: validation.WorkingDirectory, OwnershipMode: projectcontract.OwnershipModeManagedRootDedicated.String(), SourceMetadata: validation.SourceMetadata, LifecycleConfig: defaultManagedLifecycleConfig(normalized.LifecycleConfig), ParseResult: parseResult, ActorID: actorID}
}
