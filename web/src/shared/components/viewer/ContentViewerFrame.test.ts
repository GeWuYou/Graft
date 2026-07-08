import { mount, type VueWrapper } from '@vue/test-utils';
import { afterEach, describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import ContentViewerFrame from './ContentViewerFrame.vue';

const mountedWrappers: VueWrapper[] = [];

describe('ContentViewerFrame', () => {
  afterEach(() => {
    for (const wrapper of mountedWrappers.splice(0)) {
      wrapper.unmount();
    }
    localStorage.clear();
    document.body.style.overflow = '';
    document.documentElement.style.overflow = '';
  });

  it('toggles fullscreen and exits on escape', async () => {
    const wrapper = mountFrame();

    await wrapper.get('button').trigger('click');
    expect(wrapper.classes()).toContain('content-viewer-frame--fullscreen');
    expect(document.body.style.overflow).toBe('hidden');

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    await wrapper.vm.$nextTick();

    expect(wrapper.classes()).not.toContain('content-viewer-frame--fullscreen');
    expect(document.body.style.overflow).toBe('');
  });

  it('persists keyboard-driven resize height', async () => {
    const wrapper = mountFrame();

    await wrapper.get('.content-viewer-frame__resize-handle').trigger('keydown', { key: 'ArrowUp' });

    expect(localStorage.getItem('graft.test.viewer.height')).toBeTruthy();
    expect(wrapper.get('.content-viewer-frame__panel').attributes('style')).toContain('height:');
  });

  it('preserves a stored height instead of replacing it with the preferred viewport height on mount', () => {
    const originalWidth = window.innerWidth;
    const originalHeight = window.innerHeight;

    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1280 });
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 1000 });
    localStorage.setItem('graft.test.viewer.height', '640');

    try {
      const wrapper = mountFrame();

      expect(wrapper.get('.content-viewer-frame__panel').attributes('style')).toContain('height: 640px;');
    } finally {
      Object.defineProperty(window, 'innerWidth', { configurable: true, value: originalWidth });
      Object.defineProperty(window, 'innerHeight', { configurable: true, value: originalHeight });
    }
  });

  it('recomputes panel height from an updated defaultHeight prop before the user resizes', async () => {
    const wrapper = mountFrame({
      defaultHeight: 600,
    });

    expect(wrapper.get('.content-viewer-frame__panel').attributes('style')).toContain('height: 600px;');

    await wrapper.setProps({
      defaultHeight: 620,
    });

    expect(wrapper.get('.content-viewer-frame__panel').attributes('style')).toContain('height: 620px;');
  });

  it('fills the parent height when fillHeight is enabled', () => {
    const wrapper = mountFrame({
      fillHeight: true,
    });

    expect(wrapper.get('.content-viewer-frame__panel').attributes('style')).toContain('height: 100%;');
  });
});

function mountFrame(props: Record<string, unknown> = {}) {
  const wrapper = mount(ContentViewerFrame, {
    props: {
      storageKey: 'graft.test.viewer.height',
      fullscreenLabel: '全屏',
      exitFullscreenLabel: '退出全屏',
      resizeHandleLabel: '调整阅读器高度',
      ...props,
    },
    slots: {
      header: () => h('div', 'Header'),
      default: () => h('div', 'Content'),
    },
    global: {
      stubs: {
        't-button': defineComponent({
          emits: ['click'],
          setup(_, { attrs, emit, slots }) {
            return () => h('button', { ...attrs, onClick: () => emit('click') }, slots.default?.());
          },
        }),
        't-tooltip': defineComponent({
          setup(_, { slots }) {
            return () => h('div', slots.default?.());
          },
        }),
      },
    },
  });
  mountedWrappers.push(wrapper);
  return wrapper;
}
