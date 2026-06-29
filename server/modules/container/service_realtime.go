package container

import (
	"context"
	"errors"
	"strings"

	"graft/server/internal/realtime"
	containercontract "graft/server/modules/container/contract"
)

// IssueSubscription 为容器实时主题签发一次性订阅票据。
//
// 按主题类型分发到对应的容器列表、仪表盘汇总或单容器订阅路径，并在签发前完成最小权限与主题有效性校验。
func (s *service) IssueSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
) (realtime.SubscriptionResponse, error) {
	if s == nil {
		return realtime.SubscriptionResponse{}, realtime.ErrIssuerRequired
	}

	topic := realtime.NormalizeTopic(request.Topic)
	if topic == "" {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicRequired
	}
	if topic == containercontract.ContainerListStatsTopic {
		return s.issueContainerListRealtimeSubscription(ctx, request, topic)
	}
	if topic == containercontract.ContainerDashboardSummaryTopic {
		return s.issueContainerDashboardSummaryRealtimeSubscription(ctx, request, topic)
	}
	if !strings.HasPrefix(topic, containercontract.ContainerLogsTopicPrefix) &&
		!strings.HasPrefix(topic, containercontract.ContainerStatsTopicPrefix) &&
		!strings.HasPrefix(topic, containercontract.ContainerEventsTopicPrefix) {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
	}
	return s.issueProtectedContainerRealtimeSubscription(ctx, request, topic)
}

func (s *service) issueProtectedContainerRealtimeSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
) (realtime.SubscriptionResponse, error) {
	if request.RequestAuth.User == nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	if s.authorizer == nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	if strings.HasPrefix(topic, containercontract.ContainerLogsTopicPrefix) {
		return s.issueContainerLogsRealtimeSubscription(ctx, request, topic)
	}
	if strings.HasPrefix(topic, containercontract.ContainerEventsTopicPrefix) {
		return s.issueContainerEventsRealtimeSubscription(ctx, request, topic)
	}

	return s.issueContainerRealtimeSubscription(ctx, request, topic)
}

func (s *service) issueContainerEventsRealtimeSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
) (realtime.SubscriptionResponse, error) {
	containerID := strings.TrimSpace(strings.TrimPrefix(topic, containercontract.ContainerEventsTopicPrefix))
	ref, err := parseRef(containerID)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
	}
	if err := s.authorizer.Authorize(ctx, request.RequestAuth, containercontract.ContainerEventsPermission.String()); err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	if _, err := s.Detail(ctx, ref); err != nil {
		if errors.Is(err, errContainerNotFound) {
			return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
		}
		if errors.Is(err, errRuntimeDisabled) {
			return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
		}
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}

	issued, err := (realtime.TicketIssuer{Tickets: s.realtimeTickets}).IssueTopicTicket(ctx, request)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	return realtime.SubscriptionResponse{
		Topic:        topic,
		Ticket:       issued.Ticket,
		WebSocketURL: realtime.BuildTopicWebSocketURL(topic, issued.Ticket),
		ExpiresAt:    issued.ExpiresAt,
	}, nil
}

func (s *service) issueContainerLogsRealtimeSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
) (realtime.SubscriptionResponse, error) {
	containerID := strings.TrimSpace(strings.TrimPrefix(topic, containercontract.ContainerLogsTopicPrefix))
	ref, err := parseRef(containerID)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
	}
	if err := s.authorizer.Authorize(ctx, request.RequestAuth, containercontract.ContainerLogsPermission.String()); err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	logQuery, err := s.normalizeLogQuery(ctx, LogQuery{})
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	if err := s.ensureLogTopicStreaming(ctx, topic, ref, logQuery); err != nil {
		if errors.Is(err, errContainerNotFound) || errors.Is(err, errInvalidRef) {
			return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
		}
		if errors.Is(err, errRuntimeDisabled) {
			return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
		}
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}

	issued, err := (realtime.TicketIssuer{Tickets: s.realtimeTickets}).IssueTopicTicket(ctx, request)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	return realtime.SubscriptionResponse{
		Topic:        topic,
		Ticket:       issued.Ticket,
		WebSocketURL: realtime.BuildTopicWebSocketURL(topic, issued.Ticket),
		ExpiresAt:    issued.ExpiresAt,
	}, nil
}

func (s *service) issueContainerListRealtimeSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
) (realtime.SubscriptionResponse, error) {
	if request.RequestAuth.User == nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	if s.authorizer == nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	if err := s.authorizer.Authorize(ctx, request.RequestAuth, containercontract.ContainerViewPermission.String()); err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	if _, err := s.List(ctx, ListQuery{Limit: 1}); err != nil {
		if errors.Is(err, errRuntimeDisabled) {
			return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
		}
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}

	issued, err := (realtime.TicketIssuer{Tickets: s.realtimeTickets}).IssueTopicTicket(ctx, request)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	return realtime.SubscriptionResponse{
		Topic:        topic,
		Ticket:       issued.Ticket,
		WebSocketURL: realtime.BuildTopicWebSocketURL(topic, issued.Ticket),
		ExpiresAt:    issued.ExpiresAt,
	}, nil
}

func (s *service) issueContainerDashboardSummaryRealtimeSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
) (realtime.SubscriptionResponse, error) {
	if request.RequestAuth.User == nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	if s.authorizer == nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	if err := s.authorizer.Authorize(ctx, request.RequestAuth, containercontract.ContainerViewPermission.String()); err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	if err := s.requireRuntimeAccess(ctx); err != nil {
		if errors.Is(err, errRuntimeDisabled) {
			return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
		}
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	if _, err := s.runtimeForRequest(); err != nil {
		if errors.Is(err, errRuntimeDisabled) {
			return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
		}
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}

	issued, err := (realtime.TicketIssuer{Tickets: s.realtimeTickets}).IssueTopicTicket(ctx, request)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	return realtime.SubscriptionResponse{
		Topic:        topic,
		Ticket:       issued.Ticket,
		WebSocketURL: realtime.BuildTopicWebSocketURL(topic, issued.Ticket),
		ExpiresAt:    issued.ExpiresAt,
	}, nil
}

func (s *service) issueContainerRealtimeSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
) (realtime.SubscriptionResponse, error) {
	containerID := strings.TrimSpace(strings.TrimPrefix(topic, containercontract.ContainerStatsTopicPrefix))
	ref, err := parseRef(containerID)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
	}
	if err := s.authorizer.Authorize(ctx, request.RequestAuth, containercontract.ContainerDetailPermission.String()); err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	if _, err := s.Detail(ctx, ref); err != nil {
		if errors.Is(err, errContainerNotFound) {
			return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
		}
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}

	issued, err := (realtime.TicketIssuer{Tickets: s.realtimeTickets}).IssueTopicTicket(ctx, request)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	return realtime.SubscriptionResponse{
		Topic:        topic,
		Ticket:       issued.Ticket,
		WebSocketURL: realtime.BuildTopicWebSocketURL(topic, issued.Ticket),
		ExpiresAt:    issued.ExpiresAt,
	}, nil
}
