<template>
  <div
    ref="workspaceRootRef"
    class="project-configuration-workspace"
    :class="{ 'project-configuration-workspace--fullscreen': workspaceFullscreen }"
    data-page-type="editor"
  >
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
                </div>
              </template>

              <t-descriptions bordered size="small" :column="5">
                <t-descriptions-item :label="t('project.detail.configuration.ownershipMode')">
                  {{ detailRecord?.ownership_mode || '-' }}
                </t-descriptions-item>
                <t-descriptions-item :label="workspaceCopy.summaryWorkingDirectoryLabel">
                  <t-tooltip :content="detailRecord?.workspace_path || '-'" placement="top-left" theme="light">
                    <code
                      :aria-label="detailRecord?.workspace_path || '-'"
                      class="project-configuration-workspace__summary-technical"
                      data-testid="workspace-working-directory"
                    >
                      {{ workingDirectoryDisplay }}
                    </code>
                  </t-tooltip>
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.detail.configuration.driftStatus')">
                  {{ driftLabel }}
                </t-descriptions-item>
                <t-descriptions-item :label="workspaceCopy.summaryCurrentPathLabel">
                  <t-tooltip :content="currentWorkspacePathLabel" placement="top-left" theme="light">
                    <code
                      :aria-label="currentWorkspacePathLabel"
                      class="project-configuration-workspace__summary-technical"
                      data-testid="workspace-current-path"
                    >
                      {{ currentWorkspacePathDisplay }}
                    </code>
                  </t-tooltip>
                </t-descriptions-item>
                <t-descriptions-item :label="workspaceCopy.summaryOpenTabsLabel">
                  {{ openTabs.length }}
                </t-descriptions-item>
              </t-descriptions>
            </t-card>
          </section>

          <project-workspace-editor
            ref="workspaceEditorRef"
            v-model:active-path="activeTabPath"
            v-model:fullscreen="workspaceFullscreen"
            :active-buffer="workspaceEditorActiveBuffer"
            :editor-default-height="editorFrameHeight"
            :editor-aria-label="workspaceCopy.editorAriaLabel"
            :editor-height-storage-key="EDITOR_HEIGHT_STORAGE_KEY"
            :empty-description="workspaceCopy.filesEmpty"
            :labels="workspaceEditorLabels"
            :rows="workspaceEditorRows"
            :selected-path="workspaceStore.activeSession.selectedKey"
            :sidebar-max-width="SIDEBAR_MAX_WIDTH"
            :sidebar-min-width="SIDEBAR_MIN_WIDTH"
            :sidebar-resizable="isSidebarResizable"
            :sidebar-resize-aria-label="workspaceCopy.resizeFileTreeAriaLabel"
            :sidebar-width="sidebarWidth"
            show-annotation-action
            :tab-action-test-id="workspaceFileTabMenuItemTestId"
            :tab-test-id="workspaceFileTabMenuTestId"
            :tabs="workspaceEditorTabs"
            :tabs-empty-description="workspaceCopy.tabsEmpty"
            :tree-title="workspaceCopy.fileTreeTitle"
            :root-label="workspaceCopy.workspaceRootLabel"
            @close-tab="handleCloseTab"
            @context-action="handleWorkspaceEditorContextAction"
            @editor-ready="setActiveWorkspaceEditor"
            @select-entry="handleWorkspaceEditorEntry"
            @tab-action="handleWorkspaceEditorTabAction"
            @toggle-directory="toggleWorkspaceEditorDirectory"
            @update-content="updateWorkspaceEditorContent"
            @update:sidebar-width="updateSidebarWidth"
          >
            <template #tree-actions>
              <t-tooltip
                :content="showHiddenFiles ? workspaceCopy.hideHiddenAction : workspaceCopy.showHiddenAction"
                placement="bottom"
                theme="light"
              >
                <t-button
                  class="project-configuration-workspace__tree-toolbar-button"
                  theme="default"
                  variant="text"
                  shape="square"
                  size="small"
                  :data-testid="showHiddenFiles ? 'workspace-hide-hidden-toggle' : 'workspace-show-hidden-toggle'"
                  @click="showHiddenFiles = !showHiddenFiles"
                >
                  <template #icon>
                    <browse-off-icon v-if="showHiddenFiles" />
                    <browse-icon v-else />
                  </template>
                </t-button>
              </t-tooltip>
            </template>
            <template #tree-feedback>
              <t-alert
                v-if="browserError"
                class="project-configuration-workspace__browser-alert"
                theme="error"
                :message="browserError"
              />
              <t-loading :loading="browserLoading" size="small" />
            </template>
            <template #editor-actions>
              <t-tooltip :content="workspaceCopy.reloadAction" theme="light"
                ><span
                  ><t-button
                    theme="default"
                    variant="text"
                    shape="square"
                    size="small"
                    :disabled="!activeBuffer"
                    @click="reloadActiveFile"
                    ><template #icon><refresh-icon /></template
                    ><span class="project-configuration-workspace__sr-only">{{
                      workspaceCopy.reloadAction
                    }}</span></t-button
                  ></span
                ></t-tooltip
              >
              <t-tooltip :content="workspaceCopy.saveAction" theme="light"
                ><span
                  ><t-button
                    theme="default"
                    variant="text"
                    shape="square"
                    size="small"
                    :loading="Boolean(activeBuffer?.saving)"
                    :disabled="!canSaveActiveBuffer"
                    @click="saveActiveFile"
                    ><template #icon><save-icon /></template
                    ><span class="project-configuration-workspace__sr-only">{{
                      workspaceCopy.saveAction
                    }}</span></t-button
                  ></span
                ></t-tooltip
              >
              <t-tooltip :content="workspaceCopy.saveAllAction" theme="light"
                ><span
                  ><t-button
                    theme="default"
                    variant="text"
                    shape="square"
                    size="small"
                    :loading="saveConfirmLoading && pendingWorkspaceAction === 'save-all'"
                    :disabled="!canSaveAllBuffers"
                    @click="saveAllFiles"
                    ><template #icon><file-copy-icon /></template
                    ><span class="project-configuration-workspace__sr-only">{{
                      workspaceCopy.saveAllAction
                    }}</span></t-button
                  ></span
                ></t-tooltip
              >
              <t-tooltip :content="workspaceCopy.validateAction" theme="light"
                ><span
                  ><t-button
                    theme="default"
                    variant="text"
                    shape="square"
                    size="small"
                    :loading="syntaxCheckLoading"
                    :disabled="!activeBuffer"
                    @click="runCurrentFileValidation"
                    ><template #icon><check-circle-icon /></template
                    ><span class="project-configuration-workspace__sr-only">{{
                      workspaceCopy.validateAction
                    }}</span></t-button
                  ></span
                ></t-tooltip
              >
              <t-tooltip :content="workspaceCopy.deployAction" theme="light"
                ><span
                  ><t-button
                    theme="primary"
                    variant="text"
                    shape="square"
                    size="small"
                    :loading="deployLoading"
                    @click="runProjectDeploy"
                    ><template #icon><cloud-upload-icon /></template
                    ><span class="project-configuration-workspace__sr-only">{{
                      workspaceCopy.deployAction
                    }}</span></t-button
                  ></span
                ></t-tooltip
              >
            </template>
            <template #fullscreen-icon="{ fullscreen }"
              ><fullscreen-exit-icon v-if="fullscreen" /><fullscreen-icon v-else
            /></template>
            <template #editor-feedback>
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
            </template>
          </project-workspace-editor>
        </template>

        <t-empty v-else-if="!workspaceError" :description="t('project.list.retry')" />
      </t-loading>
    </management-page-content>

    <t-dialog
      v-model:visible="resultDialogVisible"
      :dialog-class-name="resultDialogClassName"
      :dialog-style="resultDialogStyle"
      :close-btn="!resultDialogFullscreen"
      :footer="false"
      :header="false"
      :mode="resultDialogFullscreen ? 'full-screen' : 'modal'"
      placement="center"
      :close-on-overlay-click="false"
      :close-on-esc-keydown="true"
      destroy-on-close
      :on-opened="handleResultDialogOpened"
      :top="resultDialogTop"
      :width="resultDialogWidth"
    >
      <div class="project-configuration-workspace__result-dialog">
        <div class="project-configuration-workspace__result-dialog-header">
          <div class="project-configuration-workspace__section-head">
            <div class="project-configuration-workspace__result-dialog-title-block">
              <h2>{{ resultDialogTitle }}</h2>
              <p v-if="resultDialogDescription">{{ resultDialogDescription }}</p>
              <div
                v-if="resultDialogSummaryItems.length"
                class="project-configuration-workspace__result-dialog-summary"
              >
                <span
                  v-for="item in resultDialogSummaryItems"
                  :key="item.label"
                  class="project-configuration-workspace__result-dialog-summary-pill"
                >
                  <span class="project-configuration-workspace__result-dialog-summary-label">{{ item.label }}</span>
                  <strong class="project-configuration-workspace__result-dialog-summary-value">{{ item.value }}</strong>
                </span>
              </div>
            </div>
            <t-space size="small" class="project-configuration-workspace__result-dialog-actions">
              <t-button
                theme="default"
                variant="outline"
                size="small"
                :disabled="!canNavigateResultIssues"
                data-testid="configuration-result-prev"
                @click="navigateResultIssue('previous')"
              >
                <template #icon><arrow-up-icon /></template>
                {{ workspaceCopy.validatePreviousAction }}
              </t-button>
              <t-button
                theme="default"
                variant="outline"
                size="small"
                :disabled="!canNavigateResultIssues"
                data-testid="configuration-result-next"
                @click="navigateResultIssue('next')"
              >
                <template #icon><arrow-down-icon /></template>
                {{ workspaceCopy.validateNextAction }}
              </t-button>
              <t-button
                theme="default"
                variant="outline"
                size="small"
                data-testid="configuration-result-fullscreen-toggle"
                @click="toggleResultDialogFullscreen"
              >
                {{ resultDialogFullscreen ? workspaceCopy.exitFullscreenAction : workspaceCopy.fullscreenAction }}
              </t-button>
            </t-space>
          </div>
        </div>

        <div
          v-if="resultDialogMode === 'diff'"
          class="project-configuration-workspace__diff-surface"
          :class="{ 'project-configuration-workspace__diff-surface--single': !showDiffSidebar }"
          data-testid="configuration-diff-modal"
        >
          <div v-if="showDiffSidebar" class="project-configuration-workspace__diff-sidebar graft-scrollbar">
            <div class="project-configuration-workspace__diff-sidebar-head">
              <span class="project-configuration-workspace__diff-sidebar-title">{{ workspaceCopy.diffTreeTitle }}</span>
              <span class="project-configuration-workspace__diff-sidebar-caption">
                {{ workspaceCopy.workspaceRootLabel }}
              </span>
            </div>
            <div
              v-for="row in diffTreeRows"
              :key="row.path"
              class="project-configuration-workspace__tree-row project-configuration-workspace__tree-row--diff"
              :class="{
                'project-configuration-workspace__tree-row--active':
                  row.type === 'file' && selectedDiffFile?.path === row.path,
              }"
              :style="{ '--workspace-tree-depth': String(row.depth) }"
            >
              <span class="project-configuration-workspace__tree-expander-placeholder" />
              <button
                class="project-configuration-workspace__tree-entry"
                :data-testid="diffFileTestId(row.path)"
                type="button"
                :disabled="row.type !== 'file'"
                @click="row.type === 'file' && (selectedDiffFilePath = row.path)"
              >
                <span class="project-configuration-workspace__browser-icon" aria-hidden="true">
                  <folder-icon v-if="row.type === 'directory'" />
                  <span v-else-if="row.file?.kind === 'compose'" class="project-configuration-workspace__docker-icon">
                    <svg viewBox="0 0 24 24" role="presentation">
                      <path
                        d="M9.3 7.2h2.3v2.1H9.3zm2.7 0h2.3v2.1H12zm-5.4 3h2.3v2.1H6.6zm2.7 0h2.3v2.1H9.3zm2.7 0h2.3v2.1H12zm2.7 0h2.3v2.1h-2.3zm-1.2 3.2c.9 0 1.7-.2 2.4-.6.4.8 1.1 1.4 2 1.7 1.7.7 3.7.2 5-1.3-1-.4-1.7-1.4-1.7-2.6 0-1.2.7-2.2 1.7-2.6-.5-.7-1.4-1.2-2.4-1.2-.6 0-1.2.2-1.7.5-.6-1.4-1.9-2.4-3.5-2.6l-.8 1.3.7 1.1c-.2 0-.5-.1-.7-.1H5.4v4.4c0 1.2.5 2.4 1.4 3.2 1 .9 2.3 1.4 3.7 1.4h2z"
                        fill="currentColor"
                      />
                    </svg>
                  </span>
                  <command-icon v-else-if="row.file?.kind === 'env'" />
                  <file-code-icon v-else />
                </span>
                <span class="project-configuration-workspace__browser-main">
                  <span class="project-configuration-workspace__browser-title">{{ row.name }}</span>
                </span>
              </button>
            </div>
          </div>

          <div class="project-configuration-workspace__diff-stage">
            <t-alert theme="warning" :message="t('project.detail.configuration.diffHasChanges')" />
            <div v-if="diffResult?.warnings?.length" class="project-configuration-workspace__warning-list">
              <t-alert v-for="warning in diffResult.warnings" :key="warning" theme="warning" :message="warning" />
            </div>
            <div v-if="selectedDiffFile" class="project-configuration-workspace__diff-pane-heads">
              <div class="project-configuration-workspace__diff-pane-head">
                <span class="project-configuration-workspace__diff-pane-label">
                  {{ t('project.detail.configuration.currentHash') }}
                </span>
                <t-tooltip :content="selectedDiffFile.current_hash" placement="top-left" theme="light">
                  <code
                    class="project-configuration-workspace__hash-text"
                    data-testid="configuration-diff-current-hash"
                  >
                    {{ formatWorkspaceHash(selectedDiffFile.current_hash) }}
                  </code>
                </t-tooltip>
              </div>
              <div class="project-configuration-workspace__diff-pane-head">
                <span class="project-configuration-workspace__diff-pane-label">
                  {{ t('project.detail.configuration.proposedHash') }}
                </span>
                <t-tooltip :content="selectedDiffFile.proposed_hash" placement="top-left" theme="light">
                  <code
                    class="project-configuration-workspace__hash-text"
                    data-testid="configuration-diff-proposed-hash"
                  >
                    {{ formatWorkspaceHash(selectedDiffFile.proposed_hash) }}
                  </code>
                </t-tooltip>
              </div>
            </div>
            <div class="project-configuration-workspace__result-viewer">
              <project-monaco-diff-surface
                v-if="selectedDiffFile && diffViewerReady"
                ref="diffViewerRef"
                class="project-configuration-workspace__monaco-viewer"
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

        <div
          v-else-if="syntaxValidationResult"
          class="project-configuration-workspace__feedback-panel"
          :class="{ 'project-configuration-workspace__feedback-panel--with-sidebar': showSyntaxSidebar }"
        >
          <div v-if="showSyntaxSidebar" class="project-configuration-workspace__diff-sidebar graft-scrollbar">
            <div class="project-configuration-workspace__diff-sidebar-head">
              <span class="project-configuration-workspace__diff-sidebar-title">
                {{ workspaceCopy.syntaxFileTreeTitle }}
              </span>
              <span class="project-configuration-workspace__diff-sidebar-caption">
                {{ workspaceCopy.workspaceRootLabel }}
              </span>
            </div>
            <div
              v-for="row in syntaxTreeRows"
              :key="row.path"
              class="project-configuration-workspace__tree-row project-configuration-workspace__tree-row--diff"
              :class="{
                'project-configuration-workspace__tree-row--active':
                  row.type === 'file' && syntaxValidationFile?.path === row.path,
                'project-configuration-workspace__tree-row--error': row.type === 'file',
              }"
              :style="{ '--workspace-tree-depth': String(row.depth) }"
            >
              <span class="project-configuration-workspace__tree-expander-placeholder" />
              <button
                class="project-configuration-workspace__tree-entry"
                type="button"
                :disabled="row.type !== 'file'"
                @click="row.type === 'file' && (selectedSyntaxFilePath = row.path)"
              >
                <span class="project-configuration-workspace__browser-icon" aria-hidden="true">
                  <folder-icon v-if="row.type === 'directory'" />
                  <file-code-icon v-else />
                </span>
                <span class="project-configuration-workspace__browser-main">
                  <span class="project-configuration-workspace__browser-title">{{ row.name }}</span>
                </span>
              </button>
            </div>
          </div>
          <div class="project-configuration-workspace__syntax-stage">
            <t-alert v-if="!showSyntaxSidebar" theme="error" :message="workspaceCopy.fileValidationEmbeddedHint" />
            <t-alert v-else theme="warning" :message="workspaceCopy.batchFileValidationRiskBody" />
            <div class="project-configuration-workspace__result-viewer">
              <project-monaco-surface
                ref="syntaxViewerRef"
                class="project-configuration-workspace__monaco-viewer"
                :model-value="syntaxValidationResult.content"
                :editor-aria-label="workspaceCopy.syntaxViewerAriaLabel"
                :language="syntaxValidationResult.language"
                :markers="currentResultIssues"
                :model-key="syntaxValidationResult.modelKey"
                :options="readonlyOptions"
                read-only
                test-id="syntax-monaco-viewer"
              />
            </div>
          </div>
        </div>

        <div v-if="resultDialogMode === 'diff'" class="project-configuration-workspace__result-dialog-footer">
          <t-space size="small">
            <t-button
              theme="default"
              variant="outline"
              data-testid="configuration-diff-cancel"
              @click="cancelDiffPreview"
            >
              {{ workspaceCopy.cancelAction }}
            </t-button>
            <t-button
              theme="primary"
              :loading="saveConfirmLoading"
              data-testid="configuration-diff-confirm-save"
              @click="confirmDiffPreview"
            >
              {{ diffConfirmActionLabel }}
            </t-button>
          </t-space>
        </div>

        <div v-else-if="syntaxValidationResult" class="project-configuration-workspace__result-dialog-footer">
          <t-space size="small">
            <t-button
              theme="default"
              variant="outline"
              data-testid="configuration-syntax-cancel"
              @click="resultDialogVisible = false"
            >
              {{ workspaceCopy.cancelAction }}
            </t-button>
            <t-button
              theme="primary"
              :loading="saveConfirmLoading"
              data-testid="configuration-syntax-confirm-save"
              @click="confirmSyntaxValidation"
            >
              {{ syntaxConfirmActionLabel }}
            </t-button>
          </t-space>
        </div>
      </div>
    </t-dialog>

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
            :data-testid="`workspace-dialog-${button.result}`"
            :theme="button.theme"
            :variant="button.variant"
            @click="resolveDialog(button.result)"
          >
            {{ button.label }}
          </t-button>
        </t-space>
      </template>
    </t-dialog>

    <t-dialog
      v-model:visible="annotationDialogState.visible"
      :header="workspaceCopy.annotationAction"
      :close-on-overlay-click="false"
      :confirm-btn="{ content: workspaceCopy.saveAction, loading: annotationDialogState.saving }"
      :cancel-btn="{ content: workspaceCopy.cancelAction }"
      @confirm="saveWorkspaceAnnotation"
      @close="closeWorkspaceAnnotationDialog"
    >
      <t-textarea
        v-model="annotationDialogState.value"
        :autosize="{ minRows: 4, maxRows: 8 }"
        :placeholder="workspaceItemTooltip(annotationDialogState.target)"
      />
    </t-dialog>

    <t-dialog
      v-model:visible="workspaceEntryDialog.visible"
      :header="t('project.create.workspace.filePath')"
      :confirm-btn="t('project.create.actions.confirm')"
      :cancel-btn="workspaceCopy.cancelAction"
      @confirm="submitWorkspaceEntryDialog"
    >
      <t-input v-model="workspaceEntryDialog.path" :placeholder="t('project.create.workspace.filePathPlaceholder')" />
      <t-alert v-if="workspaceEntryDialog.error" theme="error" :message="workspaceEntryDialog.error" />
    </t-dialog>
    <t-dialog
      v-model:visible="workspaceDeleteDialog.visible"
      :header="
        workspaceDeleteDialog.stage === 'recursive'
          ? t('project.create.workspace.recursiveDeleteTitle')
          : t('project.create.workspace.delete')
      "
      :confirm-btn="
        workspaceDeleteDialog.stage === 'recursive'
          ? t('project.create.workspace.recursiveDeleteConfirm')
          : t('project.create.actions.confirm')
      "
      :cancel-btn="workspaceCopy.cancelAction"
      @confirm="confirmWorkspaceEntryDelete"
    >
      <p>
        {{
          workspaceDeleteDialog.stage === 'recursive'
            ? t('project.create.workspace.recursiveDeleteBody', {
                path: workspaceDeleteDialog.path,
              })
            : t('project.create.workspace.deleteBody', { path: workspaceDeleteDialog.path })
        }}
      </p>
    </t-dialog>
  </div>
</template>
<script setup lang="ts">
import {
  ArrowDownIcon,
  ArrowUpIcon,
  BrowseIcon,
  BrowseOffIcon,
  CheckCircleIcon,
  CloudUploadIcon,
  CommandIcon,
  FileCodeIcon,
  FileCopyIcon,
  FolderIcon,
  FullscreenExitIcon,
  FullscreenIcon,
  RefreshIcon,
  SaveIcon,
} from 'tdesign-icons-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { useRoute } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { store } from '@/store/pinia';
import { createLogger } from '@/utils/logger';

import {
  deleteProjectWorkspaceEntry,
  getProject,
  getProjectConfiguration,
  getProjectFileContent,
  getProjectFiles,
  postProjectDeploy,
  postProjectWorkspaceEntry,
  postProjectWorkspaceRename,
  putProjectFileAnnotation,
  putProjectFileContent,
} from '../../api/project';
import ProjectMonacoDiffSurface from '../../components/ProjectMonacoDiffSurface.vue';
import ProjectMonacoSurface from '../../components/ProjectMonacoSurface.vue';
import ProjectWorkspaceEditor, {
  type ProjectWorkspaceEditorBuffer,
  type ProjectWorkspaceEditorLabels,
  type ProjectWorkspaceEditorRow,
} from '../../components/ProjectWorkspaceEditor.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import {
  hasWorkspaceUnsavedChanges,
  normalizeTextBlock,
  normalizeWorkspaceContent,
  type ProjectWorkspaceMonacoLanguage,
  resolveWorkspaceFileName,
  resolveWorkspaceMonacoLanguage,
} from '../../shared/configuration-workspace';
import {
  projectDriftStatusLabel,
  projectDriftStatusTheme,
  projectRuntimeStatusLabel,
  projectRuntimeStatusTheme,
} from '../../shared/display';
import { buildDetailTitleWithFallback } from '../../shared/navigation';
import { useProjectPageContext } from '../../shared/page-context';
import { formatProjectMonacoDebugMessage, isProjectMonacoDebugEnabled } from '../../shared/project-monaco-debug';
import { useProjectWorkspaceStore } from '../../store/workspace';
import type {
  ProjectDetailResponseWithLifecycle,
  ProjectWorkspaceFileContentResponse,
  ProjectWorkspaceFileKind,
  ProjectWorkspaceTreeItem,
} from '../../types/project';
import { resolveConfigurationWorkspaceCopy, resolveConfigurationWorkspaceCopyKey } from './workspace-copy';

defineOptions({
  name: 'ProjectConfigurationWorkspaceIndex',
});

type ResultDialogMode = 'diff' | 'syntax';
type DialogResult = 'cancel' | 'continue-disk' | 'discard' | 'save' | 'save-and-continue';
type PendingWorkspaceAction = 'deploy' | 'save-all' | 'save-current';
type WorkspaceDialogButton = {
  label: string;
  result: DialogResult;
  theme: 'default' | 'primary';
  variant: 'base' | 'outline';
};
type WorkspaceListItem = ProjectWorkspaceTreeItem;
type DiffTreeRow = {
  depth: number;
  file: WorkspacePreviewDiffFile | null;
  name: string;
  path: string;
  type: 'directory' | 'file';
};
type WorkspacePreviewDiffFile = {
  changed: boolean;
  current_content: string;
  current_hash: string;
  display_path: string;
  kind: ProjectWorkspaceFileKind;
  path: string;
  proposed_content: string;
  proposed_hash: string;
};
type WorkspacePreviewDiffResult = {
  files: WorkspacePreviewDiffFile[];
  has_changes: boolean;
  warnings: string[];
};
type WorkspaceSyntaxMarker = {
  endColumn: number;
  endLineNumber: number;
  message: string;
  severity: number;
  startColumn: number;
  startLineNumber: number;
};
type WorkspaceDiffLineChange = {
  modifiedEndLineNumber: number;
  modifiedStartLineNumber: number;
  originalEndLineNumber: number;
  originalStartLineNumber: number;
};
type WorkspaceSyntaxValidationResult = {
  content: string;
  fileName: string;
  language: ProjectWorkspaceMonacoLanguage;
  markerCount: number;
  markers: WorkspaceSyntaxMarker[];
  modelKey: string;
  path: string;
};
type WorkspaceSyntaxTreeRow = {
  depth: number;
  file: WorkspaceSyntaxValidationResult | null;
  name: string;
  path: string;
  type: 'directory' | 'file';
};
type MonacoViewerHandle = {
  getModelKey?: () => string;
  getLineChanges?: () => WorkspaceDiffLineChange[];
  getMarkers?: () => WorkspaceSyntaxMarker[];
  navigateDiff?: (direction: 'next' | 'previous') => boolean;
  relayout: () => Promise<void>;
  revealFirstDiff?: () => boolean;
  revealLineChange?: (change: WorkspaceDiffLineChange | null | undefined) => boolean;
  revealMarker?: (marker: WorkspaceSyntaxMarker | null | undefined) => boolean;
  waitForDiagnostics?: (options?: { quietMs?: number; timeoutMs?: number }) => Promise<WorkspaceSyntaxMarker[]>;
};
type WorkspaceEditorHandle = { getActiveEditor: () => MonacoViewerHandle | null };
type WorkspaceOpenFile = {
  content: string;
  readable: boolean;
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

const EDITOR_WIDTH_STORAGE_KEY = 'graft.project.configuration-workspace.sidebar.width';
const EDITOR_HEIGHT_STORAGE_KEY = 'graft.project.configuration-workspace.editor.height.v2';
const SIDEBAR_MAX_WIDTH = 360;
const SIDEBAR_MIN_WIDTH = 208;
const SIDEBAR_DEFAULT_WIDTH = 256;
const SIDEBAR_COLLAPSE_BREAKPOINT = 1024;
const MONACO_MARKER_ERROR_SEVERITY = 8;

const logger = createLogger('project.configuration-workspace');
const route = useRoute();
const { t, tabsRouterStore } = useProjectPageContext();
const workspaceStore = useProjectWorkspaceStore(store);

const workspaceRootRef = ref<HTMLElement | null>(null);
const workspaceEditorRef = ref<WorkspaceEditorHandle | null>(null);
const activeEditorRef = ref<MonacoViewerHandle | null>(null);
const workspaceLoading = ref(false);
const workspaceError = ref('');
const workspaceReady = computed(() => Boolean(detailRecord.value && metadata.value && !workspaceError.value));
const browserLoading = ref(false);
const browserError = ref('');
const showHiddenFiles = ref(false);
const currentWorkspacePath = ref('');
const selectedWorkspacePath = ref('');
const sidebarWidth = ref(resolveStoredSidebarWidth());
const viewportWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1440);
const editorViewportHeight = ref(720);
const detailRecord = ref<ProjectDetailResponseWithLifecycle | null>(null);
const metadata = ref<Awaited<ReturnType<typeof getProjectConfiguration>> | null>(null);
const workspaceFullscreen = ref(false);
const resultDialogVisible = ref(false);
const resultDialogMode = ref<ResultDialogMode>('diff');
const resultDialogFullscreen = ref(false);
const diffViewerReady = ref(false);
const diffViewerRef = ref<MonacoViewerHandle | null>(null);
const syntaxViewerRef = ref<MonacoViewerHandle | null>(null);
const activeFileTabPathForMenu = ref<string | null>(null);
const diffResult = ref<WorkspacePreviewDiffResult | null>(null);
const syntaxValidationResult = ref<WorkspaceSyntaxValidationResult | null>(null);
const syntaxValidationFiles = ref<WorkspaceSyntaxValidationResult[]>([]);
const syntaxValidationSkippedPaths = ref<string[]>([]);
const syntaxCheckLoading = ref(false);
const deployLoading = ref(false);
const saveConfirmLoading = ref(false);
const selectedDiffFilePath = ref('');
const selectedSyntaxFilePath = ref('');
const resultIssueIndex = ref(0);
const diffLineChanges = ref<WorkspaceDiffLineChange[]>([]);
const syntaxMarkers = ref<WorkspaceSyntaxMarker[]>([]);
const pendingWorkspaceAction = ref<PendingWorkspaceAction | null>(null);
const pendingWorkspaceActionPaths = ref<string[]>([]);
const openTabs = ref<string[]>([]);
const activeTabPath = ref('');
const openFileMap = reactive(new Map<string, WorkspaceOpenFile>());
const directoryBrowseStateMap = reactive(new Map<string, { hasMoreHidden: boolean; parentPath: string | null }>());
const directoryErrorMap = reactive(new Map<string, string>());
const directoryLoadingMap = reactive(new Map<string, boolean>());
const expandedDirectoryPaths = ref<string[]>([]);
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
const annotationDialogState = reactive<{
  saving: boolean;
  target: WorkspaceListItem | null;
  value: string;
  visible: boolean;
}>({
  saving: false,
  target: null,
  value: '',
  visible: false,
});
const workspaceEntryMenu = reactive<{ item: WorkspaceListItem | null; visible: boolean; x: number; y: number }>({
  item: null,
  visible: false,
  x: 0,
  y: 0,
});
const workspaceEntryDialog = reactive<{
  error: string;
  mode: 'create-file' | 'create-directory' | 'rename';
  path: string;
  visible: boolean;
}>({ error: '', mode: 'create-file', path: '', visible: false });
const workspaceDeleteDialog = reactive<{
  path: string;
  stage: 'initial' | 'recursive';
  visible: boolean;
}>({ path: '', stage: 'initial', visible: false });
const readonlyOptions = {
  fontSize: 13,
  lineNumbers: 'on' as const,
  minimap: { enabled: false },
  readOnly: true,
  renderValidationDecorations: 'on' as const,
  wordWrap: 'off' as const,
};

const workspaceCopy = computed(() => resolveConfigurationWorkspaceCopy((key) => String(t(key))));
const projectId = computed(() => (typeof route.params.id === 'string' ? route.params.id : ''));
const fallbackDisplayName = computed(() => {
  const queryName = typeof route.query.name === 'string' ? route.query.name.trim() : '';
  return queryName;
});
const pageHeaderTitle = computed(() => {
  const suffix = t('project.route.configurationWorkspace.title');
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
const openTabBuffers = computed(
  () => openTabs.value.map((path) => openFileMap.get(path)).filter(Boolean) as WorkspaceOpenFile[],
);
const activeBuffer = computed(() => (activeTabPath.value ? (openFileMap.get(activeTabPath.value) ?? null) : null));
const dirtyEditableBuffers = computed(() =>
  openTabBuffers.value.filter((tab) => tab.editable && hasWorkspaceUnsavedChanges(tab.content, tab.savedContent)),
);
const canSaveActiveBuffer = computed(() =>
  Boolean(
    activeBuffer.value?.editable &&
    !activeBuffer.value.saving &&
    activeBuffer.value.path &&
    isFileDirty(activeBuffer.value.path),
  ),
);
const canSaveAllBuffers = computed(() => Boolean(!saveConfirmLoading.value && dirtyEditableBuffers.value.length));
const isSidebarResizable = computed(() => viewportWidth.value > SIDEBAR_COLLAPSE_BREAKPOINT);
const editorFrameHeight = computed(() => Math.max(560, editorViewportHeight.value));
const hasDirtyFiles = computed(() => dirtyEditableBuffers.value.length > 0);
const selectedDiffFile = computed(
  () =>
    diffResult.value?.files.find((file) => file.path === selectedDiffFilePath.value) ??
    diffResult.value?.files[0] ??
    null,
);
const diffFiles = computed(() => diffResult.value?.files ?? []);
const diffTreeRows = computed(() => buildDiffTreeRows(diffFiles.value));
const syntaxValidationFile = computed(
  () =>
    syntaxValidationFiles.value.find((file) => file.path === selectedSyntaxFilePath.value) ??
    syntaxValidationFiles.value[0] ??
    null,
);
const syntaxTreeRows = computed(() => buildSyntaxTreeRows(syntaxValidationFiles.value));
const showDiffSidebar = computed(
  () => resultDialogMode.value === 'diff' && pendingWorkspaceAction.value !== 'save-current',
);
const showSyntaxSidebar = computed(
  () =>
    resultDialogMode.value === 'syntax' &&
    (pendingWorkspaceAction.value === 'save-all' || pendingWorkspaceAction.value === 'deploy') &&
    syntaxValidationFiles.value.length > 0,
);
const batchSyntaxValidationDescription = computed(() => {
  if (!showSyntaxSidebar.value) {
    return '';
  }
  if (!syntaxValidationSkippedPaths.value.length) {
    return workspaceCopy.value.batchFileValidationRiskBody;
  }
  return `${workspaceCopy.value.batchFileValidationRiskBody} ${workspaceCopy.value.validateSkipUnsupportedHint}`;
});
const currentResultIssues = computed(() =>
  resultDialogMode.value === 'syntax' ? (syntaxValidationFile.value?.markers ?? []) : syntaxMarkers.value,
);
const canNavigateResultIssues = computed(() =>
  resultDialogMode.value === 'diff' ? Boolean(selectedDiffFile.value) : currentResultIssues.value.length > 0,
);

function buildConfigurationWorkspaceTitle(name: string) {
  return buildDetailTitleWithFallback('project.route.configurationWorkspace.title', name);
}

function updateCurrentTabTitle(name: string) {
  tabsRouterStore.updateActiveTabTitle(
    PROJECT_BOOTSTRAP_ROUTE.CONFIGURATION_WORKSPACE.pageRouteName,
    route,
    buildConfigurationWorkspaceTitle(name),
  );
}

const resultDialogTitle = computed(() =>
  resultDialogMode.value === 'diff'
    ? pendingWorkspaceAction.value === 'save-current'
      ? workspaceCopy.value.diffCurrentFileTitle
      : t('project.detail.configuration.diffTitle')
    : showSyntaxSidebar.value
      ? workspaceCopy.value.batchFileValidationTitle
      : workspaceCopy.value.fileValidationTitle,
);
const resultDialogDescription = computed(() => {
  if (resultDialogMode.value === 'diff') {
    return pendingWorkspaceAction.value === 'save-current'
      ? workspaceCopy.value.diffCurrentFileConfirmBody
      : workspaceCopy.value.diffConfirmBody;
  }
  if (showSyntaxSidebar.value) {
    return batchSyntaxValidationDescription.value;
  }
  return workspaceCopy.value.fileValidationFailed;
});
const resultDialogSummaryItems = computed(() => {
  if (resultDialogMode.value === 'diff') {
    const items = [
      {
        label: workspaceCopy.value.resultSummaryChangedFilesLabel,
        value: String(diffFiles.value.length),
      },
    ];
    if (selectedDiffFile.value?.path) {
      items.push({
        label: workspaceCopy.value.resultSummaryCurrentFileLabel,
        value: selectedDiffFile.value.path,
      });
    }
    return items;
  }

  const items = [
    {
      label: workspaceCopy.value.resultSummaryErrorFilesLabel,
      value: String(syntaxValidationFiles.value.length || (syntaxValidationResult.value ? 1 : 0)),
    },
  ];
  if (syntaxValidationFile.value) {
    items.push({
      label: workspaceCopy.value.resultSummaryCurrentErrorsLabel,
      value: t(resolveConfigurationWorkspaceCopyKey('syntaxErrorCountLabel'), {
        count: syntaxValidationFile.value.markerCount,
      }),
    });
  }
  return items;
});
const syntaxConfirmActionLabel = computed(() => {
  if (pendingWorkspaceAction.value === 'deploy') {
    return workspaceCopy.value.confirmSaveDeployWithErrorsAction;
  }
  if (pendingWorkspaceAction.value === 'save-all') {
    return workspaceCopy.value.confirmSaveAllWithErrorsAction;
  }
  return workspaceCopy.value.confirmSaveWithErrorsAction;
});
const diffConfirmActionLabel = computed(() => {
  if (pendingWorkspaceAction.value === 'deploy') {
    return workspaceCopy.value.confirmSaveAllAction;
  }
  if (pendingWorkspaceAction.value === 'save-all') {
    return workspaceCopy.value.confirmSaveAllAction;
  }
  return workspaceCopy.value.confirmSaveCurrentAction;
});
const resultDialogClassName = computed(() =>
  resultDialogFullscreen.value
    ? 'project-configuration-workspace__result-dialog-shell project-configuration-workspace__result-dialog-shell--fullscreen'
    : 'project-configuration-workspace__result-dialog-shell',
);
const resultDialogWidth = computed(() => (resultDialogFullscreen.value ? undefined : 'min(80vw, 1600px)'));
const resultDialogTop = computed(() => (resultDialogFullscreen.value ? 0 : 24));
const resultDialogStyle = computed(() =>
  resultDialogFullscreen.value
    ? {
        height: '100vh',
        maxHeight: '100vh',
        width: '100vw',
      }
    : {
        height: '80vh',
        maxHeight: 'calc(100vh - 48px)',
        width: 'min(80vw, 1600px)',
      },
);
const workspaceItemMap = computed(() => {
  return new Map(Object.entries(workspaceStore.activeSession.nodesByKey)) as Map<string, WorkspaceListItem>;
});
const currentWorkspaceDirectoryPath = computed(() => {
  if (selectedWorkspacePath.value) {
    const selectedItem = workspaceItemMap.value.get(selectedWorkspacePath.value);
    if (selectedItem?.node_type === 'directory') {
      return selectedItem.relative_path;
    }
  }

  if (activeBuffer.value?.path) {
    return resolveWorkspaceParentPath(activeBuffer.value.path);
  }

  return currentWorkspacePath.value;
});
const currentWorkspacePathLabel = computed(
  () => currentWorkspaceDirectoryPath.value || workspaceCopy.value.workspaceRootLabel,
);
const workingDirectoryDisplay = computed(() => abbreviateWorkspacePath(detailRecord.value?.workspace_path));
const currentWorkspacePathDisplay = computed(() => abbreviateWorkspacePath(currentWorkspacePathLabel.value));
const workspaceEditorRows = computed<ProjectWorkspaceEditorRow[]>(() =>
  workspaceStore.visibleTreeRows.map((row) => ({
    depth: row.depth,
    expanded: row.expanded,
    error: directoryErrorMap.get(row.item.relative_path) ?? '',
    fileKind: row.item.file_kind,
    name: row.item.name,
    nodeType: row.item.node_type,
    path: row.item.relative_path,
    readOnly: row.item.node_type === 'file' && (!row.item.readable || !row.item.editable),
    testId: workspaceEntryTestId(row.item),
    tooltip: workspaceItemTooltip(row.item),
  })),
);
const workspaceEditorTabs = computed<ProjectWorkspaceEditorBuffer[]>(() =>
  workspaceStore.openedFiles.map((tab) => ({
    content: tab.content,
    dirty: isFileDirty(tab.path),
    error: tab.error,
    language: tab.language,
    loading: tab.loading,
    modelKey: tab.path,
    name: tab.name,
    path: tab.path,
    readOnly: !tab.editable,
  })),
);
const workspaceEditorActiveBuffer = computed<ProjectWorkspaceEditorBuffer | null>(() => {
  const tab = workspaceStore.activeFile;
  if (!tab) return null;
  return {
    content: tab.content,
    dirty: isFileDirty(tab.path),
    error: tab.error,
    language: tab.language,
    loading: tab.loading,
    modelKey: tab.path,
    name: tab.name,
    path: tab.path,
    readOnly: !tab.editable,
  };
});
const workspaceEditorLabels = computed<ProjectWorkspaceEditorLabels>(() => ({
  annotationAction: workspaceCopy.value.annotationAction,
  closeAll: t('layout.tagTabs.closeAll'),
  closeLeft: t('layout.tagTabs.closeLeft'),
  closeOther: t('layout.tagTabs.closeOther'),
  closeRight: t('layout.tagTabs.closeRight'),
  delete: t('project.create.workspace.delete'),
  entryActions: t('project.create.workspace.entryActions', { path: '{path}' }),
  exitFullscreen: workspaceCopy.value.exitFullscreenAction,
  fullscreen: workspaceCopy.value.fullscreenAction,
  newFile: t('project.create.workspace.newFile'),
  newFolder: t('project.create.workspace.newFolder'),
  refresh: t('layout.tagTabs.refresh'),
  rename: t('project.create.workspace.rename'),
}));

function abbreviateWorkspacePath(value?: string | null, maxLength = 16) {
  const normalized = value?.trim() || '-';
  if (normalized.length <= maxLength) {
    return normalized;
  }

  const edgeLength = Math.max(4, Math.floor((maxLength - 3) / 2));
  return `${normalized.slice(0, edgeLength)}...${normalized.slice(-edgeLength)}`;
}

onMounted(() => {
  window.addEventListener('keydown', handleWorkspaceKeydown);
  window.addEventListener('resize', syncWorkspaceViewport);
  void loadWorkspace();
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleWorkspaceKeydown);
  window.removeEventListener('resize', syncWorkspaceViewport);
  if (typeof document !== 'undefined') {
    document.body.style.overflow = '';
    document.documentElement.style.overflow = '';
  }
});

watch(showHiddenFiles, () => {
  currentWorkspacePath.value = '';
  selectedWorkspacePath.value = '';
  expandedDirectoryPaths.value = [];
  directoryBrowseStateMap.clear();
  directoryErrorMap.clear();
  directoryLoadingMap.clear();
  void loadWorkspaceDirectory('', { root: true });
});

watch(
  fallbackDisplayName,
  (name) => {
    if (!name) {
      return;
    }
    updateCurrentTabTitle(name);
  },
  { immediate: true },
);

watch(openFileMap, () => workspaceStore.syncOpenedFiles(openTabBuffers.value, activeTabPath.value), { deep: true });

watch(activeTabPath, (path) => workspaceStore.syncOpenedFiles(openTabBuffers.value, path));

watch(resultDialogVisible, (visible) => {
  if (visible) {
    if (resultDialogMode.value === 'diff') {
      diffViewerReady.value = false;
    }
    queueResultViewerLayout();
  } else {
    diffViewerReady.value = false;
    resultDialogFullscreen.value = false;
    resultIssueIndex.value = 0;
    diffLineChanges.value = [];
    syntaxMarkers.value = [];
    syntaxValidationResult.value = null;
    syntaxValidationFiles.value = [];
    syntaxValidationSkippedPaths.value = [];
    selectedSyntaxFilePath.value = '';
    pendingWorkspaceAction.value = null;
    pendingWorkspaceActionPaths.value = [];
    saveConfirmLoading.value = false;
  }
});

watch(workspaceFullscreen, (fullscreen) => {
  if (typeof document === 'undefined') {
    return;
  }

  document.body.style.overflow = fullscreen ? 'hidden' : '';
  document.documentElement.style.overflow = fullscreen ? 'hidden' : '';
  void nextTick(() => {
    syncWorkspaceViewport();
  });
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

watch(selectedDiffFilePath, () => {
  if (resultDialogVisible.value && resultDialogMode.value === 'diff') {
    void refreshDiffLineChanges();
  }
});

watch(selectedSyntaxFilePath, () => {
  if (!resultDialogVisible.value || resultDialogMode.value !== 'syntax') {
    return;
  }

  syntaxMarkers.value = syntaxValidationFile.value?.markers ?? [];
  resultIssueIndex.value = 0;
  void queueResultViewerLayout().then(() => {
    revealCurrentResultIssue();
  });
});

watch(activeTabPath, () => {
  if (resultDialogMode.value === 'syntax' && resultDialogVisible.value) {
    resultDialogVisible.value = false;
  }
});

async function loadWorkspace() {
  if (!projectId.value) {
    workspaceError.value = t('project.list.retry');
    return;
  }

  workspaceLoading.value = true;
  workspaceError.value = '';
  workspaceStore.activateSession(`project:${projectId.value}`);
  try {
    const [detail, configurationMetadata] = await Promise.all([
      getProject(projectId.value),
      getProjectConfiguration(projectId.value),
    ]);
    detailRecord.value = detail;
    updateCurrentTabTitle(detail.display_name);
    metadata.value = configurationMetadata;
    await loadWorkspaceDirectory('', { root: true });
    await nextTick();
    syncWorkspaceViewport();
  } catch (error) {
    logger.error('failed to load project configuration workspace', error);
    workspaceError.value = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
    MessagePlugin.error(workspaceError.value);
  } finally {
    workspaceLoading.value = false;
  }
}

async function loadWorkspaceDirectory(path: string, options?: { root?: boolean }) {
  const normalizedPath = normalizeWorkspacePath(path);
  if (options?.root) {
    browserLoading.value = true;
    browserError.value = '';
  } else {
    directoryLoadingMap.set(normalizedPath, true);
    directoryErrorMap.delete(normalizedPath);
  }

  try {
    const response = await getProjectFiles(projectId.value, {
      path: normalizedPath || undefined,
      show_hidden: showHiddenFiles.value,
    });

    const items = sortWorkspaceItems(response.items ?? []);
    directoryBrowseStateMap.set(normalizedPath, {
      hasMoreHidden: Boolean(response.has_more_hidden),
      parentPath: response.parent_path ?? null,
    });

    if (options?.root) {
      currentWorkspacePath.value = response.current_path || '';
      workspaceStore.replaceTree(items);
    } else {
      currentWorkspacePath.value = normalizedPath;
      workspaceStore.ingestTree(items, normalizedPath);
    }

    if (!activeTabPath.value && options?.root) {
      const firstFile = items.find((item) => item.node_type === 'file')?.relative_path;
      if (firstFile) {
        await openWorkspaceFile(firstFile);
      }
    }
  } catch (error) {
    const resolvedMessage = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
    if (options?.root) {
      browserError.value = resolvedMessage;
      MessagePlugin.error(browserError.value);
    } else {
      directoryErrorMap.set(normalizedPath, resolvedMessage);
      MessagePlugin.error(resolvedMessage);
    }
  } finally {
    if (options?.root) {
      browserLoading.value = false;
    } else {
      directoryLoadingMap.set(normalizedPath, false);
    }
  }
}

function sortWorkspaceItems(items: ProjectWorkspaceTreeItem[]) {
  return [...items].sort((left, right) => {
    if (left.node_type !== right.node_type) {
      return left.node_type === 'directory' ? -1 : 1;
    }
    return left.name.localeCompare(right.name, undefined, { sensitivity: 'base' });
  });
}

function handleWorkspaceEntry(item: WorkspaceListItem) {
  selectedWorkspacePath.value = item.relative_path;
  workspaceStore.selectNode(item.relative_path);
  if (item.node_type === 'directory') {
    void toggleWorkspaceDirectory(item);
    return;
  }
  if (!item.readable) {
    return;
  }
  void openWorkspaceFile(item.relative_path, item);
}

function workspaceItemForEditorRow(row: ProjectWorkspaceEditorRow) {
  return workspaceItemMap.value.get(row.path) ?? null;
}

function handleWorkspaceEditorEntry(row: ProjectWorkspaceEditorRow) {
  const item = workspaceItemForEditorRow(row);
  if (item) handleWorkspaceEntry(item);
}

function toggleWorkspaceEditorDirectory(row: ProjectWorkspaceEditorRow) {
  const item = workspaceItemForEditorRow(row);
  if (item) void toggleWorkspaceDirectory(item);
}

function updateWorkspaceEditorContent(path: string, content: string) {
  const buffer = openFileMap.get(path);
  if (buffer) {
    buffer.content = content;
    workspaceStore.setFileContent(path, content);
  }
}

function activeWorkspaceEditor() {
  return activeEditorRef.value ?? workspaceEditorRef.value?.getActiveEditor() ?? null;
}

function setActiveWorkspaceEditor(editor: unknown) {
  activeEditorRef.value = editor as MonacoViewerHandle;
}

function handleWorkspaceEditorContextAction(
  action: 'create-file' | 'create-directory' | 'annotation' | 'rename' | 'delete',
  row: ProjectWorkspaceEditorRow | null,
) {
  workspaceEntryMenu.item = row ? workspaceItemForEditorRow(row) : null;
  if (action === 'create-file') openWorkspaceEntryDialog('create-file');
  if (action === 'create-directory') openWorkspaceEntryDialog('create-directory');
  if (action === 'annotation' && workspaceEntryMenu.item) handleWorkspaceAnnotation(workspaceEntryMenu.item);
  if (action === 'rename') openWorkspaceEntryDialog('rename');
  if (action === 'delete') openWorkspaceDeleteDialog();
}

function handleWorkspaceEditorTabAction(
  action: 'refresh' | 'close-left' | 'close-right' | 'close-other' | 'close-all',
  path: string,
) {
  if (action === 'refresh') void handleRefreshFileTab(path);
  if (action === 'close-left') void handleCloseFileTabsAhead(path);
  if (action === 'close-right') void handleCloseFileTabsBehind(path);
  if (action === 'close-other') void handleCloseOtherFileTabs(path);
  if (action === 'close-all') void handleCloseAllFileTabs();
}

function normalizeWorkspaceEntryPath(path: string) {
  return String(path || '')
    .trim()
    .replace(/^\.\//, '')
    .replace(/\/+$/, '');
}

function isSafeWorkspaceEntryPath(path: string) {
  return (
    Boolean(path) && !path.startsWith('/') && !path.split('/').some((part) => !part || part === '.' || part === '..')
  );
}

function closeWorkspaceEntryMenu() {
  workspaceEntryMenu.visible = false;
}

function openWorkspaceEntryDialog(mode: 'create-file' | 'create-directory' | 'rename') {
  workspaceEntryDialog.mode = mode;
  workspaceEntryDialog.error = '';
  const target = workspaceEntryMenu.item;
  if (mode === 'rename') {
    workspaceEntryDialog.path = target?.relative_path || '';
  } else {
    const parent =
      target?.node_type === 'directory'
        ? target.relative_path
        : resolveWorkspaceParentPath(target?.relative_path || '');
    workspaceEntryDialog.path = parent ? `${parent}/` : '';
  }
  workspaceEntryDialog.visible = true;
  closeWorkspaceEntryMenu();
}

async function submitWorkspaceEntryDialog() {
  const path = normalizeWorkspaceEntryPath(workspaceEntryDialog.path);
  const target = workspaceEntryMenu.item;
  if (!isSafeWorkspaceEntryPath(path)) {
    workspaceEntryDialog.error = t('project.create.workspace.invalidFilePath');
    return;
  }
  try {
    if (workspaceEntryDialog.mode === 'rename') {
      if (!target) return;
      await postProjectWorkspaceRename(projectId.value, { path: target.relative_path, new_path: path });
      migrateWorkspaceBuffers(target.relative_path, path);
    } else {
      await postProjectWorkspaceEntry(projectId.value, {
        path,
        node_type: workspaceEntryDialog.mode === 'create-directory' ? 'directory' : 'file',
        ...(workspaceEntryDialog.mode === 'create-file' ? { content: '' } : {}),
      });
    }
    workspaceEntryDialog.visible = false;
    await refreshWorkspaceAfterEntryMutation();
    if (workspaceEntryDialog.mode === 'create-file') await openWorkspaceFile(path);
  } catch (error) {
    workspaceEntryDialog.error = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
  }
}

function openWorkspaceDeleteDialog() {
  const target = workspaceEntryMenu.item;
  if (!target) return;
  workspaceDeleteDialog.path = target.relative_path;
  workspaceDeleteDialog.stage = 'initial';
  workspaceDeleteDialog.visible = true;
  closeWorkspaceEntryMenu();
}

async function confirmWorkspaceEntryDelete() {
  const target = workspaceItemMap.value.get(workspaceDeleteDialog.path);
  const needsRecursiveConfirm = target?.node_type === 'directory' && target.has_children;
  if (needsRecursiveConfirm && workspaceDeleteDialog.stage === 'initial') {
    workspaceDeleteDialog.stage = 'recursive';
    return;
  }
  try {
    const affectedPaths = openTabs.value.filter(
      (path) => path === workspaceDeleteDialog.path || path.startsWith(`${workspaceDeleteDialog.path}/`),
    );
    if (!(await confirmWorkspaceBufferDeletion(affectedPaths))) return;
    await deleteProjectWorkspaceEntry(projectId.value, {
      path: workspaceDeleteDialog.path,
      recursive: workspaceDeleteDialog.stage === 'recursive',
    });
    workspaceDeleteDialog.visible = false;
    removeWorkspaceBuffers(affectedPaths);
    await refreshWorkspaceAfterEntryMutation();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.retry')));
  }
}

async function refreshWorkspaceAfterEntryMutation() {
  directoryBrowseStateMap.clear();
  directoryErrorMap.clear();
  directoryLoadingMap.clear();
  expandedDirectoryPaths.value = [];
  await loadWorkspaceDirectory('', { root: true });
}

function migrateWorkspaceBuffers(oldPath: string, newPath: string) {
  const remappedPaths = new Map<string, string>();
  for (const path of openTabs.value) {
    if (path === oldPath || path.startsWith(`${oldPath}/`)) {
      remappedPaths.set(path, `${newPath}${path.slice(oldPath.length)}`);
    }
  }
  for (const [path, nextPath] of remappedPaths) {
    const buffer = openFileMap.get(path);
    if (!buffer) continue;
    openFileMap.delete(path);
    buffer.path = nextPath;
    buffer.name = resolveWorkspaceFileName(nextPath);
    openFileMap.set(nextPath, buffer);
  }
  if (!remappedPaths.size) return;
  openTabs.value = openTabs.value.map((path) => remappedPaths.get(path) ?? path);
  activeTabPath.value = remappedPaths.get(activeTabPath.value) ?? activeTabPath.value;
  activeFileTabPathForMenu.value = activeFileTabPathForMenu.value
    ? (remappedPaths.get(activeFileTabPathForMenu.value) ?? activeFileTabPathForMenu.value)
    : null;
  latestOpenRequestPath = remappedPaths.get(latestOpenRequestPath) ?? latestOpenRequestPath;
}

function removeWorkspaceBuffers(paths: string[]) {
  const removedPathSet = new Set(paths);
  if (!removedPathSet.size) return;
  removedPathSet.forEach((path) => openFileMap.delete(path));
  openTabs.value = openTabs.value.filter((path) => !removedPathSet.has(path));
  if (removedPathSet.has(activeTabPath.value)) activeTabPath.value = openTabs.value.at(-1) ?? '';
  if (activeFileTabPathForMenu.value && removedPathSet.has(activeFileTabPathForMenu.value)) {
    activeFileTabPathForMenu.value = null;
  }
}

async function confirmWorkspaceBufferDeletion(paths: string[]) {
  const dirtyPaths = paths.filter((path) => isFileDirty(path));
  if (!dirtyPaths.length) return true;
  const action = await openDialog({
    body: workspaceCopy.value.dirtyProjectActionBody,
    buttons: [
      { label: workspaceCopy.value.saveThenContinueAction, result: 'save', theme: 'primary', variant: 'base' },
      { label: workspaceCopy.value.discardAction, result: 'discard', theme: 'default', variant: 'outline' },
      { label: workspaceCopy.value.cancelAction, result: 'cancel', theme: 'default', variant: 'outline' },
    ],
    title: workspaceCopy.value.dirtyProjectActionTitle,
  });
  if (action === 'cancel') return false;
  return action !== 'save' || saveTabsByPaths(dirtyPaths);
}

function workspaceEntryTestId(item: WorkspaceListItem) {
  const raw = item.relative_path || item.name || 'root';
  return `workspace-entry-${
    raw
      .replace(/[^a-z0-9]+/gi, '-')
      .replace(/^-+|-+$/g, '')
      .toLowerCase() || 'root'
  }`;
}

function workspaceFileTabMenuTestId(path: string) {
  return `workspace-file-tab-menu-${workspaceEntryTestId({ relative_path: path, name: path } as WorkspaceListItem)}`;
}

function workspaceFileTabMenuItemTestId(path: string, action: string) {
  return `${workspaceFileTabMenuTestId(path)}-${action}`;
}

function diffFileTestId(path: string) {
  return `configuration-diff-file-${workspaceEntryTestId({ relative_path: path, name: path } as WorkspaceListItem)}`;
}

async function toggleWorkspaceDirectory(item: WorkspaceListItem) {
  if (item.node_type !== 'directory') {
    return;
  }

  const path = item.relative_path;
  currentWorkspacePath.value = path;
  selectedWorkspacePath.value = path;
  if (expandedDirectoryPaths.value.includes(path)) {
    expandedDirectoryPaths.value = expandedDirectoryPaths.value.filter((value) => value !== path);
    workspaceStore.setExpanded(path, false);
    return;
  }

  expandedDirectoryPaths.value = [...expandedDirectoryPaths.value, path];
  workspaceStore.setExpanded(path, true);
  if (!workspaceStore.activeSession.nodesByKey[path]?.childrenLoaded && item.has_children) {
    await loadWorkspaceDirectory(path);
  }
}

function workspaceItemTooltip(item?: WorkspaceListItem | null) {
  if (!item) {
    return '';
  }
  const projectNote = String(item.project_note ?? '').trim();
  if (projectNote) {
    return projectNote;
  }

  const tooltip = String(item.tooltip ?? '').trim();
  return tooltip || '';
}

function handleWorkspaceAnnotation(item: WorkspaceListItem) {
  annotationDialogState.target = item;
  annotationDialogState.value = String(item.project_note ?? '').trim();
  annotationDialogState.visible = true;
}

function closeWorkspaceAnnotationDialog() {
  annotationDialogState.visible = false;
  annotationDialogState.saving = false;
  annotationDialogState.target = null;
  annotationDialogState.value = '';
}

async function saveWorkspaceAnnotation() {
  const target = annotationDialogState.target;
  if (!target) {
    closeWorkspaceAnnotationDialog();
    return;
  }

  annotationDialogState.saving = true;
  try {
    const annotation = annotationDialogState.value.trim();
    const updatedItem = await putProjectFileAnnotation(
      projectId.value,
      { path: target.relative_path },
      { annotation: annotation || null },
    );
    patchWorkspaceItem(updatedItem);
    closeWorkspaceAnnotationDialog();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, workspaceCopy.value.annotationSaveFailed));
  } finally {
    annotationDialogState.saving = false;
  }
}

function patchWorkspaceItem(nextItem: WorkspaceListItem) {
  if (!nextItem?.relative_path) {
    return;
  }

  workspaceStore.patchNode(nextItem);
}

let latestOpenRequestPath = '';

async function openWorkspaceFile(path: string, source?: WorkspaceListItem) {
  if (!path) {
    return;
  }
  latestOpenRequestPath = path;

  if (!openFileMap.has(path)) {
    openFileMap.set(path, {
      content: '',
      readable: Boolean(source?.readable ?? true),
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

  const current = openFileMap.get(path);
  const shouldDelayActivation = Boolean(current && !current.loaded && !current.loading);

  if (!shouldDelayActivation) {
    activeTabPath.value = path;
  }

  const resolvedPath = await ensureWorkspaceFileLoaded(path, source);
  if (latestOpenRequestPath === path) {
    activeTabPath.value = resolvedPath ?? path;
  }
}

async function ensureWorkspaceFileLoaded(path: string, source?: WorkspaceListItem) {
  const current = openFileMap.get(path);
  if (!current || current.loading || current.loaded) {
    return current?.path ?? path;
  }

  current.loading = true;
  current.error = '';
  try {
    const response = await getProjectFileContent(projectId.value, { path });
    return hydrateOpenFileFromResponse(path, response, source);
  } catch (error) {
    current.error = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
  } finally {
    current.loading = false;
  }

  return current.path ?? path;
}

function hydrateOpenFileFromResponse(
  requestedPath: string,
  response: ProjectWorkspaceFileContentResponse,
  source?: WorkspaceListItem,
) {
  const path = response.relative_path || requestedPath;
  const current =
    openFileMap.get(requestedPath) ??
    ({
      content: '',
      readable: response.readable,
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
  current.readable = response.readable;
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

  return path;
}

function isFileDirty(path: string): boolean {
  const current = openFileMap.get(path);
  return current ? hasWorkspaceUnsavedChanges(current.content, current.savedContent) : false;
}

async function saveActiveFile() {
  if (!canSaveActiveBuffer.value) {
    return;
  }
  await previewBeforeSave('save-current');
}

async function saveAllFiles() {
  if (!canSaveAllBuffers.value) {
    return;
  }
  await previewBeforeSave('save-all');
}

async function saveWorkspaceFile(path: string, options?: { silent?: boolean }) {
  const current = openFileMap.get(path);
  if (!current || !current.editable || current.saving) {
    return false;
  }

  current.saving = true;
  try {
    const normalizedContent = normalizeTextBlock(current.content);
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

function resolvePendingWorkspacePaths(action: PendingWorkspaceAction) {
  if (action === 'save-current') {
    return activeBuffer.value?.path && isFileDirty(activeBuffer.value.path) ? [activeBuffer.value.path] : [];
  }

  return dirtyEditableBuffers.value.map((tab) => tab.path);
}

function buildSyntaxValidationResult(file: WorkspaceOpenFile, markers: WorkspaceSyntaxMarker[]) {
  return {
    content: file.content,
    fileName: file.name,
    language: file.language,
    markerCount: markers.length,
    markers,
    modelKey: `syntax-${file.path}`,
    path: file.path,
  } satisfies WorkspaceSyntaxValidationResult;
}

function normalizeSyntaxErrors(markers: WorkspaceSyntaxMarker[] | undefined) {
  return (markers ?? [])
    .filter((marker) => marker.severity === MONACO_MARKER_ERROR_SEVERITY)
    .sort((left, right) => {
      if (left.startLineNumber !== right.startLineNumber) {
        return left.startLineNumber - right.startLineNumber;
      }
      return left.startColumn - right.startColumn;
    });
}

async function collectActiveEditorSyntaxErrors(options?: { retries?: number }) {
  const maxRetries = Math.max(0, options?.retries ?? 0);

  for (let attempt = 0; attempt <= maxRetries; attempt += 1) {
    const markers = await activeWorkspaceEditor()?.waitForDiagnostics?.({
      quietMs: 180,
      timeoutMs: 1500,
    });
    const errors = normalizeSyntaxErrors(markers);
    if (errors.length > 0 || attempt === maxRetries) {
      return errors;
    }
  }

  return [] as WorkspaceSyntaxMarker[];
}

async function waitForActiveEditorModel(path: string, options?: { maxAttempts?: number }) {
  const maxAttempts = Math.max(1, options?.maxAttempts ?? 6);

  for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
    if (activeWorkspaceEditor()?.getModelKey?.() === path) {
      return true;
    }

    await nextTick();
  }

  return activeWorkspaceEditor()?.getModelKey?.() === path;
}

function failClosedToBoundActiveEditorModel() {
  const boundModelKey = activeWorkspaceEditor()?.getModelKey?.();
  if (boundModelKey) {
    activeTabPath.value = boundModelKey;
  }
}

function resolveSyntaxValidationTargets(paths: string[]) {
  const supportedPaths: string[] = [];
  const skippedPaths: string[] = [];

  for (const path of [...new Set(paths)]) {
    const file = openFileMap.get(path);
    if (!file?.editable) {
      continue;
    }
    if (supportsExplicitSyntaxValidation(file.language)) {
      supportedPaths.push(path);
      continue;
    }
    skippedPaths.push(path);
  }

  return { skippedPaths, supportedPaths };
}

function notifySkippedSyntaxValidationPaths(paths: string[]) {
  if (!paths.length) {
    return;
  }
  const fileList = paths.join(', ');
  MessagePlugin.warning(`${workspaceCopy.value.validateSkipUnsupportedHint} ${fileList}`);
}

async function collectSyntaxValidationIssueForPath(path: string, options?: { retries?: number }) {
  if (activeTabPath.value !== path) {
    activeTabPath.value = path;
    await nextTick();
  }
  const activeModelReady = await waitForActiveEditorModel(path);
  if (!activeModelReady) {
    return null;
  }

  const file = openFileMap.get(path);
  if (!file) {
    return null;
  }

  const errors = await collectActiveEditorSyntaxErrors({
    retries: options?.retries ?? 0,
  });
  if (!errors.length) {
    return null;
  }

  return buildSyntaxValidationResult(file, errors);
}

async function collectSyntaxValidationIssues(paths: string[]) {
  const { supportedPaths } = resolveSyntaxValidationTargets(paths);
  const activePathBeforeValidation = activeTabPath.value;
  const uniquePaths =
    activePathBeforeValidation && supportedPaths.includes(activePathBeforeValidation)
      ? [activePathBeforeValidation, ...supportedPaths.filter((path) => path !== activePathBeforeValidation)]
      : supportedPaths;
  const issues = new Map<string, WorkspaceSyntaxValidationResult>();
  const unresolvedPaths: string[] = [];

  for (const path of uniquePaths) {
    const issue = await collectSyntaxValidationIssueForPath(path);
    if (issue) {
      issues.set(path, issue);
      continue;
    }
    unresolvedPaths.push(path);
  }

  for (const path of unresolvedPaths) {
    if (issues.has(path)) {
      continue;
    }
    const issue = await collectSyntaxValidationIssueForPath(path, { retries: 1 });
    if (issue) {
      issues.set(path, issue);
    }
  }

  if (activePathBeforeValidation && activeTabPath.value !== activePathBeforeValidation) {
    activeTabPath.value = activePathBeforeValidation;
    await nextTick();
    const restored = await waitForActiveEditorModel(activePathBeforeValidation, { maxAttempts: 12 });
    if (!restored) {
      failClosedToBoundActiveEditorModel();
    }
  }

  return uniquePaths.map((path) => issues.get(path)).filter(Boolean) as WorkspaceSyntaxValidationResult[];
}

async function saveDirtyFiles(paths?: string[]) {
  const dirtyPaths = paths?.length
    ? [...new Set(paths)].filter((path) => isFileDirty(path))
    : dirtyEditableBuffers.value.map((tab) => tab.path);
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

async function finalizePendingWorkspaceAction(action: PendingWorkspaceAction, paths: string[]) {
  if (action === 'save-current') {
    const targetPath = paths[0];
    return targetPath ? saveWorkspaceFile(targetPath) : true;
  }

  const saved = await saveDirtyFiles(paths);
  if (!saved) {
    return false;
  }

  if (action === 'deploy') {
    await executeProjectDeploy();
  }

  return true;
}

async function confirmSkippedSyntaxValidationPaths(action: PendingWorkspaceAction, skippedPaths: string[]) {
  if (!skippedPaths.length) {
    return true;
  }

  const skippedFileList = skippedPaths.join(', ');
  const confirmLabel =
    action === 'save-current' ? workspaceCopy.value.confirmSaveCurrentAction : workspaceCopy.value.confirmSaveAllAction;
  const dialogAction = await openDialog({
    body: `${workspaceCopy.value.validateSkipUnsupportedHint} ${skippedFileList}`,
    buttons: [
      {
        label: workspaceCopy.value.cancelAction,
        result: 'cancel',
        theme: 'default',
        variant: 'outline',
      },
      {
        label: confirmLabel,
        result: 'save',
        theme: 'primary',
        variant: 'base',
      },
    ],
    title: workspaceCopy.value.batchFileValidationRiskTitle,
  });

  return dialogAction === 'save';
}

async function proceedAfterDiffConfirmation(action: PendingWorkspaceAction, paths: string[]) {
  const { skippedPaths } = resolveSyntaxValidationTargets(paths);
  syntaxValidationSkippedPaths.value = skippedPaths;
  notifySkippedSyntaxValidationPaths(syntaxValidationSkippedPaths.value);
  const syntaxIssues = await collectSyntaxValidationIssues(paths);
  if (!syntaxIssues.length) {
    const confirmedSkippedPaths = await confirmSkippedSyntaxValidationPaths(action, syntaxValidationSkippedPaths.value);
    if (!confirmedSkippedPaths) {
      return false;
    }
    const saved = await finalizePendingWorkspaceAction(action, paths);
    if (saved) {
      resultDialogVisible.value = false;
      diffResult.value = null;
      selectedDiffFilePath.value = '';
    }
    return saved;
  }

  diffResult.value = null;
  selectedDiffFilePath.value = '';
  syntaxValidationFiles.value = syntaxIssues;
  syntaxValidationResult.value = syntaxIssues[0] ?? null;
  selectedSyntaxFilePath.value = syntaxIssues[0]?.path ?? '';
  syntaxMarkers.value = syntaxIssues[0]?.markers ?? [];
  resultIssueIndex.value = 0;
  resultDialogMode.value = 'syntax';
  resultDialogVisible.value = true;
  pendingWorkspaceAction.value = action;
  pendingWorkspaceActionPaths.value = paths;
  await queueResultViewerLayout();
  revealCurrentResultIssue();
  return false;
}

async function reloadActiveFile() {
  if (!activeBuffer.value) {
    return;
  }

  await reloadWorkspaceFile(activeBuffer.value.path);
}

async function reloadWorkspaceFile(path: string) {
  const buffer = openFileMap.get(path);
  if (!buffer) {
    return;
  }

  if (isFileDirty(path)) {
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
  const closed = await closeWorkspaceTabs([path], { skipBatchDirtyPrompt: true });
  if (closed) {
    activeFileTabPathForMenu.value = null;
  }
}

async function closeWorkspaceTabs(paths: string[], options?: { skipBatchDirtyPrompt?: boolean }) {
  const uniquePaths = [...new Set(paths)].filter((path) => openFileMap.has(path));
  if (!uniquePaths.length) {
    return false;
  }

  if (!options?.skipBatchDirtyPrompt) {
    const dirtyPaths = uniquePaths.filter((path) => isFileDirty(path));
    if (dirtyPaths.length) {
      const action = await openDialog({
        body: workspaceCopy.value.dirtyProjectActionBody,
        buttons: [
          { label: workspaceCopy.value.saveThenContinueAction, result: 'save', theme: 'primary', variant: 'base' },
          { label: workspaceCopy.value.discardAction, result: 'discard', theme: 'default', variant: 'outline' },
          { label: workspaceCopy.value.cancelAction, result: 'cancel', theme: 'default', variant: 'outline' },
        ],
        title: workspaceCopy.value.dirtyProjectActionTitle,
      });
      if (action === 'cancel') {
        return false;
      }
      if (action === 'save') {
        const saved = await saveTabsByPaths(dirtyPaths);
        if (!saved) {
          return false;
        }
      }
    }
  }

  if (uniquePaths.length === 1) {
    return closeSingleWorkspaceTab(uniquePaths[0]);
  }

  const closedPathSet = new Set(uniquePaths);
  const currentTabs = [...openTabs.value];
  const currentActivePath = activeTabPath.value;
  const activeIndex = currentTabs.findIndex((item) => item === currentActivePath);
  const nextTabs = currentTabs.filter((path) => !closedPathSet.has(path));

  uniquePaths.forEach((path) => {
    openFileMap.delete(path);
  });

  openTabs.value = nextTabs;
  if (!currentActivePath || !closedPathSet.has(currentActivePath)) {
    return true;
  }

  activeTabPath.value = resolveNextWorkspaceTabAfterClose(currentTabs, closedPathSet, activeIndex);
  return true;
}

function resolveNextWorkspaceTabAfterClose(currentTabs: string[], closedPathSet: Set<string>, activeIndex: number) {
  const nextRight = currentTabs.slice(activeIndex + 1).find((path) => !closedPathSet.has(path));
  if (nextRight) {
    return nextRight;
  }

  const nextLeft = [...currentTabs.slice(0, Math.max(activeIndex, 0))]
    .reverse()
    .find((path) => !closedPathSet.has(path));
  return nextLeft || '';
}

async function closeSingleWorkspaceTab(path: string) {
  const current = openFileMap.get(path);
  if (!current) {
    return false;
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
      return false;
    }
    if (action === 'save') {
      const saved = await saveWorkspaceFile(path);
      if (!saved) {
        return false;
      }
    }
  }

  openFileMap.delete(path);
  openTabs.value = openTabs.value.filter((item) => item !== path);
  return true;
}

async function saveTabsByPaths(paths: string[]) {
  const uniquePaths = [...new Set(paths)].filter((path) => isFileDirty(path));
  if (!uniquePaths.length) {
    return true;
  }

  for (const path of uniquePaths) {
    const saved = await saveWorkspaceFile(path, { silent: true });
    if (!saved) {
      return false;
    }
  }

  MessagePlugin.success(workspaceCopy.value.saveSuccess);
  return true;
}

async function handleRefreshFileTab(path: string) {
  await reloadWorkspaceFile(path);
  activeFileTabPathForMenu.value = null;
}

async function handleCloseFileTabsAhead(path: string) {
  const targetIndex = openTabs.value.indexOf(path);
  if (targetIndex <= 0) {
    activeFileTabPathForMenu.value = null;
    return;
  }

  const closed = await closeWorkspaceTabs(openTabs.value.slice(0, targetIndex));
  if (closed) {
    activeFileTabPathForMenu.value = null;
  }
}

async function handleCloseFileTabsBehind(path: string) {
  const targetIndex = openTabs.value.indexOf(path);
  if (targetIndex === -1 || targetIndex >= openTabs.value.length - 1) {
    activeFileTabPathForMenu.value = null;
    return;
  }

  const closed = await closeWorkspaceTabs(openTabs.value.slice(targetIndex + 1));
  if (closed) {
    activeFileTabPathForMenu.value = null;
  }
}

async function handleCloseOtherFileTabs(path: string) {
  const closed = await closeWorkspaceTabs(openTabs.value.filter((item) => item !== path));
  if (closed) {
    activeFileTabPathForMenu.value = null;
  }
}

async function handleCloseAllFileTabs() {
  const closed = await closeWorkspaceTabs([...openTabs.value]);
  if (closed) {
    activeFileTabPathForMenu.value = null;
  }
}

function supportsExplicitSyntaxValidation(language: ProjectWorkspaceMonacoLanguage) {
  return language === 'json' || language === 'yaml';
}

async function runCurrentFileValidation() {
  const current = activeBuffer.value;
  if (!current) {
    MessagePlugin.warning(workspaceCopy.value.validateNoFile);
    return;
  }

  if (!supportsExplicitSyntaxValidation(current.language)) {
    MessagePlugin.info(workspaceCopy.value.fileValidationUnavailable);
    return;
  }

  syntaxCheckLoading.value = true;

  try {
    const markers = await activeWorkspaceEditor()?.waitForDiagnostics?.();
    const errors = normalizeSyntaxErrors(markers);

    if (!errors.length) {
      MessagePlugin.success(workspaceCopy.value.fileValidationPassed);
      return;
    }

    const validationResult = buildSyntaxValidationResult(current, errors);
    syntaxValidationResult.value = validationResult;
    syntaxValidationFiles.value = [validationResult];
    syntaxValidationSkippedPaths.value = [];
    selectedSyntaxFilePath.value = current.path;
    syntaxMarkers.value = validationResult.markers;
    resultIssueIndex.value = 0;
    resultDialogMode.value = 'syntax';
    resultDialogVisible.value = true;
    await queueResultViewerLayout();
    revealCurrentResultIssue();
  } finally {
    syntaxCheckLoading.value = false;
  }
}

function toggleResultDialogFullscreen() {
  resultDialogFullscreen.value = !resultDialogFullscreen.value;
  queueResultViewerLayout();
}

function waitForAnimationFrame() {
  return new Promise<void>((resolve) => {
    requestAnimationFrame(() => {
      resolve();
    });
  });
}

async function handleResultDialogOpened() {
  if (resultDialogMode.value === 'diff') {
    logWorkspaceDiffDebug('dialog-opened', {
      diffViewerReady: diffViewerReady.value,
      fileCount: diffFiles.value.length,
      selectedPath: selectedDiffFile.value?.path ?? '',
    });

    diffViewerReady.value = false;
    await nextTick();
    await waitForAnimationFrame();
    await waitForAnimationFrame();

    diffViewerReady.value = true;
    logWorkspaceDiffDebug('dialog-diff-activated', {
      containerHeight: diffViewerRef.value ? 'mounted' : 'pending',
      fileCount: diffFiles.value.length,
      selectedPath: selectedDiffFile.value?.path ?? '',
    });
    await refreshDiffLineChanges();
  } else {
    syntaxValidationResult.value = syntaxValidationFile.value;
    syntaxMarkers.value = syntaxValidationFile.value?.markers ?? [];
  }

  await queueResultViewerLayout();
  revealCurrentResultIssue();
}

async function queueResultViewerLayout() {
  await nextTick();
  const layoutTasks: Array<Promise<void>> = [];
  if (diffViewerRef.value && typeof diffViewerRef.value.relayout === 'function') {
    layoutTasks.push(diffViewerRef.value.relayout());
  }
  if (syntaxViewerRef.value && typeof syntaxViewerRef.value.relayout === 'function') {
    layoutTasks.push(syntaxViewerRef.value.relayout());
  }
  await Promise.allSettled(layoutTasks);
}

function logWorkspaceDiffDebug(event: string, detail: Record<string, unknown>) {
  if (!isProjectMonacoDebugEnabled()) {
    return;
  }

  logger.debug(`[ConfigurationWorkspaceDiff] ${formatProjectMonacoDebugMessage(event, detail)}`, detail);
}

async function runProjectDeploy() {
  if (hasDirtyFiles.value) {
    await previewBeforeSave('deploy');
    return;
  }

  await executeProjectDeploy();
}

async function executeProjectDeploy() {
  deployLoading.value = true;
  try {
    const response = await postProjectDeploy(projectId.value);
    MessagePlugin.success(response.message || t('project.detail.configuration.deploySuccess'));
    diffResult.value = null;
    resultDialogVisible.value = false;
    await loadWorkspaceDirectory('', { root: true });
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.configuration.deployFailed')));
  } finally {
    deployLoading.value = false;
  }
}

function buildDirtyDiffFiles(paths?: string[]): WorkspacePreviewDiffFile[] {
  const targetPathSet = paths?.length ? new Set(paths) : null;
  return dirtyEditableBuffers.value
    .filter((tab) => !targetPathSet || targetPathSet.has(tab.path))
    .map((tab) => {
      const currentContent = normalizeTextBlock(tab.savedContent);
      const proposedContent = normalizeTextBlock(tab.content);
      const path = normalizeWorkspacePath(tab.path);
      return {
        changed: currentContent !== proposedContent,
        current_content: currentContent,
        current_hash: hashWorkspaceContent(currentContent),
        display_path: path,
        kind: tab.fileKind,
        path,
        proposed_content: proposedContent,
        proposed_hash: hashWorkspaceContent(proposedContent),
      } satisfies WorkspacePreviewDiffFile;
    })
    .filter((file) => file.changed);
}

async function previewBeforeSave(action: PendingWorkspaceAction) {
  const targetPaths = resolvePendingWorkspacePaths(action);
  const files = buildDirtyDiffFiles(targetPaths);
  logWorkspaceDiffDebug('preview-before-save', {
    action,
    diffFileCount: files.length,
    dirtyBufferCount: targetPaths.length,
  });
  if (!files.length) {
    MessagePlugin.info(workspaceCopy.value.diffEmptyDirectSaveHint);
    return proceedAfterDiffConfirmation(action, targetPaths);
  }

  diffResult.value = {
    files,
    has_changes: files.length > 0,
    warnings: [],
  };
  selectedDiffFilePath.value = files[0]?.path || '';
  logWorkspaceDiffDebug('preview-diff-selected', {
    currentHash: files[0]?.current_hash ?? '',
    currentLength: files[0]?.current_content.length ?? 0,
    diffViewerReady: diffViewerReady.value,
    path: files[0]?.path ?? '',
    proposedHash: files[0]?.proposed_hash ?? '',
    proposedLength: files[0]?.proposed_content.length ?? 0,
  });
  syntaxValidationFiles.value = [];
  syntaxValidationResult.value = null;
  syntaxValidationSkippedPaths.value = [];
  pendingWorkspaceActionPaths.value = targetPaths;
  pendingWorkspaceAction.value = action;
  resultDialogMode.value = 'diff';
  resultDialogVisible.value = true;
  return false;
}

function cancelDiffPreview() {
  resultDialogVisible.value = false;
}

async function confirmDiffPreview() {
  if (!pendingWorkspaceAction.value || saveConfirmLoading.value) {
    return;
  }

  saveConfirmLoading.value = true;
  const action = pendingWorkspaceAction.value;
  const actionPaths = [...pendingWorkspaceActionPaths.value];
  try {
    await proceedAfterDiffConfirmation(action, actionPaths);
  } finally {
    saveConfirmLoading.value = false;
  }
}

async function confirmSyntaxValidation() {
  if (!pendingWorkspaceAction.value || saveConfirmLoading.value) {
    return;
  }

  const action = pendingWorkspaceAction.value;
  const confirmAction = await openDialog({
    body: workspaceCopy.value.batchFileValidationRiskBody,
    buttons: [
      {
        label: syntaxConfirmActionLabel.value,
        result: 'save',
        theme: 'primary',
        variant: 'base',
      },
      { label: workspaceCopy.value.cancelAction, result: 'cancel', theme: 'default', variant: 'outline' },
    ],
    title: workspaceCopy.value.batchFileValidationRiskTitle,
  });
  if (confirmAction !== 'save') {
    return;
  }

  saveConfirmLoading.value = true;
  const actionPaths = [...pendingWorkspaceActionPaths.value];
  try {
    const confirmedSkippedPaths = await confirmSkippedSyntaxValidationPaths(action, syntaxValidationSkippedPaths.value);
    if (!confirmedSkippedPaths) {
      return;
    }

    const saved = await finalizePendingWorkspaceAction(action, actionPaths);
    if (!saved) {
      return;
    }

    resultDialogVisible.value = false;
    pendingWorkspaceAction.value = null;
    pendingWorkspaceActionPaths.value = [];
  } finally {
    saveConfirmLoading.value = false;
  }
}

async function refreshDiffLineChanges() {
  await queueResultViewerLayout();
  diffLineChanges.value = diffViewerRef.value?.getLineChanges?.() ?? [];
  resultIssueIndex.value = 0;
  diffViewerRef.value?.revealFirstDiff?.();
}

function revealCurrentResultIssue() {
  if (resultDialogMode.value === 'diff') {
    diffViewerRef.value?.revealFirstDiff?.();
    return;
  }

  syntaxValidationResult.value = syntaxValidationFile.value;
  syntaxViewerRef.value?.revealMarker?.(currentResultIssues.value[resultIssueIndex.value]);
}

function navigateResultIssue(direction: 'next' | 'previous') {
  if (resultDialogMode.value === 'diff') {
    diffViewerRef.value?.navigateDiff?.(direction);
    return;
  }

  const issueCount = currentResultIssues.value.length;
  if (!issueCount) {
    return;
  }
  if (direction === 'next') {
    resultIssueIndex.value = (resultIssueIndex.value + 1) % issueCount;
  } else {
    resultIssueIndex.value = (resultIssueIndex.value - 1 + issueCount) % issueCount;
  }
  revealCurrentResultIssue();
}

function resolveDiffFileLanguage(kind: string | undefined, path: string) {
  return resolveWorkspaceMonacoLanguage({
    fileKind: kind === 'compose' || kind === 'env' ? kind : 'config',
    path,
  });
}

function formatWorkspaceHash(value?: string | null) {
  const normalized = String(value ?? '').trim();
  if (!normalized) {
    return '-';
  }
  if (normalized.length <= 14) {
    return normalized;
  }
  return `${normalized.slice(0, 6)}...${normalized.slice(-6)}`;
}

function diffFileName(path: string) {
  const normalized = String(path).trim().replaceAll('\\', '/');
  const segments = normalized.split('/').filter(Boolean);
  return segments[segments.length - 1] || normalized || '-';
}

function hashWorkspaceContent(value: string) {
  let hash = 2166136261;
  for (const char of value) {
    hash ^= char.charCodeAt(0);
    hash = Math.imul(hash, 16777619);
  }
  return `ws-${(hash >>> 0).toString(16).padStart(8, '0')}`;
}

function buildDiffTreeRows(files: WorkspacePreviewDiffFile[]) {
  const directoryPaths = new Set<string>();
  const fileMap = new Map<string, WorkspacePreviewDiffFile>();

  for (const file of files) {
    const normalizedPath = normalizeWorkspacePath(file.path);
    fileMap.set(normalizedPath, file);

    const segments = normalizedPath.split('/').filter(Boolean);
    let currentPath = '';
    for (const segment of segments.slice(0, -1)) {
      currentPath = currentPath ? `${currentPath}/${segment}` : segment;
      directoryPaths.add(currentPath);
    }
  }

  const appendRows = (basePath: string, depth: number): DiffTreeRow[] => {
    const childDirectories = [...directoryPaths]
      .filter((path) => resolveWorkspaceParentPath(path) === basePath)
      .sort((left, right) => left.localeCompare(right, undefined, { sensitivity: 'base' }));
    const childFiles = [...fileMap.entries()]
      .filter(([path]) => resolveWorkspaceParentPath(path) === basePath)
      .sort(([left], [right]) => left.localeCompare(right, undefined, { sensitivity: 'base' }));

    const rows: DiffTreeRow[] = [];
    for (const directoryPath of childDirectories) {
      rows.push({
        depth,
        file: null,
        name: diffFileName(directoryPath),
        path: directoryPath,
        type: 'directory',
      });
      rows.push(...appendRows(directoryPath, depth + 1));
    }

    for (const [path, file] of childFiles) {
      rows.push({
        depth,
        file,
        name: diffFileName(file.display_path || path),
        path,
        type: 'file',
      });
    }

    return rows;
  };

  return appendRows('', 0);
}

function buildSyntaxTreeRows(files: WorkspaceSyntaxValidationResult[]) {
  const directoryPaths = new Set<string>();
  const fileMap = new Map<string, WorkspaceSyntaxValidationResult>();

  for (const file of files) {
    const normalizedPath = normalizeWorkspacePath(file.path);
    fileMap.set(normalizedPath, file);

    const segments = normalizedPath.split('/').filter(Boolean);
    let currentPath = '';
    for (const segment of segments.slice(0, -1)) {
      currentPath = currentPath ? `${currentPath}/${segment}` : segment;
      directoryPaths.add(currentPath);
    }
  }

  const appendRows = (basePath: string, depth: number): WorkspaceSyntaxTreeRow[] => {
    const childDirectories = [...directoryPaths]
      .filter((path) => resolveWorkspaceParentPath(path) === basePath)
      .sort((left, right) => left.localeCompare(right, undefined, { sensitivity: 'base' }));
    const childFiles = [...fileMap.entries()]
      .filter(([path]) => resolveWorkspaceParentPath(path) === basePath)
      .sort(([left], [right]) => left.localeCompare(right, undefined, { sensitivity: 'base' }));

    const rows: WorkspaceSyntaxTreeRow[] = [];
    for (const directoryPath of childDirectories) {
      rows.push({
        depth,
        file: null,
        name: diffFileName(directoryPath),
        path: directoryPath,
        type: 'directory',
      });
      rows.push(...appendRows(directoryPath, depth + 1));
    }

    for (const [path, file] of childFiles) {
      rows.push({
        depth,
        file,
        name: diffFileName(path),
        path,
        type: 'file',
      });
    }

    return rows;
  };

  return appendRows('', 0);
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
  if (event.key === 'Escape' && workspaceFullscreen.value) {
    workspaceFullscreen.value = false;
    return;
  }

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

function normalizeWorkspacePath(path: string) {
  return String(path || '')
    .replace(/\\/g, '/')
    .replace(/^\/+|\/+$/g, '');
}

function resolveWorkspaceParentPath(path: string) {
  const normalizedPath = normalizeWorkspacePath(path);
  if (!normalizedPath.includes('/')) {
    return '';
  }

  return normalizedPath.split('/').slice(0, -1).join('/');
}

function resolveStoredSidebarWidth() {
  if (typeof window === 'undefined') {
    return SIDEBAR_DEFAULT_WIDTH;
  }

  try {
    const raw = window.localStorage.getItem(EDITOR_WIDTH_STORAGE_KEY);
    const parsed = Number(raw);
    return Number.isFinite(parsed) ? clampSidebarWidth(parsed) : SIDEBAR_DEFAULT_WIDTH;
  } catch {
    return SIDEBAR_DEFAULT_WIDTH;
  }
}

function clampSidebarWidth(value: number) {
  return Math.max(SIDEBAR_MIN_WIDTH, Math.min(SIDEBAR_MAX_WIDTH, Math.round(value)));
}

function updateSidebarWidth(value: number) {
  sidebarWidth.value = clampSidebarWidth(value);
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(EDITOR_WIDTH_STORAGE_KEY, String(sidebarWidth.value));
  } catch {
    return;
  }
}

function syncWorkspaceViewport() {
  if (typeof window === 'undefined') {
    return;
  }

  viewportWidth.value = window.innerWidth;
  sidebarWidth.value = clampSidebarWidth(sidebarWidth.value);
  syncEditorViewportHeight();
}

function syncEditorViewportHeight() {
  if (typeof window === 'undefined') {
    return;
  }

  editorViewportHeight.value = Math.max(560, Math.floor(window.innerHeight - 360));
}
</script>
<style scoped lang="less">
.project-configuration-workspace {
  min-width: 0;

  --graft-workspace-editor-surface: color-mix(
    in srgb,
    var(--td-bg-color-container) 84%,
    var(--graft-shell-content-bg, var(--td-bg-color-page)) 16%
  );
  --graft-workspace-editor-surface-raised: color-mix(
    in srgb,
    var(--graft-workspace-editor-surface) 82%,
    var(--td-bg-color-container-hover) 18%
  );
  --graft-workspace-editor-surface-muted: color-mix(
    in srgb,
    var(--graft-workspace-editor-surface) 78%,
    var(--graft-shell-content-bg, var(--td-bg-color-page)) 22%
  );
  --graft-workspace-editor-border: color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  --graft-workspace-editor-foreground: var(--td-text-color-primary);
  --graft-workspace-editor-foreground-muted: color-mix(
    in srgb,
    var(--td-text-color-secondary) 92%,
    var(--td-text-color-primary)
  );
  --graft-workspace-editor-foreground-subtle: color-mix(in srgb, var(--td-text-color-placeholder) 78%, transparent);
  --graft-workspace-editor-accent: var(--td-brand-color-6);
  --graft-workspace-editor-line-highlight: color-mix(in srgb, var(--td-brand-color-6) 13%, transparent);
  --graft-workspace-editor-selection: color-mix(in srgb, var(--td-brand-color-6) 28%, transparent);
  --graft-workspace-editor-selection-inactive: color-mix(in srgb, var(--td-brand-color-6) 18%, transparent);
  --graft-workspace-editor-indent-guide: color-mix(in srgb, var(--td-text-color-placeholder) 24%, transparent);
  --graft-workspace-editor-indent-guide-active: color-mix(
    in srgb,
    var(--td-brand-color-6) 30%,
    var(--td-component-stroke)
  );
  --graft-workspace-editor-find-match: color-mix(in srgb, var(--td-brand-color-6) 24%, var(--td-warning-color-1));
  --graft-workspace-editor-find-match-border: color-mix(
    in srgb,
    var(--td-brand-color-6) 52%,
    var(--td-component-stroke)
  );
  --graft-workspace-editor-diff-added: color-mix(in srgb, var(--td-success-color-5) 18%, transparent);
  --graft-workspace-editor-diff-removed: color-mix(in srgb, var(--td-error-color-5) 18%, transparent);
  --graft-workspace-tab-indicator: var(--td-brand-color-6);
  --graft-workspace-tab-hover: color-mix(in srgb, var(--td-brand-color-6) 8%, transparent);
}

.project-configuration-workspace__sr-only {
  height: 1px;
  margin: calc(-1 * var(--graft-theme-density-scale));
  overflow: hidden;
  padding: 0;
  position: absolute;
  white-space: nowrap;
  width: 1px;
}

.project-configuration-workspace--fullscreen {
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

.project-configuration-workspace__summary-strip .project-configuration-workspace__section-head p,
.project-configuration-workspace__result-dialog-header .project-configuration-workspace__section-head p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-4) 0 0;
}

.project-configuration-workspace__summary-technical {
  display: block;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-configuration-workspace__result-dialog-title-block {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
  min-width: 0;
}

.project-configuration-workspace__result-dialog-title-block p {
  margin-top: 0 !important;
}

.project-configuration-workspace__result-dialog-summary {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  margin-top: var(--graft-density-gap-4);
}

.project-configuration-workspace__result-dialog-summary-pill {
  align-items: center;
  background: color-mix(in srgb, var(--graft-workspace-editor-surface-muted) 88%, transparent);
  border: 1px solid color-mix(in srgb, var(--graft-workspace-editor-border) 82%, transparent);
  border-radius: 999px;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  gap: var(--graft-density-gap-6);
  max-width: 100%;
  min-width: 0;
  padding: 0 var(--graft-density-gap-10);
}

.project-configuration-workspace__result-dialog-summary-label,
.project-configuration-workspace__result-dialog-summary-value {
  display: inline-block;
  font: var(--td-font-body-small);
  line-height: 28px;
}

.project-configuration-workspace__result-dialog-summary-value {
  color: var(--td-text-color-primary);
  max-width: min(42vw, 480px);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-configuration-workspace__result-dialog-actions {
  align-self: flex-end;
  padding-top: var(--graft-density-gap-6);
}

.project-configuration-workspace__main-grid {
  display: grid;
  gap: var(--graft-density-gap-4);
  grid-template-columns: minmax(0, auto) auto minmax(0, 1fr);
  min-height: 0;
}

.project-configuration-workspace--fullscreen .project-configuration-workspace__summary-strip {
  display: none;
}

.project-configuration-workspace--fullscreen .project-configuration-workspace__main-grid {
  background: var(--graft-shell-content-bg, var(--td-bg-color-page));
  height: calc(100vh - 32px);
  inset: var(--graft-density-gap-16);
  margin-top: 0;
  min-height: calc(100vh - 32px);
  padding: 0;
  position: fixed;
  z-index: 4400;
}

.project-configuration-workspace--fullscreen .project-configuration-workspace__sidebar,
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-stack,
.project-configuration-workspace--fullscreen .project-configuration-workspace__browser-card,
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-surface,
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-stage,
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-loading {
  height: 100%;
}

.project-configuration-workspace--fullscreen
  .project-configuration-workspace__editor-stack
  :deep(.content-viewer-frame),
.project-configuration-workspace--fullscreen
  .project-configuration-workspace__editor-stack
  :deep(.content-viewer-frame__panel),
.project-configuration-workspace--fullscreen
  .project-configuration-workspace__editor-stack
  :deep(.content-viewer-frame__surface),
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-stack :deep(.t-loading__parent),
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-stack :deep(.t-loading__content),
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-stack :deep(.t-loading__wrap) {
  height: 100%;
}

.project-configuration-workspace__sidebar,
.project-configuration-workspace__browser-card,
.project-configuration-workspace__diff-sidebar,
.project-configuration-workspace__diff-stage,
.project-configuration-workspace__diff-surface,
.project-configuration-workspace__editor-stack,
.project-configuration-workspace__feedback-panel,
.project-configuration-workspace__result-dialog,
.project-configuration-workspace__result-viewer,
.project-configuration-workspace__readonly-viewer {
  min-height: 0;
  min-width: 0;
}

.project-configuration-workspace__browser-alert,
.project-configuration-workspace__editor-alert {
  margin-bottom: var(--graft-density-gap-12);
}

.project-configuration-workspace__browser-card {
  background:
    radial-gradient(circle at top left, color-mix(in srgb, var(--td-brand-color-6) 7%, transparent), transparent 38%),
    var(--graft-workspace-editor-surface-muted);
  border-color: var(--graft-workspace-editor-border);
  box-shadow: none;
  height: 100%;
}

.project-configuration-workspace__browser-card :deep(.t-card__header) {
  padding-bottom: var(--graft-density-gap-8);
}

.project-configuration-workspace__browser-card :deep(.t-card__body) {
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding-top: var(--graft-density-gap-8);
}

.project-configuration-workspace__tree-toolbar-button {
  color: var(--td-text-color-secondary);
}

.project-configuration-workspace__tree-toolbar-button:hover {
  color: var(--td-brand-color-6);
}

.project-configuration-workspace__tree {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
  min-height: 0;
  overflow: auto;
}

.project-configuration-workspace__tree-row {
  align-items: center;
  border-radius: var(--td-radius-default);
  display: grid;
  gap: var(--graft-density-gap-4);
  grid-template-columns: 18px minmax(0, 1fr) auto;
  min-width: 0;
  padding-left: calc(var(--workspace-tree-depth, 0) * var(--graft-density-gap-14));
  transition: background-color 0.2s ease;
}

.project-configuration-workspace__tree-row--active {
  background: color-mix(in srgb, var(--td-brand-color-6) 10%, transparent);
}

.project-configuration-workspace__tree-row--readonly {
  opacity: 0.68;
}

.project-configuration-workspace__tree-expander,
.project-configuration-workspace__tree-expander-placeholder {
  align-items: center;
  color: var(--td-text-color-placeholder);
  display: inline-flex;
  height: 24px;
  justify-content: center;
  width: 18px;
}

.project-configuration-workspace__tree-expander {
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  cursor: pointer;
  padding: 0;
}

.project-configuration-workspace__tree-expander:hover {
  background: color-mix(in srgb, var(--td-brand-color-6) 10%, transparent);
  color: var(--td-text-color-primary);
}

.project-configuration-workspace__tree-expander-icon {
  font-size: var(--td-font-size-body-small);
  line-height: 1;
}

.project-configuration-workspace__tree-entry {
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

.project-configuration-workspace__tree-entry:hover {
  color: var(--td-text-color-primary);
}

.project-configuration-workspace__tree-menu-trigger {
  align-items: center;
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  display: inline-flex;
  flex: 0 0 auto;
  height: 30px;
  justify-content: center;
  padding: 0;
  width: 30px;
}

.project-configuration-workspace__tree-menu-trigger:hover,
.project-configuration-workspace__tree-menu-trigger:focus-visible {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.project-configuration-workspace__tree-menu-trigger:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: -2px;
}

.project-configuration-workspace__browser-icon {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  flex: 0 0 auto;
  height: 16px;
  justify-content: center;
  line-height: 0;
  width: 16px;
}

.project-configuration-workspace__browser-icon :deep(svg) {
  height: 16px;
  width: 16px;
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

.project-configuration-workspace__browser-main {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
  min-width: 0;
}

.project-configuration-workspace__browser-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-configuration-workspace__browser-meta {
  color: var(--td-text-color-placeholder);
  display: flex;
  flex-wrap: wrap;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.project-configuration-workspace__tree-actions {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-8);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.project-configuration-workspace__tree-row:hover .project-configuration-workspace__tree-actions,
.project-configuration-workspace__tree-actions--visible {
  opacity: 1;
}

.project-configuration-workspace__annotation-button {
  color: var(--td-text-color-placeholder);
}

.project-configuration-workspace__annotation-button:hover {
  color: var(--td-brand-color-6);
}

.project-configuration-workspace__tree-error {
  color: var(--td-error-color-6);
  font: var(--td-font-body-small);
  margin: 0;
  padding-left: calc(
    (var(--workspace-tree-depth, 0) * var(--graft-density-gap-14)) + var(--graft-density-gap-24) +
      var(--graft-density-gap-2)
  );
}

.project-configuration-workspace__splitter {
  align-items: center;
  cursor: col-resize;
  display: flex;
  justify-content: center;
  min-height: 0;
  width: 2px;
}

.project-configuration-workspace__splitter-grip {
  background: color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  border-radius: 999px;
  height: 40px;
  transition: background-color 0.2s ease;
  width: 1px;
}

.project-configuration-workspace__splitter:hover .project-configuration-workspace__splitter-grip {
  background: color-mix(in srgb, var(--td-brand-color-6) 55%, var(--td-component-stroke));
}

.project-configuration-workspace__editor-stack {
  display: flex;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
}

.project-configuration-workspace__editor-stack :deep(.content-viewer-frame__panel) {
  background: var(--graft-workspace-editor-surface);
  border-bottom: 0;
  border-color: var(--graft-workspace-editor-border);
  border-radius: var(--td-radius-large) var(--td-radius-large) 0 0;
  box-shadow: 0 18px 34px color-mix(in srgb, var(--td-brand-color-6) 5%, transparent);
  display: flex;
  flex-direction: column;
}

.project-configuration-workspace__editor-stack :deep(.content-viewer-frame__header) {
  background: linear-gradient(180deg, color-mix(in srgb, var(--td-brand-color-6) 6%, transparent), transparent 72%);
  border-bottom-color: var(--graft-workspace-editor-border);
}

.project-configuration-workspace__editor-stack :deep(.content-viewer-frame__surface) {
  background: var(--graft-workspace-editor-surface);
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-height: 0;
}

.project-configuration-workspace__editor-stack :deep(.content-viewer-frame__resize-grip) {
  background: color-mix(in srgb, var(--td-text-color-placeholder) 42%, transparent);
}

.project-configuration-workspace__editor-surface {
  background: var(--graft-workspace-editor-surface);
  block-size: 100%;
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-block-size: 0;
  min-inline-size: 0;
}

.project-configuration-workspace__tabs {
  margin-bottom: 0;

  :deep(.t-tabs__header) {
    background: linear-gradient(180deg, color-mix(in srgb, var(--td-brand-color-6) 4%, transparent), transparent);
    border-bottom: 1px solid var(--graft-workspace-editor-border);
    margin: 0;
    padding: 0 var(--graft-density-gap-8);
  }

  :deep(.t-tabs__nav) {
    align-items: stretch;
  }

  :deep(.t-tabs__bar) {
    display: none;
  }

  :deep(.t-tabs__nav-item) {
    background: transparent;
    border: 0;
    border-radius: 0;
    border-top: 2px solid transparent;
    color: var(--td-text-color-secondary);
    margin: 0;
    min-height: 42px;
    padding: 0 var(--graft-density-gap-12);
    transition:
      color 0.2s ease,
      background-color 0.2s ease,
      border-color 0.2s ease;
  }

  :deep(.t-tabs__nav-item:hover) {
    background: var(--graft-workspace-tab-hover);
    color: var(--td-text-color-primary);
  }

  :deep(.t-tabs__nav-item.t-is-active) {
    background: color-mix(in srgb, var(--td-brand-color-6) 6%, transparent);
    border-top-color: var(--graft-workspace-tab-indicator);
    color: var(--td-text-color-primary);
  }

  :deep(.t-tabs__nav-item + .t-tabs__nav-item) {
    box-shadow: inset 1px 0 0 color-mix(in srgb, var(--graft-workspace-editor-border) 72%, transparent);
  }
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
  background: var(--graft-workspace-editor-surface);
  block-size: 100%;
  display: block;
  min-block-size: 0;
  min-inline-size: 0;
}

.project-configuration-workspace__editor-loading {
  display: flex;
  flex: 1 1 auto;
  min-block-size: 0;
  padding: 0 var(--graft-density-gap-8) var(--graft-density-gap-8);
}

.project-configuration-workspace__editor-stage {
  display: flex;
  flex: 1 1 auto;
  min-block-size: 0;
}

.project-configuration-workspace__editor-loading :deep(.t-loading__parent),
.project-configuration-workspace__editor-loading :deep(.t-loading__content),
.project-configuration-workspace__editor-loading :deep(.t-loading__wrap) {
  display: flex;
  flex: 1 1 auto;
  min-block-size: 0;
  min-inline-size: 0;
}

.project-configuration-workspace__monaco-editor,
.project-configuration-workspace__monaco-viewer {
  flex: 1 1 auto;
  overflow: hidden;
}

.project-configuration-workspace__monaco-editor :deep(.monaco-editor),
.project-configuration-workspace__monaco-editor :deep(.monaco-diff-editor),
.project-configuration-workspace__monaco-viewer :deep(.monaco-editor),
.project-configuration-workspace__monaco-viewer :deep(.monaco-diff-editor) {
  background: var(--graft-workspace-editor-surface);
}

.project-configuration-workspace__warning-list,
.project-configuration-workspace__feedback-panel {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-height: 0;
}

.project-configuration-workspace__diff-surface {
  display: grid;
  flex: 1 1 auto;
  gap: var(--graft-density-gap-10);
  grid-template-columns: minmax(220px, 260px) minmax(0, 1fr);
  grid-template-rows: minmax(0, 1fr);
  min-height: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12) var(--graft-density-gap-12);
}

.project-configuration-workspace__diff-surface--single {
  grid-template-columns: minmax(0, 1fr);
}

.project-configuration-workspace__diff-sidebar {
  background: var(--graft-workspace-editor-surface-muted);
  border: 1px solid var(--graft-workspace-editor-border);
  border-radius: var(--td-radius-large);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
  min-height: 0;
  min-width: 0;
  overflow: auto;
  padding: var(--graft-density-gap-8);
}

.project-configuration-workspace__diff-sidebar-head {
  border-bottom: 1px solid color-mix(in srgb, var(--graft-workspace-editor-border) 82%, transparent);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
  margin-bottom: var(--graft-density-gap-6);
  padding-bottom: var(--graft-density-gap-8);
}

.project-configuration-workspace__diff-sidebar-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.project-configuration-workspace__diff-sidebar-caption {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-configuration-workspace__tree-row--diff {
  padding-right: var(--graft-density-gap-4);
}

.project-configuration-workspace__tree-row--error {
  background: color-mix(in srgb, var(--td-error-color-5) 12%, transparent);
}

.project-configuration-workspace__diff-stage {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  height: 100%;
  min-height: 0;
}

.project-configuration-workspace__syntax-stage {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-height: 0;
  min-width: 0;
}

.project-configuration-workspace__feedback-panel--with-sidebar {
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: minmax(220px, 260px) minmax(0, 1fr);
  grid-template-rows: minmax(0, 1fr);
}

.project-configuration-workspace__diff-pane-heads {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-configuration-workspace__diff-pane-head {
  align-items: center;
  background: var(--graft-workspace-editor-surface-muted);
  border: 1px solid var(--graft-workspace-editor-border);
  border-radius: var(--td-radius-default);
  display: flex;
  gap: var(--graft-density-gap-8);
  min-width: 0;
  padding: var(--graft-density-gap-8) var(--graft-density-gap-10);
}

.project-configuration-workspace__diff-pane-label {
  color: var(--td-text-color-secondary);
  flex: 0 0 auto;
  font: var(--td-font-body-small);
}

.project-configuration-workspace__hash-text {
  color: var(--td-text-color-primary);
  display: inline-block;
  flex: 1 1 auto;
  font-family: var(--td-font-family-mono, monospace);
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.project-configuration-workspace__result-dialog {
  background: var(--graft-workspace-editor-surface);
  block-size: 100%;
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-height: 0;
}

.project-configuration-workspace__result-dialog-header {
  border-bottom: 1px solid var(--graft-workspace-editor-border);
  padding: var(--graft-density-gap-6) var(--graft-density-gap-12) var(--graft-density-gap-10);
}

.project-configuration-workspace__result-dialog > .project-configuration-workspace__feedback-panel {
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12) var(--graft-density-gap-12);
}

.project-configuration-workspace__result-dialog-footer {
  border-top: 1px solid var(--graft-workspace-editor-border);
  display: flex;
  justify-content: flex-end;
  padding: var(--graft-density-gap-8) var(--graft-density-gap-12) var(--graft-density-gap-10);
}

.project-configuration-workspace__result-viewer,
.project-configuration-workspace__readonly-viewer {
  background: var(--graft-workspace-editor-surface-muted);
  border: 1px solid var(--graft-workspace-editor-border);
  border-radius: var(--td-radius-large);
  display: flex;
  flex: 1 1 auto;
  min-block-size: 360px;
  min-inline-size: 0;
  overflow: hidden;
}

.project-configuration-workspace__diff-stage .project-configuration-workspace__result-viewer {
  min-block-size: 0;
}

.project-configuration-workspace__result-viewer :deep(.project-monaco-diff-surface),
.project-configuration-workspace__result-viewer :deep(.project-monaco-surface),
.project-configuration-workspace__readonly-viewer :deep(.project-monaco-surface) {
  display: flex;
  flex: 1 1 auto;
  height: 100%;
  min-height: 0;
  min-width: 0;
  width: 100%;
}

.project-configuration-workspace__result-viewer :deep(.project-monaco-diff-surface .monaco-diff-editor),
.project-configuration-workspace__result-viewer :deep(.project-monaco-diff-surface .editor),
.project-configuration-workspace__result-viewer :deep(.monaco-diff-editor),
.project-configuration-workspace__result-viewer :deep(.monaco-editor),
.project-configuration-workspace__result-viewer :deep(.overflow-guard),
.project-configuration-workspace__readonly-viewer :deep(.monaco-editor),
.project-configuration-workspace__readonly-viewer :deep(.overflow-guard) {
  height: 100% !important;
  min-height: 0;
  min-width: 0;
  width: 100% !important;
}

:deep(.project-configuration-workspace__result-dialog-shell .t-dialog__body) {
  display: flex;
  flex: 1 1 auto;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  padding: 0;
}

:deep(.project-configuration-workspace__result-dialog-shell .t-dialog__header) {
  padding: 0;
}

:deep(.project-configuration-workspace__result-dialog-shell .t-dialog__body-content) {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

:deep(.project-configuration-workspace__result-dialog-shell .t-dialog) {
  display: flex;
  flex-direction: column;
  height: 80vh;
  max-height: calc(100vh - 48px);
  max-width: min(80vw, 1600px);
  width: min(80vw, 1600px);
}

:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog__body),
:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog__body-content),
:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog__body--fullscreen),
:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog__body--fullscreen--without-footer) {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  padding: 0;
}

:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog) {
  border-radius: 0;
  height: 100vh;
  max-height: 100vh;
  max-width: none;
  width: 100vw;
}

:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog__body) {
  height: 100%;
}

:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog__body-content) {
  height: 100%;
}

.project-configuration-workspace__dialog-body {
  margin: 0;
}

.project-configuration-workspace__context-menu {
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

.project-configuration-workspace__context-menu button {
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-primary);
  cursor: pointer;
  min-height: 32px;
  padding: 0 var(--graft-density-gap-8);
  text-align: left;
}

.project-configuration-workspace__context-menu button:hover:not(:disabled) {
  background: var(--td-bg-color-container-hover);
}

@media (width <= 1024px) {
  .project-configuration-workspace__main-grid,
  .project-configuration-workspace__diff-surface {
    grid-template-columns: minmax(0, 1fr);
  }

  .project-configuration-workspace__splitter {
    display: none;
  }
}
</style>
