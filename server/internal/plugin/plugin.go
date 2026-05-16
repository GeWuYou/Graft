// Package plugin 定义运行时插件契约与生命周期管理能力。
package plugin

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/store"
)

// Plugin 定义所有后端插件都必须实现的稳定生命周期契约。
//
// 调用方可以依赖 Register -> Boot -> Shutdown 的整体顺序；当 Register
// 或 Boot 失败时，运行时会中止后续阶段，并按已成功启动的范围执行清理。
type Plugin interface {
	// Name 返回插件的稳定标识，用于依赖声明和运行时元数据。
	Name() string
	// Version 返回当前插件版本。
	//
	// 版本值主要用于运行时观测和诊断，不参与依赖排序。
	Version() string
	// DependsOn 返回当前插件依赖的插件名称列表。
	//
	// 依赖项必须引用已经注册的插件 Name；缺失依赖会导致排序失败。
	DependsOn() []string
	// Register 负责声明路由、权限、菜单、任务和公开服务。
	//
	// Register 不应启动长期后台行为；失败会阻止后续插件继续注册或启动。
	Register(ctx *Context) error
	// Boot 在所有插件完成注册后启动运行时行为。
	//
	// Boot 可以依赖所有已注册插件暴露的稳定能力；失败时调用方会关闭
	// 之前已经成功启动的插件。
	Boot(ctx *Context) error
	// Shutdown 在停止阶段释放插件资源，调用顺序与启动顺序相反。
	//
	// Shutdown 应尽最大努力释放资源并返回错误，而不是假设失败后可以跳过
	// 其余清理动作。
	Shutdown(ctx *Context) error
}

// Context 向插件暴露允许使用的显式运行时句柄。
//
// 这里聚合的是插件生命周期真正需要的核心能力，目的是让插件通过稳定
// 边界接入平台，而不是直接触碰 core 内部实现细节。
//
// Context 只承载运行时注入的公共能力，不应被插件长期持有并在生命周期
// 之外当作隐式全局变量使用。
type Context struct {
	// LifecycleContext 提供当前插件生命周期阶段可依赖的上下文。
	//
	// Register / Boot 阶段复用 Runtime 的 runCtx；Shutdown 阶段会切换为
	// 独立的有界关闭上下文，避免 runCtx 已取消后插件失去必要的优雅收敛窗口。
	LifecycleContext context.Context
	Config           *config.Config
	// Logger 提供插件生命周期内统一的结构化日志句柄，插件应复用它记录
	// 运行状态与诊断信息，而不是各自构造分散的日志实例。
	Logger *zap.Logger
	// I18n 提供平台级 locale 解析与消息查找能力，插件应通过它输出稳定的
	// 本地化错误响应，而不是维护各自独立的文案回退规则。
	I18n *i18n.Service
	// EventBus 提供插件间使用的最小进程内事件发布与订阅能力。
	//
	// 插件应只依赖显式 Subscribe / Publish 语义，不应假设存在消息持久化、
	// 重试队列或异步工作流编排等当前阶段并未提供的行为。
	EventBus           eventbus.Bus
	Redis              *redis.Client
	Router             gin.IRouter
	Services           *container.Container
	Stores             store.Factory
	MenuRegistry       *menu.Registry
	PermissionRegistry *permission.Registry
	CronRegistry       *cronx.Registry
}

// Manager 负责维护插件集合并按依赖关系排序。
//
// Manager 不拥有插件的业务状态；它只维护生命周期顺序与注册约束，是
// Runtime 和插件实现之间的调度边界。
type Manager struct {
	plugins []Plugin
}

// NewManager 创建一个空的插件管理器。
func NewManager() *Manager {
	return &Manager{plugins: make([]Plugin, 0)}
}

// RegisterPlugin 在运行时启动前向管理器注册一个插件。
//
// 当插件为 nil 或名称重复时返回错误，避免排序阶段出现不可恢复的歧义。
func (m *Manager) RegisterPlugin(p Plugin) error {
	if p == nil {
		return errors.New("plugin is required")
	}

	for _, existing := range m.plugins {
		if existing.Name() == p.Name() {
			return fmt.Errorf("plugin already registered: %s", p.Name())
		}
	}

	m.plugins = append(m.plugins, p)
	return nil
}

// Ordered 按声明的依赖关系返回插件启动顺序。
//
// 这里使用显式拓扑排序而不是隐式注册顺序，避免插件接入规模增加后因为
// 注册位置变化而打破稳定的启动语义。
//
// 排序失败时会返回缺失依赖或依赖环错误，调用方不应在错误场景下继续
// 执行插件生命周期。
func (m *Manager) Ordered() ([]Plugin, error) {
	total := len(m.plugins)
	if total == 0 {
		return nil, nil
	}

	index, inDegree := buildPluginIndex(m.plugins)
	edges, err := buildPluginEdges(m.plugins, index, inDegree)
	if err != nil {
		return nil, err
	}

	ordered := resolvePluginOrder(index, inDegree, edges, total)
	if len(ordered) != total {
		return nil, errors.New("plugin dependency cycle detected")
	}

	return ordered, nil
}

func buildPluginIndex(plugins []Plugin) (map[string]Plugin, map[string]int) {
	index := make(map[string]Plugin, len(plugins))
	inDegree := make(map[string]int, len(plugins))
	for _, p := range plugins {
		name := p.Name()
		index[name] = p
		inDegree[name] = 0
	}

	return index, inDegree
}

func buildPluginEdges(plugins []Plugin, index map[string]Plugin, inDegree map[string]int) (map[string][]string, error) {
	edges := make(map[string][]string, len(plugins))
	for _, p := range plugins {
		for _, dependency := range p.DependsOn() {
			if _, ok := index[dependency]; !ok {
				return nil, fmt.Errorf("plugin %s depends on missing plugin %s", p.Name(), dependency)
			}

			edges[dependency] = append(edges[dependency], p.Name())
			inDegree[p.Name()]++
		}
	}

	return edges, nil
}

func resolvePluginOrder(index map[string]Plugin, inDegree map[string]int, edges map[string][]string, total int) []Plugin {
	queue := make([]string, 0, total)
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	sort.Strings(queue)
	ordered := make([]Plugin, 0, total)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		ordered = append(ordered, index[name])

		for _, next := range edges[name] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
				sort.Strings(queue)
			}
		}
	}

	return ordered
}
