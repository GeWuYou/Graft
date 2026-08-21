// Package runtimetarget 负责持久化运行时连接身份和发现事实。
package runtimetarget

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"graft/server/internal/config"
	containerdi "graft/server/internal/container"

	messagecontract "graft/server/internal/contract/message"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/event"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
	contract "graft/server/modules/runtime-target/contract"
	store "graft/server/modules/runtime-target/store"
)

const maxRuntimeTargetID = uint64(^uint64(0) >> 1)

const runtimeTargetListSummaryConcurrency = 4
const runtimeTargetListKeywordMaxLength = 128

// Module 暴露 runtime-target API 路由，并提供有界的 Docker Target 发现与 provider 能力。
type Module struct {
	repository      *store.SQLRepository
	pepper          *config.EnrollmentPepperProvider
	summaries       *summaryCache
	authorizer      moduleapi.Authorizer
	realtimeTickets realtimeauth.Service
	topicIssuers    realtime.TopicIssuerRegistry
	collector       *runtimeTargetSummaryCollector
	runtimeLogger   *zap.Logger
	i18n            *i18n.Service
	events          event.TransactionalPublisher
	savedViews      moduleapi.SavedViewService
	users           moduleapi.UserIdentityProvider
	candidates      moduleapi.UserCandidateReader
	userSummaries   moduleapi.UserSummaryBatchReader
}

// NewModule 构造 runtime-target 模块实例。省略 pepper 时，Agent Bootstrap authority 保持未配置并拒绝引导请求。
func NewModule(repository *store.SQLRepository, pepper ...*config.EnrollmentPepperProvider) *Module {
	var enrollmentPepper *config.EnrollmentPepperProvider
	if len(pepper) > 0 {
		enrollmentPepper = pepper[0]
	}
	return &Module{repository: repository, pepper: enrollmentPepper, summaries: newSummaryCache()}
}

// Register 声明 runtime-target 权限、菜单元数据和 API 路由。
//
//nolint:cyclop // 模块依赖与路由注册必须保持在同一显式装配边界。
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.repository == nil {
		return errors.New("runtime target repository is unavailable")
	}
	if err := registerModuleMetadata(ctx, moduleID); err != nil {
		return err
	}
	if err := registerRuntimeTargetDashboardWidget(ctx, m.repository); err != nil {
		return err
	}
	m.runtimeLogger = ctx.Logger
	m.i18n = ctx.I18n
	if ctx.EventTxPublisher == nil {
		return errors.New("runtime target event transaction publisher is unavailable")
	}
	m.events = ctx.EventTxPublisher
	services, err := m.resolveRegistrationServices(ctx)
	if err != nil {
		return err
	}
	m.users = services.users
	m.candidates = services.candidates
	m.userSummaries = services.userSummaries
	m.savedViews = services.savedViews
	if err := m.configureRealtime(ctx, services.authorizer); err != nil {
		return err
	}
	if err := m.registerReaders(ctx); err != nil {
		return err
	}
	m.registerRoutes(ctx, services.auth, services.authorizer)
	return nil
}

type registrationServices struct {
	auth          moduleapi.AuthService
	authorizer    moduleapi.Authorizer
	users         moduleapi.UserIdentityProvider
	candidates    moduleapi.UserCandidateReader
	userSummaries moduleapi.UserSummaryBatchReader
	savedViews    moduleapi.SavedViewService
}

func (m *Module) resolveRegistrationServices(ctx *module.Context) (registrationServices, error) {
	auth, err := module.ResolveService[moduleapi.AuthService](ctx.Services, (*moduleapi.AuthService)(nil))
	if err != nil {
		return registrationServices{}, err
	}
	authorizer, err := module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
	if err != nil {
		return registrationServices{}, err
	}
	users, err := module.ResolveService[moduleapi.UserIdentityProvider](ctx.Services, (*moduleapi.UserIdentityProvider)(nil))
	if err != nil {
		return registrationServices{}, err
	}
	candidates, err := module.ResolveService[moduleapi.UserCandidateReader](ctx.Services, (*moduleapi.UserCandidateReader)(nil))
	if err != nil {
		return registrationServices{}, err
	}
	userSummaries, err := module.ResolveService[moduleapi.UserSummaryBatchReader](ctx.Services, (*moduleapi.UserSummaryBatchReader)(nil))
	if err != nil {
		return registrationServices{}, err
	}
	savedViews, err := module.ResolveService[moduleapi.SavedViewService](ctx.Services, (*moduleapi.SavedViewService)(nil))
	if err != nil {
		return registrationServices{}, err
	}
	return registrationServices{auth: auth, authorizer: authorizer, users: users, candidates: candidates, userSummaries: userSummaries, savedViews: savedViews}, nil
}

//nolint:cyclop,gocyclo,gocognit // Runtime Target 在同一注册边界内装配公开读取器与 provider-owned build 能力。
func (m *Module) registerReaders(ctx *module.Context) error {
	reader := func(_ containerdi.Resolver) (any, error) { return runtimeTargetReader{repository: m.repository}, nil }
	if err := ctx.Services.RegisterSingleton((*moduleapi.RuntimeTargetReader)(nil), reader); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.ComposeRuntimeTargetReader)(nil), reader); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.RuntimeTargetDeploymentAssignmentReader)(nil), reader); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.BuildRuntimeTargetReader)(nil), reader); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.RuntimeTargetBuildAssignmentReader)(nil), reader); err != nil {
		return err
	}
	telemetry := func(_ containerdi.Resolver) (any, error) {
		provider := controlPlaneBuilderTelemetryProvider{repository: m.repository}
		return controlPlaneBuilderTelemetryReader{provider: provider}, nil
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.RuntimeTargetBuilderTelemetryReader)(nil), telemetry); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.RuntimeTargetBuilderTelemetryControlPlane)(nil), func(_ containerdi.Resolver) (any, error) {
		return controlPlaneBuilderTelemetryIngress{}, nil
	}); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.RuntimeTargetAgentBindingReader)(nil), func(_ containerdi.Resolver) (any, error) {
		return runtimeTargetAgentBindingReader{repository: m.repository}, nil
	}); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.RuntimeTargetAgentLedgerReader)(nil), func(_ containerdi.Resolver) (any, error) {
		return newRuntimeTargetAgentLedgerAuthority(m.repository), nil
	}); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.AgentEnrollmentAuthority)(nil), func(_ containerdi.Resolver) (any, error) {
		return newRuntimeTargetAgentEnrollmentAuthority(m.repository, m.events), nil
	}); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.AgentDeliveryAuthority)(nil), func(_ containerdi.Resolver) (any, error) {
		return newRuntimeTargetAgentDeliveryAuthority(m.repository, m.pepper), nil
	}); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.AgentBootstrapAuthority)(nil), func(resolver containerdi.Resolver) (any, error) {
		issuer, err := module.ResolveService[moduleapi.AgentCertificateIssuer](resolver, (*moduleapi.AgentCertificateIssuer)(nil))
		if err != nil {
			return nil, fmt.Errorf("resolve agent certificate issuer: %w", err)
		}
		return newRuntimeTargetAgentBootstrapAuthority(m.repository, m.pepper, issuer), nil
	}); err != nil {
		return err
	}
	connectionReader := func(_ containerdi.Resolver) (any, error) { return m.repository, nil }
	if err := ctx.Services.RegisterSingleton((*moduleapi.RuntimeTargetProviderConnectionReader)(nil), connectionReader); err != nil {
		return err
	}
	provider := func(_ containerdi.Resolver) (any, error) {
		return dockerTargetProvider{repository: m.repository}, nil
	}
	credentialProvider, credentialErr := module.ResolveService[moduleapi.CredentialProvider](ctx.Services, (*moduleapi.CredentialProvider)(nil))
	if credentialErr != nil && !errors.Is(credentialErr, containerdi.ErrServiceNotRegistered) {
		return credentialErr
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.TargetBoundDockerImageBuildCapability)(nil), provider); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.TargetBoundWorkspaceSnapshotDeliveryCapability)(nil), provider); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.TargetBoundProviderExecutionConformanceCapability)(nil), provider); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.TargetBoundDockerBuildProvider)(nil), provider); err != nil {
		return err
	}
	if credentialErr == nil && credentialProvider != nil {
		verificationAdapter := func(_ containerdi.Resolver) (any, error) {
			return dockerCredentialExecutionAdapter{provider: credentialProvider, client: dockerTargetProvider{repository: m.repository}}, nil
		}
		if err := ctx.Services.RegisterSingleton((*moduleapi.RuntimeExecutionAdapter)(nil), verificationAdapter); err != nil {
			return err
		}
		if err := ctx.Services.RegisterSingleton((*moduleapi.RuntimeOCIRegistryVerifier)(nil), verificationAdapter); err != nil {
			return err
		}
	}
	return nil
}

func (m *Module) registerRoutes(ctx *module.Context, auth moduleapi.AuthService, authorizer moduleapi.Authorizer) {
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	ctx.Router.GET("/runtime-targets", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.ViewPermission, publisher), m.handleList)
	ctx.Router.GET("/runtime-target-saved-views", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.ViewPermission, publisher), m.handleSavedViewList)
	ctx.Router.POST("/runtime-target-saved-views", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.ViewPermission, publisher), m.handleSavedViewCreate)
	ctx.Router.PUT("/runtime-target-saved-views/:viewId", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.ViewPermission, publisher), m.handleSavedViewUpdate)
	ctx.Router.DELETE("/runtime-target-saved-views/:viewId", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.ViewPermission, publisher), m.handleSavedViewDelete)
	ctx.Router.POST("/runtime-targets/discover-local-docker", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.RefreshPermission, publisher), m.handleDiscoverLocal(ctx))
	ctx.Router.POST("/runtime-target-assignments/batch", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.AssignmentManagePermission, publisher), m.handleBatchAssignments)
	ctx.Router.GET("/runtime-targets/:id", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.ViewPermission, publisher), m.handleDetail)
	ctx.Router.POST("/runtime-targets/:id/refresh", httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.RefreshPermission, publisher), m.handleRefresh(ctx))
	ctx.Router.GET(contract.AssignmentsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.AssignmentManagePermission, publisher), m.handleListAssignments)
	ctx.Router.POST(contract.AssignmentsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.AssignmentManagePermission, publisher), m.handleGrantAssignment)
	ctx.Router.PUT(contract.AssignmentsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.AssignmentManagePermission, publisher), m.handleReplaceAssignments)
	ctx.Router.GET(contract.AssignmentCandidatesRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.AssignmentManagePermission, publisher), m.handleAssignmentCandidates)
	ctx.Router.POST(contract.AssignmentDeleteRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, contract.AssignmentManagePermission, publisher), m.handleRevokeAssignment)
}

// ListAssignedComposeTargets 仅返回指定用户获授权用于部署的目标摘要。
func (r runtimeTargetReader) ListAssignedComposeTargets(ctx context.Context, userID uint64) ([]moduleapi.ComposeRuntimeTargetSummary, error) {
	items, err := r.repository.ListAssignedComposeTargets(ctx, userID)
	if err != nil {
		return nil, err
	}
	results := make([]moduleapi.ComposeRuntimeTargetSummary, 0, len(items))
	for _, target := range items {
		if summary, ok := composeTargetSummary(target); ok {
			results = append(results, summary)
		}
	}
	return results, nil
}

// CanUseComposeTarget 判断用户是否对具备 Compose 能力的目标拥有有效部署使用授权。
func (r runtimeTargetReader) CanUseComposeTarget(ctx context.Context, userID uint64, targetID uint64) (bool, error) {
	if r.repository == nil || userID == 0 || targetID == 0 {
		return false, nil
	}
	target, err := r.repository.Get(ctx, targetID)
	if err != nil {
		return false, normalizeRuntimeTargetLookupError(err)
	}
	if _, ok := composeTargetSummary(target); !ok {
		return false, nil
	}
	return r.repository.HasActiveUserAssignment(ctx, targetID, userID)
}

// ReadBuildTarget 返回指定运行目标的构建能力摘要，不携带用户授权判断。
func (r runtimeTargetReader) ReadBuildTarget(ctx context.Context, targetID int64) (moduleapi.BuildRuntimeTargetSummary, error) {
	if r.repository == nil || targetID < 1 {
		return moduleapi.BuildRuntimeTargetSummary{}, store.ErrNotFound
	}
	target, err := r.repository.Get(ctx, uint64(targetID)) //nolint:gosec // targetID is positive and bounded by the public API.
	if err != nil {
		return moduleapi.BuildRuntimeTargetSummary{}, err
	}
	if _, err := r.repository.GetDockerTargetConnection(ctx, uint64(targetID)); err != nil { //nolint:gosec // targetID is positive and bounded by the public API.
		return moduleapi.BuildRuntimeTargetSummary{}, err
	}
	summary, ok := buildTargetSummary(target)
	if !ok || summary.ID != targetID {
		return moduleapi.BuildRuntimeTargetSummary{}, store.ErrNotFound
	}
	return summary, nil
}

// ListAssignedBuildTargets 仅返回调用者获授权且当前可用于构建的目标摘要。
func (r runtimeTargetReader) ListAssignedBuildTargets(ctx context.Context, userID uint64) ([]moduleapi.BuildRuntimeTargetSummary, error) {
	targets, err := r.repository.ListBuildTargets(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]moduleapi.BuildRuntimeTargetSummary, 0, len(targets))
	for _, target := range targets {
		if _, connectionErr := r.repository.GetDockerTargetConnection(ctx, target.ID); connectionErr != nil {
			continue
		}
		summary, ok := buildTargetSummary(target)
		if !ok {
			continue
		}
		assigned, assignmentErr := r.repository.HasActiveUserAssignment(ctx, uint64(summary.ID), userID) //nolint:gosec // summary ID originated from a positive database ID.
		if assignmentErr != nil {
			return nil, assignmentErr
		}
		if assigned {
			results = append(results, summary)
		}
	}
	return results, nil
}

// CanUseBuildTarget 复核调用者是否可使用当前健康且具备构建能力的运行目标。
func (r runtimeTargetReader) CanUseBuildTarget(ctx context.Context, userID uint64, targetID int64) (bool, error) {
	if r.repository == nil || userID == 0 || targetID < 1 {
		return false, nil
	}
	target, err := r.repository.Get(ctx, uint64(targetID)) //nolint:gosec // targetID is positive and bounded by the public API.
	if err != nil {
		return false, nil
	}
	if _, err := r.repository.GetDockerTargetConnection(ctx, uint64(targetID)); err != nil { //nolint:gosec // targetID is positive and bounded by the public API.
		return false, nil
	}
	summary, ok := buildTargetSummary(target)
	if !ok || summary.ID != targetID {
		return false, nil
	}
	return r.repository.HasActiveUserAssignment(ctx, uint64(targetID), userID)
}

func (m *Module) configureRealtime(ctx *module.Context, authorizer moduleapi.Authorizer) error {
	realtimeTickets, err := module.ResolveService[realtimeauth.Service](ctx.Services, (*realtimeauth.Service)(nil))
	if err != nil {
		return err
	}
	topicIssuers, err := module.ResolveService[realtime.TopicIssuerRegistry](ctx.Services, (*realtime.TopicIssuerRegistry)(nil))
	if err != nil {
		return err
	}
	hub := ctx.Realtime
	if hub == nil {
		hub, err = module.ResolveService[realtime.Hub](ctx.Services, (*realtime.Hub)(nil))
		if err != nil {
			return err
		}
	}
	m.authorizer = authorizer
	m.realtimeTickets = realtimeTickets
	m.topicIssuers = topicIssuers
	m.collector = newRuntimeTargetSummaryCollector(hub, m.runtimeLogger, m.collectRealtimeSummaries)
	return m.topicIssuers.Register(contract.SummaryTopic, m)
}

type runtimeTargetReader struct {
	repository *store.SQLRepository
}

func (r runtimeTargetReader) ReadDockerTarget(ctx context.Context, id *int64) (moduleapi.RuntimeTargetSummary, error) {
	if r.repository == nil {
		return moduleapi.RuntimeTargetSummary{}, store.ErrNotFound
	}
	if id != nil {
		return r.readDockerTargetByID(ctx, *id)
	}
	return r.readSystemDockerTarget(ctx)
}

func (r runtimeTargetReader) readDockerTargetByID(ctx context.Context, id int64) (moduleapi.RuntimeTargetSummary, error) {
	if id < 1 {
		return moduleapi.RuntimeTargetSummary{}, store.ErrNotFound
	}
	target, err := r.repository.Get(ctx, uint64(id))
	if err != nil {
		return moduleapi.RuntimeTargetSummary{}, normalizeRuntimeTargetLookupError(err)
	}
	return dockerTargetSummary(target)
}

func (r runtimeTargetReader) readSystemDockerTarget(ctx context.Context) (moduleapi.RuntimeTargetSummary, error) {
	target, err := r.repository.FindSystemLocalDocker(ctx)
	if err != nil {
		return moduleapi.RuntimeTargetSummary{}, normalizeRuntimeTargetLookupError(err)
	}
	return dockerTargetSummary(target)
}

func normalizeRuntimeTargetLookupError(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return store.ErrNotFound
	}
	return err
}

func dockerTargetSummary(target store.Target) (moduleapi.RuntimeTargetSummary, error) {
	if target.Provider != "docker" || target.ID > maxRuntimeTargetID {
		return moduleapi.RuntimeTargetSummary{}, store.ErrNotFound
	}
	return moduleapi.RuntimeTargetSummary{ID: int64(target.ID), DisplayName: target.DisplayName, Provider: target.Provider}, nil
}

func (r runtimeTargetReader) ListDockerTargets(ctx context.Context) ([]moduleapi.RuntimeTargetSummary, error) {
	if r.repository == nil {
		return []moduleapi.RuntimeTargetSummary{}, nil
	}
	items, err := r.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]moduleapi.RuntimeTargetSummary, 0, len(items))
	for _, target := range items {
		if target.Provider == "docker" && target.ID <= maxRuntimeTargetID {
			results = append(results, moduleapi.RuntimeTargetSummary{ID: int64(target.ID), DisplayName: target.DisplayName, Provider: target.Provider})
		}
	}
	return results, nil
}

func (r runtimeTargetReader) ReadComposeTarget(ctx context.Context, id *int64) (moduleapi.ComposeRuntimeTargetSummary, error) {
	if r.repository == nil {
		return moduleapi.ComposeRuntimeTargetSummary{}, store.ErrNotFound
	}
	if id != nil {
		return r.readComposeTargetByID(ctx, *id)
	}
	return r.readFirstComposeTarget(ctx)
}

func (r runtimeTargetReader) readComposeTargetByID(ctx context.Context, id int64) (moduleapi.ComposeRuntimeTargetSummary, error) {
	if id < 1 {
		return moduleapi.ComposeRuntimeTargetSummary{}, store.ErrNotFound
	}
	target, err := r.repository.Get(ctx, uint64(id))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return moduleapi.ComposeRuntimeTargetSummary{}, store.ErrNotFound
		}
		return moduleapi.ComposeRuntimeTargetSummary{}, err
	}
	summary, ok := composeTargetSummary(target)
	if !ok {
		return moduleapi.ComposeRuntimeTargetSummary{}, store.ErrNotFound
	}
	return summary, nil
}

func (r runtimeTargetReader) readFirstComposeTarget(ctx context.Context) (moduleapi.ComposeRuntimeTargetSummary, error) {
	items, err := r.repository.List(ctx)
	if err != nil {
		return moduleapi.ComposeRuntimeTargetSummary{}, err
	}
	for _, target := range items {
		if summary, ok := composeTargetSummary(target); ok {
			return summary, nil
		}
	}
	return moduleapi.ComposeRuntimeTargetSummary{}, store.ErrNotFound
}

// ListComposeTargets 仅暴露同时支持 Compose 执行和托管工作区访问的目标。
func (r runtimeTargetReader) ListComposeTargets(ctx context.Context) ([]moduleapi.ComposeRuntimeTargetSummary, error) {
	if r.repository == nil {
		return []moduleapi.ComposeRuntimeTargetSummary{}, nil
	}
	items, err := r.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]moduleapi.ComposeRuntimeTargetSummary, 0, len(items))
	for _, target := range items {
		if summary, ok := composeTargetSummary(target); ok {
			results = append(results, summary)
		}
	}
	return results, nil
}

// composeTargetSummary 校验与 provider 无关的能力契约，同时保持
// composeTargetSummary 将满足 Compose 能力要求的运行时目标转换为摘要；目标 ID 超出支持范围或缺少必要能力时返回 false。
// 返回摘要及其是否转换成功。
func composeTargetSummary(target store.Target) (moduleapi.ComposeRuntimeTargetSummary, bool) {
	if target.ID > maxRuntimeTargetID || !hasComposeCapabilities(target.Capabilities) {
		return moduleapi.ComposeRuntimeTargetSummary{}, false
	}
	return moduleapi.ComposeRuntimeTargetSummary{ID: int64(target.ID), DisplayName: target.DisplayName, Provider: target.Provider, Capabilities: append([]string(nil), target.Capabilities...), Available: target.Availability}, true
}

// hasComposeCapabilities 判断能力集合是否同时包含 Compose 执行和工作区访问；两项能力均存在时返回 true。
func hasComposeCapabilities(capabilities []string) bool {
	seen := make(map[string]bool, len(capabilities))
	for _, capability := range capabilities {
		seen[capability] = true
	}
	return seen["compose_execution"] && seen["workspace_access"]
}

// buildTargetSummary 将具备 Docker 构建能力的目标转换为 provider-neutral 摘要。
func buildTargetSummary(target store.Target) (moduleapi.BuildRuntimeTargetSummary, bool) {
	if target.ID > maxRuntimeTargetID || !target.Availability || !hasBuildCapability(target.Capabilities) {
		return moduleapi.BuildRuntimeTargetSummary{}, false
	}
	switch target.Provider {
	case "docker":
		delivery := moduleapi.SnapshotDeliveryModeTargetLocal
		localities := []string{"target-local", "build-snapshot"}
		if target.ConnectionKind != "unix_socket" {
			delivery = moduleapi.SnapshotDeliveryModeProviderTransfer
			localities = []string{"build-snapshot"}
		}
		return moduleapi.BuildRuntimeTargetSummary{
			ID:                        int64(target.ID),
			DisplayName:               target.DisplayName,
			Provider:                  target.Provider,
			Available:                 true,
			ProviderCapabilityProfile: "oci-build",
			ProviderCapabilityVersion: "docker/v1",
			SupportedDrivers:          []string{"docker-engine"},
			SupportedPlatforms:        []string{"linux/amd64"},
			WorkspaceLocalities:       localities,
			SnapshotDeliveryModes:     []string{delivery},
			BuildFeatures:             []string{"registry-login"},
		}, true
	default:
		return moduleapi.BuildRuntimeTargetSummary{}, false
	}
}

// hasBuildCapability 判断运行目标是否显式声明镜像构建能力。
func hasBuildCapability(capabilities []string) bool {
	for _, capability := range capabilities {
		if capability == "image_build" {
			return true
		}
	}
	return false
}

// Boot 记录当前可用的本机 Docker 端点，但不让应用启动依赖 Docker 可用。
func (m *Module) Boot(ctx *module.Context) error {
	if m == nil || m.repository == nil || ctx == nil {
		return nil
	}
	if err := discoverLocalDocker(ctx.LifecycleContext, m.repository); err != nil {
		return err
	}
	if m.collector == nil {
		return nil
	}
	return m.collector.Start(ctx.LifecycleContext)
}

// Shutdown 释放模块拥有的实时采集器。
func (m *Module) Shutdown(ctx *module.Context) error {
	if m == nil || m.collector == nil {
		return nil
	}
	if ctx == nil || ctx.LifecycleContext == nil {
		return errors.New("runtime target shutdown lifecycle context is required")
	}
	return m.collector.Stop(ctx.LifecycleContext)
}

func (m *Module) handleList(c *gin.Context) {
	limit, offset, ok := runtimeTargetListWindow(c, m.i18n)
	if !ok {
		return
	}
	query := store.ListQuery{Limit: limit, Offset: offset, Keyword: c.Query("keyword"), Provider: c.Query("provider"), ConnectionKind: c.Query("connection_kind"), Health: c.Query("health"), Sort: c.Query("sort")}
	if !validRuntimeTargetListQuery(query) {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	page, err := m.repository.ListQueryPage(c.Request.Context(), query)
	if err != nil {
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
		return
	}
	mapped := mapRuntimeTargetSummaries(c.Request.Context(), page.Items, runtimeTargetListSummaryConcurrency, func(ctx context.Context, item store.Target) generated.RuntimeTargetSummary {
		return m.toHTTPSummary(ctx, item)
	})
	httpx.WriteSuccess(c, http.StatusOK, generated.RuntimeTargetListResponse{
		Items:  mapped,
		Total:  page.Total,
		Limit:  limit,
		Offset: offset,
		Summary: struct {
			Healthy     int64 `json:"healthy"`
			Total       int64 `json:"total"`
			Unavailable int64 `json:"unavailable"`
		}{Total: page.Summary.Total, Healthy: page.Summary.Healthy, Unavailable: page.Summary.Unavailable},
	})
}

//nolint:cyclop // 每个白名单字段的拒绝条件保持独立，避免隐式解析规则。
func validRuntimeTargetListQuery(query store.ListQuery) bool {
	if len(strings.TrimSpace(query.Keyword)) > runtimeTargetListKeywordMaxLength {
		return false
	}
	if query.Provider != "" && query.Provider != "docker" {
		return false
	}
	if query.ConnectionKind != "" && query.ConnectionKind != "unix_socket" {
		return false
	}
	if query.Health != "" && query.Health != "healthy" && query.Health != "unavailable" {
		return false
	}
	switch query.Sort {
	case "", "display_name:asc", "display_name:desc", "provider:asc", "provider:desc", "health:asc", "health:desc":
		return true
	default:
		return false
	}
}

func (m *Module) savedViewOwner(c *gin.Context) (uint64, bool) { return httpx.SavedViewOwnerID(c) }
func (m *Module) handleSavedViewList(c *gin.Context) {
	owner, ok := m.savedViewOwner(c)
	if !ok || m.savedViews == nil {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	views, err := m.savedViews.List(c.Request.Context(), owner, runtimeTargetSavedViewSurface)
	if err != nil {
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
		return
	}
	items := make([]generated.SavedView, 0, len(views))
	for _, view := range views {
		item, mapErr := runtimeTargetSavedViewResponse(view)
		if mapErr != nil {
			httpx.AbortLocalizedError(c, m.i18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
			return
		}
		items = append(items, item)
	}
	httpx.WriteSuccess(c, http.StatusOK, generated.SavedViewListResponse{Items: items})
}
func (m *Module) handleSavedViewCreate(c *gin.Context) {
	owner, ok := m.savedViewOwner(c)
	if !ok {
		m.writeSavedViewInvalid(c)
		return
	}
	var body generated.PostRuntimeTargetSavedViewJSONRequestBody
	if c.ShouldBindJSON(&body) != nil {
		m.writeSavedViewInvalid(c)
		return
	}
	input, err := parseRuntimeTargetSavedViewInput(body)
	if err != nil {
		m.writeSavedViewInvalid(c)
		return
	}
	view, err := m.savedViews.Create(c.Request.Context(), moduleapi.SavedViewCreateInput{OwnerUserID: owner, SurfaceKey: runtimeTargetSavedViewSurface, Name: input.Name, QueryState: input.QueryState, PageSize: input.PageSize, VisibleColumns: input.VisibleColumns, IsDefault: input.IsDefault})
	if err != nil {
		m.writeSavedViewError(c, err)
		return
	}
	result, err := runtimeTargetSavedViewResponse(view)
	if err != nil {
		m.writeSavedViewInvalid(c)
		return
	}
	httpx.WriteSuccess(c, http.StatusCreated, result)
}
func (m *Module) handleSavedViewUpdate(c *gin.Context) {
	owner, ok := m.savedViewOwner(c)
	id, validID := httpx.SavedViewID(c)
	if !ok || !validID {
		m.writeSavedViewInvalid(c)
		return
	}
	var body generated.PutRuntimeTargetSavedViewJSONRequestBody
	if c.ShouldBindJSON(&body) != nil {
		m.writeSavedViewInvalid(c)
		return
	}
	input, err := parseRuntimeTargetSavedViewInput(body)
	if err != nil {
		m.writeSavedViewInvalid(c)
		return
	}
	view, err := m.savedViews.Update(c.Request.Context(), moduleapi.SavedViewUpdateInput{ID: id, OwnerUserID: owner, SurfaceKey: runtimeTargetSavedViewSurface, Name: input.Name, QueryState: input.QueryState, PageSize: input.PageSize, VisibleColumns: input.VisibleColumns, IsDefault: input.IsDefault})
	if err != nil {
		m.writeSavedViewError(c, err)
		return
	}
	result, err := runtimeTargetSavedViewResponse(view)
	if err != nil {
		m.writeSavedViewInvalid(c)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, result)
}
func (m *Module) handleSavedViewDelete(c *gin.Context) {
	owner, ok := m.savedViewOwner(c)
	id, validID := httpx.SavedViewID(c)
	if !ok || !validID {
		m.writeSavedViewInvalid(c)
		return
	}
	if err := m.savedViews.Delete(c.Request.Context(), owner, runtimeTargetSavedViewSurface, id); err != nil {
		m.writeSavedViewError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (m *Module) writeSavedViewInvalid(c *gin.Context) {
	httpx.AbortLocalizedError(c, m.i18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
}
func (m *Module) writeSavedViewError(c *gin.Context, err error) {
	if errors.Is(err, moduleapi.ErrSavedViewNotFound) {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusNotFound, "common.not_found", nil)
		return
	}
	if errors.Is(err, moduleapi.ErrSavedViewConflict) {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusConflict, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
}

func mapRuntimeTargetSummaries(ctx context.Context, items []store.Target, concurrency int, mapItem func(context.Context, store.Target) generated.RuntimeTargetSummary) []generated.RuntimeTargetSummary {
	if len(items) == 0 {
		return []generated.RuntimeTargetSummary{}
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(items) {
		concurrency = len(items)
	}
	mapped := make([]generated.RuntimeTargetSummary, len(items))
	jobs := make(chan int)
	var workers sync.WaitGroup
	workers.Add(concurrency)
	for worker := 0; worker < concurrency; worker++ {
		go func() {
			defer workers.Done()
			for index := range jobs {
				itemCtx, cancel := context.WithTimeout(ctx, runtimeTargetSummaryTimeout)
				mapped[index] = mapItem(itemCtx, items[index])
				cancel()
			}
		}()
	}
	for index := range items {
		jobs <- index
	}
	close(jobs)
	workers.Wait()
	return mapped
}

func (m *Module) handleDetail(c *gin.Context) {
	target, ok := m.readTarget(c)
	if !ok {
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, m.toHTTP(c.Request.Context(), target))
}

func (m *Module) handleRefresh(moduleCtx *module.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		target, ok := m.readTarget(c)
		if !ok {
			return
		}
		refreshed, err := m.runRefreshAuditTransaction(c.Request.Context(), func(txCtx context.Context) (store.Target, error) {
			return refreshTarget(txCtx, m.repository, target.ID)
		})
		if err != nil {
			httpx.AbortAppError(c, moduleCtx.I18n, m.runtimeLogger, err)
			return
		}
		m.summaries.invalidate(target.ID)
		httpx.WriteSuccess(c, http.StatusOK, m.toHTTP(c.Request.Context(), refreshed))
	}
}

func (m *Module) handleDiscoverLocal(moduleCtx *module.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		target, found, err := m.discoverAndPublishAudit(c.Request.Context())
		if err != nil {
			httpx.AbortAppError(c, moduleCtx.I18n, m.runtimeLogger, err)
			return
		}
		if !found {
			httpx.WriteSuccess[any](c, http.StatusOK, nil)
			return
		}
		m.summaries.invalidate(target.ID)
		httpx.WriteSuccess(c, http.StatusOK, m.toHTTP(c.Request.Context(), target))
	}
}

// runtimeTargetListWindow 解析并校验请求中的分页参数；参数无效时直接返回 bad-request 响应。
func runtimeTargetListWindow(c *gin.Context, localizer *i18n.Service) (int, int, bool) {
	limit := 10
	offset := 0
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || (parsed != 10 && parsed != 20 && parsed != 50 && parsed != 100) {
			httpx.AbortLocalizedError(c, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
			return 0, 0, false
		}
		limit = parsed
	}
	if raw := c.Query("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			httpx.AbortLocalizedError(c, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
			return 0, 0, false
		}
		offset = parsed
	}
	return limit, offset, true
}

func (m *Module) readTarget(c *gin.Context) (store.Target, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return store.Target{}, false
	}
	target, err := m.repository.Get(c.Request.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		httpx.AbortLocalizedError(c, m.i18n, http.StatusNotFound, "common.not_found", nil)
		return store.Target{}, false
	}
	if err != nil {
		httpx.AbortAppError(c, m.i18n, m.runtimeLogger, err)
		return store.Target{}, false
	}
	return target, true
}

func (m *Module) snapshot(ctx context.Context, target store.Target) targetRuntimeSummary {
	if m == nil || m.summaries == nil {
		return unavailableTargetSummary("Runtime target is unavailable", time.Now().UTC())
	}
	return m.summaries.get(ctx, target)
}

func (m *Module) toHTTPSummary(ctx context.Context, target store.Target) generated.RuntimeTargetSummary {
	if target.ID > maxRuntimeTargetID {
		return generated.RuntimeTargetSummary{}
	}
	snapshot := m.snapshot(ctx, target)
	checkedAt := snapshot.CheckedAt
	response := generated.RuntimeTargetSummary{Id: int64(target.ID), DisplayName: target.DisplayName}
	response.Runtime.Provider = generated.RuntimeTargetSummaryRuntimeProviderDocker
	response.Runtime.Type = generated.RuntimeTargetSummaryRuntimeTypeContainerRuntime
	response.Runtime.Version = snapshot.Version
	response.Runtime.ApiVersion = snapshot.APIVersion
	response.Runtime.OperatingSystem = snapshot.OperatingSystem
	response.Runtime.HostName = snapshot.HostName
	response.Connection.Endpoint = target.EndpointLabel
	response.Connection.Kind = generated.RuntimeTargetSummaryConnectionKindUnixSocket
	response.Health.LastCheckedAt = &checkedAt
	response.Health.Diagnostic = snapshot.Diagnostic
	response.Health.Status = generated.RuntimeTargetSummaryHealthStatusUnavailable
	if snapshot.Healthy {
		response.Health.Status = generated.RuntimeTargetSummaryHealthStatusHealthy
	}
	response.Resources.Workloads = toHTTPCountMetric(snapshot.Workloads)
	response.Resources.Cpu = toHTTPUsageMetric(snapshot.CPU)
	response.Resources.Memory = toHTTPUsageMetric(snapshot.Memory)
	response.Resources.Storage = toHTTPUsageMetric(snapshot.Disk)
	return response
}

func (m *Module) toHTTP(ctx context.Context, target store.Target) generated.RuntimeTarget {
	if target.ID > maxRuntimeTargetID {
		return generated.RuntimeTarget{}
	}
	snapshot := m.snapshot(ctx, target)
	checkedAt := snapshot.CheckedAt
	response := generated.RuntimeTarget{Id: int64(target.ID), DisplayName: target.DisplayName}
	response.Runtime.Provider = generated.RuntimeTargetRuntimeProviderDocker
	response.Runtime.Type = generated.RuntimeTargetRuntimeTypeContainerRuntime
	response.Runtime.Version = snapshot.Version
	response.Runtime.ApiVersion = snapshot.APIVersion
	response.Runtime.OperatingSystem = snapshot.OperatingSystem
	response.Runtime.HostName = snapshot.HostName
	response.Connection.Endpoint = target.EndpointLabel
	response.Connection.Kind = generated.RuntimeTargetConnectionKindUnixSocket
	response.Health.LastCheckedAt = &checkedAt
	response.Health.Diagnostic = snapshot.Diagnostic
	response.Health.Status = generated.RuntimeTargetHealthStatusUnavailable
	if snapshot.Healthy {
		response.Health.Status = generated.RuntimeTargetHealthStatusHealthy
	}
	response.Resources.Workloads = toHTTPCountMetric(snapshot.Workloads)
	response.Resources.Cpu = toHTTPUsageMetric(snapshot.CPU)
	response.Resources.Memory = toHTTPUsageMetric(snapshot.Memory)
	response.Resources.Storage = toHTTPUsageMetric(snapshot.Disk)
	response.ProviderDetails.Provider = generated.RuntimeTargetProviderDetailsProviderDocker
	response.ProviderDetails.Docker.Images = toHTTPProviderCountMetric(snapshot.Images)
	response.ProviderDetails.Docker.Volumes = toHTTPProviderCountMetric(snapshot.Volumes)
	response.ProviderDetails.Docker.Networks = toHTTPProviderCountMetric(snapshot.Networks)
	return response
}

// toHTTPCountMetric 将目标计数指标转换为 HTTP 响应中的运行时目标计数指标。
func toHTTPCountMetric(metric targetCountMetric) generated.RuntimeTargetCountMetric {
	return generated.RuntimeTargetCountMetric{Available: metric.Available, Total: metric.Total, Active: metric.Active, UnavailableReason: metric.UnavailableReason}
}

// toHTTPProviderCountMetric 将镜像指标转换为 HTTP 响应表示。
func toHTTPProviderCountMetric(metric targetImageMetric) generated.RuntimeTargetImageMetric {
	return generated.RuntimeTargetImageMetric{Available: metric.Available, Total: metric.Total, UnavailableReason: metric.UnavailableReason}
}

// toHTTPUsageMetric 将运行时使用量指标转换为 HTTP 响应中的使用量指标。
func toHTTPUsageMetric(metric targetUsageMetric) generated.RuntimeTargetUsageMetric {
	return generated.RuntimeTargetUsageMetric{Available: metric.Available, UsedBytes: metric.UsedBytes, TotalBytes: metric.TotalBytes, UsagePercent: metric.UsagePercent, UnavailableReason: metric.UnavailableReason}
}

func (m *Module) discoverAndPublishAudit(ctx context.Context) (store.Target, bool, error) {
	target, err := m.runRefreshAuditTransaction(ctx, func(txCtx context.Context) (store.Target, error) {
		if err := discoverLocalDocker(txCtx, m.repository); err != nil {
			return store.Target{}, err
		}
		current, err := m.repository.FindSystemLocalDocker(txCtx)
		if errors.Is(err, store.ErrNotFound) {
			return store.Target{}, nil
		}
		return current, err
	})
	return target, target.ID != 0, err
}

// runRefreshAuditTransaction 将 runtime-target 刷新事实与 durable audit event 固定在同一个事务中。
func (m *Module) runRefreshAuditTransaction(ctx context.Context, refresh func(context.Context) (store.Target, error)) (store.Target, error) {
	if m == nil || m.repository == nil {
		return store.Target{}, errors.New("runtime target repository is unavailable")
	}
	if m.events == nil {
		return store.Target{}, errors.New("runtime target event transaction publisher is unavailable")
	}
	var refreshed store.Target
	err := m.repository.RunInTransaction(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		target, err := refresh(txCtx)
		if err != nil {
			return err
		}
		if target.ID == 0 {
			return nil
		}
		if err := m.publishRefreshAuditTx(txCtx, tx, target); err != nil {
			return err
		}
		refreshed = target
		return nil
	})
	if err != nil {
		return store.Target{}, err
	}
	return refreshed, nil
}

func (m *Module) publishRefreshAuditTx(ctx context.Context, tx *sql.Tx, target store.Target) error {
	payload := moduleapi.AuditEvent{Kind: moduleapi.AuditEventKindDomain, Action: "runtime_target.refresh", ResourceType: "runtime_target", ResourceID: strconv.FormatUint(target.ID, 10), ResourceName: strings.TrimSpace(target.DisplayName), StatusCode: http.StatusOK, Success: true, Metadata: map[string]any{"provider": target.Provider, "result": "success"}}
	envelope, err := httpx.NewAuditEvent(moduleID, payload)
	if err != nil {
		return err
	}
	_, err = m.events.PublishTx(ctx, tx, envelope, event.PublishOptions{Delivery: event.DeliveryDurable})
	return err
}
