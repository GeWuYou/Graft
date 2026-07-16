package project

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/zap"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
)

type projectLogTopicStreamer struct {
	hub     realtime.Hub
	monitor realtime.TopicSubscriptionMonitor
	logger  *zap.Logger
	service *Service

	mu      sync.Mutex
	streams map[string]*projectLogTopicStream
}

type projectLogTopicStream struct {
	topic              string
	projectID          uint64
	query              LogQuery
	unregisterObserver func()
	cancel             context.CancelFunc
	done               chan struct{}
	runID              uint64
}

func markProjectLogTopicStreamDone(done chan struct{}) {
	if done == nil {
		return
	}
	select {
	case done <- struct{}{}:
	default:
	}
}

func omitProjectLogTopicStream(
	streams map[string]*projectLogTopicStream,
	topic string,
) map[string]*projectLogTopicStream {
	if len(streams) == 0 {
		return streams
	}
	next := make(map[string]*projectLogTopicStream, len(streams))
	for key, stream := range streams {
		if key == topic {
			continue
		}
		next[key] = stream
	}
	return next
}

//nolint:dupl
func newProjectLogTopicStreamer(hub realtime.Hub, logger *zap.Logger, service *Service) (*projectLogTopicStreamer, error) {
	if hub == nil {
		return nil, errors.New("realtime hub is unavailable")
	}
	monitor, ok := hub.(realtime.TopicSubscriptionMonitor)
	if !ok {
		return nil, errors.New("realtime hub does not support topic subscription monitoring")
	}
	if service == nil {
		return nil, errors.New("project log service is unavailable")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &projectLogTopicStreamer{
		hub:     hub,
		monitor: monitor,
		logger:  logger,
		service: service,
		streams: make(map[string]*projectLogTopicStream),
	}, nil
}

func (s *projectLogTopicStreamer) EnsureTopic(topic string, projectID uint64, query LogQuery) error {
	// 日志流保存首次登记时的查询条件；重复订阅只增加观察者引用，不重建流参数。
	if s == nil {
		return errors.New("project log topic streamer is unavailable")
	}
	s.mu.Lock()
	if existing := s.streams[topic]; existing != nil {
		s.mu.Unlock()
		return nil
	}
	stream := &projectLogTopicStream{topic: topic, projectID: projectID, query: query}
	s.streams[topic] = stream
	s.mu.Unlock()

	unregister, err := s.monitor.RegisterTopicObserver(topic, func(string) {
		s.start(topic)
	}, func(string) {
		if err := s.stop(context.Background(), topic); err != nil {
			s.logger.Warn("stop project log stream failed", zap.String("topic", logsafe.SanitizeText(topic)), zap.Error(err))
		}
	})
	if err != nil {
		s.mu.Lock()
		s.streams = omitProjectLogTopicStream(s.streams, topic)
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
func (s *projectLogTopicStreamer) Close(ctx context.Context) error {
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
	// 先取消并等待日志读取，再注销观察器，保证外部运行时句柄不会在关闭后继续被使用。
	for _, topic := range topics {
		closeErr = errors.Join(closeErr, s.stop(ctx, topic))
		s.mu.Lock()
		stream := s.streams[topic]
		s.streams = omitProjectLogTopicStream(s.streams, topic)
		s.mu.Unlock()
		if stream != nil && stream.unregisterObserver != nil {
			stream.unregisterObserver()
		}
	}
	return closeErr
}

func (s *projectLogTopicStreamer) start(topic string) {
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
	query := stream.query
	done := stream.done
	s.mu.Unlock()

	go func() {
		defer markProjectLogTopicStreamDone(done)
		if err := s.service.streamProjectLogs(runCtx, topic, projectID, query); err != nil && !errors.Is(err, context.Canceled) {
			s.logger.Warn("project log stream stopped with error", zap.String("topic", logsafe.SanitizeText(topic)), zap.Error(err))
		}
		s.clearRun(topic, runID)
	}()
}

//nolint:dupl
func (s *projectLogTopicStreamer) stop(ctx context.Context, topic string) error {
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

func (s *projectLogTopicStreamer) clearRun(topic string, runID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stream := s.streams[topic]
	if stream == nil || stream.runID != runID {
		return
	}
	stream.cancel = nil
	stream.done = nil
}

func (s *Service) ensureProjectLogsTopicStreaming(topic string, projectID uint64, query LogQuery) error {
	if s == nil {
		return errors.New("project service is unavailable")
	}
	if s.realtimeHub == nil {
		return errors.New("realtime hub is unavailable")
	}
	if _, err := normalizeProjectLogQuery(query); err != nil {
		return err
	}
	streamer, err := s.projectLogTopicStreamer()
	if err != nil {
		return err
	}
	return streamer.EnsureTopic(topic, projectID, query)
}

func (s *Service) projectLogTopicStreamer() (*projectLogTopicStreamer, error) {
	s.streamersMu.Lock()
	defer s.streamersMu.Unlock()
	if s.logTopicStreamer == nil {
		streamer, err := newProjectLogTopicStreamer(s.realtimeHub, s.logger, s)
		if err != nil {
			return nil, err
		}
		s.logTopicStreamer = streamer
	}
	return s.logTopicStreamer, nil
}

func (s *Service) streamProjectLogs(ctx context.Context, topic string, projectID uint64, query LogQuery) error {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return err
	}
	if s.logReader == nil {
		return errProjectRuntimeUnavailable
	}
	normalized, err := normalizeProjectLogQuery(query)
	if err != nil {
		return err
	}
	s.logProjectLogDiagnostic("follow-started",
		zap.Uint64("project_id", projectID),
		zap.String("topic", topic),
		zap.Int("requested_tail", normalized.Tail),
		zap.Bool("follow_only", true),
	)
	emittedCount := 0
	err = s.logReader.StreamProjectLogs(
		ctx,
		aggregate.Project.HostScope,
		aggregate.Project.CanonicalProjectName,
		toContainerProjectLogFollowQuery(normalized),
		func(entry moduleapi.ContainerProjectLogEntry) error {
			emittedCount++
			payload := projectLogsRealtimePayload{
				Topic: topic,
				Entry: generated.ProjectLogEntry{
					ContainerId:   entry.ContainerID,
					ContainerName: entry.ContainerName,
					ServiceName:   entry.ServiceName,
					Line:          entry.Line,
					Stream:        generated.ProjectLogEntryStream(entry.Stream),
					OccurredAt:    parseGeneratedLogTime(entry.OccurredAt),
					Source: generated.ProjectLogEntrySource{
						ContainerId:   entry.ContainerID,
						ContainerName: entry.ContainerName,
						ServiceName:   entry.ServiceName,
					},
				},
			}
			s.realtimeHub.Publish(topic, payload)
			return nil
		},
	)
	s.logProjectLogDiagnostic("follow-stopped",
		zap.Uint64("project_id", projectID),
		zap.String("topic", topic),
		zap.Int("emitted_count", emittedCount),
		zap.Error(err),
	)
	return err
}
