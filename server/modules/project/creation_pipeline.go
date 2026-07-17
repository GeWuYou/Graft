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

// CreationCommand 是来源适配器交给注册流水线的中立输入。
// 来源适配器必须先解析或物化工作区；该流水线不执行 Compose 生命周期命令。
type CreationCommand struct {
	DisplayName              string
	ComposeProjectName       string
	ComposeProjectNameSource string
	SourceType               string
	WorkspacePath            string
	OwnershipMode            string
	SourceMetadata           map[string]string
	LifecycleConfig          LifecycleStandardConfig
	ParseResult              projectcompose.Result
	ActorID                  *uint64
	RuntimeTargetID          uint64
	ApplicationName          *string
}

// createProjectFromWorkspace 在来源生成并成功解析实际工作区后持久化通用项目聚合。
func (s *Service) createProjectFromWorkspace(ctx context.Context, command CreationCommand) (projectstore.ApplicationAggregate, time.Time, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return projectstore.ApplicationAggregate{}, time.Time{}, err
	}
	if strings.TrimSpace(command.WorkspacePath) == "" || strings.TrimSpace(command.ParseResult.ConfigHash) == "" {
		return projectstore.ApplicationAggregate{}, time.Time{}, errProjectInvalidArgument
	}
	lifecycle, err := normalizeLifecycleStandardConfig(command.LifecycleConfig)
	if err != nil {
		return projectstore.ApplicationAggregate{}, time.Time{}, err
	}
	now := time.Now().UTC()
	targetID, err := s.resolveComposeRuntimeTarget(ctx, command.RuntimeTargetID)
	if err != nil {
		return projectstore.ApplicationAggregate{}, time.Time{}, err
	}
	if err := s.ensureComposeProjectNameAvailableForCreate(ctx, targetID, command.ComposeProjectName); err != nil {
		return projectstore.ApplicationAggregate{}, time.Time{}, err
	}
	strictCreate := command.SourceType == projectcontract.SourceTypeManaged.String() || command.SourceType == projectcontract.SourceTypeTemplate.String()
	aggregate, err := repository.ImportApplication(ctx, projectstore.ImportApplicationInput{
		ApplicationID:            newApplicationID(),
		DeploymentAdapterKind:    projectcontract.DeploymentAdapterKindCompose.String(),
		ApplicationName:          command.ApplicationName,
		WorkspacePath:            strings.TrimSpace(command.WorkspacePath),
		ComposeProjectName:       strings.TrimSpace(command.ComposeProjectName),
		ComposeProjectNameSource: composeProjectNameSource(command.ComposeProjectNameSource),
		StrictCreate:             strictCreate,
		DisplayName:              strings.TrimSpace(command.DisplayName),
		SourceType:               strings.TrimSpace(command.SourceType),
		OwnershipMode:            strings.TrimSpace(command.OwnershipMode),
		SourceMetadata:           command.SourceMetadata,
		LifecycleStrategyKind:    projectcontract.LifecycleStrategyKindStandard.String(),
		LifecycleReviewStatus:    projectcontract.LifecycleReviewStatusConfirmed.String(),
		LifecycleConfig:          toStoreLifecycleConfig(lifecycle),
		LastObservedConfigHash:   command.ParseResult.ConfigHash,
		LastDriftCheckedAt:       &now,
		DriftStatus:              projectcontract.DriftStatusClean.String(),
		Files:                    toStoreFiles(command.ParseResult.ComposeFiles, command.ParseResult.EnvFiles),
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
		return projectstore.ApplicationAggregate{}, time.Time{}, mapStoreError(err)
	}
	return aggregate, now, nil
}

// ensureComposeProjectNameAvailableForCreate 防止受管工作区占用其它 Compose 项目已拥有的运行时名称。
// 目标暂不可用时有意不阻断创建，使用户可以在目标恢复前准备工作区。
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

// composeProjectNameSource 将 Compose 名称来源收敛到公开契约值。
func composeProjectNameSource(value string) string {
	if strings.TrimSpace(value) == projectcontract.ComposeProjectNameSourceOverride.String() {
		return projectcontract.ComposeProjectNameSourceOverride.String()
	}
	return projectcontract.ComposeProjectNameSourceComputed.String()
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
		return lifecycleStandardConfigFromStore(lifecycleSeedForManagedApplication())
	}
	return *config
}

// lifecycleStandardConfigFromStore 将存储层生命周期配置转换为领域层标准生命周期配置。
func lifecycleStandardConfigFromStore(config projectstore.LifecycleConfig) LifecycleStandardConfig {
	return LifecycleStandardConfig{Profiles: append([]string(nil), config.Profiles...), DownBeforeRedeploy: config.DownBeforeRedeploy, PullBeforeRedeploy: config.PullBeforeRedeploy, BuildBeforeUp: config.BuildBeforeUp, ForceRecreate: config.ForceRecreate, RemoveOrphans: config.RemoveOrphans, WaitAfterUp: config.WaitAfterUp, WaitTimeoutSeconds: config.WaitTimeoutSeconds, RenewAnonVolumes: config.RenewAnonVolumes, PruneImagesAfterRedeploy: config.PruneImagesAfterRedeploy, AdditionalArgs: append([]string(nil), config.AdditionalArgs...)}
}

// managedCreationCommand 根据已验证的项目数据、规范化请求和解析结果构建受管项目的创建命令。
func managedCreationCommand(validation ManagedApplicationCreateValidationResult, normalized normalizedManagedCreateRequest, parseResult projectcompose.Result, actorID *uint64) CreationCommand {
	return CreationCommand{DisplayName: normalized.DisplayName, ComposeProjectName: validation.ComposeProjectName, ComposeProjectNameSource: projectcontract.ComposeProjectNameSourceComputed.String(), SourceType: validation.SourceType, WorkspacePath: validation.WorkspacePath, OwnershipMode: projectcontract.OwnershipModeManagedRootDedicated.String(), SourceMetadata: validation.SourceMetadata, LifecycleConfig: defaultManagedLifecycleConfig(normalized.LifecycleConfig), ParseResult: parseResult, ActorID: actorID, RuntimeTargetID: normalized.RuntimeTargetID, ApplicationName: validation.ApplicationName}
}
