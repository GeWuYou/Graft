<template>
  <section class="project-create-workspace">
    <t-alert theme="info" :message="t('project.create.workspace.hint')" />
    <project-workspace-editor
      v-model:active-path="activePath"
      v-model:fullscreen="fullscreen"
      :inline-edit="inlineEdit"
      :active-buffer="editorActiveBuffer"
      :editor-aria-label="t('project.create.workspace.editorAriaLabel', { path: '{path}' })"
      editor-height-storage-key="graft.project.create-workspace.editor.height"
      :empty-description="t('project.create.workspace.filesEmpty')"
      :labels="editorLabels"
      :rows="editorRows"
      :selected-path="workspaceStore.session(workspaceSessionKey).selectedKey"
      :tabs="editorTabs"
      :tabs-empty-description="t('project.create.workspace.selectFile')"
      :tree-title="t('project.create.workspace.filesTitle')"
      :root-label="t('project.create.workspace.rootLabel')"
      @close-tab="(path) => workspaceStore.closeFile(workspaceSessionKey, path)"
      @context-action="(action, row) => handleContextAction(action, row?.path ?? null)"
      @inline-edit-cancel="cancelInlineEdit"
      @inline-edit-submit="submitInlineEdit"
      @select-entry="(row) => selectEntry(row.path)"
      @toggle-directory="(row) => toggleDirectory(row.path)"
      @update-content="updateContent"
      @update:inline-edit="inlineEdit = $event"
    >
      <template #editor-actions>
        <t-tooltip :content="t('project.create.workspace.formatAction')" theme="light">
          <t-button
            data-testid="workspace-create-format"
            theme="default"
            variant="text"
            shape="square"
            size="small"
            :disabled="!editorActiveBuffer"
            @click="formatActiveFile"
          >
            <template #icon><edit-icon /></template>
          </t-button>
        </t-tooltip>
        <t-tooltip :content="t('project.create.workspace.copyAction')" theme="light">
          <t-button
            data-testid="workspace-create-copy"
            theme="default"
            variant="text"
            shape="square"
            size="small"
            :disabled="!editorActiveBuffer"
            @click="copyActiveFile"
          >
            <template #icon><copy-icon /></template>
          </t-button>
        </t-tooltip>
      </template>
      <template #fullscreen-icon="{ fullscreen: isFullscreen }">
        <fullscreen-exit-icon v-if="isFullscreen" />
        <fullscreen-icon v-else />
      </template>
    </project-workspace-editor>

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
import { CopyIcon, EditIcon, FullscreenExitIcon, FullscreenIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onActivated, reactive, ref, watch } from 'vue';

import { copyText } from '@/shared/observability';
import { store } from '@/store/pinia';

import { normalizeTextBlock } from '../shared/configuration-workspace';
import { useProjectPageContext } from '../shared/page-context';
import { emitProjectWorkspaceDebug } from '../shared/project-workspace-debug';
import { type OpenedWorkspaceFile, useProjectWorkspaceStore } from '../store/workspace';
import type { ProjectWorkspaceManifestFile, ProjectWorkspaceTreeItem } from '../types/project';
import ProjectWorkspaceEditor, {
  type ProjectWorkspaceEditorBuffer,
  type ProjectWorkspaceEditorLabels,
  type ProjectWorkspaceEditorRow,
  type ProjectWorkspaceInlineEdit,
} from './ProjectWorkspaceEditor.vue';

defineOptions({ name: 'ProjectCreateWorkspaceEditor' });

type WorkspaceDraftEntry = ProjectWorkspaceManifestFile & { node_type?: 'directory' | 'file' };

const files = defineModel<WorkspaceDraftEntry[]>('files', { required: true });
const { t } = useProjectPageContext();
const workspaceStore = useProjectWorkspaceStore(store);
const workspaceSessionKey = 'project-create-workspace';
const fullscreen = ref(false);
const inlineEdit = ref<ProjectWorkspaceInlineEdit | null>(null);
const deleteDialog = reactive({ count: 0, path: '', stage: 'initial' as 'initial' | 'recursive', visible: false });

workspaceStore.ensureSession(workspaceSessionKey);

const editorLabels = computed<ProjectWorkspaceEditorLabels>(() => ({
  closeAll: t('layout.tagTabs.closeAll'),
  closeLeft: t('layout.tagTabs.closeLeft'),
  closeOther: t('layout.tagTabs.closeOther'),
  closeRight: t('layout.tagTabs.closeRight'),
  delete: t('project.create.workspace.delete'),
  entryActions: t('project.create.workspace.entryActions', { path: '{path}' }),
  exitFullscreen: t('project.detail.configuration.exitFullscreen'),
  fullscreen: t('project.detail.configuration.fullscreen'),
  newFile: t('project.create.workspace.newFile'),
  newFolder: t('project.create.workspace.newFolder'),
  refresh: t('layout.tagTabs.refresh'),
  rename: t('project.create.workspace.rename'),
}));
const activePath = computed({
  get: () => workspaceStore.session(workspaceSessionKey).activeFileKey,
  set: (path: string) => activateTab(path),
});
const editorRows = computed<ProjectWorkspaceEditorRow[]>(() =>
  workspaceStore.visibleTreeRows(workspaceSessionKey).map((row) => ({
    depth: row.depth,
    expanded: row.expanded,
    fileKind: row.item.file_kind,
    name: row.item.name,
    nodeType: row.item.node_type,
    path: row.item.relative_path,
    readOnly: !row.item.editable,
  })),
);
function toEditorBuffer(file: OpenedWorkspaceFile): ProjectWorkspaceEditorBuffer {
  return {
    content: file.content,
    dirty: file.content !== file.savedContent,
    error: file.error,
    language: file.language,
    loading: file.loading,
    modelKey: file.path,
    name: file.name,
    path: file.path,
    readOnly: !file.editable,
  };
}

const editorTabs = computed<ProjectWorkspaceEditorBuffer[]>(() =>
  workspaceStore.openedFiles(workspaceSessionKey).map(toEditorBuffer),
);
const editorActiveBuffer = computed<ProjectWorkspaceEditorBuffer | null>(() => {
  const file = workspaceStore.activeFile(workspaceSessionKey);
  return file ? toEditorBuffer(file) : null;
});

function toTreeItems(entries: WorkspaceDraftEntry[]): ProjectWorkspaceTreeItem[] {
  return entries.map((entry) => ({
    editable: entry.node_type !== 'directory',
    file_kind: entry.node_type === 'directory' ? 'directory' : 'text',
    has_children:
      entry.node_type === 'directory'
        ? entries.some((candidate) => candidate.path.startsWith(`${entry.path}/`))
        : false,
    name: entry.path.split('/').at(-1) || entry.path,
    node_type: entry.node_type === 'directory' ? 'directory' : 'file',
    readable: true,
    relative_path: entry.path,
  }));
}

watch(files, (entries) => synchronizeWorkspace(entries, 'files-watch'), { deep: true, immediate: true });

onActivated(() => {
  synchronizeWorkspace(files.value, 'keep-alive-activated');
});

function synchronizeWorkspace(entries: WorkspaceDraftEntry[], source: 'files-watch' | 'keep-alive-activated') {
  const before = workspaceStore.session(workspaceSessionKey);
  emitProjectWorkspaceDebug('create-sync-start', {
    activeFileKey: before.activeFileKey || '-',
    entryCount: entries.length,
    openedTabCount: before.openedTabs.length,
    source,
  });
  workspaceStore.replaceTree(workspaceSessionKey, toTreeItems(entries));
  if (workspaceStore.activeFile(workspaceSessionKey)) {
    emitProjectWorkspaceDebug('create-sync-complete', {
      activeBufferPresent: true,
      activeFileKey: workspaceStore.session(workspaceSessionKey).activeFileKey || '-',
      openedTabCount: workspaceStore.session(workspaceSessionKey).openedTabs.length,
      source,
    });
    return;
  }

  const activePath = workspaceStore.session(workspaceSessionKey).activeFileKey;
  const activeEntry = entries.find((entry) => entry.path === activePath && entry.node_type !== 'directory');
  const firstFile = activeEntry ?? entries.find((entry) => entry.node_type !== 'directory');
  if (firstFile) activateTab(firstFile.path);
  const session = workspaceStore.session(workspaceSessionKey);
  emitProjectWorkspaceDebug('create-sync-complete', {
    activeBufferPresent: Boolean(workspaceStore.activeFile(workspaceSessionKey)),
    activeFileKey: session.activeFileKey || '-',
    openedTabCount: session.openedTabs.length,
    source,
  });
}

function normalizeEntryName(value: string) {
  return value.trim();
}
function isSafeEntryName(value: string) {
  return Boolean(value) && value !== '.' && value !== '..' && !value.includes('/');
}
function parentDirectory(path: string, nodeType?: 'directory' | 'file') {
  if (!path) return '';
  return nodeType === 'directory' ? path : path.split('/').slice(0, -1).join('/');
}
function entryAt(path: string) {
  return files.value.find((entry) => entry.path === path);
}
function activateTab(path: string) {
  const entry = entryAt(path);
  if (!entry || entry.node_type === 'directory') return;
  workspaceStore.openFile(workspaceSessionKey, path, {
    content: entry.content,
    loaded: true,
    savedContent: entry.content,
  });
}
function selectEntry(path: string) {
  const entry = entryAt(path);
  if (!entry) return;
  if (entry.node_type === 'directory') {
    toggleDirectory(path);
    return;
  }
  workspaceStore.selectNode(workspaceSessionKey, path);
  workspaceStore.openFile(workspaceSessionKey, path, {
    content: entry.content,
    loaded: true,
    savedContent: entry.content,
  });
}
function toggleDirectory(path: string) {
  const expanded = workspaceStore.session(workspaceSessionKey).expandedKeys.includes(path);
  workspaceStore.setExpanded(workspaceSessionKey, path, !expanded);
}
function updateContent(path: string, content: string) {
  const entry = entryAt(path);
  if (!entry || entry.node_type === 'directory') return;
  entry.content = content;
  workspaceStore.setFileContent(workspaceSessionKey, path, content);
}

function formatActiveFile() {
  const activeFile = editorActiveBuffer.value;
  if (!activeFile) return;
  updateContent(activeFile.path, normalizeTextBlock(activeFile.content));
}

async function copyActiveFile() {
  const activeFile = editorActiveBuffer.value;
  if (!activeFile) return;
  const copied = await copyText(activeFile.content);
  if (copied) {
    MessagePlugin.success(t('project.create.workspace.copySuccess'));
    return;
  }
  MessagePlugin.error(t('project.create.workspace.copyFailed'));
}
function handleContextAction(
  action: 'create-file' | 'create-directory' | 'annotation' | 'rename' | 'delete',
  path: string | null,
) {
  if (action === 'annotation') return;
  if (action === 'delete') {
    if (!path) return;
    deleteDialog.path = path;
    deleteDialog.count = files.value.filter((entry) => entry.path === path || entry.path.startsWith(`${path}/`)).length;
    deleteDialog.stage = 'initial';
    deleteDialog.visible = true;
    return;
  }
  if (action === 'rename' && !path) return;
  inlineEdit.value = {
    anchorPath: path,
    mode: action === 'rename' ? 'rename' : action,
    value: action === 'rename' ? (path?.split('/').at(-1) ?? '') : '',
  };
}
function cancelInlineEdit() {
  inlineEdit.value = null;
}
function setInlineEditError(error: string) {
  if (inlineEdit.value) inlineEdit.value = { ...inlineEdit.value, error };
}
function joinEntryPath(parent: string, name: string) {
  return parent ? `${parent}/${name}` : name;
}
function migrateWorkspaceBuffers(oldPath: string, newPath: string) {
  const remapPath = (path: string) =>
    path === oldPath || path.startsWith(`${oldPath}/`) ? `${newPath}${path.slice(oldPath.length)}` : path;
  const session = workspaceStore.session(workspaceSessionKey);
  const openedFiles = workspaceStore.openedFiles(workspaceSessionKey).map((file) => {
    const path = remapPath(file.path);
    return path === file.path ? file : { ...file, name: path.split('/').at(-1) ?? path, path };
  });
  workspaceStore.syncOpenedFiles(workspaceSessionKey, openedFiles, remapPath(session.activeFileKey));
}
function submitInlineEdit() {
  const edit = inlineEdit.value;
  if (!edit) return;
  const name = normalizeEntryName(edit.value);
  if (!name && edit.mode !== 'rename') {
    cancelInlineEdit();
    return;
  }
  if (!isSafeEntryName(name)) {
    setInlineEditError(t('project.create.workspace.invalidFilePath'));
    return;
  }
  const oldPath = edit.anchorPath || '';
  const target = entryAt(oldPath);
  const path = joinEntryPath(parentDirectory(oldPath, target?.node_type), name);
  if (edit.mode === 'rename' && path === oldPath) {
    inlineEdit.value = null;
    return;
  }
  if (edit.mode !== 'rename') {
    if (
      files.value.some(
        (entry) =>
          entry.path === path ||
          entry.path.startsWith(`${path}/`) ||
          (entry.node_type !== 'directory' && path.startsWith(`${entry.path}/`)),
      )
    ) {
      setInlineEditError(t('project.create.workspace.pathConflict'));
      return;
    }
    files.value = [
      ...files.value,
      { path, content: '', node_type: edit.mode === 'create-directory' ? 'directory' : 'file' } as WorkspaceDraftEntry,
    ];
    inlineEdit.value = null;
    if (edit.mode === 'create-file') activateTab(path);
    return;
  }
  if (
    !oldPath ||
    path.startsWith(`${oldPath}/`) ||
    (path !== oldPath &&
      files.value.some(
        (entry) =>
          (entry.path !== oldPath && !entry.path.startsWith(`${oldPath}/`) && entry.path === path) ||
          entry.path.startsWith(`${path}/`) ||
          (entry.node_type !== 'directory' && path.startsWith(`${entry.path}/`)),
      ))
  ) {
    setInlineEditError(t('project.create.workspace.pathConflict'));
    return;
  }
  files.value = files.value.map((entry) =>
    entry.path === oldPath || entry.path.startsWith(`${oldPath}/`)
      ? { ...entry, path: `${path}${entry.path.slice(oldPath.length)}` }
      : entry,
  );
  migrateWorkspaceBuffers(oldPath, path);
  inlineEdit.value = null;
}
function confirmDelete() {
  const target = entryAt(deleteDialog.path);
  if (target?.node_type === 'directory' && deleteDialog.count > 1 && deleteDialog.stage === 'initial') {
    deleteDialog.stage = 'recursive';
    return;
  }
  files.value = files.value.filter(
    (entry) => entry.path !== deleteDialog.path && !entry.path.startsWith(`${deleteDialog.path}/`),
  );
  workspaceStore.closeFile(workspaceSessionKey, deleteDialog.path);
  deleteDialog.visible = false;
}
</script>
<style scoped>
.project-create-workspace {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}
</style>
