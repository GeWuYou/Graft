package container

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/realtime"
	containercontract "graft/server/modules/container/contract"
)

const (
	defaultContainerStatsCollectInterval = 2 * time.Second
	containerStatsPublishTimeout         = 2 * time.Second
)

type statsCollector struct {
	collect  func(context.Context) ([]StatsSnapshot, error)
	hub      realtime.Publisher
	logger   *zap.Logger
	source   string
	interval time.Duration

	runMu   sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	started bool
}

type containerStatsPublished struct {
	Topic       string          `json:"topic"`
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	ShortID     string          `json:"short_id"`
	Runtime     string          `json:"runtime"`
	Resource    ResourceSummary `json:"resource"`
	CollectedAt time.Time       `json:"collected_at"`
}

// 如果 logger 为空，会使用无操作日志器。
func newStatsCollector(
	collect func(context.Context) ([]StatsSnapshot, error),
	hub realtime.Publisher,
	logger *zap.Logger,
	source string,
) *statsCollector {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &statsCollector{
		collect:  collect,
		hub:      hub,
		logger:   logger,
		source:   firstNonEmpty(source, "container"),
		interval: defaultContainerStatsCollectInterval,
	}
}

func (c *statsCollector) Start(ctx context.Context) error {
	if c == nil || c.collect == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	if c.started {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	c.cancel = cancel
	c.done = done
	c.started = true
	go c.run(runCtx, done)
	return nil
}

func (c *statsCollector) Stop(ctx context.Context) error {
	if c == nil {
		return nil
	}
	c.runMu.Lock()
	if !c.started {
		c.runMu.Unlock()
		return nil
	}
	cancel := c.cancel
	done := c.done
	c.cancel = nil
	c.done = nil
	c.started = false
	c.runMu.Unlock()

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
		return fmt.Errorf("stop container stats collector: %w", ctx.Err())
	}
}

func (c *statsCollector) run(ctx context.Context, done chan struct{}) {
	defer close(done)
	c.collectAndPublish(ctx)
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collectAndPublish(ctx)
		}
	}
}

func (c *statsCollector) collectAndPublish(ctx context.Context) {
	snapshots, err := c.collect(ctx)
	if err != nil {
		c.logger.Warn("collect container stats snapshots failed", zap.Error(err))
		return
	}
	for _, snapshot := range snapshots {
		if err := c.publish(ctx, snapshot); err != nil {
			c.logger.Warn("publish container stats snapshot failed",
				zap.String("containerID", strings.TrimSpace(snapshot.ContainerID)),
				zap.Error(err),
			)
		}
	}
}

func (c *statsCollector) publish(_ context.Context, snapshot StatsSnapshot) error {
	if c.hub == nil {
		return nil
	}
	containerID := strings.TrimSpace(snapshot.ContainerID)
	if containerID == "" {
		return nil
	}
	topic := containercontract.ContainerStatsTopicPrefix + containerID
	c.hub.Publish(topic, containerStatsPublished{
		Topic:       topic,
		ID:          containerID,
		Name:        strings.TrimSpace(snapshot.Name),
		ShortID:     strings.TrimSpace(snapshot.ShortID),
		Runtime:     strings.TrimSpace(snapshot.Runtime),
		Resource:    snapshot.Resource,
		CollectedAt: snapshot.CollectedAt,
	})
	return nil
}
