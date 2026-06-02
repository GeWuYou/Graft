package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"graft/server/internal/cronx"
)

// Runtime 暴露仓库内稳定的最小调度能力。
type Runtime interface {
	RegisterJob(job cronx.Job) error
	RemoveJob(name string) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// CronRuntime 是基于 robfig/cron/v3 的最小进程内调度器封装。
//
// 它把底层 cron 细节留在包内部，对外只保留显式 job 注册、启动、停止与
// 移除语义，避免业务插件直接依赖第三方调度器实现。
type CronRuntime struct {
	logger *zap.Logger

	mu      sync.RWMutex
	cron    *cron.Cron
	started bool
	entries map[string]cron.EntryID

	lifecycleCtx    context.Context
	lifecycleCancel context.CancelFunc
}

// New 创建一个新的最小调度器运行时。
func New(logger *zap.Logger) *CronRuntime {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &CronRuntime{
		logger:  logger,
		cron:    cron.New(cron.WithSeconds()),
		entries: make(map[string]cron.EntryID),
	}
}

// RegisterJob 注册一个显式调度任务。
func (r *CronRuntime) RegisterJob(job cronx.Job) error {
	if err := job.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entries[job.Name]; exists {
		return fmt.Errorf("job already registered: %s", job.Name)
	}

	entryID, err := r.cron.AddFunc(job.Schedule, func() {
		runCtx := r.jobContext()
		if runCtx == nil {
			r.logger.Error("scheduler job skipped because lifecycle context is unavailable",
				zap.String("job", job.Name),
				zap.String("plugin", job.Module),
			)
			return
		}

		if runErr := job.Run(runCtx); runErr != nil {
			r.logger.Error("scheduler job failed",
				zap.String("job", job.Name),
				zap.String("plugin", job.Module),
				zap.Error(runErr),
			)
		}
	})
	if err != nil {
		return fmt.Errorf("register job %s: %w", job.Name, err)
	}

	r.entries[job.Name] = entryID
	return nil
}

// RemoveJob 按稳定名称移除一个已注册任务。
func (r *CronRuntime) RemoveJob(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entryID, ok := r.entries[name]
	if !ok {
		return errors.New("job not found")
	}

	r.cron.Remove(entryID)
	delete(r.entries, name)
	return nil
}

// Start 绑定生命周期上下文并启动当前调度器。
func (r *CronRuntime) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("lifecycle context is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return nil
	}

	r.lifecycleCtx, r.lifecycleCancel = context.WithCancel(ctx)
	r.cron.Start()
	r.started = true
	return nil
}

// Stop 停止当前调度器并等待在途任务结束。
//
// 若传入的 ctx 为 nil，则无限期等待所有已启动任务完成；若 ctx 非 nil，
// 则在 ctx 取消时立即返回 ctx.Err()，但底层任务仍会继续执行到自然结束。
func (r *CronRuntime) Stop(ctx context.Context) error {
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return nil
	}

	stopCtx := r.cron.Stop()
	r.started = false
	lifecycleCancel := r.lifecycleCancel
	r.lifecycleCtx = nil
	r.lifecycleCancel = nil
	r.mu.Unlock()

	if lifecycleCancel != nil {
		lifecycleCancel()
	}

	if ctx == nil {
		<-stopCtx.Done()
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-stopCtx.Done():
		return nil
	}
}

func (r *CronRuntime) jobContext() context.Context {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.lifecycleCtx
}
