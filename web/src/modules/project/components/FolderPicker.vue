<template>
  <t-dialog
    :visible="visible"
    :header="t('project.import.picker.title')"
    :width="880"
    placement="center"
    destroy-on-close
    @close="handleClose"
  >
    <div class="folder-picker">
      <div v-if="errorMessage" class="folder-picker__alerts">
        <t-alert theme="warning" :message="errorMessage" />
      </div>

      <div class="folder-picker__toolbar">
        <div v-if="showSourceSelector" class="folder-picker__source">
          <label class="folder-picker__label">{{ t('project.import.picker.sourceLabel') }}</label>
          <t-select
            :value="selectedSourceKey"
            :options="sourceOptions"
            :loading="sourcesLoading"
            @change="handleSourceChange"
          />
        </div>

        <div class="folder-picker__breadcrumb">
          <label class="folder-picker__label">{{ t('project.import.picker.pathLabel') }}</label>
          <t-breadcrumb v-if="activeDirectoryRef">
            <t-breadcrumb-item v-for="item in breadcrumbs" :key="item.key" @click="goToPath(item.path)">
              {{ item.label }}
            </t-breadcrumb-item>
          </t-breadcrumb>
          <span v-else class="folder-picker__placeholder">{{ t('project.import.picker.sourceLoading') }}</span>
        </div>

        <div class="folder-picker__actions">
          <label class="folder-picker__label">{{ t('project.import.picker.actionsLabel') }}</label>
          <t-space size="small">
            <t-button
              theme="default"
              variant="outline"
              size="small"
              :disabled="!activeDirectoryRef"
              :loading="directoriesLoading"
              @click="refreshCurrentDirectory"
            >
              {{ t('project.import.picker.refresh') }}
            </t-button>
          </t-space>
        </div>
      </div>

      <div v-if="directoryListLimited" class="folder-picker__alerts">
        <t-alert theme="info" :message="t('project.import.picker.truncatedNotice', { count: directoryLimit })" />
      </div>

      <div class="folder-picker__body graft-scrollbar">
        <t-loading :loading="directoriesLoading || sourcesLoading" size="small">
          <div v-if="showDirectoryEmpty" class="folder-picker__empty">
            <management-empty-state
              :title="t('project.import.picker.emptyTitle')"
              :description="t('project.import.picker.emptyDescription')"
            />
          </div>

          <t-list v-else split>
            <t-list-item v-if="canGoParent" class="folder-picker__list-item" @click="goParent">
              <div class="folder-picker__item">
                <div>
                  <div class="folder-picker__item-title">{{ t('project.import.picker.parentDirectory') }}</div>
                  <div class="folder-picker__item-subtitle">{{ parentDisplayPath }}</div>
                </div>
              </div>
            </t-list-item>
            <t-list-item
              v-for="item in directories"
              :key="item.path"
              class="folder-picker__list-item"
              :class="{ 'folder-picker__list-item--active': item.path === highlightedPath }"
              @click="highlightDirectory(item.path)"
              @dblclick="goToPath(item.path)"
            >
              <div class="folder-picker__item">
                <div>
                  <div class="folder-picker__item-title">{{ item.name }}</div>
                  <div class="folder-picker__item-subtitle">{{ item.path || activeSource?.path || '/' }}</div>
                </div>
                <t-button variant="text" theme="primary" @click.stop="goToPath(item.path)">
                  {{ t('project.import.picker.openDirectory') }}
                </t-button>
              </div>
            </t-list-item>
          </t-list>
        </t-loading>
      </div>
    </div>

    <template #footer>
      <t-space>
        <t-button theme="default" variant="outline" @click="handleClose">
          {{ t('project.list.actions.cancel') }}
        </t-button>
        <t-button
          theme="primary"
          :disabled="directoriesLoading || sourcesLoading || !selectionForConfirm"
          @click="confirmSelection"
        >
          {{ t('project.import.picker.confirm') }}
        </t-button>
      </t-space>
    </template>
  </t-dialog>
</template>
<script setup lang="ts">
import type { SelectValue } from 'tdesign-vue-next';
import { computed, ref, watch } from 'vue';

import { ManagementEmptyState } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { getProjectImportDirectories, getProjectImportDirectorySources } from '../api/import';
import {
  buildDirectoryBreadcrumbs,
  buildDirectorySelection,
  buildDirectorySourceLabel,
  initialDirectoryPath,
  normalizeDirectoryPath,
} from '../shared/import';
import { useProjectPageContext } from '../shared/page-context';
import type {
  ProjectImportDirectoryListItem,
  ProjectImportDirectoryListResponse,
  ProjectImportDirectoryRef,
  ProjectImportDirectorySource,
} from '../types/import';

const props = defineProps<{
  visible: boolean;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
  (event: 'confirm', value: ProjectImportDirectoryRef): void;
}>();

const { t } = useProjectPageContext();

const sourcesLoading = ref(false);
const directoriesLoading = ref(false);
const errorMessage = ref('');
const sources = ref<ProjectImportDirectorySource[]>([]);
const activeSourceKey = ref('');
const activeDirectoryRef = ref<ProjectImportDirectoryRef | null>(null);
const currentPath = ref('');
const parentPath = ref<string | null>(null);
const directories = ref<ProjectImportDirectoryListItem[]>([]);
const highlightedPath = ref('');
const directoryCache = ref<Record<string, ProjectImportDirectoryListResponse>>({});
const directoryLimit = ref(200);
const directoryListLimited = ref(false);

const sourceOptions = computed(() =>
  sources.value.map((item) => ({
    label: buildDirectorySourceLabel(item),
    value: buildSourceKey(item),
  })),
);

const selectedSourceKey = computed(() => activeSourceKey.value);
const activeSource = computed(
  () => sources.value.find((item) => buildSourceKey(item) === activeSourceKey.value) ?? null,
);
const showSourceSelector = computed(() => sources.value.length > 1);
const breadcrumbs = computed(() =>
  activeDirectoryRef.value ? buildDirectoryBreadcrumbs(activeDirectoryRef.value) : [],
);
const selectionForConfirm = computed(() => {
  if (!activeSource.value || !activeDirectoryRef.value || directoriesLoading.value || sourcesLoading.value) {
    return null;
  }

  return buildDirectorySelection(activeSource.value, highlightedPath.value || currentPath.value);
});
const canGoParent = computed(() => currentPath.value !== '');
const showDirectoryEmpty = computed(() => !directoriesLoading.value && !directories.value.length && !canGoParent.value);
const parentDisplayPath = computed(() => {
  if (parentPath.value) {
    return activeSource.value?.path === '/' ? `/${parentPath.value}` : parentPath.value;
  }
  return activeSource.value?.path || '/';
});

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      void initialize();
      return;
    }

    resetState();
  },
);

function buildSourceKey(source: ProjectImportDirectorySource) {
  return `${source.provider}:${source.root_id}`;
}

function buildDirectoryCacheKey(directory: ProjectImportDirectoryRef) {
  return `${directory.provider}:${directory.root_id}:${normalizeDirectoryPath(directory.path)}`;
}

function resetState() {
  errorMessage.value = '';
  sources.value = [];
  activeSourceKey.value = '';
  directories.value = [];
  activeDirectoryRef.value = null;
  currentPath.value = '';
  parentPath.value = null;
  highlightedPath.value = '';
  directoryCache.value = {};
  directoryLimit.value = 200;
  directoryListLimited.value = false;
}

async function initialize() {
  resetState();
  sourcesLoading.value = true;
  try {
    const response = await getProjectImportDirectorySources();
    sources.value = response.items;
    const preferredSource = response.items.find((item) => item.managed) || response.items[0];
    if (!preferredSource) {
      return;
    }

    activeSourceKey.value = buildSourceKey(preferredSource);
    await loadDirectories(buildDirectorySelection(preferredSource, initialDirectoryPath(preferredSource)));
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('project.import.messages.directorySourceLoadFailed'));
  } finally {
    sourcesLoading.value = false;
  }
}

function applyDirectoryResponse(response: ProjectImportDirectoryListResponse, directory: ProjectImportDirectoryRef) {
  activeDirectoryRef.value = directory;
  currentPath.value = normalizeDirectoryPath(response.current_path);
  parentPath.value = response.parent_path ? normalizeDirectoryPath(response.parent_path) : null;
  directories.value = response.directories;
  highlightedPath.value = currentPath.value;
  directoryLimit.value = response.limit;
  directoryListLimited.value = response.has_more;
}

async function loadDirectories(directory: ProjectImportDirectoryRef, forceRefresh = false) {
  const cacheKey = buildDirectoryCacheKey(directory);
  const cached = directoryCache.value[cacheKey];
  if (cached && !forceRefresh) {
    errorMessage.value = '';
    applyDirectoryResponse(cached, directory);
    return true;
  }
  directoriesLoading.value = true;
  errorMessage.value = '';
  try {
    const response = await getProjectImportDirectories({
      ...directory,
      limit: 200,
      order: 'asc',
      sort: 'name',
    });
    directoryCache.value = {
      ...directoryCache.value,
      [cacheKey]: response,
    };
    applyDirectoryResponse(response, directory);
    return true;
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('project.import.messages.directoryLoadFailed'));
    return false;
  } finally {
    directoriesLoading.value = false;
  }
}

async function handleSourceChange(value: SelectValue) {
  const rawValue = Array.isArray(value) ? value[0] : value;
  if (rawValue === undefined || rawValue === null) {
    return;
  }

  const nextKey = String(rawValue);
  const nextSource = sources.value.find((item) => buildSourceKey(item) === nextKey);
  if (!nextSource) {
    return;
  }

  const previousKey = activeSourceKey.value;
  activeSourceKey.value = nextKey;
  const loaded = await loadDirectories(buildDirectorySelection(nextSource, initialDirectoryPath(nextSource)));
  if (!loaded) {
    activeSourceKey.value = previousKey;
  }
}

function highlightDirectory(path: string) {
  highlightedPath.value = normalizeDirectoryPath(path);
}

async function goToPath(path: string) {
  if (!activeSource.value) {
    return;
  }

  await loadDirectories(buildDirectorySelection(activeSource.value, path));
}

async function refreshCurrentDirectory() {
  if (!activeSource.value) {
    return;
  }

  await loadDirectories(buildDirectorySelection(activeSource.value, currentPath.value), true);
}

async function goParent() {
  if (parentPath.value === null) {
    return;
  }

  await goToPath(parentPath.value);
}

function confirmSelection() {
  if (!selectionForConfirm.value) {
    return;
  }

  emit('confirm', selectionForConfirm.value);
}

function handleClose() {
  emit('close');
}
</script>
<style scoped lang="less">
.folder-picker,
.folder-picker__alerts,
.folder-picker__toolbar,
.folder-picker__breadcrumb,
.folder-picker__body,
.folder-picker__empty {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.folder-picker__toolbar {
  align-items: start;
  grid-template-columns: minmax(220px, 260px) minmax(0, 1fr) minmax(160px, 200px);
  margin-bottom: var(--graft-density-gap-12);
}

.folder-picker__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.folder-picker__source,
.folder-picker__breadcrumb,
.folder-picker__actions {
  display: grid;
  gap: var(--graft-density-gap-8);
}

.folder-picker__placeholder {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-medium);
}

.folder-picker__body {
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  max-height: 420px;
  min-height: 320px;
  overflow: auto;
  padding: var(--graft-density-gap-8);
}

.folder-picker__list-item {
  border-radius: var(--td-radius-medium);
  cursor: pointer;
}

.folder-picker__list-item--active {
  background: var(--td-bg-color-secondarycontainer);
}

.folder-picker__item {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
  width: 100%;
}

.folder-picker__item-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.folder-picker__item-subtitle {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  margin-top: var(--graft-density-gap-4);
}

@media (width <= 768px) {
  .folder-picker__toolbar {
    grid-template-columns: 1fr;
  }
}
</style>
