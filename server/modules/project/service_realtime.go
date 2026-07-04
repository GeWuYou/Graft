package project

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/realtime"
	projectcontract "graft/server/modules/project/contract"
)

const projectDetailTopicRefreshInterval = 5 * time.Second

type projectDetailRealtimePayload struct {
	Topic       string                            `json:"topic"`
	ProjectID   int64                             `json:"project_id"`
	PublishedAt time.Time                         `json:"published_at"`
	Detail      generated.ProjectDetailResponse   `json:"detail"`
	Overview    generated.ProjectOverviewResponse `json:"overview"`
	Services    generated.ProjectServicesResponse `json:"services"`
}

type projectLogsRealtimePayload struct {
	Topic string                    `json:"topic"`
	Entry generated.ProjectLogEntry `json:"entry"`
}

// IssueSubscription issues project-owned realtime topic subscriptions.
func (s *Service) IssueSubscription(
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
	switch {
	case strings.HasPrefix(topic, projectcontract.ProjectDetailTopicPrefix):
		return s.issueProjectDetailRealtimeSubscription(ctx, request, topic)
	case strings.HasPrefix(topic, projectcontract.ProjectLogsTopicPrefix):
		return s.issueProjectLogsRealtimeSubscription(ctx, request, topic)
	default:
		return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
	}
}

func (s *Service) registerRealtimeTopics() error {
	if s == nil {
		return nil
	}
	if s.topicIssuers == nil {
		return errors.New("realtime topic issuer registry is unavailable")
	}
	if err := s.topicIssuers.Register(projectcontract.ProjectDetailTopicPrefix, s); err != nil {
		return err
	}
	return s.topicIssuers.Register(projectcontract.ProjectLogsTopicPrefix, s)
}

// Close releases project-owned realtime streaming resources.
func (s *Service) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	var closeErr error
	if s.detailTopicStreamer != nil {
		closeErr = errors.Join(closeErr, s.detailTopicStreamer.Close(ctx))
	}
	if s.logTopicStreamer != nil {
		closeErr = errors.Join(closeErr, s.logTopicStreamer.Close(ctx))
	}
	return closeErr
}

func (s *Service) issueProjectDetailRealtimeSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
) (realtime.SubscriptionResponse, error) {
	projectID, err := parseProjectRealtimeTopicID(topic, projectcontract.ProjectDetailTopicPrefix)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
	}
	if err := s.ensureRealtimeAccess(ctx, request, projectcontract.ProjectViewPermission.String()); err != nil {
		return realtime.SubscriptionResponse{}, err
	}
	if _, err := s.Get(ctx, projectID); err != nil {
		return realtime.SubscriptionResponse{}, mapProjectRealtimeError(err)
	}
	if err := s.ensureProjectDetailTopicStreaming(topic, projectID); err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	return s.issueTopicTicket(ctx, request, topic)
}

func (s *Service) issueProjectLogsRealtimeSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
) (realtime.SubscriptionResponse, error) {
	projectID, err := parseProjectRealtimeTopicID(topic, projectcontract.ProjectLogsTopicPrefix)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
	}
	if err := s.ensureRealtimeAccess(ctx, request, projectcontract.ProjectViewPermission.String()); err != nil {
		return realtime.SubscriptionResponse{}, err
	}
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return realtime.SubscriptionResponse{}, mapProjectRealtimeError(err)
	}
	if err := s.ensureProjectLogsTopicStreaming(topic, aggregate.Project.ID, LogQuery{
		Tail:       defaultProjectLogsTail,
		Timestamps: true,
		Stdout:     true,
		Stderr:     true,
	}); err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	return s.issueTopicTicket(ctx, request, topic)
}

func (s *Service) issueTopicTicket(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
) (realtime.SubscriptionResponse, error) {
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

func (s *Service) ensureRealtimeAccess(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	permission string,
) error {
	if request.RequestAuth.User == nil {
		return realtime.ErrTopicForbidden
	}
	if s.authorizer == nil {
		return realtime.ErrTopicForbidden
	}
	if err := s.authorizer.Authorize(ctx, request.RequestAuth, permission); err != nil {
		return realtime.ErrTopicForbidden
	}
	return nil
}

func parseProjectRealtimeTopicID(topic string, prefix string) (uint64, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(topic, prefix))
	if raw == "" {
		return 0, errProjectInvalidArgument
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		return 0, errProjectInvalidArgument
	}
	return value, nil
}

func mapProjectRealtimeError(err error) error {
	switch {
	case errors.Is(err, errProjectNotFound):
		return realtime.ErrTopicNotFound
	case errors.Is(err, errProjectRuntimeUnavailable):
		return realtime.ErrTopicForbidden
	default:
		return realtime.ErrTopicConflict
	}
}
