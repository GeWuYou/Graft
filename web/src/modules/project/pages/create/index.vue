<template>
  <div class="project-create-page" data-page-type="editor">
    <management-page-content>
      <management-page-header
        title-key="project.create.title"
        description-key="project.create.description"
        :source="{ labelKey: 'project.create.eyebrow', fallback: t('project.create.eyebrow') }"
      >
        <template #meta>
          <t-space break-line size="small">
            <t-tag :theme="managedRootStatusTheme" variant="light-outline">
              {{ managedRootStatusLabel }}
            </t-tag>
            <t-tag theme="default" variant="light-outline">
              {{ managedRoot?.ownership_mode || '-' }}
            </t-tag>
          </t-space>
        </template>
        <template #actions>
          <t-space size="small" break-line>
            <t-button theme="default" variant="outline" @click="goToList">
              {{ t('project.create.actions.backToList') }}
            </t-button>
            <t-button theme="default" :loading="rootLoading" @click="loadManagedRoot">
              {{ t('project.create.actions.refreshAuthority') }}
            </t-button>
          </t-space>
        </template>
      </management-page-header>

      <t-alert
        v-if="managedRootNotice"
        :theme="managedRootNoticeTheme"
        :message="managedRootNotice"
        class="project-create-page__notice"
      />

      <div class="project-create-layout">
        <section class="project-create-main">
          <t-card :bordered="true" :title="t('project.create.form.title')">
            <t-form
              ref="formRef"
              :data="formData"
              :rules="formRules"
              label-align="top"
              scroll-to-first-error="smooth"
              @submit="handleSubmit"
            >
              <div class="project-create-form-grid">
                <t-form-item :label="t('project.create.form.displayName')" name="display_name">
                  <t-input
                    v-model="formData.display_name"
                    :placeholder="t('project.create.form.displayNamePlaceholder')"
                  />
                </t-form-item>
                <t-form-item :label="t('project.create.form.canonicalProjectName')" name="canonical_project_name">
                  <t-input
                    v-model="formData.canonical_project_name"
                    :placeholder="t('project.create.form.canonicalProjectNamePlaceholder')"
                  />
                </t-form-item>
                <t-form-item
                  :label="t('project.create.form.relativeProjectDirectory')"
                  name="relative_project_directory"
                >
                  <t-input
                    v-model="formData.relative_project_directory"
                    :placeholder="t('project.create.form.relativeProjectDirectoryPlaceholder')"
                  />
                </t-form-item>
                <t-form-item :label="t('project.create.form.composeFileName')" name="compose_file_name">
                  <t-input
                    v-model="formData.compose_file_name"
                    :placeholder="t('project.create.form.composeFileNamePlaceholder')"
                  />
                </t-form-item>
                <t-form-item :label="t('project.create.form.envFileName')" name="env_file_name">
                  <t-input v-model="envFileNameModel" :placeholder="t('project.create.form.envFileNamePlaceholder')" />
                </t-form-item>
              </div>

              <div class="project-create-form-actions">
                <t-button
                  theme="primary"
                  variant="outline"
                  type="button"
                  :disabled="!managedCreateEnabled"
                  :loading="validating"
                  @click="runValidate"
                >
                  {{ t('project.create.actions.validate') }}
                </t-button>
                <t-button theme="primary" type="submit" :disabled="!managedCreateEnabled" :loading="creating">
                  {{ t('project.create.actions.create') }}
                </t-button>
                <t-button theme="default" variant="text" type="button" @click="resetForm">
                  {{ t('project.create.actions.reset') }}
                </t-button>
              </div>
            </t-form>
          </t-card>

          <t-card :bordered="true" :title="t('project.create.validation.title')">
            <t-descriptions size="small" :column="1" bordered>
              <t-descriptions-item :label="t('project.create.validation.rootDirectory')">
                <code>{{ managedRoot?.configured_root_directory || '-' }}</code>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.create.validation.configKey')">
                <code>{{ managedRoot?.config_key || '-' }}</code>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.create.validation.createPermission')">
                <code>{{ managedRoot?.create_permission || '-' }}</code>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.create.validation.workingDirectory')">
                <code>{{ validationResult?.working_directory || '-' }}</code>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.create.validation.composePath')">
                <code>{{ validationResult?.compose_file_absolute_path || '-' }}</code>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.create.validation.envPath')">
                <code>{{ validationResult?.env_file_absolute_path || '-' }}</code>
              </t-descriptions-item>
            </t-descriptions>

            <div class="project-create-warnings">
              <t-alert
                v-for="(warning, index) in validationWarnings"
                :key="`${index}-${warning}`"
                theme="warning"
                :message="warning"
              />
              <t-empty v-if="!validationWarnings.length" :description="t('project.create.validation.noWarnings')" />
            </div>
          </t-card>
        </section>

        <section class="project-create-editors">
          <t-card :bordered="true" :title="t('project.create.editors.title')">
            <t-tabs v-model:value="activeEditorTab">
              <t-tab-panel value="compose" :label="t('project.create.editors.composeTab')">
                <project-file-editor
                  v-model="formData.compose_file_content"
                  v-model:mode="composeEditorMode"
                  :title="t('project.create.editors.composeTitle')"
                  :description="t('project.create.editors.composeDescription')"
                  :placeholder="t('project.create.editors.composePlaceholder')"
                  :empty-label="t('project.create.editors.composeEmpty')"
                  :edit-label="t('project.create.editors.backToEditor')"
                  :preview-label="t('project.create.editors.preview')"
                  :format-label="t('project.create.editors.format')"
                  :fullscreen-label="t('project.create.editors.fullscreen')"
                  :exit-fullscreen-label="t('project.create.editors.exitFullscreen')"
                  :resize-handle-label="t('project.create.editors.resize')"
                  storage-key="graft.project.create.compose.editor.height"
                  @format="formatComposeContent"
                />
              </t-tab-panel>
              <t-tab-panel value="env" :label="t('project.create.editors.envTab')">
                <project-file-editor
                  v-model="envEditorContent"
                  v-model:mode="envEditorMode"
                  :title="t('project.create.editors.envTitle')"
                  :description="t('project.create.editors.envDescription')"
                  :placeholder="t('project.create.editors.envPlaceholder')"
                  :empty-label="t('project.create.editors.envEmpty')"
                  :edit-label="t('project.create.editors.backToEditor')"
                  :preview-label="t('project.create.editors.preview')"
                  :format-label="t('project.create.editors.format')"
                  :fullscreen-label="t('project.create.editors.fullscreen')"
                  :exit-fullscreen-label="t('project.create.editors.exitFullscreen')"
                  :resize-handle-label="t('project.create.editors.resize')"
                  storage-key="graft.project.create.env.editor.height"
                  @format="formatEnvContent"
                />
              </t-tab-panel>
            </t-tabs>
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

import { getProjectManagedRoot, postProjectCreate, postProjectCreateValidate } from '../../api/project';
import ProjectFileEditor from '../../components/ProjectFileEditor.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../../shared/navigation';
import { useProjectPageContext } from '../../shared/page-context';
import type {
  ProjectCreateRequest,
  ProjectCreateResponse,
  ProjectCreateValidateRequest,
  ProjectCreateValidateResponse,
  ProjectManagedRootResponse,
} from '../../types/project';

defineOptions({
  name: 'ProjectManagedCreateIndex',
});

type EditorTab = 'compose' | 'env';
type EditorMode = 'edit' | 'preview';

const { router, tabsRouterStore, t } = useProjectPageContext();

const formRef = ref<FormInstanceFunctions | null>(null);
const rootLoading = ref(false);
const validating = ref(false);
const creating = ref(false);
const managedRoot = ref<ProjectManagedRootResponse | null>(null);
const validationResult = ref<ProjectCreateValidateResponse | null>(null);
const activeEditorTab = ref<EditorTab>('compose');
const composeEditorMode = ref<EditorMode>('edit');
const envEditorMode = ref<EditorMode>('edit');

const formData = reactive<ProjectCreateRequest>({
  display_name: '',
  canonical_project_name: '',
  relative_project_directory: '',
  compose_file_name: 'compose.yaml',
  compose_file_content: defaultComposeContent(),
  env_file_name: '.env',
  env_file_content: '',
});

const formRules: FormProps['rules'] = {
  display_name: [{ required: true, message: t('project.create.validation.displayNameRequired') }],
  canonical_project_name: [
    { required: true, message: t('project.create.validation.canonicalProjectNameRequired') },
    {
      validator: (value) => /^[a-z0-9][a-z0-9_.-]*$/i.test(String(value ?? '')),
      message: t('project.create.validation.canonicalProjectNamePattern'),
    },
  ],
  relative_project_directory: [
    { required: true, message: t('project.create.validation.relativeDirectoryRequired') },
    {
      validator: (value) => {
        const normalized = String(value ?? '').trim();
        return Boolean(normalized) && !normalized.startsWith('/') && !normalized.includes('..');
      },
      message: t('project.create.validation.relativeDirectoryPattern'),
    },
  ],
  compose_file_name: [{ required: true, message: t('project.create.validation.composeFileNameRequired') }],
  env_file_name: [
    {
      validator: (value) => {
        const normalized = String(value ?? '').trim();
        return !normalized || (!normalized.includes('/') && !normalized.includes('\\'));
      },
      message: t('project.create.validation.envFileNamePattern'),
    },
  ],
};

const managedCreateEnabled = computed(
  () => managedRoot.value?.supports_managed_create && managedRoot.value.status === 'ready',
);
const managedRootStatusLabel = computed(() => {
  const status = managedRoot.value?.status;
  if (status === 'ready') return t('project.create.root.status.ready');
  if (status === 'invalid') return t('project.create.root.status.invalid');
  if (status === 'unconfigured') return t('project.create.root.status.unconfigured');
  return t('project.create.root.status.unknown');
});
const managedRootStatusTheme = computed(() => {
  const status = managedRoot.value?.status;
  if (status === 'ready') return 'success';
  if (status === 'invalid') return 'danger';
  return 'warning';
});
const managedRootNotice = computed(() => {
  if (!managedRoot.value) {
    return '';
  }
  if (managedRoot.value.status === 'ready') {
    return t('project.create.root.readyHint', {
      directory: managedRoot.value.configured_root_directory || '-',
    });
  }
  return managedRoot.value.status_reason || t('project.create.root.unavailableHint');
});
const managedRootNoticeTheme = computed(() => (managedRoot.value?.status === 'invalid' ? 'error' : 'warning'));
const validationWarnings = computed(() => validationResult.value?.warnings || []);
const envFileNameModel = computed({
  get: () => String(formData.env_file_name || ''),
  set: (value: string) => {
    formData.env_file_name = value;
  },
});
const envEditorContent = computed({
  get: () => formData.env_file_content || '',
  set: (value: string) => {
    formData.env_file_content = value;
    if (!value.trim() && !String(formData.env_file_name || '').trim()) {
      formData.env_file_content = '';
    }
  },
});

void loadManagedRoot();

async function loadManagedRoot() {
  rootLoading.value = true;
  try {
    managedRoot.value = await getProjectManagedRoot();
  } catch (error) {
    managedRoot.value = null;
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.create.messages.rootLoadFailed')));
  } finally {
    rootLoading.value = false;
  }
}

async function runValidate() {
  const validateResult = await formRef.value?.validate();
  if (validateResult !== true) {
    return false;
  }

  validating.value = true;
  try {
    validationResult.value = await postProjectCreateValidate(toValidateRequest());
    MessagePlugin.success(t('project.create.messages.validateSuccess'));
    return true;
  } catch (error) {
    validationResult.value = null;
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.create.messages.validateFailed')));
    return false;
  } finally {
    validating.value = false;
  }
}

async function handleSubmit() {
  if (!managedCreateEnabled.value) {
    MessagePlugin.warning(managedRootNotice.value || t('project.create.root.unavailableHint'));
    return;
  }

  const valid = await runValidate();
  if (!valid) {
    return;
  }

  creating.value = true;
  try {
    const response = await postProjectCreate(toCreateRequest());
    MessagePlugin.success(response.message || t('project.create.messages.createSuccess'));
    openCreatedProject(response);
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.create.messages.createFailed')));
  } finally {
    creating.value = false;
  }
}

function resetForm() {
  formData.display_name = '';
  formData.canonical_project_name = '';
  formData.relative_project_directory = '';
  formData.compose_file_name = 'compose.yaml';
  formData.compose_file_content = defaultComposeContent();
  formData.env_file_name = '.env';
  formData.env_file_content = '';
  validationResult.value = null;
  activeEditorTab.value = 'compose';
  composeEditorMode.value = 'edit';
  envEditorMode.value = 'edit';
  formRef.value?.reset({ type: 'initial' });
}

function goToList() {
  void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.LIST.routeName });
}

function toValidateRequest(): ProjectCreateValidateRequest {
  return {
    display_name: formData.display_name.trim(),
    canonical_project_name: formData.canonical_project_name.trim(),
    relative_project_directory: formData.relative_project_directory.trim(),
    compose_file_name: formData.compose_file_name.trim(),
    ...(normalizedEnvFileName.value ? { env_file_name: normalizedEnvFileName.value } : {}),
  };
}

function toCreateRequest(): ProjectCreateRequest {
  return {
    ...toValidateRequest(),
    compose_file_content: normalizeTextBlock(formData.compose_file_content),
    ...(normalizedEnvFileName.value
      ? {
          env_file_name: normalizedEnvFileName.value,
          env_file_content: normalizeTextBlock(formData.env_file_content || ''),
        }
      : {}),
  };
}

const normalizedEnvFileName = computed(() => {
  const normalized = String(formData.env_file_name || '').trim();
  return normalized || null;
});

function formatComposeContent() {
  formData.compose_file_content = normalizeTextBlock(formData.compose_file_content);
}

function formatEnvContent() {
  formData.env_file_content = normalizeTextBlock(formData.env_file_content || '');
}

function openCreatedProject(response: ProjectCreateResponse) {
  const target = {
    name: PROJECT_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    params: { id: response.project_id },
    query: { tab: 'configuration', name: response.display_name },
  };
  const resolved = router.resolve(target);
  appendResolvedTab(
    tabsRouterStore,
    resolved,
    buildDetailTitleWithFallback('project.route.detail.title', response.display_name),
  );
  void router.push(target);
}

function defaultComposeContent() {
  return ['services:', '  app:', '    image: nginx:alpine', '    ports:', "      - '8080:80'"].join('\n');
}

function normalizeTextBlock(value: string) {
  const normalized = String(value ?? '')
    .replace(/\r\n/g, '\n')
    .split('\n')
    .map((line) => line.replace(/\s+$/g, ''))
    .join('\n')
    .trim();
  return normalized ? `${normalized}\n` : '';
}
</script>
<style scoped lang="less">
.project-create-page,
.project-create-layout,
.project-create-main,
.project-create-editors,
.project-create-form-actions,
.project-create-form-grid,
.project-create-warnings {
  display: flex;
}

.project-create-page {
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.project-create-page__notice {
  margin-top: calc(var(--graft-density-gap-12) * -1);
}

.project-create-layout {
  align-items: flex-start;
  gap: var(--graft-density-gap-16);
}

.project-create-main,
.project-create-editors,
.project-create-warnings {
  flex: 1 1 0;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.project-create-main {
  max-width: 520px;
}

.project-create-form-grid {
  display: grid;
  gap: var(--graft-density-gap-12) var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-create-form-grid :deep(.t-form__item:last-child) {
  grid-column: 1 / -1;
}

.project-create-form-actions {
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  margin-top: var(--graft-density-gap-8);
}

.project-create-warnings {
  min-height: 140px;
}

@media (width <= 1200px) {
  .project-create-layout {
    flex-direction: column;
  }

  .project-create-main {
    max-width: none;
    width: 100%;
  }
}

@media (width <= 768px) {
  .project-create-form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
