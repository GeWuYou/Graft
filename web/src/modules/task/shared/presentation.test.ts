import { describe, expect, it } from 'vitest';
import { createI18n } from 'vue-i18n';

import enUS from '../locales/en-US.json';
import zhCN from '../locales/zh-CN.json';
import { resolveTaskFailureMessage } from './failure-copy';
import { taskFailureAttribution, taskStageExecutionLabel } from './presentation';

describe('Task execution diagnostics', () => {
  it.each(['zh-CN', 'en-US'])('resolves existing and local failures from the actual %s messages', (locale) => {
    const i18n = createI18n({ legacy: false, locale, messages: { 'zh-CN': zhCN, 'en-US': enUS } });
    const translate = (key: string) => i18n.global.t(key);
    for (const code of [
      'docker_runtime_unavailable',
      'local_material_resolve_failed',
      'local_compose_load_failed',
      'local_compose_apply_failed',
      'local_container_apply_failed',
      'local_build_execute_failed',
      'local_execution_result_unconfirmed',
      'runner_interrupted',
      'local_execution_not_ready',
      'local_controller_pull_failed',
    ]) {
      const message = resolveTaskFailureMessage(code, 'sensitive provider output', translate);
      expect(message).not.toMatch(/^task\./);
      expect(message).not.toContain('sensitive');
      expect(message).not.toBe('');
    }
  });

  it('keeps historical stages readable and presents frozen execution provenance', () => {
    expect(taskStageExecutionLabel({}, (key) => key)).toBe('');
    expect(taskStageExecutionLabel({ execution_mode: 'local', execution_version: 'local-sdk/v1' }, (key) => key)).toBe(
      'task.executionModes.local · local-sdk/v1',
    );
    expect(taskStageExecutionLabel({ execution_mode: 'controller' }, (key) => key)).toBe(
      'task.executionModes.controller',
    );
  });

  it.each(['zh-CN', 'en-US'])('localizes bounded failure stages and classes from the actual %s messages', (locale) => {
    const i18n = createI18n({ legacy: false, locale, messages: { 'zh-CN': zhCN, 'en-US': enUS } });
    const translate = (key: string) => i18n.global.t(key);
    for (const stage of ['material_resolve', 'compose_load', 'compose_apply', 'container_apply', 'controller_pull']) {
      expect(taskFailureAttribution({ failure_stage: stage }, translate)).not.toMatch(/^task\./);
    }
    for (const failureClass of [
      'transient',
      'permanent',
      'configuration',
      'authorization',
      'provider',
      'infrastructure',
      'internal',
      'unknown',
    ]) {
      expect(taskFailureAttribution({ failure_class: failureClass }, translate)).not.toMatch(/^task\./);
    }
    expect(taskFailureAttribution({}, translate)).toBe('');
    expect(taskFailureAttribution({ failure_stage: 'compose_load', failure_class: 'infrastructure' }, translate)).toBe(
      [
        translate('task.detail.failureStages.compose_load'),
        translate('task.detail.failureClasses.infrastructure'),
      ].join(' · '),
    );
  });
});
