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
            <t-button
              theme="primary"
              variant="outline"
              :disabled="!selectedDirectory"
              :loading="inspectLoading"
              @click="handleRefreshInspect"
            >
              {{ t('project.import.actions.refreshInspect') }}
            </t-button>
          </t-space>
        </template>
      </management-page-header>

      <folder-picker :visible="pickerVisible" @close="pickerVisible = false" @confirm="handleDirectoryConfirm" />

      <div class="project-import-layout">
        <section class="project-import-main">
          <t-card :bordered="true" :title="t('project.import.directory.title')">
            <div class="project-import-directory">
              <t-form label-align="top">
                <t-form-item :label="t('project.import.directory.workingDirectory')">
                  <t-input :value="resolvedWorkingDirectory" readonly>
                    <template #suffixIcon>
                      <span class="project-import-directory__suffix">{{
                        t('project.import.directory.readonlyHint')
                      }}</span>
                    </template>
                  </t-input>
                </t-form-item>
              </t-form>

              <div class="project-import-directory__actions">
                <t-button theme="primary" type="button" @click="pickerVisible = true">
                  {{ t('project.import.actions.selectDirectory') }}
                </t-button>
                <t-button
                  theme="default"
                  variant="outline"
                  type="button"
                  :disabled="!selectedDirectory"
                  @click="pickerVisible = true"
                >
                  {{ t('project.import.actions.changeDirectory') }}
                </t-button>
                <t-button
                  theme="default"
                  variant="text"
                  type="button"
                  :disabled="!selectedDirectory && !hasPreview"
                  @click="handleReset"
                >
                  {{ t('project.import.actions.reset') }}
                </t-button>
              </div>
            </div>
          </t-card>

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
                >
                  <template #actions>
                    <t-button theme="primary" type="button" @click="pickerVisible = true">
                      {{ t('project.import.actions.selectDirectory') }}
                    </t-button>
                  </template>
                </management-empty-state>
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
                    :disabled="!selectedDirectory"
                    @click="handleRefreshInspect"
                  >
                    {{ t('project.import.actions.refreshInspect') }}
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
                {{ formatList(inspectResult?.compose_files.map((item) => item.display_path)) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.import.preview.envFiles')">
                {{ formatList(inspectResult?.env_files.map((item) => item.display_path)) }}
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
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import type { FormInstanceFunctions, FormProps, SubmitContext } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, reactive, ref } from 'vue';

import { ManagementEmptyState, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';

import FolderPicker from '../../components/FolderPicker.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../../shared/navigation';
import { useProjectPageContext } from '../../shared/page-context';
import { useProjectImportFlow } from '../../shared/useProjectImportFlow';
import type { ProjectImportDirectoryRef, ProjectImportExecuteResponse } from '../../types/import';

defineOptions({
  name: 'ProjectImportIndex',
});

const { router, tabsRouterStore, t } = useProjectPageContext();
const formRef = ref<FormInstanceFunctions | null>(null);
const pickerVisible = ref(false);

const {
  canImport,
  canonicalProjectNameOverride,
  displayName,
  hasPreview,
  importLoading,
  inspectError,
  inspectLoading,
  inspectResult,
  refreshInspect,
  reset,
  selectDirectory,
  selectedDirectory,
  submitImport,
} = useProjectImportFlow(t);

const formData = reactive({
  display_name: displayName,
  canonical_project_name_override: canonicalProjectNameOverride,
});

const formRules: FormProps['rules'] = {
  display_name: [{ required: true, message: t('project.import.validation.displayNameRequired') }],
};

const resolvedWorkingDirectory = computed(
  () => inspectResult.value?.resolved_working_directory || renderDirectoryFallback(selectedDirectory.value),
);

function renderDirectoryFallback(directory: ProjectImportDirectoryRef | null) {
  if (!directory) {
    return '';
  }
  return directory.path ? `/${directory.path}` : '/';
}

function formatList(items?: string[]) {
  return items?.length ? items.join(', ') : t('project.import.preview.none');
}

async function handleDirectoryConfirm(directory: ProjectImportDirectoryRef) {
  pickerVisible.value = false;
  try {
    const result = await selectDirectory(directory);
    if (result === 'applied') {
      MessagePlugin.success(t('project.import.messages.inspectSuccess'));
    }
  } catch {
    MessagePlugin.error(inspectError.value || t('project.import.messages.inspectFailed'));
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
  pickerVisible.value = false;
  reset();
}
</script>
<style scoped lang="less">
.project-import-layout,
.project-import-preview,
.project-import-preview__alerts,
.project-import-feedback,
.project-import-directory {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.project-import-layout {
  grid-template-columns: minmax(0, 1.35fr) minmax(320px, 1fr);
  margin-top: var(--graft-density-gap-16);
}

.project-import-directory__actions,
.project-import-form-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}

.project-import-directory__suffix {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
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
