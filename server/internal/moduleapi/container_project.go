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

// ContainerProjectServiceResourceSummary describes one bounded per-service runtime resource aggregate.
type ContainerProjectServiceResourceSummary struct {
	ServiceName                  string
	ContainerCount               int
	RunningCount                 int
	StoppedCount                 int
	TransitioningCount           int
	IssueCount                   int
	HealthyContainerCount        int
	UnhealthyContainerCount      int
	StartingContainerCount       int
	RestartCount                 int
	StatsAvailable               bool
	StatsAvailableContainerCount int
	CPUPercent                   float64
	MemoryUsageBytes             int64
	MemoryLimitBytes             int64
}

// ContainerProjectResourceSummary describes one bounded project-level runtime resource aggregate.
type ContainerProjectResourceSummary struct {
	CanonicalProjectName         string
	CollectedAt                  string
	StatsAvailable               bool
	StatsAvailableContainerCount int
	HealthyContainerCount        int
	UnhealthyContainerCount      int
	StartingContainerCount       int
	RestartCount                 int
	CPUPercent                   float64
	MemoryUsageBytes             int64
	MemoryLimitBytes             int64
	RxBytes                      int64
	TxBytes                      int64
	Services                     []ContainerProjectServiceResourceSummary
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
	ListImportCandidateMembers(
		ctx context.Context,
		hostScope string,
		candidate ContainerProjectRuntimeCandidate,
	) ([]ContainerProjectMember, error)
}

// ContainerProjectResourceReader exposes the narrow shared boundary for Project-owned overview resource aggregation.
//
// Container remains the authority for stats collection, normalization, cache, and realtime topics. Project may only
// consume this bounded aggregate rather than direct container stats snapshots or runtime internals.
type ContainerProjectResourceReader interface {
	ReadProjectResourceSummary(
		ctx context.Context,
		hostScope string,
		canonicalProjectName string,
	) (ContainerProjectResourceSummary, error)
}
