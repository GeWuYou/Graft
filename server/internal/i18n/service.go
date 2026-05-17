package i18n

import (
	"net/http"
	"strings"

	"golang.org/x/text/language"

	"graft/server/internal/config"
	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
)

// LocaleHeader 允许调用方显式指定当前请求期望的语言。
const LocaleHeader = string(httpheader.Locale)

type catalogEntry struct {
	key  messagecontract.Key
	zhCN string
	enUS string
}

// #nosec G101 -- 这里保存的是本地化 message key 与展示文案，不是凭据。
var defaultCatalogEntries = []catalogEntry{
	{key: messagecontract.AuthInvalidCredentials, zhCN: "用户名或密码错误", enUS: "Invalid username or password"},
	{key: messagecontract.AuthTokenMissing, zhCN: "缺少访问令牌", enUS: "Missing access token"},
	{key: messagecontract.AuthTokenExpired, zhCN: "访问令牌已过期", enUS: "Access token expired"},
	{key: messagecontract.AuthTokenInvalid, zhCN: "访问令牌无效", enUS: "Invalid access token"},
	{key: messagecontract.AuthForbidden, zhCN: "权限不足", enUS: "Forbidden"},
	{key: messagecontract.AuthInvalidRefreshSession, zhCN: "刷新会话无效或已失效", enUS: "Invalid or expired refresh session"},
	{key: messagecontract.AuthPasswordPolicyViolation, zhCN: "新密码不符合安全要求", enUS: "New password does not meet security requirements"},
	{key: messagecontract.AuthPasswordReuseForbidden, zhCN: "新密码不能重复使用默认密码或当前密码", enUS: "New password must not reuse the default or current password"},
	{key: messagecontract.AuthCurrentPasswordInvalid, zhCN: "当前密码错误", enUS: "Current password is invalid"},
	{key: messagecontract.AuthMissingActor, zhCN: "缺少请求身份信息", enUS: "Missing request actor"},
	{key: messagecontract.AuthMissingPermission, zhCN: "缺少所需权限", enUS: "Missing required permission"},
	{key: messagecontract.AuthSessionNotFound, zhCN: "会话不存在或已失效", enUS: "Session not found or already inactive"},
	{key: messagecontract.CommonConjunction, zhCN: "和", enUS: "and"},
	{key: messagecontract.CommonCopyright, zhCN: "Copyright (C) 2021-2026 Tencent. All Rights Reserved", enUS: "Copyright (C) 2021-2026 Tencent. All Rights Reserved"},
	{key: messagecontract.CommonInternalError, zhCN: "服务内部错误", enUS: "Internal server error"},
	{key: messagecontract.CommonInvalidArgument, zhCN: "请求参数不合法", enUS: "Invalid request parameters"},
	{key: messagecontract.RoleNotFound, zhCN: "角色不存在", enUS: "Role not found"},
	{key: messagecontract.UserNotFound, zhCN: "用户不存在", enUS: "User not found"},
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
		catalogs:       buildDefaultCatalogs(),
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

	requested := strings.TrimSpace(request.Header.Get(httpheader.Locale.String()))
	if requested == "" {
		requested = strings.TrimSpace(request.Header.Get(httpheader.AcceptLanguage.String()))
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

func buildDefaultCatalogs() map[string]map[string]string {
	catalogs := map[string]map[string]string{
		"zh-CN": make(map[string]string, len(defaultCatalogEntries)),
		"en-US": make(map[string]string, len(defaultCatalogEntries)),
	}

	for _, entry := range defaultCatalogEntries {
		key := entry.key.String()
		catalogs["zh-CN"][key] = entry.zhCN
		catalogs["en-US"][key] = entry.enUS
	}

	return catalogs
}
