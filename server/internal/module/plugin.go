// Package module 定义运行时模块契约与生命周期管理能力。
package module

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

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
)

// Module 定义所有后端模块都必须实现的稳定生命周期契约。
//
// 调用方可以依赖 Register -> Boot -> Shutdown 的整体顺序；当 Register
// 或 Boot 失败时，运行时会中止后续阶段，并按已成功启动的范围执行清理。
type Module interface {
	// Register 负责声明路由、权限、菜单、任务和公开服务。
	//
	// Register 不应启动长期后台行为；失败会阻止后续模块继续注册或启动。
	Register(ctx *Context) error
	// Boot 在所有模块完成注册后启动运行时行为。
	//
	// Boot 可以依赖所有已注册模块暴露的稳定能力；失败时调用方会关闭
	// 之前已经成功启动的模块。
	Boot(ctx *Context) error
	// Shutdown 在停止阶段释放模块资源，调用顺序与启动顺序相反。
	//
	// Shutdown 应尽最大努力释放资源并返回错误，而不是假设失败后可以跳过
	// 其余清理动作。
	Shutdown(ctx *Context) error
}

// RuntimeModule 暴露 compile-time 模块元数据与运行时生命周期的组合视图。
//
// core runtime 只通过这个包装后的稳定表面感知模块身份和依赖，避免要求
// 业务模块实例再维护第二份会漂移的 Name / DependsOn authority。
type RuntimeModule interface {
	Module
	Name() string
	DependsOn() []string
}

// Builder 定义 compile-time 模块描述符到运行时模块实例的显式构造边界。
//
// Builder 当前只负责构造模块实例；后续 capability 或模块私有依赖装配
// 可以继续沿这条边界扩展，而不把共享接线重新塞回中心化 CLI 文件。
type Builder interface {
	Build(BuildContext) (Module, error)
}

// BuildContext 暴露模块构造阶段允许消费的最小 core 资源。
//
// 它只服务于 compile-time builder wiring，不进入模块运行时热路径。
// 这里保留显式服务解析边界，避免 builder 重新拿回泛化的业务仓储工厂入口。
type BuildContext struct {
	Services *container.Container
}

// BuilderFunc 允许用普通函数实现 Builder。
type BuilderFunc func(BuildContext) (Module, error)

// Build 执行函数式 Builder。
func (f BuilderFunc) Build(ctx BuildContext) (Module, error) {
	if f == nil {
		return nil, errors.New("module builder is required")
	}

	return f(ctx)
}

// ResolveService 解析一个 builder 或模块生命周期允许消费的显式单例服务。
func ResolveService[T any](resolver container.Resolver, key any) (T, error) {
	var zero T
	if resolver == nil {
		return zero, errors.New("service resolver is required")
	}

	resolvedAny, err := resolver.Resolve(key)
	if err != nil {
		return zero, err
	}

	resolved, ok := resolvedAny.(T)
	if !ok {
		return zero, fmt.Errorf("resolved service %T has unexpected type %T", key, resolvedAny)
	}

	return resolved, nil
}

// Spec 定义历史命名下的 compile-time 模块元数据与运行时构造入口。
//
// Spec 是生成式 module registry 的稳定输入。它收敛模块名、版本、
// 依赖与迁移目录等
// 最小元数据，并把真正的运行时实例化动作交给 Builder。
type Spec struct {
	ID            string
	Dependencies  []string
	MigrationPath []string
	Builder       Builder
}

// Name 返回模块定义的稳定模块标识。
func (d Spec) Name() string {
	return strings.TrimSpace(d.ID)
}

// DependsOn 返回模块定义声明的模块依赖列表。
func (d Spec) DependsOn() []string {
	return trimStringsPreserveDuplicates(d.Dependencies)
}

// MigrationDirs 返回模块定义声明的模块自有迁移目录。
func (d Spec) MigrationDirs() []string {
	return trimNonEmptyStrings(d.MigrationPath)
}

// Validate 校验模块定义的最小 compile-time 元数据完整性。
func (d Spec) Validate() error {
	if d.Name() == "" {
		return errors.New("module spec name is required")
	}
	if d.Builder == nil {
		return fmt.Errorf("module spec %s builder is required", d.Name())
	}
	if _, err := normalizeDependencies(d.Name(), d.DependsOn()); err != nil {
		return err
	}

	return nil
}

// Build 根据模块定义构造一个运行时模块实例，并校验运行时元数据没有偏离
// compile-time 模块定义的 canonical truth。
func (d Spec) Build(ctx BuildContext) (RuntimeModule, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}

	built, err := d.Builder.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("build module %s: %w", d.Name(), err)
	}
	if built == nil {
		return nil, fmt.Errorf("build module %s: builder returned nil module", d.Name())
	}

	return NewModule(d, built), nil
}

// NewModule 使用 compile-time 模块定义包装一个运行时模块实例。
func NewModule(spec Spec, instance Module) RuntimeModule {
	return describedModule{moduleSpec: spec, delegate: instance}
}

// Context 向模块暴露允许使用的显式运行时句柄。
//
// 这里聚合的是模块生命周期真正需要的核心能力，目的是让模块通过稳定
// 边界接入平台，而不是直接触碰 core 内部实现细节。
//
// Context 只承载运行时注入的公共能力，不应被模块长期持有并在生命周期
// 之外当作隐式全局变量使用。
type Context struct {
	// LifecycleContext 提供当前模块生命周期阶段可依赖的上下文。
	//
	// Register / Boot 阶段复用 Runtime 的 runCtx；Shutdown 阶段会切换为
	// 独立的有界关闭上下文，避免 runCtx 已取消后模块失去必要的优雅收敛窗口。
	LifecycleContext context.Context
	Config           *config.Config
	// Logger 提供模块生命周期内统一的结构化日志句柄，模块应复用它记录
	// 运行状态与诊断信息，而不是各自构造分散的日志实例。
	Logger *zap.Logger
	// I18n 提供平台级 locale 解析与消息查找能力，模块应通过它输出稳定的
	// 本地化错误响应，而不是维护各自独立的文案回退规则。
	I18n *i18n.Service
	// EventBus 提供模块间使用的最小进程内事件发布与订阅能力。
	//
	// 模块应只依赖显式 Subscribe / Publish 语义，不应假设存在消息持久化、
	// 重试队列或异步工作流编排等当前阶段并未提供的行为。
	EventBus           eventbus.Bus
	Redis              *redis.Client
	Router             gin.IRouter
	Services           *container.Container
	RuntimeMetadata    RuntimeMetadata
	MenuRegistry       *menu.Registry
	PermissionRegistry *permission.Registry
	CronRegistry       *cronx.Registry
}

// Manager 负责维护模块集合并按依赖关系排序。
//
// Manager 不拥有模块的业务状态；它只维护生命周期顺序与注册约束，是
// Runtime 和模块实现之间的调度边界。
type Manager struct {
	modules []RuntimeModule
}

// NewManager 创建一个空的模块管理器。
func NewManager() *Manager {
	return &Manager{modules: make([]RuntimeModule, 0)}
}

// RegisterModule 在运行时启动前向管理器注册一个模块。
//
// 当模块为 nil 或名称重复时返回错误，避免排序阶段出现不可恢复的歧义。
func (m *Manager) RegisterModule(current RuntimeModule) error {
	if current == nil {
		return errors.New("module is required")
	}

	for _, existing := range m.modules {
		if existing.Name() == current.Name() {
			return fmt.Errorf("module already registered: %s", current.Name())
		}
	}

	m.modules = append(m.modules, current)
	return nil
}

// Ordered 按声明的依赖关系返回模块启动顺序。
//
// 这里使用显式拓扑排序而不是隐式注册顺序，避免模块接入规模增加后因为
// 注册位置变化而打破稳定的启动语义。
//
// 排序失败时会返回缺失依赖或依赖环错误，调用方不应在错误场景下继续
// 执行模块生命周期。
func (m *Manager) Ordered() ([]RuntimeModule, error) {
	return orderByDependencies(m.modules)
}

// OrderSpecs 按依赖关系返回稳定的模块定义顺序。
//
// 它复用与运行时模块相同的拓扑排序规则，使 compile-time registry 和
// runtime lifecycle 使用同一套依赖真相，而不是各自维护第二份排序逻辑。
func OrderSpecs(descriptors []Spec) ([]Spec, error) {
	return orderByDependencies(descriptors)
}

type describedModule struct {
	moduleSpec Spec
	delegate   Module
}

func (p describedModule) Name() string {
	return p.moduleSpec.Name()
}

func (p describedModule) DependsOn() []string {
	return p.moduleSpec.DependsOn()
}

func (p describedModule) Register(ctx *Context) error {
	return p.delegate.Register(ctx)
}

func (p describedModule) Boot(ctx *Context) error {
	return p.delegate.Boot(ctx)
}

func (p describedModule) Shutdown(ctx *Context) error {
	return p.delegate.Shutdown(ctx)
}

type dependencyTarget interface {
	Name() string
	DependsOn() []string
}

func orderByDependencies[T dependencyTarget](items []T) ([]T, error) {
	total := len(items)
	if total == 0 {
		return nil, nil
	}

	index, inDegree, err := buildDependencyIndex(items)
	if err != nil {
		return nil, err
	}
	edges, err := buildDependencyEdges(items, index, inDegree)
	if err != nil {
		return nil, err
	}

	return resolveDependencyOrder(index, inDegree, edges, total)
}

func buildDependencyIndex[T dependencyTarget](items []T) (map[string]T, map[string]int, error) {
	index := make(map[string]T, len(items))
	inDegree := make(map[string]int, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name())
		if name == "" {
			return nil, nil, errors.New("module name is required")
		}
		if _, exists := index[name]; exists {
			return nil, nil, fmt.Errorf("module already registered: %s", name)
		}

		index[name] = item
		inDegree[name] = 0
	}

	return index, inDegree, nil
}

func buildDependencyEdges[T dependencyTarget](items []T, index map[string]T, inDegree map[string]int) (map[string][]string, error) {
	edges := make(map[string][]string, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name())
		dependencies, err := normalizeDependencies(name, item.DependsOn())
		if err != nil {
			return nil, err
		}

		for _, dependency := range dependencies {
			if _, ok := index[dependency]; !ok {
				return nil, fmt.Errorf("module %s depends on missing module %s", name, dependency)
			}

			edges[dependency] = append(edges[dependency], name)
			inDegree[name]++
		}
	}

	return edges, nil
}

func resolveDependencyOrder[T dependencyTarget](index map[string]T, inDegree map[string]int, edges map[string][]string, total int) ([]T, error) {
	queue := make([]string, 0, total)
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	sort.Strings(queue)
	ordered := make([]T, 0, total)
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

	if len(ordered) != total {
		return nil, errors.New("module dependency cycle detected")
	}

	return ordered, nil
}

func normalizeDependencies(moduleName string, dependencies []string) ([]string, error) {
	normalized := trimStringsPreserveDuplicates(dependencies)
	seen := make(map[string]struct{}, len(normalized))
	for _, dependency := range normalized {
		if dependency == "" {
			return nil, fmt.Errorf("module %s has an empty dependency name", moduleName)
		}
		if dependency == moduleName {
			return nil, fmt.Errorf("module %s cannot depend on itself", moduleName)
		}
		if _, exists := seen[dependency]; exists {
			return nil, fmt.Errorf("module %s depends on duplicate module %s", moduleName, dependency)
		}
		seen[dependency] = struct{}{}
	}

	return normalized, nil
}

func trimStringsPreserveDuplicates(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		trimmed = append(trimmed, strings.TrimSpace(value))
	}

	return trimmed
}

func trimNonEmptyStrings(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		trimmed = append(trimmed, value)
	}

	return trimmed
}
