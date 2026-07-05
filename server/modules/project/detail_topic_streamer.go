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

type projectDetailTopicStreamer struct {
	hub     realtime.Hub
	monitor realtime.TopicSubscriptionMonitor
	logger  *zap.Logger
	service *Service

	mu      sync.Mutex
	streams map[string]*projectDetailTopicStream
}

type projectDetailTopicStream struct {
	topic              string
	projectID          uint64
	unregisterObserver func()
	cancel             context.CancelFunc
	done               chan struct{}
	runID              uint64
}

func markProjectDetailTopicStreamDone(done chan struct{}) {
	if done == nil {
		return
	}
	select {
	case done <- struct{}{}:
	default:
	}
}

func omitProjectDetailTopicStream(
	streams map[string]*projectDetailTopicStream,
	topic string,
) map[string]*projectDetailTopicStream {
	if len(streams) == 0 {
		return streams
	}
	next := make(map[string]*projectDetailTopicStream, len(streams))
	for key, stream := range streams {
		if key == topic {
			continue
		}
		next[key] = stream
	}
	return next
}

//nolint:dupl
func newProjectDetailTopicStreamer(hub realtime.Hub, logger *zap.Logger, service *Service) (*projectDetailTopicStreamer, error) {
	if hub == nil {
		return nil, errors.New("realtime hub is unavailable")
	}
	monitor, ok := hub.(realtime.TopicSubscriptionMonitor)
	if !ok {
		return nil, errors.New("realtime hub does not support topic subscription monitoring")
	}
	if service == nil {
		return nil, errors.New("project detail service is unavailable")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &projectDetailTopicStreamer{
		hub:     hub,
		monitor: monitor,
		logger:  logger,
		service: service,
		streams: make(map[string]*projectDetailTopicStream),
	}, nil
}

func (s *projectDetailTopicStreamer) EnsureTopic(topic string, projectID uint64) error {
	if s == nil {
		return errors.New("project detail topic streamer is unavailable")
	}
	s.mu.Lock()
	if s.streams[topic] != nil {
		s.mu.Unlock()
		return nil
	}
	stream := &projectDetailTopicStream{topic: topic, projectID: projectID}
	s.streams[topic] = stream
	s.mu.Unlock()

	unregister, err := s.monitor.RegisterTopicObserver(topic, func(string) {
		s.start(topic)
	}, func(string) {
		if err := s.stop(context.Background(), topic); err != nil {
			s.logger.Warn("stop project detail stream failed", zap.String("topic", logsafe.SanitizeText(topic)), zap.Error(err))
		}
	})
	if err != nil {
		s.mu.Lock()
		s.streams = omitProjectDetailTopicStream(s.streams, topic)
		s.mu.Unlock()
		return err
	}
	stream.unregisterObserver = unregister
	return nil
}

//nolint:dupl
func (s *projectDetailTopicStreamer) Close(ctx context.Context) error {
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
		s.streams = omitProjectDetailTopicStream(s.streams, topic)
		s.mu.Unlock()
		if stream != nil && stream.unregisterObserver != nil {
			stream.unregisterObserver()
		}
	}
	return closeErr
}

func (s *projectDetailTopicStreamer) start(topic string) {
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
	projectID := stream.projectID
	done := stream.done
	s.mu.Unlock()

	go func() {
		defer markProjectDetailTopicStreamDone(done)
		s.publish(topic, projectID)
		ticker := time.NewTicker(projectDetailTopicRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				s.clearRun(topic, runID)
				return
			case <-ticker.C:
				s.publish(topic, projectID)
			}
		}
	}()
}

func (s *projectDetailTopicStreamer) publish(topic string, projectID uint64) {
	if s == nil || s.service == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), projectDetailTopicRefreshInterval)
	defer cancel()
	payload, err := s.service.buildProjectDetailRealtimePayload(ctx, topic, projectID)
	if err != nil {
		s.logger.Warn("publish project detail realtime snapshot failed", zap.String("topic", logsafe.SanitizeText(topic)), zap.Error(err))
		return
	}
	s.hub.Publish(topic, payload)
}

//nolint:dupl
func (s *projectDetailTopicStreamer) stop(ctx context.Context, topic string) error {
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

func (s *projectDetailTopicStreamer) clearRun(topic string, runID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stream := s.streams[topic]
	if stream == nil || stream.runID != runID {
		return
	}
	stream.cancel = nil
	stream.done = nil
}

func (s *Service) ensureProjectDetailTopicStreaming(topic string, projectID uint64) error {
	if s == nil {
		return errors.New("project service is unavailable")
	}
	if s.realtimeHub == nil {
		return errors.New("realtime hub is unavailable")
	}
	streamer, err := s.projectDetailTopicStreamer()
	if err != nil {
		return err
	}
	return streamer.EnsureTopic(topic, projectID)
}

func (s *Service) projectDetailTopicStreamer() (*projectDetailTopicStreamer, error) {
	s.streamersMu.Lock()
	defer s.streamersMu.Unlock()
	if s.detailTopicStreamer == nil {
		streamer, err := newProjectDetailTopicStreamer(s.realtimeHub, s.logger, s)
		if err != nil {
			return nil, err
		}
		s.detailTopicStreamer = streamer
	}
	return s.detailTopicStreamer, nil
}

func (s *Service) buildProjectDetailRealtimePayload(
	ctx context.Context,
	topic string,
	projectID uint64,
) (projectDetailRealtimePayload, error) {
	detail, err := s.Get(ctx, projectID)
	if err != nil {
		return projectDetailRealtimePayload{}, err
	}
	overview, err := s.Overview(ctx, projectID)
	if err != nil {
		return projectDetailRealtimePayload{}, err
	}
	services, err := s.Services(ctx, projectID)
	if err != nil {
		return projectDetailRealtimePayload{}, err
	}
	return projectDetailRealtimePayload{
		Topic:       topic,
		ProjectID:   mustGeneratedID(projectID),
		PublishedAt: time.Now().UTC(),
		Detail:      detail,
		Overview:    overview,
		Services:    services,
	}, nil
}
