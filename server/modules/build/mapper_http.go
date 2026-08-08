package build

import (
	"math"

	"graft/server/internal/moduleapi"

	openapigen "graft/server/internal/contract/openapi/generated"
	buildstore "graft/server/modules/build/store"
)

func toBuildWorkspace(workspace moduleapi.BuildWorkspace) openapigen.BuildWorkspace {
	return openapigen.BuildWorkspace{WorkspaceId: workspace.ID, Name: workspace.Name, SourceKind: openapigen.BuildWorkspaceSourceKind(workspace.SourceKind), SourceReference: workspace.SourceReference, RetentionPolicy: workspace.RetentionPolicy, CreatedAt: workspace.CreatedAt, UpdatedAt: workspace.UpdatedAt}
}

func mapBuildWorkspaces(items []moduleapi.BuildWorkspace) []openapigen.BuildWorkspace {
	result := make([]openapigen.BuildWorkspace, 0, len(items))
	for _, item := range items {
		result = append(result, toBuildWorkspace(item))
	}
	return result
}

func mapBuildRuntimeTargets(items []moduleapi.BuildRuntimeTargetSummary) []openapigen.BuildRuntimeTarget {
	result := make([]openapigen.BuildRuntimeTarget, 0, len(items))
	for _, item := range items {
		result = append(result, openapigen.BuildRuntimeTarget{TargetId: item.ID, DisplayName: item.DisplayName, Provider: item.Provider, Available: item.Available, SupportedDrivers: item.SupportedDrivers, SupportedPlatforms: item.SupportedPlatforms, WorkspaceLocalities: item.WorkspaceLocalities, SnapshotDeliveryModes: item.SnapshotDeliveryModes})
	}
	return result
}

func mapBuilderPools(items []moduleapi.BuilderPool) []openapigen.BuildBuilderPool {
	result := make([]openapigen.BuildBuilderPool, 0, len(items))
	for _, item := range items {
		result = append(result, openapigen.BuildBuilderPool{PoolId: item.ID, DisplayName: item.DisplayName, SchedulingPolicy: openapigen.BuildBuilderPoolSchedulingPolicy(item.SchedulingPolicy)})
	}
	return result
}

func toBuildJobList(result buildstore.ListResult, query buildstore.ListQuery) openapigen.BuildJobList {
	items := make([]openapigen.BuildJobSummary, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toBuildJobSummary(item))
	}
	return openapigen.BuildJobList{Items: items, Total: result.Total, Limit: query.Limit, Offset: query.Offset}
}

func toBuildArtifactList(result buildstore.V2ArtifactListResult, limit, offset int) openapigen.BuildArtifactList {
	items := make([]openapigen.BuildArtifact, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, openapigen.BuildArtifact{ArtifactId: item.ArtifactID, Digest: item.Digest, MediaType: item.MediaType, Platforms: item.Platforms, SizeBytes: item.SizeBytes, CreatedAt: item.CreatedAt})
	}
	return openapigen.BuildArtifactList{Items: items, Total: result.Total, Limit: limit, Offset: offset}
}

func toBuildJobDetail(item buildstore.JobProjection) openapigen.BuildJobDetail {
	args := make([]struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}, 0, len(item.BuildArgs))
	for _, arg := range item.BuildArgs {
		args = append(args, struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}{Name: arg.Name, Value: arg.Value})
	}
	summary := toBuildJobSummary(item)
	return openapigen.BuildJobDetail{ApplicationId: summary.ApplicationId, ApplicationName: summary.ApplicationName, Artifact: summary.Artifact, BuildArgs: args, BuildId: summary.BuildId, ContextPath: summary.ContextPath, CreatedAt: summary.CreatedAt, DockerfilePath: summary.DockerfilePath, ImageRepository: summary.ImageRepository, ImageTag: summary.ImageTag, RuntimeProvider: item.RuntimeProvider, TaskId: summary.TaskId, Builder: summary.Builder, Execution: summary.Execution}
}
func toBuildJobSummary(item buildstore.JobProjection) openapigen.BuildJobSummary {
	summary := openapigen.BuildJobSummary{ApplicationId: item.ApplicationID, ApplicationName: item.ApplicationName, BuildId: item.BuildID, ContextPath: item.ContextPath, CreatedAt: item.CreatedAt, DockerfilePath: item.DockerfilePath, ImageRepository: item.ImageRepository, ImageTag: item.ImageTag, TaskId: buildJobResponseID(item.TaskID), Builder: openapigen.BuildBuilderSnapshot{Id: buildJobResponseID(item.RuntimeTargetID), Name: item.RuntimeTargetName, Provider: item.RuntimeProvider}, Execution: openapigen.BuildTaskExecution{Status: openapigen.TaskStatus(item.Execution.Status), StageCount: item.Execution.StageCount, CompletedStageCount: item.Execution.CompletedStageCount, Capabilities: openapigen.TaskCapabilities{Cancel: item.Execution.Capabilities.Cancel, Retry: item.Execution.Capabilities.Retry, DownloadLog: item.Execution.Capabilities.DownloadLog}}}
	summary.Execution.CurrentStageKey = item.Execution.CurrentStageKey
	summary.Execution.DurationMs = item.Execution.DurationMS
	summary.Execution.FailureCode = item.Execution.FailureCode
	summary.Execution.FailureMessage = item.Execution.FailureMessage
	summary.Execution.RecoveryReason = item.Execution.RecoveryReason
	if item.Artifact != nil {
		summary.Artifact = &openapigen.BuildJobArtifact{ArtifactId: item.Artifact.ArtifactID, ImageId: item.Artifact.ImageID, Repository: item.Artifact.Repository, Tag: item.Artifact.Tag}
		if item.Artifact.Digest != "" {
			summary.Artifact.Digest = &item.Artifact.Digest
		}
		summary.Artifact.SizeBytes = &item.Artifact.SizeBytes
		if item.Artifact.Platform != "" {
			summary.Artifact.Platform = &item.Artifact.Platform
		}
	}
	return summary
}

func buildJobResponseID(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}
