<template>
  <section class="build-create-page">
    <header>
      <h1>{{ t('build.jobs.create.title') }}</h1>
    </header>
    <t-form :data="form" :rules="rules" @submit="submit"
      ><t-form-item name="application_id" :label="t('build.jobs.create.applicationId')"
        ><t-input v-model="form.application_id" /></t-form-item
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
// 创建表单只提交 Build 所有的规范请求，应用授权仍由服务端边界负责。
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
  application_id: '',
  context_path: '.',
  dockerfile_path: 'Dockerfile',
  image_repository: '',
  image_tag: 'latest',
});
// 相同表单的失败重试必须复用同一幂等键；成功或输入变化后才允许生成新键。
let idempotencyKey: string | undefined;
let idempotencyPayload: string | undefined;
let idempotencySequence = 0;
const rules = {
  application_id: [{ required: true, message: t('build.jobs.create.applicationIdRequired') }],
  context_path: [{ required: true, message: t('build.jobs.create.contextPathRequired') }],
  dockerfile_path: [{ required: true, message: t('build.jobs.create.dockerfilePathRequired') }],
  image_repository: [{ required: true, message: t('build.jobs.create.repositoryRequired') }],
  image_tag: [{ required: true, message: t('build.jobs.create.tagRequired') }],
};
async function submit({ validateResult }: SubmitContext) {
  if (validateResult !== true) return;
  submitting.value = true;
  message.value = '';
  const payload = { ...form.value };
  const payloadSnapshot = JSON.stringify(payload);
  if (idempotencyPayload !== payloadSnapshot) {
    idempotencyPayload = payloadSnapshot;
    idempotencyKey = createIdempotencyKey();
  }
  const currentIdempotencyKey = idempotencyKey ?? createIdempotencyKey();
  idempotencyKey = currentIdempotencyKey;
  try {
    await createBuildJob(payload, currentIdempotencyKey);
    messageTheme.value = 'success';
    message.value = t('build.jobs.create.submitted');
    idempotencyKey = undefined;
    idempotencyPayload = undefined;
    await router.push(BUILD_ROUTE_PATH.JOBS);
  } catch (error) {
    messageTheme.value = 'error';
    message.value = resolveLocalizedErrorMessage(t, error, t('build.jobs.create.submitFailed'));
  } finally {
    submitting.value = false;
  }
}

function createIdempotencyKey() {
  const uuid = globalThis.crypto?.randomUUID?.();
  if (uuid) return uuid;

  idempotencySequence += 1;
  return `build-job-create-${Date.now()}-${idempotencySequence}`;
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
