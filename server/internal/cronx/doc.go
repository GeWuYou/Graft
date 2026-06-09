// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package cronx 提供模块任务使用的调度注册表。
//
// 当前阶段由模块在 Register 阶段显式声明任务元数据与执行入口，再由独立
// scheduler 封装在 Boot 阶段接入底层调度实现。
package cronx
