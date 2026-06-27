package container

import (
	"context"
	"errors"
	"strings"
)

var errRuntimeEventHistoryUnavailable = errors.New("container runtime event history unavailable")

// RuntimeEventHistory returns bounded per-container runtime event history with seq for reconnect-safe merge/dedupe.
func (s *service) RuntimeEventHistory(ctx context.Context, ref Ref) (RuntimeEventsHistory, error) {
	if s == nil {
		return RuntimeEventsHistory{}, errRuntimeEventHistoryUnavailable
	}
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return RuntimeEventsHistory{}, err
	}
	manager := s.runtimeEventManagerForRead()
	if manager == nil {
		return RuntimeEventsHistory{}, errRuntimeEventHistoryUnavailable
	}
	detail, err := s.Detail(ctx, ref)
	if err != nil {
		return RuntimeEventsHistory{}, err
	}
	resourceID := strings.TrimSpace(detail.ID)
	if resourceID == "" {
		resourceID = strings.TrimSpace(ref.Value)
	}
	if resourceID == "" {
		return RuntimeEventsHistory{}, errRuntimeEventHistoryUnavailable
	}
	return manager.History(resourceID), nil
}
