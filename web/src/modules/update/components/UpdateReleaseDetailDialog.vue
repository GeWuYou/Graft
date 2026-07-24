<template>
  <t-dialog
    :visible="visible"
    :header="t('update.preview.detail.title')"
    :footer="false"
    width="min(720px, calc(100vw - 32px))"
    @close="emit('update:visible', false)"
  >
    <template v-if="release">
      <header class="update-release-detail__header">
        <div>
          <strong>{{ release.version }}</strong>
          <p>{{ t('update.preview.detail.releaseDescription') }}</p>
        </div>
        <div class="update-release-detail__meta">
          <t-tag size="small" theme="success" variant="light">
            {{ t(`update.center.channels.${release.channel}`) }}
          </t-tag>
          <span>{{ t('update.preview.detail.publishedAt') }} {{ publishedAt }}</span>
        </div>
      </header>

      <main class="update-release-detail__body graft-scrollbar">
        <markdown-viewer :source="releaseNotes" />
        <section v-if="release.upgrade_notes" class="update-release-detail__upgrade-notes">
          <h3>{{ t('update.preview.detail.upgradeNotes') }}</h3>
          <p>{{ release.upgrade_notes }}</p>
        </section>
      </main>

      <footer class="update-release-detail__footer">
        <t-link v-if="canViewRelease" theme="primary" :href="releaseUrl" target="_blank" rel="noopener noreferrer">
          {{ t('update.preview.viewRelease') }}
        </t-link>
        <t-button theme="primary" @click="emit('update:visible', false)">
          {{ t('update.preview.detail.close') }}
        </t-button>
      </footer>
    </template>
  </t-dialog>
</template>
<script setup lang="ts">
// 详情弹窗只消费更新模块的发行快照，正文使用共享安全 Markdown 渲染器。
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { MarkdownViewer } from '@/shared/components/markdown';
import { formatLocaleDateTime } from '@/shared/observability';

import type { UpdateRelease } from '../types/update';

const props = defineProps<{
  release: UpdateRelease | null;
  visible: boolean;
}>();

const emit = defineEmits<{
  'update:visible': [value: boolean];
}>();

const { locale, t } = useI18n();
const releaseUrl = computed(() => props.release?.notes_url?.trim() ?? '');
const canViewRelease = computed(() => /^https:\/\//i.test(releaseUrl.value));
const releaseNotes = computed(() => props.release?.notes || t('update.center.release.notesEmpty'));
const publishedAt = computed(() =>
  props.release ? formatLocaleDateTime(props.release.published_at, locale.value) : '',
);
</script>
<style scoped lang="less">
.update-release-detail__header {
  border-bottom: 1px solid var(--td-component-stroke);
  display: grid;
  gap: var(--graft-density-gap-10);
  padding-bottom: var(--td-comp-paddingTB-l);
}

.update-release-detail__header strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  font-variant-numeric: tabular-nums;
}

.update-release-detail__header p,
.update-release-detail__meta {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: var(--td-comp-margin-xs) 0 0;
}

.update-release-detail__meta {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  margin-top: 0;
}

.update-release-detail__body {
  max-height: min(60vh, 560px);
  overflow: auto;
  padding: var(--td-comp-paddingTB-l) 0;
}

.update-release-detail__upgrade-notes {
  border-top: 1px solid var(--td-component-stroke);
  margin-top: var(--td-comp-margin-xl);
  padding-top: var(--td-comp-paddingTB-l);
}

.update-release-detail__upgrade-notes h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0 0 var(--td-comp-margin-s);
}

.update-release-detail__upgrade-notes p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  margin: 0;
  white-space: pre-line;
}

.update-release-detail__footer {
  align-items: center;
  border-top: 1px solid var(--td-component-stroke);
  display: flex;
  gap: var(--graft-density-gap-10);
  justify-content: flex-end;
  padding-top: var(--td-comp-paddingTB-l);
}
</style>
