package container

import (
	"context"
	"errors"
	"strings"
	"sync"

	"go.uber.org/zap"

	"graft/server/internal/logger/logsafe"
	"graft/server/internal/realtime"
)

type logTopicStreamer struct {
	hub           realtime.Hub
	monitor       realtime.TopicSubscriptionMonitor
	logger        *zap.Logger
	runtimeLoader func() (Runtime, error)

	mu      sync.Mutex
	streams map[string]*logTopicStream
}

type logTopicStream struct {
	topic              string
	ref                Ref
	query              LogQuery
	canonicalID        string
	unregisterObserver func()
	cancel             context.CancelFunc
	done               chan struct{}
	runID              uint64
}

type containerLogPublished struct {
	Topic string    `json:"topic"`
	ID    string    `json:"id"`
	Entry LogEntry  `json:"entry"`
}

// newLogTopicStreamer 创建一个用于按 topic 管理容器日志流的实例。
// 它要求 realtime hub 可用且支持 topic 订阅监控，并且提供 runtime 加载器；
// 当 logger 为空时会使用无操作 logger。
func newLogTopicStreamer(
	hub realtime.Hub,
	logger *zap.Logger,
	runtimeLoader func() (Runtime, error),
) (*logTopicStreamer, error) {
	if hub == nil {
		return nil, errors.New("realtime hub is unavailable")
	}
	monitor, ok := hub.(realtime.TopicSubscriptionMonitor)
	if !ok {
		return nil, errors.New("realtime hub does not support topic subscription monitoring")
	}
	if runtimeLoader == nil {
		return nil, errors.New("container runtime loader is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &logTopicStreamer{
		hub:           hub,
		monitor:       monitor,
		logger:        logger,
		runtimeLoader: runtimeLoader,
		streams:       make(map[string]*logTopicStream),
	}, nil
}

func (s *logTopicStreamer) EnsureTopic(ctx context.Context, topic string, ref Ref, query LogQuery) error {
	if s == nil {
		return errors.New("container log topic streamer is unavailable")
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return realtime.ErrTopicRequired
	}

	stream := &logTopicStream{
		topic: topic,
		ref:   ref,
		query: query,
	}

	s.mu.Lock()
	if existing := s.streams[topic]; existing != nil {
		s.mu.Unlock()
		return nil
	}
	s.streams[topic] = stream
	s.mu.Unlock()

	runtime, err := s.runtimeLoader()
	if err != nil {
		s.mu.Lock()
		delete(s.streams, topic)
		s.mu.Unlock()
		return err
	}
	detail, err := runtime.Detail(ctx, ref)
	if err != nil {
		s.mu.Lock()
		delete(s.streams, topic)
		s.mu.Unlock()
		return err
	}
	stream.canonicalID = strings.TrimSpace(detail.ID)
	if stream.canonicalID == "" {
		stream.canonicalID = strings.TrimSpace(ref.Value)
	}

	unregister, err := s.monitor.RegisterTopicObserver(topic, func(_ string) {
		s.start(topic)
	}, func(_ string) {
		_ = s.stop(context.Background(), topic)
	})
	if err != nil {
		s.mu.Lock()
		delete(s.streams, topic)
		s.mu.Unlock()
		return err
	}
	stream.unregisterObserver = unregister
	return nil
}

func (s *logTopicStreamer) Close(ctx context.Context) error {
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
	for _, topic := range topics {
		if err := s.stop(ctx, topic); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
		s.mu.Lock()
		stream := s.streams[topic]
		delete(s.streams, topic)
		s.mu.Unlock()
		if stream != nil && stream.unregisterObserver != nil {
			stream.unregisterObserver()
		}
	}
	return closeErr
}

func (s *logTopicStreamer) start(topic string) {
	s.mu.Lock()
	stream := s.streams[topic]
	if stream == nil || stream.cancel != nil {
		s.mu.Unlock()
		return
	}
	runCtx, cancel := context.WithCancel(context.Background())
	stream.cancel = cancel
	stream.done = make(chan struct{})
	stream.runID++
	runID := stream.runID
	ref := stream.ref
	query := stream.query
	canonicalID := stream.canonicalID
	done := stream.done
	s.mu.Unlock()

	go func() {
		defer close(done)
		runtime, err := s.runtimeLoader()
		if err != nil {
			s.logger.Warn(
				"start container log stream failed",
				zap.String("topic", logsafe.SanitizeText(topic)),
				zap.Error(err),
			)
			s.clearRun(topic, runID)
			return
		}
		err = runtime.StreamLogs(runCtx, ref, query, func(chunk LogChunk) error {
			s.hub.Publish(topic, containerLogPublished{
				Topic: topic,
				ID:    canonicalID,
				Entry: logEntryFromChunk(chunk),
			})
			return nil
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			s.logger.Warn(
				"container log stream stopped with error",
				zap.String("topic", logsafe.SanitizeText(topic)),
				zap.Error(err),
			)
		}
		s.clearRun(topic, runID)
	}()
}

func (s *logTopicStreamer) stop(ctx context.Context, topic string) error {
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

func (s *logTopicStreamer) clearRun(topic string, runID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stream := s.streams[topic]
	if stream == nil || stream.runID != runID {
		return
	}
	stream.cancel = nil
	stream.done = nil
}

func (s *service) ensureLogTopicStreaming(ctx context.Context, topic string, ref Ref, query LogQuery) error {
	if s == nil {
		return errors.New("container service is unavailable")
	}
	if s.realtimeHub == nil {
		return errors.New("realtime hub is unavailable")
	}
	s.logTopicStreamerMu.Lock()
	streamer := s.logTopicStreamer
	if streamer == nil {
		factory := s.logTopicStreamerFactory
		if factory == nil {
			factory = newLogTopicStreamer
		}
		var err error
		streamer, err = factory(s.realtimeHub, s.logger, s.runtimeForRequest)
		if err != nil {
			s.logTopicStreamerMu.Unlock()
			return err
		}
		s.logTopicStreamer = streamer
	}
	s.logTopicStreamerMu.Unlock()
	return streamer.EnsureTopic(ctx, topic, ref, query)
}
