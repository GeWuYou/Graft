<template>
  <div ref="workspaceRootRef" class="project-configuration-workspace" data-page-type="editor">
    <management-page-content>
      <management-page-header compact :title="pageHeaderTitle" :description="workspaceCopy.summaryDescription">
        <template #meta>
          <t-space break-line size="small">
            <t-tag :theme="runtimeTheme" variant="light-outline">
              {{ runtimeLabel }}
            </t-tag>
            <t-tag :theme="driftTheme" variant="light-outline">
              {{ driftLabel }}
            </t-tag>
            <t-tag :theme="refreshTheme" variant="light-outline">
              {{ refreshLabel }}
            </t-tag>
            <t-tag theme="default" variant="light-outline">
              {{ detailRecord?.ownership_mode || '-' }}
            </t-tag>
          </t-space>
        </template>
      </management-page-header>

      <t-loading :loading="workspaceLoading" size="small">
        <template v-if="workspaceReady">
          <section class="project-configuration-workspace__summary-strip">
            <t-card bordered>
              <template #header>
                <div class="project-configuration-workspace__section-head">
                  <div>
                    <h2>{{ workspaceCopy.summaryTitle }}</h2>
                    <p>{{ t('project.detail.configuration.title') }}</p>
                  </div>
                  <t-space size="small" break-line>
                    <t-button theme="default" variant="outline" @click="openSnapshotDrawer">
                      {{ workspaceCopy.snapshotAction }}
                    </t-button>
                  </t-space>
                </div>
              </template>

              <t-descriptions bordered size="small" :column="5">
                <t-descriptions-item :label="t('project.detail.configuration.ownershipMode')">
                  {{ detailRecord?.ownership_mode || '-' }}
                </t-descriptions-item>
                <t-descriptions-item :label="workspaceCopy.summaryWorkingDirectoryLabel">
                  <code>{{ detailRecord?.working_directory || '-' }}</code>
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.detail.configuration.driftStatus')">
                  {{ driftLabel }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.detail.configuration.refreshStatus')">
                  {{ refreshLabel }}
                </t-descriptions-item>
                <t-descriptions-item :label="workspaceCopy.summaryOpenTabsLabel">
                  {{ openTabs.length }}
                </t-descriptions-item>
              </t-descriptions>
            </t-card>
          </section>

          <section class="project-configuration-workspace__main-grid">
            <t-card class="project-configuration-workspace__tree-card" bordered>
              <template #header>
                <div class="project-configuration-workspace__section-head">
                  <div>
                    <h2>{{ workspaceCopy.fileTreeTitle }}</h2>
                    <p>{{ workspaceCopy.fileTreeHint }}</p>
                  </div>
                  <t-button theme="default" variant="outline" size="small" @click="showHiddenFiles = !showHiddenFiles">
                    {{ showHiddenFiles ? workspaceCopy.hideHiddenAction : workspaceCopy.showHiddenAction }}
                  </t-button>
                </div>
              </template>

              <t-alert
                v-if="treeError"
                class="project-configuration-workspace__tree-alert"
                theme="error"
                :message="treeError"
              />

              <t-loading :loading="treeLoading" size="small">
                <t-tree
                  v-if="treeData.length"
                  :data="treeData"
                  :actived="activeTreeValues"
                  :expanded="expandedTreeKeys"
                  :empty="workspaceCopy.filesEmpty"
                  activable
                  hover
                  lazy
                  line
                  expand-on-click-node
                  :load="treeLoadHandler"
                  @active="treeActiveHandler"
                  @expand="handleTreeExpand"
                >
                  <template #label="{ node }">
                    <div
                      class="project-configuration-workspace__tree-node"
                      :class="{
                        'project-configuration-workspace__tree-node--readonly':
                          !node.data.editable && node.data.node_type === 'file',
                      }"
                    >
                      <span class="project-configuration-workspace__tree-node-icon" aria-hidden="true">
                        <folder-icon v-if="node.data.node_type === 'directory'" />
                        <span
                          v-else-if="node.data.file_kind === 'compose'"
                          class="project-configuration-workspace__docker-icon"
                        >
                          <svg viewBox="0 0 24 24" role="presentation">
                            <path
                              d="M9.3 7.2h2.3v2.1H9.3zm2.7 0h2.3v2.1H12zm-5.4 3h2.3v2.1H6.6zm2.7 0h2.3v2.1H9.3zm2.7 0h2.3v2.1H12zm2.7 0h2.3v2.1h-2.3zm-1.2 3.2c.9 0 1.7-.2 2.4-.6.4.8 1.1 1.4 2 1.7 1.7.7 3.7.2 5-1.3-1-.4-1.7-1.4-1.7-2.6 0-1.2.7-2.2 1.7-2.6-.5-.7-1.4-1.2-2.4-1.2-.6 0-1.2.2-1.7.5-.6-1.4-1.9-2.4-3.5-2.6l-.8 1.3.7 1.1c-.2 0-.5-.1-.7-.1H5.4v4.4c0 1.2.5 2.4 1.4 3.2 1 .9 2.3 1.4 3.7 1.4h2z"
                              fill="currentColor"
                            />
                          </svg>
                        </span>
                        <command-icon v-else-if="node.data.file_kind === 'env'" />
                        <file-code-icon
                          v-else-if="node.data.file_kind === 'config' || node.data.file_kind === 'text'"
                        />
                        <file-icon v-else />
                      </span>
                      <span class="project-configuration-workspace__tree-node-main">
                        <span class="project-configuration-workspace__tree-node-title">{{ node.data.name }}</span>
                        <small>{{ node.data.relative_path || '.' }}</small>
                      </span>
                    </div>
                  </template>
                </t-tree>
                <t-empty v-else :description="workspaceCopy.filesEmpty" />
              </t-loading>
            </t-card>

            <div class="project-configuration-workspace__editor-column">
              <content-viewer-frame
                exit-fullscreen-label="Exit Fullscreen"
                fullscreen-label="Fullscreen"
                fullscreen-surface-padding="none"
                resize-handle-label="Resize Editor Height"
                storage-key="graft.project.configuration-workspace.editor.height"
                surface-padding="none"
              >
                <template #header>
                  <div class="project-configuration-workspace__editor-head">
                    <div>
                      <strong>{{ activeBuffer?.name || workspaceCopy.fileTreeTitle }}</strong>
                      <p>{{ activeBuffer?.path || workspaceCopy.selectFileToStart }}</p>
                    </div>
                  </div>
                </template>

                <template #header-actions>
                  <t-space size="small" break-line>
                    <t-button theme="default" variant="outline" :disabled="!activeBuffer" @click="reloadActiveFile">
                      {{ workspaceCopy.reloadAction }}
                    </t-button>
                    <t-button
                      theme="default"
                      variant="outline"
                      :loading="activeBuffer?.saving"
                      :disabled="!canSaveActiveBuffer"
                      @click="saveActiveFile"
                    >
                      {{ workspaceCopy.saveAction }}
                    </t-button>
                    <t-button theme="default" variant="outline" :loading="diffLoading" @click="runProjectDiff">
                      {{ workspaceCopy.diffAction }}
                    </t-button>
                    <t-button theme="default" variant="outline" :loading="validateLoading" @click="runProjectValidate">
                      {{ workspaceCopy.validateAction }}
                    </t-button>
                    <t-button theme="primary" :loading="deployLoading" @click="runProjectDeploy">
                      {{ workspaceCopy.deployAction }}
                    </t-button>
                  </t-space>
                </template>

                <div class="project-configuration-workspace__editor-surface">
                  <t-tabs
                    v-if="openTabBuffers.length"
                    v-model:value="activeTabPath"
                    class="project-configuration-workspace__tabs"
                    theme="card"
                  >
                    <t-tab-panel
                      v-for="tab in openTabBuffers"
                      :key="tab.path"
                      :value="tab.path"
                      removable
                      @remove="handleCloseTab(tab.path)"
                    >
                      <template #label>
                        <span class="project-configuration-workspace__tab-label">
                          <span v-if="isFileDirty(tab.path)" class="project-configuration-workspace__tab-dirty">●</span>
                          <span>{{ tab.name }}</span>
                        </span>
                      </template>
                    </t-tab-panel>
                  </t-tabs>

                  <t-alert
                    v-if="activeBuffer && !activeBuffer.editable"
                    class="project-configuration-workspace__editor-alert"
                    theme="warning"
                    :message="workspaceCopy.readonlyHint"
                  />
                  <t-alert
                    v-if="activeBuffer?.fileKind === 'env'"
                    class="project-configuration-workspace__editor-alert"
                    theme="info"
                    :message="workspaceCopy.envRedeployHint"
                  />
                  <t-alert
                    v-if="activeBuffer?.error"
                    class="project-configuration-workspace__editor-alert"
                    theme="error"
                    :message="activeBuffer.error"
                  />

                  <t-loading
                    v-if="activeBuffer"
                    class="project-configuration-workspace__editor-loading"
                    :loading="activeBuffer.loading"
                    size="small"
                  >
                    <project-monaco-surface
                      v-if="!activeBuffer.error"
                      v-model="activeBuffer.content"
                      class="project-configuration-workspace__monaco-editor"
                      :editor-aria-label="workspaceCopy.editorAriaLabel"
                      :language="activeBuffer.language"
                      :model-key="activeBuffer.path"
                      :options="editorOptions"
                      :read-only="!activeBuffer.editable"
                      test-id="workspace-monaco-editor"
                    />
                  </t-loading>
                  <t-empty v-else :description="workspaceCopy.tabsEmpty" />
                </div>
              </content-viewer-frame>

              <t-card class="project-configuration-workspace__feedback" bordered>
                <template #header>
                  <div class="project-configuration-workspace__section-head">
                    <div>
                      <h2>{{ workspaceCopy.feedbackTitle }}</h2>
                      <p>{{ workspaceCopy.feedbackHint }}</p>
                    </div>
                  </div>
                </template>

                <template v-if="hasFeedback">
                  <t-tabs v-model:value="feedbackTab" theme="card">
                    <t-tab-panel value="diff" :label="t('project.detail.configuration.diffTitle')">
                      <div v-if="diffResult" class="project-configuration-workspace__feedback-panel">
                        <t-alert
                          :theme="diffResult.has_changes ? 'warning' : 'success'"
                          :message="
                            diffResult.has_changes
                              ? t('project.detail.configuration.diffHasChanges')
                              : t('project.detail.configuration.diffNoChanges')
                          "
                        />
                        <t-space size="small" break-line>
                          <t-tag theme="default" variant="light-outline">
                            {{ t('project.detail.configuration.currentHash') }}: {{ diffResult.current_config_hash }}
                          </t-tag>
                          <t-tag theme="primary" variant="light-outline">
                            {{ t('project.detail.configuration.proposedHash') }}: {{ diffResult.proposed_config_hash }}
                          </t-tag>
                        </t-space>
                        <div v-if="diffResult.warnings?.length" class="project-configuration-workspace__warning-list">
                          <t-alert
                            v-for="warning in diffResult.warnings"
                            :key="warning"
                            theme="warning"
                            :message="warning"
                          />
                        </div>
                        <div class="project-configuration-workspace__diff-layout">
                          <div class="project-configuration-workspace__diff-files">
                            <button
                              v-for="file in diffResult.files"
                              :key="`${file.kind}-${file.path}`"
                              class="project-configuration-workspace__diff-file"
                              :class="{
                                'project-configuration-workspace__diff-file--active':
                                  selectedDiffFile?.path === file.path,
                              }"
                              type="button"
                              @click="selectedDiffFilePath = file.path"
                            >
                              <span>{{ file.path }}</span>
                              <t-tag :theme="file.changed ? 'warning' : 'success'" variant="light-outline">
                                {{
                                  file.changed
                                    ? t('project.detail.configuration.diffFileChanged')
                                    : t('project.detail.configuration.diffFileUnchanged')
                                }}
                              </t-tag>
                            </button>
                          </div>
                          <div class="project-configuration-workspace__diff-viewer">
                            <div v-if="selectedDiffFile" class="project-configuration-workspace__diff-meta">
                              <span>
                                {{ t('project.detail.configuration.currentHash') }}: {{ selectedDiffFile.current_hash }}
                              </span>
                              <span>
                                {{ t('project.detail.configuration.proposedHash') }}:
                                {{ selectedDiffFile.proposed_hash }}
                              </span>
                            </div>
                            <project-monaco-diff-surface
                              v-if="selectedDiffFile"
                              :editor-aria-label="workspaceCopy.diffViewerAriaLabel"
                              :language="resolveDiffFileLanguage(selectedDiffFile.kind, selectedDiffFile.path)"
                              :modified-key="`diff-modified-${selectedDiffFile.path}`"
                              :modified-value="selectedDiffFile.proposed_content"
                              :original-key="`diff-original-${selectedDiffFile.path}`"
                              :original-value="selectedDiffFile.current_content"
                              test-id="configuration-diff-viewer"
                            />
                            <t-empty v-else :description="workspaceCopy.selectDiffFile" />
                          </div>
                        </div>
                      </div>
                    </t-tab-panel>

                    <t-tab-panel value="validation" :label="t('project.detail.configuration.validationTitle')">
                      <div v-if="validateResult" class="project-configuration-workspace__feedback-panel">
                        <t-space size="small" break-line>
                          <t-tag theme="primary" variant="light-outline">
                            {{ t('project.detail.configuration.proposedHash') }}:
                            {{ validateResult.proposed_config_hash }}
                          </t-tag>
                          <t-tag theme="default" variant="light-outline">
                            {{ t('project.detail.configuration.declaredServices') }}:
                            {{ validateResult.declared_service_names.join(', ') || '-' }}
                          </t-tag>
                        </t-space>
                        <div
                          v-if="validateResult.warnings?.length"
                          class="project-configuration-workspace__warning-list"
                        >
                          <t-alert
                            v-for="warning in validateResult.warnings"
                            :key="warning"
                            theme="warning"
                            :message="warning"
                          />
                        </div>
                        <div class="project-configuration-workspace__readonly-viewer">
                          <project-monaco-surface
                            class="project-configuration-workspace__monaco-viewer"
                            :model-value="validateResult.normalized_compose_yaml"
                            :editor-aria-label="workspaceCopy.snapshotViewerAriaLabel"
                            language="yaml"
                            model-key="validation-normalized-yaml"
                            :options="readonlyOptions"
                            read-only
                            test-id="validation-monaco-viewer"
                          />
                        </div>
                      </div>
                    </t-tab-panel>
                  </t-tabs>
                </template>
                <t-empty v-else :description="workspaceCopy.feedbackHint" />
              </t-card>
            </div>
          </section>
        </template>

        <t-empty v-else-if="!workspaceError" :description="t('project.list.retry')" />
      </t-loading>
    </management-page-content>

    <t-drawer v-model:visible="snapshotDrawerVisible" :header="workspaceCopy.snapshotDrawerTitle" size="720px">
      <t-loading :loading="snapshotLoading" size="small">
        <template v-if="snapshotPreview">
          <t-descriptions bordered size="small" :column="1">
            <t-descriptions-item :label="t('project.detail.configuration.previewHash')">
              {{ snapshotPreview.config_hash }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('project.detail.configuration.previewUpdatedAt')">
              {{ formatProjectTime(locale, snapshotPreview.refreshed_at) }}
            </t-descriptions-item>
          </t-descriptions>
          <div class="project-configuration-workspace__drawer-viewer">
            <project-monaco-surface
              class="project-configuration-workspace__monaco-viewer"
              :model-value="snapshotPreview.normalized_compose_yaml"
              :editor-aria-label="workspaceCopy.snapshotViewerAriaLabel"
              language="yaml"
              model-key="snapshot-preview"
              :options="readonlyOptions"
              read-only
              test-id="snapshot-monaco-viewer"
            />
          </div>
        </template>
        <t-empty v-else :description="t('project.detail.configuration.previewEmpty')" />
      </t-loading>
    </t-drawer>

    <t-dialog
      v-model:visible="dialogState.visible"
      :header="dialogState.title"
      :close-on-overlay-click="false"
      :close-on-esc-keydown="true"
      @close="resolveDialog('cancel')"
    >
      <p class="project-configuration-workspace__dialog-body">{{ dialogState.body }}</p>
      <template #footer>
        <t-space size="small" break-line>
          <t-button
            v-for="button in dialogState.buttons"
            :key="button.result"
            :theme="button.theme"
            :variant="button.variant"
            @click="resolveDialog(button.result)"
          >
            {{ button.label }}
          </t-button>
        </t-space>
      </template>
    </t-dialog>
  </div>
</template>
<script setup lang="ts">
import { CommandIcon, FileCodeIcon, FileIcon, FolderIcon } from 'tdesign-icons-vue-next';
import type { TreeNodeValue, TreeProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { useRoute } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import ContentViewerFrame from '@/shared/components/viewer/ContentViewerFrame.vue';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { createLogger } from '@/utils/logger';

import {
  getProject,
  getProjectConfiguration,
  getProjectConfigurationPreview,
  getProjectFileContent,
  getProjectFiles,
  postProjectConfigurationDiff,
  postProjectConfigurationValidate,
  postProjectDeploy,
  putProjectFileContent,
} from '../../api/project';
import ProjectMonacoDiffSurface from '../../components/ProjectMonacoDiffSurface.vue';
import ProjectMonacoSurface from '../../components/ProjectMonacoSurface.vue';
import {
  canOpenWorkspaceFile,
  hasWorkspaceUnsavedChanges,
  normalizeWorkspaceContent,
  type ProjectWorkspaceMonacoLanguage,
  resolveWorkspaceFileName,
  resolveWorkspaceMonacoLanguage,
} from '../../shared/configuration-workspace';
import {
  formatProjectTime,
  projectDriftStatusLabel,
  projectDriftStatusTheme,
  projectRefreshStatusLabel,
  projectRefreshStatusTheme,
  projectRuntimeStatusLabel,
  projectRuntimeStatusTheme,
} from '../../shared/display';
import { useProjectPageContext } from '../../shared/page-context';
import type {
  ProjectConfigurationDiffResponse,
  ProjectConfigurationPreviewResponse,
  ProjectConfigurationValidateResponse,
  ProjectDetailResponseWithLifecycle,
  ProjectWorkspaceFileContentResponse,
  ProjectWorkspaceFileKind,
  ProjectWorkspaceTreeItem,
} from '../../types/project';
import { resolveConfigurationWorkspaceCopy } from './workspace-copy';

defineOptions({
  name: 'ProjectConfigurationWorkspaceIndex',
});

type FeedbackTab = 'diff' | 'validation';
type DialogResult = 'cancel' | 'continue-disk' | 'discard' | 'save' | 'save-and-continue';
type WorkspaceDialogButton = {
  label: string;
  result: DialogResult;
  theme: 'default' | 'primary';
  variant: 'base' | 'outline';
};
type WorkspaceTreeNode = ProjectWorkspaceTreeItem & {
  children?: WorkspaceTreeNode[] | boolean;
  value: string;
};
type WorkspaceOpenFile = {
  content: string;
  editable: boolean;
  error: string;
  fileKind: ProjectWorkspaceFileKind;
  language: ProjectWorkspaceMonacoLanguage;
  loaded: boolean;
  loading: boolean;
  name: string;
  path: string;
  savedContent: string;
  saving: boolean;
  sizeBytes?: number | null;
};

const logger = createLogger('project.configuration-workspace');
const route = useRoute();
const { locale, t } = useProjectPageContext();

const workspaceRootRef = ref<HTMLElement | null>(null);
const workspaceLoading = ref(false);
const workspaceError = ref('');
const workspaceReady = computed(() => Boolean(detailRecord.value && metadata.value && !workspaceError.value));
const treeLoading = ref(false);
const treeError = ref('');
const showHiddenFiles = ref(false);
const treeData = ref<WorkspaceTreeNode[]>([]);
const expandedTreeKeys = ref<TreeNodeValue[]>([]);
const activeTreeValues = ref<TreeNodeValue[]>([]);
const detailRecord = ref<ProjectDetailResponseWithLifecycle | null>(null);
const metadata = ref<Awaited<ReturnType<typeof getProjectConfiguration>> | null>(null);
const snapshotPreview = ref<ProjectConfigurationPreviewResponse | null>(null);
const snapshotLoading = ref(false);
const snapshotDrawerVisible = ref(false);
const feedbackTab = ref<FeedbackTab>('diff');
const diffResult = ref<ProjectConfigurationDiffResponse | null>(null);
const validateResult = ref<ProjectConfigurationValidateResponse | null>(null);
const diffLoading = ref(false);
const validateLoading = ref(false);
const deployLoading = ref(false);
const selectedDiffFilePath = ref('');
const openTabs = ref<string[]>([]);
const activeTabPath = ref('');
const openFileMap = reactive(new Map<string, WorkspaceOpenFile>());
const dialogState = reactive<{
  body: string;
  buttons: WorkspaceDialogButton[];
  resolver: ((result: DialogResult) => void) | null;
  title: string;
  visible: boolean;
}>({
  body: '',
  buttons: [],
  resolver: null,
  title: '',
  visible: false,
});

const editorOptions = {
  fontSize: 13,
  lineNumbers: 'on' as const,
  wordWrap: 'off' as const,
};

const readonlyOptions = {
  fontSize: 13,
  lineNumbers: 'on' as const,
  minimap: { enabled: false },
  readOnly: true,
  wordWrap: 'off' as const,
};

const workspaceCopy = computed(() => resolveConfigurationWorkspaceCopy(locale.value));
const projectId = computed(() => Number(route.params.id));
const fallbackDisplayName = computed(() => {
  const queryName = typeof route.query.name === 'string' ? route.query.name.trim() : '';
  return queryName;
});
const pageHeaderTitle = computed(() => {
  const suffix = t('project.detail.configuration.title');
  return detailRecord.value?.display_name
    ? `${detailRecord.value.display_name} · ${suffix}`
    : fallbackDisplayName.value
      ? `${fallbackDisplayName.value} · ${suffix}`
      : suffix;
});
const runtimeTheme = computed(() => projectRuntimeStatusTheme(detailRecord.value?.runtime_status));
const runtimeLabel = computed(() => projectRuntimeStatusLabel(t, detailRecord.value?.runtime_status));
const driftTheme = computed(() => projectDriftStatusTheme(metadata.value?.drift_status));
const driftLabel = computed(() =>
  metadata.value?.drift_status ? projectDriftStatusLabel(t, metadata.value.drift_status) : '-',
);
const refreshTheme = computed(() => projectRefreshStatusTheme(metadata.value?.last_refresh_status));
const refreshLabel = computed(() =>
  metadata.value?.last_refresh_status ? projectRefreshStatusLabel(t, metadata.value.last_refresh_status) : '-',
);
const openTabBuffers = computed(
  () => openTabs.value.map((path) => openFileMap.get(path)).filter(Boolean) as WorkspaceOpenFile[],
);
const activeBuffer = computed(() => (activeTabPath.value ? (openFileMap.get(activeTabPath.value) ?? null) : null));
const canSaveActiveBuffer = computed(() => Boolean(activeBuffer.value?.editable && !activeBuffer.value.saving));
const hasDirtyFiles = computed(() =>
  openTabBuffers.value.some((tab) => tab.editable && hasWorkspaceUnsavedChanges(tab.content, tab.savedContent)),
);
const hasFeedback = computed(() => Boolean(diffResult.value || validateResult.value));
const selectedDiffFile = computed(
  () =>
    diffResult.value?.files.find((file) => file.path === selectedDiffFilePath.value) ??
    diffResult.value?.files[0] ??
    null,
);
const treeActiveHandler: NonNullable<TreeProps['onActive']> = (value, context) => {
  handleTreeActive(value, context as unknown as { node: { data: WorkspaceTreeNode } });
};
const treeLoadHandler: NonNullable<TreeProps['load']> = (node) =>
  loadWorkspaceTreeChildren(node as unknown as { data: WorkspaceTreeNode });

onMounted(() => {
  window.addEventListener('keydown', handleWorkspaceKeydown);
  void loadWorkspace();
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleWorkspaceKeydown);
});

watch(showHiddenFiles, () => {
  void loadWorkspaceTreeRoot();
});

watch(
  openTabs,
  (nextTabs) => {
    if (!nextTabs.length) {
      activeTabPath.value = '';
      return;
    }
    if (!nextTabs.includes(activeTabPath.value)) {
      activeTabPath.value = nextTabs[nextTabs.length - 1];
    }
  },
  { deep: true },
);

async function loadWorkspace() {
  if (!Number.isFinite(projectId.value)) {
    workspaceError.value = t('project.list.retry');
    return;
  }

  workspaceLoading.value = true;
  workspaceError.value = '';
  try {
    const [detail, configurationMetadata] = await Promise.all([
      getProject(projectId.value),
      getProjectConfiguration(projectId.value),
    ]);
    detailRecord.value = detail;
    metadata.value = configurationMetadata;
    await loadWorkspaceTreeRoot();
  } catch (error) {
    logger.error('failed to load project configuration workspace', error);
    workspaceError.value = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
    MessagePlugin.error(workspaceError.value);
  } finally {
    workspaceLoading.value = false;
  }
}

async function loadWorkspaceTreeRoot() {
  treeLoading.value = true;
  treeError.value = '';
  try {
    const response = await getProjectFiles(projectId.value, {
      show_hidden: showHiddenFiles.value,
    });
    treeData.value = normalizeTreeItems(response.items);
    expandedTreeKeys.value = [];
    activeTreeValues.value = [];
    if (!activeTabPath.value) {
      const firstFile = findFirstFilePath(treeData.value);
      if (firstFile) {
        await openWorkspaceFile(firstFile);
      }
    }
  } catch (error) {
    treeError.value = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
    MessagePlugin.error(treeError.value);
  } finally {
    treeLoading.value = false;
  }
}

function normalizeTreeItems(items: ProjectWorkspaceTreeItem[]) {
  return items.map<WorkspaceTreeNode>((item) => ({
    ...item,
    children: item.node_type === 'directory' ? (item.has_children ? true : []) : undefined,
    value: item.relative_path,
  }));
}

function findFirstFilePath(nodes: WorkspaceTreeNode[]): string {
  for (const node of nodes) {
    if (node.node_type === 'file') {
      return node.relative_path;
    }
  }
  return '';
}

async function loadWorkspaceTreeChildren(node: { data: WorkspaceTreeNode }) {
  const currentNode = node.data;
  const response = await getProjectFiles(projectId.value, {
    path: currentNode.relative_path,
    show_hidden: showHiddenFiles.value,
  });
  const children = normalizeTreeItems(response.items);
  treeData.value = replaceTreeChildren(treeData.value, currentNode.relative_path, children);
  return children;
}

function replaceTreeChildren(
  nodes: WorkspaceTreeNode[],
  path: string,
  children: WorkspaceTreeNode[],
): WorkspaceTreeNode[] {
  return nodes.map((node) => {
    if (node.relative_path === path) {
      return { ...node, children };
    }
    if (Array.isArray(node.children)) {
      return { ...node, children: replaceTreeChildren(node.children, path, children) };
    }
    return node;
  });
}

function handleTreeActive(value: TreeNodeValue[], context: { node: { data: WorkspaceTreeNode } }) {
  activeTreeValues.value = value;
  const target = context.node.data;
  if (!canOpenWorkspaceFile(target)) {
    return;
  }
  void openWorkspaceFile(target.relative_path, target);
}

function handleTreeExpand(value: TreeNodeValue[]) {
  expandedTreeKeys.value = value;
}

async function openWorkspaceFile(path: string, source?: WorkspaceTreeNode) {
  if (!path) {
    return;
  }

  if (!openFileMap.has(path)) {
    openFileMap.set(path, {
      content: '',
      editable: Boolean(source?.editable),
      error: '',
      fileKind: source?.file_kind || 'text',
      language: resolveWorkspaceMonacoLanguage({
        fileKind: source?.file_kind,
        languageHint: source?.language_hint,
        path,
      }),
      loaded: false,
      loading: false,
      name: resolveWorkspaceFileName(path),
      path,
      savedContent: '',
      saving: false,
      sizeBytes: source?.size_bytes ?? null,
    });
    openTabs.value = [...openTabs.value, path];
  }

  activeTabPath.value = path;
  activeTreeValues.value = [path];
  await ensureWorkspaceFileLoaded(path, source);
}

async function ensureWorkspaceFileLoaded(path: string, source?: WorkspaceTreeNode) {
  const current = openFileMap.get(path);
  if (!current || current.loading || current.loaded) {
    return;
  }

  current.loading = true;
  current.error = '';
  try {
    const response = await getProjectFileContent(projectId.value, { path });
    hydrateOpenFileFromResponse(path, response, source);
  } catch (error) {
    current.error = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
  } finally {
    current.loading = false;
  }
}

function hydrateOpenFileFromResponse(
  requestedPath: string,
  response: ProjectWorkspaceFileContentResponse,
  source?: WorkspaceTreeNode,
) {
  const path = response.relative_path || requestedPath;
  const current =
    openFileMap.get(requestedPath) ??
    ({
      content: '',
      editable: response.editable,
      error: '',
      fileKind: response.file_kind,
      language: resolveWorkspaceMonacoLanguage({
        fileKind: response.file_kind,
        languageHint: response.language_hint,
        path,
      }),
      loaded: false,
      loading: false,
      name: resolveWorkspaceFileName(path),
      path,
      savedContent: '',
      saving: false,
    } satisfies WorkspaceOpenFile);

  current.content = normalizeWorkspaceContent(response.content);
  current.editable = response.editable;
  current.error = '';
  current.fileKind = response.file_kind || source?.file_kind || 'text';
  current.language = resolveWorkspaceMonacoLanguage({
    fileKind: current.fileKind,
    languageHint: response.language_hint ?? source?.language_hint,
    path,
  });
  current.loaded = true;
  current.loading = false;
  current.name = resolveWorkspaceFileName(path);
  current.path = path;
  current.savedContent = normalizeWorkspaceContent(response.content);
  current.sizeBytes = response.size_bytes ?? source?.size_bytes ?? null;
  openFileMap.set(path, current);

  if (path !== requestedPath) {
    openFileMap.delete(requestedPath);
    openTabs.value = openTabs.value.map((item) => (item === requestedPath ? path : item));
    if (activeTabPath.value === requestedPath) {
      activeTabPath.value = path;
    }
  }
}

function isFileDirty(path: string) {
  const current = openFileMap.get(path);
  return current ? hasWorkspaceUnsavedChanges(current.content, current.savedContent) : false;
}

async function saveActiveFile() {
  if (!activeBuffer.value) {
    return;
  }
  await saveWorkspaceFile(activeBuffer.value.path);
}

async function saveWorkspaceFile(path: string, options?: { silent?: boolean }) {
  const current = openFileMap.get(path);
  if (!current || !current.editable || current.saving) {
    return false;
  }

  current.saving = true;
  try {
    const normalizedContent = normalizeWorkspaceContent(current.content);
    await putProjectFileContent(projectId.value, { path }, { content: normalizedContent });
    current.content = normalizedContent;
    current.savedContent = normalizedContent;
    if (!options?.silent) {
      MessagePlugin.success(workspaceCopy.value.saveSuccess);
    }
    return true;
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, workspaceCopy.value.saveFailed));
    return false;
  } finally {
    current.saving = false;
  }
}

async function saveDirtyFiles() {
  const dirtyPaths = openTabBuffers.value
    .filter((tab) => tab.editable && hasWorkspaceUnsavedChanges(tab.content, tab.savedContent))
    .map((tab) => tab.path);
  if (!dirtyPaths.length) {
    return true;
  }

  for (const path of dirtyPaths) {
    const saved = await saveWorkspaceFile(path, { silent: true });
    if (!saved) {
      return false;
    }
  }
  MessagePlugin.success(workspaceCopy.value.saveSuccess);
  return true;
}

async function reloadActiveFile() {
  if (!activeBuffer.value) {
    return;
  }

  const buffer = activeBuffer.value;
  if (isFileDirty(buffer.path)) {
    const action = await openDialog({
      body: workspaceCopy.value.reloadConfirmBody,
      buttons: [
        { label: workspaceCopy.value.discardAction, result: 'discard', theme: 'primary', variant: 'base' },
        { label: workspaceCopy.value.cancelAction, result: 'cancel', theme: 'default', variant: 'outline' },
      ],
      title: workspaceCopy.value.reloadConfirmTitle,
    });
    if (action !== 'discard') {
      return;
    }
  }

  buffer.loading = true;
  buffer.error = '';
  try {
    const response = await getProjectFileContent(projectId.value, { path: buffer.path });
    hydrateOpenFileFromResponse(buffer.path, response);
  } catch (error) {
    buffer.error = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
  } finally {
    buffer.loading = false;
  }
}

async function handleCloseTab(path: string) {
  const current = openFileMap.get(path);
  if (!current) {
    return;
  }

  if (isFileDirty(path)) {
    const action = await openDialog({
      body: workspaceCopy.value.dirtyCloseBody,
      buttons: [
        { label: workspaceCopy.value.saveAction, result: 'save', theme: 'primary', variant: 'base' },
        { label: workspaceCopy.value.discardAction, result: 'discard', theme: 'default', variant: 'outline' },
        { label: workspaceCopy.value.cancelAction, result: 'cancel', theme: 'default', variant: 'outline' },
      ],
      title: workspaceCopy.value.dirtyCloseTitle,
    });
    if (action === 'cancel') {
      return;
    }
    if (action === 'save') {
      const saved = await saveWorkspaceFile(path);
      if (!saved) {
        return;
      }
    }
  }

  openFileMap.delete(path);
  openTabs.value = openTabs.value.filter((item) => item !== path);
}

async function runProjectDiff() {
  const proceed = await resolveProjectActionDirtyState('diff');
  if (!proceed) {
    return;
  }

  diffLoading.value = true;
  try {
    diffResult.value = await postProjectConfigurationDiff(projectId.value);
    selectedDiffFilePath.value = diffResult.value.files[0]?.path || '';
    feedbackTab.value = 'diff';
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.configuration.diffFailed')));
  } finally {
    diffLoading.value = false;
  }
}

async function runProjectValidate() {
  const proceed = await resolveProjectActionDirtyState('validate');
  if (!proceed) {
    return;
  }

  validateLoading.value = true;
  try {
    validateResult.value = await postProjectConfigurationValidate(projectId.value);
    feedbackTab.value = 'validation';
    MessagePlugin.success(t('project.detail.configuration.validateSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.configuration.validateFailed')));
  } finally {
    validateLoading.value = false;
  }
}

async function runProjectDeploy() {
  const proceed = await resolveProjectActionDirtyState('deploy');
  if (!proceed) {
    return;
  }

  deployLoading.value = true;
  try {
    const response = await postProjectDeploy(projectId.value);
    MessagePlugin.success(response.message || t('project.detail.configuration.deploySuccess'));
    diffResult.value = null;
    validateResult.value = null;
    snapshotPreview.value = null;
    await loadWorkspaceTreeRoot();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.configuration.deployFailed')));
  } finally {
    deployLoading.value = false;
  }
}

async function resolveProjectActionDirtyState(action: 'deploy' | 'diff' | 'validate') {
  if (!hasDirtyFiles.value) {
    return true;
  }

  const dialogResult = await openDialog(
    action === 'deploy'
      ? {
          body: workspaceCopy.value.deployDirtyBody,
          buttons: [
            { label: workspaceCopy.value.saveThenContinueAction, result: 'save', theme: 'primary', variant: 'base' },
            {
              label: workspaceCopy.value.deployContinueWithDiskAction,
              result: 'continue-disk',
              theme: 'default',
              variant: 'outline',
            },
            { label: workspaceCopy.value.cancelAction, result: 'cancel', theme: 'default', variant: 'outline' },
          ],
          title: workspaceCopy.value.deployDirtyTitle,
        }
      : {
          body: workspaceCopy.value.dirtyProjectActionBody,
          buttons: [
            {
              label: workspaceCopy.value.saveAndContinueAction,
              result: 'save-and-continue',
              theme: 'primary',
              variant: 'base',
            },
            {
              label: workspaceCopy.value.continueWithDiskAction,
              result: 'continue-disk',
              theme: 'default',
              variant: 'outline',
            },
            { label: workspaceCopy.value.cancelAction, result: 'cancel', theme: 'default', variant: 'outline' },
          ],
          title: workspaceCopy.value.dirtyProjectActionTitle,
        },
  );

  if (dialogResult === 'cancel') {
    return false;
  }
  if (dialogResult === 'continue-disk') {
    return true;
  }
  return saveDirtyFiles();
}

async function openSnapshotDrawer() {
  snapshotDrawerVisible.value = true;
  if (snapshotPreview.value || !Number.isFinite(projectId.value)) {
    return;
  }
  snapshotLoading.value = true;
  try {
    snapshotPreview.value = await getProjectConfigurationPreview(projectId.value);
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.retry')));
  } finally {
    snapshotLoading.value = false;
  }
}

function resolveDiffFileLanguage(kind: string | undefined, path: string) {
  return resolveWorkspaceMonacoLanguage({
    fileKind: kind === 'compose' || kind === 'env' ? kind : 'config',
    path,
  });
}

function openDialog(config: { body: string; buttons: WorkspaceDialogButton[]; title: string }) {
  if (dialogState.resolver) {
    dialogState.resolver('cancel');
  }

  dialogState.body = config.body;
  dialogState.buttons = config.buttons;
  dialogState.title = config.title;
  dialogState.visible = true;

  return new Promise<DialogResult>((resolve) => {
    dialogState.resolver = resolve;
  });
}

function resolveDialog(result: DialogResult) {
  if (!dialogState.resolver) {
    return;
  }

  const resolver = dialogState.resolver;
  dialogState.body = '';
  dialogState.buttons = [];
  dialogState.title = '';
  dialogState.visible = false;
  dialogState.resolver = null;
  resolver(result);
}

function handleWorkspaceKeydown(event: KeyboardEvent) {
  const root = workspaceRootRef.value;
  if (!root || !activeBuffer.value || !canSaveActiveBuffer.value) {
    return;
  }

  const target = event.target;
  if (!(target instanceof Node) || !root.contains(target)) {
    return;
  }

  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
    event.preventDefault();
    void saveActiveFile();
  }
}
</script>
<style scoped lang="less">
.project-configuration-workspace {
  min-width: 0;
}

.project-configuration-workspace__summary-strip,
.project-configuration-workspace__main-grid {
  margin-top: var(--graft-density-gap-16);
}

.project-configuration-workspace__section-head {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.project-configuration-workspace__section-head h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
}

.project-configuration-workspace__section-head p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-4) 0 0;
}

.project-configuration-workspace__main-grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: minmax(260px, 300px) minmax(0, 1fr);
}

.project-configuration-workspace__tree-card,
.project-configuration-workspace__editor-column,
.project-configuration-workspace__feedback-panel,
.project-configuration-workspace__diff-layout,
.project-configuration-workspace__diff-viewer,
.project-configuration-workspace__readonly-viewer,
.project-configuration-workspace__drawer-viewer {
  min-height: 0;
  min-width: 0;
}

.project-configuration-workspace__tree-alert,
.project-configuration-workspace__editor-alert {
  margin-bottom: var(--graft-density-gap-12);
}

.project-configuration-workspace__tree-node {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.project-configuration-workspace__tree-node--readonly {
  opacity: 0.6;
}

.project-configuration-workspace__tree-node-icon {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  flex: 0 0 auto;
  justify-content: center;
}

.project-configuration-workspace__docker-icon {
  color: #2496ed;
  display: inline-flex;
  height: 16px;
  width: 16px;
}

.project-configuration-workspace__docker-icon svg {
  fill: currentcolor;
  height: 100%;
  width: 100%;
}

.project-configuration-workspace__tree-node-main {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-width: 0;
}

.project-configuration-workspace__tree-node-title {
  color: var(--td-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-configuration-workspace__tree-node-main small {
  color: var(--td-text-color-placeholder);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-configuration-workspace__editor-column {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.project-configuration-workspace__editor-head strong {
  color: var(--td-text-color-primary);
  display: block;
  font: var(--td-font-title-small);
}

.project-configuration-workspace__editor-head p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-4) 0 0;
  word-break: break-all;
}

.project-configuration-workspace__editor-surface {
  block-size: 100%;
  min-block-size: 560px;
  min-inline-size: 0;
}

.project-configuration-workspace__tabs {
  margin-bottom: var(--graft-density-gap-12);
}

.project-configuration-workspace__tab-label {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-4);
}

.project-configuration-workspace__tab-dirty {
  color: var(--td-warning-color-6);
  font: var(--td-font-body-small);
  line-height: 1;
}

.project-configuration-workspace__editor-loading,
.project-configuration-workspace__monaco-editor,
.project-configuration-workspace__monaco-viewer {
  block-size: 100%;
  display: block;
  min-block-size: 0;
  min-inline-size: 0;
}

.project-configuration-workspace__warning-list,
.project-configuration-workspace__feedback-panel {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.project-configuration-workspace__diff-layout {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: minmax(220px, 260px) minmax(0, 1fr);
  min-height: 420px;
}

.project-configuration-workspace__diff-files {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.project-configuration-workspace__diff-file {
  align-items: center;
  background: var(--td-bg-color-container);
  border: 1px solid transparent;
  border-radius: var(--td-radius-default);
  color: inherit;
  cursor: pointer;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
  text-align: left;
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease;
}

.project-configuration-workspace__diff-file:hover {
  border-color: var(--td-brand-color-5);
}

.project-configuration-workspace__diff-file--active {
  background: var(--td-brand-color-1);
  border-color: var(--td-brand-color-6);
}

.project-configuration-workspace__diff-viewer {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.project-configuration-workspace__diff-meta {
  color: var(--td-text-color-secondary);
  display: flex;
  flex-wrap: wrap;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-12);
}

.project-configuration-workspace__readonly-viewer,
.project-configuration-workspace__drawer-viewer {
  block-size: 480px;
  margin-top: var(--graft-density-gap-12);
}

.project-configuration-workspace__dialog-body {
  margin: 0;
}

@media (width <= 1024px) {
  .project-configuration-workspace__main-grid,
  .project-configuration-workspace__diff-layout {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
