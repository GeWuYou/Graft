package project

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

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
			s.logger.Warn("stop project list stream failed", zap.String("topic", logsafe.SanitizeText(topic)), zap.Error(err))
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
		s.logger.Warn("publish project list realtime snapshot failed", zap.String("topic", logsafe.SanitizeText(topic)), zap.Error(err))
		return
	}
	s.hub.Publish(topic, payload)
}

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

