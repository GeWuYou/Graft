package build

import (
	"math"

	openapigen "graft/server/internal/contract/openapi/generated"
	buildstore "graft/server/modules/build/store"
)

func toBuildJobList(result buildstore.ListResult, query buildstore.ListQuery) openapigen.BuildJobList {
	items := make([]openapigen.BuildJobSummary, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toBuildJobSummary(item))
	}
	return openapigen.BuildJobList{Items: items, Total: result.Total, Limit: query.Limit, Offset: query.Offset}
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
		summary.Artifact = &openapigen.BuildArtifact{ArtifactId: item.Artifact.ArtifactID, ImageId: item.Artifact.ImageID, Repository: item.Artifact.Repository, Tag: item.Artifact.Tag}
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
