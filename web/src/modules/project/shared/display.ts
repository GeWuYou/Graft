import type { ComposerTranslation } from 'vue-i18n';

import { formatCompactDateTime } from '@/shared/components/management';

import type {
  ProjectDriftStatus,
  ProjectOwnershipMode,
  ProjectRefreshStatus,
  ProjectRuntimeStatus,
  ProjectSourceKind,
} from '../types/project';

type Translate = ComposerTranslation;

/**
 * 格式化项目时间。
 *
 * @param locale - 本地化标识
 * @param value - 要格式化的时间值
 * @returns 格式化后的时间字符串
 */
export function formatProjectTime(locale: string, value?: string | null) {
  return formatCompactDateTime(value, locale);
}

/**
 * 获取项目来源类型的显示文案。
 *
 * @param value - 项目来源类型
 * @returns 对应的项目来源类型文案
 */
export function projectSourceKindLabel(t: Translate, value: ProjectSourceKind) {
  return t(`project.list.sourceKinds.${value}`);
}

/**
 * 获取项目所有权模式的本地化文案。
 *
 * @param t - 翻译函数
 * @param value - 项目所有权模式值
 * @returns 对应 `project.detail.ownershipMode.${value}` 的翻译结果
 */
export function projectOwnershipModeLabel(t: Translate, value: ProjectOwnershipMode) {
  return t(`project.detail.ownershipMode.${value}`);
}

/**
 * 获取项目漂移状态的显示文案。
 *
 * @returns 与 `project.list.driftStatus.${value}` 对应的翻译结果。
 */
export function projectDriftStatusLabel(t: Translate, value: ProjectDriftStatus) {
  return t(`project.list.driftStatus.${value}`);
}

/**
 * 将项目漂移状态映射为主题语义。
 *
 * @param value - 项目漂移状态
 * @returns `success`、`default` 或 `warning` 主题值
 */
export function projectDriftStatusTheme(value?: ProjectDriftStatus) {
  if (value === 'clean') return 'success';
  if (value === 'unknown') return 'default';
  return 'warning';
}

/**
 * 获取项目刷新状态的显示文案。
 *
 * @param value - 刷新状态值
 * @returns 对应刷新状态的本地化文案
 */
export function projectRefreshStatusLabel(t: Translate, value: ProjectRefreshStatus) {
  return t(`project.list.refreshStatus.${value}`);
}

/**
 * 将刷新状态映射为主题语义。
 *
 * @param value - 刷新状态值
 * @returns 对应的主题值；`success` 对应 `success`，`failed` 对应 `danger`，其他值对应 `default`
 */
export function projectRefreshStatusTheme(value?: ProjectRefreshStatus) {
  if (value === 'success') return 'success';
  if (value === 'failed') return 'danger';
  return 'default';
}

/**
 * 将项目运行时状态映射为主题语义。
 *
 * @param value - 项目运行时状态
 * @returns 对应的主题值
 */
export function projectRuntimeStatusTheme(value?: ProjectRuntimeStatus | null) {
  if (value === 'running') return 'success';
  if (value === 'partial') return 'warning';
  if (value === 'stopped') return 'default';
  if (value === 'empty') return 'default';
  return 'default';
}

/**
 * 获取运行时状态对应的展示文案。
 *
 * @param value - 运行时状态
 * @returns 对应状态的翻译文本
 */
export function projectRuntimeStatusLabel(t: Translate, value?: ProjectRuntimeStatus | null) {
  if (value === 'running') return t('project.list.status.runtimeRunning');
  if (value === 'partial') return t('project.list.status.runtimePartial');
  if (value === 'stopped') return t('project.list.status.runtimeStopped');
  if (value === 'empty') return t('project.list.status.runtimeEmpty');
  return t('project.list.status.runtimeUnknown');
}
