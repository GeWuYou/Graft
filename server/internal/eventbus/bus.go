// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package eventbus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Event 描述一次进程内事件发布的稳定载荷外壳。
//
// 事件名用于订阅匹配；Source 仅用于日志和诊断；Payload 保持开放，
// 以便当前阶段由发布方在模块边界内定义具体 DTO。
type Event struct {
	Name       string
	Source     string
	Payload    any
	OccurredAt time.Time
}

// Handler 定义单个事件处理器的稳定签名。
//
// 处理器应尽量快速返回；当前 MVP bus 采用顺序同步派发，因此长耗时逻辑应
// 由订阅方在自己的边界内显式转交后台任务，而不是阻塞整个发布链路。
type Handler func(ctx context.Context, event Event) error

// Bus 定义模块可依赖的最小事件总线能力。
//
// 这个接口只提供订阅和发布语义，不隐藏调度策略，也不引入取消订阅、
// 重试队列或消息持久化等当前阶段尚未需要的行为。
type Bus interface {
	Subscribe(eventName string, handler Handler) error
	Publish(ctx context.Context, event Event) error
}

// MemoryBus 是当前 MVP 阶段使用的最小进程内事件总线实现。
//
// 它按订阅顺序同步调用处理器，并在单个处理器失败或 panic 时继续执行
// 其余处理器，最后把全部错误聚合返回给发布方。
type MemoryBus struct {
	logger   *zap.Logger
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// New 创建一个新的进程内事件总线。
//
// 当 logger 为空时，函数会回退到 nop logger，避免测试或极简装配路径
// 因缺失日志实例而触发额外分支判断。
func New(logger *zap.Logger) *MemoryBus {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &MemoryBus{
		logger:   logger,
		handlers: make(map[string][]Handler),
	}
}

// Subscribe 为指定事件名追加一个处理器。
//
// 订阅顺序会影响发布时的执行顺序，因此调用方应把它视为显式生命周期
// 装配的一部分，而不是依赖隐式扫描或全局注册副作用。
func (b *MemoryBus) Subscribe(eventName string, handler Handler) error {
	name := strings.TrimSpace(eventName)
	if name == "" {
		return errors.New("event name is required")
	}
	if handler == nil {
		return errors.New("handler is required")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[name] = append(b.handlers[name], handler)
	return nil
}

// Publish 按订阅顺序同步派发一次事件。
//
// 当前实现会继续执行全部已注册处理器，并把返回错误或 panic 恢复后的
// 错误统一聚合返回；这样发布方既能得到失败信号，也不会因为首个处理器
// 出错而跳过其余订阅者。
func (b *MemoryBus) Publish(ctx context.Context, event Event) error {
	name := strings.TrimSpace(event.Name)
	if name == "" {
		return errors.New("event name is required")
	}

	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers[name]...)
	b.mu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}

	var publishErr error
	for _, handler := range handlers {
		if err := b.invokeHandler(ctx, handler, event); err != nil {
			publishErr = errors.Join(publishErr, err)
		}
	}

	return publishErr
}

func (b *MemoryBus) invokeHandler(ctx context.Context, handler Handler, event Event) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("event handler panic for %s: %v", event.Name, recovered)
			b.logger.Error(
				"event handler panicked",
				zap.String("event", event.Name),
				zap.String("source", event.Source),
				zap.Any("panic", recovered),
			)
		}
	}()

	if handlerErr := handler(ctx, event); handlerErr != nil {
		b.logger.Error(
			"event handler failed",
			zap.String("event", event.Name),
			zap.String("source", event.Source),
			zap.Error(handlerErr),
		)
		return fmt.Errorf("handle event %s: %w", event.Name, handlerErr)
	}

	return nil
}
