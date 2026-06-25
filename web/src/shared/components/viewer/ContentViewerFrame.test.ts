import { mount } from '@vue/test-utils';
import { afterEach, describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import ContentViewerFrame from './ContentViewerFrame.vue';

describe('ContentViewerFrame', () => {
  afterEach(() => {
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
});

function mountFrame() {
  return mount(ContentViewerFrame, {
    props: {
      storageKey: 'graft.test.viewer.height',
      fullscreenLabel: '全屏',
      exitFullscreenLabel: '退出全屏',
      resizeHandleLabel: '调整阅读器高度',
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
}
