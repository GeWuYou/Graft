<template>
  <section
    :class="[
      'stream-viewport-state-surface',
      `stream-viewport-state-surface--${state}`,
      {
        'stream-viewport-state-surface--busy': effectiveShowBusy,
        'stream-viewport-state-surface--cursor': effectiveShowCursor,
      },
    ]"
    :aria-label="ariaLabel || title || badgeLabel"
    aria-live="polite"
    role="status"
  >
    <div class="stream-viewport-state-surface__chrome" aria-hidden="true">
      <div class="stream-viewport-state-surface__chrome-lights">
        <span class="stream-viewport-state-surface__chrome-light"></span>
        <span class="stream-viewport-state-surface__chrome-light"></span>
        <span class="stream-viewport-state-surface__chrome-light"></span>
      </div>
      <span class="stream-viewport-state-surface__chrome-channel"></span>
    </div>

    <div class="stream-viewport-state-surface__viewport">
      <div class="stream-viewport-state-surface__faux-lines" aria-hidden="true">
        <div
          v-for="(width, index) in fauxLineWidths"
          :key="`${state}-${index}`"
          class="stream-viewport-state-surface__faux-line"
        >
          <span class="stream-viewport-state-surface__faux-line-marker"></span>
          <span class="stream-viewport-state-surface__faux-line-bar" :style="{ width }"></span>
        </div>
      </div>

      <div class="stream-viewport-state-surface__panel">
        <div v-if="badgeLabel || (effectiveShowBusy && busyLabel)" class="stream-viewport-state-surface__status-row">
          <span v-if="badgeLabel" class="stream-viewport-state-surface__badge">
            <span class="stream-viewport-state-surface__badge-dot"></span>
            {{ badgeLabel }}
          </span>
          <span v-if="effectiveShowBusy && busyLabel" class="stream-viewport-state-surface__busy-copy">
            <span class="stream-viewport-state-surface__busy-indicator"></span>
            {{ busyLabel }}
          </span>
        </div>

        <div v-if="title || effectiveShowCursor" class="stream-viewport-state-surface__title-row">
          <h3 v-if="title" class="stream-viewport-state-surface__title">{{ title }}</h3>
          <span v-if="effectiveShowCursor" class="stream-viewport-state-surface__cursor" aria-hidden="true"></span>
        </div>

        <p v-if="description" class="stream-viewport-state-surface__description">{{ description }}</p>

        <div v-if="hint" class="stream-viewport-state-surface__hint-row">
          <span class="stream-viewport-state-surface__prompt-marker" aria-hidden="true"></span>
          <span class="stream-viewport-state-surface__hint">{{ hint }}</span>
        </div>
      </div>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import {
  isStreamViewportBusyState,
  isStreamViewportCursorState,
  type StreamViewportState,
} from './stream-viewport-state';

const props = withDefaults(
  defineProps<{
    state: StreamViewportState;
    ariaLabel?: string;
    badgeLabel?: string;
    busyLabel?: string;
    description?: string;
    fauxLineCount?: number;
    hint?: string;
    showBusy?: boolean | null;
    showCursor?: boolean | null;
    title?: string;
  }>(),
  {
    ariaLabel: '',
    badgeLabel: '',
    busyLabel: '',
    description: '',
    fauxLineCount: 7,
    hint: '',
    showBusy: null,
    showCursor: null,
    title: '',
  },
);

const fauxLinePattern = [100, 98, 94, 100, 90, 100, 86, 96] as const;

const effectiveShowBusy = computed(() => props.showBusy ?? isStreamViewportBusyState(props.state));
const effectiveShowCursor = computed(() => props.showCursor ?? isStreamViewportCursorState(props.state));
const fauxLineWidths = computed(() =>
  Array.from({ length: normalizeFauxLineCount(props.fauxLineCount) }, (_, index) => {
    return `${fauxLinePattern[index % fauxLinePattern.length]}%`;
  }),
);

function normalizeFauxLineCount(value: number) {
  return Math.min(Math.max(Math.trunc(value) || 0, 4), 10);
}
</script>
<style scoped lang="less">
.stream-viewport-state-surface {
  --stream-viewport-accent: var(--td-brand-color-6);
  --stream-viewport-accent-soft: #89c2ff;
  --stream-viewport-copy: #d9e2ec;
  --stream-viewport-copy-muted: rgb(217 226 236 / 72%);
  --stream-viewport-panel: rgb(9 14 21 / 84%);

  background:
    radial-gradient(
      circle at top right,
      color-mix(in srgb, var(--stream-viewport-accent) 16%, transparent),
      transparent 34%
    ),
    linear-gradient(180deg, #111923 0%, #0a1017 100%);
  border: 1px solid color-mix(in srgb, var(--stream-viewport-accent) 18%, rgb(129 150 173 / 26%));
  border-radius: var(--td-radius-large);
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 4%),
    0 14px 40px rgb(7 11 16 / 20%);
  color: var(--stream-viewport-copy);
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-height: clamp(240px, 38vh, 360px);
  min-width: 0;
  overflow: hidden;
  position: relative;
}

.stream-viewport-state-surface--idle,
.stream-viewport-state-surface--empty {
  --stream-viewport-accent: var(--td-text-color-placeholder);
  --stream-viewport-accent-soft: #95a9bf;
}

.stream-viewport-state-surface--paused,
.stream-viewport-state-surface--disconnected {
  --stream-viewport-accent: var(--td-warning-color-6);
  --stream-viewport-accent-soft: #ffda94;
}

.stream-viewport-state-surface--error {
  --stream-viewport-accent: var(--td-error-color-6);
  --stream-viewport-accent-soft: #ffb7aa;
}

.stream-viewport-state-surface__chrome {
  align-items: center;
  background: linear-gradient(180deg, rgb(255 255 255 / 6%), rgb(255 255 255 / 1%));
  border-bottom: 1px solid rgb(255 255 255 / 6%);
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  padding: var(--graft-density-gap-12) var(--graft-density-gap-16);
}

.stream-viewport-state-surface__chrome-lights {
  display: flex;
  gap: var(--graft-density-gap-6);
}

.stream-viewport-state-surface__chrome-light {
  background: color-mix(in srgb, var(--stream-viewport-accent-soft) 78%, rgb(255 255 255 / 18%));
  border-radius: 999px;
  display: block;
  height: 10px;
  opacity: 0.72;
  width: 10px;
}

.stream-viewport-state-surface__chrome-channel {
  background: linear-gradient(
    90deg,
    color-mix(in srgb, var(--stream-viewport-accent) 30%, transparent),
    rgb(255 255 255 / 6%)
  );
  border-radius: 999px;
  display: block;
  height: 8px;
  max-width: 200px;
  width: 28%;
}

.stream-viewport-state-surface__viewport {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  position: relative;
}

.stream-viewport-state-surface__viewport::after {
  background: linear-gradient(180deg, rgb(0 0 0 / 0%) 0%, rgb(0 0 0 / 18%) 100%);
  content: '';
  inset: 0;
  pointer-events: none;
  position: absolute;
}

.stream-viewport-state-surface__faux-lines {
  display: grid;
  gap: var(--graft-density-gap-12);
  inset: 0;
  opacity: 0.78;
  padding: calc(var(--graft-density-gap-24) + var(--graft-density-gap-4)) var(--graft-density-gap-20)
    calc(var(--graft-density-gap-24) + var(--graft-density-gap-4));
  position: absolute;
}

.stream-viewport-state-surface__faux-line {
  align-items: center;
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: 10px minmax(0, 1fr);
}

.stream-viewport-state-surface__faux-line-marker {
  background: color-mix(in srgb, var(--stream-viewport-accent-soft) 78%, rgb(255 255 255 / 16%));
  border-radius: 999px;
  display: block;
  height: 10px;
  width: 10px;
}

.stream-viewport-state-surface__faux-line-bar {
  background: linear-gradient(
    90deg,
    color-mix(in srgb, var(--stream-viewport-accent) 26%, rgb(255 255 255 / 12%)) 0%,
    rgb(255 255 255 / 4%) 100%
  );
  border-radius: 999px;
  display: block;
  height: 14px;
  max-width: 100%;
}

.stream-viewport-state-surface__panel {
  align-self: end;
  background: linear-gradient(180deg, rgb(8 12 18 / 18%), var(--stream-viewport-panel));
  border-top: 1px solid color-mix(in srgb, var(--stream-viewport-accent) 18%, rgb(255 255 255 / 8%));
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  justify-content: flex-end;
  margin-top: auto;
  min-width: 0;
  padding: var(--graft-density-gap-24) var(--graft-density-gap-20);
  position: relative;
  z-index: 1;
}

.stream-viewport-state-surface__status-row,
.stream-viewport-state-surface__busy-copy,
.stream-viewport-state-surface__badge,
.stream-viewport-state-surface__hint-row,
.stream-viewport-state-surface__title-row {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.stream-viewport-state-surface__badge,
.stream-viewport-state-surface__busy-copy,
.stream-viewport-state-surface__description,
.stream-viewport-state-surface__hint {
  font-family: var(--td-font-family-monospace);
}

.stream-viewport-state-surface__badge,
.stream-viewport-state-surface__busy-copy {
  color: var(--stream-viewport-copy-muted);
  font-size: 12px;
  letter-spacing: 0.04em;
  line-height: 1.4;
}

.stream-viewport-state-surface__badge-dot,
.stream-viewport-state-surface__busy-indicator {
  background: var(--stream-viewport-accent-soft);
  border-radius: 999px;
  display: inline-flex;
  flex: 0 0 auto;
  height: 9px;
  width: 9px;
}

.stream-viewport-state-surface__busy-indicator {
  animation: stream-viewport-pulse 1.2s ease-in-out infinite;
  box-shadow: 0 0 0 5px color-mix(in srgb, var(--stream-viewport-accent) 16%, transparent);
}

.stream-viewport-state-surface__title {
  color: var(--stream-viewport-copy);
  font-family: var(--td-font-family-monospace);
  font-size: 18px;
  font-weight: 600;
  line-height: 1.45;
  margin: 0;
  min-width: 0;
}

.stream-viewport-state-surface__description {
  color: var(--stream-viewport-copy-muted);
  font-size: 13px;
  line-height: 1.7;
  margin: 0;
  max-width: 68ch;
  min-width: 0;
}

.stream-viewport-state-surface__prompt-marker {
  background: color-mix(in srgb, var(--stream-viewport-accent-soft) 84%, rgb(255 255 255 / 22%));
  border-radius: 3px;
  display: inline-flex;
  flex: 0 0 auto;
  height: 12px;
  width: 12px;
}

.stream-viewport-state-surface__hint {
  color: color-mix(in srgb, var(--stream-viewport-copy) 82%, transparent);
  font-size: 12px;
  line-height: 1.6;
  min-width: 0;
}

.stream-viewport-state-surface__cursor {
  animation: stream-viewport-cursor-blink 1.1s steps(1) infinite;
  background: var(--stream-viewport-accent-soft);
  border-radius: 1px;
  display: inline-flex;
  flex: 0 0 auto;
  height: 18px;
  width: 10px;
}

@media (width <= 767px) {
  .stream-viewport-state-surface__chrome,
  .stream-viewport-state-surface__panel {
    padding-inline: var(--graft-density-gap-16);
  }

  .stream-viewport-state-surface__faux-lines {
    padding-inline: var(--graft-density-gap-16);
  }

  .stream-viewport-state-surface__title {
    font-size: 16px;
  }
}

@keyframes stream-viewport-pulse {
  0%,
  100% {
    opacity: 0.42;
    transform: scale(0.94);
  }

  50% {
    opacity: 1;
    transform: scale(1);
  }
}

@keyframes stream-viewport-cursor-blink {
  0%,
  45% {
    opacity: 1;
  }

  50%,
  100% {
    opacity: 0;
  }
}
</style>
