package moduleapi

import "context"

// RuntimeTargetSummary is the minimum target identity that a provider resource may expose.
type RuntimeTargetSummary struct {
	ID          int64
	DisplayName string
	Provider    string
}

// RuntimeTargetReader is the narrow runtime-target authority used by provider modules.
// It deliberately exposes neither endpoints nor credentials.
type RuntimeTargetReader interface {
	ReadDockerTarget(context.Context, *int64) (RuntimeTargetSummary, error)
	ListDockerTargets(context.Context) ([]RuntimeTargetSummary, error)
}

// ComposeRuntimeTargetSummary is the capability-scoped identity projection for Compose applications.
type ComposeRuntimeTargetSummary struct {
	ID           int64
	DisplayName  string
	Provider     string
	Capabilities []string
	Available    bool
}

// ComposeProjectNameState is the provider-neutral result of checking a Compose
// project name against one Runtime Target.
type ComposeProjectNameState string

const (
	// ComposeProjectNameStateAvailable means the target is reachable and does not own the name.
	ComposeProjectNameStateAvailable ComposeProjectNameState = "available"
	// ComposeProjectNameStateOccupied means the target already has Compose resources with the name.
	ComposeProjectNameStateOccupied ComposeProjectNameState = "occupied"
	// ComposeProjectNameStateUnavailable means the target cannot currently be queried.
	ComposeProjectNameStateUnavailable ComposeProjectNameState = "unavailable"
	// ComposeProjectNameStateError means the provider query failed unexpectedly.
	ComposeProjectNameStateError ComposeProjectNameState = "error"
)

// ComposeProjectNameAvailability is a provider-neutral occupancy result.
type ComposeProjectNameAvailability struct {
	State ComposeProjectNameState
}

// ComposeRuntimeTargetReader resolves Runtime Targets that can execute Compose and access workspaces.
type ComposeRuntimeTargetReader interface {
	ReadComposeTarget(context.Context, *int64) (ComposeRuntimeTargetSummary, error)
	ListComposeTargets(context.Context) ([]ComposeRuntimeTargetSummary, error)
	CheckComposeProjectName(context.Context, int64, string) (ComposeProjectNameAvailability, error)
}
