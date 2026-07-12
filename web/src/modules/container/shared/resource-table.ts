import type { TdPrimaryTableProps } from 'tdesign-vue-next';

import type {
  ContainerOrchestratorType,
  ContainerSourceGroupKind,
  ContainerSourceMemberKind,
  ContainerSummaryRecord,
} from '../types/container';

type Translate = (key: string, params?: Record<string, unknown>) => string;

export type ContainerSourceQuickFilterTarget = 'group' | 'member';

export type ContainerSourceQuickFilter = {
  kind: ContainerSourceGroupKind | ContainerSourceMemberKind;
  orchestrator: ContainerOrchestratorType;
  value: string;
};

export type ContainerResourceRowAction = {
  disabled?: boolean;
  fallbackLabel: string;
  label: string;
  testId?: string;
  value: string;
};

export type ContainerResourceMetric = {
  available: boolean;
  changeClass: Record<string, boolean>;
  percentage: number;
  progressStatus: 'success' | 'warning' | undefined;
  tooltip: string;
  value: string;
};

export const CONTAINER_RESOURCE_COLUMN_STORAGE_KEY = 'graft.container.list.visibleColumns';
export const CONTAINER_RESOURCE_DEFAULT_VISIBLE_COLUMNS = [
  'row-select',
  'state',
  'name',
  'image',
  'source',
  'cpu',
  'memory',
  'ports',
  'network',
  'runtime_status',
  'created_at',
  'operation',
];
export const CONTAINER_RESOURCE_ALWAYS_VISIBLE_COLUMNS = ['row-select', 'state', 'name', 'operation'];
export const CONTAINER_RESOURCE_ALL_COLUMN_KEYS = [
  'row-select',
  'state',
  'name',
  'image',
  'source',
  'cpu',
  'memory',
  'ports',
  'network',
  'runtime_status',
  'created_at',
  'started_at',
  'restart_policy',
  'image_id',
  'labels',
  'resource',
  'operation',
];

export function buildContainerResourceColumns(t: Translate): NonNullable<TdPrimaryTableProps['columns']> {
  return [
    { colKey: 'row-select', fixed: 'left', type: 'multiple' as const, width: 48, align: 'center' },
    { title: t('container.list.columns.status'), colKey: 'state', width: 104, align: 'center', ellipsis: false },
    {
      title: t('container.list.columns.name'),
      colKey: 'name',
      minWidth: 260,
      ellipsis: { theme: 'default', placement: 'top-left' },
    },
    {
      title: t('container.list.columns.image'),
      colKey: 'image',
      minWidth: 280,
      ellipsis: { theme: 'default', placement: 'top-left' },
    },
    { title: t('container.list.columns.source'), colKey: 'source', width: 188, ellipsis: false },
    { title: t('container.list.columns.cpu'), colKey: 'cpu', width: 132, align: 'center', ellipsis: false },
    { title: t('container.list.columns.memory'), colKey: 'memory', width: 180, align: 'center', ellipsis: false },
    { title: t('container.list.columns.ports'), colKey: 'ports', width: 220, ellipsis: false },
    { title: t('container.list.columns.network'), colKey: 'network', width: 176, ellipsis: false },
    { title: t('container.list.columns.resource'), colKey: 'resource', width: 168, ellipsis: false },
    {
      title: t('container.list.columns.runtimeStatus'),
      colKey: 'runtime_status',
      minWidth: 220,
      ellipsis: { theme: 'default', placement: 'top-left' },
    },
    { title: t('container.list.columns.createdAt'), colKey: 'created_at', width: 168, align: 'center' },
    { title: t('container.list.columns.startedAt'), colKey: 'started_at', width: 168, align: 'center' },
    { title: t('container.list.columns.restartPolicy'), colKey: 'restart_policy', width: 140, align: 'center' },
    {
      title: t('container.list.columns.imageId'),
      colKey: 'image_id',
      width: 220,
      ellipsis: { theme: 'default', placement: 'top-left' },
    },
    {
      title: t('container.list.columns.labels'),
      colKey: 'labels',
      width: 180,
      ellipsis: { theme: 'default', placement: 'top-left' },
    },
    {
      title: t('container.list.columns.operation'),
      colKey: 'operation',
      width: 152,
      fixed: 'right',
      align: 'center',
      ellipsis: false,
    },
  ];
}

export function buildContainerResourceColumnSettingOptions(t: Translate) {
  return [
    { label: t('container.list.columns.selection'), value: 'row-select' },
    { label: t('container.list.columns.status'), value: 'state' },
    { label: t('container.list.columns.name'), value: 'name' },
    { label: t('container.list.columns.image'), value: 'image' },
    { label: t('container.list.columns.source'), value: 'source' },
    { label: t('container.list.columns.cpu'), value: 'cpu' },
    { label: t('container.list.columns.memory'), value: 'memory' },
    { label: t('container.list.columns.ports'), value: 'ports' },
    { label: t('container.list.columns.network'), value: 'network' },
    { label: t('container.list.columns.resource'), value: 'resource' },
    { label: t('container.list.columns.runtimeStatus'), value: 'runtime_status' },
    { label: t('container.list.columns.createdAt'), value: 'created_at' },
    { label: t('container.list.columns.startedAt'), value: 'started_at' },
    { label: t('container.list.columns.restartPolicy'), value: 'restart_policy' },
    { label: t('container.list.columns.imageId'), value: 'image_id' },
    { label: t('container.list.columns.labels'), value: 'labels' },
    { label: t('container.list.columns.operation'), value: 'operation' },
  ];
}

export function displayContainerName(row: Pick<ContainerSummaryRecord, 'id' | 'name' | 'names'>) {
  return row.name || row.names?.[0] || row.id;
}

export function shortContainerId(id: string) {
  return id.length > 12 ? id.slice(0, 12) : id;
}

export function readContainerOrchestratorType(row: ContainerSummaryRecord): ContainerOrchestratorType {
  return row.deployment?.type || 'standalone';
}

export function createContainerSourceQuickFilter(
  row: ContainerSummaryRecord,
  target: ContainerSourceQuickFilterTarget,
): ContainerSourceQuickFilter | null {
  const sourceFilter = target === 'group' ? sourceGroupFilter(row) : sourceMemberFilter(row);
  if (!sourceFilter) {
    return null;
  }

  return {
    ...sourceFilter,
    orchestrator: readContainerOrchestratorType(row),
  };
}

/**
 * 从容器编排信息中构建群组作用域筛选条件。
 *
 * @param row - 包含编排信息的容器记录
 * @returns 有效的群组筛选条件；缺少作用域类型或值时返回 `null`
 */
function sourceGroupFilter(row: ContainerSummaryRecord): Omit<ContainerSourceQuickFilter, 'orchestrator'> | null {
  return toQuickFilterValue(
    row.deployment?.type === 'compose' ? 'compose_project' : undefined,
    row.deployment?.project,
  );
}

/**
 * 从容器编排器信息中创建成员范围筛选条件。
 *
 * @param row - 包含编排器成员范围信息的容器记录
 * @returns 有效的成员范围筛选条件；缺少成员范围类型或值时返回 `null`
 */
function sourceMemberFilter(row: ContainerSummaryRecord): Omit<ContainerSourceQuickFilter, 'orchestrator'> | null {
  return toQuickFilterValue(
    row.deployment?.type === 'compose' ? 'compose_service' : undefined,
    row.deployment?.service,
  );
}

/**
 * 构造经过清洗且有效的容器来源快速筛选值。
 *
 * @param kind - 快速筛选值的作用域类型
 * @param value - 快速筛选值的原始内容
 * @returns 包含作用域类型和值的快速筛选对象；输入缺少类型或有效值时返回 `null`
 */
function toQuickFilterValue(
  kind: ContainerSourceGroupKind | ContainerSourceMemberKind | null | undefined,
  value?: string | null,
): Omit<ContainerSourceQuickFilter, 'orchestrator'> | null {
  const normalizedValue = value?.trim();
  if (!kind || !normalizedValue) {
    return null;
  }

  return {
    kind,
    value: normalizedValue,
  };
}
