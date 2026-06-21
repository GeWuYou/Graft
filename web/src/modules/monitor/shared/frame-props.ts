import type { PageHeaderSource } from '@/shared/components/page';
import type { RefreshControlStatus } from '@/shared/components/refresh';

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
  refreshControlStatus: RefreshControlStatus;
  remainingRefreshSeconds: number | null;
  loading: boolean;
  refreshIntervalOptions: RefreshIntervalOption[];
  refreshIntervalValue: number | string;
  status: ServerStatusTone;
  statusLabel: string;
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
  | 'refreshControlStatus'
  | 'remainingRefreshSeconds'
  | 'loading'
  | 'refreshIntervalOptions'
  | 'refreshIntervalValue'
  | 'errorMessage'
  | 'initialized'
  | 'hasServerStatus'
>;

type MonitorStatusFrameCommonLabels = Pick<MonitorStatusPageFrameProps, 'errorTitle'>;

/**
 * Builds complete monitor status frame props from separate page, state, and label property groups.
 *
 * @returns A `MonitorStatusPageFrameProps` object with all combined properties.
 */
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
  refreshControlStatus: RefreshControlStatus;
  remainingRefreshSeconds: number | null;
  loading: boolean;
  refreshIntervalOptions: RefreshIntervalOption[];
  refreshIntervalValue: number | string;
  errorMessage?: string;
  initialized: boolean;
  hasServerStatus: boolean;
};

type MonitorTranslate = (key: string) => string;

/**
 * Builds monitor status frame properties from page copy, state snapshot, and default translations.
 *
 * @returns A fully populated `MonitorStatusPageFrameProps` object.
 */
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
      errorTitle: args.t('monitor.shared.errorTitle'),
    },
  });
}
