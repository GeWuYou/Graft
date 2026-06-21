import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import CronScheduleDialog from './CronScheduleDialog.vue';

const tDesignStubs = {
  TButton: defineComponent({
    name: 'TButton',
    props: ['disabled'],
    emits: ['click'],
    setup(props, { emit, slots }) {
      return () =>
        h(
          'button',
          {
            disabled: props.disabled,
            onClick: (event: MouseEvent) => emit('click', event),
          },
          slots.default?.(),
        );
    },
  }),
  TDialog: defineComponent({
    name: 'TDialog',
    props: ['cancelBtn', 'confirmBtn', 'header', 'visible', 'width'],
    emits: ['update:visible', 'confirm', 'cancel', 'close'],
    setup(props, { emit, slots }) {
      return () =>
        props.visible
          ? h('section', { 'data-testid': 'cron-dialog', 'data-width': props.width }, [
              h('h2', props.header),
              slots.default?.(),
              h(
                'button',
                { 'data-testid': 'dialog-cancel', onClick: () => emit('cancel', { e: new MouseEvent('click') }) },
                props.cancelBtn,
              ),
              h(
                'button',
                {
                  'data-testid': 'dialog-confirm',
                  disabled: props.confirmBtn?.disabled,
                  onClick: () => emit('confirm', { e: new MouseEvent('click') }),
                },
                props.confirmBtn?.content,
              ),
            ])
          : null;
    },
  }),
  TForm: defineComponent({
    name: 'TForm',
    setup(_props, { slots }) {
      return () => h('form', slots.default?.());
    },
  }),
  TFormItem: defineComponent({
    name: 'TFormItem',
    props: ['label'],
    setup(props, { slots }) {
      return () => h('label', [h('span', props.label), slots.default?.()]);
    },
  }),
  TInput: defineComponent({
    name: 'TInput',
    props: ['modelValue', 'value'],
    emits: ['update:modelValue', 'update:value', 'change', 'blur'],
    setup(props, { emit }) {
      return () =>
        h('input', {
          'data-testid': 'raw-expression-input',
          value: props.modelValue ?? props.value,
          onInput: (event: Event) => {
            const value = (event.target as HTMLInputElement).value;
            emit('update:modelValue', value);
            emit('update:value', value);
            emit('change', value);
          },
          onBlur: (event: Event) => emit('blur', (event.target as HTMLInputElement).value),
        });
    },
  }),
  TInputNumber: defineComponent({
    name: 'TInputNumber',
    inheritAttrs: false,
    props: ['modelValue', 'value'],
    emits: ['update:modelValue', 'update:value', 'change'],
    setup(props, { attrs, emit }) {
      return () =>
        h('input', {
          ...attrs,
          'data-testid': attrs['data-testid'] ?? 'input-number',
          type: 'number',
          value: props.modelValue ?? props.value,
          onInput: (event: Event) => {
            const value = Number((event.target as HTMLInputElement).value);
            emit('update:modelValue', value);
            emit('update:value', value);
            emit('change', value);
          },
        });
    },
  }),
  TRadioGroup: defineComponent({
    name: 'TRadioGroup',
    props: ['options', 'value'],
    emits: ['update:value', 'change'],
    setup(props, { emit }) {
      return () =>
        h(
          'div',
          { 'data-testid': 'weekday-radio-group' },
          (props.options ?? []).map((option: { label: string; value: number }) =>
            h(
              'button',
              {
                'data-testid': `weekday-${option.value}`,
                type: 'button',
                onClick: () => {
                  emit('update:value', option.value);
                  emit('change', option.value);
                },
              },
              option.label,
            ),
          ),
        );
    },
  }),
  TTag: defineComponent({
    name: 'TTag',
    props: ['theme'],
    setup(props, { slots }) {
      return () => h('span', { 'data-testid': 'preview-status', 'data-theme': props.theme }, slots.default?.());
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
        cronScheduleDialog: {
          cancel: '取消',
          confirm: '确认使用',
          examples: {
            dailyAtFive: '每天 17:00',
            everyFiveMinutes: '每 5 分钟',
            monthlyFirst: '每月 1 日 00:00',
            weeklyMonday: '每周一 09:00',
          },
          fields: {
            dayOfMonth: '每月日期',
            executionMinute: '执行分钟',
            executionTime: '执行时间',
            hour: '小时',
            intervalMinutes: '间隔分钟',
            minute: '分钟',
            rawExpression: 'Cron 表达式',
            weekday: '星期',
          },
          planTypeLabel: '常用计划类型',
          planTypes: {
            advanced: '高级 Cron',
            daily: '每天',
            hourly: '每小时',
            intervalMinutes: '每 N 分钟',
            monthly: '每月',
            weekly: '每周',
          },
          hourlyPrefix: '每小时第',
          hourlySuffix: '分钟执行',
          monthDayWarning: '部分月份没有该日期时不会执行。',
          preview: {
            description: '说明',
            emptyExpression: '暂无表达式',
            expression: '表达式',
            interval: '预计间隔',
            intervalDays: '约 {count} 天',
            intervalHours: '约 {count} 小时',
            intervalMinutes: '约 {count} 分钟',
            intervalSeconds: '约 {count} 秒',
            nextRun: '下次执行',
            noPreview: '暂无预览',
            status: '状态',
            upcomingRuns: '后续执行',
            valid: '有效',
            invalid: '无效',
          },
          quickMinutes: '{count} 分钟',
          quickSelect: '快捷选择',
          quickTemplates: '快捷模板',
          rawPlaceholder: '例如 0 17 * * *',
          title: '配置执行计划',
          weekdays: {
            0: '周日',
            1: '周一',
            2: '周二',
            3: '周三',
            4: '周四',
            5: '周五',
            6: '周六',
          },
        },
        cronValidation: {
          fieldCount: 'Cron 表达式必须是 {unixFields} 字段 Unix Cron 或 {secondsFields} 字段秒级 Cron。',
          fieldRange: 'Cron {field} 字段必须是 * 或 {min} 到 {max} 之间的数字。',
          stepRange: 'Cron {field} 步长必须介于 {min} 到 {max} 之间。',
        },
      },
    },
  },
});

function mountDialog(props: { modelValue?: string; visible?: boolean } = {}) {
  return mount(CronScheduleDialog, {
    props: {
      modelValue: props.modelValue ?? '0 17 * * *',
      visible: props.visible ?? true,
    },
    global: {
      components: tDesignStubs,
      plugins: [i18n],
    },
  });
}

describe('CronScheduleDialog', () => {
  it('renders the native cron schedule dialog', () => {
    const wrapper = mountDialog();

    expect(wrapper.get('[data-testid="cron-dialog"]').attributes('data-width')).toBe('820px');
    expect(wrapper.find('[data-testid="cron-preview"]').exists()).toBe(true);
    expect(wrapper.get('[data-testid="cron-preview-expression"]').text()).toBe('0 17 * * *');
  });

  it('confirms expressions entered in advanced cron mode', async () => {
    const wrapper = mountDialog({ modelValue: '15 8 * * *' });

    await wrapper.get('[data-testid="dialog-confirm"]').trigger('click');

    expect(wrapper.emitted('confirm')?.at(-1)).toEqual(['0 15 8 * * *']);
    expect(wrapper.emitted('update:visible')?.at(-1)).toEqual([false]);
  });

  it('keeps the parent value untouched when cancelled', async () => {
    const wrapper = mountDialog();

    await wrapper
      .findAll('button')
      .find((button) => button.text() === '每天')!
      .trigger('click');
    await wrapper.get('[data-testid="dialog-cancel"]').trigger('click');

    expect(wrapper.emitted('confirm')).toBeUndefined();
    expect(wrapper.emitted('update:visible')?.at(-1)).toEqual([false]);
  });

  it('generates a daily expression from the common plan type', async () => {
    const wrapper = mountDialog({ modelValue: '*/5 * * * *' });

    await wrapper
      .findAll('button')
      .find((button) => button.text() === '每天')!
      .trigger('click');
    await wrapper.get('[data-testid="dialog-confirm"]').trigger('click');

    expect(wrapper.emitted('confirm')?.at(-1)).toEqual(['0 0 17 * * *']);
  });

  it('keeps daily time inputs visually separated from the separator', () => {
    const wrapper = mountDialog({ modelValue: '0 17 * * *' });
    const timeFields = wrapper.get('.cron-schedule-dialog__time-fields');
    const hourInput = wrapper.get('[data-testid="cron-time-hour-input"]');
    const separator = wrapper.get('[data-testid="cron-time-separator"]');
    const minuteInput = wrapper.get('[data-testid="cron-time-minute-input"]');

    expect(timeFields.element.children[0]).toBe(hourInput.element);
    expect(timeFields.element.children[1]).toBe(separator.element);
    expect(timeFields.element.children[2]).toBe(minuteInput.element);
    expect(hourInput.classes()).toContain('cron-schedule-dialog__time-input');
    expect(minuteInput.classes()).toContain('cron-schedule-dialog__time-input');
    expect(separator.classes()).toContain('cron-schedule-dialog__time-separator');
    expect(separator.text()).toBe(':');
  });

  it('generates an interval expression from quick choices', async () => {
    const wrapper = mountDialog({ modelValue: '0 17 * * *' });

    await wrapper
      .findAll('button')
      .find((button) => button.text() === '每 N 分钟')!
      .trigger('click');
    await wrapper
      .findAll('button')
      .find((button) => button.text() === '15 分钟')!
      .trigger('click');
    await wrapper.get('[data-testid="dialog-confirm"]').trigger('click');

    expect(wrapper.emitted('confirm')?.at(-1)).toEqual(['0 */15 * * * *']);
  });
});
