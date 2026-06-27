package container

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/logger/logsafe"
	"graft/server/internal/realtime"
	containercontract "graft/server/modules/container/contract"
)

type runtimeEventManager struct {
	hub           realtime.Publisher
	logger        *zap.Logger
	runtimeLoader func() (Runtime, error)
	historyLimit  int
	historyTTL    time.Duration
	streamContext RuntimeEventStreamContext

	mu          sync.RWMutex
	seqByID     map[string]int64
	historyByID map[string][]RuntimeEventRecord
	lastSeenAt  map[string]time.Time

	runMu   sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	started bool
}

func newRuntimeEventManager(
	hub realtime.Publisher,
	logger *zap.Logger,
	runtimeLoader func() (Runtime, error),
	streamContext RuntimeEventStreamContext,
) *runtimeEventManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	limit := defaultRuntimeEventsHistoryLimit
	if limit <= 0 {
		limit = 1
	}
	return &runtimeEventManager{
		hub:           hub,
		logger:        logger,
		runtimeLoader: runtimeLoader,
		historyLimit:  limit,
		historyTTL:    defaultRuntimeEventsHistoryTTL,
		streamContext: streamContext,
		seqByID:       make(map[string]int64),
		historyByID:   make(map[string][]RuntimeEventRecord),
		lastSeenAt:    make(map[string]time.Time),
	}
}

func (m *runtimeEventManager) Start(ctx context.Context) error {
	if m == nil || m.runtimeLoader == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	m.runMu.Lock()
	defer m.runMu.Unlock()
	if m.started {
		return nil
	}

	runtime, err := m.runtimeLoader()
	if err != nil {
		return err
	}
	source, ok := runtime.(RuntimeEventSource)
	if !ok {
		return nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	m.cancel = cancel
	m.done = done
	m.started = true

	go m.run(runCtx, done, source)
	return nil
}

func (m *runtimeEventManager) Stop(ctx context.Context) error {
	if m == nil {
		return nil
	}

	m.runMu.Lock()
	if !m.started {
		m.runMu.Unlock()
		return nil
	}
	cancel := m.cancel
	done := m.done
	m.cancel = nil
	m.done = nil
	m.started = false
	m.runMu.Unlock()

	if cancel != nil {
		cancel()
	}
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

func (m *runtimeEventManager) run(ctx context.Context, done chan struct{}, source RuntimeEventSource) {
	defer close(done)
	err := source.StreamRuntimeEvents(ctx, func(candidate RuntimeEventCandidate) error {
		return m.Append(candidate)
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		m.logger.Warn("container runtime event stream stopped with error", zap.Error(err))
	}
}

func (m *runtimeEventManager) Append(candidate RuntimeEventCandidate) error {
	if m == nil {
		return errors.New("container runtime event manager is unavailable")
	}
	event, err := newRuntimeEvent(candidate)
	if err != nil {
		return err
	}

	record := m.appendRecord(event)
	if m.hub != nil {
		topic := containercontract.ContainerEventsTopicPrefix + event.ResourceID
		m.hub.Publish(topic, struct {
			ResourceID string                    `json:"resource_id"`
			Context    RuntimeEventStreamContext `json:"context"`
			Record     RuntimeEventRecord        `json:"record"`
		}{
			ResourceID: event.ResourceID,
			Context:    m.streamContext,
			Record:     record,
		})
	}
	return nil
}

func (m *runtimeEventManager) appendRecord(event RuntimeEvent) RuntimeEventRecord {
	m.mu.Lock()
	defer m.mu.Unlock()

	resourceID := strings.TrimSpace(event.ResourceID)
	nextSeq := m.seqByID[resourceID] + 1
	m.seqByID[resourceID] = nextSeq
	record := RuntimeEventRecord{
		Seq:   nextSeq,
		Event: event,
	}
	history := append(m.historyByID[resourceID], record)
	if len(history) > m.historyLimit {
		history = history[len(history)-m.historyLimit:]
	}
	m.historyByID[resourceID] = history
	m.lastSeenAt[resourceID] = time.Now().UTC()
	m.evictExpiredLocked(time.Now().UTC())
	return record
}

func (m *runtimeEventManager) History(resourceID string) RuntimeEventsHistory {
	resourceID = strings.TrimSpace(resourceID)
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := append([]RuntimeEventRecord(nil), m.historyByID[resourceID]...)
	return RuntimeEventsHistory{
		ResourceID: resourceID,
		Context:    m.streamContext,
		Items:      items,
	}
}

func (m *runtimeEventManager) evictExpiredLocked(now time.Time) {
	if m.historyTTL <= 0 {
		return
	}
	for resourceID, lastSeenAt := range m.lastSeenAt {
		if now.Sub(lastSeenAt) < m.historyTTL {
			continue
		}
		delete(m.lastSeenAt, resourceID)
		delete(m.seqByID, resourceID)
		delete(m.historyByID, resourceID)
	}
}

func (m *runtimeEventManager) LogCandidateDrop(candidate RuntimeEventCandidate, err error) {
	if m == nil || m.logger == nil || err == nil {
		return
	}
	m.logger.Warn(
		"drop invalid container runtime event candidate",
		zap.String("resourceID", logsafe.SanitizeText(strings.TrimSpace(candidate.ResourceID))),
		zap.String("eventType", candidate.EventType.String()),
		zap.Error(err),
	)
}
