package project

import (
	"context"
	"errors"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
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
	RuntimeTargetID            uint64
	ApplicationName               *string
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
	targetID, err := s.resolveComposeRuntimeTarget(ctx, command.RuntimeTargetID)
	if err != nil {
		return projectstore.ProjectAggregate{}, time.Time{}, err
	}
	if err := s.ensureComposeProjectNameAvailableForCreate(ctx, targetID, command.CanonicalProjectName); err != nil {
		return projectstore.ProjectAggregate{}, time.Time{}, err
	}
	strictCreate := command.SourceKind == projectcontract.SourceKindManaged.String() || command.SourceKind == projectcontract.SourceKindTemplate.String()
	aggregate, err := repository.ImportProject(ctx, projectstore.ImportProjectInput{
		ApplicationID:              newApplicationID(),
		ApplicationName:               command.ApplicationName,
		WorkspacePath:              strings.TrimSpace(command.WorkingDirectory),
		ComposeProjectName:         strings.TrimSpace(command.CanonicalProjectName),
		ComposeProjectNameSource:   composeProjectNameSource(command.CanonicalProjectNameSource),
		StrictCreate:               strictCreate,
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
		ActorID:         command.ActorID,
		RuntimeTargetID: targetID,
	})
	if err != nil {
		return projectstore.ProjectAggregate{}, time.Time{}, mapStoreError(err)
	}
	return aggregate, now, nil
}

// ensureComposeProjectNameAvailableForCreate prevents a managed workspace from
// claiming a runtime name already owned by another Compose project. An
// unavailable target is intentionally non-blocking so users can prepare a
// workspace before the target recovers.
func (s *Service) ensureComposeProjectNameAvailableForCreate(ctx context.Context, targetID uint64, name string) error {
	state := s.composeProjectNameState(ctx, targetID, name)
	switch state {
	case moduleapi.ComposeProjectNameStateOccupied:
		return errors.Join(errProjectConflict, errProjectComposeNameOccupied)
	case moduleapi.ComposeProjectNameStateAvailable, moduleapi.ComposeProjectNameStateUnavailable:
		return nil
	default:
		return errProjectRuntimeUnavailable
	}
}

func (s *Service) composeProjectNameState(ctx context.Context, targetID uint64, name string) moduleapi.ComposeProjectNameState {
	if s == nil || s.runtimeTargets == nil {
		return moduleapi.ComposeProjectNameStateUnavailable
	}
	if targetID == 0 || targetID > uint64(^uint64(0)>>1) || strings.TrimSpace(name) == "" {
		return moduleapi.ComposeProjectNameStateError
	}
	availability, err := s.runtimeTargets.CheckComposeProjectName(ctx, int64(targetID), strings.TrimSpace(name))
	if err != nil {
		return moduleapi.ComposeProjectNameStateError
	}
	return availability.State
}

// composeProjectNameSource classifies a canonical Compose project name source.
// It returns "declared" for the canonical override source and "derived" for all other values.
func composeProjectNameSource(value string) string {
	if strings.TrimSpace(value) == projectcontract.CanonicalProjectNameSourceOverride.String() {
		return "declared"
	}
	return "derived"
}

func (s *Service) resolveComposeRuntimeTarget(ctx context.Context, requested uint64) (uint64, error) {
	if s == nil || s.runtimeTargets == nil {
		return 0, nil // Unit tests construct the service without module wiring.
	}
	if requested == 0 {
		targets, err := s.runtimeTargets.ListComposeTargets(ctx)
		if err != nil || len(targets) != 1 || targets[0].ID < 1 {
			return 0, errProjectInvalidArgument
		}
		return uint64(targets[0].ID), nil // #nosec G115 -- positivity is checked immediately above.
	}
	var id *int64
	if requested > uint64(^uint64(0)>>1) {
		return 0, errProjectInvalidArgument
	}
	value := int64(requested)
	id = &value
	target, err := s.runtimeTargets.ReadComposeTarget(ctx, id)
	if err != nil || target.ID < 1 {
		return 0, errProjectInvalidArgument
	}
	return uint64(target.ID), nil
}

// defaultManagedLifecycleConfig 返回用于受管项目的生命周期配置；未提供配置时使用默认配置。
func defaultManagedLifecycleConfig(config *LifecycleStandardConfig) LifecycleStandardConfig {
	if config == nil {
		return lifecycleStandardConfigFromStore(lifecycleSeedForManagedProject())
	}
	return *config
}

// lifecycleStandardConfigFromStore 将存储层生命周期配置转换为领域层标准生命周期配置。
func lifecycleStandardConfigFromStore(config projectstore.LifecycleConfig) LifecycleStandardConfig {
	return LifecycleStandardConfig{Profiles: append([]string(nil), config.Profiles...), DownBeforeRedeploy: config.DownBeforeRedeploy, PullBeforeRedeploy: config.PullBeforeRedeploy, BuildBeforeUp: config.BuildBeforeUp, ForceRecreate: config.ForceRecreate, RemoveOrphans: config.RemoveOrphans, WaitAfterUp: config.WaitAfterUp, WaitTimeoutSeconds: config.WaitTimeoutSeconds, RenewAnonVolumes: config.RenewAnonVolumes, PruneImagesAfterRedeploy: config.PruneImagesAfterRedeploy, AdditionalArgs: append([]string(nil), config.AdditionalArgs...)}
}

// managedCreationCommand 构建受管项目创建流程使用的源无关创建命令。
func managedCreationCommand(validation ManagedProjectCreateValidationResult, normalized normalizedManagedCreateRequest, parseResult projectcompose.Result, actorID *uint64) CreationCommand {
	return CreationCommand{DisplayName: normalized.DisplayName, CanonicalProjectName: validation.ComposeProjectName, CanonicalProjectNameSource: "generated", SourceKind: projectcontract.SourceKindManaged.String(), HostScope: projectcontract.HostScopeLocal.String(), WorkingDirectory: validation.WorkspacePath, OwnershipMode: projectcontract.OwnershipModeManagedRootDedicated.String(), SourceMetadata: validation.SourceMetadata, LifecycleConfig: defaultManagedLifecycleConfig(normalized.LifecycleConfig), ParseResult: parseResult, ActorID: actorID, RuntimeTargetID: normalized.RuntimeTargetID, ApplicationName: validation.ApplicationName}
}
