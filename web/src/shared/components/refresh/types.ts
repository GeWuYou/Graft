// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export type RefreshControlValue = number | string;
export type RefreshControlStatus = 'running' | 'paused' | 'off';

export type RefreshControlOption = {
  label: string;
  value: RefreshControlValue;
};
