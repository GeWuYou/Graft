import type { ComposerTranslation } from 'vue-i18n';

import { formatCompactDateTime } from '@/shared/components/management';

import type { ApplicationDriftStatus, ApplicationRuntimeStatus, ApplicationSourceType } from '../types/project';

type Translate = ComposerTranslation;

const projectTaskTypeLabelKeys: Readonly<Record<string, string>> = {
  'project.compose.redeploy': 'project.taskTypes.redeploy',
  'project.compose.restart': 'project.taskTypes.restart',
  'project.compose.stop': 'project.taskTypes.stop',
  'project.compose.up': 'project.taskTypes.up',
};

export function formatApplicationTime(locale: string, value?: string | null) {
  return formatCompactDateTime(value, locale);
}

export function projectSourceTypeLabel(t: Translate, value: ApplicationSourceType) {
  return t(`project.list.sourceTypes.${value}`);
}

export function projectDriftStatusLabel(t: Translate, value: ApplicationDriftStatus) {
  return t(`project.list.driftStatus.${value}`);
}

export function projectDriftStatusTheme(value?: ApplicationDriftStatus) {
  if (value === 'clean') return 'success';
  if (value === 'unknown') return 'default';
  return 'warning';
}

export function projectRuntimeStatusTheme(value?: ApplicationRuntimeStatus | null) {
  if (value === 'running') return 'success';
  if (value === 'degraded') return 'warning';
  if (value === 'stopped') return 'default';
  if (value === 'transitioning') return 'primary';
  return 'default';
}

export function projectRuntimeStatusLabel(t: Translate, value?: ApplicationRuntimeStatus | null) {
  if (value === 'running') return t('project.list.status.runtimeRunning');
  if (value === 'degraded') return t('project.list.status.runtimeDegraded');
  if (value === 'stopped') return t('project.list.status.runtimeStopped');
  if (value === 'transitioning') return t('project.list.status.runtimeTransitioning');
  return t('project.list.status.runtimeUnknown');
}

export function projectTaskTypeLabel(t: Translate, taskType: string) {
  const key = projectTaskTypeLabelKeys[taskType];
  return key ? t(key) : undefined;
}

export type ApplicationLifecycleAction = 'up' | 'stop' | 'restart' | 'redeploy' | 'unregister';

type ApplicationLifecycleActionVisibility = Record<ApplicationLifecycleAction, boolean>;

type ApplicationLifecycleActionVisibilityOptions = {
  hideLifecycleActions?: boolean;
};

export function projectLifecycleActionVisibility(
  value?: ApplicationRuntimeStatus | null,
  options: ApplicationLifecycleActionVisibilityOptions = {},
): ApplicationLifecycleActionVisibility {
  if (options.hideLifecycleActions) {
    return {
      up: false,
      stop: false,
      restart: false,
      redeploy: false,
      unregister: true,
    };
  }

  if (value === 'running' || value === 'degraded') {
    return {
      up: false,
      stop: true,
      restart: true,
      redeploy: true,
      unregister: true,
    };
  }

  if (value === 'stopped') {
    return {
      up: true,
      stop: false,
      restart: true,
      redeploy: true,
      unregister: true,
    };
  }

  return {
    up: true,
    stop: true,
    restart: true,
    redeploy: true,
    unregister: true,
  };
}
