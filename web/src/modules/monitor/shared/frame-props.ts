// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { PageHeaderSource } from '@/shared/components/page';

import type { ServerStatusTone } from '../components/server-status-ui';
import type { RefreshIntervalOption } from '../composables/use-monitor-refresh-preferences';

export type MonitorSummaryItem = {
  key: string;
  label: string;
  value: string;
  description: string;
};

export type MonitorStatusPageFrameProps = {
  eyebrow: string;
  title: string;
  description: string;
  titleKey?: string;
  descriptionKey?: string;
  source?: PageHeaderSource;
  compactHeader?: boolean;
  autoRefreshEnabled: boolean;
  loading: boolean;
  pauseAutoRefreshLabel: string;
  refreshIntervalLabel: string;
  refreshIntervalOptions: RefreshIntervalOption[];
  refreshIntervalValue: number | string;
  refreshNowLabel: string;
  resumeAutoRefreshLabel: string;
  status: ServerStatusTone;
  statusLabel: string;
  trendRangeLabelPlaceholder: string;
  summaryItems: MonitorSummaryItem[];
  errorTitle: string;
  errorMessage?: string;
  initialized: boolean;
  hasServerStatus: boolean;
  emptyDescription: string;
};

type MonitorStatusFramePageCopy = Pick<
  MonitorStatusPageFrameProps,
  | 'eyebrow'
  | 'title'
  | 'description'
  | 'titleKey'
  | 'descriptionKey'
  | 'source'
  | 'compactHeader'
  | 'status'
  | 'statusLabel'
  | 'summaryItems'
  | 'emptyDescription'
>;

type MonitorStatusFrameSharedState = Pick<
  MonitorStatusPageFrameProps,
  | 'autoRefreshEnabled'
  | 'loading'
  | 'refreshIntervalOptions'
  | 'refreshIntervalValue'
  | 'errorMessage'
  | 'initialized'
  | 'hasServerStatus'
>;

type MonitorStatusFrameCommonLabels = Pick<
  MonitorStatusPageFrameProps,
  | 'pauseAutoRefreshLabel'
  | 'refreshIntervalLabel'
  | 'refreshNowLabel'
  | 'resumeAutoRefreshLabel'
  | 'trendRangeLabelPlaceholder'
  | 'errorTitle'
>;

function buildMonitorStatusFrameBaseProps(args: {
  page: MonitorStatusFramePageCopy;
  state: MonitorStatusFrameSharedState;
  labels: MonitorStatusFrameCommonLabels;
}): MonitorStatusPageFrameProps {
  return {
    ...args.page,
    ...args.state,
    ...args.labels,
  };
}

type MonitorStatusFrameSharedRefs = {
  autoRefreshEnabled: boolean;
  loading: boolean;
  refreshIntervalOptions: RefreshIntervalOption[];
  refreshIntervalValue: number | string;
  errorMessage?: string;
  initialized: boolean;
  hasServerStatus: boolean;
};

type MonitorTranslate = (key: string) => string;

export function buildStandardMonitorStatusFrameProps(args: {
  t: MonitorTranslate;
  page: Omit<MonitorStatusFramePageCopy, 'emptyDescription'>;
  snapshot: MonitorStatusFrameSharedRefs;
}): MonitorStatusPageFrameProps {
  return buildMonitorStatusFrameBaseProps({
    page: {
      ...args.page,
      emptyDescription: args.t('monitor.shared.empty'),
    },
    state: args.snapshot,
    labels: {
      pauseAutoRefreshLabel: args.t('monitor.serverStatus.pauseRefresh'),
      refreshIntervalLabel: args.t('monitor.serverStatus.refreshIntervalLabel'),
      refreshNowLabel: args.t('monitor.serverStatus.refreshNow'),
      resumeAutoRefreshLabel: args.t('monitor.serverStatus.resumeRefresh'),
      trendRangeLabelPlaceholder: args.t('monitor.serverStatus.trendWindowLabel'),
      errorTitle: args.t('monitor.shared.errorTitle'),
    },
  });
}
