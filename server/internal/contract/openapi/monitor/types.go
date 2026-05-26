package monitor

import "context"

// ServerInterface is the minimal monitor-only generated handler contract used by this spike.
type ServerInterface interface {
	GetMonitorServerStatus(ctx context.Context, params GetMonitorServerStatusParams) error
}
