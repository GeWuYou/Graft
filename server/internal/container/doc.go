// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package container 提供运行时使用的显式单例容器。
//
// 这个包只负责服务注册、单例解析与并发安全控制，不负责反射扫描、
// 隐式注入或隐藏装配过程。核心与模块代码仍应保持依赖构造可见。
package container
