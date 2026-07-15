<template>
  <section class="project-workspace-editor" :class="{ 'project-workspace-editor--fullscreen': fullscreen }">
    <div ref="mainGridRef" class="project-workspace-editor__main-grid" :style="sidebarGridStyle">
      <aside class="project-workspace-editor__sidebar">
        <t-card bordered class="project-workspace-editor__browser-card">
          <template #header>
            <div class="project-workspace-editor__browser-heading">
              <div>
                <h2>{{ treeTitle }}</h2>
                <p>{{ rootLabel }}</p>
              </div>
              <slot name="tree-actions" />
            </div>
          </template>

          <slot name="tree-feedback" />
          <div class="project-workspace-editor__tree graft-scrollbar" @contextmenu.prevent="openMenu(null, $event)">
            <template v-if="rows.length">
              <template v-for="row in rows" :key="row.path">
                <div
                  class="project-workspace-editor__tree-row"
                  :class="{
                    'project-workspace-editor__tree-row--active': row.path === activePath,
                    'project-workspace-editor__tree-row--selected': row.path === selectedPath,
                    'project-workspace-editor__tree-row--readonly': row.nodeType === 'file' && row.readOnly,
                  }"
                  :style="{ '--workspace-tree-depth': String(row.depth) }"
                  @contextmenu.prevent.stop="openMenu(row, $event)"
                >
                  <button
                    v-if="row.nodeType === 'directory'"
                    class="project-workspace-editor__tree-expander"
                    type="button"
                    :aria-expanded="Boolean(row.expanded)"
                    @click.stop="$emit('toggle-directory', row)"
                  >
                    <span>{{ row.expanded ? '▾' : '▸' }}</span>
                  </button>
                  <span v-else class="project-workspace-editor__tree-expander-placeholder" />

                  <button
                    class="project-workspace-editor__tree-entry"
                    :data-testid="row.testId"
                    type="button"
                    @click="$emit('select-entry', row)"
                  >
                    <span class="project-workspace-editor__browser-icon" aria-hidden="true">
                      <folder-icon v-if="row.nodeType === 'directory'" />
                      <command-icon v-else-if="row.fileKind === 'env'" />
                      <file-code-icon
                        v-else-if="row.fileKind === 'compose' || row.fileKind === 'config' || row.fileKind === 'text'"
                      />
                      <file-icon v-else />
                    </span>
                    <t-tooltip v-if="row.tooltip" :content="row.tooltip" placement="top-left" theme="light">
                      <span class="project-workspace-editor__browser-title">{{ row.name }}</span>
                    </t-tooltip>
                    <span v-else class="project-workspace-editor__browser-title">{{ row.name }}</span>
                  </button>

                  <div class="project-workspace-editor__tree-actions">
                    <slot name="entry-actions" :row="row" />
                    <button
                      class="project-workspace-editor__tree-menu-trigger"
                      type="button"
                      :aria-label="labels.entryActions.replace('{path}', row.path)"
                      @click.stop="openMenu(row, $event, true)"
                    >
                      <ellipsis-icon />
                    </button>
                  </div>
                </div>
                <p v-if="row.error && row.expanded" class="project-workspace-editor__tree-error">{{ row.error }}</p>
              </template>
            </template>
            <t-empty v-else :description="emptyDescription" />
          </div>
        </t-card>
      </aside>

      <div
        v-if="sidebarResizable"
        class="project-workspace-editor__splitter"
        role="separator"
        :aria-label="sidebarResizeAriaLabel"
        aria-orientation="vertical"
        tabindex="0"
        @keydown.end.prevent="setSidebarWidth(resolveSidebarMaxWidth())"
        @keydown.home.prevent="setSidebarWidth(sidebarMinWidth)"
        @keydown.left.prevent="nudgeSidebarWidth(-16)"
        @keydown.right.prevent="nudgeSidebarWidth(16)"
        @pointerdown.prevent="startSidebarResize"
      >
        <span class="project-workspace-editor__splitter-grip" />
      </div>

      <div class="project-workspace-editor__editor-stack">
        <content-viewer-frame
          :default-height="editorDefaultHeight"
          :exit-fullscreen-label="labels.exitFullscreen"
          :fill-height="fullscreen"
          :fullscreen-label="labels.fullscreen"
          :resizable="!fullscreen"
          resize-handle-label="Resize Editor Height"
          :show-header="false"
          :show-fullscreen-button="false"
          :storage-key="editorHeightStorageKey"
          fullscreen-surface-padding="none"
          surface-padding="none"
        >
          <div class="project-workspace-editor__editor-surface">
            <div v-if="tabs.length" class="project-workspace-editor__tabbar">
              <t-tabs :value="activePath" theme="card" @change="changeActivePath">
                <t-tab-panel
                  v-for="tab in tabs"
                  :key="tab.path"
                  :value="tab.path"
                  removable
                  :destroy-on-hide="false"
                  @remove="$emit('close-tab', tab.path)"
                >
                  <template #label>
                    <t-dropdown
                      trigger="context-menu"
                      :hide-after-item-click="true"
                      :min-column-width="128"
                      :data-testid="tabTestId?.(tab.path)"
                    >
                      <span class="project-workspace-editor__tab-label">
                        <span v-if="tab.dirty" class="project-workspace-editor__tab-dirty">●</span>
                        <span>{{ tab.name }}</span>
                      </span>
                      <template #dropdown>
                        <t-dropdown-menu>
                          <t-dropdown-item
                            :data-testid="tabActionTestId?.(tab.path, 'refresh')"
                            @click="emitTabAction('refresh', tab.path)"
                          >
                            {{ labels.refresh }}
                          </t-dropdown-item>
                          <t-dropdown-item
                            :data-testid="tabActionTestId?.(tab.path, 'close-left')"
                            @click="emitTabAction('close-left', tab.path)"
                          >
                            {{ labels.closeLeft }}
                          </t-dropdown-item>
                          <t-dropdown-item
                            :data-testid="tabActionTestId?.(tab.path, 'close-right')"
                            @click="emitTabAction('close-right', tab.path)"
                          >
                            {{ labels.closeRight }}
                          </t-dropdown-item>
                          <t-dropdown-item
                            :data-testid="tabActionTestId?.(tab.path, 'close-other')"
                            @click="emitTabAction('close-other', tab.path)"
                          >
                            {{ labels.closeOther }}
                          </t-dropdown-item>
                          <t-dropdown-item
                            :data-testid="tabActionTestId?.(tab.path, 'close-all')"
                            @click="emitTabAction('close-all', tab.path)"
                          >
                            {{ labels.closeAll }}
                          </t-dropdown-item>
                        </t-dropdown-menu>
                      </template>
                    </t-dropdown>
                  </template>
                </t-tab-panel>
              </t-tabs>
              <div class="project-workspace-editor__tab-actions">
                <slot name="editor-actions" />
                <t-tooltip :content="fullscreen ? labels.exitFullscreen : labels.fullscreen" theme="light">
                  <t-button
                    data-testid="workspace-fullscreen-toggle"
                    theme="default"
                    variant="text"
                    shape="square"
                    size="small"
                    @click="$emit('update:fullscreen', !fullscreen)"
                  >
                    <template #icon><slot name="fullscreen-icon" :fullscreen="fullscreen" /></template>
                  </t-button>
                </t-tooltip>
              </div>
            </div>
            <slot name="editor-feedback" :buffer="activeBuffer" />
            <div v-if="activeBuffer" class="project-workspace-editor__editor-stage">
              <t-loading class="project-workspace-editor__editor-loading" :loading="activeBuffer.loading" size="small">
                <project-monaco-surface
                  v-if="!activeBuffer.error"
                  ref="activeEditorRef"
                  :model-value="activeBuffer.content"
                  :editor-aria-label="editorAriaLabel.replace('{path}', activeBuffer.path)"
                  :language="activeBuffer.language"
                  :model-key="activeBuffer.modelKey || activeBuffer.path"
                  :options="editorOptions"
                  :read-only="Boolean(activeBuffer.readOnly)"
                  test-id="workspace-monaco-editor"
                  @update:model-value="$emit('update-content', activeBuffer.path, $event)"
                />
              </t-loading>
            </div>
            <t-empty v-else :description="tabsEmptyDescription" />
          </div>
        </content-viewer-frame>
      </div>
    </div>

    <div
      v-if="contextMenu.visible"
      ref="contextMenuRef"
      class="project-workspace-editor__context-menu"
      :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
      role="menu"
      @keydown.down.prevent="moveMenuFocus(1)"
      @keydown.end.prevent="moveMenuFocus(-1, true)"
      @keydown.esc.prevent="closeContextMenu(true)"
      @keydown.home.prevent="moveMenuFocus(1, true)"
      @keydown.up.prevent="moveMenuFocus(-1)"
    >
      <button type="button" role="menuitem" @click="emitContextAction('create-file')">{{ labels.newFile }}</button>
      <button type="button" role="menuitem" @click="emitContextAction('create-directory')">
        {{ labels.newFolder }}
      </button>
      <button
        v-if="showAnnotationAction"
        type="button"
        role="menuitem"
        :data-testid="contextMenu.row?.testId ? `${contextMenu.row.testId}-annotation` : undefined"
        :disabled="!contextMenu.row"
        @click="emitContextAction('annotation')"
      >
        {{ labels.annotationAction }}
      </button>
      <button type="button" role="menuitem" :disabled="!contextMenu.row" @click="emitContextAction('rename')">
        {{ labels.rename }}
      </button>
      <button type="button" role="menuitem" :disabled="!contextMenu.row" @click="emitContextAction('delete')">
        {{ labels.delete }}
      </button>
    </div>
  </section>
</template>
<script setup lang="ts">
import { CommandIcon, EllipsisIcon, FileCodeIcon, FileIcon, FolderIcon } from 'tdesign-icons-vue-next';
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue';

import ContentViewerFrame from '@/shared/components/viewer/ContentViewerFrame.vue';

import type { ProjectWorkspaceMonacoLanguage } from '../shared/configuration-workspace';
import ProjectMonacoSurface from './ProjectMonacoSurface.vue';

defineOptions({ name: 'ProjectWorkspaceEditor' });

export type ProjectWorkspaceEditorRow = {
  depth: number;
  error?: string;
  expanded?: boolean;
  fileKind?: string;
  name: string;
  nodeType: 'directory' | 'file';
  path: string;
  readOnly?: boolean;
  testId?: string;
  tooltip?: string;
};

export type ProjectWorkspaceEditorBuffer = {
  content: string;
  dirty?: boolean;
  error?: string;
  language: ProjectWorkspaceMonacoLanguage;
  loading?: boolean;
  modelKey?: string;
  name: string;
  path: string;
  readOnly?: boolean;
};

export type ProjectWorkspaceEditorLabels = {
  annotationAction?: string;
  closeAll: string;
  closeLeft: string;
  closeOther: string;
  closeRight: string;
  delete: string;
  entryActions: string;
  exitFullscreen: string;
  fullscreen: string;
  newFile: string;
  newFolder: string;
  refresh: string;
  rename: string;
};

const props = withDefaults(
  defineProps<{
    activeBuffer: ProjectWorkspaceEditorBuffer | null;
    activePath: string;
    editorAriaLabel: string;
    editorDefaultHeight?: number;
    editorHeightStorageKey?: string;
    emptyDescription: string;
    fullscreen?: boolean;
    labels: ProjectWorkspaceEditorLabels;
    rows: ProjectWorkspaceEditorRow[];
    selectedPath?: string;
    sidebarMaxWidth?: number;
    sidebarMinWidth?: number;
    sidebarResizable?: boolean;
    sidebarResizeAriaLabel?: string;
    sidebarWidth?: number;
    showAnnotationAction?: boolean;
    tabActionTestId?: (path: string, action: string) => string;
    tabTestId?: (path: string) => string;
    tabs: ProjectWorkspaceEditorBuffer[];
    tabsEmptyDescription: string;
    treeTitle: string;
    rootLabel: string;
  }>(),
  {
    editorDefaultHeight: 520,
    editorHeightStorageKey: undefined,
    fullscreen: false,
    sidebarMaxWidth: 360,
    sidebarMinWidth: 208,
    sidebarResizable: false,
    sidebarResizeAriaLabel: 'Resize file tree',
    sidebarWidth: 256,
    showAnnotationAction: false,
    selectedPath: '',
    tabActionTestId: undefined,
    tabTestId: undefined,
  },
);

const emit = defineEmits<{
  'close-tab': [path: string];
  'context-action': [
    action: 'create-file' | 'create-directory' | 'annotation' | 'rename' | 'delete',
    row: ProjectWorkspaceEditorRow | null,
  ];
  'editor-ready': [editor: unknown];
  'select-entry': [row: ProjectWorkspaceEditorRow];
  'tab-action': [action: 'refresh' | 'close-left' | 'close-right' | 'close-other' | 'close-all', path: string];
  'toggle-directory': [row: ProjectWorkspaceEditorRow];
  'update:activePath': [path: string];
  'update:fullscreen': [fullscreen: boolean];
  'update-content': [path: string, content: string];
  'update:sidebarWidth': [width: number];
}>();

const contextMenuRef = ref<HTMLElement | null>(null);
const activeEditorRef = ref<InstanceType<typeof ProjectMonacoSurface> | null>(null);
const mainGridRef = ref<HTMLElement | null>(null);
let contextMenuTrigger: HTMLElement | null = null;
let sidebarResizeStartWidth = 0;
let sidebarResizeStartX = 0;
let removeSidebarResizeListeners: (() => void) | null = null;
const contextMenu = reactive<{ row: ProjectWorkspaceEditorRow | null; visible: boolean; x: number; y: number }>({
  row: null,
  visible: false,
  x: 0,
  y: 0,
});
const editorOptions = { fontSize: 13, lineNumbers: 'on' as const, wordWrap: 'off' as const };
const sidebarGridStyle = computed(() => ({
  '--project-workspace-sidebar-width': `${clampSidebarWidth(props.sidebarWidth)}px`,
}));

function changeActivePath(path: string | number) {
  emit('update:activePath', String(path));
}

function openMenu(row: ProjectWorkspaceEditorRow | null, event: MouseEvent, focusMenu = false) {
  contextMenu.row = row;
  contextMenu.x = event.clientX;
  contextMenu.y = event.clientY;
  contextMenuTrigger = event.currentTarget instanceof HTMLElement ? event.currentTarget : null;
  contextMenu.visible = true;
  if (focusMenu)
    void nextTick(() => contextMenuRef.value?.querySelector<HTMLButtonElement>('[role="menuitem"]')?.focus());
}

function emitContextAction(action: 'create-file' | 'create-directory' | 'annotation' | 'rename' | 'delete') {
  emit('context-action', action, contextMenu.row);
  contextMenu.visible = false;
}

function emitTabAction(action: 'refresh' | 'close-left' | 'close-right' | 'close-other' | 'close-all', path: string) {
  emit('tab-action', action, path);
}

function moveMenuFocus(direction: 1 | -1, boundary = false) {
  const items = Array.from(contextMenuRef.value?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? []).filter(
    (item) => !item.disabled,
  );
  if (!items.length) return;
  const currentIndex = items.findIndex((item) => item === document.activeElement);
  const nextIndex = boundary
    ? direction > 0
      ? 0
      : items.length - 1
    : (currentIndex + direction + items.length) % items.length;
  items[nextIndex]?.focus();
}

function closeContextMenu(restoreFocus = false) {
  contextMenu.visible = false;
  if (restoreFocus) void nextTick(() => contextMenuTrigger?.focus());
}

function resolveSidebarMaxWidth() {
  const gridWidth = mainGridRef.value?.clientWidth || 1280;
  return Math.min(props.sidebarMaxWidth, Math.max(props.sidebarMinWidth, gridWidth - 420));
}

function clampSidebarWidth(value: number) {
  return Math.max(props.sidebarMinWidth, Math.min(resolveSidebarMaxWidth(), Math.round(value)));
}

function setSidebarWidth(value: number) {
  emit('update:sidebarWidth', clampSidebarWidth(value));
}

function nudgeSidebarWidth(delta: number) {
  setSidebarWidth(props.sidebarWidth + delta);
}

function startSidebarResize(event: PointerEvent) {
  if (!props.sidebarResizable || typeof window === 'undefined') return;
  sidebarResizeStartX = event.clientX;
  sidebarResizeStartWidth = clampSidebarWidth(props.sidebarWidth);
  const handlePointerMove = (moveEvent: PointerEvent) => {
    setSidebarWidth(sidebarResizeStartWidth + moveEvent.clientX - sidebarResizeStartX);
  };
  const handlePointerUp = () => stopSidebarResize();
  window.addEventListener('pointermove', handlePointerMove);
  window.addEventListener('pointerup', handlePointerUp, { once: true });
  removeSidebarResizeListeners = () => {
    window.removeEventListener('pointermove', handlePointerMove);
    window.removeEventListener('pointerup', handlePointerUp);
  };
}

function stopSidebarResize() {
  removeSidebarResizeListeners?.();
  removeSidebarResizeListeners = null;
}

defineExpose({
  getActiveEditor: () => activeEditorRef.value,
});

watch(activeEditorRef, (editor) => {
  if (editor) emit('editor-ready', editor);
});

function handleDocumentClick() {
  closeContextMenu();
}

document.addEventListener('click', handleDocumentClick);
onBeforeUnmount(() => {
  document.removeEventListener('click', handleDocumentClick);
  stopSidebarResize();
});
</script>
<style scoped>
.project-workspace-editor {
  min-width: 0;
}

.project-workspace-editor__main-grid {
  display: grid;
  gap: var(--graft-density-gap-12) 0;
  grid-template-columns: var(--project-workspace-sidebar-width) 6px minmax(0, 1fr);
  min-height: 520px;
}

.project-workspace-editor__sidebar,
.project-workspace-editor__editor-stack {
  min-width: 0;
}

.project-workspace-editor__splitter {
  align-items: center;
  cursor: col-resize;
  display: flex;
  justify-content: center;
  min-height: 0;
  position: relative;
  touch-action: none;
  width: 6px;
}

.project-workspace-editor__splitter-grip {
  background: color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  border-radius: var(--td-radius-default);
  height: 56px;
  transition: background-color 0.2s ease;
  width: 4px;
}

.project-workspace-editor__splitter:hover .project-workspace-editor__splitter-grip,
.project-workspace-editor__splitter:focus-visible .project-workspace-editor__splitter-grip {
  background: color-mix(in srgb, var(--td-brand-color-6) 55%, var(--td-component-stroke));
}

.project-workspace-editor__browser-card {
  height: 100%;
}

.project-workspace-editor__browser-card :deep(.t-card__body) {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.project-workspace-editor__browser-heading {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
}

.project-workspace-editor__browser-heading h2,
.project-workspace-editor__browser-heading p {
  margin: 0;
}

.project-workspace-editor__browser-heading p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin-top: var(--graft-density-gap-2);
}

.project-workspace-editor__tree {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
  min-height: 0;
  overflow: auto;
}

.project-workspace-editor__tree-row {
  align-items: center;
  border-radius: var(--td-radius-default);
  display: grid;
  gap: var(--graft-density-gap-4);
  grid-template-columns: 18px minmax(0, 1fr) auto;
  min-width: 0;
  padding-left: calc(var(--workspace-tree-depth, 0) * var(--graft-density-gap-14));
}

.project-workspace-editor__tree-row--active {
  background: color-mix(in srgb, var(--td-brand-color-6) 10%, transparent);
}

.project-workspace-editor__tree-row--selected {
  background: color-mix(in srgb, var(--td-brand-color-6) 10%, transparent);
}

.project-workspace-editor__tree-row--readonly {
  opacity: 0.68;
}

.project-workspace-editor__tree-expander,
.project-workspace-editor__tree-expander-placeholder {
  align-items: center;
  color: var(--td-text-color-placeholder);
  display: inline-flex;
  height: 24px;
  justify-content: center;
  width: 18px;
}

.project-workspace-editor__tree-expander {
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  cursor: pointer;
  padding: 0;
}

.project-workspace-editor__tree-entry {
  align-items: center;
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  color: inherit;
  cursor: pointer;
  display: flex;
  gap: var(--graft-density-gap-8);
  min-height: 30px;
  min-width: 0;
  padding: 0 var(--graft-density-gap-8) 0 0;
  text-align: left;
  width: 100%;
}

.project-workspace-editor__browser-icon {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  flex: 0 0 auto;
  height: 16px;
  justify-content: center;
  width: 16px;
}

.project-workspace-editor__browser-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-workspace-editor__tree-actions {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-4);
}

.project-workspace-editor__tree-menu-trigger {
  align-items: center;
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  display: inline-flex;
  height: 30px;
  justify-content: center;
  padding: 0;
  width: 30px;
}

.project-workspace-editor__tree-menu-trigger:hover,
.project-workspace-editor__tree-menu-trigger:focus-visible {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.project-workspace-editor__tree-error {
  color: var(--td-error-color-6);
  font: var(--td-font-body-small);
  margin: 0;
  padding-left: var(--graft-density-gap-24);
}

.project-workspace-editor__editor-stack {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  border-radius: var(--td-radius-default);
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.project-workspace-editor__editor-surface {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}

.project-workspace-editor__tabbar {
  background: var(--td-bg-color-container);
  border-bottom: 1px solid var(--td-component-border);
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  min-width: 0;
}

.project-workspace-editor__tabbar :deep(.t-tabs) {
  display: flex;
  flex: 0 0 auto;
  flex-direction: column;
  min-height: 0;
}

.project-workspace-editor__tabbar :deep(.t-tabs__header) {
  background: transparent;
  border-bottom: 0;
  margin: 0;
  padding: 0;
}

.project-workspace-editor__tabbar :deep(.t-tabs__nav) {
  align-items: stretch;
  min-width: 0;
}

.project-workspace-editor__tabbar :deep(.t-tabs__nav-container) {
  flex: 1 1 auto;
  min-width: 0;
}

.project-workspace-editor__tabbar :deep(.t-tabs__nav-item) {
  min-height: 42px;
}

.project-workspace-editor__tabbar :deep(.t-tabs__content) {
  display: none;
}

.project-workspace-editor__tab-actions {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  height: 100%;
  padding: 0 var(--graft-density-gap-8);
}

.project-workspace-editor__tab-actions > :deep(span) {
  align-items: center;
  display: inline-flex;
  height: 100%;
}

.project-workspace-editor__tab-label {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-6);
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-workspace-editor__tab-dirty {
  color: var(--td-warning-color-6);
  font-size: var(--td-font-size-body-small);
}

.project-workspace-editor__editor-stage,
.project-workspace-editor__editor-loading {
  display: flex;
  flex: 1;
  min-height: 420px;
  min-width: 0;
}

.project-workspace-editor__editor-loading :deep(.t-loading__parent),
.project-workspace-editor__editor-loading :deep(.t-loading__content),
.project-workspace-editor__editor-loading :deep(.t-loading__wrap),
.project-workspace-editor__editor-stage :deep(.project-monaco-surface) {
  height: 100%;
  width: 100%;
}

.project-workspace-editor__context-menu {
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

.project-workspace-editor__context-menu button {
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-primary);
  cursor: pointer;
  min-height: 32px;
  padding: 0 var(--graft-density-gap-8);
  text-align: left;
}

.project-workspace-editor__context-menu button:hover:not(:disabled) {
  background: var(--td-bg-color-container-hover);
}

.project-workspace-editor--fullscreen {
  background: var(--td-bg-color-page);
  height: 100dvh;
  inset: 0;
  isolation: isolate;
  pointer-events: auto;
  position: fixed;
  width: 100dvw;
  z-index: 2400;
}

.project-workspace-editor--fullscreen .project-workspace-editor__main-grid {
  height: 100%;
  min-height: 0;
}

.project-workspace-editor--fullscreen .project-workspace-editor__sidebar,
.project-workspace-editor--fullscreen .project-workspace-editor__editor-stack,
.project-workspace-editor--fullscreen :deep(.content-viewer-frame),
.project-workspace-editor--fullscreen :deep(.content-viewer-frame__panel),
.project-workspace-editor--fullscreen :deep(.content-viewer-frame__surface),
.project-workspace-editor--fullscreen :deep(.project-monaco-surface) {
  height: 100%;
}

.project-workspace-editor--fullscreen :deep(.content-viewer-frame__panel) {
  border: 0;
  border-radius: 0;
  box-shadow: none;
}

@media (width <= 1024px) {
  .project-workspace-editor__main-grid {
    grid-template-columns: 1fr;
    min-height: 0;
  }

  .project-workspace-editor__tree {
    min-height: 220px;
  }

  .project-workspace-editor__splitter {
    display: none;
  }

  .project-workspace-editor__editor-stage,
  .project-workspace-editor__editor-loading {
    min-height: 360px;
  }
}
</style>
