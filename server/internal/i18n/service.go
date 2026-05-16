package i18n

import (
	"net/http"
	"strings"

	"golang.org/x/text/language"

	"graft/server/internal/config"
)

// LocaleHeader 允许调用方显式指定当前请求期望的语言。
const LocaleHeader = "X-Graft-Locale"

// #nosec G101 -- 这里保存的是本地化 message key 与展示文案，不是凭据。
var defaultCatalogs = map[string]map[string]string{
	"zh-CN": {
		"auth.invalid_credentials":       "用户名或密码错误",
		"auth.token_missing":             "缺少访问令牌",
		"auth.token_expired":             "访问令牌已过期",
		"auth.token_invalid":             "访问令牌无效",
		"auth.forbidden":                 "权限不足",
		"auth.invalid_refresh_session":   "刷新会话无效或已失效",
		"auth.password_policy_violation": "新密码不符合安全要求",
		"auth.password_reuse_forbidden":  "新密码不能重复使用默认密码或当前密码",
		"auth.current_password_invalid":  "当前密码错误",
		"auth.missing_actor":             "缺少请求身份信息",
		"auth.missing_permission":        "缺少所需权限",
		"auth.session_not_found":         "会话不存在或已失效",
		"common.internal_error":          "服务内部错误",
		"common.invalid_argument":        "请求参数不合法",
		"user.not_found":                 "用户不存在",
	},
	"en-US": {
		"auth.invalid_credentials":       "Invalid username or password",
		"auth.token_missing":             "Missing access token",
		"auth.token_expired":             "Access token expired",
		"auth.token_invalid":             "Invalid access token",
		"auth.forbidden":                 "Forbidden",
		"auth.invalid_refresh_session":   "Invalid or expired refresh session",
		"auth.password_policy_violation": "New password does not meet security requirements",
		"auth.password_reuse_forbidden":  "New password must not reuse the default or current password",
		"auth.current_password_invalid":  "Current password is invalid",
		"auth.missing_actor":             "Missing request actor",
		"auth.missing_permission":        "Missing required permission",
		"auth.session_not_found":         "Session not found or already inactive",
		"common.internal_error":          "Internal server error",
		"common.invalid_argument":        "Invalid request parameters",
		"user.not_found":                 "User not found",
	},
}

// Service 提供平台级 locale 解析与消息查找能力。
//
// Service 不关心调用方来自 core 还是插件；它只对稳定 message key、默认
// 语言和回退语义负责。
type Service struct {
	defaultLocale  string
	fallbackLocale string
	supported      []language.Tag
	matcher        language.Matcher
	catalogs       map[string]map[string]string
}

// New 使用配置快照创建最小本地化服务。
func New(cfg config.I18nConfig) *Service {
	supported := make([]language.Tag, 0, len(cfg.SupportedLocales))
	for _, locale := range cfg.SupportedLocales {
		tag, err := language.Parse(locale)
		if err != nil {
			continue
		}
		supported = append(supported, tag)
	}
	if len(supported) == 0 {
		supported = []language.Tag{language.MustParse("zh-CN")}
	}

	return &Service{
		defaultLocale:  canonicalizeLocale(cfg.DefaultLocale, supported),
		fallbackLocale: canonicalizeLocale(cfg.FallbackLocale, supported),
		supported:      supported,
		matcher:        language.NewMatcher(supported),
		catalogs:       cloneCatalogs(defaultCatalogs),
	}
}

// DefaultLocale 返回当前服务使用的默认语言。
func (s *Service) DefaultLocale() string {
	return s.defaultLocale
}

// FallbackLocale 返回消息查找失败时的最终回退语言。
func (s *Service) FallbackLocale() string {
	return s.fallbackLocale
}

// ResolveLocale 根据请求显式语言、会话语言和默认配置返回最终语言。
//
// 解析优先级固定为：显式请求语言、会话语言、默认语言、回退语言。
func (s *Service) ResolveLocale(requestLocale string, sessionLocale string) string {
	for _, candidate := range []string{requestLocale, sessionLocale, s.defaultLocale, s.fallbackLocale} {
		if resolved := s.matchLocale(candidate); resolved != "" {
			return resolved
		}
	}

	return "zh-CN"
}

// ResolveRequestLocale 从 HTTP 请求中提取显式语言偏好并执行统一回退。
func (s *Service) ResolveRequestLocale(request *http.Request, sessionLocale string) string {
	if request == nil {
		return s.ResolveLocale("", sessionLocale)
	}

	requested := strings.TrimSpace(request.Header.Get(LocaleHeader))
	if requested == "" {
		requested = strings.TrimSpace(request.Header.Get("Accept-Language"))
	}

	return s.ResolveLocale(requested, sessionLocale)
}

// Message 返回给定语言和消息 key 对应的最终文案。
//
// 当指定语言缺失对应 key 时，会回退到 fallback/default 语言；如果所有
// 已知目录都缺失，则直接返回 key，避免响应中出现空字符串。
func (s *Service) Message(locale string, key string) string {
	if key == "" {
		return ""
	}

	resolvedLocale := s.ResolveLocale(locale, "")
	for _, candidate := range []string{resolvedLocale, s.fallbackLocale, s.defaultLocale} {
		if message := messageFromCatalog(s.catalogs, candidate, key); message != "" {
			return message
		}
	}

	return key
}

func (s *Service) matchLocale(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	if tags, _, err := language.ParseAcceptLanguage(input); err == nil && len(tags) > 0 {
		_, index, _ := s.matcher.Match(tags...)
		return s.supported[index].String()
	}

	tag, err := language.Parse(input)
	if err != nil {
		return ""
	}

	_, index, _ := s.matcher.Match(tag)
	return s.supported[index].String()
}

func canonicalizeLocale(input string, supported []language.Tag) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return supported[0].String()
	}

	tag, err := language.Parse(input)
	if err != nil {
		return supported[0].String()
	}

	matcher := language.NewMatcher(supported)
	_, index, _ := matcher.Match(tag)
	return supported[index].String()
}

func messageFromCatalog(catalogs map[string]map[string]string, locale string, key string) string {
	if locale == "" {
		return ""
	}

	messages, ok := catalogs[locale]
	if !ok {
		return ""
	}

	return messages[key]
}

func cloneCatalogs(source map[string]map[string]string) map[string]map[string]string {
	cloned := make(map[string]map[string]string, len(source))
	for locale, messages := range source {
		items := make(map[string]string, len(messages))
		for key, value := range messages {
			items[key] = value
		}
		cloned[locale] = items
	}

	return cloned
}
