package auth

import "graft/server/internal/plugin"

// Plugin 是 auth 插件的 Phase 1 生命周期骨架。
//
// 本阶段只声明稳定插件身份与未来依赖顺序，不引入 token/session/cookie
// 或 `/auth/*` 运行时行为，避免提前把 Phase 2/3 混入当前切片。
type Plugin struct{}

// NewPlugin 创建 auth 插件最小骨架实例。
func NewPlugin() *Plugin {
	return &Plugin{}
}

// Name 返回插件稳定标识。
func (p *Plugin) Name() string {
	return pluginID
}

// Version 返回当前插件版本。
func (p *Plugin) Version() string {
	return pluginVersion
}

// DependsOn 返回当前插件依赖列表。
func (p *Plugin) DependsOn() []string {
	return []string{"user"}
}

// Register 当前阶段不注册运行时能力；后续 Phase 2/3 再显式接入 auth capability 与路由。
func (p *Plugin) Register(_ *plugin.Context) error {
	return nil
}

// Boot 当前没有额外运行时行为需要启动。
func (p *Plugin) Boot(_ *plugin.Context) error {
	return nil
}

// Shutdown 当前没有额外资源需要释放。
func (p *Plugin) Shutdown(_ *plugin.Context) error {
	return nil
}
