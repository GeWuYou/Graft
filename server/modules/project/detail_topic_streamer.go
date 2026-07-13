package project

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

	generated "graft/server/internal/contract/openapi/generated"
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

// markProjectRuntimeTopicStreamDone signals stream completion without blocking when done is available.
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

// newProjectRuntimeTopicStreamer creates a project runtime topic streamer with the provided hub, logger, and service.
// It returns an error if the hub or service is unavailable, or if the hub does not support topic subscription monitoring.
//
//nolint:dupl // Runtime and lifecycle streams keep distinct concrete stream types and lifecycle ownership.
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
			s.logger.Warn("stop project runtime stream failed", zap.String("topic", logsafe.SanitizeText(topic)), zap.Error(err))
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
	runCtx, cancel := context.WithCancel(context.Background())
	stream.cancel = cancel
	stream.done = make(chan struct{}, 1)
	stream.runID++
	runID := stream.runID
	projectID := stream.projectID
	done := stream.done
	s.mu.Unlock()

	go func() {
		defer markProjectRuntimeTopicStreamDone(done)
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

func (s *projectRuntimeTopicStreamer) publish(topic string, projectID uint64) {
	if s == nil || s.service == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), projectDetailTopicRefreshInterval)
	defer cancel()
	payload, err := s.service.buildProjectRuntimeRealtimePayload(ctx, topic, projectID)
	if err != nil {
		s.logger.Warn("publish project runtime realtime snapshot failed", zap.String("topic", logsafe.SanitizeText(topic)), zap.Error(err))
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
		Topic:       topic,
		ProjectID:   mustGeneratedID(projectID),
		PublishedAt: time.Now().UTC(),
		Detail:      detail,
		Overview:    overview,
		Services:    services,
	}, nil
}

func (s *Service) buildProjectListSummaryRealtimePayload(
	ctx context.Context,
	topic string,
) (projectListSummaryRealtimePayload, error) {
	items := make([]projectListSummaryRealtimeItem, 0)
	offset := 0
	for {
		result, err := s.List(ctx, ListQuery{
			Limit:  maxProjectListLimit,
			Offset: offset,
		})
		if err != nil {
			return projectListSummaryRealtimePayload{}, err
		}
		for _, item := range result.Items {
			runtimeStatus := generated.ProjectRuntimeStatusUnknown
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
