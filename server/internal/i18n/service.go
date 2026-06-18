// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package i18n

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"

	"golang.org/x/text/language"

	"graft/server/internal/config"
	"graft/server/internal/contract/httpheader"
)

// LocaleHeader 允许调用方显式指定当前请求期望的语言。
const LocaleHeader = string(httpheader.Locale)

// Namespace 是 i18n registry 对外暴露的稳定消息归属边界。
type Namespace string

// LocaleTag 是项目 facade 收口后的稳定语言标签。
type LocaleTag string

// MessageKey 是 facade 内部 registry 使用的稳定消息键。
type MessageKey string

const (
	// CoreNamespace 承载平台级公共消息资源。
	CoreNamespace Namespace = "core"

	// LocaleZHCN 是当前平台允许的中文 locale。
	LocaleZHCN LocaleTag = "zh-CN"
	// LocaleENUS 是当前平台允许的英文 locale。
	LocaleENUS LocaleTag = "en-US"
)

// MessageID 标识一条具备 owner 语义的稳定消息资源。
type MessageID struct {
	Namespace Namespace
	Key       MessageKey
}

// MessageResource 表示一条具体 locale 下的消息文案。
type MessageResource struct {
	Key  MessageKey
	Text string
}

// Registration 描述一次显式消息资源注册。
type Registration struct {
	Namespace Namespace
	Locale    LocaleTag
	Messages  []MessageResource
}

// LookupRequest 描述 facade 级别的一次消息查找。
type LookupRequest struct {
	Namespace       Namespace
	Locale          LocaleTag
	Key             MessageKey
	FallbackMessage string
	TemplateData    map[string]any
}

var (
	errEmptyNamespace           = errors.New("i18n namespace is required")
	errEmptyMessageKey          = errors.New("i18n message key is required")
	errUnsupportedLocale        = errors.New("i18n locale is not supported")
	errFrozenRegistry           = errors.New("i18n registry is frozen")
	errDuplicateMessageResource = errors.New("i18n message already registered")
)

// Service 提供平台级 locale 解析、消息查找与模块注册能力。
//
// Service 保持为 `server` 唯一项目级 facade：调用方只消费 namespace、locale、
// message key、fallback message 等稳定项目概念，而不直接依赖底层实现细节。
// 该 facade 只拥有 key 注册、校验和回退规则，不拥有前端显示文案的长期所有权。
type Service struct {
	defaultLocale  string
	fallbackLocale string
	supported      []language.Tag
	supportedSet   map[string]struct{}
	matcher        language.Matcher

	mu       sync.RWMutex
	frozen   bool
	catalogs map[string]map[string]string
}

// New 使用配置快照创建项目级 i18n facade。
func New(cfg config.I18nConfig) (*Service, error) {
	supported := make([]language.Tag, 0, len(cfg.SupportedLocales))
	supportedSet := make(map[string]struct{}, len(cfg.SupportedLocales))
	for _, locale := range cfg.SupportedLocales {
		tag, err := language.Parse(locale)
		if err != nil {
			continue
		}

		canonical := tag.String()
		if _, exists := supportedSet[canonical]; exists {
			continue
		}

		supported = append(supported, tag)
		supportedSet[canonical] = struct{}{}
	}
	if len(supported) == 0 {
		tag := language.MustParse(string(LocaleZHCN))
		supported = []language.Tag{tag}
		supportedSet[tag.String()] = struct{}{}
	}

	service := &Service{
		supported:    supported,
		supportedSet: supportedSet,
		matcher:      language.NewMatcher(supported),
		catalogs:     make(map[string]map[string]string, len(supported)),
	}
	service.defaultLocale = canonicalizeLocale(cfg.DefaultLocale, supported)
	service.fallbackLocale = canonicalizeLocale(cfg.FallbackLocale, supported)
	if err := service.registerEmbeddedCatalogs(); err != nil {
		return nil, fmt.Errorf("register embedded i18n catalogs: %w", err)
	}
	return service, nil
}

// MustNew is a test-oriented helper for callers that expect a static i18n
// fixture and prefer setup failure over plumbing errors through each test body.
func MustNew(cfg config.I18nConfig) *Service {
	service, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return service
}

// DefaultLocale 返回当前服务使用的默认语言。
func (s *Service) DefaultLocale() string {
	return s.defaultLocale
}

// FallbackLocale 返回消息查找失败时的最终回退语言。
func (s *Service) FallbackLocale() string {
	return s.fallbackLocale
}

// SupportedLocales 返回当前 facade 允许的稳定 locale 列表。
func (s *Service) SupportedLocales() []string {
	items := make([]string, len(s.supported))
	for index, tag := range s.supported {
		items[index] = tag.String()
	}

	return items
}

// Freeze 将 registry 切换到只读状态。
func (s *Service) Freeze() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.frozen = true
	return nil
}

// IsFrozen 返回当前 registry 是否已经冻结。
func (s *Service) IsFrozen() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.frozen
}

// RegisterMessages 在注册期向 facade 注入一批显式消息资源。
func (s *Service) RegisterMessages(reg Registration) error {
	namespace := strings.TrimSpace(string(reg.Namespace))
	if namespace == "" {
		return errEmptyNamespace
	}

	locale, err := s.normalizeSupportedLocale(string(reg.Locale))
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.frozen {
		return errFrozenRegistry
	}

	catalog := s.ensureCatalog(locale)
	for _, item := range reg.Messages {
		key := strings.TrimSpace(string(item.Key))
		if key == "" {
			return errEmptyMessageKey
		}

		canonicalKey := composeMessageID(reg.Namespace, item.Key)
		if _, exists := catalog[canonicalKey]; exists {
			return fmt.Errorf("%w: %s", errDuplicateMessageResource, canonicalKey)
		}

		catalog[canonicalKey] = item.Text
	}

	return nil
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

	return string(LocaleZHCN)
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

// Message 返回给定语言和平台级消息 key 对应的最终文案。
//
// 这是保留给现有 `httpx`、guard 和模块调用面的兼容入口，内部会路由到
// `core` namespace。
func (s *Service) Message(locale string, key string) string {
	return s.Lookup(LookupRequest{
		Namespace: CoreNamespace,
		Locale:    LocaleTag(locale),
		Key:       MessageKey(key),
	})
}

// Lookup 使用 facade 的稳定项目概念做一次消息解析。
//
// 当 catalog 中缺少目标消息时，返回顺序固定为：
// 1. fallback locale / default locale 中的已注册文案
// 2. 显式传入的 FallbackMessage
// 3. 稳定 message key 自身
//
// 这保证跨边界调用方可以长期消费 `key + fallback`，而不是把 server 文案当成唯一真相。
func (s *Service) Lookup(req LookupRequest) string {
	key := strings.TrimSpace(string(req.Key))
	if key == "" {
		return ""
	}

	namespace := req.Namespace
	if strings.TrimSpace(string(namespace)) == "" {
		namespace = CoreNamespace
	}

	resolvedLocale := s.ResolveLocale(string(req.Locale), "")
	canonicalKey := composeMessageID(namespace, req.Key)
	for _, candidate := range []string{resolvedLocale, s.fallbackLocale, s.defaultLocale} {
		if message := s.messageFromCatalog(candidate, canonicalKey); message != "" {
			return message
		}
	}

	if strings.TrimSpace(req.FallbackMessage) != "" {
		return req.FallbackMessage
	}

	return key
}

func (s *Service) messageFromCatalog(locale string, key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if locale == "" {
		return ""
	}

	messages, ok := s.catalogs[locale]
	if !ok {
		return ""
	}

	return messages[key]
}

func (s *Service) ensureCatalog(locale string) map[string]string {
	catalog, ok := s.catalogs[locale]
	if ok {
		return catalog
	}

	catalog = make(map[string]string)
	s.catalogs[locale] = catalog
	return catalog
}

func (s *Service) normalizeSupportedLocale(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("%w: %s", errUnsupportedLocale, input)
	}

	tag, err := language.Parse(input)
	if err != nil {
		return "", fmt.Errorf("%w: %s", errUnsupportedLocale, input)
	}

	canonical := tag.String()
	if _, ok := s.supportedSet[canonical]; !ok {
		return "", fmt.Errorf("%w: %s", errUnsupportedLocale, canonical)
	}

	return canonical, nil
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

func composeMessageID(namespace Namespace, key MessageKey) string {
	namespaceValue := strings.TrimSpace(string(namespace))
	keyValue := strings.TrimSpace(string(key))
	if namespaceValue == "" {
		return keyValue
	}

	return fmt.Sprintf("%s.%s", namespaceValue, keyValue)
}

// RegisteredMessageKeys 返回指定 namespace + locale 下当前已注册的 canonical key 列表。
//
// 该接口主要服务测试与后续 registry 可观测性，不应在业务热路径中按它做控制流。
func (s *Service) RegisteredMessageKeys(namespace Namespace, locale LocaleTag) []string {
	normalizedLocale, err := s.normalizeSupportedLocale(string(locale))
	if err != nil {
		return nil
	}

	prefix := strings.TrimSpace(string(namespace))
	if prefix != "" {
		prefix += "."
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	catalog, ok := s.catalogs[normalizedLocale]
	if !ok {
		return nil
	}

	keys := make([]string, 0, len(catalog))
	for key := range catalog {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}

		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// RegisteredMessageKeyIDs returns canonical message IDs currently registered for
// a bare message key in the given locale.
//
// This is intended for diagnostics and governance tests that validate external
// descriptors, such as menu title keys, without coupling those descriptors to a
// runtime namespace lookup path.
func (s *Service) RegisteredMessageKeyIDs(locale LocaleTag, key MessageKey) []string {
	normalizedLocale, err := s.normalizeSupportedLocale(string(locale))
	if err != nil {
		return nil
	}

	keyValue := strings.TrimSpace(string(key))
	if keyValue == "" {
		return nil
	}
	suffix := "." + keyValue

	s.mu.RLock()
	defer s.mu.RUnlock()

	catalog, ok := s.catalogs[normalizedLocale]
	if !ok {
		return nil
	}

	keys := make([]string, 0)
	for canonicalKey := range catalog {
		if canonicalKey == keyValue || strings.HasSuffix(canonicalKey, suffix) {
			keys = append(keys, canonicalKey)
		}
	}
	slices.Sort(keys)
	return keys
}

// RegisteredMessageResources returns canonical message IDs and texts registered
// for a bare message key in the given locale.
//
// Like RegisteredMessageKeyIDs, this is intended for diagnostics and governance
// tests that need to prove descriptors resolve to real catalog messages.
func (s *Service) RegisteredMessageResources(locale LocaleTag, key MessageKey) []MessageResource {
	normalizedLocale, err := s.normalizeSupportedLocale(string(locale))
	if err != nil {
		return nil
	}

	keyValue := strings.TrimSpace(string(key))
	if keyValue == "" {
		return nil
	}
	suffix := "." + keyValue

	s.mu.RLock()
	defer s.mu.RUnlock()

	catalog, ok := s.catalogs[normalizedLocale]
	if !ok {
		return nil
	}

	items := make([]MessageResource, 0)
	for canonicalKey, text := range catalog {
		if canonicalKey == keyValue || strings.HasSuffix(canonicalKey, suffix) {
			items = append(items, MessageResource{Key: MessageKey(canonicalKey), Text: text})
		}
	}
	slices.SortFunc(items, func(left, right MessageResource) int {
		return strings.Compare(string(left.Key), string(right.Key))
	})
	return items
}
