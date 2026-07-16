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
              ><t-form-item :label="t('project.create.form.applicationName')" name="application_name"
                ><t-input
                  v-model="formData.application_name"
                  :placeholder="t('project.create.form.applicationNamePlaceholder')"
              /></t-form-item>
            </div>
            <div class="project-create-page__actions">
              <t-button theme="primary" type="submit">{{ t('project.create.actions.next') }}</t-button>
            </div></t-form
          >
        </section>
        <section v-else-if="step === 1">
          <project-create-workspace-editor v-model:files="workspaceFiles" />
          <div class="project-create-page__actions">
            <t-button variant="outline" @click="step--">{{ t('project.create.actions.back') }}</t-button
            ><t-button theme="primary" @click="nextFromWorkspace">{{ t('project.create.actions.next') }}</t-button>
          </div>
        </section>
        <section v-else-if="step === 2">
          <project-lifecycle-configuration-step
            v-if="lifecycleDraft"
            v-model:draft="lifecycleDraft"
            :title="t('project.create.lifecycle.title')"
            :description="t('project.create.lifecycle.description')"
            :authority-message="t('project.create.lifecycle.authorityHint')"
            :configuration-title="t('project.create.lifecycle.configurationTitle')"
            :command-preview-title="t('project.create.lifecycle.commandPreviewTitle')"
            :back-label="t('project.create.actions.back')"
            :continue-label="t('project.create.actions.next')"
            @back="step--"
            @continue="nextFromLifecycle"
          />
          <t-alert v-else theme="error" :message="lifecycleConfigError" />
        </section>
        <section v-else>
          <h2>{{ t('project.create.review.title') }}</h2>
          <p>{{ t('project.create.review.noAutoDeploy') }}</p>
          <t-descriptions bordered size="small" :column="1"
            ><t-descriptions-item :label="t('project.create.form.displayName')">{{
              formData.display_name
            }}</t-descriptions-item
            ><t-descriptions-item :label="t('project.create.form.applicationName')"
              ><code>{{ formData.application_name || '-' }}</code></t-descriptions-item
            ><t-descriptions-item :label="t('project.create.lifecycle.configurationTitle')">
              {{ lifecycleSummary }}
            </t-descriptions-item></t-descriptions
          >
          <div class="project-create-page__actions">
            <t-button variant="outline" @click="step--">{{ t('project.create.actions.back') }}</t-button
            ><t-button theme="primary" :loading="creating" @click="() => createProject()">{{
              t('project.create.actions.create')
            }}</t-button>
          </div>
        </section>
      </t-card>
      <t-dialog
        v-model:visible="reuseDirectoryDialogVisible"
        :header="t('project.create.reuseDirectory.title')"
        :body="t('project.create.reuseDirectory.body', { path: pendingReusableWorkspace?.workspace_path || '' })"
        theme="warning"
        :cancel-btn="t('project.create.reuseDirectory.cancel')"
        :confirm-btn="t('project.create.reuseDirectory.confirm')"
        @confirm="confirmReuseDirectory"
      />
      <t-dialog
        v-model:visible="reuseWriteDialogVisible"
        :header="t('project.create.reuseWrite.title')"
        :body="t('project.create.reuseWrite.body', { path: reusedWorkspacePath })"
        theme="warning"
        :cancel-btn="t('project.create.reuseWrite.cancel')"
        :confirm-btn="t('project.create.reuseWrite.confirm')"
        :confirm-loading="creating"
        @confirm="confirmCreateReuse"
      />
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import type { FormInstanceFunctions, FormProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import {
  getProjectWorkspaceDefaults,
  postProjectApplicationNameAvailability,
  postProjectCreate,
} from '../../api/project';
import ProjectCreateWorkspaceEditor from '../../components/ProjectCreateWorkspaceEditor.vue';
import ProjectLifecycleConfigurationStep from '../../components/ProjectLifecycleConfigurationStep.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { buildBlankLifecycleConfigurationDraft, buildLifecycleConfigurationRequest } from '../../shared/lifecycle';
import {
  appendResolvedTab,
  buildDetailTitleWithFallback,
  navigateToProjectCreateSource,
  refreshProjectCreatePage,
} from '../../shared/navigation';
import { useProjectPageContext } from '../../shared/page-context';
import type {
  ProjectApplicationNameAvailabilityResponse,
  ProjectCreateRequest,
  ProjectCreateResponse,
  ProjectLifecycleConfigurationDraft,
  ProjectWorkspaceDraftEntry,
  ProjectWorkspaceDraftFile,
  ProjectWorkspaceEntry,
} from '../../types/project';

defineOptions({ name: 'ProjectManagedCreateIndex' });
// 托管创建页拥有表单草稿和校验反馈，工作区编辑器与创建 API 分别维护各自的边界。
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
const formData = reactive({ display_name: '', application_name: '' });
const workspaceFiles = ref<ProjectWorkspaceDraftEntry[]>([
  { path: 'compose.yaml', node_type: 'file', content: '' },
  { path: '.env', node_type: 'file', content: '' },
]);
const primaryComposePath = ref('compose.yaml');
const workspaceDefaultsLoading = ref(true);
const workspaceDefaultsError = ref('');
const workspaceDefaults = ref<Awaited<ReturnType<typeof getProjectWorkspaceDefaults>> | null>(null);
const lifecycleDraft = ref<ProjectLifecycleConfigurationDraft | null>(null);
const lifecycleConfigError = ref('');
const reuseDirectoryDialogVisible = ref(false);
const reuseWriteDialogVisible = ref(false);
const pendingReusableWorkspace = ref<ProjectApplicationNameAvailabilityResponse | null>(null);
const reusingExistingWorkspace = ref(false);
const reusedWorkspacePath = ref('');
const formRules: FormProps['rules'] = {
  display_name: [{ required: true, message: t('project.create.validation.displayNameRequired') }],
  application_name: [
    {
      required: true,
      message: t('project.create.validation.applicationNameRequired'),
    },
    {
      validator: (value) => /^[a-z0-9][a-z0-9-]*$/.test(String(value)),
      message: t('project.create.validation.applicationNamePattern'),
    },
  ],
};
const stepOptions = computed(() =>
  ['identity', 'workspace', 'lifecycle', 'review'].map((key) => ({ title: t(`project.create.steps.${key}`) })),
);
const lifecycleSummary = computed(() => {
  if (!lifecycleDraft.value) return '-';
  return lifecycleDraft.value.additional_args.trim()
    ? t('project.create.lifecycle.configuredWithAdditionalArgs')
    : t('project.create.lifecycle.configured');
});
const composePath = computed(
  () =>
    workspaceFiles.value.find((entry) => entry.node_type === 'file' && entry.path === primaryComposePath.value)?.path ||
    workspaceFiles.value.find(
      (entry) => entry.node_type === 'file' && /(^|\/)(compose|docker-compose)(\..+)?\.ya?ml$/i.test(entry.path),
    )?.path,
);
onMounted(async () => {
  if (typeof route.query.display_name === 'string') formData.display_name = route.query.display_name;
  if (typeof route.query.application_name === 'string') formData.application_name = route.query.application_name;
  try {
    const defaults = await getProjectWorkspaceDefaults();
    if (defaults.workspace_entries?.length) {
      workspaceFiles.value = defaults.workspace_entries.map(toWorkspaceDraftEntry);
    }
    if (defaults.compose_file_path) primaryComposePath.value = defaults.compose_file_path;
    workspaceDefaults.value = defaults;
  } catch (error) {
    workspaceDefaultsError.value = resolveLocalizedErrorMessage(
      t,
      error,
      t('project.create.workspace.defaultsLoadFailed'),
    );
    MessagePlugin.error(workspaceDefaultsError.value);
  } finally {
    workspaceDefaultsLoading.value = false;
  }
});
async function nextFromIdentity() {
  if ((await formRef.value?.validate()) !== true) return;
  if (workspaceDefaultsLoading.value) return;
  if (workspaceDefaultsError.value) {
    MessagePlugin.error(workspaceDefaultsError.value);
    return;
  }
  if (runtimeTargetId.value === null) {
    MessagePlugin.warning(t('project.runtimeTarget.unavailableTooltip'));
    return;
  }
  try {
    const availability = await postProjectApplicationNameAvailability({
      application_name: formData.application_name.trim(),
    });
    if (availability.status === 'registered') {
      MessagePlugin.error(t('project.create.validation.applicationNameRegistered'));
      return;
    }
    if (availability.status === 'reusable_workspace') {
      pendingReusableWorkspace.value = availability;
      if (availability.workspace_non_empty) {
        reuseDirectoryDialogVisible.value = true;
        return;
      }
      useReusableWorkspace(availability);
    } else {
      reusingExistingWorkspace.value = false;
      reusedWorkspacePath.value = '';
    }
    step.value++;
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.create.messages.createFailed')));
  }
}
function payload(runtimeTargetIdValue: number): ProjectCreateRequest {
  if (!lifecycleDraft.value) {
    throw new Error('missing lifecycle configuration');
  }
  return {
    display_name: formData.display_name.trim(),
    runtime_target_id: runtimeTargetIdValue,
    application_name: formData.application_name.trim(),
    workspace_entries: workspaceFiles.value.map(toProjectWorkspaceEntry),
    compose_file_path: composePath.value as string,
    reuse_existing_workspace: reusingExistingWorkspace.value,
    lifecycle_configuration: buildLifecycleConfigurationRequest(lifecycleDraft.value),
  };
}
function useReusableWorkspace(availability: ProjectApplicationNameAvailabilityResponse) {
  reusingExistingWorkspace.value = true;
  reusedWorkspacePath.value = availability.workspace_path;
  if (availability.workspace_entries?.length) {
    workspaceFiles.value = availability.workspace_entries.map(toWorkspaceDraftEntry);
  }
  if (availability.compose_file_path) primaryComposePath.value = availability.compose_file_path;
}
function confirmReuseDirectory() {
  const availability = pendingReusableWorkspace.value;
  if (!availability) return;
  reuseDirectoryDialogVisible.value = false;
  useReusableWorkspace(availability);
  step.value++;
}
function toWorkspaceDraftEntry(entry: ProjectWorkspaceEntry): ProjectWorkspaceDraftEntry {
  if (entry.node_type === 'directory') {
    return { path: entry.path, node_type: 'directory' };
  }
  return { path: entry.path, node_type: 'file', content: entry.content ?? '' };
}
function toProjectWorkspaceEntry(entry: ProjectWorkspaceDraftEntry): ProjectWorkspaceEntry {
  if (entry.node_type === 'directory') {
    return { path: entry.path, node_type: 'directory' };
  }
  return { path: entry.path, node_type: 'file', content: entry.content };
}
function nextFromWorkspace() {
  if (workspaceDefaultsLoading.value) return;
  if (workspaceDefaultsError.value) {
    MessagePlugin.error(workspaceDefaultsError.value);
    return;
  }
  const compose = workspaceFiles.value.find(
    (entry): entry is ProjectWorkspaceDraftFile => entry.node_type !== 'directory' && entry.path === composePath.value,
  );
  if (!composePath.value || !compose?.content?.trim()) {
    MessagePlugin.warning(t('project.create.validation.composeFileNameRequired'));
    return;
  }
  if (!workspaceDefaults.value?.lifecycle_configuration) {
    lifecycleConfigError.value = t('project.create.lifecycle.defaultsLoadFailed');
    MessagePlugin.error(lifecycleConfigError.value);
    return;
  }
  const composeFilePath = composePath.value;
  const canonicalProjectName = formData.application_name.trim();
  if (!lifecycleDraft.value) {
    lifecycleDraft.value = buildBlankLifecycleConfigurationDraft(workspaceDefaults.value, {
      composeFilePath,
      canonicalProjectName,
    });
  } else {
    lifecycleDraft.value.compose_files = [composeFilePath];
    lifecycleDraft.value.canonical_project_name = canonicalProjectName;
  }
  step.value++;
}
function nextFromLifecycle() {
  if (!lifecycleDraft.value) {
    lifecycleConfigError.value = t('project.create.lifecycle.defaultsLoadFailed');
    MessagePlugin.error(lifecycleConfigError.value);
    return;
  }
  step.value++;
}
async function createProject(confirmedReuse = false) {
  if (runtimeTargetId.value === null) {
    MessagePlugin.warning(t('project.runtimeTarget.unavailableTooltip'));
    return;
  }
  if (!composePath.value) {
    MessagePlugin.warning(t('project.create.validation.composeFileNameRequired'));
    return;
  }
  if (!lifecycleDraft.value) {
    MessagePlugin.error(t('project.create.lifecycle.defaultsLoadFailed'));
    return;
  }

  if (reusingExistingWorkspace.value && !confirmedReuse) {
    reuseWriteDialogVisible.value = true;
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
function confirmCreateReuse() {
  reuseWriteDialogVisible.value = false;
  void createProject(true);
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
