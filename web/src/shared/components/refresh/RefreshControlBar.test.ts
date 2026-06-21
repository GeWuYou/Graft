import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import RefreshControlBar from './RefreshControlBar.vue';

const translations: Record<string, string> = {
  'app.refreshControl.labels.interval': '自动刷新：',
  'app.refreshControl.labels.trendWindow': '趋势窗口：',
  'app.refreshControl.status.running': '自动刷新：{interval}',
  'app.refreshControl.status.paused': '自动刷新已暂停',
  'app.refreshControl.status.off': '自动刷新关闭',
  'app.refreshControl.countdown': '{countdown} 后刷新',
  'app.refreshControl.pending': '等待下次刷新',
  'app.refreshControl.actions.refresh': '立即刷新',
  'app.refreshControl.actions.pause': '暂停自动刷新',
  'app.refreshControl.actions.resume': '恢复自动刷新',
  'app.refreshControl.actions.enable': '开启自动刷新',
  'app.refreshControl.actions.pauseCompact': '暂停',
  'app.refreshControl.actions.resumeCompact': '恢复',
  'app.refreshControl.actions.enableCompact': '开启',
};

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      const template = translations[key] ?? key;
      return Object.entries(params ?? {}).reduce(
        (message, [name, value]) => message.replace(`{${name}}`, String(value)),
        template,
      );
    },
  }),
}));

const selectStub = defineComponent({
  inheritAttrs: false,
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  setup:
    (props, { attrs, emit }) =>
    () =>
      h(
        'select',
        {
          ...attrs,
          value: String(props.modelValue ?? ''),
          onChange: (event: Event) => {
            const rawValue = (event.target as HTMLSelectElement).value;
            const value = Number.isNaN(Number(rawValue)) ? rawValue : Number(rawValue);
            emit('update:modelValue', value);
          },
        },
        (props.options as Array<{ label: string; value: string | number }>).map((option) =>
          h('option', { value: option.value }, option.label),
        ),
      ),
});

const buttonStub = defineComponent({
  props: ['loading'],
  emits: ['click'],
  setup:
    (props, { attrs, emit, slots }) =>
    () =>
      h(
        'button',
        {
          ...attrs,
          'data-loading': String(Boolean(props.loading)),
          onClick: () => emit('click'),
        },
        [slots.icon?.(), slots.default?.()],
      ),
});

function mountBar(props: Partial<InstanceType<typeof RefreshControlBar>['$props']> = {}) {
  return mount(RefreshControlBar, {
    props: {
      status: 'running',
      interval: 5,
      intervalOptions: [
        { label: '每 5 秒', value: 5 },
        { label: '每 10 秒', value: 10 },
      ],
      ...props,
    },
    global: {
      stubs: {
        RefreshIcon: defineComponent({ name: 'RefreshIconStub', setup: () => () => h('span') }),
        TButton: buttonStub,
        TSelect: selectStub,
      },
    },
  });
}

describe('RefreshControlBar', () => {
  it('renders page running state with interval, countdown and actions', () => {
    const wrapper = mountBar({
      countdownSeconds: 5,
      showCountdown: true,
      statusLabel: '健康',
      statusTone: 'healthy',
      variant: 'page',
    });

    expect(wrapper.find('[data-refresh-interval-select="true"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('健康');
    expect(wrapper.text()).toContain('自动刷新：');
    expect(wrapper.text()).toContain('每 5 秒');
    expect(wrapper.get('[data-refresh-countdown="true"]').text()).toBe('5s 后刷新');
    expect(wrapper.get('[data-refresh-now="true"]').text()).toContain('立即刷新');
    expect(wrapper.get('[data-refresh-toggle-auto="true"]').text()).toContain('暂停自动刷新');
  });

  it('renders page paused state without countdown', () => {
    const wrapper = mountBar({
      countdownSeconds: 5,
      showCountdown: true,
      status: 'paused',
      variant: 'page',
    });

    expect(wrapper.find('[data-refresh-countdown="true"]').exists()).toBe(false);
    expect(wrapper.get('[data-refresh-auto-state="true"]').text()).toBe('自动刷新已暂停');
    expect(wrapper.get('[data-refresh-toggle-auto="true"]').text()).toContain('恢复自动刷新');
  });

  it('renders page off state without countdown', () => {
    const wrapper = mountBar({
      countdownSeconds: 5,
      showCountdown: true,
      status: 'off',
      variant: 'page',
    });

    expect(wrapper.find('[data-refresh-countdown="true"]').exists()).toBe(false);
    expect(wrapper.get('[data-refresh-auto-state="true"]').text()).toBe('自动刷新关闭');
    expect(wrapper.get('[data-refresh-toggle-auto="true"]').text()).toContain('开启自动刷新');
  });

  it('renders compact running state without page shell styling', () => {
    const wrapper = mountBar({
      countdownSeconds: 5,
      showCountdown: true,
      variant: 'compact',
    });

    expect(wrapper.get('[data-refresh-control-bar="true"]').attributes('data-refresh-variant')).toBe('compact');
    expect(wrapper.text()).toContain('自动刷新：');
    expect(wrapper.text()).toContain('每 5 秒');
    expect(wrapper.get('[data-refresh-countdown="true"]').text()).toBe('5s 后刷新');
    expect(wrapper.classes()).toContain('refresh-control-bar--compact');
    expect(wrapper.classes()).not.toContain('refresh-control-bar--page');
  });

  it('renders a plain appearance without outlined shell styling', () => {
    const wrapper = mountBar({
      appearance: 'plain',
      countdownSeconds: 5,
      showCountdown: true,
      variant: 'compact',
    });

    expect(wrapper.get('[data-refresh-control-bar="true"]').attributes('data-refresh-appearance')).toBe('plain');
    expect(wrapper.classes()).toContain('refresh-control-bar--plain');
    expect(wrapper.classes()).not.toContain('refresh-control-bar--outlined');
  });

  it('shows a pending message when running without countdown data', () => {
    const wrapper = mountBar({
      countdownSeconds: null,
      showCountdown: true,
      variant: 'page',
    });

    expect(wrapper.get('[data-refresh-countdown="true"]').text()).toBe('等待下次刷新');
  });

  it('emits pause and resume based on status', async () => {
    const wrapper = mountBar({ status: 'running' });

    await wrapper.get('[data-refresh-toggle-auto="true"]').trigger('click');
    expect(wrapper.emitted('pause')).toHaveLength(1);

    await wrapper.setProps({ status: 'paused' });
    await wrapper.get('[data-refresh-toggle-auto="true"]').trigger('click');
    expect(wrapper.emitted('resume')).toHaveLength(1);

    await wrapper.setProps({ status: 'off' });
    await wrapper.get('[data-refresh-toggle-auto="true"]').trigger('click');
    expect(wrapper.emitted('resume')).toHaveLength(2);
  });
});
