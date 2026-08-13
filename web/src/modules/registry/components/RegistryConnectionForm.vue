<template>
  <t-form ref="formRef" :data="model" :rules="rules" label-align="top">
    <t-form-item v-if="!editing" :label="t('registry.list.form.connectionRef')" name="connection_ref">
      <t-input v-model="model.connection_ref" :disabled="editing" />
    </t-form-item>
    <t-form-item :label="t('registry.list.form.displayName')" name="display_name">
      <t-input v-model="model.display_name" />
    </t-form-item>
    <t-form-item :label="t('registry.list.form.endpoint')" name="endpoint">
      <t-input v-model="model.endpoint" placeholder="https://registry.example.com" />
    </t-form-item>
    <t-form-item :label="t('registry.list.form.description')">
      <t-textarea v-model="model.description" />
    </t-form-item>
    <t-form-item :label="t('registry.list.form.enabled')"><t-switch v-model="model.enabled" /></t-form-item>
    <t-form-item :label="t('registry.list.form.insecure')"><t-switch v-model="model.insecure" /></t-form-item>
  </t-form>
</template>
<script setup lang="ts">
// Registry Connection 表单只承载可编辑的非机密连接字段，调用页分别拥有创建或更新提交语义。
import type { FormInstanceFunctions } from 'tdesign-vue-next';
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

export type RegistryConnectionFormData = {
  connection_ref: string;
  display_name: string;
  endpoint: string;
  enabled: boolean;
  insecure: boolean;
  description: string;
};

defineProps<{ editing?: boolean }>();

const model = defineModel<RegistryConnectionFormData>({ required: true });
const { t } = useI18n();
const formRef = ref<FormInstanceFunctions | null>(null);
const rules = computed(() => ({
  connection_ref: [{ required: true, message: t('registry.list.form.connectionRef') }],
  display_name: [{ required: true, message: t('registry.list.form.displayName') }],
  endpoint: [{ required: true, message: t('registry.list.form.endpoint') }],
}));

defineExpose({
  validate: () => formRef.value?.validate(),
});
</script>
