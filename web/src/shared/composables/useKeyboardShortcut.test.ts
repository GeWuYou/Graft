import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, nextTick, ref } from 'vue';

import { useKeyboardShortcut } from './useKeyboardShortcut';

function dispatchShortcut(target: EventTarget, options: KeyboardEventInit = {}) {
  const event = new KeyboardEvent('keydown', {
    bubbles: true,
    code: 'KeyS',
    key: 's',
    ...options,
  });
  target.dispatchEvent(event);
  return event;
}

describe('useKeyboardShortcut', () => {
  it('matches Ctrl+S and Cmd+S through the $mod shortcut', () => {
    const callback = vi.fn();
    const Harness = defineComponent({
      setup() {
        useKeyboardShortcut('$mod+KeyS', callback);
        return () => null;
      },
    });
    const wrapper = mount(Harness);

    dispatchShortcut(window, { ctrlKey: true });
    dispatchShortcut(window, { metaKey: true });

    expect(callback).toHaveBeenCalledTimes(2);
    wrapper.unmount();
  });

  it('does not bind while disabled and responds after enabled is restored', async () => {
    const callback = vi.fn();
    const enabled = ref(false);
    const Harness = defineComponent({
      setup() {
        useKeyboardShortcut('$mod+KeyS', callback, { enabled });
        return () => null;
      },
    });
    const wrapper = mount(Harness);

    dispatchShortcut(window, { ctrlKey: true });
    enabled.value = true;
    await nextTick();
    dispatchShortcut(window, { ctrlKey: true });

    expect(callback).toHaveBeenCalledTimes(1);
    wrapper.unmount();
  });

  it('prevents the browser default before invoking the callback', () => {
    const callback = vi.fn();
    const Harness = defineComponent({
      setup() {
        useKeyboardShortcut('$mod+KeyS', callback, { preventDefault: true });
        return () => null;
      },
    });
    const wrapper = mount(Harness);
    const event = dispatchShortcut(window, { cancelable: true, ctrlKey: true });

    expect(event.defaultPrevented).toBe(true);
    expect(callback).toHaveBeenCalledOnce();
    wrapper.unmount();
  });

  it('suppresses repeated shortcut callbacks while preserving preventDefault', () => {
    const callback = vi.fn();
    const Harness = defineComponent({
      setup() {
        useKeyboardShortcut('$mod+KeyS', callback, { ignoreRepeat: true, preventDefault: true });
        return () => null;
      },
    });
    const wrapper = mount(Harness);
    const event = dispatchShortcut(window, { cancelable: true, ctrlKey: true, repeat: true });

    expect(event.defaultPrevented).toBe(true);
    expect(callback).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it('only listens within its configured target and cleans up on unmount', () => {
    const callback = vi.fn();
    const target = document.createElement('div');
    const editor = document.createElement('textarea');
    target.append(editor);
    const Harness = defineComponent({
      setup() {
        useKeyboardShortcut('$mod+KeyS', callback, { target: ref(target) });
        return () => null;
      },
    });
    const wrapper = mount(Harness);

    dispatchShortcut(window, { ctrlKey: true });
    dispatchShortcut(editor, { ctrlKey: true });
    expect(callback).toHaveBeenCalledTimes(1);

    wrapper.unmount();
    dispatchShortcut(target, { ctrlKey: true });
    expect(callback).toHaveBeenCalledTimes(1);
  });
});
