package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"graft/server/internal/moduleapi"
)

var errDiagnosticTargetNotFound = errors.New("outbound diagnostic target not found")

// Service 将 module-managed 配置存储适配为 Network 模块的策略和诊断读取模型。
type Service struct {
	configs     moduleapi.ModuleConfigManager
	diagnostics moduleapi.OutboundDiagnosticRegistry
}

// NewService 创建 Network 模块服务。
func NewService(configs moduleapi.ModuleConfigManager, diagnostics moduleapi.OutboundDiagnosticRegistry) *Service {
	return &Service{configs: configs, diagnostics: diagnostics}
}

// Overview 读取当前有效策略及可执行的固定诊断目标。
func (s *Service) Overview(ctx context.Context) (Overview, error) {
	if s == nil || s.configs == nil || s.diagnostics == nil {
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
	return Overview{Policy: policy, HasOverride: value.HasOverride, DiagnosticTargets: s.diagnostics.OutboundDiagnosticTargets()}, nil
}

// Update 保存经过 Network 领域校验的出站策略；该策略只包含代理选择，不包含 HTTP client 行为。
func (s *Service) Update(ctx context.Context, input moduleapi.OutboundNetworkPolicy, userID *uint64) (Overview, error) {
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
	if _, err := s.configs.UpdateModuleConfig(ctx, moduleID, outboundConfigKey, raw, userID); err != nil {
		return Overview{}, fmt.Errorf("update outbound network config: %w", err)
	}
	return s.Overview(ctx)
}

// Reset 恢复模块默认的直接连接策略。
func (s *Service) Reset(ctx context.Context) (Overview, error) {
	if s == nil || s.configs == nil {
		return Overview{}, errors.New("platform network service is unavailable")
	}
	if _, err := s.configs.ResetModuleConfig(ctx, moduleID, outboundConfigKey); err != nil {
		return Overview{}, fmt.Errorf("reset outbound network config: %w", err)
	}
	return s.Overview(ctx)
}

// Diagnose 执行名称精确匹配的固定注册目标，不接受或解析用户提供的网络位置。
func (s *Service) Diagnose(ctx context.Context, targetName string) (moduleapi.OutboundDiagnosticResult, error) {
	if s == nil || s.diagnostics == nil {
		return moduleapi.OutboundDiagnosticResult{}, errors.New("platform network service is unavailable")
	}
	target, ok := s.diagnostics.OutboundDiagnosticTarget(strings.TrimSpace(targetName))
	if !ok {
		return moduleapi.OutboundDiagnosticResult{}, errDiagnosticTargetNotFound
	}
	return target.ExecuteOutboundDiagnostic(ctx)
}

// Overview 是 Network 页面与路由使用的模块内投影。
type Overview struct {
	Policy            moduleapi.OutboundNetworkPolicy
	HasOverride       bool
	DiagnosticTargets []moduleapi.OutboundDiagnosticTarget
}
