package runtimetarget

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/config"
	containerdi "graft/server/internal/container"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
	runtimetargetlocales "graft/server/modules/runtime-target/locales"
)

func TestLocalRuntimeTargetStartsWithoutAgentSecrets(t *testing.T) {
	for _, pepperFile := range []string{"", filepath.Join(t.TempDir(), "missing-agent-pepper")} {
		t.Run(filepath.Base(pepperFile), func(t *testing.T) {
			cfg := &config.Config{Execution: config.ExecutionConfig{Mode: config.ExecutionModeLocal}, EnrollmentSecurity: config.EnrollmentSecurityConfig{PepperFile: pepperFile}}
			assertLocalRuntimeTargetStartup(t, cfg)
		})
	}
}

func assertLocalRuntimeTargetStartup(t *testing.T, cfg *config.Config) {
	t.Helper()
	ctx, mock := newLocalStartupContext(t, cfg)
	built, err := NewModuleSpec().Builder.Build(module.BuildContext{Services: ctx.Services})
	if err != nil {
		t.Fatalf("build local runtime target: %v", err)
	}
	if err := built.Register(ctx); err != nil {
		t.Fatalf("register local runtime target: %v", err)
	}
	if _, err := os.Stat("/var/run/docker.sock"); errors.Is(err, os.ErrNotExist) {
		mock.ExpectQuery("SELECT .* FROM runtime_targets").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	} else {
		mock.ExpectExec("INSERT INTO runtime_targets").WillReturnResult(sqlmock.NewResult(1, 1))
	}
	if err := built.Boot(ctx); err != nil {
		t.Fatalf("boot local runtime target: %v", err)
	}
	t.Cleanup(func() {
		if err := built.Shutdown(ctx); err != nil {
			t.Errorf("shutdown local runtime target: %v", err)
		}
	})
	bootstrap, err := module.ResolveService[moduleapi.AgentBootstrapAuthority](ctx.Services, (*moduleapi.AgentBootstrapAuthority)(nil))
	if err != nil {
		t.Fatalf("resolve disabled bootstrap without Vault issuer: %v", err)
	}
	if _, err := bootstrap.BootstrapAgent(ctx.LifecycleContext, moduleapi.AgentBootstrapRequest{}); !errors.Is(err, moduleapi.ErrAgentBootstrapRejected) {
		t.Fatalf("local bootstrap error = %v, want rejected", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("startup database expectations: %v", err)
	}
}

func TestAgentRuntimeTargetRejectsUnavailableConfiguredPepper(t *testing.T) {
	for _, mode := range []string{"", config.ExecutionModeAgent} {
		cfg := &config.Config{Execution: config.ExecutionConfig{Mode: mode}, EnrollmentSecurity: config.EnrollmentSecurityConfig{PepperFile: filepath.Join(t.TempDir(), "missing-agent-pepper")}}
		ctx, _ := newLocalStartupContext(t, cfg)
		if _, err := NewModuleSpec().Builder.Build(module.BuildContext{Services: ctx.Services}); err == nil || !strings.Contains(err.Error(), "enrollment pepper source is unavailable") {
			t.Fatalf("agent mode %q build error = %v, want unavailable pepper", mode, err)
		}
	}
}

func newLocalStartupContext(t *testing.T, cfg *config.Config) (*module.Context, sqlmock.Sqlmock) {
	t.Helper()
	db, mock := newRuntimeTargetAuditMock(t)
	services := containerdi.New()
	for _, item := range []struct{ key, value any }{
		{(*sql.DB)(nil), db},
		{(*config.Config)(nil), cfg},
		{(*moduleapi.AuthService)(nil), struct{ moduleapi.AuthService }{}},
		{(*moduleapi.Authorizer)(nil), struct{ moduleapi.Authorizer }{}},
		{(*moduleapi.UserIdentityProvider)(nil), struct{ moduleapi.UserIdentityProvider }{}},
		{(*moduleapi.UserCandidateReader)(nil), struct{ moduleapi.UserCandidateReader }{}},
		{(*moduleapi.UserSummaryBatchReader)(nil), struct {
			moduleapi.UserSummaryBatchReader
		}{}},
		{(*moduleapi.SavedViewService)(nil), struct{ moduleapi.SavedViewService }{}},
		{(*realtimeauth.Service)(nil), realtimeauth.NewMemoryService()},
		{(*realtime.TopicIssuerRegistry)(nil), realtime.NewTopicIssuerRegistry()},
	} {
		if err := services.RegisterSingleton(item.key, func(containerdi.Resolver) (any, error) { return item.value, nil }); err != nil {
			t.Fatalf("register startup dependency: %v", err)
		}
	}
	localizer := i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}})
	resources, err := runtimetargetlocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("load runtime target locales: %v", err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register runtime target locales: %v", err)
	}
	return &module.Context{
		LifecycleContext: context.Background(), Config: cfg, Logger: zap.NewNop(), I18n: localizer,
		Services: services, Router: gin.New(), Realtime: &runtimeTargetRealtimeHubStub{},
		PermissionRegistry: permission.NewRegistry(), MenuRegistry: menu.NewRegistry(),
		EventTxPublisher: &runtimeTargetRecordingPublisher{},
	}, mock
}
