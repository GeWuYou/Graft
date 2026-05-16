package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"graft/server/internal/config"
)

func newTestService() *Service {
	return New(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "en-US",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
}

// TestResolveLocaleUsesConfiguredFallbackOrder 验证 locale 解析会按请求、
// 会话、默认语言、回退语言的固定顺序收敛。
func TestResolveLocaleUsesConfiguredFallbackOrder(t *testing.T) {
	service := newTestService()

	if locale := service.ResolveLocale("en-US", "zh-CN"); locale != "en-US" {
		t.Fatalf("expected request locale to win, got %q", locale)
	}
	if locale := service.ResolveLocale("", "en-US"); locale != "en-US" {
		t.Fatalf("expected session locale to win, got %q", locale)
	}
	if locale := service.ResolveLocale("", ""); locale != "zh-CN" {
		t.Fatalf("expected default locale fallback, got %q", locale)
	}
	if locale := service.ResolveLocale("@@@", ""); locale != "zh-CN" {
		t.Fatalf("expected invalid locale to fall back to default, got %q", locale)
	}
}

// TestResolveRequestLocalePrefersExplicitHeader 验证平台自定义请求头会优先于
// Accept-Language 参与 locale 解析。
func TestResolveRequestLocalePrefersExplicitHeader(t *testing.T) {
	service := newTestService()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set(LocaleHeader, "en-US")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	locale := service.ResolveRequestLocale(request, "")
	if locale != "en-US" {
		t.Fatalf("expected explicit locale header to win, got %q", locale)
	}
}

// TestResolveRequestLocaleFallsBackToAcceptLanguage 验证缺少显式 locale 头时，
// 服务会回退解析 Accept-Language。
func TestResolveRequestLocaleFallsBackToAcceptLanguage(t *testing.T) {
	service := newTestService()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Accept-Language", "en-US,en;q=0.8")

	locale := service.ResolveRequestLocale(request, "")
	if locale != "en-US" {
		t.Fatalf("expected Accept-Language locale, got %q", locale)
	}
}

// TestMessageFallsBackToConfiguredLocalesAndKey 验证文案查找会优先命中解析后
// 的语言目录，缺失时再返回稳定 message key。
func TestMessageFallsBackToConfiguredLocalesAndKey(t *testing.T) {
	service := newTestService()

	if message := service.Message("en-US", "auth.missing_actor"); message != "Missing request actor" {
		t.Fatalf("expected en-US catalog message, got %q", message)
	}
	if message := service.Message("zh-CN", "auth.invalid_credentials"); message != "用户名或密码错误" {
		t.Fatalf("expected zh-CN auth login message, got %q", message)
	}
	if message := service.Message("en-US", "auth.invalid_refresh_session"); message != "Invalid or expired refresh session" {
		t.Fatalf("expected en-US auth refresh message, got %q", message)
	}
	if message := service.Message("en-US", "auth.session_not_found"); message != "Session not found or already inactive" {
		t.Fatalf("expected en-US auth session-not-found message, got %q", message)
	}
	if message := service.Message("en-US", "common.conjunction"); message != "and" {
		t.Fatalf("expected en-US shared conjunction message, got %q", message)
	}
	if message := service.Message("zh-CN", "common.copyright"); message != "Copyright (C) 2021-2026 Tencent. All Rights Reserved" {
		t.Fatalf("expected zh-CN shared copyright message, got %q", message)
	}
	if message := service.Message("en-US", "missing.key"); message != "missing.key" {
		t.Fatalf("expected missing key fallback, got %q", message)
	}
}
