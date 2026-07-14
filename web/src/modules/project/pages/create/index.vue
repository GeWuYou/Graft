<template>
  <div class="project-create-page" data-page-type="workflow">
    <management-page-content>
      <management-page-header
        title-key="project.route.createBlank.title"
        description-key="project.create.wizardDescription"
      >
        <template #actions
          ><t-space size="small"
            ><t-button variant="outline" data-testid="project-create-back-source" @click="goToSource">{{
              t('project.create.actions.backToSource')
            }}</t-button
            ><t-button @click="refreshPage">{{ t('project.create.actions.refresh') }}</t-button></t-space
          ></template
        >
      </management-page-header>
      <t-steps :current="step" readonly :options="stepOptions" />
      <t-card bordered class="project-create-page__card">
        <section v-if="step === 0">
          <t-form ref="formRef" :data="formData" :rules="formRules" label-align="top" @submit="nextFromIdentity"
            ><div class="project-create-page__form-grid">
              <t-form-item :label="t('project.create.form.displayName')" name="display_name"
                ><t-input
                  v-model="formData.display_name"
                  :placeholder="t('project.create.form.displayNamePlaceholder')" /></t-form-item
              ><t-form-item :label="t('project.create.form.workspaceKey')" name="workspace_key"
                ><t-input
                  v-model="formData.workspace_key"
                  :placeholder="t('project.create.form.workspaceKeyPlaceholder')"
              /></t-form-item>
            </div>
            <div class="project-create-page__actions">
              <t-button theme="primary" type="submit">{{ t('project.create.actions.continue') }}</t-button>
            </div></t-form
          >
        </section>
        <section v-else-if="step === 1">
          <project-create-workspace-editor v-model:files="workspaceFiles" />
          <div class="project-create-page__actions">
            <t-button variant="outline" @click="step--">{{ t('project.create.actions.back') }}</t-button
            ><t-button theme="primary" @click="step++">{{ t('project.create.actions.review') }}</t-button>
          </div>
        </section>
        <section v-else>
          <h2>{{ t('project.create.review.title') }}</h2>
          <p>{{ t('project.create.review.noAutoDeploy') }}</p>
          <t-descriptions bordered size="small" :column="1"
            ><t-descriptions-item :label="t('project.create.form.displayName')">{{
              formData.display_name
            }}</t-descriptions-item
            ><t-descriptions-item :label="t('project.create.form.workspaceKey')"
              ><code>{{ formData.workspace_key || '-' }}</code></t-descriptions-item
            ></t-descriptions
          >
          <div class="project-create-page__actions">
            <t-button variant="outline" @click="step--">{{ t('project.create.actions.back') }}</t-button
            ><t-button theme="primary" :loading="creating" @click="createProject">{{
              t('project.create.actions.create')
            }}</t-button>
          </div>
        </section>
      </t-card>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import type { FormInstanceFunctions, FormProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { postProjectCreate } from '../../api/project';
import ProjectCreateWorkspaceEditor from '../../components/ProjectCreateWorkspaceEditor.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import {
  appendResolvedTab,
  buildDetailTitleWithFallback,
  navigateToProjectCreateSource,
  refreshProjectCreatePage,
} from '../../shared/navigation';
import { useProjectPageContext } from '../../shared/page-context';
import type { ProjectCreateRequest, ProjectCreateResponse, ProjectWorkspaceManifestFile } from '../../types/project';
defineOptions({ name: 'ProjectManagedCreateIndex' });
const { router, tabsRouterStore, t } = useProjectPageContext();
const route = useRoute();
const formRef = ref<FormInstanceFunctions | null>(null);
const creating = ref(false);
const step = ref(0);
const runtimeTargetId = computed(() => {
  const raw = route.query.runtime_target_id;
  if (typeof raw !== 'string' || !/^[1-9]\d*$/.test(raw)) return null;
  const value = Number(raw);
  return Number.isSafeInteger(value) ? value : null;
});
const formData = reactive({ display_name: '', workspace_key: '' });
const workspaceFiles = ref<ProjectWorkspaceManifestFile[]>([
  { path: 'compose.yaml', content: t('project.create.workspace.defaultCompose.unit') },
]);
const formRules: FormProps['rules'] = {
  display_name: [{ required: true, message: t('project.create.validation.displayNameRequired') }],
  workspace_key: [
    {
      validator: (value) => !value || /^[a-z0-9][a-z0-9-]*$/.test(String(value)),
      message: t('project.create.validation.workspaceKeyPattern'),
    },
  ],
};
const stepOptions = computed(() =>
  ['identity', 'workspace', 'review'].map((key) => ({ title: t(`project.create.steps.${key}`) })),
);
const composePath = computed(
  () => workspaceFiles.value.find((file) => /(^|\/)(compose|docker-compose)(\..+)?\.ya?ml$/i.test(file.path))?.path,
);
async function nextFromIdentity() {
  if ((await formRef.value?.validate()) !== true) return;
  if (runtimeTargetId.value === null) {
    MessagePlugin.warning(t('project.runtimeTarget.unavailableTooltip'));
    return;
  }
  step.value++;
}
function payload(runtimeTargetIdValue: number): ProjectCreateRequest {
  const compose = workspaceFiles.value.find((file) => file.path === composePath.value);
  return {
    display_name: formData.display_name.trim(),
    runtime_target_id: runtimeTargetIdValue,
    ...(formData.workspace_key.trim() ? { workspace_key: formData.workspace_key.trim() } : {}),
    compose_file_name: composePath.value as string,
    compose_file_content: compose?.content || '',
    workspace_files: workspaceFiles.value,
    compose_file_path: composePath.value,
  };
}
async function createProject() {
  if (runtimeTargetId.value === null) {
    MessagePlugin.warning(t('project.runtimeTarget.unavailableTooltip'));
    return;
  }
  if (!composePath.value) {
    MessagePlugin.warning(t('project.create.validation.composeFileNameRequired'));
    return;
  }
  creating.value = true;
  try {
    const response = await postProjectCreate(payload(runtimeTargetId.value));
    MessagePlugin.success(t('project.create.messages.createSuccess'));
    openCreatedProject(response);
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.create.messages.createFailed')));
  } finally {
    creating.value = false;
  }
}
function goToSource() {
  navigateToProjectCreateSource(router, route.query);
}
function refreshPage() {
  refreshProjectCreatePage(router, PROJECT_BOOTSTRAP_ROUTE.CREATE_BLANK.pageRouteName, route.query);
}
function openCreatedProject(response: ProjectCreateResponse) {
  const target = {
    name: PROJECT_BOOTSTRAP_ROUTE.CONFIGURATION_WORKSPACE.pageRouteName,
    params: { id: response.application_id },
    query: { name: response.display_name },
  };
  const resolved = router.resolve(target);
  appendResolvedTab(
    tabsRouterStore,
    resolved,
    buildDetailTitleWithFallback('project.route.configurationWorkspace.title', response.display_name),
  );
  void router.push(target);
}
</script>
<style scoped>
.project-create-page {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.project-create-page__card {
  margin-top: var(--graft-density-gap-16);
}

.project-create-page__form-grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-create-page__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  margin-top: var(--graft-density-gap-16);
}

h2 {
  margin-top: 0;
}

@media (width <= 768px) {
  .project-create-page__form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
