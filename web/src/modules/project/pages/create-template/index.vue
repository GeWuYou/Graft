<template>
  <div class="application-template-create" data-page-type="workflow">
    <management-page-content>
      <management-page-header
        title-key="project.route.templateCreate.title"
        description-key="project.templates.createDescription"
        :source="{ labelKey: 'project.templates.eyebrow', fallback: t('project.templates.eyebrow') }"
      />

      <t-steps :current="step" readonly :options="stepOptions" />

      <t-card bordered class="application-template-create__surface">
        <section v-if="step === 0" class="application-template-create__step">
          <t-form ref="formRef" :data="formData" :rules="formRules" label-align="top" @submit="nextFromTemplateInfo">
            <div class="application-template-create__form-grid">
              <t-form-item :label="t('project.templates.name')" name="display_name">
                <t-input v-model="formData.display_name" :placeholder="t('project.templates.namePlaceholder')" />
              </t-form-item>
              <t-form-item :label="t('project.templates.adapter')" name="deployment_adapter_kind">
                <t-select
                  v-model="formData.deployment_adapter_kind"
                  filterable
                  :options="adapterOptions"
                  :placeholder="t('project.templates.adapterPlaceholder')"
                  :scroll="{ type: 'virtual', rowHeight: 32, threshold: 8 }"
                />
              </t-form-item>
              <t-form-item :label="t('project.templateCatalog.category')" name="category">
                <t-select v-model="formData.category" :options="categoryOptions" />
              </t-form-item>
            </div>
            <div class="application-template-create__actions">
              <t-button variant="outline" @click="cancelVisible = true">{{ t('project.templates.cancel') }}</t-button>
              <t-button theme="primary" type="submit">{{ t('project.templates.next') }}</t-button>
            </div>
          </t-form>
        </section>

        <section v-else-if="step === 1" class="application-template-create__step">
          <project-create-workspace-editor v-model:files="workspaceFiles" />
          <div class="application-template-create__actions">
            <t-button variant="outline" @click="step--">{{ t('project.templates.backStep') }}</t-button>
            <t-button theme="primary" @click="nextFromWorkspace">{{ t('project.templates.next') }}</t-button>
          </div>
        </section>

        <section v-else class="application-template-create__step">
          <project-lifecycle-configuration-review
            v-model:draft="lifecycleDraft"
            :title="t('project.templates.lifecycleTitle')"
            :description="t('project.templates.lifecycleDescription')"
            :authority-message="t('project.templates.lifecycleAuthority')"
            :configuration-title="t('project.create.lifecycle.configurationTitle')"
            :command-preview-title="t('project.create.lifecycle.commandPreviewTitle')"
          />
          <div class="application-template-create__actions">
            <t-button variant="outline" :disabled="creating" @click="step--">{{
              t('project.templates.backStep')
            }}</t-button>
            <t-button theme="primary" :loading="creating" @click="createTemplate">{{
              t('project.templates.createComplete')
            }}</t-button>
          </div>
        </section>
      </t-card>
    </management-page-content>

    <t-dialog
      v-model:visible="cancelVisible"
      theme="warning"
      :header="t('project.templates.cancelCreateTitle')"
      :body="t('project.templates.cancelCreateBody')"
      :confirm-btn="t('project.templates.cancelCreateConfirm')"
      :cancel-btn="t('project.templates.continueCreate')"
      @confirm="leaveWizard"
    />
  </div>
</template>
<script setup lang="ts">
import type { FormInstanceFunctions, FormProps, SelectProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, reactive, ref } from 'vue';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { postApplicationTemplate } from '../../api/project';
import ProjectCreateWorkspaceEditor from '../../components/ProjectCreateWorkspaceEditor.vue';
import ProjectLifecycleConfigurationReview from '../../components/ProjectLifecycleConfigurationReview.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { APPLICATION_TEMPLATE_CATEGORIES } from '../../contract/categories';
import { createBlankComposeWorkspaceFiles } from '../../shared/blank-compose-workspace';
import { buildBlankLifecycleConfigurationDraft, buildLifecycleConfigurationRequest } from '../../shared/lifecycle';
import { useApplicationPageContext } from '../../shared/page-context';
import { emitApplicationTemplateDebug } from '../../shared/project-template-debug';
import type {
  ApplicationLifecycleConfigurationDraft,
  ApplicationTemplateCategory,
  ApplicationTemplateDraftRequest,
  ApplicationWorkspaceDraftEntry,
  ApplicationWorkspaceDraftFile,
} from '../../types/project';

defineOptions({ name: 'ApplicationTemplateCreateWizardIndex' });

// 创建向导只在浏览器内持有未完成定义，最后一步才写入模板草稿，避免中途退出留下无效记录。
const { router, t } = useApplicationPageContext();
const formRef = ref<FormInstanceFunctions | null>(null);
const step = ref(0);
const creating = ref(false);
const cancelVisible = ref(false);
const formData = reactive<{
  display_name: string;
  category: ApplicationTemplateCategory;
  deployment_adapter_kind: 'compose';
}>({
  display_name: '',
  category: 'other',
  deployment_adapter_kind: 'compose',
});
const workspaceFiles = ref<ApplicationWorkspaceDraftEntry[]>(createBlankComposeWorkspaceFiles());
const lifecycleDraft = ref<ApplicationLifecycleConfigurationDraft>(
  buildBlankLifecycleConfigurationDraft(
    { lifecycle_configuration: {} },
    { composeFilePath: 'compose.yaml', composeProjectName: 'template' },
  ),
);

const formRules: FormProps['rules'] = {
  display_name: [{ required: true, message: t('project.templates.nameRequired') }],
  deployment_adapter_kind: [{ required: true, message: t('project.templates.adapterRequired') }],
};
const adapterOptions = computed<SelectProps['options']>(() => [
  { label: t('project.templates.adapterCompose'), value: 'compose' },
]);
const categoryOptions = computed<SelectProps['options']>(() =>
  APPLICATION_TEMPLATE_CATEGORIES.map((value) => ({
    value,
    label: t(`project.templateCatalog.categories.${value}`),
  })),
);
const stepOptions = computed(() =>
  ['info', 'workspace', 'lifecycle'].map((key) => ({ title: t(`project.templates.steps.${key}`) })),
);

async function nextFromTemplateInfo() {
  if ((await formRef.value?.validate()) !== true) return;
  step.value = 1;
}

function nextFromWorkspace() {
  const composeFile = workspaceFiles.value.find(
    (entry): entry is ApplicationWorkspaceDraftFile => entry.node_type !== 'directory' && entry.path === 'compose.yaml',
  );
  if (!composeFile?.content.trim()) {
    MessagePlugin.warning(t('project.templates.composeFileRequired'));
    return;
  }
  lifecycleDraft.value.compose_files = ['compose.yaml'];
  lifecycleDraft.value.compose_project_name = 'template';
  step.value = 2;
}

function payload(): ApplicationTemplateDraftRequest {
  return {
    display_name: formData.display_name.trim(),
    description: '',
    category: formData.category,
    deployment_adapter_kind: formData.deployment_adapter_kind as 'compose',
    definition_schema_version: 1,
    definition: {
      compose_file_path: 'compose.yaml',
      workspace_entries: workspaceFiles.value.map((entry) =>
        entry.node_type === 'directory'
          ? { path: entry.path, node_type: 'directory' }
          : { path: entry.path, node_type: 'file', content: entry.content },
      ),
      lifecycle_configuration: buildLifecycleConfigurationRequest(lifecycleDraft.value),
    },
  } satisfies ApplicationTemplateDraftRequest;
}

async function createTemplate() {
  creating.value = true;
  try {
    emitApplicationTemplateDebug('create-requested', {
      routeName: PROJECT_BOOTSTRAP_ROUTE.TEMPLATE_CREATE.pageRouteName,
    });
    const template = await postApplicationTemplate(payload());
    emitApplicationTemplateDebug('create-succeeded', { templateId: template.template_id });
    MessagePlugin.success(t('project.templates.createSuccess'));
    await router.push({
      name: PROJECT_BOOTSTRAP_ROUTE.TEMPLATE_DETAIL.pageRouteName,
      params: { templateId: template.template_id },
    });
  } catch (error) {
    emitApplicationTemplateDebug('create-failed', { errorName: error instanceof Error ? error.name : typeof error });
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.createFailed')));
  } finally {
    creating.value = false;
  }
}

async function leaveWizard() {
  cancelVisible.value = false;
  await router.push({ name: PROJECT_BOOTSTRAP_ROUTE.TEMPLATES.pageRouteName });
}
</script>
<style scoped>
.application-template-create {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.application-template-create__surface {
  margin-top: var(--graft-density-gap-16);
}

.application-template-create__step {
  min-width: 0;
}

.application-template-create__form-grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.application-template-create__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  margin-top: var(--graft-density-gap-16);
}

@media (width <= 720px) {
  .application-template-create__form-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
