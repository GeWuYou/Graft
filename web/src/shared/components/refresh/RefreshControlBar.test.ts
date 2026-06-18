// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import RefreshControlBar from './RefreshControlBar.vue';

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
      autoRefreshEnabled: true,
      interval: 5,
      intervalLabel: '刷新频率',
      intervalOptions: [
        { label: '每 5 秒', value: 5 },
        { label: '每 10 秒', value: 10 },
      ],
      pauseLabel: '暂停自动刷新',
      refreshLabel: '立即刷新',
      resumeLabel: '恢复自动刷新',
      ...props,
    },
    global: {
      stubs: {
        'refresh-icon': defineComponent({ setup: () => () => h('span') }),
        't-button': buttonStub,
        't-select': selectStub,
        't-tag': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('span', slots.default?.()),
        }),
      },
    },
  });
}

describe('RefreshControlBar', () => {
  it('renders the refresh interval select', () => {
    const wrapper = mountBar();

    expect(wrapper.get('[data-refresh-interval-select="true"]').element).toBeTruthy();
    expect(wrapper.text()).toContain('刷新频率');
  });

  it('optionally renders the trend window select', () => {
    const wrapper = mountBar({
      showTrendWindow: true,
      trendWindow: '10m',
      trendWindowLabel: '趋势窗口',
      trendWindowOptions: [
        { label: '10 分钟', value: '10m' },
        { label: '30 分钟', value: '30m' },
      ],
    });

    expect(wrapper.get('[data-refresh-trend-window-select="true"]').element).toBeTruthy();
    expect(wrapper.text()).toContain('趋势窗口');
  });

  it('emits refresh when the refresh button is clicked and shows loading state', async () => {
    const wrapper = mountBar({ refreshing: true });

    expect(wrapper.get('[data-refresh-now="true"]').attributes('data-loading')).toBe('true');
    await wrapper.get('[data-refresh-now="true"]').trigger('click');

    expect(wrapper.emitted('refresh')).toHaveLength(1);
  });

  it('renders only the countdown value when callers omit localized countdown labels', () => {
    const wrapper = mountBar({ countdownSeconds: 4, countdownLabel: '', showCountdown: true });

    expect(wrapper.get('[data-refresh-countdown="true"]').text()).toBe('4s');
  });

  it('renders caller-provided localized countdown labels', () => {
    const wrapper = mountBar({ countdownLabel: '下次刷新', countdownSeconds: 4, showCountdown: true });

    expect(wrapper.get('[data-refresh-countdown="true"]').text()).toContain('下次刷新4s');
  });

  it('renders a locale-neutral paused countdown state when callers omit labels', () => {
    const wrapper = mountBar({ countdownSeconds: 4, paused: true, pausedLabel: '', showCountdown: true });

    expect(wrapper.get('[data-refresh-countdown="true"]').text()).toBe('');
  });

  it('renders caller-provided localized paused labels', () => {
    const wrapper = mountBar({ countdownSeconds: 4, paused: true, pausedLabel: '已暂停', showCountdown: true });

    expect(wrapper.get('[data-refresh-countdown="true"]').text()).toContain('已暂停');
    expect(wrapper.text()).toContain('恢复自动刷新');
  });

  it('renders locale-neutral manual refresh when auto refresh is disabled and callers omit labels', () => {
    const wrapper = mountBar({ autoRefreshEnabled: false, manualLabel: '', showCountdown: true });

    expect(wrapper.get('[data-refresh-countdown="true"]').text()).toBe('');
    expect(wrapper.find('[data-refresh-toggle-auto="true"]').exists()).toBe(false);
  });

  it('renders caller-provided manual refresh when auto refresh is disabled', () => {
    const wrapper = mountBar({ autoRefreshEnabled: false, manualLabel: '手动刷新', showCountdown: true });

    expect(wrapper.get('[data-refresh-countdown="true"]').text()).toContain('手动刷新');
    expect(wrapper.find('[data-refresh-toggle-auto="true"]').exists()).toBe(false);
  });

  it('emits pause and resume based on paused state', async () => {
    const wrapper = mountBar({ autoRefreshEnabled: true, paused: false });

    await wrapper.get('[data-refresh-toggle-auto="true"]').trigger('click');
    expect(wrapper.emitted('pause')).toHaveLength(1);

    await wrapper.setProps({ paused: true });
    await wrapper.get('[data-refresh-toggle-auto="true"]').trigger('click');
    expect(wrapper.emitted('resume')).toHaveLength(1);
  });

  it('emits selected interval and trend window values', async () => {
    const wrapper = mountBar({
      showTrendWindow: true,
      trendWindow: '10m',
      trendWindowOptions: [
        { label: '10 分钟', value: '10m' },
        { label: '30 分钟', value: '30m' },
      ],
    });

    await wrapper.get('[data-refresh-interval-select="true"]').setValue('10');
    await wrapper.get('[data-refresh-trend-window-select="true"]').setValue('30m');

    expect(wrapper.emitted('update:interval')?.[0]).toEqual([10]);
    expect(wrapper.emitted('update:trendWindow')?.[0]).toEqual(['30m']);
  });

  it('does not coerce stringified numeric interval values', async () => {
    const wrapper = mountBar();

    wrapper.getComponent(selectStub).vm.$emit('update:modelValue', '10');

    expect(wrapper.emitted('update:interval')).toBeUndefined();
  });
});
