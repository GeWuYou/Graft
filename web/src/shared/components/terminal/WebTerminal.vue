<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="web-terminal" :data-state="connectionState">
    <div class="web-terminal__surface" :class="{ 'web-terminal__surface--focused': focused }">
      <div v-if="showOverlay" class="web-terminal__overlay" role="status" aria-live="polite">
        <slot name="status" :state="connectionState" :error-message="displayError">
          <div class="web-terminal__overlay-card">
            <strong>{{ overlayTitle }}</strong>
            <span>{{ overlayDescription }}</span>
          </div>
        </slot>
      </div>
      <div ref="hostRef" class="web-terminal__host" tabindex="0" @focus="focused = true" @blur="focused = false" />
    </div>
  </div>
</template>
<script setup lang="ts">
import '@xterm/xterm/css/xterm.css';

import { FitAddon } from '@xterm/addon-fit';
import { SearchAddon } from '@xterm/addon-search';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { Terminal } from '@xterm/xterm';
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';

import { createTerminalThemeOptions } from './terminal-theme';
import type {
  TerminalConnectionState,
  TerminalLifecycleCloseReason,
  TerminalResizePayload,
  TerminalServerMessage,
  TerminalSessionConnector,
} from './terminal-types';
import { useTerminalSession } from './useTerminalSession';

const props = withDefaults(
  defineProps<{
    autoConnect?: boolean;
    connector: TerminalSessionConnector;
    connectingDescription?: string;
    connectingTitle?: string;
    disconnectedDescription?: string;
    disconnectedTitle?: string;
    emptyDescription?: string;
    emptyTitle?: string;
    errorDescription?: string;
    errorTitle?: string;
    modelValue?: boolean;
  }>(),
  {
    autoConnect: false,
    connectingDescription: '',
    connectingTitle: '',
    disconnectedDescription: '',
    disconnectedTitle: '',
    emptyDescription: '',
    emptyTitle: '',
    errorDescription: '',
    errorTitle: '',
    modelValue: false,
  },
);

const emit = defineEmits<{
  close: [reason: TerminalLifecycleCloseReason];
  message: [message: TerminalServerMessage];
  'state-change': [state: TerminalConnectionState];
}>();

const hostRef = ref<HTMLElement | null>(null);
const focused = ref(false);
let terminal: Terminal | null = null;
let fitAddon: FitAddon | null = null;
let searchAddon: SearchAddon | null = null;
let webLinksAddon: WebLinksAddon | null = null;

let resizeObserver: ResizeObserver | null = null;

const session = useTerminalSession({
  connector: props.connector,
  onClose: (reason) => emit('close', reason),
  onMessage: handleServerMessage,
  onOpen: () => {
    queueFitAndResize();
    focusTerminal();
  },
  onStateChange: (state) => emit('state-change', state),
});

const connectionState = computed(() => session.state.value);
const displayError = computed(() => session.lastError.value || props.errorDescription);
const showOverlay = computed(() => connectionState.value !== 'connected');
const overlayTitle = computed(() => {
  if (connectionState.value === 'connecting') return props.connectingTitle || props.emptyTitle || '';
  if (connectionState.value === 'error') return props.errorTitle || props.disconnectedTitle || '';
  if (connectionState.value === 'disconnected') return props.disconnectedTitle || props.emptyTitle || '';
  return props.emptyTitle || '';
});
const overlayDescription = computed(() => {
  if (connectionState.value === 'connecting') return props.connectingDescription || props.emptyDescription || '';
  if (connectionState.value === 'error') return displayError.value || props.errorDescription || '';
  if (connectionState.value === 'disconnected') {
    return props.disconnectedDescription || props.emptyDescription || '';
  }
  return props.emptyDescription || '';
});

watch(
  () => props.modelValue,
  (active) => {
    if (active) {
      void ensureConnected();
      return;
    }
    session.disconnect('manual_disconnect');
  },
);

onMounted(() => {
  if (!hostRef.value) {
    return;
  }
  terminal = new Terminal({
    ...createTerminalThemeOptions(),
  });
  fitAddon = new FitAddon();
  searchAddon = new SearchAddon();
  webLinksAddon = new WebLinksAddon();
  terminal.loadAddon(fitAddon);
  terminal.loadAddon(searchAddon);
  terminal.loadAddon(webLinksAddon);
  terminal.open(hostRef.value);
  terminal.onData((data) => {
    session.sendInput(data);
  });
  resizeObserver = new ResizeObserver(() => {
    queueFitAndResize();
  });
  resizeObserver.observe(hostRef.value);
  if (props.autoConnect || props.modelValue) {
    void ensureConnected();
  } else {
    queueFitAndResize();
  }
});

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  resizeObserver = null;
  session.disconnect('component_unmount');
  terminal?.dispose();
  terminal = null;
  fitAddon = null;
  searchAddon = null;
  webLinksAddon = null;
});

async function ensureConnected() {
  await nextTick();
  if (!terminal || !hostRef.value) {
    return;
  }
  const size = measureTerminal();
  try {
    await session.connect(size);
  } catch {
    // The session composable already propagates error state through reactive state and callbacks.
  }
}

function queueFitAndResize() {
  window.requestAnimationFrame(() => {
    if (!hostRef.value || !fitAddon || !terminal) {
      return;
    }
    try {
      fitAddon.fit();
      const size = measureTerminal();
      session.sendResize(size);
    } catch {
      // xterm fit can throw during transient unmounted/zero-size layout states.
    }
  });
}

function measureTerminal(): TerminalResizePayload {
  return {
    cols: terminal?.cols || 120,
    rows: terminal?.rows || 32,
  };
}

function handleServerMessage(message: TerminalServerMessage) {
  if (message.type === 'output') {
    terminal?.write(message.data);
  }
  emit('message', message);
}

function focusTerminal() {
  terminal?.focus();
  hostRef.value?.focus();
}

defineExpose({
  connect: ensureConnected,
  disconnect: session.disconnect,
  findNext: (value: string) => searchAddon?.findNext(value) ?? false,
  fit: queueFitAndResize,
  focus: focusTerminal,
  getState: () => connectionState.value,
});
</script>
<style scoped lang="less">
.web-terminal {
  display: flex;
  flex: 1 1 auto;
  height: 100%;
  min-height: 0;
  min-width: 0;
}

.web-terminal__surface {
  background:
    radial-gradient(circle at top right, rgb(80 140 255 / 12%), transparent 32%),
    linear-gradient(180deg, #111a24 0%, #0c131b 100%);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 30%, #0f1720 70%);
  border-radius: var(--td-radius-large);
  box-shadow: inset 0 1px 0 rgb(255 255 255 / 4%);
  display: flex;
  flex: 1 1 auto;
  height: 100%;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  position: relative;
}

.web-terminal__surface--focused {
  box-shadow:
    0 0 0 1px color-mix(in srgb, var(--td-brand-color-6) 55%, transparent),
    inset 0 1px 0 rgb(255 255 255 / 4%);
}

.web-terminal__host {
  flex: 1 1 auto;
  height: 100%;
  min-height: 0;
  min-width: 0;
  outline: none;
  padding: var(--graft-density-gap-16);
}

.web-terminal__host :deep(.xterm) {
  height: 100%;
}

.web-terminal__host :deep(.xterm-screen),
.web-terminal__host :deep(.xterm-helpers),
.web-terminal__host :deep(.xterm-viewport) {
  height: 100%;
}

.web-terminal__host :deep(.xterm-viewport) {
  border-radius: calc(var(--td-radius-medium) - 2px);
}

.web-terminal__overlay {
  align-items: center;
  backdrop-filter: blur(2px);
  background: rgb(8 12 18 / 48%);
  display: flex;
  inset: 0;
  justify-content: center;
  padding: var(--graft-density-gap-24);
  position: absolute;
  z-index: 2;
}

.web-terminal__overlay-card {
  align-items: flex-start;
  background: rgb(12 18 26 / 92%);
  border: 1px solid rgb(215 226 240 / 14%);
  border-radius: var(--td-radius-medium);
  color: #d9e2ec;
  display: grid;
  gap: var(--graft-density-gap-8);
  max-width: 420px;
  padding: var(--graft-density-gap-16) calc(var(--graft-density-gap-16) + var(--graft-density-gap-2));
}

.web-terminal__overlay-card strong {
  font: var(--td-font-title-small);
}

.web-terminal__overlay-card span {
  color: rgb(217 226 236 / 84%);
  font: var(--td-font-body-small);
}
</style>
