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

export function formatProjectTime(locale: string, value?: string | null) {
  return formatCompactDateTime(value, locale);
}

export function projectSourceKindLabel(t: Translate, value: ProjectSourceKind) {
  return t(`project.list.sourceKinds.${value}`);
}

export function projectOwnershipModeLabel(t: Translate, value: ProjectOwnershipMode) {
  return t(`project.detail.ownershipMode.${value}`);
}

export function projectDriftStatusLabel(t: Translate, value: ProjectDriftStatus) {
  return t(`project.list.driftStatus.${value}`);
}

export function projectDriftStatusTheme(value?: ProjectDriftStatus) {
  if (value === 'clean') return 'success';
  if (value === 'unknown') return 'default';
  return 'warning';
}

export function projectRefreshStatusLabel(t: Translate, value: ProjectRefreshStatus) {
  return t(`project.list.refreshStatus.${value}`);
}

export function projectRefreshStatusTheme(value?: ProjectRefreshStatus) {
  if (value === 'success') return 'success';
  if (value === 'failed') return 'danger';
  return 'default';
}

export function projectRuntimeStatusTheme(value?: ProjectRuntimeStatus | null) {
  if (value === 'running') return 'success';
  if (value === 'partial') return 'warning';
  if (value === 'stopped') return 'default';
  if (value === 'empty') return 'default';
  return 'default';
}

export function projectRuntimeStatusLabel(t: Translate, value?: ProjectRuntimeStatus | null) {
  if (value === 'running') return t('project.list.status.runtimeRunning');
  if (value === 'partial') return t('project.list.status.runtimePartial');
  if (value === 'stopped') return t('project.list.status.runtimeStopped');
  if (value === 'empty') return t('project.list.status.runtimeEmpty');
  return t('project.list.status.runtimeUnknown');
}
