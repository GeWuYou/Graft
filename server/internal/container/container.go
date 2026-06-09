// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package container 提供核心与模块共用的显式单例注册能力。
package container

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// ErrServiceNotRegistered 表示请求解析的服务 key 尚未在容器中注册。
var ErrServiceNotRegistered = errors.New("service not registered")

// Provider 定义单例服务的构造函数。
//
// Provider 在第一次成功解析前最多执行一次；返回错误时不会写入缓存，
// 后续 Resolve 会继续尝试重新构建。
type Provider func(resolver Resolver) (any, error)

// Registry 定义显式单例注册能力。
//
// Registry 只负责登记稳定的服务 key 与构造函数映射，不负责隐式扫描或
// 自动装配调用链。
type Registry interface {
	// RegisterSingleton 使用给定 key 注册单例构造函数。
	//
	// 当 key 已被注册或 provider 为空时返回错误。
	RegisterSingleton(key any, provider Provider) error
}

// Resolver 定义显式单例解析能力。
//
// Resolver 调用方只能依赖“同一 key 返回同一共享实例”的稳定语义，不应
// 假设底层采用反射注入或其它隐藏构造机制。
type Resolver interface {
	// Resolve 根据给定 key 返回单例实例。
	//
	// 当服务未注册或构造失败时返回错误；构造错误不会污染缓存。
	Resolve(key any) (any, error)
}

// Container 是运行时使用的显式单例容器。
//
// Container 负责保存单例构造函数、缓存已构建实例，并在并发解析时
// 复用同一次构建过程，避免重复创建共享资源。Container 支持并发访问。
type Container struct {
	// mu 串行化 provider、实例缓存和构建中状态的读写。
	mu sync.RWMutex
	// providers 保存服务 key 到单例构造函数的映射。
	providers map[string]Provider
	// instances 缓存已经成功构建的共享单例。
	instances map[string]any
	// inflight 记录正在构建的单例，确保并发调用共享同一次构建结果。
	inflight map[string]*inflightCall
}

type inflightCall struct {
	done chan struct{}
	val  any
	err  error
}

// New 创建一个空的单例容器。
//
// 返回的 Container 可以被 core 和模块共享，并在并发解析同一服务时复用
// 同一次构造过程。
func New() *Container {
	return &Container{
		providers: make(map[string]Provider),
		instances: make(map[string]any),
		inflight:  make(map[string]*inflightCall),
	}
}

// RegisterSingleton 为一个服务 key 注册唯一的单例构造函数。
//
// key 会被转换为稳定的类型名；重复注册同一 key 会立即失败，避免运行时
// 因覆盖 provider 而出现不可见的装配分歧。
func (c *Container) RegisterSingleton(key any, provider Provider) error {
	if provider == nil {
		return errors.New("provider is required")
	}

	name := keyName(key)

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.providers[name]; exists {
		return fmt.Errorf("service already registered: %s", name)
	}

	c.providers[name] = provider
	return nil
}

// Resolve 解析指定 key 对应的单例实例。
//
// 如果实例尚未创建，Resolve 会触发一次构建并缓存结果；如果同一实例
// 正在被其它 goroutine 构建，当前调用会等待并复用那次构建结果。
//
// 构建失败时不会缓存半成品实例；调用方需要显式处理错误，而不是假设
// 容器会降级返回零值。
func (c *Container) Resolve(key any) (any, error) {
	name := keyName(key)

	c.mu.Lock()
	if instance, ok := c.instances[name]; ok {
		c.mu.Unlock()
		return instance, nil
	}

	// 第一个进入的调用负责实际构建，后续并发调用等待 inflight 结果，
	// 这样可以避免高成本 provider 被重复执行。
	if call, ok := c.inflight[name]; ok {
		c.mu.Unlock()
		<-call.done
		if call.err != nil {
			return nil, fmt.Errorf("build service %s: %w", name, call.err)
		}
		return call.val, nil
	}

	provider, ok := c.providers[name]
	if !ok {
		c.mu.Unlock()
		return nil, fmt.Errorf("%w: %s", ErrServiceNotRegistered, name)
	}

	call := &inflightCall{done: make(chan struct{})}
	c.inflight[name] = call
	c.mu.Unlock()

	instance, err := provider(c)

	c.mu.Lock()
	if err == nil {
		c.instances[name] = instance
	}
	call.val = instance
	call.err = err
	close(call.done)
	delete(c.inflight, name)
	c.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("build service %s: %w", name, err)
	}
	return instance, nil
}

func keyName(key any) string {
	if key == nil {
		// 这里保留显式占位名，避免 nil key 在错误消息里退化成空字符串，
		// 影响调用方定位注册或解析问题。
		return "<nil>"
	}

	return reflect.TypeOf(key).String()
}
