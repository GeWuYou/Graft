package project

import (
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
)

func buildProjectContainerCounts(summary *moduleapi.ContainerProjectRuntimeSummary) generated.ProjectContainerCounts {
	counts := generated.ProjectContainerCounts{}
	if summary == nil {
		return counts
	}
	for _, member := range summary.Members {
		switch member.CanonicalState {
		case "running":
			counts.Running++
		case "exited":
			counts.Stopped++
		case "created", "restarting", "removing":
			counts.Transitioning++
		case "paused", "dead", "unknown":
			counts.Issue++
		default:
			counts.Issue++
		}
	}
	counts.Total = len(summary.Members)
	return counts
}

func deriveProjectRuntimeStatus(
	summary *moduleapi.ContainerProjectRuntimeSummary,
	runtimeErr error,
) *generated.ProjectRuntimeStatus {
	if runtimeErr != nil {
		status := generated.ProjectRuntimeStatusUnknown
		return &status
	}
	if summary == nil || len(summary.Members) == 0 {
		status := generated.ProjectRuntimeStatusUnknown
		return &status
	}

	counts := buildProjectContainerCounts(summary)

	switch {
	case counts.Transitioning > 0:
		status := generated.ProjectRuntimeStatusTransitioning
		return &status
	case counts.Running == counts.Total:
		status := generated.ProjectRuntimeStatusRunning
		return &status
	case counts.Stopped == counts.Total:
		status := generated.ProjectRuntimeStatusStopped
		return &status
	default:
		status := generated.ProjectRuntimeStatusDegraded
		return &status
	}
}
