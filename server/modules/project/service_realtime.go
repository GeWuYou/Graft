package project

import (
	"context"
	"errors"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/realtime"
	projectcontract "graft/server/modules/project/contract"
)

const projectDetailTopicRefreshInterval = 5 * time.Second

type projectListSummaryRealtimeItem struct {
	ApplicationID   string                               `json:"application_id"`
	RuntimeStatus   generated.ApplicationRuntimeStatus   `json:"runtime_status"`
	ServiceCount    int                                  `json:"service_count"`
	ContainerCounts generated.ApplicationContainerCounts `json:"container_counts"`
	DriftStatus     generated.ApplicationDriftStatus     `json:"drift_status"`
}

type projectListSummaryRealtimePayload struct {
	Topic       string                           `json:"topic"`
	PublishedAt time.Time                        `json:"published_at"`
	Items       []projectListSummaryRealtimeItem `json:"items"`
}

type projectRuntimeRealtimePayload struct {
	Topic         string                                `json:"topic"`
	ApplicationID string                                `json:"application_id"`
	PublishedAt   time.Time                             `json:"published_at"`
	Detail        generated.ApplicationDetailResponse   `json:"detail"`
	Overview      generated.ApplicationOverviewResponse `json:"overview"`
	Services      generated.ApplicationServicesResponse `json:"services"`
}

type projectLifecycleConfigRealtimePayload struct {
	Topic         string                              `json:"topic"`
	ApplicationID string                              `json:"application_id"`
	PublishedAt   time.Time                           `json:"published_at"`
	Detail        generated.ApplicationDetailResponse `json:"detail"`
}

type projectLogsRealtimePayload struct {
	Topic string                        `json:"topic"`
	Entry generated.ApplicationLogEntry `json:"entry"`
}

// IssueSubscription 为项目模块主题签发实时订阅票据，并在签发前校验主题格式、项目归属和查看权限。
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
	case topic == projectcontract.ApplicationListSummaryTopic:
		return s.issueProjectListSummaryRealtimeSubscription(ctx, request, topic)
	case strings.HasPrefix(topic, projectcontract.ApplicationRuntimeTopicPrefix):
		return s.issueProjectRuntimeRealtimeSubscription(ctx, request, topic)
	case strings.HasPrefix(topic, projectcontract.ApplicationLifecycleConfigTopicPrefix):
		return s.issueProjectLifecycleConfigRealtimeSubscription(ctx, request, topic)
	case strings.HasPrefix(topic, projectcontract.ApplicationLogsTopicPrefix):
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
	// 先注册精确主题，再注册带前缀的项目主题；注册表按声明顺序解析匹配关系。
	if err := s.topicIssuers.Register(projectcontract.ApplicationListSummaryTopic, s); err != nil {
		return err
	}
	if err := s.topicIssuers.Register(projectcontract.ApplicationRuntimeTopicPrefix, s); err != nil {
		return err
	}
	if err := s.topicIssuers.Register(projectcontract.ApplicationLifecycleConfigTopicPrefix, s); err != nil {
		return err
	}
	return s.topicIssuers.Register(projectcontract.ApplicationLogsTopicPrefix, s)
}

// Close 停止项目模块拥有的实时流，并等待各流退出或返回调用方提供的关闭上下文错误。
func (s *Service) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.streamersMu.Lock()
	listStreamer := s.listTopicStreamer
	runtimeStreamer := s.runtimeTopicStreamer
	lifecycleConfigStreamer := s.lifecycleConfigTopicStreamer
	logStreamer := s.logTopicStreamer
	s.streamersMu.Unlock()
	var closeErr error
	if listStreamer != nil {
		closeErr = errors.Join(closeErr, listStreamer.Close(ctx))
	}
	if runtimeStreamer != nil {
		closeErr = errors.Join(closeErr, runtimeStreamer.Close(ctx))
	}
	if lifecycleConfigStreamer != nil {
		closeErr = errors.Join(closeErr, lifecycleConfigStreamer.Close(ctx))
	}
	if logStreamer != nil {
		closeErr = errors.Join(closeErr, logStreamer.Close(ctx))
	}
	return closeErr
}

func (s *Service) issueProjectListSummaryRealtimeSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
) (realtime.SubscriptionResponse, error) {
	if topic != projectcontract.ApplicationListSummaryTopic {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
	}
	if err := s.ensureRealtimeAccess(ctx, request, projectcontract.ApplicationViewPermission.String()); err != nil {
		return realtime.SubscriptionResponse{}, err
	}
	if err := s.ensureProjectListSummaryTopicStreaming(topic); err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	return s.issueTopicTicket(ctx, request, topic)
}

func (s *Service) issueProjectRuntimeRealtimeSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
) (realtime.SubscriptionResponse, error) {
	return s.issueProjectScopedRealtimeSubscription(
		ctx,
		request,
		topic,
		projectcontract.ApplicationRuntimeTopicPrefix,
		s.ensureProjectRuntimeTopicStreaming,
	)
}

func (s *Service) issueProjectLifecycleConfigRealtimeSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
) (realtime.SubscriptionResponse, error) {
	return s.issueProjectScopedRealtimeSubscription(
		ctx,
		request,
		topic,
		projectcontract.ApplicationLifecycleConfigTopicPrefix,
		s.ensureProjectLifecycleConfigTopicStreaming,
	)
}

func (s *Service) issueProjectScopedRealtimeSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
	prefix string,
	ensureStreaming func(string, uint64) error,
) (realtime.SubscriptionResponse, error) {
	applicationID, err := parseProjectRealtimeTopicApplicationID(topic, prefix)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
	}
	if err := s.ensureRealtimeAccess(ctx, request, projectcontract.ApplicationViewPermission.String()); err != nil {
		return realtime.SubscriptionResponse{}, err
	}
	projectID, err := s.ResolveApplicationID(ctx, applicationID)
	if err != nil {
		return realtime.SubscriptionResponse{}, mapProjectRealtimeError(err)
	}
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return realtime.SubscriptionResponse{}, mapProjectRealtimeError(err)
	}
	if err := s.ensureApplicationScope(ctx, aggregate, projectcontract.ApplicationViewPermission.String()); err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	if err := ensureStreaming(topic, projectID); err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	return s.issueTopicTicket(ctx, request, topic)
}

func (s *Service) issueProjectLogsRealtimeSubscription(
	ctx context.Context,
	request realtime.SubscriptionRequest,
	topic string,
) (realtime.SubscriptionResponse, error) {
	applicationID, err := parseProjectRealtimeTopicApplicationID(topic, projectcontract.ApplicationLogsTopicPrefix)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicNotFound
	}
	if err := s.ensureRealtimeAccess(ctx, request, projectcontract.ApplicationViewPermission.String()); err != nil {
		return realtime.SubscriptionResponse{}, err
	}
	projectID, err := s.ResolveApplicationID(ctx, applicationID)
	if err != nil {
		return realtime.SubscriptionResponse{}, mapProjectRealtimeError(err)
	}
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return realtime.SubscriptionResponse{}, mapProjectRealtimeError(err)
	}
	if err := s.ensureApplicationScope(ctx, aggregate, projectcontract.ApplicationViewPermission.String()); err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	if err := s.ensureProjectLogsTopicStreaming(topic, aggregate.Application.ApplicationRecordID, LogQuery{
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

// 如果主题不包含指定前缀或应用 ID 格式无效，则返回错误。
func parseProjectRealtimeTopicApplicationID(topic string, prefix string) (string, error) {
	if !strings.HasPrefix(topic, prefix) {
		return "", errProjectInvalidArgument
	}
	applicationID := strings.TrimSpace(strings.TrimPrefix(topic, prefix))
	if !isApplicationID(applicationID) {
		return "", errProjectInvalidArgument
	}
	return applicationID, nil
}

// mapProjectRealtimeError 将项目错误映射为实时订阅错误，保持订阅接口的外部错误语义稳定。
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
