<template>
  <section class="project-create-workspace">
    <t-alert theme="info" :message="t('project.create.workspace.hint')" />
    <project-workspace-editor
      v-model:active-path="activePath"
      v-model:fullscreen="fullscreen"
      :active-buffer="editorActiveBuffer"
      :editor-aria-label="t('project.create.workspace.editorAriaLabel', { path: '{path}' })"
      :empty-description="t('project.create.workspace.filesEmpty')"
      :labels="editorLabels"
      :rows="editorRows"
      :selected-path="workspaceStore.activeSession.selectedKey"
      :tabs="editorTabs"
      :tabs-empty-description="t('project.create.workspace.selectFile')"
      :tree-title="t('project.create.workspace.filesTitle')"
      :root-label="t('project.create.workspace.rootLabel')"
      @close-tab="workspaceStore.closeFile"
      @context-action="(action, row) => handleContextAction(action, row?.path ?? null)"
      @select-entry="(row) => selectEntry(row.path)"
      @toggle-directory="(row) => toggleDirectory(row.path)"
      @update-content="updateContent"
    />

    <t-dialog
      v-model:visible="entryDialog.visible"
      :header="dialogTitle"
      :confirm-btn="t('project.create.actions.confirm')"
      :cancel-btn="t('project.create.actions.cancel')"
      @confirm="submitEntryDialog"
    >
      <t-form
        ><t-form-item :label="t('project.create.workspace.filePath')"
          ><t-input v-model="entryDialog.path" /></t-form-item
      ></t-form>
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
import { computed, reactive, ref, watch } from 'vue';

import { store } from '@/store/pinia';

import { useProjectPageContext } from '../shared/page-context';
import { type OpenedWorkspaceFile, useProjectWorkspaceStore } from '../store/workspace';
import type { ProjectWorkspaceManifestFile, ProjectWorkspaceTreeItem } from '../types/project';
import ProjectWorkspaceEditor, {
  type ProjectWorkspaceEditorBuffer,
  type ProjectWorkspaceEditorLabels,
  type ProjectWorkspaceEditorRow,
} from './ProjectWorkspaceEditor.vue';

defineOptions({ name: 'ProjectCreateWorkspaceEditor' });

type WorkspaceDraftEntry = ProjectWorkspaceManifestFile & { node_type?: 'directory' | 'file' };

const files = defineModel<WorkspaceDraftEntry[]>('files', { required: true });
const { t } = useProjectPageContext();
const workspaceStore = useProjectWorkspaceStore(store);
const fullscreen = ref(false);
const contextPath = ref<string | null>(null);
const entryDialog = reactive({
  error: '',
  mode: 'create' as 'create' | 'rename',
  nodeType: 'file' as 'directory' | 'file',
  path: '',
  visible: false,
});
const deleteDialog = reactive({ count: 0, path: '', stage: 'initial' as 'initial' | 'recursive', visible: false });

workspaceStore.activateSession('project-create-workspace');

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
  get: () => workspaceStore.activeSession.activeFileKey,
  set: (path: string) => activateTab(path),
});
const editorRows = computed<ProjectWorkspaceEditorRow[]>(() =>
  workspaceStore.visibleTreeRows.map((row) => ({
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

const editorTabs = computed<ProjectWorkspaceEditorBuffer[]>(() => workspaceStore.openedFiles.map(toEditorBuffer));
const editorActiveBuffer = computed<ProjectWorkspaceEditorBuffer | null>(() => {
  const file = workspaceStore.activeFile;
  return file ? toEditorBuffer(file) : null;
});
const dialogTitle = computed(() =>
  entryDialog.mode === 'rename'
    ? t('project.create.workspace.rename')
    : entryDialog.nodeType === 'directory'
      ? t('project.create.workspace.newFolder')
      : t('project.create.workspace.newFile'),
);

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

watch(
  files,
  (entries) => {
    workspaceStore.replaceTree(toTreeItems(entries));
    if (!workspaceStore.activeFile) {
      const firstFile = entries.find((entry) => entry.node_type !== 'directory');
      if (firstFile) activateTab(firstFile.path);
    }
  },
  { deep: true, immediate: true },
);

function normalizePath(path: string) {
  return path.trim().replace(/^\.\//, '').replace(/\/+$/, '');
}
function isSafePath(path: string) {
  return (
    Boolean(path) && !path.startsWith('/') && !path.split('/').some((part) => !part || part === '.' || part === '..')
  );
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
  workspaceStore.openFile(path, { content: entry.content, loaded: true, savedContent: entry.content });
}
function selectEntry(path: string) {
  const entry = entryAt(path);
  if (!entry) return;
  workspaceStore.selectNode(path);
  if (entry.node_type === 'directory') {
    toggleDirectory(path);
    return;
  }
  workspaceStore.openFile(path, { content: entry.content, loaded: true, savedContent: entry.content });
}
function toggleDirectory(path: string) {
  const expanded = workspaceStore.activeSession.expandedKeys.includes(path);
  workspaceStore.setExpanded(path, !expanded);
}
function updateContent(path: string, content: string) {
  const entry = entryAt(path);
  if (!entry || entry.node_type === 'directory') return;
  entry.content = content;
  workspaceStore.setFileContent(path, content);
}
function handleContextAction(
  action: 'create-file' | 'create-directory' | 'annotation' | 'rename' | 'delete',
  path: string | null,
) {
  if (action === 'annotation') return;
  contextPath.value = path;
  if (action === 'delete') {
    if (!path) return;
    deleteDialog.path = path;
    deleteDialog.count = files.value.filter((entry) => entry.path === path || entry.path.startsWith(`${path}/`)).length;
    deleteDialog.stage = 'initial';
    deleteDialog.visible = true;
    return;
  }
  entryDialog.mode = action === 'rename' ? 'rename' : 'create';
  entryDialog.nodeType =
    action === 'create-directory' ? 'directory' : entryAt(path || '')?.node_type === 'directory' ? 'directory' : 'file';
  entryDialog.error = '';
  entryDialog.path =
    action === 'rename'
      ? path || ''
      : (() => {
          const target = entryAt(path || '');
          const parent = parentDirectory(path || '', target?.node_type);
          return parent ? `${parent}/` : '';
        })();
  entryDialog.visible = true;
}
function submitEntryDialog() {
  const path = normalizePath(entryDialog.path);
  const oldPath = contextPath.value || '';
  if (!isSafePath(path)) {
    entryDialog.error = t('project.create.workspace.invalidFilePath');
    return;
  }
  if (entryDialog.mode === 'create') {
    if (
      files.value.some(
        (entry) => entry.path === path || entry.path.startsWith(`${path}/`) || path.startsWith(`${entry.path}/`),
      )
    ) {
      entryDialog.error = t('project.create.workspace.pathConflict');
      return;
    }
    files.value = [...files.value, { path, content: '', node_type: entryDialog.nodeType } as WorkspaceDraftEntry];
    entryDialog.visible = false;
    if (entryDialog.nodeType === 'file') activateTab(path);
    return;
  }
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
  entryDialog.visible = false;
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
  workspaceStore.closeFile(deleteDialog.path);
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
