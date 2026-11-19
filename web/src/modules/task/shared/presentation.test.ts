import { describe, expect, it } from 'vitest';
import { createI18n } from 'vue-i18n';

import enUS from '../locales/en-US.json';
import zhCN from '../locales/zh-CN.json';
import { resolveTaskFailureMessage } from './failure-copy';
import {
  taskDiagnosticSummary,
  taskFailureAttribution,
  taskFailureDetail,
  taskStageExecutionLabel,
} from './presentation';

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

  it('renders the trusted sanitized summary and withholds historical raw text', () => {
    const summary = 'yaml: line 3: could not find expected';
    expect(taskDiagnosticSummary(` ${summary} `)).toBe(summary);
    expect(taskDiagnosticSummary('first\nsecond')).toBe('');
    expect(taskDiagnosticSummary('first\rsecond')).toBe('');
    expect(taskDiagnosticSummary('长'.repeat(257))).toBe('');
    expect(taskDiagnosticSummary('')).toBe('');
    expect(taskDiagnosticSummary(null)).toBe('');
    expect(taskDiagnosticSummary(undefined)).toBe('');
  });

  it.each(['zh-CN', 'en-US'])(
    'composes failure detail with the sanitized summary actually rendered in %s',
    (locale) => {
      const i18n = createI18n({ legacy: false, locale, messages: { 'zh-CN': zhCN, 'en-US': enUS } });
      const translate = (key: string) => i18n.global.t(key);
      const summary = 'yaml: line 3: could not find expected';
      const detail = taskFailureDetail(
        {
          failure_code: 'provider_operation_failed',
          failure_message: summary,
          failure_stage: 'compose_load',
          failure_class: 'infrastructure',
        },
        translate,
      );
      expect(detail).toContain(translate('task.detail.failureCodes.providerOperationFailed'));
      expect(detail).toContain(summary);
      expect(detail).toContain(translate('task.detail.failureStages.compose_load'));
      expect(detail).toContain(translate('task.detail.failureClasses.infrastructure'));
      // 摘要与稳定码文案是独立段落，不能被通用 unknown 文案吞掉。
      const segments = detail.split(' · ');
      expect(segments).toContain(summary);
      expect(segments.length).toBe(4);
      expect(
        taskFailureDetail({ failure_code: 'docker_runtime_unavailable', failure_message: undefined }, translate),
      ).toBe(translate('task.detail.failureCodes.runtimeUnavailable'));
    },
  );
});
