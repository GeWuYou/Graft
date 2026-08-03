// Package network 提供平台主动 HTTP(S) 访问的统一策略与 client factory。
//
// 本模块只负责平台代理、绕过规则和固定诊断目标，不管理 Docker daemon、SMTP
// 或调用方的重试、超时等 HTTP client 行为。
package network
