// Package auth 定义认证与会话生命周期插件的长期边界。
//
// Phase 1 只建立最小插件骨架与 capability owner，不接管现有 user 插件里的
// token、refresh session、cookie 或 `/auth/*` 运行时实现。
package auth
