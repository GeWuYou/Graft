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
}
