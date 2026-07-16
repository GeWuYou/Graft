package project

import (
	"context"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

// ConfigurationMetadata 返回只读配置元数据；项目不存在或已删除时沿用聚合读取错误。
func (s *Service) ConfigurationMetadata(ctx context.Context, projectID uint64) (ConfigurationMetadataResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ConfigurationMetadataResult{}, err
	}
	return ConfigurationMetadataResult{
		ProjectID:          projectID,
		ComposeFiles:       toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())),
		EnvFiles:           toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindEnv.String())),
		OwnershipMode:      aggregate.Project.OwnershipMode,
		DriftStatus:        aggregate.Project.DriftStatus,
		DiagnosticsSummary: nil,
	}, nil
}

// ConfigurationPreview 返回最近一次静态规范化 Compose 配置预览，不执行生命周期动作。
func (s *Service) ConfigurationPreview(ctx context.Context, projectID uint64) (ConfigurationPreviewResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ConfigurationPreviewResult{}, err
	}
	parseResult, err := s.loadFromAggregate(aggregate)
	if err != nil {
		return ConfigurationPreviewResult{}, err
	}
	return ConfigurationPreviewResult{
		ProjectID:             projectID,
		CanonicalProjectName:  aggregate.Project.CanonicalProjectName,
		ConfigHash:            parseResult.ConfigHash,
		NormalizedComposeYAML: parseResult.NormalizedComposeYAML,
		RefreshedAt:           snapshotRefreshedAt(aggregate.Snapshot),
	}, nil
}

// DeployConfiguration 根据当前磁盘文件刷新项目状态并执行 docker compose up -d；部署前先持久化最新配置快照。
func (s *Service) DeployConfiguration(
	ctx context.Context,
	projectID uint64,
	actorID *uint64,
) (result DeployResult, err error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return DeployResult{}, err
	}
	parseResult, err := s.loadFromAggregate(aggregate)
	if err != nil {
		return DeployResult{}, err
	}
	repository, err := s.repositoryOrErr()
	if err != nil {
		return DeployResult{}, err
	}
	now := time.Now().UTC()
	updated, err := repository.RefreshProject(ctx, buildRefreshProjectInput(projectID, parseResult, now, actorID))
	if err != nil {
		return DeployResult{}, mapStoreError(err)
	}
	upArgs, err := lifecycleUpArgs(updated, lifecycleConfigurationFromAggregate(updated))
	if err != nil {
		return DeployResult{}, err
	}
	if _, err := s.executeLifecycleActionWithAggregate(ctx, updated, generated.ProjectActionResponseActionProjectActionDeploy, upArgs); err != nil {
		return DeployResult{}, err
	}
	messageKey := projectcontract.ProjectDeployCompleted.String()
	result = DeployResult{
		ProjectID:            projectID,
		Action:               "deploy",
		Result:               "completed",
		CanonicalProjectName: updated.Project.CanonicalProjectName,
		OwnershipMode:        updated.Project.OwnershipMode,
		ConfigHash:           parseResult.ConfigHash,
		RefreshedAt:          now,
		DeclaredServiceCount: len(parseResult.ServiceNames),
		MessageKey:           &messageKey,
		Message:              &messageKey,
		GuardResults: []GuardResult{
			guardDetail("command", strings.Join(upArgs, " ")),
			guardCode("saved_disk_state_used"),
			guardCode("snapshot_refreshed"),
		},
	}
	return result, nil
}

func snapshotRefreshedAt(snapshot *projectstore.Snapshot) *time.Time {
	if snapshot == nil {
		return nil
	}
	refreshedAt := snapshot.RefreshedAt
	return &refreshedAt
}
