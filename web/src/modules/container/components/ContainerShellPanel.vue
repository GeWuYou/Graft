<template>
  <section class="container-shell-panel" data-testid="container-shell-panel">
    <header class="container-shell-panel__header">
      <div class="container-shell-panel__title-block">
        <div class="container-shell-panel__title-row">
          <t-icon name="terminal-window" />
          <h3>{{ t('container.detail.shell.title') }}</h3>
          <t-tag :theme="statusTheme" variant="light-outline" size="small">
            {{ statusLabel }}
          </t-tag>
        </div>
        <p>{{ t('container.detail.shell.description') }}</p>
      </div>
      <div class="container-shell-panel__actions">
        <t-select v-model="selectedCommand" class="container-shell-panel__command" :options="shellOptions" />
        <t-button
          theme="primary"
          variant="outline"
          :loading="sessionLoading"
          :disabled="reconnectDisabled"
          @click="handleReconnect"
        >
          {{ t('container.detail.shell.reconnect') }}
        </t-button>
      </div>
    </header>

    <t-alert v-if="availabilityState === 'disabled'" class="container-shell-panel__alert" theme="warning">
      <template #title>{{ t('container.detail.shell.disabled') }}</template>
      {{ t('container.detail.shell.disabledHint') }}
    </t-alert>

    <t-alert v-else-if="availabilityState === 'forbidden'" class="container-shell-panel__alert" theme="error">
      <template #title>{{ t('container.detail.shell.forbidden') }}</template>
      {{ t('container.detail.shell.forbiddenHint') }}
    </t-alert>

    <t-alert v-else-if="availabilityState === 'not-running'" class="container-shell-panel__alert" theme="info">
      <template #title>{{ t('container.detail.shell.notRunning') }}</template>
      {{ t('container.detail.shell.notRunningHint') }}
    </t-alert>

    <div v-else class="container-shell-panel__terminal">
      <web-terminal
        ref="terminalRef"
        :model-value="terminalActive"
        :connector="connector"
        :auto-connect="false"
        :connecting-description="t('container.detail.shell.connectingHint')"
        :connecting-title="t('container.detail.shell.connecting')"
        :disconnected-description="displayDisconnectedDescription"
        :disconnected-title="t('container.detail.shell.disconnected')"
        :empty-description="t('container.detail.shell.emptyHint')"
        :empty-title="t('container.detail.shell.empty')"
        :error-description="displayErrorDescription"
        :error-title="t('container.detail.shell.connectionFailed')"
        @close="handleTerminalClose"
        @message="handleTerminalMessage"
        @state-change="handleStateChange"
      />
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import type {
  TerminalConnectionState,
  TerminalLifecycleCloseReason,
  TerminalServerMessage,
  TerminalSessionConnector,
} from '@/shared/components/terminal/terminal-types';
import WebTerminal from '@/shared/components/terminal/WebTerminal.vue';
import { localizedApiErrorMessage } from '@/shared/localized-api-error';
import { getPermissionStore } from '@/store';
import type { ApiRequestError } from '@/types/axios';
import { isApiRequestError } from '@/utils/request';

import { postContainerShellSession } from '../api/container';
import { CONTAINER_PERMISSION_CODE } from '../contract/permissions';
import type { ContainerState } from '../types/container';

type ShellAvailabilityState = 'ready' | 'disabled' | 'forbidden' | 'not-running';
type ServerAvailabilityState = 'unknown' | 'ready' | 'disabled' | 'forbidden' | 'not-running';

const SHELL_DISABLED_MESSAGE_KEY = 'ops.container.error.shellDisabled';
const SHELL_FORBIDDEN_MESSAGE_KEY = 'ops.container.error.shellForbidden';
const SHELL_TICKET_EXPIRED_MESSAGE_KEY = 'ops.container.error.shellTicketExpired';
const SHELL_TICKET_USED_MESSAGE_KEY = 'ops.container.error.shellTicketUsed';
const SHELL_NOT_RUNNING_MESSAGE_KEY = 'ops.container.error.shellContainerNotRunning';
const SHELL_ORIGIN_DENIED_MESSAGE_KEY = 'ops.container.error.shellOriginDenied';

const SHELL_MESSAGE_KEY_TO_LOCALE_KEY: Record<string, string> = {
  [SHELL_DISABLED_MESSAGE_KEY]: 'container.detail.shell.disabledHint',
  [SHELL_FORBIDDEN_MESSAGE_KEY]: 'container.detail.shell.forbiddenHint',
  [SHELL_TICKET_EXPIRED_MESSAGE_KEY]: 'container.detail.shell.ticketExpired',
  [SHELL_TICKET_USED_MESSAGE_KEY]: 'container.detail.shell.ticketUsed',
  [SHELL_NOT_RUNNING_MESSAGE_KEY]: 'container.detail.shell.notRunningHint',
  [SHELL_ORIGIN_DENIED_MESSAGE_KEY]: 'container.detail.shell.originDenied',
};

const SHELL_FALLBACK_MESSAGE_TO_LOCALE_KEY: Record<string, string> = {
  'container shell disabled': 'container.detail.shell.disabledHint',
  'container shell forbidden': 'container.detail.shell.forbiddenHint',
  'container shell ticket invalid': 'container.detail.shell.sessionFailed',
  'container shell ticket expired': 'container.detail.shell.ticketExpired',
  'container shell ticket used': 'container.detail.shell.ticketUsed',
  'container shell origin denied': 'container.detail.shell.originDenied',
  'container shell command not found': 'container.detail.shell.sessionFailed',
  'container shell session failed': 'container.detail.shell.sessionFailed',
  'unsupported terminal control message': 'container.detail.shell.shellUnsupportedControlMessage',
  'terminal transport error': 'container.detail.shell.transportError',
};

const props = defineProps<{
  active: boolean;
  containerId: string;
  containerState?: ContainerState | null;
}>();

const { t } = useI18n();
const permissionStore = getPermissionStore();

const terminalRef = ref<InstanceType<typeof WebTerminal> | null>(null);
const terminalActive = ref(false);
const terminalState = ref<TerminalConnectionState>('idle');
const terminalError = ref('');
const sessionLoading = ref(false);
const selectedCommand = ref<'sh' | 'bash' | 'ash'>('sh');
const serverAvailability = ref<ServerAvailabilityState>('unknown');

const shellOptions = computed(() => [
  { label: t('container.detail.shell.commands.sh'), value: 'sh' },
  { label: t('container.detail.shell.commands.bash'), value: 'bash' },
  { label: t('container.detail.shell.commands.ash'), value: 'ash' },
]);

const hasShellPermission = computed(() => permissionStore.hasPermission(CONTAINER_PERMISSION_CODE.SHELL));
const isRunning = computed(() => props.containerState === 'running');
const availabilityState = computed<ShellAvailabilityState>(() => {
  if (serverAvailability.value === 'disabled') return 'disabled';
  if (serverAvailability.value === 'forbidden') return 'forbidden';
  if (serverAvailability.value === 'not-running') return 'not-running';
  if (!hasShellPermission.value) return 'forbidden';
  if (!isRunning.value) return 'not-running';
  return 'ready';
});

const reconnectDisabled = computed(() => availabilityState.value !== 'ready');
const statusTheme = computed(() => {
  if (terminalState.value === 'connected') return 'success';
  if (terminalState.value === 'connecting') return 'warning';
  if (terminalState.value === 'error') return 'danger';
  return 'default';
});
const statusLabel = computed(() => {
  if (terminalState.value === 'connected') return t('container.detail.shell.connected');
  if (terminalState.value === 'connecting') return t('container.detail.shell.connecting');
  if (terminalState.value === 'error') return t('container.detail.shell.connectionFailed');
  return t('container.detail.shell.disconnected');
});
const displayDisconnectedDescription = computed(() => {
  if (terminalError.value) {
    return terminalError.value;
  }
  return t('container.detail.shell.connectionClosed');
});
const displayErrorDescription = computed(() => terminalError.value || t('container.detail.shell.sessionFailed'));

const connector: TerminalSessionConnector = {
  async open(context) {
    sessionLoading.value = true;
    terminalError.value = '';
    serverAvailability.value = 'ready';
    try {
      const session = await postContainerShellSession(props.containerId, {
        command: selectedCommand.value,
        cols: context.cols,
        rows: context.rows,
      });
      return {
        url: toWebSocketUrl(session.websocket_url),
      };
    } catch (error) {
      const message = resolveShellErrorMessage(error);
      terminalError.value = message;
      throw new Error(message);
    } finally {
      sessionLoading.value = false;
    }
  },
};

watch(
  () => props.active,
  async (active) => {
    if (!active) {
      terminalActive.value = false;
      terminalRef.value?.disconnect();
      return;
    }
    if (availabilityState.value !== 'ready') {
      return;
    }
    await nextTick();
    terminalActive.value = true;
  },
  { immediate: true },
);

watch(
  () => availabilityState.value,
  async (state) => {
    if (state !== 'ready') {
      terminalActive.value = false;
      terminalRef.value?.disconnect();
      return;
    }
    if (props.active) {
      await nextTick();
      terminalActive.value = true;
    }
  },
);

onBeforeUnmount(() => {
  terminalActive.value = false;
  terminalRef.value?.disconnect();
});

async function handleReconnect() {
  if (availabilityState.value !== 'ready') {
    return;
  }
  terminalActive.value = false;
  await nextTick();
  terminalActive.value = true;
}

function handleTerminalClose(reason: TerminalLifecycleCloseReason) {
  if (reason === 'connect_error' || reason === 'session_error') {
    terminalState.value = 'error';
  }
}

function handleTerminalMessage(message: TerminalServerMessage) {
  if (message.type === 'error') {
    terminalError.value = localizeShellServerMessage(message);
  }
}

function handleStateChange(state: TerminalConnectionState) {
  terminalState.value = state;
  if (state === 'connected') {
    terminalError.value = '';
  }
}

function resolveShellErrorMessage(error: unknown) {
  if (isApiRequestError(error)) {
    applyServerAvailability(error);
    return localizeShellApiError(error);
  }
  if (error instanceof Error && error.message.trim()) {
    return localizeShellMessage(error.message);
  }
  return t('container.detail.shell.sessionFailed');
}

function localizeShellApiError(error: ApiRequestError) {
  return (
    localizeShellMessage(error.message, error.messageKey) ||
    localizedApiErrorMessage(t, error.messageKey, error.message) ||
    t('container.detail.shell.sessionFailed')
  );
}

function localizeShellServerMessage(message: Extract<TerminalServerMessage, { type: 'error' }>) {
  return (
    localizeShellMessage(message.message, message.messageKey) ||
    localizedApiErrorMessage(t, message.messageKey, message.message) ||
    t('container.detail.shell.sessionFailed')
  );
}

function localizeShellMessage(message: string, messageKey?: string) {
  const localizedByKey = localizeShellMessageKey(messageKey);
  if (localizedByKey) {
    return localizedByKey;
  }
  const normalized = message.trim().toLowerCase();
  const localeKey = SHELL_FALLBACK_MESSAGE_TO_LOCALE_KEY[normalized];
  if (localeKey) {
    return t(localeKey);
  }
  return message.trim() || '';
}

function localizeShellMessageKey(messageKey?: string) {
  if (!messageKey) {
    return '';
  }
  const localeKey = SHELL_MESSAGE_KEY_TO_LOCALE_KEY[messageKey];
  if (!localeKey) {
    return '';
  }
  return t(localeKey);
}

function applyServerAvailability(error: ApiRequestError) {
  if (error.messageKey === SHELL_DISABLED_MESSAGE_KEY) {
    serverAvailability.value = 'disabled';
    return;
  }
  if (error.messageKey === SHELL_FORBIDDEN_MESSAGE_KEY) {
    serverAvailability.value = 'forbidden';
    return;
  }
  if (error.messageKey === SHELL_ORIGIN_DENIED_MESSAGE_KEY) {
    serverAvailability.value = 'forbidden';
    return;
  }
  if (error.messageKey === SHELL_NOT_RUNNING_MESSAGE_KEY) {
    serverAvailability.value = 'not-running';
    return;
  }
  serverAvailability.value = 'ready';
}

function toWebSocketUrl(relativePath: string) {
  const base = new URL(window.location.href);
  const protocol = base.protocol === 'https:' ? 'wss:' : 'ws:';
  return new URL(relativePath, `${protocol}//${base.host}`).toString();
}

defineExpose({
  reconnect: handleReconnect,
});
</script>
<style scoped lang="less">
.container-shell-panel {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  height: 100%;
  min-height: 0;
  min-width: 0;
}

.container-shell-panel__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.container-shell-panel__title-block {
  display: grid;
  gap: var(--graft-density-gap-8);
}

.container-shell-panel__title-row {
  align-items: center;
  color: var(--td-text-color-primary);
  display: flex;
  gap: var(--graft-density-gap-8);
}

.container-shell-panel__title-row h3 {
  font: var(--td-font-title-medium);
  margin: 0;
}

.container-shell-panel__title-block p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.container-shell-panel__actions {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
}

.container-shell-panel__command {
  min-width: 120px;
}

.container-shell-panel__alert {
  flex: 0 0 auto;
}

.container-shell-panel__terminal {
  display: flex;
  flex: 1 1 auto;
  height: var(--container-shell-terminal-height);
  min-height: var(--container-shell-terminal-height);
  min-width: 0;
  overflow: hidden;
}

@media (width <= 1024px) {
  .container-shell-panel__header {
    align-items: stretch;
    flex-direction: column;
  }

  .container-shell-panel__actions {
    justify-content: flex-start;
  }
}
</style>
