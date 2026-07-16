package logger

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogCategory 是 logger 注册表管理的运行时诊断类别标识。
type LogCategory string

const (
	// CategoryDockerStats 标识 Docker 运行时统计轮询诊断。
	CategoryDockerStats LogCategory = "docker.stats"
	// CategoryDockerEvents 标识 Docker 事件流诊断。
	CategoryDockerEvents LogCategory = "docker.events"
	// CategoryRuntimeCache 标识运行时缓存刷新诊断。
	CategoryRuntimeCache LogCategory = "runtime.cache"
	// CategoryRuntimeMetrics 标识运行时指标采样诊断。
	CategoryRuntimeMetrics LogCategory = "runtime.metrics"
	// CategoryRuntimeStats 标识运行时统计聚合诊断。
	CategoryRuntimeStats LogCategory = "runtime.stats"
	// CategoryComposeRuntime 标识 Compose 运行时探测诊断。
	CategoryComposeRuntime LogCategory = "compose.runtime"
	// CategorySchedulerPoll 标识调度器轮询诊断。
	CategorySchedulerPoll LogCategory = "scheduler.poll"
	// CategoryDatabaseEnt 标识 Ent 数据库运行时诊断。
	CategoryDatabaseEnt LogCategory = "database.ent"
)

const (
	categoryFieldKey      = "category"
	categoryRulePartCount = 2
)

var registeredCategories = map[LogCategory]struct{}{
	CategoryDockerStats: {}, CategoryDockerEvents: {}, CategoryRuntimeCache: {}, CategoryRuntimeMetrics: {},
	CategoryRuntimeStats: {}, CategoryComposeRuntime: {}, CategorySchedulerPoll: {}, CategoryDatabaseEnt: {},
}

// RegisteredCategories 返回注册表中的叶子类别，供测试和受控工具读取。
func RegisteredCategories() []LogCategory {
	categories := make([]LogCategory, 0, len(registeredCategories))
	for category := range registeredCategories {
		categories = append(categories, category)
	}
	sort.Slice(categories, func(i, j int) bool { return categories[i] < categories[j] })
	return categories
}

func isRegisteredCategory(category LogCategory) bool {
	_, ok := registeredCategories[category]
	return ok
}

// CategoryRules 保存启动时解析完成的类别开关；空规则保持正常级别日志，且仅抑制类别 TRACE。
type CategoryRules map[LogCategory]bool

// ParseCategoryRules 严格解析逗号分隔的 category=bool 规则。
func ParseCategoryRules(raw string) (CategoryRules, error) {
	rules := make(CategoryRules)
	if strings.TrimSpace(raw) == "" {
		return rules, nil
	}

	for _, entry := range strings.Split(raw, ",") {
		parts := strings.Split(entry, "=")
		if len(parts) != categoryRulePartCount {
			return nil, fmt.Errorf("invalid GRAFT_LOG_CATEGORIES entry %q: expected category=bool", entry)
		}
		category := LogCategory(strings.TrimSpace(parts[0]))
		if !isRegistryPrefix(category) {
			return nil, fmt.Errorf("unknown GRAFT_LOG_CATEGORIES category %q", category)
		}
		if _, exists := rules[category]; exists {
			return nil, fmt.Errorf("duplicate GRAFT_LOG_CATEGORIES category %q", category)
		}
		enabled, err := strconv.ParseBool(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid GRAFT_LOG_CATEGORIES boolean for %q: %w", category, err)
		}
		rules[category] = enabled
	}
	return rules, nil
}

func isRegistryPrefix(category LogCategory) bool {
	if category == "" {
		return false
	}
	prefix := string(category)
	for registered := range registeredCategories {
		if string(registered) == prefix || strings.HasPrefix(string(registered), prefix+".") {
			return true
		}
	}
	return false
}

func (rules CategoryRules) allowed(category LogCategory, level zapcore.Level) bool {
	matched, configured := rules.longestMatch(category)
	_ = matched
	if configured && !rules[matched] {
		return false
	}
	if level == TraceLevel {
		return configured && rules[matched]
	}
	return true
}

func (rules CategoryRules) longestMatch(category LogCategory) (LogCategory, bool) {
	var best LogCategory
	for candidate := range rules {
		if category == candidate || strings.HasPrefix(string(category), string(candidate)+".") {
			if len(candidate) > len(best) {
				best = candidate
			}
		}
	}
	return best, best != ""
}

// CategoryLogger 是绑定注册类别的薄日志 facade，不替代现有 *zap.Logger DI。
type CategoryLogger struct {
	base     *zap.Logger
	category LogCategory
}

// Category 绑定一个已注册的类别；未知类别返回不会写出的 logger，避免绕过注册表。
func Category(base *zap.Logger, category LogCategory) CategoryLogger {
	if base == nil || !isRegisteredCategory(category) {
		return CategoryLogger{base: zap.NewNop(), category: category}
	}
	return CategoryLogger{base: base.With(zap.String(categoryFieldKey, string(category))), category: category}
}

// WithCategory 是 Category 的低成本别名。
func WithCategory(base *zap.Logger, category LogCategory) CategoryLogger {
	return Category(base, category)
}

// Enabled 报告指定类别和级别是否会写入任一 sink。
func (l CategoryLogger) Enabled(level zapcore.Level) bool {
	return l.base != nil && l.base.Core().Enabled(level)
}

// Trace 写出高频过程诊断；该级别仅在类别显式启用时可见。
func (l CategoryLogger) Trace(message string, fields ...zap.Field) {
	l.write(TraceLevel, message, fields...)
}

// TraceLazy 仅在 TRACE 已启用时构造字段；调用方对 Trace 之外的昂贵计算仍须先调用 Enabled。
func (l CategoryLogger) TraceLazy(message string, buildFields func() []zap.Field) {
	if !l.Enabled(TraceLevel) {
		return
	}
	l.Trace(message, buildFields()...)
}

// Debug 写出调试日志。
func (l CategoryLogger) Debug(message string, fields ...zap.Field) {
	l.write(zap.DebugLevel, message, fields...)
}

// Info 写出信息日志。
func (l CategoryLogger) Info(message string, fields ...zap.Field) {
	l.write(zap.InfoLevel, message, fields...)
}

// Warn 写出警告日志。
func (l CategoryLogger) Warn(message string, fields ...zap.Field) {
	l.write(zap.WarnLevel, message, fields...)
}

// Error 写出错误日志。
func (l CategoryLogger) Error(message string, fields ...zap.Field) {
	l.write(zap.ErrorLevel, message, fields...)
}

// With 返回携带附加结构化字段的同类别 facade。
func (l CategoryLogger) With(fields ...zap.Field) CategoryLogger {
	if l.base == nil {
		return l
	}
	l.base = l.base.With(fields...)
	return l
}

func (l CategoryLogger) write(level zapcore.Level, message string, fields ...zap.Field) {
	if l.base == nil {
		return
	}
	if checked := l.base.Check(level, message); checked != nil {
		checked.Write(fields...)
	}
}
