package project

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
)

const (
	defaultProjectLogsTail = 200
	maxProjectLogsTail     = 1000
)

// LogQuery 描述项目聚合日志查询；未指定输出流时由服务同时查询 stdout 和 stderr。
type LogQuery struct {
	Tail       int
	Since      string
	Timestamps bool
	Stdout     bool
	Stderr     bool
}

// Logs 返回项目聚合日志快照，并为每条日志保留容器和服务来源标识。
func (s *Service) Logs(ctx context.Context, projectID uint64, query LogQuery) (generated.ProjectLogResponse, error) {
	if s == nil || s.logReader == nil {
		return generated.ProjectLogResponse{}, errProjectRuntimeUnavailable
	}
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return generated.ProjectLogResponse{}, err
	}
	logQuery, err := normalizeProjectLogQuery(query)
	if err != nil {
		return generated.ProjectLogResponse{}, err
	}
	s.logProjectLogDiagnostic("snapshot-started",
		zap.Uint64("project_id", projectID),
		zap.String("canonical_project_name", aggregate.Project.CanonicalProjectName),
		zap.Int("tail", logQuery.Tail),
		zap.String("since", logQuery.Since),
	)
	snapshot, err := s.logReader.ReadProjectLogs(ctx, aggregate.Project.HostScope, aggregate.Project.CanonicalProjectName, toContainerProjectLogQuery(logQuery))
	if err != nil {
		s.logProjectLogDiagnostic("snapshot-failed", zap.Uint64("project_id", projectID), zap.Error(err))
		return generated.ProjectLogResponse{}, err
	}
	s.logProjectLogDiagnostic("snapshot-completed",
		zap.Uint64("project_id", projectID),
		zap.Int("entry_count", len(snapshot.Entries)),
		zap.Int("tail", snapshot.Tail),
		zap.Bool("truncated", snapshot.Truncated),
	)
	return toProjectLogResponse(projectID, aggregate.Project.CanonicalProjectName, snapshot), nil
}

func normalizeProjectLogQuery(query LogQuery) (LogQuery, error) {
	normalized := query
	if normalized.Tail == 0 {
		normalized.Tail = defaultProjectLogsTail
	}
	if normalized.Tail < 0 || normalized.Tail > maxProjectLogsTail {
		return LogQuery{}, errProjectInvalidArgument
	}
	if !normalized.Stdout && !normalized.Stderr {
		normalized.Stdout = true
		normalized.Stderr = true
	}
	normalized.Since = strings.TrimSpace(normalized.Since)
	return normalized, nil
}

// toContainerProjectLogQuery 将项目 API 日志查询转换为容器运行时查询。
func toContainerProjectLogQuery(query LogQuery) moduleapi.ContainerProjectLogQuery {
	return moduleapi.ContainerProjectLogQuery{
		Tail:       query.Tail,
		Since:      strings.TrimSpace(query.Since),
		Timestamps: query.Timestamps,
		Stdout:     query.Stdout,
		Stderr:     query.Stderr,
	}
}

// toContainerProjectLogFollowQuery 将日志查询转换为仅跟随模式的容器日志查询。
func toContainerProjectLogFollowQuery(query LogQuery) moduleapi.ContainerProjectLogQuery {
	followQuery := toContainerProjectLogQuery(query)
	followQuery.FollowOnly = true
	return followQuery
}

// toProjectLogResponse 将容器日志快照转换为项目日志响应。
func toProjectLogResponse(
	projectID uint64,
	canonicalProjectName string,
	snapshot moduleapi.ContainerProjectLogSnapshot,
) generated.ProjectLogResponse {
	response := generated.ProjectLogResponse{
		ProjectId:            mustGeneratedID(projectID),
		CanonicalProjectName: canonicalProjectName,
		Tail:                 snapshot.Tail,
		Timestamps:           snapshot.Timestamps,
		Stdout:               snapshot.Stdout,
		Stderr:               snapshot.Stderr,
		Truncated:            snapshot.Truncated,
		Entries:              make([]generated.ProjectLogEntry, 0, len(snapshot.Entries)),
	}
	if snapshot.Since != nil {
		response.Since = stringPointer(strings.TrimSpace(*snapshot.Since))
	}
	for _, entry := range snapshot.Entries {
		response.Entries = append(response.Entries, generated.ProjectLogEntry{
			ContainerId:   entry.ContainerID,
			ContainerName: entry.ContainerName,
			ServiceName:   entry.ServiceName,
			Line:          entry.Line,
			Stream:        generated.ProjectLogEntryStream(strings.TrimSpace(entry.Stream)),
			OccurredAt:    parseGeneratedLogTime(entry.OccurredAt),
			Source: generated.ProjectLogEntrySource{
				ContainerId:   entry.ContainerID,
				ContainerName: entry.ContainerName,
				ServiceName:   entry.ServiceName,
			},
		})
	}
	return response
}

func parseGeneratedLogTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}
