<template>
  <section class="build-create-page">
    <header>
      <h1>{{ t('build.jobs.create.title') }}</h1>
    </header>
    <t-form :data="form" :rules="rules" @submit="submit"
      ><t-form-item name="application_id" :label="t('build.jobs.create.applicationId')"
        ><t-input-number v-model="form.application_id" :min="1" /></t-form-item
      ><t-form-item name="context_path" :label="t('build.jobs.create.contextPath')"
        ><t-input v-model="form.context_path" /></t-form-item
      ><t-form-item name="dockerfile_path" :label="t('build.jobs.create.dockerfilePath')"
        ><t-input v-model="form.dockerfile_path" /></t-form-item
      ><t-form-item name="image_repository" :label="t('build.jobs.create.repository')"
        ><t-input v-model="form.image_repository" /></t-form-item
      ><t-form-item name="image_tag" :label="t('build.jobs.create.tag')"
        ><t-input v-model="form.image_tag" /></t-form-item
      ><t-button theme="primary" type="submit" :loading="submitting">{{
        t('build.jobs.create.submit')
      }}</t-button></t-form
    ><t-alert v-if="message" :theme="messageTheme" :message="message" />
  </section>
</template>
<script setup lang="ts">
// The create form submits the Build-owned canonical request and leaves application authorization to the server boundary.
import type { SubmitContext } from 'tdesign-vue-next';
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { createBuildJob } from '../../api/build';
import { BUILD_ROUTE_PATH } from '../../contract/paths';
import type { BuildJobCreateRequest } from '../../types/build';
const { t } = useI18n();
const router = useRouter();
const submitting = ref(false);
const message = ref('');
const messageTheme = ref<'success' | 'error'>('success');
const form = ref<BuildJobCreateRequest>({
  application_id: 0,
  context_path: '.',
  dockerfile_path: 'Dockerfile',
  image_repository: '',
  image_tag: 'latest',
});
const rules = {
  application_id: [{ required: true, min: 1 }],
  context_path: [{ required: true }],
  dockerfile_path: [{ required: true }],
  image_repository: [{ required: true }],
  image_tag: [{ required: true }],
};
async function submit({ validateResult }: SubmitContext) {
  if (validateResult !== true) return;
  submitting.value = true;
  message.value = '';
  try {
    await createBuildJob(form.value, globalThis.crypto?.randomUUID?.() ?? `${Date.now()}`);
    messageTheme.value = 'success';
    message.value = t('build.jobs.create.submitted');
    await router.push(BUILD_ROUTE_PATH.JOBS);
  } catch (error) {
    messageTheme.value = 'error';
    message.value = resolveLocalizedErrorMessage(t, error, t('build.jobs.create.submitFailed'));
  } finally {
    submitting.value = false;
  }
}
</script>
<style scoped lang="less">
.build-create-page {
  display: grid;
  gap: var(--graft-density-gap-16);
  max-width: 720px;
}

.build-create-page h1 {
  margin: 0;
}
</style>
