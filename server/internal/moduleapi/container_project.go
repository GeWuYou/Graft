package moduleapi

import "context"

// ContainerProjectMember describes the narrow runtime projection Project may consume for one container.
//
// It intentionally excludes logs, events, stats, shell, inspect payloads, and other container-detail
// fields so Project cannot become a second runtime authority.
type ContainerProjectMember struct {
	ContainerID    string
	ContainerName  string
	ServiceName    string
	CanonicalState string
}

// ContainerProjectRuntimeSummary describes the bounded runtime aggregate for one Compose project identity.
type ContainerProjectRuntimeSummary struct {
	CanonicalProjectName string
	RunningCount         int
	StoppedCount         int
	Members              []ContainerProjectMember
}

// ContainerProjectRuntimeReader exposes the narrow shared boundary for Project-owned service aggregation.
//
// This boundary exists so the project module can aggregate runtime counts without importing container
// module internals. Container remains the runtime authority.
type ContainerProjectRuntimeReader interface {
	ListProjectMembers(ctx context.Context, hostScope string, canonicalProjectName string) (ContainerProjectRuntimeSummary, error)
}
