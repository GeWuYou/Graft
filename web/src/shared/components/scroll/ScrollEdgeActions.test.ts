import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { nextTick, ref } from 'vue';
import { defineComponent, h } from 'vue';

import { useScrollEdgeActions } from '@/shared/composables/useScrollEdgeActions';

import ScrollEdgeActions from './ScrollEdgeActions.vue';

const buttonStub = defineComponent({
  inheritAttrs: false,
  emits: ['click'],
  setup(_, { attrs, emit, slots }) {
    return () => h('button', { ...attrs, onClick: () => emit('click') }, slots.default?.() ?? slots.icon?.());
  },
});

function createTarget() {
  const target = document.createElement('div');
  Object.defineProperties(target, {
    scrollTop: { configurable: true, writable: true, value: 0 },
    clientHeight: { configurable: true, value: 100 },
    scrollHeight: { configurable: true, value: 300 },
  });
  return target;
}

function mountActions(
  compact: boolean,
  labels: { toTop: string; toBottom: string; group?: string } = {
    toTop: '返回顶部',
    toBottom: '跳到底部',
    group: '滚动控制',
  },
) {
  const target = createTarget();
  const controller = useScrollEdgeActions({ target });
  return mount(ScrollEdgeActions, {
    props: { controller, compact, labels },
    global: {
      stubs: {
        TButton: buttonStub,
        ArrowUpIcon: defineComponent({ setup: () => () => h('span') }),
        ArrowDownIcon: defineComponent({ setup: () => () => h('span') }),
      },
    },
  });
}

describe('ScrollEdgeActions', () => {
  it('renders only the bottom action at the top in compact mode', () => {
    const wrapper = mountActions(true, { toTop: '返回顶部', toBottom: '跳到底部', group: '滚动控制' });
    expect(wrapper.get('[data-scroll-edge-actions="true"]').attributes('aria-label')).toBe('滚动控制');
    expect(wrapper.find('[data-scroll-edge-to-top="true"]').exists()).toBe(false);
    const bottomButton = wrapper.get('[data-scroll-edge-to-bottom="true"]');
    expect(bottomButton.attributes('aria-label')).toBe('跳到底部');
    expect(bottomButton.attributes('block')).toBe('');
    expect(bottomButton.attributes('theme')).toBe('primary');
    expect(bottomButton.attributes('size')).toBe('medium');
  });

  it('renders both actions in the middle and invokes the controller', async () => {
    const target = createTarget();
    const controller = useScrollEdgeActions({ target });
    const toTop = vi.spyOn(controller, 'scrollToTop');
    const toBottom = vi.spyOn(controller, 'scrollToBottom');
    target.scrollTop = 100;
    target.dispatchEvent(new Event('scroll'));
    const wrapper = mount(ScrollEdgeActions, {
      props: {
        controller,
        compact: true,
        labels: { toTop: '返回顶部', toBottom: '跳到底部', group: '滚动控制' },
      },
      global: {
        stubs: {
          TButton: buttonStub,
          ArrowUpIcon: true,
          ArrowDownIcon: true,
        },
      },
    });
    expect(wrapper.findAll('button')).toHaveLength(2);
    await wrapper.get('[data-scroll-edge-to-top="true"]').trigger('click');
    await wrapper.get('[data-scroll-edge-to-bottom="true"]').trigger('click');
    expect(toTop).toHaveBeenCalledOnce();
    expect(toBottom).toHaveBeenCalledOnce();
  });

  it('preserves the bottom action DOM identity when the top action appears', async () => {
    const target = createTarget();
    const controller = useScrollEdgeActions({ target });
    const wrapper = mount(ScrollEdgeActions, {
      props: {
        controller,
        compact: true,
        labels: { toTop: '返回顶部', toBottom: '跳到底部', group: '滚动控制' },
      },
      global: {
        stubs: {
          TButton: buttonStub,
          ArrowUpIcon: true,
          ArrowDownIcon: true,
        },
      },
    });
    const bottomButton = wrapper.get('[data-scroll-edge-to-bottom="true"]').element;

    target.scrollTop = 100;
    target.dispatchEvent(new Event('scroll'));
    await nextTick();

    expect(wrapper.get('[data-scroll-edge-to-bottom="true"]').element).toBe(bottomButton);
  });

  it('does not render on wide screens', () => {
    const wrapper = mountActions(false);
    expect(wrapper.find('[data-scroll-edge-actions="true"]').exists()).toBe(false);
  });

  it('allows callers to replace the action icons through slots', () => {
    const target = createTarget();
    target.scrollTop = 100;
    target.dispatchEvent(new Event('scroll'));
    const controller = useScrollEdgeActions({ target });
    const wrapper = mount(ScrollEdgeActions, {
      props: {
        controller,
        compact: true,
        labels: { toTop: '返回顶部', toBottom: '跳到底部', group: '滚动控制' },
      },
      slots: {
        'top-icon': () => h('span', { 'data-custom-top-icon': 'true' }),
        'bottom-icon': () => h('span', { 'data-custom-bottom-icon': 'true' }),
      },
      global: {
        stubs: { TButton: buttonStub, ArrowUpIcon: true, ArrowDownIcon: true },
      },
    });
    expect(wrapper.find('[data-custom-top-icon="true"]').exists()).toBe(true);
    expect(wrapper.find('[data-custom-bottom-icon="true"]').exists()).toBe(true);
  });

  it('updates after the target is replaced', async () => {
    const first = createTarget();
    const second = createTarget();
    const target = ref<HTMLElement | null>(first);
    const controller = useScrollEdgeActions({ target });
    const wrapper = mount(ScrollEdgeActions, {
      props: {
        controller,
        compact: true,
        labels: { toTop: '返回顶部', toBottom: '跳到底部', group: '滚动控制' },
      },
      global: {
        stubs: { TButton: buttonStub, ArrowUpIcon: true, ArrowDownIcon: true },
      },
    });
    expect(wrapper.find('[data-scroll-edge-to-bottom="true"]').exists()).toBe(true);
    Object.defineProperty(second, 'scrollHeight', { configurable: true, value: 100 });
    Object.defineProperty(second, 'clientHeight', { configurable: true, value: 100 });
    target.value = second;
    await nextTick();
    expect(wrapper.find('[data-scroll-edge-actions="true"]').exists()).toBe(false);
  });
});
