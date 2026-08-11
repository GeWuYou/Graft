<template>
  <section class="build-create-page">
    <header>
      <h1>{{ t('build.jobs.create.title') }}</h1>
    </header>
    <t-form :data="form" :rules="rules" @submit="submit"
      ><t-form-item name="workspace_id" :label="t('build.jobs.create.workspace')"
        ><t-select
          v-model="form.workspace_id"
          :options="workspaceOptions"
          :loading="selectorLoading"
          :disabled="selectorLoading || selectorOptionsUnavailable"
          :placeholder="t('build.jobs.create.workspacePlaceholder')"
          clearable /></t-form-item
      ><t-form-item name="builder_selection" :label="t('build.jobs.create.builderSelection')"
        ><t-radio-group v-model="selectionMode">
          <t-radio value="target">{{ t('build.jobs.create.runtimeTargetMode') }}</t-radio>
          <t-radio value="pool">{{ t('build.jobs.create.builderPoolMode') }}</t-radio>
        </t-radio-group></t-form-item
      ><t-form-item
        v-if="selectionMode === 'target'"
        name="runtime_target_id"
        :label="t('build.jobs.create.runtimeTarget')"
        ><t-select
          v-model="form.runtime_target_id"
          :options="runtimeTargetOptions"
          :loading="selectorLoading"
          :disabled="selectorLoading || selectorOptionsUnavailable"
          :placeholder="t('build.jobs.create.runtimeTargetPlaceholder')"
          clearable /></t-form-item
      ><t-form-item v-else name="builder_pool_id" :label="t('build.jobs.create.builderPool')"
        ><t-select
          v-model="form.builder_pool_id"
          :options="builderPoolOptions"
          :loading="selectorLoading"
          :disabled="selectorLoading || selectorOptionsUnavailable"
          :placeholder="t('build.jobs.create.builderPoolPlaceholder')"
          clearable /></t-form-item
      ><t-form-item name="template_ref" :label="t('build.jobs.create.template')"
        ><t-input v-model="form.template_ref" disabled /></t-form-item
      ><t-form-item name="driver" :label="t('build.jobs.create.driver')"
        ><t-input v-model="form.driver" disabled /></t-form-item
      ><t-form-item name="destination.connection_ref" :label="t('build.jobs.create.registry')"
        ><t-select
          v-model="form.destination.connection_ref"
          :options="registryOptions"
          :loading="destinationLoading"
          :placeholder="t('build.jobs.create.registryPlaceholder')"
          clearable /></t-form-item
      ><t-form-item name="destination.repository_ref" :label="t('build.jobs.create.repository')"
        ><t-select
          v-model="form.destination.repository_ref"
          :options="repositoryOptions"
          :loading="destinationLoading"
          :disabled="!form.destination.connection_ref"
          :placeholder="t('build.jobs.create.repositoryPlaceholder')"
          clearable /></t-form-item
      ><t-form-item name="destination.reference" :label="t('build.jobs.create.tag')"
        ><t-input v-model="form.destination.reference" /></t-form-item
      ><t-button theme="primary" type="submit" :loading="submitting">{{
        t('build.jobs.create.submit')
      }}</t-button></t-form
    ><t-alert
      v-if="!selectorLoading && selectorOptionsUnavailable && !selectorError"
      theme="warning"
      :message="t('build.jobs.create.selectorsEmpty')"
    />
    <t-alert v-if="selectorError" theme="warning" :message="selectorError" />
    <t-alert v-if="!destinationLoading && !destinationError && registryOptions.length === 0" theme="warning">
      <template #message>{{ t('build.jobs.create.destinationsEmpty') }}</template>
      <template #operation
        ><t-button size="small" variant="outline" @click="openRegistries">{{
          t('build.jobs.create.addRegistry')
        }}</t-button></template
      >
    </t-alert>
    <t-alert v-if="destinationError" theme="warning" :message="destinationError" />
    <t-alert v-if="message" :theme="messageTheme" :message="message" />
  </section>
</template>
<script setup lang="ts">
// 创建表单只提交 Build 所有的规范请求，应用授权仍由服务端边界负责。
import type { SubmitContext } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import {
  createBuildJob,
  getBuildBuilderPools,
  getBuildRegistryDestinations,
  getBuildRuntimeTargets,
  getBuildWorkspaces,
} from '../../api/build';
import { BUILD_ROUTE_PATH } from '../../contract/paths';
import type { BuildBuilderPool } from '../../types/build';
import { BUILD_DRIVER_REF, BUILD_TEMPLATE_REF } from '../../types/build';
const { locale, t } = useI18n();
const router = useRouter();
const submitting = ref(false);
const message = ref('');
const messageTheme = ref<'success' | 'error'>('success');
const selectionMode = ref<'target' | 'pool'>('target');
type BuildJobForm = Parameters<typeof createBuildJob>[0];
const form = ref<BuildJobForm>({
  workspace_id: '',
  runtime_target_id: 0,
  template_ref: BUILD_TEMPLATE_REF,
  driver: BUILD_DRIVER_REF,
  platforms: ['linux/amd64'],
  destination: { kind: 'oci_registry', connection_ref: '', repository_ref: '', reference: 'latest' },
});
type SelectorOption = { label: string; value: string | number };
type BuilderPoolOption = SelectorOption & { policy: BuildBuilderPool['scheduling_policy'] };
const workspaceOptions = ref<SelectorOption[]>([]);
const runtimeTargetOptions = ref<SelectorOption[]>([]);
const builderPools = ref<BuildBuilderPool[]>([]);
const builderPoolOptions = computed<BuilderPoolOption[]>(() => {
  return builderPools.value.map((item) => ({
    label: `${item.display_name} (${builderPoolPolicyLabel(item.scheduling_policy)})`,
    value: item.pool_id,
    policy: item.scheduling_policy,
  }));
});
const selectorLoading = ref(false);
const selectorError = ref('');
const destinationLoading = ref(false);
const destinationError = ref('');
type RegistryDestination = Awaited<ReturnType<typeof getBuildRegistryDestinations>>['items'][number];
const destinations = ref<RegistryDestination[]>([]);
const registryOptions = computed(() => {
  const unique = new Map<string, string>();
  for (const item of destinations.value) unique.set(item.connection_ref, item.connection_display_name);
  return [...unique].map(([value, label]) => ({ value, label }));
});
const repositoryOptions = computed(() =>
  destinations.value
    .filter((item) => item.connection_ref === form.value.destination.connection_ref)
    .map((item) => ({ value: item.repository_ref, label: item.repository_display_name || item.repository_ref })),
);
const selectorOptionsUnavailable = computed(
  () =>
    !selectorLoading.value &&
    (workspaceOptions.value.length === 0 ||
      (selectionMode.value === 'target'
        ? runtimeTargetOptions.value.length === 0
        : builderPoolOptions.value.length === 0)),
);

onMounted(loadSelectorOptions);

async function loadSelectorOptions() {
  selectorLoading.value = true;
  selectorError.value = '';
  destinationLoading.value = true;
  destinationError.value = '';
  try {
    const [workspaces, targets, pools] = await Promise.all([
      getBuildWorkspaces(),
      getBuildRuntimeTargets(),
      getBuildBuilderPools(),
    ]);
    workspaceOptions.value = (workspaces.items ?? []).map((item) => ({ label: item.name, value: item.workspace_id }));
    runtimeTargetOptions.value = (targets.items ?? []).map((item) => ({
      label: item.display_name,
      value: item.target_id,
    }));
    builderPools.value = pools.items ?? [];
  } catch (error) {
    selectorError.value = resolveLocalizedErrorMessage(t, error, t('build.jobs.create.selectorsLoadFailed'));
  } finally {
    selectorLoading.value = false;
  }
  try {
    const registryDestinations = await getBuildRegistryDestinations();
    destinations.value = registryDestinations.items ?? [];
  } catch (error) {
    destinationError.value = resolveLocalizedErrorMessage(t, error, t('build.jobs.create.destinationsLoadFailed'));
  } finally {
    destinationLoading.value = false;
  }
}

function builderPoolPolicyLabel(policy: BuildBuilderPool['scheduling_policy']) {
  switch (policy) {
    case 'manual':
      return t('build.jobs.create.builderPoolPolicy.manual', locale.value);
    case 'round_robin':
      return t('build.jobs.create.builderPoolPolicy.roundRobin', locale.value);
    case 'random':
      return t('build.jobs.create.builderPoolPolicy.random', locale.value);
    case 'least_load':
      return t('build.jobs.create.builderPoolPolicy.leastLoad', locale.value);
    case 'capacity':
      return t('build.jobs.create.builderPoolPolicy.capacity', locale.value);
    case 'affinity':
      return t('build.jobs.create.builderPoolPolicy.affinity', locale.value);
  }
}

watch(selectionMode, (mode) => {
  if (mode === 'pool') {
    form.value.runtime_target_id = undefined;
  } else {
    form.value.builder_pool_id = undefined;
  }
});
watch(
  () => form.value.destination.connection_ref,
  () => {
    form.value.destination.repository_ref = '';
  },
);
// 相同表单的失败重试必须复用同一幂等键；成功或输入变化后才允许生成新键。
let idempotencyKey: string | undefined;
let idempotencyPayload: string | undefined;
let idempotencySequence = 0;
const rules = computed(() => ({
  workspace_id: [{ required: true, message: t('build.jobs.create.workspaceIdRequired') }],
  runtime_target_id:
    selectionMode.value === 'target'
      ? [{ required: true, min: 1, message: t('build.jobs.create.runtimeTargetRequired') }]
      : [],
  builder_pool_id:
    selectionMode.value === 'pool' ? [{ required: true, message: t('build.jobs.create.builderPoolRequired') }] : [],
  'destination.connection_ref': [{ required: true, message: t('build.jobs.create.registryRequired') }],
  'destination.repository_ref': [{ required: true, message: t('build.jobs.create.repositoryRequired') }],
  'destination.reference': [{ required: true, message: t('build.jobs.create.tagRequired') }],
}));

function openRegistries() {
  void router.push('/infrastructure/registries');
}
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
