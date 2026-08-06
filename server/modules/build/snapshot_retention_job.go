package build

import (
	"context"
	"errors"
	"time"

	"graft/server/internal/cronx"
)

const (
	snapshotMaterializationCleanupJobName        = "build.snapshot-materialization-cleanup"
	snapshotMaterializationCleanupJobSchedule    = "0 0 * * * *"
	snapshotMaterializationCleanupJobTitleKey    = "scheduler.job.buildSnapshotMaterializationCleanup.title"
	snapshotMaterializationCleanupJobDescription = "scheduledTask.buildSnapshotMaterializationCleanup.description"
	snapshotMaterializationCleanupBatchSize      = 100
)

func registerSnapshotMaterializationCleanupJob(registry *cronx.Registry, service *Service) error {
	if registry == nil {
		return errors.New("cron registry is required")
	}
	if service == nil {
		return errors.New("build service is required")
	}

	registry.Register(cronx.Job{
		Name:           snapshotMaterializationCleanupJobName,
		Key:            snapshotMaterializationCleanupJobName,
		ModuleKey:      moduleID,
		Category:       cronx.JobCategoryRetention,
		TitleKey:       snapshotMaterializationCleanupJobTitleKey,
		DescriptionKey: snapshotMaterializationCleanupJobDescription,
		Schedule:       snapshotMaterializationCleanupJobSchedule,
		DefaultEnabled: true,
		Module:         moduleID,
		Handler: func(ctx context.Context, _ string) (cronx.JobRunResult, error) {
			purged, err := service.CleanupExpiredSnapshotMaterializations(ctx, time.Now().UTC(), snapshotMaterializationCleanupBatchSize)
			if err != nil {
				return cronx.JobRunResult{Stage: "failed", Warnings: []string{err.Error()}}, err
			}
			return cronx.JobRunResult{
				Stage:            "completed",
				AffectedResource: "build_workspace_snapshot",
				Metrics:          map[string]any{"purgedCount": purged},
			}, nil
		},
	})
	return nil
}
