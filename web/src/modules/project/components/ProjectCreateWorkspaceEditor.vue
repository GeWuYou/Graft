<template>
  <section class="project-create-workspace">
    <t-alert theme="info" :message="t('project.create.workspace.hint')" />
    <div class="project-create-workspace__layout">
      <t-card :title="t('project.create.workspace.filesTitle')" bordered class="project-create-workspace__tree-card">
        <div
          class="project-create-workspace__tree graft-scrollbar"
          @contextmenu.prevent="openMenu('', 'directory', $event)"
        >
          <div class="project-create-workspace__root-heading">
            <p class="project-create-workspace__root-label">{{ t('project.create.workspace.rootLabel') }}</p>
            <button
              type="button"
              class="project-create-workspace__entry-actions"
              :aria-label="
                t('project.create.workspace.entryActions', { path: t('project.create.workspace.rootLabel') })
              "
              @click.stop="openMenu('', 'directory', $event, true)"
            >
              <ellipsis-icon />
            </button>
          </div>
          <template v-for="row in treeRows" :key="row.path">
            <div
              class="project-create-workspace__tree-row"
              :class="{ 'project-create-workspace__tree-row--active': row.path === activePath }"
              :style="{ '--workspace-tree-depth': String(row.depth) }"
              @contextmenu.prevent.stop="openMenu(row.path, row.nodeType, $event)"
            >
              <button
                class="project-create-workspace__tree-entry"
                type="button"
                @click="row.nodeType === 'directory' ? toggleDirectory(row.path) : (activePath = row.path)"
              >
                <span v-if="row.nodeType === 'directory'" class="project-create-workspace__tree-expander">{{
                  isDirectoryExpanded(row.path) ? '▾' : '▸'
                }}</span>
                <folder-icon v-if="row.nodeType === 'directory'" />
                <file-code-icon v-else />
                <span>{{ row.name }}</span>
              </button>
              <button
                type="button"
                class="project-create-workspace__entry-actions"
                :aria-label="t('project.create.workspace.entryActions', { path: row.path })"
                @click.stop="openMenu(row.path, row.nodeType, $event, true)"
              >
                <ellipsis-icon />
              </button>
            </div>
          </template>
          <t-empty v-if="!treeRows.length" :description="t('project.create.workspace.filesEmpty')" />
        </div>
      </t-card>
      <t-card
        :title="activeFile?.path || t('project.create.workspace.filesTitle')"
        bordered
        class="project-create-workspace__editor-card"
      >
        <project-monaco-surface
          v-if="activeFile"
          v-model="activeFile.content"
          :editor-aria-label="t('project.create.workspace.editorAriaLabel', { path: activeFile.path })"
          :language="activeLanguage"
          :model-key="`managed-create:${activeFile.path}`"
        />
        <t-empty v-else :description="t('project.create.workspace.selectFile')" />
      </t-card>
    </div>

    <div
      v-if="contextMenu.visible"
      class="project-create-workspace__context-menu"
      :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
      role="menu"
      @keydown.down.prevent="moveMenuFocus(1)"
      @keydown.end.prevent="moveMenuFocus(-1, true)"
      @keydown.esc.prevent="closeMenu(true)"
      @keydown.home.prevent="moveMenuFocus(1, true)"
      @keydown.up.prevent="moveMenuFocus(-1)"
    >
      <button ref="firstMenuItem" type="button" role="menuitem" @click="beginCreate('file')">
        {{ t('project.create.workspace.newFile') }}
      </button>
      <button type="button" role="menuitem" @click="beginCreate('directory')">
        {{ t('project.create.workspace.newFolder') }}
      </button>
      <button type="button" role="menuitem" :disabled="!contextMenu.path" @click="beginRename">
        {{ t('project.create.workspace.rename') }}
      </button>
      <button type="button" role="menuitem" :disabled="!contextMenu.path" @click="beginDelete">
        {{ t('project.create.workspace.delete') }}
      </button>
    </div>

    <t-dialog
      v-model:visible="entryDialog.visible"
      :header="
        entryDialog.mode === 'rename'
          ? t('project.create.workspace.rename')
          : entryDialog.nodeType === 'directory'
            ? t('project.create.workspace.newFolder')
            : t('project.create.workspace.newFile')
      "
      :confirm-btn="t('project.create.actions.confirm')"
      :cancel-btn="t('project.create.actions.cancel')"
      @confirm="submitEntryDialog"
    >
      <t-form>
        <t-form-item :label="t('project.create.workspace.filePath')">
          <t-input v-model="entryDialog.path" :placeholder="t('project.create.workspace.filePathPlaceholder')" />
        </t-form-item>
      </t-form>
      <t-alert v-if="entryDialog.error" theme="error" :message="entryDialog.error" />
    </t-dialog>

    <t-dialog
      v-model:visible="deleteDialog.visible"
      :header="
        deleteDialog.stage === 'recursive'
          ? t('project.create.workspace.recursiveDeleteTitle')
          : t('project.create.workspace.delete')
      "
      :confirm-btn="
        deleteDialog.stage === 'recursive'
          ? t('project.create.workspace.recursiveDeleteConfirm')
          : t('project.create.actions.confirm')
      "
      :cancel-btn="t('project.create.actions.cancel')"
      @confirm="confirmDelete"
    >
      <p v-if="deleteDialog.stage === 'recursive'">
        {{
          t('project.create.workspace.recursiveDeleteBody', {
            path: deleteDialog.path,
            count: String(deleteDialog.count),
          })
        }}
      </p>
      <p v-else>{{ t('project.create.workspace.deleteBody', { path: deleteDialog.path }) }}</p>
    </t-dialog>
  </section>
</template>
<script setup lang="ts">
import { EllipsisIcon, FileCodeIcon, FolderIcon } from 'tdesign-icons-vue-next';
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue';

import { resolveWorkspaceMonacoLanguage } from '../shared/configuration-workspace';
import { useProjectPageContext } from '../shared/page-context';
import type { ProjectWorkspaceManifestFile } from '../types/project';
import ProjectMonacoSurface from './ProjectMonacoSurface.vue';

defineOptions({ name: 'ProjectCreateWorkspaceEditor' });

type WorkspaceDraftEntry = ProjectWorkspaceManifestFile & { node_type?: 'directory' | 'file' };
type TreeRow = { depth: number; name: string; nodeType: 'directory' | 'file'; path: string };

const files = defineModel<WorkspaceDraftEntry[]>('files', { required: true });
const { t } = useProjectPageContext();
const activePath = ref(files.value.find((entry) => entry.node_type !== 'directory')?.path || '');
const expandedDirectories = ref<string[]>([]);
const firstMenuItem = ref<HTMLButtonElement | null>(null);
const contextMenu = reactive({ nodeType: 'directory' as 'directory' | 'file', path: '', visible: false, x: 0, y: 0 });
let contextMenuTrigger: HTMLElement | null = null;
const entryDialog = reactive({
  error: '',
  mode: 'create' as 'create' | 'rename',
  nodeType: 'file' as 'directory' | 'file',
  path: '',
  visible: false,
});
const deleteDialog = reactive({ count: 0, path: '', stage: 'initial' as 'initial' | 'recursive', visible: false });

const treeRows = computed(() => {
  const explicitDirectories = new Set(
    files.value.filter((entry) => entry.node_type === 'directory').map((entry) => entry.path),
  );
  const filePaths = new Set(files.value.filter((entry) => entry.node_type !== 'directory').map((entry) => entry.path));
  for (const path of filePaths) {
    const parts = path.split('/');
    for (let index = 1; index < parts.length; index += 1) explicitDirectories.add(parts.slice(0, index).join('/'));
  }
  const directories = [...explicitDirectories].sort((left, right) => left.localeCompare(right));
  const result: TreeRow[] = [];
  for (const path of directories)
    result.push({
      depth: path.split('/').length - 1,
      name: path.split('/').at(-1) || path,
      nodeType: 'directory',
      path,
    });
  for (const path of [...filePaths].sort((left, right) => left.localeCompare(right)))
    result.push({ depth: path.split('/').length - 1, name: path.split('/').at(-1) || path, nodeType: 'file', path });
  return result
    .sort((left, right) => left.path.localeCompare(right.path) || (left.nodeType === 'directory' ? -1 : 1))
    .filter((row) => {
      const parents = row.path.split('/').slice(0, -1);
      return parents.every((_, index) => expandedDirectories.value.includes(parents.slice(0, index + 1).join('/')));
    });
});
const activeFile = computed(() =>
  files.value.find((entry) => entry.path === activePath.value && entry.node_type !== 'directory'),
);
const activeLanguage = computed(() => resolveWorkspaceMonacoLanguage({ path: activeFile.value?.path }));

watch(
  files,
  () => {
    if (!activeFile.value) activePath.value = files.value.find((entry) => entry.node_type !== 'directory')?.path || '';
  },
  { deep: true },
);

function normalizePath(path: string) {
  return path.trim().replace(/^\.\//, '').replace(/\/+$/, '');
}
function isSafePath(path: string) {
  return (
    Boolean(path) && !path.startsWith('/') && !path.split('/').some((part) => !part || part === '.' || part === '..')
  );
}
function parentDirectory(path: string, nodeType: 'directory' | 'file') {
  if (!path) return '';
  return nodeType === 'directory' ? path : path.split('/').slice(0, -1).join('/');
}
function openMenu(path: string, nodeType: 'directory' | 'file', event: MouseEvent, focusMenu = false) {
  contextMenu.path = path;
  contextMenu.nodeType = nodeType;
  contextMenu.x = event.clientX;
  contextMenu.y = event.clientY;
  contextMenuTrigger = event.currentTarget instanceof HTMLElement ? event.currentTarget : null;
  contextMenu.visible = true;
  if (focusMenu) void nextTick(() => firstMenuItem.value?.focus());
}
function isDirectoryExpanded(path: string) {
  return expandedDirectories.value.includes(path);
}
function toggleDirectory(path: string) {
  expandedDirectories.value = isDirectoryExpanded(path)
    ? expandedDirectories.value.filter((value) => value !== path)
    : [...expandedDirectories.value, path];
}
function closeMenu(restoreFocus = false) {
  contextMenu.visible = false;
  if (restoreFocus) void nextTick(() => contextMenuTrigger?.focus());
}
function moveMenuFocus(direction: 1 | -1, boundary = false) {
  const items = Array.from(
    document.querySelectorAll<HTMLButtonElement>('.project-create-workspace__context-menu [role="menuitem"]'),
  ).filter((item) => !item.disabled);
  if (!items.length) return;
  const currentIndex = items.findIndex((item) => item === document.activeElement);
  const nextIndex = boundary
    ? direction > 0
      ? 0
      : items.length - 1
    : (currentIndex + direction + items.length) % items.length;
  items[nextIndex].focus();
}
function beginCreate(nodeType: 'directory' | 'file') {
  entryDialog.mode = 'create';
  entryDialog.nodeType = nodeType;
  entryDialog.error = '';
  const parent = parentDirectory(contextMenu.path, contextMenu.nodeType);
  entryDialog.path = parent ? `${parent}/` : '';
  entryDialog.visible = true;
  closeMenu();
}
function beginRename() {
  entryDialog.mode = 'rename';
  entryDialog.nodeType = contextMenu.nodeType;
  entryDialog.error = '';
  entryDialog.path = contextMenu.path;
  entryDialog.visible = true;
  closeMenu();
}
function submitEntryDialog() {
  const path = normalizePath(entryDialog.path);
  if (!isSafePath(path)) {
    entryDialog.error = t('project.create.workspace.invalidFilePath');
    return;
  }
  const oldPath = contextMenu.path;
  if (entryDialog.mode === 'create') {
    if (
      files.value.some(
        (entry) => entry.path === path || entry.path.startsWith(`${path}/`) || path.startsWith(`${entry.path}/`),
      )
    ) {
      entryDialog.error = t('project.create.workspace.pathConflict');
      return;
    }
    files.value = [
      ...files.value,
      entryDialog.nodeType === 'directory'
        ? ({ path, content: '', node_type: 'directory' } as WorkspaceDraftEntry)
        : ({ path, content: '', node_type: 'file' } as WorkspaceDraftEntry),
    ];
    if (entryDialog.nodeType === 'file') activePath.value = path;
  } else {
    if (
      !oldPath ||
      path.startsWith(`${oldPath}/`) ||
      (path !== oldPath &&
        files.value.some(
          (entry) =>
            entry.path !== oldPath &&
            !entry.path.startsWith(`${oldPath}/`) &&
            (entry.path === path || entry.path.startsWith(`${path}/`) || path.startsWith(`${entry.path}/`)),
        ))
    ) {
      entryDialog.error = t('project.create.workspace.pathConflict');
      return;
    }
    files.value = files.value.map((entry) =>
      entry.path === oldPath || entry.path.startsWith(`${oldPath}/`)
        ? { ...entry, path: `${path}${entry.path.slice(oldPath.length)}` }
        : entry,
    );
    if (activePath.value === oldPath || activePath.value.startsWith(`${oldPath}/`))
      activePath.value = `${path}${activePath.value.slice(oldPath.length)}`;
    expandedDirectories.value = expandedDirectories.value.map((value) =>
      value === oldPath || value.startsWith(`${oldPath}/`) ? `${path}${value.slice(oldPath.length)}` : value,
    );
  }
  entryDialog.visible = false;
}
function beginDelete() {
  deleteDialog.path = contextMenu.path;
  deleteDialog.count = files.value.filter(
    (entry) => entry.path === contextMenu.path || entry.path.startsWith(`${contextMenu.path}/`),
  ).length;
  deleteDialog.stage = 'initial';
  deleteDialog.visible = true;
  closeMenu();
}
function confirmDelete() {
  const target = files.value.find((entry) => entry.path === deleteDialog.path);
  if (target?.node_type === 'directory' && deleteDialog.count > 1 && deleteDialog.stage === 'initial') {
    deleteDialog.stage = 'recursive';
    return;
  }
  files.value = files.value.filter(
    (entry) => entry.path !== deleteDialog.path && !entry.path.startsWith(`${deleteDialog.path}/`),
  );
  if (activePath.value === deleteDialog.path || activePath.value.startsWith(`${deleteDialog.path}/`))
    activePath.value = files.value.find((entry) => entry.node_type !== 'directory')?.path || '';
  deleteDialog.visible = false;
}
function handleDocumentClick() {
  closeMenu();
}
document.addEventListener('click', handleDocumentClick);
onBeforeUnmount(() => document.removeEventListener('click', handleDocumentClick));
</script>
<style scoped>
.project-create-workspace {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.project-create-workspace__layout {
  display: flex;
  gap: var(--graft-density-gap-16);
  min-height: 480px;
  min-width: 0;
}

.project-create-workspace__tree-card {
  flex: 0 0 280px;
}

.project-create-workspace__editor-card {
  flex: 1;
  min-width: 0;
}

.project-create-workspace__tree {
  min-height: 420px;
  overflow: auto;
}

.project-create-workspace__root-label {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-small);
  margin: 0 0 var(--graft-density-gap-8);
}

.project-create-workspace__root-heading {
  align-items: center;
  display: flex;
  justify-content: space-between;
  margin-bottom: var(--graft-density-gap-8);
}

.project-create-workspace__root-heading .project-create-workspace__root-label {
  margin-bottom: 0;
}

.project-create-workspace__tree-row {
  align-items: center;
  display: flex;
  padding-left: calc(var(--workspace-tree-depth) * var(--graft-density-gap-16));
}

.project-create-workspace__tree-entry {
  align-items: center;
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-primary);
  cursor: pointer;
  display: flex;
  gap: var(--graft-density-gap-8);
  min-height: 32px;
  padding: 0 var(--graft-density-gap-8);
  text-align: left;
  width: 100%;
}

.project-create-workspace__entry-actions {
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  flex: 0 0 auto;
  min-height: 32px;
  min-width: 32px;
}

.project-create-workspace__entry-actions:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: -2px;
}

.project-create-workspace__tree-expander {
  color: var(--td-text-color-secondary);
  width: 12px;
}

.project-create-workspace__tree-row--active .project-create-workspace__tree-entry,
.project-create-workspace__tree-entry:hover {
  background: color-mix(in srgb, var(--td-brand-color-6) 10%, transparent);
}

.project-create-workspace__context-menu {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  border-radius: var(--td-radius-default);
  box-shadow: var(--td-shadow-2);
  display: flex;
  flex-direction: column;
  min-width: 152px;
  padding: var(--graft-density-gap-4);
  position: fixed;
  z-index: 1000;
}

.project-create-workspace__context-menu button {
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-primary);
  cursor: pointer;
  min-height: 32px;
  padding: 0 var(--graft-density-gap-8);
  text-align: left;
}

.project-create-workspace__context-menu button:hover:not(:disabled) {
  background: var(--td-bg-color-container-hover);
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

  .project-create-workspace__tree {
    min-height: 220px;
  }

  .project-create-workspace__editor-card :deep(.t-card__body) {
    height: 360px;
  }
}
</style>
