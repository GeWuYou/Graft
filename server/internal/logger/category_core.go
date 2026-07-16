package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TraceLevel 位于 zap DEBUG 之下，用于默认关闭的高频过程诊断。
const TraceLevel zapcore.Level = zapcore.DebugLevel - 1

func wrapCategoryCore(rules CategoryRules) func(zapcore.Core) zapcore.Core {
	return func(core zapcore.Core) zapcore.Core {
		if core == nil {
			return zapcore.NewNopCore()
		}
		return categoryCore{Core: core, rules: rules}
	}
}

type categoryCore struct {
	zapcore.Core
	rules    CategoryRules
	category LogCategory
}

func (c categoryCore) With(fields []zap.Field) zapcore.Core {
	category := c.category
	for _, field := range fields {
		if field.Key == categoryFieldKey && field.Type == zapcore.StringType {
			category = LogCategory(field.String)
		}
	}
	return categoryCore{Core: c.Core.With(fields), rules: c.rules, category: category}
}

func (c categoryCore) Enabled(level zapcore.Level) bool {
	if !c.CategoryGateEnabled(string(c.category), level) {
		return false
	}
	return c.Core.Enabled(level)
}

// CategoryGateEnabled 仅报告类别规则，不叠加 sink 级别阈值，供 AppLogger 保持无输出 sink 时的持久化语义。
func (c categoryCore) CategoryGateEnabled(category string, level zapcore.Level) bool {
	return category == "" || c.rules.allowed(LogCategory(category), level)
}

func (c categoryCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(entry.Level) {
		return checked
	}
	return checked.AddCore(entry, c)
}
