<template>
  <section class="saved-query-view-control" :aria-label="t('app.savedQueryViews.label')">
    <t-tooltip :content="selectedViewName" :disabled="!selectedViewName" placement="top">
      <t-select
        :model-value="selectedId"
        class="saved-query-view-control__select"
        clearable
        :disabled="controller.isBusy.value"
        :empty="t('app.savedQueryViews.noResults')"
        filterable
        :filter="() => true"
        :loading="controller.loading.value"
        :placeholder="t('app.savedQueryViews.placeholder')"
        :input-value="viewSearchText"
        @update:input-value="viewSearchText = normalizeSearchValue($event)"
        @update:model-value="selectView"
      >
        <t-option v-for="view in displayedViews" :key="view.id" :value="view.id" :label="view.name" />
      </t-select>
    </t-tooltip>
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
      <t-checkbox v-model="draftIsDefault">{{ t('app.savedQueryViews.default') }}</t-checkbox>
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
const draftIsDefault = ref(false);
const nameError = ref('');
const viewSearchText = ref('');

const selectedId = computed(() => props.controller.selectedId.value);
const selectedViewName = computed(() => props.controller.selectedView.value?.name ?? '');
const displayedViews = computed(() => {
  const search = viewSearchText.value.trim().toLowerCase();
  const matchingViews = props.controller.views.value.filter((view) =>
    search ? view.name.toLowerCase().includes(search) : true,
  );
  const selectedView = props.controller.selectedView.value;
  const firstViews = matchingViews.slice(0, 10);

  if (search || !selectedView || firstViews.some((view) => view.id === selectedView.id)) {
    return firstViews;
  }

  return [...firstViews.slice(0, 9), selectedView];
});

function openSaveDialog(mode: 'create' | 'update') {
  saveDialogMode.value = mode;
  draftName.value = mode === 'update' ? (props.controller.selectedView.value?.name ?? '') : '';
  draftIsDefault.value = mode === 'update' ? Boolean(props.controller.selectedView.value?.isDefault) : false;
  nameError.value = '';
  saveDialogVisible.value = true;
}

async function saveView() {
  if (!draftName.value.trim()) {
    nameError.value = t('app.savedQueryViews.nameRequired');
    return;
  }

  if (await props.controller.save(draftName.value, saveDialogMode.value, draftIsDefault.value)) {
    saveDialogVisible.value = false;
  }
}

async function deleteView() {
  if (await props.controller.removeSelected()) {
    deleteDialogVisible.value = false;
  }
}

function selectView(value: string | number | Array<string | number> | undefined) {
  viewSearchText.value = '';
  if (Array.isArray(value) || (typeof value !== 'string' && typeof value !== 'number')) {
    void props.controller.select(undefined);
    return;
  }
  void props.controller.select(value);
}

function normalizeSearchValue(value: string | number | undefined) {
  return typeof value === 'string' ? value : '';
}
</script>
<style scoped lang="less">
.saved-query-view-control {
  align-items: center;
  display: flex;
  flex: 0 1 380px;
  flex-wrap: nowrap;
  gap: var(--graft-density-gap-8);
  margin-left: auto;
  max-width: 380px;
  min-width: 300px;
}

.saved-query-view-control__select {
  min-width: 0;
  width: 100%;
}

.saved-query-view-control__actions {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-4);
}

.saved-query-view-control__validation-error {
  color: var(--td-error-color);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-8) 0 0;
}

@media (width <= 768px) {
  .saved-query-view-control {
    margin-left: 0;
  }

  .saved-query-view-control__select {
    min-width: 0;
  }
}
</style>
