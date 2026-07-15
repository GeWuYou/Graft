<template>
  <management-page-content>
    <management-page-header
      title-key="project.route.createTemplate.title"
      :description="t('project.sourceCreate.templateDescription')"
      :source="{ labelKey: 'project.creation.eyebrow', fallback: t('project.creation.eyebrow') }"
    >
      <template #actions
        ><t-space size="small"
          ><t-button variant="outline" @click="goToSource">{{ t('project.create.actions.backToSource') }}</t-button
          ><t-button @click="refreshPage">{{ t('project.create.actions.refresh') }}</t-button></t-space
        ></template
      >
    </management-page-header>
    <t-card :bordered="true"
      ><t-form ref="formRef" :data="form" :rules="formRules" layout="vertical" @submit="onCreate"
        ><div class="source-create__grid">
          <t-form-item :label="t('project.sourceCreate.displayName')" name="display_name"
            ><t-input v-model="form.display_name" /></t-form-item
          ><t-form-item :label="t('project.sourceCreate.applicationName')" name="application_name"
            ><t-input
              v-model="form.application_name"
              :placeholder="t('project.sourceCreate.applicationNamePlaceholder')" /></t-form-item
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
import { type FormInstanceFunctions, type FormProps, MessagePlugin } from 'tdesign-vue-next';
import { computed, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { postProjectCreateTemplate } from '../../api/project';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { navigateToProjectCreateSource, refreshProjectCreatePage } from '../../shared/navigation';
import type { ProjectTemplateCreateRequest } from '../../types/project';
defineOptions({ name: 'ProjectSourceCreate' });
const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const creating = ref(false);
const formRef = ref<FormInstanceFunctions | null>(null);
const runtimeTargetId = computed(() => {
  const raw = route.query.runtime_target_id;
  return typeof raw === 'string' && /^[1-9]\d*$/.test(raw) && Number.isSafeInteger(Number(raw)) ? Number(raw) : null;
});
const form = reactive({ display_name: '', application_name: '' });
const formRules: FormProps['rules'] = {
  display_name: [{ required: true, message: t('project.create.validation.displayNameRequired') }],
  application_name: [
    { required: true, message: t('project.create.validation.applicationNameRequired') },
    {
      validator: (value) => /^[a-z0-9][a-z0-9-]*$/.test(String(value)),
      message: t('project.create.validation.applicationNamePattern'),
    },
  ],
};
const templateForm = reactive({ template_key: 'empty-compose', template_version: 'v1', template_instance_name: '' });
const templateOptions = computed(() => [
  { label: t('project.sourceCreate.emptyComposeTemplate'), value: 'empty-compose' },
]);
function templatePayload(runtimeTargetIdValue: number): ProjectTemplateCreateRequest {
  return {
    display_name: form.display_name.trim(),
    runtime_target_id: runtimeTargetIdValue,
    application_name: form.application_name.trim(),
    template_key: templateForm.template_key === 'empty-compose' ? 'empty-compose' : undefined,
    template_version: templateForm.template_version === 'v1' ? 'v1' : undefined,
    template_instance_name: templateForm.template_instance_name.trim() || undefined,
  };
}
async function onCreate() {
  if ((await formRef.value?.validate()) !== true) return;
  if (runtimeTargetId.value === null) {
    MessagePlugin.warning(t('project.runtimeTarget.unavailableTooltip'));
    goToSource();
    return;
  }
  if (templateForm.template_key !== 'empty-compose' || templateForm.template_version !== 'v1') {
    MessagePlugin.warning(t('project.sourceCreate.template'));
    return;
  }
  creating.value = true;
  try {
    const result = await postProjectCreateTemplate(templatePayload(runtimeTargetId.value));
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
function goToSource() {
  navigateToProjectCreateSource(router, route.query);
}
function refreshPage() {
  refreshProjectCreatePage(router, PROJECT_BOOTSTRAP_ROUTE.CREATE_TEMPLATE.pageRouteName, route.query);
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
