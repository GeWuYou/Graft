<template>
  <div class="project-create-page" data-page-type="editor">
    <management-page-content>
      <management-page-header
        title-key="project.route.createManaged.title"
        description-key="project.create.wizardDescription"
      >
        <template #meta
          ><t-tag :theme="managedRootStatusTheme" variant="light-outline">{{ managedRootStatusLabel }}</t-tag></template
        >
        <template #actions
          ><t-space size="small"
            ><t-button theme="default" variant="outline" @click="goToList">{{
              t('project.create.actions.backToList')
            }}</t-button
            ><t-button theme="default" :loading="rootLoading" @click="loadManagedRoot">{{
              t('project.create.actions.refreshAuthority')
            }}</t-button></t-space
          ></template
        >
      </management-page-header>
      <t-alert v-if="managedRootNotice" :theme="managedRootNoticeTheme" :message="managedRootNotice" />
      <t-steps :current="step" readonly :options="stepOptions" />
      <t-card bordered class="project-create-page__card">
        <section v-if="step === 0">
          <t-form ref="formRef" :data="formData" :rules="formRules" label-align="top" @submit="nextFromIdentity">
            <div class="project-create-page__form-grid">
              <t-form-item :label="t('project.create.form.displayName')" name="display_name"
                ><t-input
                  v-model="formData.display_name"
                  :placeholder="t('project.create.form.displayNamePlaceholder')"
              /></t-form-item>
              <t-form-item :label="t('project.create.form.canonicalProjectName')" name="canonical_project_name"
                ><t-input
                  v-model="formData.canonical_project_name"
                  :placeholder="t('project.create.form.canonicalProjectNamePlaceholder')"
                  @change="syncDefaultDirectory"
              /></t-form-item>
              <t-form-item :label="t('project.create.form.relativeProjectDirectory')" name="relative_project_directory"
                ><t-input
                  v-model="formData.relative_project_directory"
                  :placeholder="t('project.create.form.relativeProjectDirectoryPlaceholder')"
              /></t-form-item>
            </div>
            <t-descriptions bordered size="small" :column="1"
              ><t-descriptions-item :label="t('project.create.validation.rootDirectory')"
                ><code>{{ managedRoot?.configured_root_directory || '-' }}</code></t-descriptions-item
              ><t-descriptions-item :label="t('project.create.validation.createPermission')"
                ><code>{{ managedRoot?.create_permission || '-' }}</code></t-descriptions-item
              ></t-descriptions
            >
            <div class="project-create-page__actions">
              <t-button theme="primary" type="submit" :disabled="!managedCreateEnabled">{{
                t('project.create.actions.continue')
              }}</t-button>
            </div>
          </t-form>
        </section>
        <section v-else-if="step === 1">
          <project-create-workspace-editor v-model:files="workspaceFiles" />
          <div class="project-create-page__actions">
            <t-button theme="default" variant="outline" @click="step--">{{ t('project.create.actions.back') }}</t-button
            ><t-button theme="primary" @click="step++">{{ t('project.create.actions.continue') }}</t-button>
          </div>
        </section>
        <section v-else-if="step === 2">
          <project-lifecycle-configuration-review
            v-model:draft="lifecycleDraft"
            :title="t('project.create.lifecycle.title')"
            :description="t('project.create.lifecycle.description')"
            :authority-message="t('project.create.lifecycle.authorityHint')"
            :configuration-title="t('project.create.lifecycle.configurationTitle')"
            :command-preview-title="t('project.create.lifecycle.commandPreviewTitle')"
          />
          <div class="project-create-page__actions">
            <t-button theme="default" variant="outline" @click="step--">{{ t('project.create.actions.back') }}</t-button
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
            ><t-descriptions-item :label="t('project.create.form.canonicalProjectName')"
              ><code>{{ formData.canonical_project_name }}</code></t-descriptions-item
            ><t-descriptions-item :label="t('project.create.review.workingDirectory')"
              ><code>{{ validationResult?.working_directory || resolvedDirectory }}</code></t-descriptions-item
            ><t-descriptions-item :label="t('project.create.review.workspaceFiles')">{{
              workspaceFiles.map((file) => file.path).join(', ')
            }}</t-descriptions-item></t-descriptions
          >
          <t-alert v-for="warning in validationWarnings" :key="warning" theme="warning" :message="warning" />
          <t-checkbox v-model="deployAfterCreate" :disabled="!canDeployAfterCreate">
            {{ t('project.create.review.deployAfterCreate') }}
          </t-checkbox>
          <t-alert
            v-if="!canDeployAfterCreate"
            theme="info"
            :message="t('project.create.review.deployPermissionRequired')"
          />
          <div class="project-create-page__actions">
            <t-button theme="default" variant="outline" @click="step--">{{ t('project.create.actions.back') }}</t-button
            ><t-button theme="primary" :loading="creating" :disabled="!managedCreateEnabled" @click="createProject">{{
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

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { usePermissionStore } from '@/store';

import {
  getProjectManagedRoot,
  postProjectCreate,
  postProjectCreateValidate,
  postProjectDeploy,
} from '../../api/project';
import ProjectCreateWorkspaceEditor from '../../components/ProjectCreateWorkspaceEditor.vue';
import ProjectLifecycleConfigurationReview from '../../components/ProjectLifecycleConfigurationReview.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { PROJECT_PERMISSION_CODE } from '../../contract/permissions';
import { isValidProjectCanonicalName } from '../../shared/canonical-name';
import { createWithOptionalDeploy } from '../../shared/create-with-optional-deploy';
import { buildLifecycleConfigurationRequest } from '../../shared/lifecycle';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../../shared/navigation';
import { useProjectPageContext } from '../../shared/page-context';
import type {
  ProjectCreateRequest,
  ProjectCreateResponse,
  ProjectCreateValidateRequest,
  ProjectCreateValidateResponse,
  ProjectLifecycleConfigurationDraft,
  ProjectManagedRootResponse,
  ProjectWorkspaceManifestFile,
} from '../../types/project';
defineOptions({ name: 'ProjectManagedCreateIndex' });
const { router, tabsRouterStore, t } = useProjectPageContext();
const permissionStore = usePermissionStore();
const formRef = ref<FormInstanceFunctions | null>(null);
const rootLoading = ref(false);
const creating = ref(false);
const step = ref(0);
const deployAfterCreate = ref(false);
const managedRoot = ref<ProjectManagedRootResponse | null>(null);
const validationResult = ref<ProjectCreateValidateResponse | null>(null);
const formData = reactive({ display_name: '', canonical_project_name: '', relative_project_directory: '' });
const workspaceFiles = ref<ProjectWorkspaceManifestFile[]>([
  { path: 'compose.yaml', content: defaultComposeContent() },
  { path: '.env', content: '' },
]);
const lifecycleDraft = reactive<ProjectLifecycleConfigurationDraft>({
  strategy_kind: 'standard',
  working_directory: '',
  compose_files: ['compose.yaml'],
  canonical_project_name: '',
  profiles: [],
  down_before_redeploy: true,
  pull_before_redeploy: false,
  build_before_up: false,
  force_recreate: false,
  remove_orphans: true,
  wait_after_up: false,
  wait_timeout_seconds: 120,
  renew_anon_volumes: false,
  prune_images_after_redeploy: false,
  additional_args: '',
  review_status: 'confirmed',
  generated_commands: null,
});
const formRules: FormProps['rules'] = {
  display_name: [{ required: true, message: t('project.create.validation.displayNameRequired') }],
  canonical_project_name: [
    { required: true, message: t('project.create.validation.canonicalProjectNameRequired') },
    {
      validator: (value) => isValidProjectCanonicalName(String(value ?? '')),
      message: t('project.create.validation.canonicalProjectNamePattern'),
    },
  ],
  relative_project_directory: [
    { required: true, message: t('project.create.validation.relativeDirectoryRequired') },
    {
      validator: (value) => {
        const path = String(value ?? '').trim();
        return (
          Boolean(path) &&
          !path.startsWith('/') &&
          !path.split('/').some((part) => !part || part === '.' || part === '..')
        );
      },
      message: t('project.create.validation.relativeDirectoryPattern'),
    },
  ],
};
const stepOptions = computed(() =>
  ['identity', 'workspace', 'lifecycle', 'review'].map((key) => ({ title: t(`project.create.steps.${key}`) })),
);
const managedCreateEnabled = computed(
  () => managedRoot.value?.supports_managed_create && managedRoot.value.status === 'ready',
);
const canDeployAfterCreate = computed(() => permissionStore.hasPermission(PROJECT_PERMISSION_CODE.DEPLOY));
const validationWarnings = computed(() => validationResult.value?.warnings || []);
const resolvedDirectory = computed(
  () => `${managedRoot.value?.configured_root_directory || '-'}/${formData.relative_project_directory}`,
);
const managedRootStatusLabel = computed(() =>
  t(`project.create.root.status.${managedRoot.value?.status || 'unknown'}`),
);
const managedRootStatusTheme = computed(() =>
  managedRoot.value?.status === 'ready' ? 'success' : managedRoot.value?.status === 'invalid' ? 'danger' : 'warning',
);
const managedRootNotice = computed(() =>
  managedRoot.value?.status === 'ready'
    ? t('project.create.root.readyHint', { directory: managedRoot.value.configured_root_directory || '-' })
    : managedRoot.value?.status_reason || t('project.create.root.unavailableHint'),
);
const managedRootNoticeTheme = computed(() => (managedRoot.value?.status === 'invalid' ? 'error' : 'warning'));
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
function syncDefaultDirectory() {
  if (
    !formData.relative_project_directory ||
    formData.relative_project_directory === lifecycleDraft.canonical_project_name
  )
    formData.relative_project_directory = formData.canonical_project_name;
  lifecycleDraft.canonical_project_name = formData.canonical_project_name;
}
async function nextFromIdentity() {
  const valid = await formRef.value?.validate();
  if (valid !== true) return;
  lifecycleDraft.canonical_project_name = formData.canonical_project_name.trim();
  lifecycleDraft.working_directory = resolvedDirectory.value;
  lifecycleDraft.compose_files = [composePath.value];
  step.value++;
}
const composePath = computed(
  () =>
    workspaceFiles.value.find((file) => /(^|\/)(compose|docker-compose)(\..+)?\.ya?ml$/i.test(file.path))?.path ||
    'compose.yaml',
);
const envPaths = computed(() =>
  workspaceFiles.value.filter((file) => /^\.env(?:\.|$)/.test(file.path)).map((file) => file.path),
);
function requestBase() {
  const compose = workspaceFiles.value.find((file) => file.path === composePath.value);
  const env = workspaceFiles.value.find((file) => file.path === envPaths.value[0]);
  return {
    display_name: formData.display_name.trim(),
    canonical_project_name: formData.canonical_project_name.trim(),
    relative_project_directory: formData.relative_project_directory.trim(),
    compose_file_name: composePath.value,
    compose_file_content: compose?.content || '',
    ...(env ? { env_file_name: env.path, env_file_content: env.content } : {}),
    workspace_files: workspaceFiles.value,
    compose_file_path: composePath.value,
    ...(envPaths.value.length ? { env_file_paths: envPaths.value } : {}),
    lifecycle_configuration: buildLifecycleConfigurationRequest(lifecycleDraft),
  };
}
async function validateRequest() {
  validationResult.value = await postProjectCreateValidate(requestBase() as ProjectCreateValidateRequest);
}
async function createProject() {
  if (!managedCreateEnabled.value) {
    MessagePlugin.warning(managedRootNotice.value);
    return;
  }
  creating.value = true;
  try {
    await validateRequest();
    const result = await createWithOptionalDeploy({
      create: () => postProjectCreate(requestBase() as ProjectCreateRequest),
      deploy: postProjectDeploy,
      deployAfterCreate: deployAfterCreate.value && canDeployAfterCreate.value,
    });
    const response = result.created;
    MessagePlugin.success(response.message || t('project.create.messages.createSuccess'));
    if (result.deployment.status === 'succeeded') {
      MessagePlugin.success(t('project.create.messages.deploySuccess'));
    }
    if (result.deployment.status === 'failed') {
      MessagePlugin.error(
        resolveLocalizedErrorMessage(t, result.deployment.error, t('project.create.messages.deployFailed')),
      );
    }
    openCreatedProject(response);
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.create.messages.createFailed')));
  } finally {
    creating.value = false;
  }
}
function goToList() {
  void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.LIST.routeName });
}
function openCreatedProject(response: ProjectCreateResponse) {
  const target = {
    name: PROJECT_BOOTSTRAP_ROUTE.CONFIGURATION_WORKSPACE.pageRouteName,
    params: { id: response.project_id },
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
function defaultComposeContent() {
  return ['services:', '  app:', '    image: nginx:alpine'].join('\n');
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

.project-create-page__form-grid :last-child {
  grid-column: 1/-1;
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
