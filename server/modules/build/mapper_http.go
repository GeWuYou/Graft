package build

import (
	"math"

	"graft/server/internal/moduleapi"

	openapigen "graft/server/internal/contract/openapi/generated"
	buildstore "graft/server/modules/build/store"
)

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

func toBuildWorkspaceList(result buildstore.WorkspaceListResult, query buildstore.WorkspaceListQuery) openapigen.BuildWorkspaceList {
	items := make([]openapigen.BuildWorkspace, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, openapigen.BuildWorkspace{WorkspaceId: item.ID, Name: item.Name, SourceKind: openapigen.BuildWorkspaceSourceKind(item.SourceKind), SourceReference: item.SourceReference, RetentionPolicy: item.RetentionPolicy, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return openapigen.BuildWorkspaceList{Items: items, Total: result.Total, Limit: query.Limit, Offset: query.Offset}
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

func toBuildArtifactPublicationList(result buildstore.ArtifactPublicationListResult, limit, offset int) openapigen.BuildArtifactPublicationList {
	items := make([]openapigen.BuildArtifactPublication, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, openapigen.BuildArtifactPublication{
			PublicationId: item.PublicationID,
			ArtifactId:    item.ArtifactID,
			Digest:        item.Digest,
			MediaType:     item.MediaType,
			Destination: struct {
				ConnectionRef string                                             `json:"connection_ref"`
				Kind          openapigen.BuildArtifactPublicationDestinationKind `json:"kind"`
				Reference     string                                             `json:"reference"`
				RepositoryRef string                                             `json:"repository_ref"`
			}{ConnectionRef: item.ConnectionRef, Kind: openapigen.BuildArtifactPublicationDestinationKind(item.DestinationKind), RepositoryRef: item.RepositoryRef, Reference: item.Reference},
			CredentialExecutionMode: item.CredentialExecutionMode,
			CreatedAt:               item.CreatedAt,
		})
	}
	return openapigen.BuildArtifactPublicationList{Items: items, Total: result.Total, Limit: limit, Offset: offset}
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
	return openapigen.BuildJobDetail{ApplicationId: summary.ApplicationId, ApplicationName: summary.ApplicationName, Artifact: summary.Artifact, BuildArgs: args, BuildId: summary.BuildId, ContextPath: summary.ContextPath, CreatedAt: summary.CreatedAt, DockerfilePath: summary.DockerfilePath, ImageRepository: summary.ImageRepository, ImageTag: summary.ImageTag, InputSnapshotId: item.InputSnapshotID, InputSnapshotDigest: item.InputSnapshotDigest, SourceKind: item.SourceKind, RuntimeProvider: item.RuntimeProvider, TaskId: summary.TaskId, Builder: summary.Builder, Execution: summary.Execution}
}
func toBuildJobSummary(item buildstore.JobProjection) openapigen.BuildJobSummary {
	summary := openapigen.BuildJobSummary{BuildId: item.BuildID, InputSnapshotId: item.InputSnapshotID, InputSnapshotDigest: item.InputSnapshotDigest, SourceKind: item.SourceKind, TaskId: buildJobResponseID(item.TaskID), Builder: openapigen.BuildBuilderSnapshot{Id: buildJobResponseID(item.RuntimeTargetID), Name: item.RuntimeTargetName, Provider: item.RuntimeProvider}, Execution: openapigen.BuildTaskExecution{Status: openapigen.TaskStatus(item.Execution.Status), StageCount: item.Execution.StageCount, CompletedStageCount: item.Execution.CompletedStageCount, Capabilities: openapigen.TaskCapabilities{Cancel: item.Execution.Capabilities.Cancel, Retry: item.Execution.Capabilities.Retry, DownloadLog: item.Execution.Capabilities.DownloadLog}}}
	if item.ApplicationID != "" {
		value := openapigen.ApplicationId(item.ApplicationID)
		summary.ApplicationId = &value
	}
	if item.ApplicationName != "" {
		value := item.ApplicationName
		summary.ApplicationName = &value
	}
	if item.ContextPath != "" {
		value := item.ContextPath
		summary.ContextPath = &value
	}
	if item.DockerfilePath != "" {
		value := item.DockerfilePath
		summary.DockerfilePath = &value
	}
	if item.ImageRepository != "" {
		value := item.ImageRepository
		summary.ImageRepository = &value
	}
	if item.ImageTag != "" {
		value := item.ImageTag
		summary.ImageTag = &value
	}
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
