// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import { useSettingStore } from '@/store';

import ThemeWorkbenchDock from './ThemeWorkbenchDock.vue';

vi.mock('@/locales', () => ({
  t: (key: string) => key,
}));

vi.mock('@/utils/color', () => ({
  composeThemeTokenMap: (tokens: Record<string, string>) => tokens,
  generateBrandColorMap: (brandTheme: string) => ({
    '--td-brand-color': brandTheme,
  }),
  insertThemeStylesheet: vi.fn(),
}));

const buttonStub = defineComponent({
  name: 'TButtonStub',
  props: {
    class: { type: [String, Object, Array], required: false, default: undefined },
    title: { type: String, required: false, default: undefined },
  },
  emits: ['click', 'pointerdown', 'pointermove', 'pointerup', 'pointercancel'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          class: props.class,
          title: props.title,
          type: 'button',
          onClick: () => emit('click'),
          onPointerdown: (event: PointerEvent) => emit('pointerdown', event),
          onPointermove: (event: PointerEvent) => emit('pointermove', event),
          onPointerup: (event: PointerEvent) => emit('pointerup', event),
          onPointercancel: (event: PointerEvent) => emit('pointercancel', event),
        },
        [slots.icon?.(), slots.default?.()],
      );
  },
});

const iconStub = defineComponent({
  name: 'TIconStub',
  setup() {
    return () => h('i');
  },
});

const mountDock = () =>
  mount(ThemeWorkbenchDock, {
    attachTo: document.body,
    global: {
      stubs: {
        't-button': buttonStub,
        't-icon': iconStub,
      },
    },
  });

const createPointerEvent = (type: string, options: PointerEventInit = {}) => {
  const event = new Event(type, { bubbles: true, cancelable: true }) as PointerEvent;
  Object.assign(event, {
    button: options.button ?? 0,
    clientX: options.clientX ?? 0,
    clientY: options.clientY ?? 0,
    pointerId: options.pointerId ?? 1,
  });
  return event;
};

const dispatchPointerEvent = async (element: Element, type: string, options: PointerEventInit = {}) => {
  element.dispatchEvent(createPointerEvent(type, options));
  await nextTick();
};

describe('ThemeWorkbenchDock', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    setActivePinia(createPinia());
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1200 });
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 800 });
    Element.prototype.setPointerCapture = vi.fn();
    Element.prototype.releasePointerCapture = vi.fn();
    Element.prototype.hasPointerCapture = vi.fn(() => true);
  });

  afterEach(() => {
    vi.useRealTimers();
    document.body.innerHTML = '';
  });

  it('opens and closes the theme workbench on normal click', async () => {
    const store = useSettingStore();
    const wrapper = mountDock();

    await wrapper.get('button').trigger('click');
    expect(store.showThemeWorkbench).toBe(true);

    await wrapper.get('button').trigger('click');
    expect(store.showThemeWorkbench).toBe(false);
  });

  it('persists a dragged dock position after long press and suppresses the release click', async () => {
    const store = useSettingStore();
    const wrapper = mountDock();
    const dock = wrapper.get('[data-testid="theme-workbench-dock"]').element as HTMLElement;
    vi.spyOn(dock, 'getBoundingClientRect').mockReturnValue({
      bottom: 780,
      height: 56,
      left: 572,
      right: 628,
      top: 724,
      width: 56,
      x: 572,
      y: 724,
      toJSON: () => ({}),
    });

    const button = wrapper.get('button');
    await dispatchPointerEvent(button.element, 'pointerdown', { clientX: 600, clientY: 752 });
    vi.advanceTimersByTime(450);
    await dispatchPointerEvent(button.element, 'pointermove', { clientX: 700, clientY: 620 });
    await dispatchPointerEvent(button.element, 'pointerup', { clientX: 700, clientY: 620 });
    await button.trigger('click');

    expect(store.themeWorkbenchDockPosition?.xRatio).toBeCloseTo(700 / 1200);
    expect(store.themeWorkbenchDockPosition?.yRatio).toBeCloseTo(620 / 800);
    expect(store.showThemeWorkbench).toBe(false);
  });
});
