// Package pluginapi defines stable cross-plugin capability contracts.
package pluginapi

import "context"

// UserSummary is the stable DTO shared across plugin boundaries.
type UserSummary struct {
	ID       uint64
	Username string
	Display  string
}

// UserService exposes the minimal user capability that other plugins may depend on.
type UserService interface {
	// GetUserByID returns one stable summary DTO instead of an internal persistence model.
	GetUserByID(ctx context.Context, id uint64) (UserSummary, error)
}
