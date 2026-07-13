<template>
  <section class="saved-query-view-control" :aria-label="t('app.savedQueryViews.label')">
    <t-select
      :model-value="selectedId"
      class="saved-query-view-control__select"
      clearable
      :disabled="controller.isBusy.value"
      :loading="controller.loading.value"
      :placeholder="t('app.savedQueryViews.placeholder')"
      @update:model-value="selectView"
    >
      <t-option v-for="view in controller.views.value" :key="view.id" :value="view.id" :label="view.name" />
    </t-select>
    <div class="saved-query-view-control__actions">
      <t-button size="small" variant="text" :disabled="controller.isBusy.value" @click="openSaveDialog('create')">
        {{ t('app.savedQueryViews.actions.saveAs') }}
      </t-button>
      <t-button
        size="small"
        variant="text"
        :disabled="controller.isBusy.value || !controller.hasSelectedView.value"
        @click="openSaveDialog('update')"
      >
        {{ t('app.savedQueryViews.actions.update') }}
      </t-button>
      <t-button
        size="small"
        theme="danger"
        variant="text"
        :disabled="controller.isBusy.value || !controller.hasSelectedView.value"
        @click="deleteDialogVisible = true"
      >
        {{ t('app.savedQueryViews.actions.delete') }}
      </t-button>
    </div>

    <t-dialog
      v-model:visible="saveDialogVisible"
      :cancel-btn="t('app.savedQueryViews.actions.cancel')"
      :confirm-btn="
        saveDialogMode === 'create' ? t('app.savedQueryViews.actions.save') : t('app.savedQueryViews.actions.update')
      "
      :confirm-loading="controller.submitting.value"
      :header="
        saveDialogMode === 'create'
          ? t('app.savedQueryViews.dialog.createTitle')
          : t('app.savedQueryViews.dialog.updateTitle')
      "
      @confirm="saveView"
    >
      <t-input
        v-model="draftName"
        clearable
        :maxlength="120"
        :placeholder="t('app.savedQueryViews.namePlaceholder')"
        @update:model-value="nameError = ''"
      />
      <p v-if="nameError" class="saved-query-view-control__validation-error">{{ nameError }}</p>
    </t-dialog>

    <t-dialog
      v-model:visible="deleteDialogVisible"
      :cancel-btn="t('app.savedQueryViews.actions.cancel')"
      :confirm-btn="{ content: t('app.savedQueryViews.actions.delete'), theme: 'danger' }"
      :confirm-loading="controller.deleting.value"
      :header="t('app.savedQueryViews.dialog.deleteTitle')"
      theme="warning"
      @confirm="deleteView"
    >
      {{ t('app.savedQueryViews.dialog.deleteDescription', { name: controller.selectedView.value?.name ?? '' }) }}
    </t-dialog>
  </section>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import type { SavedQueryViewController, SavedQueryViewId } from './saved-query-views';

const props = defineProps<{
  controller: SavedQueryViewController<unknown, SavedQueryViewId>;
}>();

const { t } = useI18n();

const saveDialogVisible = ref(false);
const deleteDialogVisible = ref(false);
const saveDialogMode = ref<'create' | 'update'>('create');
const draftName = ref('');
const nameError = ref('');

const selectedId = computed(() => props.controller.selectedId.value);

function openSaveDialog(mode: 'create' | 'update') {
  saveDialogMode.value = mode;
  draftName.value = mode === 'update' ? (props.controller.selectedView.value?.name ?? '') : '';
  nameError.value = '';
  saveDialogVisible.value = true;
}

async function saveView() {
  if (!draftName.value.trim()) {
    nameError.value = t('app.savedQueryViews.nameRequired');
    return;
  }

  if (await props.controller.save(draftName.value, saveDialogMode.value)) {
    saveDialogVisible.value = false;
  }
}

async function deleteView() {
  if (await props.controller.removeSelected()) {
    deleteDialogVisible.value = false;
  }
}

function selectView(value: string | number | Array<string | number> | undefined) {
  if (Array.isArray(value) || (typeof value !== 'string' && typeof value !== 'number')) {
    void props.controller.select(undefined);
    return;
  }
  void props.controller.select(value);
}
</script>
<style scoped lang="less">
.saved-query-view-control {
  align-items: center;
  display: flex;
  flex: 1 1 320px;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  min-width: min(100%, 280px);
}

.saved-query-view-control__select {
  flex: 1 1 180px;
  min-width: 180px;
}

.saved-query-view-control__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-4);
}

.saved-query-view-control__validation-error {
  color: var(--td-error-color);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-8) 0 0;
}

@media (width <= 768px) {
  .saved-query-view-control__select {
    min-width: 0;
  }
}
</style>
