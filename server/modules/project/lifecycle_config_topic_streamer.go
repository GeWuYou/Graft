package project

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/logger"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/realtime"
)

type projectLifecycleConfigTopicStreamer struct {
	hub     realtime.Hub
	monitor realtime.TopicSubscriptionMonitor
	logger  *zap.Logger
	service *Service

	mu      sync.Mutex
	streams map[string]*projectLifecycleConfigTopicStream
}

type projectLifecycleConfigTopicStream struct {
	topic              string
	projectID          uint64
	unregisterObserver func()
	cancel             context.CancelFunc
	done               chan struct{}
	runID              uint64
}

// markProjectLifecycleConfigTopicStreamDone 以非阻塞方式通知生命周期配置流已结束。
func markProjectLifecycleConfigTopicStreamDone(done chan struct{}) {
	if done == nil {
		return
	}
	select {
	case done <- struct{}{}:
	default:
	}
}

// omitProjectLifecycleConfigTopicStream 返回移除给定 topic 后的流快照，避免原地修改读侧快照。
func omitProjectLifecycleConfigTopicStream(
	streams map[string]*projectLifecycleConfigTopicStream,
	topic string,
) map[string]*projectLifecycleConfigTopicStream {
	if len(streams) == 0 {
		return streams
	}
	next := make(map[string]*projectLifecycleConfigTopicStream, len(streams))
	for key, stream := range streams {
		if key == topic {
			continue
		}
		next[key] = stream
	}
	return next
}

// newProjectLifecycleConfigTopicStreamer 创建生命周期配置流；logger 为空时使用无操作日志器。
//
//nolint:dupl // 运行时流与生命周期流拥有不同具体类型和生命周期责任，因此保留独立实现。
func newProjectLifecycleConfigTopicStreamer(
	hub realtime.Hub,
	logger *zap.Logger,
	service *Service,
) (*projectLifecycleConfigTopicStreamer, error) {
	if hub == nil {
		return nil, errors.New("realtime hub is unavailable")
	}
	monitor, ok := hub.(realtime.TopicSubscriptionMonitor)
	if !ok {
		return nil, errors.New("realtime hub does not support topic subscription monitoring")
	}
	if service == nil {
		return nil, errors.New("project lifecycle configuration service is unavailable")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &projectLifecycleConfigTopicStreamer{
		hub:     hub,
		monitor: monitor,
		logger:  logger,
		service: service,
		streams: make(map[string]*projectLifecycleConfigTopicStream),
	}, nil
}

func (s *projectLifecycleConfigTopicStreamer) EnsureTopic(topic string, projectID uint64) error {
	// 生命周期配置流与运行时流共享订阅驱动，但保留独立状态以隔离各自的发布和关闭责任。
	if s == nil {
		return errors.New("project lifecycle configuration topic streamer is unavailable")
	}
	s.mu.Lock()
	if s.streams[topic] != nil {
		s.mu.Unlock()
		return nil
	}
	stream := &projectLifecycleConfigTopicStream{topic: topic, projectID: projectID}
	s.streams[topic] = stream
	s.mu.Unlock()

	unregister, err := s.monitor.RegisterTopicObserver(topic, func(string) {
		s.start(topic)
	}, func(string) {
		if err := s.stop(context.Background(), topic); err != nil {
			logger.Category(s.logger, logger.CategoryComposeRuntime).Warn(
				"stop project lifecycle configuration stream failed",
				zap.String("topic", logsafe.SanitizeText(topic)),
				zap.Error(err),
			)
		}
	})
	if err != nil {
		s.mu.Lock()
		s.streams = omitProjectLifecycleConfigTopicStream(s.streams, topic)
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
func (s *projectLifecycleConfigTopicStreamer) Close(ctx context.Context) error {
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
	// 只有停止成功才移除主题登记；超时保留状态，避免遗留协程失去可追踪的所有权。
	for _, topic := range topics {
		stopErr := s.stop(ctx, topic)
		closeErr = errors.Join(closeErr, stopErr)
		if stopErr != nil {
			continue
		}
		s.mu.Lock()
		stream := s.streams[topic]
		s.streams = omitProjectLifecycleConfigTopicStream(s.streams, topic)
		s.mu.Unlock()
		if stream != nil && stream.unregisterObserver != nil {
			stream.unregisterObserver()
		}
	}
	return closeErr
}

func (s *projectLifecycleConfigTopicStreamer) start(topic string) {
	s.mu.Lock()
	stream := s.streams[topic]
	if stream == nil || stream.cancel != nil {
		s.mu.Unlock()
		return
	}
	parent := s.service.realtimeStreamContext()
	if parent == nil {
		s.mu.Unlock()
		return
	}
	runCtx, cancel := context.WithCancel(parent)
	stream.cancel = cancel
	stream.done = make(chan struct{}, 1)
	stream.runID++
	runID := stream.runID
	projectID := stream.projectID
	done := stream.done
	s.mu.Unlock()

	go func() {
		defer markProjectLifecycleConfigTopicStreamDone(done)
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

func (s *projectLifecycleConfigTopicStreamer) publish(parent context.Context, topic string, projectID uint64) {
	if s == nil || s.service == nil {
		return
	}
	ctx, cancel := context.WithTimeout(parent, projectDetailTopicRefreshInterval)
	defer cancel()
	payload, err := s.service.buildProjectLifecycleConfigRealtimePayload(ctx, topic, projectID)
	if err != nil {
		logger.Category(s.logger, logger.CategoryComposeRuntime).Warn(
			"publish project lifecycle configuration snapshot failed",
			zap.String("topic", logsafe.SanitizeText(topic)),
			zap.Error(err),
		)
		return
	}
	s.hub.Publish(topic, payload)
}

//nolint:dupl
func (s *projectLifecycleConfigTopicStreamer) stop(ctx context.Context, topic string) error {
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

func (s *projectLifecycleConfigTopicStreamer) clearRun(topic string, runID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stream := s.streams[topic]
	if stream == nil || stream.runID != runID {
		return
	}
	stream.cancel = nil
	stream.done = nil
}

func (s *Service) ensureProjectLifecycleConfigTopicStreaming(topic string, projectID uint64) error {
	if s == nil {
		return errors.New("project service is unavailable")
	}
	if s.realtimeHub == nil {
		return errors.New("realtime hub is unavailable")
	}
	streamer, err := s.projectLifecycleConfigTopicStreamer()
	if err != nil {
		return err
	}
	return streamer.EnsureTopic(topic, projectID)
}

func (s *Service) projectLifecycleConfigTopicStreamer() (*projectLifecycleConfigTopicStreamer, error) {
	s.streamersMu.Lock()
	defer s.streamersMu.Unlock()
	if s.lifecycleConfigTopicStreamer == nil {
		streamer, err := newProjectLifecycleConfigTopicStreamer(s.realtimeHub, s.logger, s)
		if err != nil {
			return nil, err
		}
		s.lifecycleConfigTopicStreamer = streamer
	}
	return s.lifecycleConfigTopicStreamer, nil
}

func (s *Service) buildProjectLifecycleConfigRealtimePayload(
	ctx context.Context,
	topic string,
	projectID uint64,
) (projectLifecycleConfigRealtimePayload, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return projectLifecycleConfigRealtimePayload{}, err
	}
	return projectLifecycleConfigRealtimePayload{
		Topic:         topic,
		ApplicationID: aggregate.Application.ApplicationID,
		PublishedAt:   time.Now().UTC(),
		Detail: toProjectDetailResponseWithManagedRoot(
			aggregate,
			s.readyManagedRootDirectory(ctx),
			nil,
			errProjectRuntimeUnavailable,
		),
	}, nil
}
