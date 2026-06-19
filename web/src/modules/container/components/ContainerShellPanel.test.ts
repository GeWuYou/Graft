// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, ref, watch } from 'vue';
import { createI18n } from 'vue-i18n';

import type { ApiRequestError } from '@/types/axios';

import ContainerShellPanel from './ContainerShellPanel.vue';

const shellSessionMock = vi.fn();
const permissionState = {
  hasPermission: vi.fn<(code: string) => boolean>(),
};
const terminalStubState = {
  connectCalls: 0,
  disconnectCalls: 0,
};

vi.mock('../api/container', () => ({
  postContainerShellSession: (...args: unknown[]) => shellSessionMock(...args),
}));

vi.mock('@/store', () => ({
  getPermissionStore: () => ({
    hasPermission: permissionState.hasPermission,
  }),
}));

vi.mock('@/shared/components/terminal/WebTerminal.vue', () => ({
  default: defineComponent({
    name: 'WebTerminalStub',
    props: {
      connector: { type: Object, required: true },
      modelValue: { type: Boolean, default: false },
      disconnectedDescription: { type: String, default: '' },
      disconnectedTitle: { type: String, default: '' },
      emptyDescription: { type: String, default: '' },
      emptyTitle: { type: String, default: '' },
      errorDescription: { type: String, default: '' },
      errorTitle: { type: String, default: '' },
    },
    emits: ['close', 'message', 'state-change'],
    setup(props, { emit, expose }) {
      const state = ref('idle');

      async function connect() {
        terminalStubState.connectCalls += 1;
        emit('state-change', 'connecting');
        state.value = 'connecting';
        try {
          await (props.connector as { open: (context: { cols: number; rows: number }) => Promise<unknown> }).open({
            cols: 120,
            rows: 32,
          });
          emit('state-change', 'connected');
          state.value = 'connected';
        } catch (error) {
          emit('close', 'connect_error');
          emit('state-change', 'error');
          state.value = 'error';
          throw error;
        }
      }

      function disconnect() {
        terminalStubState.disconnectCalls += 1;
        emit('close', 'manual_disconnect');
        emit('state-change', 'disconnected');
        state.value = 'disconnected';
      }

      expose({
        connect,
        disconnect,
        getState: () => state.value,
      });

      watch(
        () => props.modelValue,
        async (active) => {
          if (active) {
            try {
              await connect();
            } catch {
              // ContainerShellPanel maps connector errors into UI state.
            }
            return;
          }
          if (state.value !== 'idle') {
            disconnect();
          }
        },
        { immediate: true },
      );

      return () =>
        h('div', { 'data-testid': 'web-terminal-stub', 'data-active': String(Boolean(props.modelValue)) }, [
          h('div', { 'data-testid': 'terminal-empty-title' }, props.emptyTitle),
          h('div', { 'data-testid': 'terminal-empty-description' }, props.emptyDescription),
          h('div', { 'data-testid': 'terminal-error-title' }, props.errorTitle),
          h('div', { 'data-testid': 'terminal-error-description' }, props.errorDescription),
          h('div', { 'data-testid': 'terminal-disconnected-title' }, props.disconnectedTitle),
          h('div', { 'data-testid': 'terminal-disconnected-description' }, props.disconnectedDescription),
        ]);
    },
  }),
}));

const i18n = createI18n({
  legacy: false,
  locale: 'en-US',
  messages: {
    'en-US': {
      container: {
        detail: {
          shell: {
            title: 'Shell',
            description:
              'Access the current container through an interactive terminal. The session disconnects when the page closes.',
            disabled: 'Shell Is Disabled',
            disabledHint: 'Enable ops.container.shell.enabled in System Config before using Shell.',
            forbidden: 'No Shell Access',
            forbiddenHint: 'Required Permission: ops.container.shell',
            commands: {
              sh: 'SH',
              bash: 'Bash',
              ash: 'Ash',
            },
            empty: 'Shell Session Idle',
            emptyHint: 'Open the Shell tab to create an interactive session with the selected shell command.',
            connecting: 'Connecting',
            connectingHint: 'Preparing an interactive terminal session.',
            connected: 'Connected',
            disconnected: 'Disconnected',
            connectionFailed: 'Connection Failed',
            reconnect: 'Reconnect',
            sessionFailed: 'The shell session could not be created.',
            ticketUsed: 'The shell session ticket was already used. Reconnect to continue.',
            notRunning: 'Container Is Not Running',
            notRunningHint: 'The current container is not running, so an interactive shell session cannot be opened.',
            ticketExpired: 'The shell session ticket expired. Reconnect to continue.',
            connectionClosed: 'The shell connection has closed.',
            originDenied: 'The current request origin cannot open a container shell connection.',
            transportError:
              'The terminal transport connection failed. Verify the frontend WebSocket proxy and server origin allowlist.',
          },
        },
      },
      ops: {
        container: {
          error: {
            shellDisabled: 'Enable ops.container.shell.enabled in System Config before using Shell.',
            shellForbidden: 'Required Permission: ops.container.shell',
            shellContainerNotRunning:
              'The current container is not running, so an interactive shell session cannot be opened.',
          },
        },
      },
    },
  },
});

function createApiError(messageKey: string, message: string): ApiRequestError {
  const error = new Error(message) as ApiRequestError;
  error.name = 'ApiRequestError';
  error.status = 403;
  error.code = 'COMMON_FORBIDDEN';
  error.traceId = 'trace-shell';
  error.messageKey = messageKey;
  error.locale = 'en-US';
  error.responseData = {};
  error.isApiRequestError = true;
  return error;
}

function mountPanel(props?: Partial<InstanceType<typeof ContainerShellPanel>['$props']>) {
  return mount(ContainerShellPanel, {
    props: {
      active: false,
      containerId: 'container-1',
      containerState: 'running',
      ...props,
    },
    global: {
      plugins: [i18n],
      stubs: {
        't-alert': defineComponent({
          props: ['theme'],
          setup(props, { slots }) {
            return () =>
              h('div', { 'data-testid': 't-alert', 'data-theme': String(props.theme ?? '') }, [
                h('strong', { 'data-testid': 't-alert-title' }, slots.title?.()),
                slots.default?.(),
              ]);
          },
        }),
        't-button': defineComponent({
          props: ['disabled', 'loading'],
          emits: ['click'],
          setup(props, { attrs, emit, slots }) {
            return () =>
              h(
                'button',
                {
                  ...attrs,
                  disabled: Boolean(props.disabled),
                  'data-loading': String(Boolean(props.loading)),
                  onClick: () => {
                    if (!props.disabled) {
                      emit('click');
                    }
                  },
                },
                slots.default?.(),
              );
          },
        }),
        't-icon': defineComponent({
          props: ['name'],
          setup(props) {
            return () => h('span', { 'data-testid': 't-icon' }, String(props.name ?? ''));
          },
        }),
        't-select': defineComponent({
          inheritAttrs: false,
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          setup(props, { attrs, emit }) {
            return () =>
              h(
                'select',
                {
                  ...attrs,
                  value: String(props.modelValue ?? ''),
                  onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
                },
                (props.options as Array<{ label: string; value: string }>).map((option) =>
                  h('option', { value: option.value }, option.label),
                ),
              );
          },
        }),
        't-tag': defineComponent({
          props: ['theme'],
          setup(props, { slots }) {
            return () =>
              h('span', { 'data-testid': 't-tag', 'data-theme': String(props.theme ?? '') }, slots.default?.());
          },
        }),
      },
    },
  });
}

describe('ContainerShellPanel', () => {
  beforeEach(() => {
    permissionState.hasPermission.mockReset();
    permissionState.hasPermission.mockReturnValue(true);
    shellSessionMock.mockReset();
    terminalStubState.connectCalls = 0;
    terminalStubState.disconnectCalls = 0;
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('does not request a shell session before the tab becomes active', async () => {
    shellSessionMock.mockResolvedValue({
      websocket_url: '/api/ops/containers/container-1/shell/ws?ticket=opaque-ticket',
    });

    mountPanel({ active: false });
    await flushPromises();

    expect(shellSessionMock).not.toHaveBeenCalled();
    expect(terminalStubState.connectCalls).toBe(0);
  });

  it('shows forbidden state without requesting a shell session when permission is missing', async () => {
    permissionState.hasPermission.mockReturnValue(false);

    const wrapper = mountPanel({ active: true });
    await flushPromises();

    expect(wrapper.text()).toContain('No Shell Access');
    expect(wrapper.text()).toContain('Required Permission: ops.container.shell');
    expect(shellSessionMock).not.toHaveBeenCalled();
    expect(wrapper.find('[data-testid="web-terminal-stub"]').exists()).toBe(false);
  });

  it('shows not running state without requesting a shell session when the container is stopped', async () => {
    const wrapper = mountPanel({ active: true, containerState: 'exited' });
    await flushPromises();

    expect(wrapper.text()).toContain('Container Is Not Running');
    expect(wrapper.text()).toContain(
      'The current container is not running, so an interactive shell session cannot be opened.',
    );
    expect(shellSessionMock).not.toHaveBeenCalled();
    expect(wrapper.find('[data-testid="web-terminal-stub"]').exists()).toBe(false);
  });

  it('requests a shell session only after the tab becomes active', async () => {
    shellSessionMock.mockResolvedValue({
      websocket_url: '/api/ops/containers/container-1/shell/ws?ticket=opaque-ticket',
    });

    const wrapper = mountPanel({ active: false });
    await flushPromises();
    expect(shellSessionMock).not.toHaveBeenCalled();

    await wrapper.setProps({ active: true });
    await flushPromises();
    await nextTick();

    expect(shellSessionMock).toHaveBeenCalledTimes(1);
    expect(shellSessionMock).toHaveBeenCalledWith('container-1', {
      command: 'sh',
      cols: 120,
      rows: 32,
    });
    expect(wrapper.get('[data-testid="t-tag"]').text()).toBe('Connected');
  });

  it('maps shell session API errors to disabled state and avoids websocket creation after rejection', async () => {
    shellSessionMock.mockRejectedValue(
      createApiError(
        'ops.container.error.shellDisabled',
        'Enable ops.container.shell.enabled in System Config before using Shell.',
      ),
    );

    const wrapper = mountPanel({ active: true });
    await flushPromises();
    await nextTick();

    expect(shellSessionMock).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).toContain('Shell Is Disabled');
    expect(wrapper.text()).toContain('Enable ops.container.shell.enabled in System Config before using Shell.');
    expect(wrapper.find('[data-testid="web-terminal-stub"]').exists()).toBe(false);
  });

  it('maps terminal transport errors to the localized websocket guidance', async () => {
    shellSessionMock.mockRejectedValue(new Error('Terminal transport error'));

    const wrapper = mountPanel({ active: true });
    await flushPromises();
    await nextTick();

    expect(wrapper.text()).toContain(
      'The terminal transport connection failed. Verify the frontend WebSocket proxy and server origin allowlist.',
    );
  });

  it('prefers canonical shell message keys for websocket error localization', async () => {
    shellSessionMock.mockResolvedValue({
      websocket_url: '/api/ops/containers/container-1/shell/ws?ticket=opaque-ticket',
    });

    const wrapper = mountPanel({ active: true });
    await flushPromises();
    await nextTick();

    const terminal = wrapper.getComponent({ name: 'WebTerminalStub' });
    terminal.vm.$emit('message', {
      type: 'error',
      message: 'The current user cannot open a container shell session',
      messageKey: 'ops.container.error.shellForbidden',
    });
    await flushPromises();

    expect(wrapper.text()).toContain('Required Permission: ops.container.shell');
  });

  it('only falls back for exact known shell transport messages', async () => {
    shellSessionMock.mockRejectedValue(new Error('container shell ticket expired'));

    const wrapper = mountPanel({ active: true });
    await flushPromises();
    await nextTick();

    expect(wrapper.text()).toContain('The shell session ticket expired. Reconnect to continue.');
  });

  it('reconnects by requesting a fresh shell session', async () => {
    shellSessionMock.mockResolvedValue({
      websocket_url: '/api/ops/containers/container-1/shell/ws?ticket=opaque-ticket',
    });

    const wrapper = mountPanel({ active: true });
    await flushPromises();
    await nextTick();

    expect(shellSessionMock).toHaveBeenCalledTimes(1);

    await wrapper.get('button').trigger('click');
    await flushPromises();
    await nextTick();

    expect(shellSessionMock).toHaveBeenCalledTimes(2);
    expect(terminalStubState.disconnectCalls).toBeGreaterThanOrEqual(1);
  });

  it('disconnects when the tab deactivates after a live session', async () => {
    shellSessionMock.mockResolvedValue({
      websocket_url: '/api/ops/containers/container-1/shell/ws?ticket=opaque-ticket',
    });

    const wrapper = mountPanel({ active: true });
    await flushPromises();
    await nextTick();
    expect(shellSessionMock).toHaveBeenCalledTimes(1);

    await wrapper.setProps({ active: false });
    await flushPromises();

    expect(terminalStubState.disconnectCalls).toBeGreaterThanOrEqual(1);
  });
});
