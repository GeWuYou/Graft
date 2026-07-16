import { describe, expect, it } from 'vitest';

import { projectLifecycleActionVisibility, projectRuntimeStatusLabel, projectRuntimeStatusTheme } from './display';

const enUSMessages = {
  'project.list.status.runtimeDegraded': '🟡 Degraded',
  'project.list.status.runtimeRunning': '🟢 Running',
  'project.list.status.runtimeStopped': '⚫ Stopped',
  'project.list.status.runtimeTransitioning': '🟠 Transitioning',
  'project.list.status.runtimeUnknown': '⚪ Unknown',
} as const;

const zhCNMessages = {
  'project.list.status.runtimeDegraded': '🟡 降级',
  'project.list.status.runtimeRunning': '🟢 运行中',
  'project.list.status.runtimeStopped': '⚫ 已停止',
  'project.list.status.runtimeTransitioning': '🟠 过渡中',
  'project.list.status.runtimeUnknown': '⚪ 未知',
} as const;

function createTranslator(messages: Record<string, string>) {
  return ((key: string) => messages[key] ?? key) as never;
}

describe('project display helpers', () => {
  it('returns emoji-prefixed localized runtime labels', () => {
    const enUSTranslator = createTranslator(enUSMessages);
    const zhCNTranslator = createTranslator(zhCNMessages);

    expect(projectRuntimeStatusLabel(enUSTranslator, 'running')).toBe('🟢 Running');
    expect(projectRuntimeStatusLabel(enUSTranslator, 'degraded')).toBe('🟡 Degraded');
    expect(projectRuntimeStatusLabel(enUSTranslator, 'stopped')).toBe('⚫ Stopped');
    expect(projectRuntimeStatusLabel(enUSTranslator, 'transitioning')).toBe('🟠 Transitioning');
    expect(projectRuntimeStatusLabel(enUSTranslator, 'unknown')).toBe('⚪ Unknown');

    expect(projectRuntimeStatusLabel(zhCNTranslator, 'running')).toBe('🟢 运行中');
    expect(projectRuntimeStatusLabel(zhCNTranslator, 'degraded')).toBe('🟡 降级');
    expect(projectRuntimeStatusLabel(zhCNTranslator, 'stopped')).toBe('⚫ 已停止');
    expect(projectRuntimeStatusLabel(zhCNTranslator, 'transitioning')).toBe('🟠 过渡中');
    expect(projectRuntimeStatusLabel(zhCNTranslator)).toBe('⚪ 未知');
  });

  it('keeps runtime tag themes aligned with the canonical enum', () => {
    expect(projectRuntimeStatusTheme('running')).toBe('success');
    expect(projectRuntimeStatusTheme('degraded')).toBe('warning');
    expect(projectRuntimeStatusTheme('stopped')).toBe('default');
    expect(projectRuntimeStatusTheme('transitioning')).toBe('primary');
    expect(projectRuntimeStatusTheme('unknown')).toBe('default');
    expect(projectRuntimeStatusTheme()).toBe('default');
  });

  it('applies the shared lifecycle visibility rules', () => {
    expect(projectLifecycleActionVisibility('running')).toEqual({
      up: false,
      stop: true,
      restart: true,
      redeploy: true,
      unregister: true,
    });
    expect(projectLifecycleActionVisibility('degraded')).toEqual({
      up: false,
      stop: true,
      restart: true,
      redeploy: true,
      unregister: true,
    });
    expect(projectLifecycleActionVisibility('stopped')).toEqual({
      up: true,
      stop: false,
      restart: true,
      redeploy: true,
      unregister: true,
    });
    expect(projectLifecycleActionVisibility('unknown')).toEqual({
      up: true,
      stop: true,
      restart: true,
      redeploy: true,
      unregister: true,
    });
    expect(projectLifecycleActionVisibility('transitioning', { hideLifecycleActions: true })).toEqual({
      up: false,
      stop: false,
      restart: false,
      redeploy: false,
      unregister: true,
    });
  });
});
