package projection

import (
	"fmt"
	"sort"
	"strings"
)

// Kind 标识跨边界契约值的稳定语义类别。
type Kind string

const (
	// KindAuthScheme 表示 HTTP Authorization 认证方案名称。
	KindAuthScheme Kind = "auth-scheme"
	// KindErrorCode 表示 API 响应错误码。
	KindErrorCode Kind = "error-code"
	// KindHTTPHeader 表示 HTTP 请求头名称。
	KindHTTPHeader Kind = "http-header"
	// KindMessageKey 表示本地化消息键。
	KindMessageKey Kind = "message-key"
)

// Lifecycle 描述契约值在兼容演进中的当前状态。
type Lifecycle string

const (
	// LifecycleActive 表示可供新调用方使用的稳定契约值。
	LifecycleActive Lifecycle = "active"
	// LifecycleDeprecated 表示仍为兼容保留、但不应新增引用的契约值。
	LifecycleDeprecated Lifecycle = "deprecated"
)

// Visibility 限制契约值可以出现的派生消费面。
type Visibility string

const (
	// VisibilityWeb 允许将契约值投影到 web generated artifact。
	VisibilityWeb Visibility = "web"
	// VisibilityServer 限制契约值仅供 server 运行时使用。
	VisibilityServer Visibility = "server"
	// VisibilityInternal 限制契约值仅供内部实现使用。
	VisibilityInternal Visibility = "internal"
)

// Entry 是一个已存在 typed Go contract 的导出索引。
//
// Value 只能引用 canonical owner 中已经定义的 typed constant；本结构不承载值字面量。
type Entry struct {
	ID          string
	Name        string
	Kind        Kind
	Owner       string
	Lifecycle   Lifecycle
	Visibility  Visibility
	Replacement string
	Value       fmt.Stringer
}

// Validate 验证 descriptor metadata 的完整性、生命周期约束及同类值的唯一性。
func Validate(entries []Entry) error {
	seenIDs := make(map[string]struct{}, len(entries))
	seenValues := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if err := validateEntry(entry); err != nil {
			return err
		}
		if _, exists := seenIDs[entry.ID]; exists {
			return fmt.Errorf("duplicate contract projection descriptor id %q", entry.ID)
		}
		seenIDs[entry.ID] = struct{}{}

		semanticValue := string(entry.Kind) + "\x00" + entry.Value.String()
		if _, exists := seenValues[semanticValue]; exists {
			return fmt.Errorf("duplicate contract projection value %q for kind %q", entry.Value.String(), entry.Kind)
		}
		seenValues[semanticValue] = struct{}{}
	}
	return nil
}

func validateEntry(entry Entry) error {
	if err := validateMetadata(entry); err != nil {
		return err
	}
	if err := validateLifecycle(entry); err != nil {
		return err
	}
	if err := validateValue(entry); err != nil {
		return err
	}
	return nil
}

func validateMetadata(entry Entry) error {
	if !hasRequiredMetadata(entry) {
		return fmt.Errorf("contract projection descriptor requires id, name, and owner")
	}
	if !validKind(entry.Kind) || !validLifecycle(entry.Lifecycle) || !validVisibility(entry.Visibility) {
		return fmt.Errorf("contract projection descriptor %q has invalid kind, lifecycle, or visibility", entry.ID)
	}
	return nil
}

func hasRequiredMetadata(entry Entry) bool {
	return strings.TrimSpace(entry.ID) != "" && strings.TrimSpace(entry.Name) != "" && strings.TrimSpace(entry.Owner) != ""
}

func validateLifecycle(entry Entry) error {
	if entry.Lifecycle == LifecycleDeprecated && strings.TrimSpace(entry.Replacement) == "" {
		return fmt.Errorf("deprecated contract projection descriptor %q requires replacement", entry.ID)
	}
	return nil
}

func validateValue(entry Entry) error {
	if entry.Value == nil || strings.TrimSpace(entry.Value.String()) == "" {
		return fmt.Errorf("contract projection descriptor %q requires a typed value reference", entry.ID)
	}
	return nil
}

func validKind(kind Kind) bool {
	return kind == KindAuthScheme || kind == KindErrorCode || kind == KindHTTPHeader || kind == KindMessageKey
}

func validLifecycle(lifecycle Lifecycle) bool {
	return lifecycle == LifecycleActive || lifecycle == LifecycleDeprecated
}

func validVisibility(visibility Visibility) bool {
	return visibility == VisibilityWeb || visibility == VisibilityServer || visibility == VisibilityInternal
}

func webEntries(entries []Entry) []Entry {
	filtered := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.Visibility == VisibilityWeb {
			filtered = append(filtered, entry)
		}
	}
	sort.Slice(filtered, func(left int, right int) bool {
		if filtered[left].Kind == filtered[right].Kind {
			return filtered[left].Name < filtered[right].Name
		}
		return filtered[left].Kind < filtered[right].Kind
	})
	return filtered
}
