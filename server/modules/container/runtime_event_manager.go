package container

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/logger/logsafe"
	"graft/server/internal/realtime"
	containercontract "graft/server/modules/container/contract"
)

type runtimeEventSourceRegistration struct {
	name          string
	streamContext RuntimeEventStreamContext
	load          func() (RuntimeEventSource, error)
}

type runtimeEventSourceDiagnostics struct {
	streamStarts   int64
	streamErrors   int64
	invalidDrops   int64
	duplicateDrops int64
	lastError      string
	lastEventAt    time.Time
}

type runtimeEventManagerDiagnostics struct {
	sources map[string]runtimeEventSourceDiagnostics
}

type runtimeEventManager struct {
	hub                  realtime.Publisher
	logger               *zap.Logger
	sourceRegistrations  []runtimeEventSourceRegistration
	historyLimit         int
	historyTTL           time.Duration
	defaultStreamContext RuntimeEventStreamContext

	mu                sync.RWMutex
	seqByID           map[string]int64
	historyByID       map[string][]RuntimeEventRecord
	lastSeenAt        map[string]time.Time
	contextByID       map[string]RuntimeEventStreamContext
	sourceDiagnostics map[string]runtimeEventSourceDiagnostics

	runMu   sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	started bool
}

// newRuntimeEventManager 创建并初始化一个运行时事件管理器。
// 它会在 logger 为空时使用无操作日志器，将历史上限至少设为 1，规范化默认流上下文，并复制来源注册配置。
// 返回配置好的 runtimeEventManager。
func newRuntimeEventManager(
	hub realtime.Publisher,
	logger *zap.Logger,
	sourceRegistrations []runtimeEventSourceRegistration,
	defaultStreamContext RuntimeEventStreamContext,
) *runtimeEventManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	limit := defaultRuntimeEventsHistoryLimit
	if limit <= 0 {
		limit = 1
	}
	return &runtimeEventManager{
		hub:                  hub,
		logger:               logger,
		sourceRegistrations:  append([]runtimeEventSourceRegistration(nil), sourceRegistrations...),
		historyLimit:         limit,
		historyTTL:           defaultRuntimeEventsHistoryTTL,
		defaultStreamContext: normalizeRuntimeEventStreamContext(defaultStreamContext),
		seqByID:              make(map[string]int64),
		historyByID:          make(map[string][]RuntimeEventRecord),
		lastSeenAt:           make(map[string]time.Time),
		contextByID:          make(map[string]RuntimeEventStreamContext),
		sourceDiagnostics:    make(map[string]runtimeEventSourceDiagnostics),
	}
}

func (m *runtimeEventManager) Start(ctx context.Context) error {
	if m == nil {
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

	sources := make([]loadedRuntimeEventSource, 0, len(m.sourceRegistrations))
	for _, registration := range m.sourceRegistrations {
		if registration.load == nil {
			continue
		}
		source, err := registration.load()
		if err != nil {
			return fmt.Errorf(
				"load container runtime event source %q: %w",
				normalizeRuntimeEventSourceName(registration.name),
				err,
			)
		}
		if source == nil {
			continue
		}
		sources = append(sources, loadedRuntimeEventSource{
			name:          normalizeRuntimeEventSourceName(registration.name),
			streamContext: m.effectiveStreamContext(registration.streamContext),
			source:        source,
		})
	}
	if len(sources) == 0 {
		return nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	m.cancel = cancel
	m.done = done
	m.started = true

	go m.run(runCtx, done, sources)
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

type loadedRuntimeEventSource struct {
	name          string
	streamContext RuntimeEventStreamContext
	source        RuntimeEventSource
}

type runtimeEventAppendResult struct {
	record        RuntimeEventRecord
	streamContext RuntimeEventStreamContext
	accepted      bool
}

func (m *runtimeEventManager) run(ctx context.Context, done chan struct{}, sources []loadedRuntimeEventSource) {
	defer m.finishRun(done)
	var waitGroup sync.WaitGroup
	for _, source := range sources {
		waitGroup.Add(1)
		go func(source loadedRuntimeEventSource) {
			defer waitGroup.Done()
			m.runSource(ctx, source)
		}(source)
	}
	waitGroup.Wait()
}

func (m *runtimeEventManager) finishRun(done chan struct{}) {
	if done == nil {
		return
	}
	m.runMu.Lock()
	if m.done == done {
		m.cancel = nil
		m.done = nil
		m.started = false
	}
	close(done)
	m.runMu.Unlock()
}

func (m *runtimeEventManager) runSource(ctx context.Context, source loadedRuntimeEventSource) {
	m.recordSourceStreamStart(source.name)
	err := source.source.StreamRuntimeEvents(ctx, func(candidate RuntimeEventCandidate) error {
		return m.appendCandidateFromSource(source.name, source.streamContext, candidate)
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		m.recordSourceStreamError(source.name, err)
		m.logger.Warn(
			"container runtime event stream stopped with error",
			zap.String("source", source.name),
			zap.Error(err),
		)
	}
}

func (m *runtimeEventManager) Append(candidate RuntimeEventCandidate) error {
	if m == nil {
		return errors.New("container runtime event manager is unavailable")
	}
	result, err := m.appendCandidate(m.defaultStreamContext, candidate)
	if err != nil {
		return err
	}
	if result.accepted {
		m.publishRecord(result.record, result.streamContext)
	}
	return err
}

func (m *runtimeEventManager) Diagnostics() runtimeEventManagerDiagnostics {
	if m == nil {
		return runtimeEventManagerDiagnostics{sources: map[string]runtimeEventSourceDiagnostics{}}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	sources := make(map[string]runtimeEventSourceDiagnostics, len(m.sourceDiagnostics))
	for name, diagnostics := range m.sourceDiagnostics {
		sources[name] = diagnostics
	}
	return runtimeEventManagerDiagnostics{sources: sources}
}

func (m *runtimeEventManager) appendCandidateFromSource(
	sourceName string,
	streamContext RuntimeEventStreamContext,
	candidate RuntimeEventCandidate,
) error {
	result, err := m.appendCandidate(streamContext, candidate)
	if err != nil {
		m.recordInvalidCandidateDrop(sourceName)
		m.LogCandidateDrop(candidate, err)
		return nil
	}
	if !result.accepted {
		m.recordDuplicateCandidateDrop(sourceName)
		return nil
	}
	m.publishRecord(result.record, result.streamContext)
	m.recordAcceptedCandidate(sourceName, result.record.Event.OccurredAt)
	return nil
}

func (m *runtimeEventManager) appendCandidate(
	streamContext RuntimeEventStreamContext,
	candidate RuntimeEventCandidate,
) (runtimeEventAppendResult, error) {
	event, err := newRuntimeEvent(candidate)
	if err != nil {
		return runtimeEventAppendResult{}, err
	}
	record, effectiveContext, accepted := m.appendRecord(streamContext, event)
	return runtimeEventAppendResult{
		record:        record,
		streamContext: effectiveContext,
		accepted:      accepted,
	}, nil
}

func (m *runtimeEventManager) appendRecord(
	streamContext RuntimeEventStreamContext,
	event RuntimeEvent,
) (RuntimeEventRecord, RuntimeEventStreamContext, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	m.evictExpiredLocked(now)
	resourceID := strings.TrimSpace(event.ResourceID)
	for _, existing := range m.historyByID[resourceID] {
		if existing.Event.ID == event.ID {
			return existing, m.contextForResourceLocked(resourceID), false
		}
	}

	effectiveContext := m.effectiveStreamContext(streamContext)
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
	m.contextByID[resourceID] = effectiveContext
	m.lastSeenAt[resourceID] = now

	return record, effectiveContext, true
}

func (m *runtimeEventManager) History(resourceID string) RuntimeEventsHistory {
	resourceID = strings.TrimSpace(resourceID)
	m.mu.Lock()
	defer m.mu.Unlock()

	m.evictExpiredLocked(time.Now().UTC())
	items := append([]RuntimeEventRecord(nil), m.historyByID[resourceID]...)
	return RuntimeEventsHistory{
		ResourceID: resourceID,
		Context:    m.contextForResourceLocked(resourceID),
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
		delete(m.contextByID, resourceID)
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

func (m *runtimeEventManager) contextForResourceLocked(resourceID string) RuntimeEventStreamContext {
	if context, ok := m.contextByID[resourceID]; ok {
		return context
	}
	return m.defaultStreamContext
}

func (m *runtimeEventManager) publishRecord(record RuntimeEventRecord, streamContext RuntimeEventStreamContext) {
	if m == nil || m.hub == nil {
		return
	}
	topic := containercontract.ContainerEventsTopicPrefix + record.Event.ResourceID
	m.hub.Publish(topic, struct {
		ResourceID string                    `json:"resource_id"`
		Context    RuntimeEventStreamContext `json:"context"`
		Record     RuntimeEventRecord        `json:"record"`
	}{
		ResourceID: record.Event.ResourceID,
		Context:    streamContext,
		Record:     record,
	})
}

func (m *runtimeEventManager) effectiveStreamContext(streamContext RuntimeEventStreamContext) RuntimeEventStreamContext {
	streamContext = normalizeRuntimeEventStreamContext(streamContext)
	if strings.TrimSpace(streamContext.Runtime) != "" {
		return streamContext
	}
	return m.defaultStreamContext
}

// normalizeRuntimeEventStreamContext 去除运行时流上下文中 Runtime 字段的首尾空白。
func normalizeRuntimeEventStreamContext(streamContext RuntimeEventStreamContext) RuntimeEventStreamContext {
	streamContext.Runtime = strings.TrimSpace(streamContext.Runtime)
	return streamContext
}

// normalizeRuntimeEventSourceName 清理运行时事件源名称并提供默认值。
// 它会去除首尾空白；若结果为空，则返回 "runtime"。
func normalizeRuntimeEventSourceName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "runtime"
	}
	return name
}

func (m *runtimeEventManager) recordSourceStreamStart(sourceName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	diagnostics := m.sourceDiagnostics[sourceName]
	diagnostics.streamStarts++
	m.sourceDiagnostics[sourceName] = diagnostics
}

func (m *runtimeEventManager) recordSourceStreamError(sourceName string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	diagnostics := m.sourceDiagnostics[sourceName]
	diagnostics.streamErrors++
	if err != nil {
		diagnostics.lastError = strings.TrimSpace(err.Error())
	}
	m.sourceDiagnostics[sourceName] = diagnostics
}

func (m *runtimeEventManager) recordInvalidCandidateDrop(sourceName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	diagnostics := m.sourceDiagnostics[sourceName]
	diagnostics.invalidDrops++
	m.sourceDiagnostics[sourceName] = diagnostics
}

func (m *runtimeEventManager) recordDuplicateCandidateDrop(sourceName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	diagnostics := m.sourceDiagnostics[sourceName]
	diagnostics.duplicateDrops++
	m.sourceDiagnostics[sourceName] = diagnostics
}

func (m *runtimeEventManager) recordAcceptedCandidate(sourceName string, occurredAt time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	diagnostics := m.sourceDiagnostics[sourceName]
	diagnostics.lastEventAt = occurredAt.UTC()
	m.sourceDiagnostics[sourceName] = diagnostics
}
