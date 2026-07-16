package container

import (
	"context"
	"errors"
	"strings"
)

var errRuntimeEventHistoryUnavailable = errors.New("container runtime event history unavailable")

// RuntimeEventHistory 返回有界的容器运行时事件历史，并提供 seq 供重连后的合并与去重使用。
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
