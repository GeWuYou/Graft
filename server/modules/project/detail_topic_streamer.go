package project

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/logger"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/realtime"
)

type projectRuntimeTopicStreamer struct {
	hub     realtime.Hub
	monitor realtime.TopicSubscriptionMonitor
	logger  *zap.Logger
	service *Service

	mu      sync.Mutex
	streams map[string]*projectRuntimeTopicStream
}

type projectRuntimeTopicStream struct {
	topic              string
	projectID          uint64
	unregisterObserver func()
	cancel             context.CancelFunc
	done               chan struct{}
	runID              uint64
}

// markProjectRuntimeTopicStreamDone 在完成信号可用时无阻塞地标记实时流结束。
func markProjectRuntimeTopicStreamDone(done chan struct{}) {
	if done == nil {
		return
	}
	select {
	case done <- struct{}{}:
	default:
	}
}

// omitProjectRuntimeTopicStream 返回移除指定主题后的项目运行时主题流映射。
func omitProjectRuntimeTopicStream(
	streams map[string]*projectRuntimeTopicStream,
	topic string,
) map[string]*projectRuntimeTopicStream {
	if len(streams) == 0 {
		return streams
	}
	next := make(map[string]*projectRuntimeTopicStream, len(streams))
	for key, stream := range streams {
		if key == topic {
			continue
		}
		next[key] = stream
	}
	return next
}

// newProjectRuntimeTopicStreamer 使用指定的 hub、logger 和 service 创建项目运行时主题流。
// hub 或 service 不可用，或 hub 不支持主题订阅监测时返回错误。
//
//nolint:dupl // 运行时流与生命周期流保留独立的具体类型和生命周期所有权。
func newProjectRuntimeTopicStreamer(hub realtime.Hub, logger *zap.Logger, service *Service) (*projectRuntimeTopicStreamer, error) {
	if hub == nil {
		return nil, errors.New("realtime hub is unavailable")
	}
	monitor, ok := hub.(realtime.TopicSubscriptionMonitor)
	if !ok {
		return nil, errors.New("realtime hub does not support topic subscription monitoring")
	}
	if service == nil {
		return nil, errors.New("project runtime service is unavailable")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &projectRuntimeTopicStreamer{
		hub:     hub,
		monitor: monitor,
		logger:  logger,
		service: service,
		streams: make(map[string]*projectRuntimeTopicStream),
	}, nil
}

func (s *projectRuntimeTopicStreamer) EnsureTopic(topic string, projectID uint64) error {
	// 主题记录先于观察器注册，保证并发订阅不会重复创建运行实例；无订阅时不启动刷新协程。
	if s == nil {
		return errors.New("project runtime topic streamer is unavailable")
	}
	s.mu.Lock()
	if s.streams[topic] != nil {
		s.mu.Unlock()
		return nil
	}
	stream := &projectRuntimeTopicStream{topic: topic, projectID: projectID}
	s.streams[topic] = stream
	s.mu.Unlock()

	unregister, err := s.monitor.RegisterTopicObserver(topic, func(string) {
		s.start(topic)
	}, func(string) {
		if err := s.stop(context.Background(), topic); err != nil {
			logger.Category(s.logger, logger.CategoryComposeRuntime).Warn("stop project runtime stream failed", zap.String("topic", logsafe.SanitizeText(topic)), zap.Error(err))
		}
	})
	if err != nil {
		s.mu.Lock()
		s.streams = omitProjectRuntimeTopicStream(s.streams, topic)
		s.mu.Unlock()
		return err
	}
	s.mu.Lock()
	current := s.streams[topic]
	if current == stream {
		current.unregisterObserver = unregister
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()
	unregister()
	return nil
}

//nolint:dupl
func (s *projectRuntimeTopicStreamer) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	topics := make([]string, 0, len(s.streams))
	for topic := range s.streams {
		topics = append(topics, topic)
	}
	s.mu.Unlock()
	var closeErr error
	// 关闭必须等待正在执行的发布完成；超时主题保留登记，便于后续继续收敛资源。
	for _, topic := range topics {
		stopErr := s.stop(ctx, topic)
		closeErr = errors.Join(closeErr, stopErr)
		if stopErr != nil {
			continue
		}
		s.mu.Lock()
		stream := s.streams[topic]
		s.streams = omitProjectRuntimeTopicStream(s.streams, topic)
		s.mu.Unlock()
		if stream != nil && stream.unregisterObserver != nil {
			stream.unregisterObserver()
		}
	}
	return closeErr
}

func (s *projectRuntimeTopicStreamer) start(topic string) {
	s.mu.Lock()
	stream := s.streams[topic]
	if stream == nil || stream.cancel != nil {
		s.mu.Unlock()
		return
	}
	// runID 防止旧一轮协程在取消后覆盖新一轮主题状态。
	runCtx, cancel := context.WithCancel(s.service.realtimeStreamContext())
	stream.cancel = cancel
	stream.done = make(chan struct{}, 1)
	stream.runID++
	runID := stream.runID
	projectID := stream.projectID
	done := stream.done
	s.mu.Unlock()

	go func() {
		defer markProjectRuntimeTopicStreamDone(done)
		s.publish(runCtx, topic, projectID)
		ticker := time.NewTicker(projectDetailTopicRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				s.clearRun(topic, runID)
				return
			case <-ticker.C:
				s.publish(runCtx, topic, projectID)
			}
		}
	}()
}

func (s *projectRuntimeTopicStreamer) publish(parent context.Context, topic string, projectID uint64) {
	if s == nil || s.service == nil {
		return
	}
	ctx, cancel := context.WithTimeout(parent, projectDetailTopicRefreshInterval)
	defer cancel()
	payload, err := s.service.buildProjectRuntimeRealtimePayload(ctx, topic, projectID)
	if err != nil {
		logger.Category(s.logger, logger.CategoryComposeRuntime).Warn("publish project runtime realtime snapshot failed", zap.String("topic", logsafe.SanitizeText(topic)), zap.Error(err))
		return
	}
	s.hub.Publish(topic, payload)
}

//nolint:dupl
func (s *projectRuntimeTopicStreamer) stop(ctx context.Context, topic string) error {
	s.mu.Lock()
	stream := s.streams[topic]
	if stream == nil || stream.cancel == nil {
		s.mu.Unlock()
		return nil
	}
	cancel := stream.cancel
	done := stream.done
	s.mu.Unlock()
	cancel()
	if done == nil {
		return nil
	}
	if ctx == nil {
		<-done
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *projectRuntimeTopicStreamer) clearRun(topic string, runID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stream := s.streams[topic]
	if stream == nil || stream.runID != runID {
		return
	}
	stream.cancel = nil
	stream.done = nil
}

func (s *Service) ensureProjectRuntimeTopicStreaming(topic string, projectID uint64) error {
	if s == nil {
		return errors.New("project service is unavailable")
	}
	if s.realtimeHub == nil {
		return errors.New("realtime hub is unavailable")
	}
	streamer, err := s.projectRuntimeTopicStreamer()
	if err != nil {
		return err
	}
	return streamer.EnsureTopic(topic, projectID)
}

func (s *Service) projectRuntimeTopicStreamer() (*projectRuntimeTopicStreamer, error) {
	s.streamersMu.Lock()
	defer s.streamersMu.Unlock()
	if s.runtimeTopicStreamer == nil {
		streamer, err := newProjectRuntimeTopicStreamer(s.realtimeHub, s.logger, s)
		if err != nil {
			return nil, err
		}
		s.runtimeTopicStreamer = streamer
	}
	return s.runtimeTopicStreamer, nil
}

func (s *Service) buildProjectRuntimeRealtimePayload(
	ctx context.Context,
	topic string,
	projectID uint64,
) (projectRuntimeRealtimePayload, error) {
	detail, err := s.Get(ctx, projectID)
	if err != nil {
		return projectRuntimeRealtimePayload{}, err
	}
	overview, err := s.Overview(ctx, projectID)
	if err != nil {
		return projectRuntimeRealtimePayload{}, err
	}
	services, err := s.Services(ctx, projectID)
	if err != nil {
		return projectRuntimeRealtimePayload{}, err
	}
	return projectRuntimeRealtimePayload{
		Topic:         topic,
		ApplicationID: detail.ApplicationId,
		PublishedAt:   time.Now().UTC(),
		Detail:        detail,
		Overview:      overview,
		Services:      services,
	}, nil
}

func (s *Service) buildProjectListSummaryRealtimePayload(
	ctx context.Context,
	topic string,
) (projectListSummaryRealtimePayload, error) {
	items := make([]projectListSummaryRealtimeItem, 0)
	offset := 0
	for {
		result, err := s.listAllApplicationsForRealtime(ctx, ListQuery{
			Limit:  maxProjectListLimit,
			Offset: offset,
		})
		if err != nil {
			return projectListSummaryRealtimePayload{}, err
		}
		for _, item := range result.Items {
			runtimeStatus := generated.ApplicationRuntimeStatusUnknown
			if item.RuntimeStatus != nil {
				runtimeStatus = *item.RuntimeStatus
			}
			items = append(items, projectListSummaryRealtimeItem{
				ApplicationID:   item.ApplicationId,
				RuntimeStatus:   runtimeStatus,
				ServiceCount:    item.ServiceCount,
				ContainerCounts: item.ContainerCounts,
				DriftStatus:     item.DriftStatus,
			})
		}
		offset += len(result.Items)
		if len(result.Items) == 0 || offset >= result.Total {
			break
		}
	}

	return projectListSummaryRealtimePayload{
		Topic:       topic,
		PublishedAt: time.Now().UTC(),
		Items:       items,
	}, nil
}

// listAllApplicationsForRealtime 在不携带请求主体的前提下构建模块拥有的全局快照。
// 它仅由已限制为全量权限订阅者的列表主题发布器调用，不能作为常规用户列表读取路径复用。
func (s *Service) listAllApplicationsForRealtime(ctx context.Context, query ListQuery) (ListResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ListResult{}, err
	}
	if result, handled, err := validateApplicationListQuery(query); handled {
		return result, err
	}
	targets, err := s.listComposeTargets(ctx)
	if err != nil {
		return ListResult{}, err
	}
	targetByID := runtimeTargetLookup(targets)
	if !validRuntimeTargetID(query.RuntimeTargetID, targetByID) {
		return ListResult{}, errProjectInvalidArgument
	}
	storeQuery := toProjectStoreListQuery(query)
	if query.RuntimeStatus != "" {
		return s.listRuntimeStatusPage(ctx, repository, storeQuery, query, targetByID)
	}
	storeResult, err := repository.List(ctx, storeQuery)
	if err != nil {
		return ListResult{}, mapStoreError(err)
	}
	items := s.mapProjectListItems(ctx, storeResult.Items, targetByID, "")
	return ListResult{Items: items, Total: storeResult.Total, Limit: normalizeListLimit(query.Limit), Offset: maxInt(query.Offset, 0)}, nil
}
