package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/moduleapi"
)

var (
	errDiagnosticTargetNotFound  = errors.New("outbound diagnostic target not found")
	errDiagnosticExecutionFailed = errors.New("outbound diagnostic execution failed")
)

// Service 将 module-managed 配置存储适配为 Network 模块的策略和诊断读取模型。
type Service struct {
	configs       moduleapi.ModuleConfigManager
	diagnostics   moduleapi.OutboundDiagnosticRegistry
	consumers     moduleapi.OutboundNetworkConsumerRegistry
	history       DiagnosticHistoryStore
	connectivity  ConnectivityStore
	customTargets CustomConnectivityTargetStore
	logger        *zap.Logger
}

// NewService 创建 Network 模块服务。
func NewService(configs moduleapi.ModuleConfigManager, diagnostics moduleapi.OutboundDiagnosticRegistry, consumers moduleapi.OutboundNetworkConsumerRegistry, history DiagnosticHistoryStore, logger *zap.Logger) *Service {
	return &Service{configs: configs, diagnostics: diagnostics, consumers: consumers, history: history, logger: logger}
}

// RunConnectivity 执行已注册 target 并持久化净化后的不可变报告。
//
//nolint:gocyclo,cyclop // 自定义目标策略、报告身份和持久化是必须按序可审计的安全边界。
func (s *Service) RunConnectivity(ctx context.Context, targetID moduleapi.ConnectivityTargetID) (ConnectivityCheck, moduleapi.ConnectivityReport, error) {
	if s == nil || s.diagnostics == nil || s.connectivity == nil {
		return ConnectivityCheck{}, moduleapi.ConnectivityReport{}, errors.New("connectivity service is unavailable")
	}
	target, isCustom, targetErr := s.connectivityTarget(ctx, targetID)
	if targetErr != nil {
		return ConnectivityCheck{}, moduleapi.ConnectivityReport{}, targetErr
	}
	if isCustom {
		policy, policyErr := s.currentPolicy(ctx)
		if policyErr != nil {
			return ConnectivityCheck{}, moduleapi.ConnectivityReport{}, policyErr
		}
		if policy.Enabled && (policy.HTTPProxy != "" || policy.HTTPSProxy != "") {
			return ConnectivityCheck{}, moduleapi.ConnectivityReport{}, errors.New("custom connectivity targets cannot run while an outbound proxy is enabled")
		}
	}
	report, err := target.RunConnectivityProbes(ctx)
	if err != nil {
		return ConnectivityCheck{}, moduleapi.ConnectivityReport{}, fmt.Errorf("run connectivity probes: %w", err)
	}
	if report.TargetID != targetID {
		return ConnectivityCheck{}, moduleapi.ConnectivityReport{}, errors.New("connectivity target returned mismatched report")
	}
	if report.Route == nil {
		policy, policyErr := s.currentPolicy(ctx)
		if policyErr != nil {
			return ConnectivityCheck{}, moduleapi.ConnectivityReport{}, policyErr
		}
		host := ""
		if destination, ok := target.(moduleapi.ConnectivityRouteDestination); ok {
			host = destination.ConnectivityRouteHost()
		}
		report = withRouteExplanation(report, policy, host)
	}
	check, err := s.connectivity.Append(ctx, report)
	if err != nil {
		return ConnectivityCheck{}, moduleapi.ConnectivityReport{}, err
	}
	return check, report.Snapshot(), nil
}

// RunAllConnectivity 经由相同的单目标流水线执行当前已注册且已启用的目标集合。
// 它有意由 registry 和自定义目标管理边界限制，不接受调用方提供的无界 ID 列表。
//
//nolint:cyclop // 内建与自定义目标的受限聚合、失败审计及持久化调用必须保持单一可审计的顺序。
func (s *Service) RunAllConnectivity(ctx context.Context) ([]ConnectivityCheck, error) {
	if s == nil || s.diagnostics == nil {
		return nil, errors.New("connectivity service is unavailable")
	}
	ids := make([]moduleapi.ConnectivityTargetID, 0)
	for _, target := range s.diagnostics.ConnectivityTargetDescriptors() {
		ids = append(ids, target.ID)
	}
	if s.customTargets != nil {
		customTargets, err := s.customTargets.ListCustomTargets(ctx)
		if err != nil {
			return nil, err
		}
		for _, target := range customTargets {
			if target.Enabled {
				ids = append(ids, target.ID)
			}
		}
	}
	checks := make([]ConnectivityCheck, 0, len(ids))
	for _, targetID := range ids {
		check, _, err := s.RunConnectivity(ctx, targetID)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn("connectivity target failed", zap.String("target_id", string(targetID)), zap.Error(err))
			}
			continue
		}
		checks = append(checks, check)
	}
	return checks, nil
}

// ConnectivityTargets 返回 registry 的稳定描述快照，供批量健康检查和 target 诊断共用。
func (s *Service) ConnectivityTargets(ctx context.Context) ([]moduleapi.ConnectivityTargetDescriptor, error) {
	if s == nil || s.diagnostics == nil {
		return nil, errors.New("connectivity service is unavailable")
	}
	targets := s.diagnostics.ConnectivityTargetDescriptors()
	if len(targets) >= maxConnectivityTargetListSize {
		return targets[:maxConnectivityTargetListSize], nil
	}
	if s.customTargets == nil {
		return targets, nil
	}
	customTargets, err := s.customTargets.ListCustomTargets(ctx)
	if err != nil {
		return nil, err
	}
	for _, target := range customTargets {
		if target.Enabled {
			targets = append(targets, target.ConnectivityTargetDescriptor().Snapshot())
			if len(targets) == maxConnectivityTargetListSize {
				break
			}
		}
	}
	return targets, nil
}

// ConnectivityLatest 返回每个已执行目标的最新轻量检查投影。
func (s *Service) ConnectivityLatest(ctx context.Context) ([]ConnectivityCheck, error) {
	if s == nil || s.connectivity == nil {
		return nil, errors.New("connectivity service is unavailable")
	}
	return s.connectivity.Latest(ctx)
}

// ConnectivityAggregate 返回由最新检查计算的健康摘要，不另建缓存真相。
func (s *Service) ConnectivityAggregate(ctx context.Context) (ConnectivityAggregate, error) {
	if s == nil || s.connectivity == nil {
		return ConnectivityAggregate{}, errors.New("connectivity service is unavailable")
	}
	return s.connectivity.Aggregate(ctx)
}

// ConnectivityHistory 返回注册 target 的有界历史摘要。
func (s *Service) ConnectivityHistory(ctx context.Context, targetID moduleapi.ConnectivityTargetID, limit int) ([]ConnectivityCheck, error) {
	if s == nil || s.diagnostics == nil || s.connectivity == nil {
		return nil, errors.New("connectivity service is unavailable")
	}
	if _, _, err := s.connectivityTarget(ctx, targetID); err != nil {
		return nil, err
	}
	return s.connectivity.History(ctx, targetID, limit)
}

// ConnectivityReport 读取 target 自己的一次报告，避免报告 ID 成为可横向枚举的页面身份。
func (s *Service) ConnectivityReport(ctx context.Context, targetID moduleapi.ConnectivityTargetID, checkID int64) (moduleapi.ConnectivityReport, error) {
	if s == nil || s.diagnostics == nil || s.connectivity == nil {
		return moduleapi.ConnectivityReport{}, errors.New("connectivity service is unavailable")
	}
	if _, _, err := s.connectivityTarget(ctx, targetID); err != nil {
		return moduleapi.ConnectivityReport{}, err
	}
	return s.connectivity.Report(ctx, targetID, checkID)
}

// CreateCustomConnectivityTarget 在自定义目标进入共享流水线前完成校验和持久化。
func (s *Service) CreateCustomConnectivityTarget(ctx context.Context, input CustomConnectivityTargetInput, actorID uint64) (CustomConnectivityTarget, error) {
	if s == nil || s.customTargets == nil {
		return CustomConnectivityTarget{}, errors.New("custom connectivity targets are unavailable")
	}
	target, err := validateCustomConnectivityTargetInput(input)
	if err != nil {
		return CustomConnectivityTarget{}, err
	}
	if _, ok := s.diagnostics.ConnectivityTarget(target.ID); ok {
		return CustomConnectivityTarget{}, errors.New("custom connectivity target ID conflicts with a registered target")
	}
	return s.customTargets.CreateCustomTarget(ctx, target, actorID)
}

// CustomConnectivityTargets 返回管理所需的全部有效自定义目标元数据。
func (s *Service) CustomConnectivityTargets(ctx context.Context) ([]CustomConnectivityTarget, error) {
	if s == nil || s.customTargets == nil {
		return nil, errors.New("custom connectivity targets are unavailable")
	}
	return s.customTargets.ListCustomTargets(ctx)
}

// DeleteCustomConnectivityTarget 将自定义目标移出未来批量和单目标执行。
func (s *Service) DeleteCustomConnectivityTarget(ctx context.Context, targetID moduleapi.ConnectivityTargetID, actorID uint64) error {
	if s == nil || s.customTargets == nil {
		return errors.New("custom connectivity targets are unavailable")
	}
	return s.customTargets.DeleteCustomTarget(ctx, targetID, actorID)
}

func (s *Service) connectivityTarget(ctx context.Context, targetID moduleapi.ConnectivityTargetID) (moduleapi.ConnectivityTarget, bool, error) {
	if target, ok := s.diagnostics.ConnectivityTarget(targetID); ok {
		return target, false, nil
	}
	if s.customTargets == nil {
		return nil, false, errDiagnosticTargetNotFound
	}
	target, err := s.customTargets.CustomTarget(ctx, targetID)
	if errors.Is(err, errCustomConnectivityTargetNotFound) {
		return nil, false, errDiagnosticTargetNotFound
	}
	if err != nil {
		return nil, false, err
	}
	if !target.Enabled {
		return nil, false, errDiagnosticTargetNotFound
	}
	return target, true, nil
}

func (s *Service) currentPolicy(ctx context.Context) (moduleapi.OutboundNetworkPolicy, error) {
	if s == nil || s.configs == nil {
		return moduleapi.OutboundNetworkPolicy{}, errors.New("platform network service is unavailable")
	}
	value, err := s.configs.GetModuleConfig(ctx, moduleID, outboundConfigKey)
	if err != nil {
		return moduleapi.OutboundNetworkPolicy{}, fmt.Errorf("read outbound network config: %w", err)
	}
	return decodeOutboundPolicy(value.EffectiveValue)
}

// Overview 读取当前有效策略及可执行的固定诊断目标。
func (s *Service) Overview(ctx context.Context) (Overview, error) {
	if s == nil || s.configs == nil || s.diagnostics == nil || s.consumers == nil {
		return Overview{}, errors.New("platform network service is unavailable")
	}
	value, err := s.configs.GetModuleConfig(ctx, moduleID, outboundConfigKey)
	if err != nil {
		return Overview{}, fmt.Errorf("read outbound network config: %w", err)
	}
	policy, err := decodeOutboundPolicy(value.EffectiveValue)
	if err != nil {
		return Overview{}, err
	}
	return Overview{Policy: policy, HasOverride: value.HasOverride, Version: value.Version, UpdatedAt: value.UpdatedAt, UpdatedByName: value.UpdatedByName, DiagnosticTargets: s.diagnostics.OutboundDiagnosticTargets(), Consumers: s.consumers.OutboundNetworkConsumers()}, nil
}

// Update 保存经过 Network 领域校验的出站策略；该策略只包含代理选择，不包含 HTTP client 行为。
func (s *Service) Update(ctx context.Context, input moduleapi.OutboundNetworkPolicy, userID *uint64, expectedVersion int64) (Overview, error) {
	if s == nil || s.configs == nil {
		return Overview{}, errors.New("platform network service is unavailable")
	}
	raw, err := json.Marshal(outboundConfig{Enabled: input.Enabled, HTTPProxy: input.HTTPProxy, HTTPSProxy: input.HTTPSProxy, NoProxy: input.NoProxy})
	if err != nil {
		return Overview{}, fmt.Errorf("encode outbound network config: %w", err)
	}
	if _, err := decodeOutboundPolicy(raw); err != nil {
		return Overview{}, err
	}
	if _, err := s.configs.UpdateModuleConfig(ctx, moduleID, outboundConfigKey, raw, userID, expectedVersion); err != nil {
		return Overview{}, fmt.Errorf("update outbound network config: %w", err)
	}
	return s.Overview(ctx)
}

// Reset 恢复模块默认的直接连接策略。
func (s *Service) Reset(ctx context.Context, userID *uint64, expectedVersion int64) (Overview, error) {
	if s == nil || s.configs == nil {
		return Overview{}, errors.New("platform network service is unavailable")
	}
	if _, err := s.configs.ResetModuleConfig(ctx, moduleID, outboundConfigKey, userID, expectedVersion); err != nil {
		return Overview{}, fmt.Errorf("reset outbound network config: %w", err)
	}
	return s.Overview(ctx)
}

// Diagnose 执行名称精确匹配的固定注册目标，不接受或解析用户提供的网络位置。
func (s *Service) Diagnose(ctx context.Context, targetName string) (moduleapi.OutboundDiagnosticResult, error) {
	if s == nil || s.diagnostics == nil || s.history == nil {
		return moduleapi.OutboundDiagnosticResult{}, errors.New("platform network service is unavailable")
	}
	targetName = strings.TrimSpace(targetName)
	target, ok := s.diagnostics.OutboundDiagnosticTarget(targetName)
	if !ok {
		return moduleapi.OutboundDiagnosticResult{}, errDiagnosticTargetNotFound
	}
	result, err := target.ExecuteOutboundDiagnostic(ctx)
	if err != nil {
		s.logDiagnosticFailure("outbound diagnostic execution failed", target.Name(), err)
		result = moduleapi.OutboundDiagnosticResult{TestedAt: time.Now().UTC(), Message: errDiagnosticExecutionFailed.Error()}
	}
	if result.TestedAt.IsZero() {
		result.TestedAt = time.Now().UTC()
	}
	if err := s.history.Append(ctx, targetName, result); err != nil {
		s.logDiagnosticFailure("persist outbound diagnostic history failed", target.Name(), err)
		return result, nil
	}
	return result, nil
}

func (s *Service) logDiagnosticFailure(message string, targetName string, err error) {
	if s.logger == nil {
		return
	}
	s.logger.Error(message, zap.String("module", moduleID), zap.String("target_id", targetName), zap.Error(err))
}

// DiagnosticHistory 读取固定注册目标的有限诊断历史，拒绝未注册目标以维持诊断边界。
func (s *Service) DiagnosticHistory(ctx context.Context, targetName string, limit int) ([]moduleapi.OutboundDiagnosticResult, error) {
	if s == nil || s.diagnostics == nil || s.history == nil {
		return nil, errors.New("platform network service is unavailable")
	}
	if _, ok := s.diagnostics.OutboundDiagnosticTarget(strings.TrimSpace(targetName)); !ok {
		return nil, errDiagnosticTargetNotFound
	}
	return s.history.List(ctx, strings.TrimSpace(targetName), limit)
}

// Overview 是 Network 页面与路由使用的模块内投影。
type Overview struct {
	Policy            moduleapi.OutboundNetworkPolicy
	HasOverride       bool
	Version           int64
	UpdatedAt         *time.Time
	UpdatedByName     string
	DiagnosticTargets []moduleapi.OutboundDiagnosticTarget
	Consumers         []moduleapi.OutboundNetworkConsumer
}
