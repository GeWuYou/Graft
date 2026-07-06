<template>
  <div class="project-configuration-workspace" data-page-type="editor">
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
              {{ metadata?.ownership_mode || detailRecord?.ownership_mode || '-' }}
            </t-tag>
          </t-space>
        </template>
      </management-page-header>

      <t-alert
        v-if="authorityNotice"
        class="project-configuration-workspace__notice"
        :theme="managedConfigurationEnabled ? 'info' : 'warning'"
        :message="authorityNotice"
      />

      <t-loading :loading="workspaceLoading" size="small">
        <template v-if="workspaceReady">
          <section class="project-configuration-workspace__summary-strip">
            <t-card :bordered="true">
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
                    <t-button theme="default" variant="outline" @click="openSourceDrawer">
                      {{ workspaceCopy.sourceAction }}
                    </t-button>
                  </t-space>
                </div>
              </template>

              <t-descriptions bordered size="small" :column="5">
                <t-descriptions-item :label="t('project.detail.configuration.composeFiles')">
                  {{ metadata?.compose_files.length || 0 }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.detail.configuration.envFiles')">
                  {{ metadata?.env_files.length || 0 }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.detail.configuration.ownershipMode')">
                  {{ metadata?.ownership_mode || '-' }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.detail.configuration.driftStatus')">
                  {{ driftLabel }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.detail.configuration.refreshStatus')">
                  {{ refreshLabel }}
                </t-descriptions-item>
              </t-descriptions>
            </t-card>
          </section>

          <section class="project-configuration-workspace__main-grid">
            <t-card class="project-configuration-workspace__drafts-nav" :bordered="true">
              <template #header>
                <div class="project-configuration-workspace__section-head">
                  <div>
                    <h2>{{ workspaceCopy.draftsTitle }}</h2>
                    <p>{{ workspaceCopy.draftsHint }}</p>
                  </div>
                </div>
              </template>

              <div class="project-configuration-workspace__draft-list" role="list">
                <button
                  v-for="item in draftItems"
                  :key="item.value"
                  class="project-configuration-workspace__draft-item"
                  :class="{ 'project-configuration-workspace__draft-item--active': activeDraft === item.value }"
                  :aria-current="activeDraft === item.value ? 'true' : undefined"
                  role="listitem"
                  type="button"
                  @click="activeDraft = item.value"
                >
                  <span class="project-configuration-workspace__draft-item-main">
                    <span class="project-configuration-workspace__draft-item-icon" aria-hidden="true" />
                    <span class="project-configuration-workspace__draft-item-title">{{ item.fileName }}</span>
                  </span>
                </button>
              </div>
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
                      <strong>{{ activeDraftItem.label }}</strong>
                      <p>{{ activeDraftDescription }}</p>
                    </div>
                  </div>
                </template>

                <template #header-actions>
                  <t-space size="small" break-line>
                    <t-button theme="default" variant="outline" @click="resetDraftFromCurrent">
                      {{ t('project.detail.configuration.resetDraft') }}
                    </t-button>
                    <t-button
                      theme="default"
                      variant="outline"
                      :disabled="!managedConfigurationEnabled"
                      @click="formatActiveDraft"
                    >
                      {{ t('project.detail.configuration.formatDraft') }}
                    </t-button>
                    <t-button
                      theme="default"
                      variant="outline"
                      :loading="diffLoading"
                      :disabled="!managedConfigurationEnabled"
                      @click="runConfigurationDiff"
                    >
                      {{ t('project.detail.configuration.runDiff') }}
                    </t-button>
                    <t-button
                      theme="default"
                      variant="outline"
                      :loading="validateLoading"
                      :disabled="!managedConfigurationEnabled"
                      @click="runConfigurationValidate"
                    >
                      {{ t('project.detail.configuration.runValidate') }}
                    </t-button>
                    <t-button
                      theme="primary"
                      :loading="deployLoading"
                      :disabled="!managedConfigurationEnabled"
                      @click="runConfigurationDeploy"
                    >
                      {{ t('project.detail.configuration.deploy') }}
                    </t-button>
                  </t-space>
                </template>

                <div class="project-configuration-workspace__editor-surface">
                  <project-monaco-surface
                    v-if="activeDraft === 'compose'"
                    v-model="draft.compose_file_content"
                    class="project-configuration-workspace__monaco-editor"
                    :editor-aria-label="workspaceCopy.composeEditorAriaLabel"
                    language="yaml"
                    model-key="compose-draft"
                    :options="editorOptions"
                    :read-only="!managedConfigurationEnabled"
                    test-id="compose-monaco-editor"
                  />
                  <project-monaco-surface
                    v-else
                    v-model="envDraftContent"
                    class="project-configuration-workspace__monaco-editor"
                    :editor-aria-label="workspaceCopy.envEditorAriaLabel"
                    language="shell"
                    model-key="env-draft"
                    :options="editorOptions"
                    :read-only="!managedConfigurationEnabled"
                    test-id="env-monaco-editor"
                  />
                </div>
              </content-viewer-frame>

              <t-card class="project-configuration-workspace__feedback" :bordered="true">
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
                              :language="resolveFileLanguage(selectedDiffFile.kind)"
                              :modified-key="`diff-modified-${selectedDiffFile.kind}-${selectedDiffFile.path}`"
                              :modified-value="selectedDiffFile.proposed_content"
                              :original-key="`diff-original-${selectedDiffFile.kind}-${selectedDiffFile.path}`"
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
                            :editor-aria-label="workspaceCopy.normalizedPreviewAriaLabel"
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

    <t-drawer v-model:visible="sourceDrawerVisible" :header="workspaceCopy.sourceDrawerTitle" size="760px">
      <template v-if="sourceFiles.length">
        <t-tabs
          v-model:value="selectedSourceFileTab"
          class="project-configuration-workspace__source-tabs"
          @change="handleSourceFileChange"
        >
          <t-tab-panel
            v-for="file in sourceFiles"
            :key="String(file.id)"
            :value="String(file.id)"
            :label="file.display_path.split('/').at(-1) || file.display_path"
          />
        </t-tabs>
        <t-loading :loading="sourceFileLoading" size="small">
          <template v-if="selectedSourceFileResponse">
            <t-descriptions bordered size="small" :column="1">
              <t-descriptions-item :label="t('project.detail.configuration.fileContentTitle')">
                {{ selectedSourceFileResponse.path }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.detail.configuration.downloadName')">
                {{ selectedSourceFileResponse.download_name }}
              </t-descriptions-item>
            </t-descriptions>
            <div class="project-configuration-workspace__drawer-actions">
              <t-button theme="default" variant="outline" @click="copySelectedSourceFile">
                {{ t('project.detail.configuration.copyContent') }}
              </t-button>
            </div>
            <div class="project-configuration-workspace__drawer-viewer">
              <project-monaco-surface
                class="project-configuration-workspace__monaco-viewer"
                :model-value="selectedSourceFileResponse.content"
                :editor-aria-label="workspaceCopy.sourceViewerAriaLabel"
                :language="resolveFileLanguage(selectedSourceFile.kind)"
                :model-key="`source-${selectedSourceFile.id}`"
                :options="readonlyOptions"
                read-only
                test-id="source-monaco-viewer"
              />
            </div>
          </template>
          <t-empty v-else :description="t('project.detail.configuration.fileEmpty')" />
        </t-loading>
      </template>
      <t-empty v-else :description="workspaceCopy.noSourceFiles" />
    </t-drawer>
  </div>
</template>
<script setup lang="ts">
import { DialogPlugin } from 'tdesign-vue-next/es/dialog';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';

import type { components } from '@/contracts/openapi/generated/schema';
import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import ContentViewerFrame from '@/shared/components/viewer/ContentViewerFrame.vue';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { copyText } from '@/shared/observability/copy';
import { createLogger } from '@/utils/logger';

import {
  getProject,
  getProjectConfiguration,
  getProjectConfigurationFile,
  getProjectConfigurationPreview,
  postProjectConfigurationDiff,
  postProjectConfigurationValidate,
  postProjectDeploy,
} from '../../api/project';
import ProjectMonacoDiffSurface from '../../components/ProjectMonacoDiffSurface.vue';
import ProjectMonacoSurface from '../../components/ProjectMonacoSurface.vue';
import {
  buildConfigurationDraftRequest,
  normalizeTextBlock,
  type ProjectConfigurationDraft,
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
import { resolveConfigurationWorkspaceCopy } from './workspace-copy';

defineOptions({
  name: 'ProjectConfigurationWorkspaceIndex',
});

type ProjectFileKind = components['schemas']['project-file-kind'];
type DraftTab = 'compose' | 'env';
type FeedbackTab = 'diff' | 'validation';

const logger = createLogger('project.configuration-workspace');
const route = useRoute();
const { locale, t } = useProjectPageContext();

const workspaceLoading = ref(false);
const workspaceError = ref('');
const detailRecord = ref<Awaited<ReturnType<typeof getProject>> | null>(null);
const metadata = ref<Awaited<ReturnType<typeof getProjectConfiguration>> | null>(null);
const snapshotPreview = ref<Awaited<ReturnType<typeof getProjectConfigurationPreview>> | null>(null);
const sourceFileCache = reactive(new Map<number, Awaited<ReturnType<typeof getProjectConfigurationFile>>>());
const activeDraft = ref<DraftTab>('compose');
const feedbackTab = ref<FeedbackTab>('diff');
const diffResult = ref<Awaited<ReturnType<typeof postProjectConfigurationDiff>> | null>(null);
const validateResult = ref<Awaited<ReturnType<typeof postProjectConfigurationValidate>> | null>(null);
const diffLoading = ref(false);
const validateLoading = ref(false);
const deployLoading = ref(false);
const snapshotLoading = ref(false);
const sourceFileLoading = ref(false);
const snapshotDrawerVisible = ref(false);
const sourceDrawerVisible = ref(false);
const selectedDiffFilePath = ref('');
const selectedSourceFileTab = ref('');
const originalDraft = reactive<ProjectConfigurationDraft>({
  compose_file_content: '',
  env_file_content: '',
});
const draft = reactive<ProjectConfigurationDraft>({
  compose_file_content: '',
  env_file_content: '',
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
const managedConfigurationEnabled = computed(() => detailRecord.value?.ownership_mode === 'managed-root-dedicated');
const authorityNotice = computed(() => {
  if (!detailRecord.value) {
    return '';
  }
  return managedConfigurationEnabled.value
    ? t('project.detail.configuration.managedAuthorityHint')
    : t('project.detail.configuration.externalAuthorityHint');
});
const sourceFiles = computed(() => [...(metadata.value?.compose_files || []), ...(metadata.value?.env_files || [])]);
const selectedSourceFile = computed(
  () =>
    sourceFiles.value.find((file) => String(file.id) === selectedSourceFileTab.value) ?? sourceFiles.value[0] ?? null,
);
const selectedSourceFileResponse = computed(() =>
  selectedSourceFile.value ? (sourceFileCache.get(selectedSourceFile.value.id) ?? null) : null,
);
const envDraftContent = computed({
  get: () => draft.env_file_content,
  set: (value: string) => {
    draft.env_file_content = value;
  },
});
const workspaceReady = computed(() => Boolean(detailRecord.value && metadata.value && !workspaceError.value));
const composeDraftFilePath = computed(() => metadata.value?.compose_files[0]?.display_path || 'compose.yaml');
const envDraftFilePath = computed(() => metadata.value?.env_files[0]?.display_path || '.env');
const activeDraftItem = computed(() => {
  return activeDraft.value === 'compose'
    ? { label: t('project.detail.configuration.composeEditorTab'), path: composeDraftFilePath.value }
    : { label: t('project.detail.configuration.envEditorTab'), path: envDraftFilePath.value };
});
const activeDraftDescription = computed(() =>
  activeDraft.value === 'compose'
    ? t('project.detail.configuration.composeEditorDescription')
    : t('project.detail.configuration.envEditorDescription'),
);
const draftItems = computed(() => [
  {
    fileName: resolveFileName(composeDraftFilePath.value),
    value: 'compose' as const,
  },
  {
    fileName: resolveFileName(envDraftFilePath.value),
    value: 'env' as const,
  },
]);
const hasFeedback = computed(() => Boolean(diffResult.value || validateResult.value));
const selectedDiffFile = computed(
  () =>
    diffResult.value?.files.find((file) => file.path === selectedDiffFilePath.value) ??
    diffResult.value?.files[0] ??
    null,
);

onMounted(() => {
  void loadWorkspace();
});

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
    await hydrateDraftFromCurrent(configurationMetadata);
    if (sourceFiles.value[0]) {
      selectedSourceFileTab.value = String(sourceFiles.value[0].id);
    }
  } catch (error) {
    logger.error('failed to load project configuration workspace', error);
    workspaceError.value = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
    MessagePlugin.error(workspaceError.value);
  } finally {
    workspaceLoading.value = false;
  }
}

async function hydrateDraftFromCurrent(configurationMetadata: NonNullable<typeof metadata.value>) {
  const composeFileId = configurationMetadata.compose_files[0]?.id;
  const envFileId = configurationMetadata.env_files[0]?.id;
  const [composeResponse, envResponse] = await Promise.all([
    typeof composeFileId === 'number' ? ensureFileLoaded(composeFileId) : Promise.resolve(null),
    typeof envFileId === 'number' ? ensureFileLoaded(envFileId) : Promise.resolve(null),
  ]);

  originalDraft.compose_file_content = composeResponse?.content || '';
  originalDraft.env_file_content = envResponse?.content || '';
  draft.compose_file_content = originalDraft.compose_file_content;
  draft.env_file_content = originalDraft.env_file_content;
}

async function ensureFileLoaded(fileId: number) {
  const cached = sourceFileCache.get(fileId);
  if (cached) {
    return cached;
  }

  const response = await getProjectConfigurationFile(projectId.value, fileId);
  sourceFileCache.set(fileId, response);
  return response;
}

function resetDraftFromCurrent() {
  draft.compose_file_content = originalDraft.compose_file_content;
  draft.env_file_content = originalDraft.env_file_content;
  diffResult.value = null;
  validateResult.value = null;
  selectedDiffFilePath.value = '';
  feedbackTab.value = 'diff';
}

function formatActiveDraft() {
  if (activeDraft.value === 'compose') {
    draft.compose_file_content = normalizeTextBlock(draft.compose_file_content);
    return;
  }
  draft.env_file_content = normalizeTextBlock(draft.env_file_content);
}

async function runConfigurationDiff() {
  if (!Number.isFinite(projectId.value) || !managedConfigurationEnabled.value) {
    MessagePlugin.warning(authorityNotice.value);
    return;
  }
  diffLoading.value = true;
  try {
    diffResult.value = await postProjectConfigurationDiff(projectId.value, buildConfigurationDraftRequest(draft));
    selectedDiffFilePath.value = diffResult.value.files[0]?.path || '';
    feedbackTab.value = 'diff';
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.configuration.diffFailed')));
  } finally {
    diffLoading.value = false;
  }
}

async function runConfigurationValidate() {
  if (!Number.isFinite(projectId.value) || !managedConfigurationEnabled.value) {
    MessagePlugin.warning(authorityNotice.value);
    return;
  }
  validateLoading.value = true;
  try {
    validateResult.value = await postProjectConfigurationValidate(
      projectId.value,
      buildConfigurationDraftRequest(draft),
    );
    feedbackTab.value = 'validation';
    MessagePlugin.success(t('project.detail.configuration.validateSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.configuration.validateFailed')));
  } finally {
    validateLoading.value = false;
  }
}

function runConfigurationDeploy() {
  if (!Number.isFinite(projectId.value) || !managedConfigurationEnabled.value) {
    MessagePlugin.warning(authorityNotice.value);
    return;
  }

  const dialog = DialogPlugin.confirm({
    body: t('project.detail.configuration.deployConfirmDescription'),
    cancelBtn: t('project.list.actions.cancel'),
    confirmBtn: {
      content: t('project.detail.configuration.deploy'),
      theme: 'primary',
    },
    header: t('project.detail.configuration.deployConfirmTitle'),
    onConfirm: async () => {
      deployLoading.value = true;
      try {
        const response = await postProjectDeploy(projectId.value, buildConfigurationDraftRequest(draft));
        MessagePlugin.success(response.message || t('project.detail.configuration.deploySuccess'));
        diffResult.value = null;
        validateResult.value = null;
        snapshotPreview.value = null;
        sourceFileCache.clear();
        await loadWorkspace();
      } catch (error) {
        MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.configuration.deployFailed')));
      } finally {
        deployLoading.value = false;
        dialog.destroy();
      }
    },
  });
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

async function openSourceDrawer() {
  sourceDrawerVisible.value = true;
  if (!selectedSourceFile.value) {
    return;
  }
  await ensureSelectedSourceFileLoaded();
}

async function handleSourceFileChange(value: string | number) {
  selectedSourceFileTab.value = String(value);
  await ensureSelectedSourceFileLoaded();
}

async function ensureSelectedSourceFileLoaded() {
  if (!selectedSourceFile.value) {
    return;
  }
  sourceFileLoading.value = true;
  try {
    await ensureFileLoaded(selectedSourceFile.value.id);
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.retry')));
  } finally {
    sourceFileLoading.value = false;
  }
}

async function copySelectedSourceFile() {
  if (!selectedSourceFileResponse.value?.content) {
    return;
  }
  const copied = await copyText(selectedSourceFileResponse.value.content);
  if (copied) {
    MessagePlugin.success(t('project.detail.configuration.copySuccess'));
    return;
  }
  MessagePlugin.error(t('project.detail.configuration.copyError'));
}

function resolveFileLanguage(kind: ProjectFileKind | undefined) {
  return kind === 'compose' ? 'yaml' : 'shell';
}

function resolveFileName(path: string) {
  const normalized = String(path || '').trim();
  if (!normalized) {
    return 'untitled';
  }
  return normalized.split('/').at(-1) || normalized;
}
</script>
<style scoped lang="less">
.project-configuration-workspace {
  min-width: 0;
}

.project-configuration-workspace__notice {
  margin-block: var(--graft-density-gap-16);
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
  grid-template-columns: minmax(220px, 260px) minmax(0, 1fr);
}

.project-configuration-workspace__drafts-nav,
.project-configuration-workspace__editor-column,
.project-configuration-workspace__feedback-panel,
.project-configuration-workspace__diff-layout,
.project-configuration-workspace__diff-viewer,
.project-configuration-workspace__readonly-viewer,
.project-configuration-workspace__drawer-viewer {
  min-height: 0;
  min-width: 0;
}

.project-configuration-workspace__editor-column {
  display: grid;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.project-configuration-workspace__editor-surface {
  block-size: 100%;
  min-block-size: 560px;
  min-inline-size: 0;
}

.project-configuration-workspace__draft-list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
}

.project-configuration-workspace__draft-item,
.project-configuration-workspace__diff-file {
  background: var(--td-bg-color-container);
  border: 1px solid transparent;
  border-radius: var(--td-radius-default);
  color: inherit;
  cursor: pointer;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
  text-align: left;
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease;
}

.project-configuration-workspace__draft-item:hover,
.project-configuration-workspace__diff-file:hover {
  border-color: var(--td-brand-color-5);
}

.project-configuration-workspace__draft-item--active,
.project-configuration-workspace__diff-file--active {
  background: var(--td-brand-color-1);
  border-color: var(--td-brand-color-6);
}

.project-configuration-workspace__draft-item-main {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-10);
  min-width: 0;
}

.project-configuration-workspace__draft-item-icon {
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--td-brand-color) 16%, transparent), transparent),
    var(--td-bg-color-component);
  border: 1px solid color-mix(in srgb, var(--td-brand-color) 28%, var(--td-component-border));
  border-radius: var(--td-radius-small);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--td-brand-color) 10%, transparent);
  flex: 0 0 14px;
  height: 16px;
  position: relative;
  width: 14px;
}

.project-configuration-workspace__draft-item-icon::before {
  background: color-mix(in srgb, var(--td-brand-color) 58%, transparent);
  border-radius: 1px;
  content: '';
  height: 2px;
  left: 3px;
  position: absolute;
  top: 4px;
  width: 8px;
}

.project-configuration-workspace__draft-item-title {
  color: var(--td-text-color-primary);
  flex: 1 1 auto;
  font: var(--td-font-body-medium);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
}

.project-configuration-workspace__warning-list,
.project-configuration-workspace__drawer-actions {
  display: flex;
  gap: var(--graft-density-gap-12);
}

.project-configuration-workspace__drawer-actions {
  align-items: center;
}

.project-configuration-workspace__feedback-panel,
.project-configuration-workspace__warning-list {
  display: flex;
  flex-direction: column;
}

.project-configuration-workspace__feedback-panel,
.project-configuration-workspace__warning-list,
.project-configuration-workspace__diff-viewer {
  gap: var(--graft-density-gap-12);
}

.project-configuration-workspace__monaco-editor,
.project-configuration-workspace__monaco-viewer {
  block-size: 100%;
  display: block;
  flex: 1 1 auto;
  min-block-size: 0;
  min-inline-size: 0;
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
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
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

.project-configuration-workspace__source-tabs {
  margin-bottom: var(--graft-density-gap-12);
}

@media (width <= 1024px) {
  .project-configuration-workspace__main-grid,
  .project-configuration-workspace__diff-layout {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
