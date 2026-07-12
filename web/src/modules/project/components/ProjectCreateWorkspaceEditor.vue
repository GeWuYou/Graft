<template>
  <section class="project-create-workspace">
    <t-alert theme="info" :message="t('project.create.workspace.hint')" />
    <div class="project-create-workspace__layout">
      <t-card :title="t('project.create.workspace.filesTitle')" bordered class="project-create-workspace__tree-card">
        <template #actions
          ><t-button theme="primary" variant="text" @click="dialogVisible = true">{{
            t('project.create.workspace.addFile')
          }}</t-button></template
        >
        <div class="project-create-workspace__tree">
          <button
            v-for="file in files"
            :key="file.path"
            type="button"
            class="project-create-workspace__file"
            :class="{ 'project-create-workspace__file--active': file.path === activePath }"
            @click="activePath = file.path"
          >
            {{ file.path }}
          </button>
        </div>
      </t-card>
      <t-card
        :title="activeFile?.path || t('project.create.workspace.filesTitle')"
        bordered
        class="project-create-workspace__editor-card"
      >
        <template #actions
          ><t-button theme="danger" variant="text" :disabled="!canRemoveActive" @click="removeActive">{{
            t('project.create.workspace.removeFile')
          }}</t-button></template
        >
        <project-monaco-surface
          v-if="activeFile"
          v-model="activeFile.content"
          :editor-aria-label="t('project.create.workspace.editorAriaLabel', { path: activeFile.path })"
          :language="activeLanguage"
          :model-key="`managed-create:${activeFile.path}`"
        />
      </t-card>
    </div>
    <t-dialog
      v-model:visible="dialogVisible"
      :header="t('project.create.workspace.addFile')"
      :confirm-btn="t('project.create.workspace.addFile')"
      :cancel-btn="t('project.create.actions.cancel')"
      @confirm="addFile"
    >
      <t-form :data="newFile"
        ><t-form-item :label="t('project.create.workspace.filePath')"
          ><t-input
            v-model="newFile.path"
            :placeholder="t('project.create.workspace.filePathPlaceholder')" /></t-form-item
      ></t-form>
      <t-alert v-if="addError" theme="error" :message="addError" />
    </t-dialog>
  </section>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';

import { resolveWorkspaceMonacoLanguage } from '../shared/configuration-workspace';
import { useProjectPageContext } from '../shared/page-context';
import type { ProjectWorkspaceManifestFile } from '../types/project';
import ProjectMonacoSurface from './ProjectMonacoSurface.vue';
defineOptions({ name: 'ProjectCreateWorkspaceEditor' });
const files = defineModel<ProjectWorkspaceManifestFile[]>('files', { required: true });
const { t } = useProjectPageContext();
const activePath = ref(files.value[0]?.path || 'compose.yaml');
const dialogVisible = ref(false);
const newFile = ref({ path: '' });
const addError = ref('');
const activeFile = computed(() => files.value.find((file) => file.path === activePath.value));
const activeLanguage = computed(() => resolveWorkspaceMonacoLanguage({ path: activeFile.value?.path }));
const canRemoveActive = computed(() => Boolean(activeFile.value) && files.value.length > 1);
function normalizePath(path: string) {
  return path.trim().replace(/^\.\//, '');
}
function addFile() {
  const path = normalizePath(newFile.value.path);
  if (
    !path ||
    path.startsWith('/') ||
    path.split('/').some((part) => !part || part === '.' || part === '..') ||
    files.value.some((file) => file.path === path)
  ) {
    addError.value = t('project.create.workspace.invalidFilePath');
    return;
  }
  files.value = [...files.value, { path, content: '' }];
  activePath.value = path;
  newFile.value.path = '';
  addError.value = '';
  dialogVisible.value = false;
}
function removeActive() {
  if (!activeFile.value || !canRemoveActive.value) return;
  const next = files.value.filter((file) => file.path !== activeFile.value?.path);
  files.value = next;
  activePath.value = next[0]?.path || '';
}
</script>
<style scoped>
.project-create-workspace,
.project-create-workspace__layout,
.project-create-workspace__tree {
  display: flex;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.project-create-workspace {
  flex-direction: column;
}

.project-create-workspace__layout {
  min-height: 480px;
}

.project-create-workspace__tree-card {
  flex: 0 0 260px;
}

.project-create-workspace__editor-card {
  flex: 1;
}

.project-create-workspace__tree {
  flex-direction: column;
}

.project-create-workspace__file {
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-primary);
  cursor: pointer;
  padding: var(--graft-density-gap-8);
  text-align: left;
}

.project-create-workspace__file--active,
.project-create-workspace__file:hover {
  background: color-mix(in srgb, var(--td-brand-color-6) 10%, transparent);
}

.project-create-workspace__editor-card :deep(.t-card__body) {
  height: 420px;
  padding: 0;
}

@media (width <= 768px) {
  .project-create-workspace__layout {
    flex-direction: column;
    min-height: 0;
  }

  .project-create-workspace__tree-card {
    flex: auto;
  }

  .project-create-workspace__editor-card :deep(.t-card__body) {
    height: 360px;
  }
}
</style>
