// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export type TColorToken = Record<string, string>;
export type TColorSeries = Record<string, TColorToken>;

export const DEFAULT_CHART_COLORS = {
  textColor: 'var(--td-text-color-primary)',
  placeholderColor: 'var(--td-text-color-placeholder)',
  borderColor: 'var(--td-component-border)',
  containerColor: 'var(--td-bg-color-container)',
};

export type TChartColor = typeof DEFAULT_CHART_COLORS;

export const DEFAULT_COLOR_OPTIONS = [
  '#0052D9',
  '#0594FA',
  '#00A870',
  '#EBB105',
  '#ED7B2F',
  '#E34D59',
  '#ED49B4',
  '#834EC2',
];
