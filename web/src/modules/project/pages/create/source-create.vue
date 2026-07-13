<template>
  <management-page-content>
    <management-page-header
      title-key="project.route.createTemplate.title"
      :description="t('project.sourceCreate.templateDescription')"
      :source="{ labelKey: 'project.creation.eyebrow', fallback: t('project.creation.eyebrow') }"
    >
      <template #actions
        ><t-space size="small"
          ><t-button variant="outline" @click="goToList">{{ t('project.create.actions.back') }}</t-button
          ><t-button @click="refreshPage">{{ t('project.create.actions.refresh') }}</t-button></t-space
        ></template
      >
    </management-page-header>
    <t-card :bordered="true"
      ><t-form :data="form" layout="vertical" @submit="onCreate"
        ><div class="source-create__grid">
          <t-form-item :label="t('project.sourceCreate.displayName')" name="display_name"
            ><t-input v-model="form.display_name" /></t-form-item
          ><t-form-item :label="t('project.sourceCreate.workspaceKey')" name="workspace_key"
            ><t-input
              v-model="form.workspace_key"
              :placeholder="t('project.sourceCreate.workspaceKeyPlaceholder')" /></t-form-item
          ><t-form-item :label="t('project.sourceCreate.template')" name="template_key"
            ><t-select v-model="templateForm.template_key" :options="templateOptions"
          /></t-form-item>
        </div>
        <t-space class="source-create__actions"
          ><t-button theme="primary" type="submit" :loading="creating">{{
            t('project.sourceCreate.create')
          }}</t-button></t-space
        ></t-form
      ></t-card
    >
  </management-page-content>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { postProjectCreateTemplate } from '../../api/project';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import type { ProjectTemplateCreateRequest } from '../../types/project';
defineOptions({ name: 'ProjectSourceCreate' });
const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const creating = ref(false);
const runtimeTargetId = computed(() => Number(route.query.runtime_target_id));
const form = reactive({ display_name: '', workspace_key: '' });
const templateForm = reactive({ template_key: 'empty-compose', template_version: 'v1', template_instance_name: '' });
const templateOptions = computed(() => [
  { label: t('project.sourceCreate.emptyComposeTemplate'), value: 'empty-compose' },
]);
function templatePayload(): ProjectTemplateCreateRequest {
  return {
    display_name: form.display_name.trim(),
    runtime_target_id: runtimeTargetId.value,
    ...(form.workspace_key.trim() ? { workspace_key: form.workspace_key.trim() } : {}),
    ...templateForm,
  };
}
async function onCreate() {
  creating.value = true;
  try {
    const result = await postProjectCreateTemplate(templatePayload());
    MessagePlugin.success(t('project.sourceCreate.createSuccess'));
    await router.push({
      name: PROJECT_BOOTSTRAP_ROUTE.CONFIGURATION_WORKSPACE.pageRouteName,
      params: { id: result.application_id },
    });
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.sourceCreate.createFailed')));
  } finally {
    creating.value = false;
  }
}
function goToList() {
  void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.LIST.routeName });
}
function refreshPage() {
  void router.replace({ name: PROJECT_BOOTSTRAP_ROUTE.CREATE_TEMPLATE.pageRouteName, query: route.query });
}
</script>
<style scoped>
.source-create__grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
}

.source-create__actions {
  margin-top: var(--graft-density-gap-16);
}
</style>
