<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <teleport to="body">
    <transition name="announcement-read-panel">
      <div v-if="visible && announcement" class="announcement-read-panel">
        <button
          class="announcement-read-panel__overlay"
          type="button"
          :aria-label="t('announcement.readPanel.close')"
          @click="emitClose"
        />
        <section class="announcement-read-panel__surface" role="dialog" aria-modal="true" tabindex="-1">
          <header class="announcement-read-panel__header">
            <div class="announcement-read-panel__title-row">
              <h2>{{ announcement.title }}</h2>
              <t-button
                shape="square"
                theme="default"
                variant="text"
                :aria-label="t('announcement.readPanel.close')"
                @click="emitClose"
              >
                <t-icon name="close" />
              </t-button>
            </div>
            <div class="announcement-read-panel__meta">
              <t-tag :theme="announcement.unread ? 'primary' : 'default'" variant="light">
                {{ announcement.unreadLabel }}
              </t-tag>
              <t-tag :theme="announcement.levelTheme" variant="light">
                {{ announcement.levelLabel }}
              </t-tag>
              <t-tag v-if="announcement.pinned" theme="primary" variant="light">
                {{ announcement.pinnedLabel }}
              </t-tag>
              <span>{{ t('announcement.readPanel.publishAt') }} {{ announcement.publishAtLabel }}</span>
            </div>
          </header>

          <main class="announcement-read-panel__body">
            <markdown-viewer :source="announcement.content" />
          </main>

          <footer class="announcement-read-panel__footer">
            <t-button theme="default" variant="outline" @click="emitClose">
              {{ t('announcement.readPanel.viewLater') }}
            </t-button>
            <t-button v-if="source !== 'center'" theme="default" variant="outline" @click="emitOpenCenter">
              {{ t('announcement.readPanel.openCenter') }}
            </t-button>
            <t-button v-if="announcement.unread" theme="primary" :loading="markingRead" @click="emitMarkRead">
              {{ t('announcement.readPanel.markRead') }}
            </t-button>
          </footer>
        </section>
      </div>
    </transition>
  </teleport>
</template>
<script setup lang="ts">
import { onBeforeUnmount, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { MarkdownViewer } from '@/shared/components/markdown';

import type { AnnouncementViewModel } from '../domain/announcement-presenter';

const props = defineProps<{
  announcement: AnnouncementViewModel | null;
  markingRead?: boolean;
  source: 'center' | 'header' | 'popup';
  visible: boolean;
}>();

const emit = defineEmits<{
  close: [];
  'mark-read': [];
  'open-center': [];
}>();

const { t } = useI18n();

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      window.addEventListener('keydown', handleWindowKeydown);
    } else {
      window.removeEventListener('keydown', handleWindowKeydown);
    }
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleWindowKeydown);
});

function handleWindowKeydown(event: KeyboardEvent) {
  if (props.visible && event.key === 'Escape') {
    emitClose();
  }
}

function emitClose() {
  emit('close');
}

function emitMarkRead() {
  emit('mark-read');
}

function emitOpenCenter() {
  emit('open-center');
}
</script>
<style scoped lang="less">
.announcement-read-panel {
  inset: 0;
  position: fixed;
  z-index: 2600;
}

.announcement-read-panel__overlay {
  background: color-mix(in srgb, var(--td-bg-color-page) 72%, transparent);
  border: 0;
  cursor: default;
  inset: 0;
  padding: 0;
  position: absolute;
}

.announcement-read-panel__surface {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  box-shadow: var(--td-shadow-3);
  display: flex;
  flex-direction: column;
  left: 50%;
  max-height: calc(100vh - 96px);
  min-height: 240px;
  overflow: hidden;
  position: absolute;
  top: var(--graft-density-gap-24);
  transform: translateX(-50%);
  width: min(960px, calc(100vw - 48px));
}

.announcement-read-panel__header,
.announcement-read-panel__footer {
  flex: 0 0 auto;
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-xl);
}

.announcement-read-panel__header {
  border-bottom: 1px solid var(--td-component-stroke);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
}

.announcement-read-panel__title-row {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.announcement-read-panel__title-row h2 {
  -webkit-box-orient: vertical;
  color: var(--td-text-color-primary);
  display: -webkit-box;
  font: var(--td-font-title-large);
  -webkit-line-clamp: 2;
  margin: 0;
  min-width: 0;
  overflow: hidden;
}

.announcement-read-panel__meta {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  flex-wrap: wrap;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-8);
}

.announcement-read-panel__body {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-xl);
  scrollbar-color: var(--td-scrollbar-color) transparent;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
}

.announcement-read-panel__body::-webkit-scrollbar {
  height: 8px;
  width: 8px;
}

.announcement-read-panel__body::-webkit-scrollbar-track {
  background: transparent;
}

.announcement-read-panel__body::-webkit-scrollbar-thumb {
  background-color: var(--td-scrollbar-color);
  border-radius: var(--td-radius-round);
}

.announcement-read-panel__footer {
  align-items: center;
  background: color-mix(in srgb, var(--td-bg-color-container) 94%, var(--td-bg-color-page));
  border-top: 1px solid var(--td-component-stroke);
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-10);
  justify-content: flex-end;
}

.announcement-read-panel-enter-active,
.announcement-read-panel-leave-active {
  transition: opacity 180ms ease;
}

.announcement-read-panel-enter-active .announcement-read-panel__surface,
.announcement-read-panel-leave-active .announcement-read-panel__surface {
  transition:
    opacity 180ms ease,
    transform 180ms ease;
}

.announcement-read-panel-enter-from,
.announcement-read-panel-leave-to {
  opacity: 0;
}

.announcement-read-panel-enter-from .announcement-read-panel__surface,
.announcement-read-panel-leave-to .announcement-read-panel__surface {
  opacity: 0;
  transform: translate(-50%, -18px);
}

@media (width <= 768px) {
  .announcement-read-panel__surface {
    max-height: calc(100vh - 48px);
    top: var(--graft-density-gap-12);
    width: calc(100vw - 24px);
  }

  .announcement-read-panel__header,
  .announcement-read-panel__body,
  .announcement-read-panel__footer {
    padding-left: var(--td-comp-paddingLR-l);
    padding-right: var(--td-comp-paddingLR-l);
  }

  .announcement-read-panel__footer {
    align-items: stretch;
    flex-direction: column-reverse;
  }
}
</style>
