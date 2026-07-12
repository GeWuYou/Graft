<template>
  <management-page-content>
    <management-page-header
      title-key="project.route.createTemplate.title"
      :description="t('project.sourceCreate.templateDescription')"
      :source="{ labelKey: 'project.creation.eyebrow', fallback: t('project.creation.eyebrow') }"
    />

    <t-card :bordered="true">
      <t-form :data="form" layout="vertical" @submit="onCreate">
        <div class="source-create__grid">
          <t-form-item :label="t('project.sourceCreate.displayName')" name="display_name">
            <t-input v-model="form.display_name" />
          </t-form-item>
          <t-form-item :label="t('project.sourceCreate.canonicalName')" name="canonical_project_name">
            <t-input v-model="form.canonical_project_name" />
          </t-form-item>
          <t-form-item :label="t('project.sourceCreate.relativeDirectory')" name="relative_project_directory">
            <t-input v-model="form.relative_project_directory" />
          </t-form-item>
          <t-form-item :label="t('project.sourceCreate.template')" name="template_key">
            <t-select v-model="templateForm.template_key" :options="templateOptions" />
          </t-form-item>
          <t-form-item :label="t('project.sourceCreate.instanceName')" name="template_instance_name">
            <t-input v-model="templateForm.template_instance_name" :placeholder="form.canonical_project_name" />
          </t-form-item>
        </div>

        <t-alert theme="info" :message="t('project.sourceCreate.lifecycleHint')" class="source-create__notice" />
        <t-checkbox v-model="deployAfterCreate" :disabled="!canDeployAfterCreate">
          {{ t('project.sourceCreate.deployAfterCreate') }}
        </t-checkbox>
        <t-alert
          v-if="!canDeployAfterCreate"
          theme="info"
          :message="t('project.sourceCreate.deployPermissionRequired')"
          class="source-create__notice"
        />
        <t-alert
          v-if="validation"
          theme="success"
          :message="t('project.sourceCreate.validated', { directory: validation.working_directory })"
          class="source-create__notice"
        />

        <t-space class="source-create__actions">
          <t-button variant="outline" :loading="validating" @click="onValidate">{{
            t('project.sourceCreate.validate')
          }}</t-button>
          <t-button theme="primary" type="submit" :loading="creating">{{ t('project.sourceCreate.create') }}</t-button>
        </t-space>
      </t-form>
    </t-card>
  </management-page-content>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { usePermissionStore } from '@/store';

import { postProjectCreateTemplate, postProjectCreateTemplateValidate, postProjectDeploy } from '../../api/project';
import { PROJECT_PERMISSION_CODE } from '../../contract/permissions';
import { createWithOptionalDeploy } from '../../shared/create-with-optional-deploy';
import type { ProjectCreateValidateResponse, ProjectTemplateCreateRequest } from '../../types/project';

defineOptions({ name: 'ProjectSourceCreate' });

const router = useRouter();
const { t } = useI18n();
const permissionStore = usePermissionStore();
const validating = ref(false);
const creating = ref(false);
const deployAfterCreate = ref(false);
const validation = ref<ProjectCreateValidateResponse | null>(null);
const form = reactive({ display_name: '', canonical_project_name: '', relative_project_directory: '' });
const templateForm = reactive({ template_key: 'empty-compose', template_version: 'v1', template_instance_name: '' });
const templateOptions = computed(() => [
  { label: t('project.sourceCreate.emptyComposeTemplate'), value: 'empty-compose' },
]);
const canDeployAfterCreate = computed(() => permissionStore.hasPermission(PROJECT_PERMISSION_CODE.DEPLOY));

function templatePayload(): ProjectTemplateCreateRequest {
  return { ...form, ...templateForm };
}

async function onValidate() {
  validating.value = true;
  try {
    validation.value = await postProjectCreateTemplateValidate(templatePayload());
    MessagePlugin.success(t('project.sourceCreate.validateSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.sourceCreate.validateFailed')));
  } finally {
    validating.value = false;
  }
}

async function onCreate() {
  creating.value = true;
  try {
    const result = await createWithOptionalDeploy({
      create: () => postProjectCreateTemplate(templatePayload()),
      deploy: postProjectDeploy,
      deployAfterCreate: deployAfterCreate.value && canDeployAfterCreate.value,
    });
    MessagePlugin.success(t('project.sourceCreate.createSuccess'));
    if (result.deployment.status === 'succeeded') {
      MessagePlugin.success(t('project.sourceCreate.deploySuccess'));
    }
    if (result.deployment.status === 'failed') {
      MessagePlugin.error(
        resolveLocalizedErrorMessage(t, result.deployment.error, t('project.sourceCreate.deployFailed')),
      );
    }
    await router.push({ name: 'ProjectDetailIndex', params: { id: result.created.project_id } });
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.sourceCreate.createFailed')));
  } finally {
    creating.value = false;
  }
}
</script>
<style scoped>
.source-create__grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
}

.source-create__notice {
  margin-top: var(--graft-density-gap-16);
}

.source-create__actions {
  margin-top: var(--graft-density-gap-16);
}
</style>
