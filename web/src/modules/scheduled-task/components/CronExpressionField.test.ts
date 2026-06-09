// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import CronExpressionField from './CronExpressionField.vue';

const CronScheduleDialogStub = defineComponent({
  name: 'CronScheduleDialog',
  props: ['modelValue', 'visible'],
  emits: ['update:visible', 'confirm'],
  setup(props, { emit }) {
    return () =>
      h('section', { 'data-testid': 'cron-schedule-dialog', 'data-visible': String(props.visible) }, [
        h('span', { 'data-testid': 'dialog-current-value' }, props.modelValue),
        h('button', { 'data-testid': 'dialog-confirm-daily', onClick: () => emit('confirm', '0 17 * * *') }, 'confirm'),
        h('button', { 'data-testid': 'dialog-cancel', onClick: () => emit('update:visible', false) }, 'cancel'),
      ]);
  },
});

const tDesignStubs = {
  TButton: defineComponent({
    name: 'TButton',
    props: ['disabled', 'theme', 'variant'],
    emits: ['click'],
    setup(props, { emit, slots }) {
      return () =>
        h(
          'button',
          {
            'data-testid': 'cron-config-button',
            'data-theme': props.theme,
            'data-variant': props.variant,
            disabled: props.disabled,
            onClick: (event: MouseEvent) => emit('click', event),
          },
          slots.default?.(),
        );
    },
  }),
  TInput: defineComponent({
    name: 'TInput',
    inheritAttrs: false,
    props: ['clearable', 'modelValue', 'status', 'value'],
    emits: ['update:modelValue', 'update:value', 'change', 'blur', 'focus'],
    setup(props, { attrs, emit, slots }) {
      return () =>
        h('div', { class: 't-input__wrap' }, [
          h('div', { class: 't-input' }, [
            h('input', {
              ...attrs,
              'data-testid': 'cron-expression-input',
              'data-clearable': String(props.clearable),
              'data-status': props.status,
              value: props.modelValue ?? props.value,
              onInput: (event: Event) => {
                const value = (event.target as HTMLInputElement).value;
                emit('update:modelValue', value);
                emit('update:value', value);
                emit('change', value);
              },
              onBlur: (event: Event) => emit('blur', (event.target as HTMLInputElement).value),
              onFocus: (event: Event) => emit('focus', (event.target as HTMLInputElement).value),
            }),
            slots.suffix?.(),
          ]),
        ]);
    },
  }),
  TInputAdornment: defineComponent({
    name: 'TInputAdornment',
    setup(_props, { slots }) {
      return () => h('div', [slots.default?.(), slots.append?.()]);
    },
  }),
  TTag: defineComponent({
    name: 'TTag',
    setup(_props, { slots }) {
      return () => h('span', { 'data-testid': 'cron-valid-tag' }, slots.default?.());
    },
  }),
};

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages: {
    'zh-CN': {
      scheduledTask: {
        cronDescription: {
          custom: '当前表达式：{expression}',
          daily: '每天 {hour}:00 执行一次。',
          everyMinute: '每分钟执行一次。',
          everyNMinutes: '每 {interval} 分钟执行一次。',
          hourly: '每小时整点执行一次。',
          invalid: 'Cron 表达式不合法。',
          monthly: '每月 {dayOfMonth} 日 {hour}:00 执行一次。',
          weekly: '每周第 {dayOfWeek} 天 {hour}:00 执行一次。',
        },
        cronExpressionField: {
          clear: '清空 Cron 表达式',
          configure: '配置',
          placeholder: '例如 0 17 * * *',
          validStatus: '有效',
        },
        cronValidation: {
          required: '请填写 Cron 表达式。',
          fieldCount: 'Cron 表达式必须是 {unixFields} 字段 Unix Cron 或 {secondsFields} 字段秒级 Cron。',
          fieldRange: 'Cron {field} 字段必须是 * 或 {min} 到 {max} 之间的数字。',
          stepRange: 'Cron {field} 步长必须介于 {min} 到 {max} 之间。',
        },
      },
    },
  },
});

function mountField(props: { modelValue?: string; error?: string } = {}) {
  return mount(CronExpressionField, {
    props: {
      error: props.error,
      modelValue: props.modelValue ?? '* * * * *',
    },
    global: {
      plugins: [i18n],
      stubs: {
        CronScheduleDialog: CronScheduleDialogStub,
        ...tDesignStubs,
      },
    },
  });
}

describe('CronExpressionField', () => {
  it('normalizes raw input and emits validation metadata', async () => {
    const wrapper = mountField();

    await wrapper.get('[data-testid="cron-expression-input"]').setValue('*/5 * * * *');

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['0 */5 * * * *']);
    expect(wrapper.emitted('validate')?.at(-1)).toEqual([
      {
        normalizedExpression: '0 */5 * * * *',
        valid: true,
      },
    ]);
    expect(wrapper.get('[data-testid="cron-expression-meta"]').text()).toContain('每 5 分钟执行一次。');
  });

  it('opens the schedule dialog from the configure button', async () => {
    const wrapper = mountField();

    expect(wrapper.get('[data-testid="cron-schedule-dialog"]').attributes('data-visible')).toBe('false');
    await wrapper.get('[data-testid="cron-config-button"]').trigger('click');

    expect(wrapper.get('[data-testid="cron-schedule-dialog"]').attributes('data-visible')).toBe('true');
  });

  it('keeps the cron input and configure button in one stable row', () => {
    const wrapper = mountField();
    const row = wrapper.get('[data-testid="cron-expression-row"]');
    const inputArea = wrapper.get('.scheduled-task-cron-input');
    const configureButton = wrapper.get('[data-testid="cron-config-button"]');

    expect(row.element.children[0]).toBe(inputArea.element);
    expect(row.element.children[1]).toBe(configureButton.element);
    expect(configureButton.attributes('data-theme')).toBe('primary');
    expect(configureButton.attributes('data-variant')).toBe('outline');
  });

  it('uses a legal cron placeholder and TDesign clearable input behavior', () => {
    const wrapper = mountField();
    const input = wrapper.get('[data-testid="cron-expression-input"]');

    expect(input.attributes('placeholder')).toBe('例如 0 17 * * *');
    expect(input.attributes('data-clearable')).toBeDefined();
  });

  it('applies confirmed dialog expressions', async () => {
    const wrapper = mountField();

    await wrapper.get('[data-testid="dialog-confirm-daily"]').trigger('click');

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['0 0 17 * * *']);
  });

  it('shows invalid input state without embedding a scheduler builder in the field surface', async () => {
    const wrapper = mountField({ modelValue: '0 0 24 * * *' });

    expect(wrapper.get('[data-testid="cron-expression-input"]').attributes('data-status')).toBe('error');
    expect(wrapper.get('[data-testid="cron-expression-error"]').text()).toContain(
      'Cron hours 字段必须是 * 或 0 到 23 之间的数字。',
    );
    expect(wrapper.find('[data-testid="cron-expression-meta"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="cron-preview"]').exists()).toBe(false);
  });

  it('shows field count errors below the whole cron row', async () => {
    const wrapper = mountField({ modelValue: '*/5 * * *' });

    expect(wrapper.get('[data-testid="cron-expression-error"]').text()).toContain(
      'Cron 表达式必须是 5 字段 Unix Cron 或 6 字段秒级 Cron。',
    );
    expect(wrapper.find('[data-testid="cron-expression-meta"]').exists()).toBe(false);
  });

  it('shows required errors for empty input', async () => {
    const wrapper = mountField({ modelValue: '' });

    expect(wrapper.get('[data-testid="cron-expression-error"]').text()).toContain('请填写 Cron 表达式。');
    expect(wrapper.find('[data-testid="cron-valid-tag"]').exists()).toBe(false);
  });
});
