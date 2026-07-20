package security

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/logger"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	securitycontract "graft/server/modules/security/contract"
)

const securityMenuOrderOverview = 1

// Module 拥有安全域聚合只读接口，并负责其权限、菜单和路由生命周期。
type Module struct{}

// NewModule 创建安全概览模块实例；聚合数据由跨模块只读服务提供。
func NewModule() *Module { return &Module{} }

// Register 声明安全概览权限、菜单和 HTTP 路由；此阶段只做注册，不读取业务数据。
func (m *Module) Register(ctx *module.Context) error {
	if ctx == nil || ctx.Services == nil {
		return errors.New("security module context services are unavailable")
	}
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	registerPermissions(ctx.PermissionRegistry)
	registerMenu(ctx.MenuRegistry)

	authService, err := module.ResolveService[moduleapi.AuthService](ctx.Services, (*moduleapi.AuthService)(nil))
	if err != nil {
		return fmt.Errorf("resolve auth service: %w", err)
	}
	authorizer, err := module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
	if err != nil {
		return fmt.Errorf("resolve route authorizer: %w", err)
	}
	rbacPosture, err := module.ResolveService[moduleapi.RBACSecurityPostureService](ctx.Services, (*moduleapi.RBACSecurityPostureService)(nil))
	if err != nil {
		return fmt.Errorf("resolve rbac security posture service: %w", err)
	}
	auditReader, err := module.ResolveService[moduleapi.AuditSecurityReader](ctx.Services, (*moduleapi.AuditSecurityReader)(nil))
	if err != nil {
		return fmt.Errorf("resolve audit security reader: %w", err)
	}
	if ctx.Router == nil {
		return nil
	}

	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	group := ctx.Router.Group(securitycontract.SecurityGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(securitycontract.OverviewCollection,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, securitycontract.OverviewReadPermission.String(), publisher),
		handleOverview(ctx, rbacPosture, auditReader),
	)
	return nil
}

// Boot 当前没有安全模块自有的运行时启动行为。
func (m *Module) Boot(_ *module.Context) error { return nil }

// Shutdown 当前没有安全模块自有的运行时资源需要释放。
func (m *Module) Shutdown(_ *module.Context) error { return nil }

// registerPermissions 将 Security Overview 的读取权限注册到权限注册表中。
func registerPermissions(registry *permission.Registry) {
	if registry == nil {
		return
	}
	registry.Register(permission.Item{
		Code:           securitycontract.OverviewReadPermission.String(),
		DisplayKey:     securitycontract.OverviewReadDisplay.String(),
		DescriptionKey: securitycontract.OverviewReadDescription.String(),
		Module:         moduleID,
	})
}

// registerMenu 在菜单注册表可用时注册安全概览菜单；菜单注册表缺失不改变 API 能力。
func registerMenu(registry *menu.Registry) {
	if registry == nil {
		return
	}
	registry.Register(menu.Item{
		Code:       "security.overview",
		ParentCode: "domain.security",
		Kind:       menu.NodeKindEntry,
		Title:      "Security Overview",
		TitleKey:   securitycontract.OverviewMenuTitle.String(),
		Path:       securitycontract.OverviewMenuPath,
		Icon:       "security-posture",
		Order:      securityMenuOrderOverview,
		Permission: securitycontract.OverviewReadPermission.String(),
		Module:     moduleID,
	})
}

// registerMessages 验证安全模块所需的中英文国际化消息资源是否已注册。
// 如果本地化服务不可用或任一语言缺少所需消息资源，则返回错误。
func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}
	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range []securitycontract.MessageKey{
			securitycontract.OverviewMenuTitle,
			securitycontract.OverviewReadDisplay,
			securitycontract.OverviewReadDescription,
		} {
			if len(localizer.RegisteredMessageResources(locale, i18n.MessageKey(key.String()))) == 0 {
				return fmt.Errorf("register security module messages: locale resource %s missing key %s", locale, key)
			}
		}
	}
	return nil
}

type overviewResponse struct {
	TimePreset    moduleapi.AuditOverviewPreset   `json:"time_preset"`
	AccessControl moduleapi.SecurityPosture       `json:"access_control"`
	Audit         moduleapi.AuditSecuritySnapshot `json:"audit"`
}

// handleOverview 创建安全概览 HTTP 处理器。
// 处理器从请求参数解析有界时间窗口，分别读取 RBAC 和审计快照并聚合为统一响应；
// 任一依赖失败都通过模块上下文的本地化错误出口返回，不向调用方泄漏底层存储错误。
func handleOverview(
	ctx *module.Context,
	rbacPosture moduleapi.RBACSecurityPostureService,
	auditReader moduleapi.AuditSecurityReader,
) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		preset, ok := parseOverviewPreset(ginCtx.Query("preset"))
		if !ok {
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "preset"})
			return
		}
		requestCtx := ginCtx.Request.Context()
		posture, err := rbacPosture.ReadSecurityPosture(requestCtx)
		if err != nil {
			reported := logger.ReportError(requestCtx, resolveSecurityAppLogger(ctx), "read security posture failed", err,
				logger.StringField(logger.FieldOperation, "read_security_posture"),
			)
			httpx.AbortAppError(ginCtx, ctx.I18n, ctx.Logger, reported)
			return
		}
		auditSnapshot, err := auditReader.ReadSecuritySnapshot(requestCtx, preset)
		if err != nil {
			reported := logger.ReportError(requestCtx, resolveSecurityAppLogger(ctx), "read security audit snapshot failed", err,
				logger.StringField(logger.FieldOperation, "read_security_audit_snapshot"),
				logger.StringField("preset", string(preset)),
			)
			httpx.AbortAppError(ginCtx, ctx.I18n, ctx.Logger, reported)
			return
		}
		httpx.WriteSuccess(ginCtx, http.StatusOK, overviewResponse{TimePreset: preset, AccessControl: posture, Audit: auditSnapshot})
	}
}

func resolveSecurityAppLogger(ctx *module.Context) logger.AppLogger {
	if ctx == nil || ctx.AppLogger == nil {
		return nil
	}
	return ctx.AppLogger.Named("modules.security.overview")
}

// parseOverviewPreset 将审计概览时间参数解析为受支持的时间范围。
// 返回规范化的时间范围及其是否有效。
func parseOverviewPreset(value string) (moduleapi.AuditOverviewPreset, bool) {
	switch moduleapi.AuditOverviewPreset(strings.TrimSpace(value)) {
	case "", moduleapi.AuditOverviewPresetLast24Hours:
		return moduleapi.AuditOverviewPresetLast24Hours, true
	case moduleapi.AuditOverviewPresetLast7Days:
		return moduleapi.AuditOverviewPresetLast7Days, true
	case moduleapi.AuditOverviewPresetLast30Days:
		return moduleapi.AuditOverviewPresetLast30Days, true
	default:
		return "", false
	}
}

var _ module.Module = (*Module)(nil)
