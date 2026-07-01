<template>
  <div class="project-import-page" data-page-type="editor">
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
            <t-button theme="primary" variant="outline" :loading="validating" @click="runValidate">
              {{ t('project.import.actions.validate') }}
            </t-button>
          </t-space>
        </template>
      </management-page-header>

      <div class="project-import-layout">
        <section class="project-import-main">
          <t-card :bordered="true" :title="t('project.import.form.title')">
            <t-form
              ref="formRef"
              :data="formData"
              :rules="formRules"
              label-align="top"
              scroll-to-first-error="smooth"
              @submit="handleSubmit"
            >
              <div class="project-import-form-grid">
                <t-form-item :label="t('project.import.form.workingDirectory')" name="working_directory">
                  <t-input
                    v-model="formData.working_directory"
                    :placeholder="t('project.import.form.workingDirectoryPlaceholder')"
                  />
                </t-form-item>
                <t-form-item :label="t('project.import.form.displayName')" name="display_name">
                  <t-input
                    v-model="formData.display_name"
                    :placeholder="t('project.import.form.displayNamePlaceholder')"
                  />
                </t-form-item>
                <t-form-item
                  :label="t('project.import.form.canonicalProjectNameOverride')"
                  name="canonical_project_name_override"
                >
                  <t-input
                    v-model="canonicalProjectNameOverrideModel"
                    :placeholder="t('project.import.form.canonicalProjectNameOverridePlaceholder')"
                  />
                </t-form-item>
              </div>

              <div class="project-import-form-grid">
                <t-form-item :label="t('project.import.form.composeFiles')" name="compose_files">
                  <t-textarea
                    v-model="composeFilesText"
                    :autosize="{ minRows: 3, maxRows: 6 }"
                    :placeholder="t('project.import.form.composeFilesPlaceholder')"
                  />
                </t-form-item>
                <t-form-item :label="t('project.import.form.envFiles')" name="env_files">
                  <t-textarea
                    v-model="envFilesText"
                    :autosize="{ minRows: 3, maxRows: 6 }"
                    :placeholder="t('project.import.form.envFilesPlaceholder')"
                  />
                </t-form-item>
              </div>

              <div class="project-import-form-actions">
                <t-button theme="primary" variant="outline" type="button" :loading="validating" @click="runValidate">
                  {{ t('project.import.actions.validate') }}
                </t-button>
                <t-button theme="primary" type="submit" :loading="importing">
                  {{ t('project.import.actions.import') }}
                </t-button>
                <t-button theme="default" variant="text" type="button" @click="resetForm">
                  {{ t('project.import.actions.reset') }}
                </t-button>
              </div>
            </t-form>
          </t-card>
        </section>

        <section class="project-import-preview">
          <t-card :bordered="true" :title="t('project.import.preview.title')">
            <t-descriptions size="small" :column="1" bordered>
              <t-descriptions-item :label="t('project.import.preview.canonicalProjectName')">
                <code>{{ validationResult?.canonical_project_name || '-' }}</code>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.import.preview.canonicalNameSource')">
                {{ validationResult?.canonical_project_name_source || '-' }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.import.preview.serviceCount')">
                {{ validationResult?.service_count ?? '-' }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.import.preview.configHash')">
                <code>{{ validationResult?.normalized_preview_summary?.config_hash || '-' }}</code>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.import.preview.declaredServices')">
                {{
                  validationResult?.normalized_preview_summary?.declared_service_names?.join(', ') ||
                  t('project.import.preview.none')
                }}
              </t-descriptions-item>
            </t-descriptions>

            <div class="project-import-preview__alerts">
              <t-alert
                v-for="(warning, index) in validationWarnings"
                :key="`warning-${index}-${warning}`"
                theme="warning"
                :message="warning"
              />
              <t-alert
                v-for="(conflict, index) in validationConflicts"
                :key="`conflict-${index}-${conflict}`"
                theme="error"
                :message="conflict"
              />
              <t-empty
                v-if="!validationWarnings.length && !validationConflicts.length && validationResult"
                :description="t('project.import.preview.noDiagnostics')"
              />
              <t-empty v-else-if="!validationResult" :description="t('project.import.preview.empty')" />
            </div>
          </t-card>

          <t-card :bordered="true" :title="t('project.import.preview.filesTitle')">
            <t-descriptions size="small" :column="1" bordered>
              <t-descriptions-item :label="t('project.import.preview.composeFiles')">
                {{ composeFileSummary }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.import.preview.envFiles')">
                {{ envFileSummary }}
              </t-descriptions-item>
            </t-descriptions>
          </t-card>
        </section>
      </div>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import type { FormInstanceFunctions, FormProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, reactive, ref } from 'vue';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { postProjectImport, postProjectImportValidate } from '../../api/project';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../../shared/navigation';
import { useProjectPageContext } from '../../shared/page-context';
import type {
  ProjectImportResponse,
  ProjectImportValidateRequest,
  ProjectImportValidateResponse,
} from '../../types/project';

defineOptions({
  name: 'ProjectImportIndex',
});

const { router, tabsRouterStore, t } = useProjectPageContext();

const formRef = ref<FormInstanceFunctions | null>(null);
const validating = ref(false);
const importing = ref(false);
const validationResult = ref<ProjectImportValidateResponse | null>(null);

const formData = reactive<ProjectImportValidateRequest>({
  working_directory: '',
  compose_files: [],
  env_files: [],
  display_name: '',
  canonical_project_name_override: null,
});

const formRules: FormProps['rules'] = {
  working_directory: [{ required: true, message: t('project.import.validation.workingDirectoryRequired') }],
};

const composeFilesText = computed({
  get: () => formData.compose_files?.join('\n') || '',
  set: (value: string) => {
    formData.compose_files = parseLineArray(value);
  },
});

const envFilesText = computed({
  get: () => formData.env_files?.join('\n') || '',
  set: (value: string) => {
    formData.env_files = parseLineArray(value);
  },
});

const canonicalProjectNameOverrideModel = computed({
  get: () => formData.canonical_project_name_override ?? '',
  set: (value: string) => {
    const normalized = value.trim();
    formData.canonical_project_name_override = normalized ? normalized : null;
  },
});

const validationWarnings = computed(() => validationResult.value?.warnings ?? []);
const validationConflicts = computed(() => validationResult.value?.conflicts ?? []);
const composeFileSummary = computed(
  () =>
    validationResult.value?.compose_files.map((item) => item.display_path || item.absolute_path).join(', ') ||
    t('project.import.preview.none'),
);
const envFileSummary = computed(
  () =>
    validationResult.value?.env_files.map((item) => item.display_path || item.absolute_path).join(', ') ||
    t('project.import.preview.none'),
);

function parseLineArray(value: string) {
  return value
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean);
}

async function runValidate() {
  const valid = await formRef.value?.validate();
  if (valid !== true) return;
  validating.value = true;
  try {
    validationResult.value = await postProjectImportValidate({ ...formData });
    MessagePlugin.success(t('project.import.messages.validateSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.import.messages.validateFailed')));
  } finally {
    validating.value = false;
  }
}

async function handleSubmit() {
  await submitImport();
}

async function submitImport() {
  const valid = await formRef.value?.validate();
  if (valid !== true) return;
  importing.value = true;
  try {
    const response = await postProjectImport({ ...formData });
    MessagePlugin.success(t('project.import.messages.importSuccess'));
    openDetail(response);
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.import.messages.importFailed')));
  } finally {
    importing.value = false;
  }
}

function openDetail(response: ProjectImportResponse) {
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

function resetForm() {
  formData.working_directory = '';
  formData.compose_files = [];
  formData.env_files = [];
  formData.display_name = '';
  formData.canonical_project_name_override = null;
  validationResult.value = null;
}
</script>
<style scoped>
.project-import-layout,
.project-import-preview__alerts {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.project-import-layout {
  grid-template-columns: minmax(0, 1.4fr) minmax(320px, 1fr);
  margin-top: var(--graft-density-gap-16);
}

.project-import-form-grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-import-form-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}

@media (width <= 1080px) {
  .project-import-layout,
  .project-import-form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
