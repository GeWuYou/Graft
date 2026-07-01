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

// ContainerProjectRuntimeContainerCounts describes one bounded runtime container aggregate.
type ContainerProjectRuntimeContainerCounts struct {
	Running int
	Stopped int
	Total   int
}

// ContainerProjectRuntimeCandidate describes one Compose import candidate projected from container runtime metadata.
//
// The container module owns runtime metadata extraction and status/reason codes tied to runtime authority.
// Project may consume these candidates but must not parse Docker labels on its own.
type ContainerProjectRuntimeCandidate struct {
	CandidateKey           string
	CanonicalProjectName   string
	Status                 string
	StatusReasonCodes      []string
	Importable             bool
	RuntimeType            string
	RuntimeVersion         string
	WorkingDirectory       string
	WorkingDirectorySource string
	ConfigFiles            []string
	ServiceNames           []string
	ContainerCounts        ContainerProjectRuntimeContainerCounts
	Warnings               []string
}

// ContainerProjectRuntimeReader exposes the narrow shared boundary for Project-owned service aggregation.
//
// This boundary exists so the project module can aggregate runtime counts without importing container
// module internals. Container remains the runtime authority.
type ContainerProjectRuntimeReader interface {
	ListProjectMembers(ctx context.Context, hostScope string, canonicalProjectName string) (ContainerProjectRuntimeSummary, error)
	ListImportCandidates(ctx context.Context, hostScope string) ([]ContainerProjectRuntimeCandidate, error)
}
