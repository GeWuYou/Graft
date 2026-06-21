import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { TerminalSessionConnector } from './terminal-types';
import WebTerminal from './WebTerminal.vue';

const terminalCtor = vi.fn();
const terminalOpen = vi.fn();
const terminalOnData = vi.fn();
const terminalLoadAddon = vi.fn();
const terminalWrite = vi.fn();
const terminalFocus = vi.fn();
const terminalDispose = vi.fn();
const terminalReset = vi.fn();
const terminalClear = vi.fn();

const fitAddonCtor = vi.fn();
const fitAddonFit = vi.fn();
const searchAddonCtor = vi.fn();
const searchAddonFindNext = vi.fn();
const webLinksAddonCtor = vi.fn();

vi.mock('@xterm/xterm', () => ({
  Terminal: vi.fn().mockImplementation(() => {
    terminalCtor();
    return {
      cols: 120,
      rows: 32,
      dispose: terminalDispose,
      focus: terminalFocus,
      clear: terminalClear,
      loadAddon: terminalLoadAddon,
      onData: terminalOnData,
      open: terminalOpen,
      reset: terminalReset,
      write: terminalWrite,
    };
  }),
}));

vi.mock('@xterm/addon-fit', () => ({
  FitAddon: vi.fn().mockImplementation(() => {
    fitAddonCtor();
    return {
      fit: fitAddonFit,
    };
  }),
}));

vi.mock('@xterm/addon-search', () => ({
  SearchAddon: vi.fn().mockImplementation(() => {
    searchAddonCtor();
    return {
      findNext: searchAddonFindNext,
    };
  }),
}));

vi.mock('@xterm/addon-web-links', () => ({
  WebLinksAddon: vi.fn().mockImplementation(() => {
    webLinksAddonCtor();
    return {};
  }),
}));

class ResizeObserverMock {
  observe = vi.fn();
  disconnect = vi.fn();
}

describe('WebTerminal', () => {
  const connector: TerminalSessionConnector = {
    open: vi.fn(),
  };
  const originalRequestAnimationFrame = window.requestAnimationFrame;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('ResizeObserver', ResizeObserverMock);
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    window.requestAnimationFrame = originalRequestAnimationFrame;
  });

  it('does not instantiate terminal or addons during setup', () => {
    mount(WebTerminal, {
      props: {
        connector,
        modelValue: false,
      },
      shallow: true,
    });

    expect(terminalCtor).toHaveBeenCalledTimes(1);
    expect(fitAddonCtor).toHaveBeenCalledTimes(1);
    expect(searchAddonCtor).toHaveBeenCalledTimes(1);
    expect(webLinksAddonCtor).toHaveBeenCalledTimes(1);
  });

  it('swallows internal ensureConnected rejection when auto-connect activation fails', async () => {
    const rejectionSpy = vi.fn();
    window.addEventListener('unhandledrejection', rejectionSpy);
    const failingConnector: TerminalSessionConnector = {
      open: vi.fn().mockRejectedValue(new Error('open failed')),
    };

    mount(WebTerminal, {
      props: {
        connector: failingConnector,
        modelValue: true,
      },
      attachTo: document.body,
    });

    await flushPromises();

    expect(failingConnector.open).toHaveBeenCalledTimes(1);
    expect(rejectionSpy).not.toHaveBeenCalled();
    window.removeEventListener('unhandledrejection', rejectionSpy);
  });

  it('resets terminal surface before reconnecting after tab re-entry', async () => {
    const successfulConnector: TerminalSessionConnector = {
      open: vi.fn().mockResolvedValue({ url: 'ws://localhost/terminal' }),
    };

    const wrapper = mount(WebTerminal, {
      props: {
        connector: successfulConnector,
        modelValue: true,
      },
      attachTo: document.body,
    });

    await flushPromises();
    expect(terminalReset).toHaveBeenCalledTimes(1);
    expect(terminalClear).toHaveBeenCalledTimes(1);

    await wrapper.setProps({ modelValue: false });
    await flushPromises();
    await wrapper.setProps({ modelValue: true });
    await flushPromises();

    expect(terminalReset).toHaveBeenCalledTimes(2);
    expect(terminalClear).toHaveBeenCalledTimes(2);
  });
});
