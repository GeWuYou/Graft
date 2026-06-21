import { formatNanosecondsAsDuration, formatPercent } from '@/shared/observability';
import type { ResourceNumberFormatLocale } from '@/shared/observability/resource-format';

import type { ContainerDetail } from '../../../types/container';

const EMPTY_TEXT = '—';
const DEFAULT_CPU_NUMBER_LOCALE = 'en-US';

/** CpuDetailMetric 描述 CPU 详情面板中一项可渲染指标，保留强调、弱化与提示语义供页面统一消费。 */
export type CpuDetailMetric = {
  emphasized?: boolean;
  hint?: string;
  key: string;
  label: string;
  muted?: boolean;
  value: string;
};

/** CpuDetailPresenterLabels 收口 CPU 指标 presenter 需要的展示文案，避免 helper 直接依赖页面 i18n。 */
export type CpuDetailPresenterLabels = {
  cpuLimit: string;
  cpuPercent: string;
  kernelTime: string;
  systemCpuTime: string;
  throttlingCount: string;
  throttlingInactiveHint: string;
  throttlingSignalHint: string;
  throttlingTime: string;
  totalCpuTime: string;
  userTime: string;
};

type ContainerResourceSummary = NonNullable<ContainerDetail['resource']>;

/** buildCpuDetailMetrics 将容器资源快照转换为 CPU 详情面板模型，并在这里集中判定 throttling 状态语义。 */
export function buildCpuDetailMetrics(
  resource: ContainerResourceSummary | null | undefined,
  labels: CpuDetailPresenterLabels,
  locale: ResourceNumberFormatLocale = DEFAULT_CPU_NUMBER_LOCALE,
): CpuDetailMetric[] {
  const throttlingActive =
    isPositiveNumber(resource?.throttling_throttled_periods) || isPositiveNumber(resource?.throttling_throttled_time);
  const throttlingHint = throttlingActive ? labels.throttlingSignalHint : labels.throttlingInactiveHint;

  return [
    {
      key: 'cpu-percent',
      label: labels.cpuPercent,
      value: formatCpuPercent(resource?.cpu_percent),
    },
    {
      key: 'cpu-capacity',
      label: labels.cpuLimit,
      value: `${EMPTY_TEXT} / ${formatCpuCount(resource?.online_cpus, locale)}`,
    },
    {
      key: 'total-cpu-time',
      label: labels.totalCpuTime,
      value: formatCpuDuration(resource?.total_cpu_usage, locale),
    },
    {
      key: 'system-cpu-time',
      label: labels.systemCpuTime,
      value: formatCpuDuration(resource?.system_cpu_usage, locale),
    },
    {
      key: 'user-cpu-time',
      label: labels.userTime,
      value: formatCpuDuration(resource?.cpu_usage_in_usermode, locale),
    },
    {
      key: 'kernel-cpu-time',
      label: labels.kernelTime,
      value: formatCpuDuration(resource?.cpu_usage_in_kernelmode, locale),
    },
    {
      emphasized: throttlingActive,
      hint: throttlingHint,
      key: 'throttling-count',
      label: labels.throttlingCount,
      muted: !throttlingActive,
      value: formatPlainNumber(resource?.throttling_throttled_periods, locale),
    },
    {
      emphasized: throttlingActive,
      hint: throttlingHint,
      key: 'throttling-time',
      label: labels.throttlingTime,
      muted: !throttlingActive,
      value: formatCpuDuration(resource?.throttling_throttled_time, locale),
    },
  ];
}

export function formatCpuDuration(
  value?: number | null,
  locale: ResourceNumberFormatLocale = DEFAULT_CPU_NUMBER_LOCALE,
) {
  return formatNanosecondsAsDuration(value, EMPTY_TEXT, locale);
}

export function formatCpuSystemTime(
  value?: number | null,
  locale: ResourceNumberFormatLocale = DEFAULT_CPU_NUMBER_LOCALE,
) {
  // 保留系统 CPU 时间入口，作为页面与测试表达 Docker 字段语义的稳定边界。
  return formatCpuDuration(value, locale);
}

export function formatCpuCountText(
  value?: number | null,
  locale: ResourceNumberFormatLocale = DEFAULT_CPU_NUMBER_LOCALE,
) {
  // 保留 CPU 数量入口，避免调用方依赖 presenter 内部的单位拼接细节。
  return formatCpuCount(value, locale);
}

function formatCpuCount(value?: number | null, locale: ResourceNumberFormatLocale = DEFAULT_CPU_NUMBER_LOCALE) {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return EMPTY_TEXT;
  }
  return `${formatPlainNumber(value, locale)} CPU`;
}

function formatCpuPercent(value?: number | null) {
  return formatPercent(value, EMPTY_TEXT);
}

function formatPlainNumber(value?: number | null, locale: ResourceNumberFormatLocale = DEFAULT_CPU_NUMBER_LOCALE) {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return EMPTY_TEXT;
  }
  return new Intl.NumberFormat(locale, {
    maximumFractionDigits: 0,
  }).format(value);
}

function isPositiveNumber(value?: number | null) {
  return typeof value === 'number' && Number.isFinite(value) && value > 0;
}
