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

type projectListTopicStreamer struct {
	hub     realtime.Hub
	monitor realtime.TopicSubscriptionMonitor
	logger  *zap.Logger
	service *Service

	mu      sync.Mutex
	streams map[string]*projectListTopicStream
}

type projectListTopicStream struct {
	topic              string
	unregisterObserver func()
	cancel             context.CancelFunc
	done               chan struct{}
	runID              uint64
}

func markProjectListTopicStreamDone(done chan struct{}) {
	if done == nil {
		return
	}
	select {
	case done <- struct{}{}:
	default:
	}
}

func omitProjectListTopicStream(
	streams map[string]*projectListTopicStream,
	topic string,
) map[string]*projectListTopicStream {
	if len(streams) == 0 {
		return streams
	}
	next := make(map[string]*projectListTopicStream, len(streams))
	for key, stream := range streams {
		if key == topic {
			continue
		}
		next[key] = stream
	}
	return next
}

//nolint:dupl // 应用列表与日志流使用相同的有界实时生命周期，但拥有不同载荷和订阅职责。
func newProjectListTopicStreamer(hub realtime.Hub, logger *zap.Logger, service *Service) (*projectListTopicStreamer, error) {
	if hub == nil {
		return nil, errors.New("realtime hub is unavailable")
	}
	monitor, ok := hub.(realtime.TopicSubscriptionMonitor)
	if !ok {
		return nil, errors.New("realtime hub does not support topic subscription monitoring")
	}
	if service == nil {
		return nil, errors.New("project list service is unavailable")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &projectListTopicStreamer{
		hub:     hub,
		monitor: monitor,
		logger:  logger,
		service: service,
		streams: make(map[string]*projectListTopicStream),
	}, nil
}

func (s *projectListTopicStreamer) EnsureTopic(topic string) error {
	// 主题只登记一次；真正的刷新协程由订阅观察器在首个订阅到来时启动。
	if s == nil {
		return errors.New("project list topic streamer is unavailable")
	}
	s.mu.Lock()
	if s.streams[topic] != nil {
		s.mu.Unlock()
		return nil
	}
	stream := &projectListTopicStream{topic: topic}
	s.streams[topic] = stream
	s.mu.Unlock()

	unregister, err := s.monitor.RegisterTopicObserver(topic, func(string) {
		s.start(topic)
	}, func(string) {
		if err := s.stop(context.Background(), topic); err != nil {
			logger.Category(s.logger, logger.CategoryComposeRuntime).Warn("stop project list stream failed", zap.String("topic", logsafe.SanitizeText(topic)), zap.Error(err))
		}
	})
	if err != nil {
		s.mu.Lock()
		s.streams = omitProjectListTopicStream(s.streams, topic)
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

//nolint:dupl // 应用列表与日志流使用相同的有界实时生命周期，但拥有不同载荷和订阅职责。
func (s *projectListTopicStreamer) Close(ctx context.Context) error {
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
	// 先请求停止并等待协程，再注销观察器，避免关闭后仍有发布回调访问状态。
	for _, topic := range topics {
		closeErr = errors.Join(closeErr, s.stop(ctx, topic))
		s.mu.Lock()
		stream := s.streams[topic]
		s.streams = omitProjectListTopicStream(s.streams, topic)
		s.mu.Unlock()
		if stream != nil && stream.unregisterObserver != nil {
			stream.unregisterObserver()
		}
	}
	return closeErr
}

func (s *projectListTopicStreamer) start(topic string) {
	s.mu.Lock()
	stream := s.streams[topic]
	if stream == nil || stream.cancel != nil {
		s.mu.Unlock()
		return
	}
	// 每个主题最多保留一个运行实例；runID 用于防止旧协程清理新一轮运行状态。
	runCtx, cancel := context.WithCancel(context.Background())
	stream.cancel = cancel
	stream.done = make(chan struct{}, 1)
	stream.runID++
	runID := stream.runID
	done := stream.done
	s.mu.Unlock()

	go func() {
		defer markProjectListTopicStreamDone(done)
		s.publish(topic)
		ticker := time.NewTicker(projectDetailTopicRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				s.clearRun(topic, runID)
				return
			case <-ticker.C:
				s.publish(topic)
			}
		}
	}()
}

func (s *projectListTopicStreamer) publish(topic string) {
	if s == nil || s.service == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), projectDetailTopicRefreshInterval)
	defer cancel()
	payload, err := s.service.buildProjectListSummaryRealtimePayload(ctx, topic)
	if err != nil {
		logger.Category(s.logger, logger.CategoryComposeRuntime).Warn("publish project list realtime snapshot failed", zap.String("topic", logsafe.SanitizeText(topic)), zap.Error(err))
		return
	}
	s.hub.Publish(topic, payload)
}

//nolint:dupl // 应用列表与日志流使用相同的有界实时生命周期，但拥有不同载荷和订阅职责。
func (s *projectListTopicStreamer) stop(ctx context.Context, topic string) error {
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

func (s *projectListTopicStreamer) clearRun(topic string, runID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stream := s.streams[topic]
	if stream == nil || stream.runID != runID {
		return
	}
	stream.cancel = nil
	stream.done = nil
}

func (s *Service) ensureProjectListSummaryTopicStreaming(topic string) error {
	if s == nil {
		return errors.New("project service is unavailable")
	}
	if s.realtimeHub == nil {
		return errors.New("realtime hub is unavailable")
	}
	streamer, err := s.projectListTopicStreamer()
	if err != nil {
		return err
	}
	return streamer.EnsureTopic(topic)
}

func (s *Service) projectListTopicStreamer() (*projectListTopicStreamer, error) {
	s.streamersMu.Lock()
	defer s.streamersMu.Unlock()
	if s.listTopicStreamer == nil {
		streamer, err := newProjectListTopicStreamer(s.realtimeHub, s.logger, s)
		if err != nil {
			return nil, err
		}
		s.listTopicStreamer = streamer
	}
	return s.listTopicStreamer, nil
}
