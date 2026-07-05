package project

import (
	"context"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
)

const (
	defaultProjectLogsTail = 200
	maxProjectLogsTail     = 1000
)

// LogQuery describes the project-owned aggregated log query.
type LogQuery struct {
	Tail       int
	Since      string
	Timestamps bool
	Stdout     bool
	Stderr     bool
}

// Logs returns a project-owned aggregated log snapshot with explicit source attribution.
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
	snapshot, err := s.logReader.ReadProjectLogs(ctx, aggregate.Project.HostScope, aggregate.Project.CanonicalProjectName, toContainerProjectLogQuery(logQuery))
	if err != nil {
		return generated.ProjectLogResponse{}, err
	}
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

func toContainerProjectLogQuery(query LogQuery) moduleapi.ContainerProjectLogQuery {
	return moduleapi.ContainerProjectLogQuery{
		Tail:       query.Tail,
		Since:      strings.TrimSpace(query.Since),
		Timestamps: query.Timestamps,
		Stdout:     query.Stdout,
		Stderr:     query.Stderr,
	}
}

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
