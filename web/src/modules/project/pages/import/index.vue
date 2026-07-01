<template>
  <div class="project-import-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="project.route.import.title"
        description-key="project.import.description"
        :source="{ labelKey: 'project.import.eyebrow', fallback: t('project.import.eyebrow') }"
      >
        <template #actions>
          <t-space size="small" break-line>
            <t-button theme="default" variant="outline" @click="goToList">
              {{ t('project.import.actions.backToList') }}
            </t-button>
            <t-button theme="default" variant="outline" :loading="candidatesLoading" @click="loadCandidates">
              {{ t('project.import.actions.refreshCandidates') }}
            </t-button>
            <t-button
              theme="primary"
              variant="outline"
              :disabled="!selectedCandidateKey"
              :loading="inspectLoading"
              @click="handleRefreshInspect"
            >
              {{ t('project.import.actions.refreshInspect') }}
            </t-button>
          </t-space>
        </template>
      </management-page-header>

      <div class="project-import-surface">
        <t-card :bordered="true" :title="t('project.import.candidates.title')">
          <template #actions>
            <t-space size="small" break-line>
              <t-tag theme="primary" variant="light-outline">
                {{
                  t('project.import.candidates.summary', {
                    ready: readyCandidates.length,
                    unavailable: unavailableCandidates.length,
                  })
                }}
              </t-tag>
            </t-space>
          </template>

          <div class="project-import-candidates">
            <t-alert v-if="candidatesError" theme="error" :message="candidatesError" close-btn />
            <t-loading :loading="candidatesLoading" size="small">
              <management-empty-state
                v-if="!candidates.length && !candidatesError && !candidatesLoading"
                :title="t('project.import.candidates.emptyTitle')"
                :description="t('project.import.candidates.emptyDescription')"
              />

              <template v-else-if="readyCandidates.length || unavailableCandidates.length">
                <section v-if="readyCandidates.length" class="project-import-candidate-section">
                  <div class="project-import-candidate-section__header">
                    <div class="project-import-candidate-section__title">
                      {{ t('project.import.candidates.readyTitle') }}
                    </div>
                    <div class="project-import-candidate-section__description">
                      {{ t('project.import.candidates.readyDescription') }}
                    </div>
                  </div>

                  <div class="project-import-candidate-grid">
                    <t-card
                      v-for="candidate in readyCandidates"
                      :key="candidate.candidate_key"
                      :bordered="true"
                      :title="candidate.canonical_project_name"
                      :class="{
                        'project-import-candidate-card--active': candidate.candidate_key === selectedCandidateKey,
                      }"
                    >
                      <template #actions>
                        <t-tag :theme="candidateStatusTheme(candidate.status)" variant="light-outline">
                          {{ t(`project.import.candidates.status.${candidate.status}`) }}
                        </t-tag>
                      </template>

                      <div class="project-import-candidate-card">
                        <t-descriptions size="small" :column="1" bordered>
                          <t-descriptions-item :label="t('project.import.preview.canonicalProjectName')">
                            <code>{{ candidate.canonical_project_name }}</code>
                          </t-descriptions-item>
                          <t-descriptions-item :label="t('project.import.candidates.configFiles')">
                            {{ formatPathList(collectCandidateConfigFiles(candidate)) }}
                          </t-descriptions-item>
                          <t-descriptions-item :label="t('project.import.candidates.workingDirectory')">
                            <code>{{ candidate.working_directory }}</code>
                          </t-descriptions-item>
                          <t-descriptions-item :label="t('project.import.candidates.workingDirectorySource')">
                            {{
                              t(
                                `project.import.candidates.workingDirectorySourceValues.${candidate.working_directory_source}`,
                              )
                            }}
                          </t-descriptions-item>
                          <t-descriptions-item :label="t('project.import.candidates.runtime')">
                            {{ formatRuntimeLabel(candidate.runtime_type, candidate.runtime_version) }}
                          </t-descriptions-item>
                          <t-descriptions-item :label="t('project.import.candidates.serviceNames')">
                            {{ formatList(candidate.service_names) }}
                          </t-descriptions-item>
                          <t-descriptions-item :label="t('project.import.candidates.containerCounts')">
                            {{ formatContainerCounts(candidate.container_counts) }}
                          </t-descriptions-item>
                        </t-descriptions>

                        <div class="project-import-candidate-card__alerts">
                          <t-alert
                            v-for="(warning, index) in candidate.warnings"
                            :key="`${candidate.candidate_key}-warning-${index}`"
                            theme="warning"
                            :message="formatRuntimeCandidateWarning(warning)"
                          />
                        </div>

                        <div class="project-import-candidate-card__actions">
                          <t-button
                            theme="primary"
                            :loading="inspectLoading && selectedCandidateKey === candidate.candidate_key"
                            :disabled="inspectLoading && selectedCandidateKey !== candidate.candidate_key"
                            @click="handleCandidateInspect(candidate)"
                          >
                            {{
                              selectedCandidateKey === candidate.candidate_key && hasPreview
                                ? t('project.import.actions.reinspectCandidate')
                                : t('project.import.actions.inspectCandidate')
                            }}
                          </t-button>
                        </div>
                      </div>
                    </t-card>
                  </div>
                </section>

                <section v-if="unavailableCandidates.length" class="project-import-candidate-section">
                  <div class="project-import-candidate-section__header">
                    <div class="project-import-candidate-section__title">
                      {{ t('project.import.candidates.unavailableTitle') }}
                    </div>
                    <div class="project-import-candidate-section__description">
                      {{ t('project.import.candidates.unavailableDescription') }}
                    </div>
                  </div>

                  <div class="project-import-candidate-grid">
                    <t-card
                      v-for="candidate in unavailableCandidates"
                      :key="candidate.candidate_key"
                      :bordered="true"
                      :title="candidate.canonical_project_name"
                    >
                      <template #actions>
                        <t-tag :theme="candidateStatusTheme(candidate.status)" variant="light-outline">
                          {{ t(`project.import.candidates.status.${candidate.status}`) }}
                        </t-tag>
                      </template>

                      <div class="project-import-candidate-card">
                        <t-descriptions size="small" :column="1" bordered>
                          <t-descriptions-item :label="t('project.import.preview.canonicalProjectName')">
                            <code>{{ candidate.canonical_project_name }}</code>
                          </t-descriptions-item>
                          <t-descriptions-item :label="t('project.import.candidates.configFiles')">
                            {{ formatPathList(collectCandidateConfigFiles(candidate)) }}
                          </t-descriptions-item>
                          <t-descriptions-item :label="t('project.import.candidates.workingDirectory')">
                            <code>{{ candidate.working_directory }}</code>
                          </t-descriptions-item>
                          <t-descriptions-item :label="t('project.import.candidates.workingDirectorySource')">
                            {{
                              t(
                                `project.import.candidates.workingDirectorySourceValues.${candidate.working_directory_source}`,
                              )
                            }}
                          </t-descriptions-item>
                          <t-descriptions-item :label="t('project.import.candidates.runtime')">
                            {{ formatRuntimeLabel(candidate.runtime_type, candidate.runtime_version) }}
                          </t-descriptions-item>
                          <t-descriptions-item :label="t('project.import.candidates.serviceNames')">
                            {{ formatList(candidate.service_names) }}
                          </t-descriptions-item>
                          <t-descriptions-item :label="t('project.import.candidates.containerCounts')">
                            {{ formatContainerCounts(candidate.container_counts) }}
                          </t-descriptions-item>
                        </t-descriptions>

                        <div class="project-import-candidate-card__alerts">
                          <t-alert theme="warning" :message="candidateUnavailableReason(candidate)" />
                          <t-alert
                            v-for="(reasonCode, index) in candidate.status_reason_codes"
                            :key="`${candidate.candidate_key}-reason-${index}`"
                            theme="error"
                            :message="formatRuntimeCandidateReason(reasonCode)"
                          />
                          <t-alert
                            v-for="(warning, index) in candidate.warnings"
                            :key="`${candidate.candidate_key}-warning-${index}`"
                            theme="info"
                            :message="formatRuntimeCandidateWarning(warning)"
                          />
                        </div>
                      </div>
                    </t-card>
                  </div>
                </section>
              </template>
            </t-loading>
          </div>
        </t-card>

        <div class="project-import-layout">
          <section class="project-import-main">
            <t-card :bordered="true" :title="t('project.import.form.title')">
              <t-loading :loading="inspectLoading" size="small">
                <div v-if="inspectError" class="project-import-feedback">
                  <management-empty-state
                    tone="error"
                    :title="t('project.import.state.inspectErrorTitle')"
                    :description="inspectError"
                  >
                    <template #actions>
                      <t-button theme="primary" type="button" @click="handleRefreshInspect">
                        {{ t('project.import.actions.retryInspect') }}
                      </t-button>
                    </template>
                  </management-empty-state>
                </div>

                <div v-else-if="!hasPreview" class="project-import-feedback">
                  <management-empty-state
                    :title="t('project.import.state.awaitingSelectionTitle')"
                    :description="t('project.import.state.awaitingSelectionDescription')"
                  />
                </div>

                <t-form
                  v-else
                  ref="formRef"
                  :data="formData"
                  :rules="formRules"
                  label-align="top"
                  scroll-to-first-error="smooth"
                  @submit="handleSubmit"
                >
                  <div class="project-import-authority">
                    <t-alert theme="info" :message="t('project.import.form.authorityHint')" />
                    <t-descriptions size="small" :column="1" bordered>
                      <t-descriptions-item :label="t('project.import.directory.configFiles')">
                        {{ formatDisplayPaths(inspectedConfigFiles) }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.import.directory.workingDirectory')">
                        <code>{{ resolvedWorkingDirectory || '-' }}</code>
                      </t-descriptions-item>
                    </t-descriptions>
                  </div>

                  <div class="project-import-form-grid">
                    <t-form-item :label="t('project.import.form.displayName')" name="display_name">
                      <t-input v-model="displayName" :placeholder="t('project.import.form.displayNamePlaceholder')" />
                    </t-form-item>
                    <t-form-item
                      :label="t('project.import.form.canonicalProjectNameOverride')"
                      name="canonical_project_name_override"
                    >
                      <t-input
                        v-model="canonicalProjectNameOverride"
                        :placeholder="t('project.import.form.canonicalProjectNameOverridePlaceholder')"
                      />
                    </t-form-item>
                  </div>

                  <div class="project-import-form-actions">
                    <t-button theme="primary" type="submit" :disabled="!canImport" :loading="importLoading">
                      {{ t('project.import.actions.import') }}
                    </t-button>
                    <t-button
                      theme="default"
                      variant="outline"
                      type="button"
                      :disabled="!selectedCandidateKey"
                      @click="handleRefreshInspect"
                    >
                      {{ t('project.import.actions.refreshInspect') }}
                    </t-button>
                    <t-button theme="default" variant="text" type="button" @click="handleReset">
                      {{ t('project.import.actions.reset') }}
                    </t-button>
                  </div>
                </t-form>
              </t-loading>
            </t-card>
          </section>

          <section class="project-import-preview">
            <t-card :bordered="true" :title="t('project.import.preview.title')">
              <div v-if="!inspectResult && inspectLoading" class="project-import-preview__skeleton">
                <t-skeleton
                  :loading="true"
                  :row-col="[
                    { type: 'text', width: '96%' },
                    { type: 'text', width: '88%' },
                    { type: 'text', width: '92%' },
                    { type: 'text', width: '76%' },
                  ]"
                />
              </div>
              <t-descriptions v-else size="small" :column="1" bordered>
                <t-descriptions-item :label="t('project.import.preview.canonicalProjectName')">
                  <code>{{ inspectResult?.canonical_project_name || '-' }}</code>
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.import.preview.canonicalNameSource')">
                  {{ inspectResult?.canonical_project_name_source || '-' }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.import.preview.validationStatus')">
                  {{ inspectResult?.validation_status || '-' }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.import.preview.serviceCount')">
                  {{ inspectResult?.services.length ?? '-' }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.import.preview.configHash')">
                  <code>{{ inspectResult?.config_hash || '-' }}</code>
                </t-descriptions-item>
              </t-descriptions>

              <div class="project-import-preview__alerts">
                <t-alert
                  v-for="(warning, index) in inspectResult?.warnings || []"
                  :key="`warning-${index}-${warning}`"
                  theme="warning"
                  :message="warning"
                />
                <t-alert
                  v-for="(conflict, index) in inspectResult?.conflicts || []"
                  :key="`conflict-${index}-${conflict}`"
                  theme="error"
                  :message="conflict"
                />
                <t-empty
                  v-if="inspectResult && !(inspectResult.warnings.length || inspectResult.conflicts.length)"
                  :description="t('project.import.preview.noDiagnostics')"
                />
              </div>
            </t-card>

            <t-card :bordered="true" :title="t('project.import.preview.discoveryTitle')">
              <t-descriptions size="small" :column="1" bordered>
                <t-descriptions-item :label="t('project.import.preview.composeFiles')">
                  {{ formatDisplayPaths(inspectResult?.compose_files) }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.import.preview.envFiles')">
                  {{ formatDisplayPaths(inspectResult?.env_files) }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.import.preview.services')">
                  {{ formatList(inspectResult?.services) }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.import.preview.networks')">
                  {{ formatList(inspectResult?.networks) }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.import.preview.volumes')">
                  {{ formatList(inspectResult?.volumes) }}
                </t-descriptions-item>
              </t-descriptions>
            </t-card>
          </section>
        </div>
      </div>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import type { FormInstanceFunctions, FormProps, SubmitContext } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, reactive, ref } from 'vue';

import { ManagementEmptyState, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { getProjectImportRuntimeCandidates } from '../../api/import';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import {
  collectProjectImportRuntimeCandidateConfigFiles,
  isProjectImportRuntimeCandidateReady,
  resolveProjectImportRuntimeCandidateReasonKey,
} from '../../shared/import';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../../shared/navigation';
import { useProjectPageContext } from '../../shared/page-context';
import { useProjectImportFlow } from '../../shared/useProjectImportFlow';
import type { ProjectImportExecuteResponse, ProjectImportRuntimeCandidate } from '../../types/import';

defineOptions({
  name: 'ProjectImportIndex',
});

const { router, tabsRouterStore, t } = useProjectPageContext();
const formRef = ref<FormInstanceFunctions | null>(null);
const candidatesLoading = ref(true);
const candidatesError = ref('');
const candidates = ref<ProjectImportRuntimeCandidate[]>([]);

const {
  canImport,
  canonicalProjectNameOverride,
  displayName,
  hasPreview,
  importLoading,
  inspectCandidate,
  inspectError,
  inspectLoading,
  inspectResult,
  refreshInspect,
  reset,
  selectedCandidateKey,
  submitImport,
} = useProjectImportFlow(t);

const formData = reactive({
  display_name: displayName,
  canonical_project_name_override: canonicalProjectNameOverride,
});

const formRules: FormProps['rules'] = {
  display_name: [{ required: true, message: t('project.import.validation.displayNameRequired') }],
};

const readyCandidates = computed(() => candidates.value.filter((item) => isProjectImportRuntimeCandidateReady(item)));
const unavailableCandidates = computed(() =>
  candidates.value.filter((item) => !isProjectImportRuntimeCandidateReady(item)),
);
const selectedCandidate = computed(
  () => candidates.value.find((item) => item.candidate_key === selectedCandidateKey.value) ?? null,
);
const resolvedWorkingDirectory = computed(
  () => inspectResult.value?.resolved_working_directory || selectedCandidate.value?.working_directory || '',
);
const inspectedConfigFiles = computed(() => [
  ...(inspectResult.value?.compose_files || []),
  ...(inspectResult.value?.env_files || []),
]);

onMounted(() => {
  void loadCandidates();
});

function formatList(items?: string[]) {
  return items?.length ? items.join(', ') : t('project.import.preview.none');
}

function formatDisplayPaths(items?: Array<{ display_path: string }>) {
  return items?.length ? items.map((item) => item.display_path).join(', ') : t('project.import.preview.none');
}

function formatPathList(items?: string[]) {
  return items?.length ? items.join(', ') : t('project.import.preview.none');
}

function formatContainerCounts(counts: ProjectImportRuntimeCandidate['container_counts']) {
  return t('project.import.candidates.containerCountsValue', counts);
}

function formatRuntimeLabel(runtimeType: string, runtimeVersion?: string | null) {
  return runtimeVersion?.trim() ? `${runtimeType} ${runtimeVersion.trim()}` : runtimeType;
}

function formatRuntimeCandidateReason(reasonCode: string) {
  const translationKey = `project.import.candidates.reason.${reasonCode}`;
  const translated = t(translationKey);
  return translated === translationKey ? reasonCode : translated;
}

function formatRuntimeCandidateWarning(warningCode: string) {
  const translationKey = `project.import.candidates.warning.${warningCode}`;
  const translated = t(translationKey);
  return translated === translationKey ? warningCode : translated;
}

function collectCandidateConfigFiles(candidate: ProjectImportRuntimeCandidate) {
  return collectProjectImportRuntimeCandidateConfigFiles(candidate);
}

function candidateStatusTheme(status: ProjectImportRuntimeCandidate['status']) {
  if (status === 'ready') return 'success';
  if (status === 'broken_compose') return 'danger';
  if (status === 'incomplete_metadata') return 'warning';
  if (status === 'unsupported_runtime') return 'default';
  return 'default';
}

function candidateUnavailableReason(candidate: ProjectImportRuntimeCandidate) {
  const reasonKey = resolveProjectImportRuntimeCandidateReasonKey(candidate);
  const translated = t(`project.import.candidates.reason.${reasonKey}`);
  if (translated === `project.import.candidates.reason.${reasonKey}`) {
    return t('project.import.candidates.reason.unavailable');
  }
  return translated;
}

async function loadCandidates() {
  candidatesLoading.value = true;
  candidatesError.value = '';
  try {
    const response = await getProjectImportRuntimeCandidates();
    candidates.value = response?.items ?? [];
  } catch (error) {
    candidates.value = [];
    candidatesError.value = resolveLocalizedErrorMessage(t, error, t('project.import.messages.candidateLoadFailed'));
  } finally {
    candidatesLoading.value = false;
  }
}

async function handleRefreshInspect() {
  try {
    const result = await refreshInspect();
    if (result === 'applied' && inspectResult.value) {
      MessagePlugin.success(t('project.import.messages.inspectSuccess'));
    }
  } catch {
    MessagePlugin.error(inspectError.value || t('project.import.messages.inspectFailed'));
  }
}

async function handleCandidateInspect(candidate: ProjectImportRuntimeCandidate) {
  try {
    const result = await inspectCandidate(candidate);
    if (result === 'applied') {
      MessagePlugin.success(t('project.import.messages.inspectSuccess'));
    }
  } catch {
    MessagePlugin.error(inspectError.value || t('project.import.messages.inspectFailed'));
  }
}

async function handleSubmit(context: SubmitContext) {
  if (context.validateResult !== true) {
    return;
  }

  try {
    const response = await submitImport();
    MessagePlugin.success(t('project.import.messages.importSuccess'));
    openDetail(response);
  } catch {
    MessagePlugin.error(t('project.import.messages.importFailed'));
  }
}

function openDetail(response: ProjectImportExecuteResponse) {
  const project = response.project;
  const target = {
    name: PROJECT_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    params: { id: project.id },
    query: { tab: 'overview' },
  };
  const resolved = router.resolve(target);
  appendResolvedTab(
    tabsRouterStore,
    resolved,
    buildDetailTitleWithFallback('project.route.detail.title', project.display_name),
  );
  void router.push(target);
}

function goToList() {
  void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.LIST.routeName });
}

function handleReset() {
  reset();
}
</script>
<style scoped lang="less">
.project-import-surface,
.project-import-layout,
.project-import-preview,
.project-import-preview__alerts,
.project-import-feedback,
.project-import-authority,
.project-import-candidates,
.project-import-candidate-card,
.project-import-candidate-card__alerts {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.project-import-surface {
  margin-top: var(--graft-density-gap-16);
}

.project-import-layout {
  grid-template-columns: minmax(0, 1.35fr) minmax(320px, 1fr);
  margin-top: var(--graft-density-gap-20);
}

.project-import-candidate-section {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.project-import-candidate-section__header {
  display: grid;
  gap: var(--graft-density-gap-4);
}

.project-import-candidate-section__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  font-weight: 600;
}

.project-import-candidate-section__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-import-candidate-grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
}

.project-import-candidate-card--active {
  box-shadow: 0 0 0 1px var(--td-brand-color);
}

.project-import-candidate-card__actions,
.project-import-form-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}

.project-import-form-grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-import-preview__skeleton {
  padding: var(--graft-density-gap-8) 0;
}

@media (width <= 1080px) {
  .project-import-layout,
  .project-import-form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
