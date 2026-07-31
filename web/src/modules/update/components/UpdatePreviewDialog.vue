<template>
  <t-dialog
    :visible="visible"
    :header="t('update.preview.title')"
    :cancel-btn="null"
    :confirm-btn="null"
    width="480px"
    @close="emit('update:visible', false)"
  >
    <template v-if="status">
      <dl class="update-preview__versions">
        <div>
          <dt>{{ t('update.preview.current') }}</dt>
          <dd>{{ status.current_version }}</dd>
        </div>
        <div v-if="availableRelease">
          <dt>{{ t('update.preview.available') }}</dt>
          <dd>{{ availableRelease.version }}</dd>
        </div>
      </dl>
      <template v-if="availableRelease">
        <h3>{{ t('update.preview.summary') }}</h3>
        <p class="update-preview__summary">{{ summary }}</p>
        <div class="update-preview__actions">
          <t-button variant="text" @click="emit('view-management')">{{ t('update.preview.viewManagement') }}</t-button>
          <t-button v-if="canStartUpgrade" theme="primary" @click="emit('start-upgrade')">{{
            t('update.preview.startUpgrade')
          }}</t-button>
        </div>
      </template>
      <p v-else class="update-preview__up-to-date">{{ t('update.preview.upToDate') }}</p>
    </template>
    <t-alert v-else theme="warning" :message="t('update.preview.unavailable')" />
  </t-dialog>
</template>
<script setup lang="ts">
// 预览弹窗只呈现 discovery snapshot；执行升级仍由调用者进入精确版本确认链路。
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { getAvailableUpdateRelease } from '../composables/releaseSelection';
import type { UpdateStatus } from '../types/update';

const props = defineProps<{ visible: boolean; status: UpdateStatus | null; canStartUpgrade: boolean }>();
const emit = defineEmits<{ 'update:visible': [value: boolean]; 'view-management': []; 'start-upgrade': [] }>();
const { t } = useI18n();
const availableRelease = computed(() => getAvailableUpdateRelease(props.status));
const summary = computed(
  () => availableRelease.value?.upgrade_notes || availableRelease.value?.notes || t('update.preview.summaryEmpty'),
);
</script>
<style scoped lang="less">
.update-preview__versions {
  display: grid;
  gap: var(--td-comp-margin-l);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.update-preview__versions div {
  border-left: 2px solid var(--td-brand-color);
  padding-left: var(--td-comp-paddingLR-s);
}

.update-preview__versions dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.update-preview__versions dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  font-variant-numeric: tabular-nums;
  margin: var(--td-comp-margin-xs) 0 0;
}

.update-preview__summary {
  color: var(--td-text-color-secondary);
  margin: 0;
  white-space: pre-line;
}

.update-preview__actions {
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: flex-end;
  margin-top: var(--td-comp-margin-xl);
}

.update-preview__up-to-date {
  color: var(--td-success-color);
  margin: var(--td-comp-margin-l) 0 0;
}
</style>
