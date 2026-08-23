<template>
  <section class="build-create-page" data-page-type="workflow">
    <management-page-content>
      <management-page-header
        class="build-create-page__surface"
        title-key="build.jobs.create.title"
        description-key="build.jobs.create.description"
        :source="{ labelKey: 'build.jobs.create.eyebrow', fallback: t('build.jobs.create.eyebrow') }"
      />

      <t-card bordered class="build-create-page__surface" data-testid="build-create-form-card">
        <t-form class="build-create-page__form" label-align="top" :data="form" :rules="rules" @submit="submit">
          <section class="build-create-page__section" data-testid="build-create-section-source">
            <header class="build-create-page__section-header">
              <h2>{{ t('build.jobs.create.sections.source.title') }}</h2>
              <p>{{ t('build.jobs.create.sections.source.description') }}</p>
            </header>
            <div class="build-create-page__grid">
              <t-form-item
                class="build-create-page__field--full"
                name="source_mode"
                :label="t('build.jobs.create.sourceMode')"
              >
                <t-radio-group v-model="sourceMode">
                  <t-radio value="upload">{{ t('build.jobs.create.uploadArchive') }}</t-radio>
                  <t-radio value="reuse">{{ t('build.jobs.create.reuseSnapshot') }}</t-radio>
                </t-radio-group>
              </t-form-item>
              <t-form-item
                v-if="sourceMode === 'upload'"
                class="build-create-page__field--full"
                name="archive"
                :label="t('build.jobs.create.archive')"
              >
                <input
                  class="build-create-page__file-input"
                  data-testid="build-input-snapshot-file"
                  type="file"
                  accept=".zip,.tar,.tgz,.tar.gz,application/zip,application/x-tar,application/gzip"
                  @change="onArchiveChange"
                />
                <p class="build-create-page__field-hint">
                  {{ archiveFile?.name || t('build.jobs.create.archivePlaceholder') }}
                </p>
              </t-form-item>
              <t-form-item
                v-else
                class="build-create-page__field--full"
                name="input_snapshot_id"
                :label="t('build.jobs.create.snapshot')"
              >
                <t-select
                  v-model="form.input_snapshot_id"
                  :options="snapshotOptions"
                  :loading="snapshotLoading"
                  :disabled="snapshotLoading || snapshotOptions.length === 0"
                  :placeholder="t('build.jobs.create.snapshotPlaceholder')"
                  clearable
                />
                <t-button
                  v-if="snapshotHasMore"
                  data-testid="build-load-more-snapshots"
                  class="build-create-page__load-more"
                  variant="text"
                  :loading="snapshotLoading"
                  :disabled="snapshotLoading"
                  @click="loadSnapshots()"
                >
                  {{ t('components.commonTable.more') }}
                </t-button>
              </t-form-item>
            </div>
            <div class="build-create-page__section-feedback">
              <t-alert
                v-if="sourceMode === 'reuse' && !snapshotLoading && !snapshotError && snapshotOptions.length === 0"
                theme="warning"
                :message="t('build.jobs.create.snapshotEmpty')"
              />
              <t-alert v-if="snapshotError" theme="warning" :message="snapshotError" />
            </div>
          </section>

          <section class="build-create-page__section" data-testid="build-create-section-execution">
            <header class="build-create-page__section-header">
              <h2>{{ t('build.jobs.create.sections.execution.title') }}</h2>
              <p>{{ t('build.jobs.create.sections.execution.description') }}</p>
            </header>
            <div class="build-create-page__grid">
              <t-form-item
                class="build-create-page__field--full"
                name="builder_selection"
                :label="t('build.jobs.create.builderSelection')"
              >
                <t-radio-group v-model="selectionMode">
                  <t-radio value="target">{{ t('build.jobs.create.runtimeTargetMode') }}</t-radio>
                  <t-radio value="pool">{{ t('build.jobs.create.builderPoolMode') }}</t-radio>
                </t-radio-group>
              </t-form-item>
              <t-form-item
                v-if="selectionMode === 'target'"
                class="build-create-page__field--full"
                name="runtime_target_id"
                :label="t('build.jobs.create.runtimeTarget')"
              >
                <t-select
                  v-model="form.runtime_target_id"
                  :options="runtimeTargetOptions"
                  :loading="runtimeTargetLoading"
                  :disabled="runtimeTargetLoading || runtimeTargetOptions.length === 0"
                  :placeholder="t('build.jobs.create.runtimeTargetPlaceholder')"
                  clearable
                />
              </t-form-item>
              <t-form-item
                v-else
                class="build-create-page__field--full"
                name="builder_pool_id"
                :label="t('build.jobs.create.builderPool')"
              >
                <t-select
                  v-model="form.builder_pool_id"
                  :options="builderPoolOptions"
                  :loading="builderPoolLoading"
                  :disabled="builderPoolLoading || builderPoolOptions.length === 0"
                  :placeholder="t('build.jobs.create.builderPoolPlaceholder')"
                  clearable
                />
              </t-form-item>
              <t-form-item name="template_ref" :label="t('build.jobs.create.template')">
                <t-input v-model="form.template_ref" disabled />
              </t-form-item>
              <t-form-item name="driver" :label="t('build.jobs.create.driver')">
                <t-select v-model="form.driver" :options="driverOptions" :disabled="selectionMode === 'target'" />
              </t-form-item>
              <t-form-item
                class="build-create-page__field--full"
                name="platforms"
                :label="t('build.jobs.create.platforms')"
              >
                <t-checkbox-group v-model="form.platforms" :options="platformOptions" />
                <p v-if="selectionMode === 'target'" class="build-create-page__field-hint">
                  {{ t('build.jobs.create.arm64PoolHint') }}
                </p>
              </t-form-item>
            </div>
            <div class="build-create-page__section-feedback">
              <t-alert
                v-if="
                  selectionMode === 'target' &&
                  !runtimeTargetLoading &&
                  !runtimeTargetError &&
                  runtimeTargetOptions.length === 0
                "
                theme="warning"
                :message="t('build.jobs.create.runtimeTargetEmpty')"
              />
              <t-alert
                v-if="
                  selectionMode === 'pool' &&
                  !builderPoolLoading &&
                  !builderPoolError &&
                  builderPoolOptions.length === 0
                "
                theme="warning"
                :message="t('build.jobs.create.builderPoolEmpty')"
              />
              <t-alert
                v-if="selectionMode === 'target' && runtimeTargetError"
                theme="warning"
                :message="runtimeTargetError"
              />
              <t-alert
                v-if="selectionMode === 'pool' && builderPoolError"
                theme="warning"
                :message="builderPoolError"
              />
            </div>
          </section>

          <section class="build-create-page__section" data-testid="build-create-section-destination">
            <header class="build-create-page__section-header">
              <h2>{{ t('build.jobs.create.sections.destination.title') }}</h2>
              <p>{{ t('build.jobs.create.sections.destination.description') }}</p>
            </header>
            <div class="build-create-page__grid">
              <t-form-item name="destination.connection_ref" :label="t('build.jobs.create.registry')">
                <t-select
                  v-model="form.destination.connection_ref"
                  :options="registryOptions"
                  :loading="destinationLoading"
                  :placeholder="t('build.jobs.create.registryPlaceholder')"
                  clearable
                />
              </t-form-item>
              <t-form-item name="destination.repository_ref" :label="t('build.jobs.create.repository')">
                <t-select
                  v-model="form.destination.repository_ref"
                  :options="repositoryOptions"
                  :loading="destinationLoading"
                  :disabled="!form.destination.connection_ref"
                  :placeholder="t('build.jobs.create.repositoryPlaceholder')"
                  clearable
                />
              </t-form-item>
              <t-form-item
                class="build-create-page__field--full"
                name="destination.reference"
                :label="t('build.jobs.create.tag')"
              >
                <t-input v-model="form.destination.reference" />
              </t-form-item>
            </div>
            <div class="build-create-page__section-feedback">
              <t-alert v-if="!destinationLoading && !destinationError && registryOptions.length === 0" theme="warning">
                <template #message>{{ t('build.jobs.create.destinationsEmpty') }}</template>
                <template #operation>
                  <t-button size="small" variant="outline" @click="openRegistries">
                    {{ t('build.jobs.create.addRegistry') }}
                  </t-button>
                </template>
              </t-alert>
              <t-alert v-if="destinationError" theme="warning" :message="destinationError" />
            </div>
          </section>

          <t-alert
            v-if="message"
            class="build-create-page__submission-feedback"
            :theme="messageTheme"
            :message="message"
          />

          <div class="build-create-page__actions">
            <t-button class="build-create-page__action" variant="outline" :disabled="submitting" @click="returnToJobs">
              {{ t('build.jobs.create.back') }}
            </t-button>
            <t-button class="build-create-page__action" theme="primary" type="submit" :loading="submitting">
              {{ t('build.jobs.create.submit') }}
            </t-button>
          </div>
        </t-form>
      </t-card>
    </management-page-content>
  </section>
</template>
<script setup lang="ts">
// 创建表单只提交 Build 所有的规范请求；Application/Project 不参与输入选择。
import type { SubmitContext } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { REGISTRY_ROUTE_PATH } from '@/modules/registry/contract/paths';
import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import {
  createBuildJob,
  getBuildBuilderPools,
  getBuildInputSnapshots,
  getBuildRegistryDestinations,
  getBuildRuntimeTargets,
  uploadBuildInputSnapshot,
} from '../../api/build';
import { BUILD_ROUTE_PATH } from '../../contract/paths';
import type { BuildBuilderPool } from '../../types/build';
import {
  BUILD_DRIVER_REF,
  BUILD_MULTI_PLATFORM_DRIVER_REF,
  BUILD_PLATFORM_OPTIONS,
  BUILD_TEMPLATE_REF,
} from '../../types/build';
const { locale, t } = useI18n();
const router = useRouter();
const submitting = ref(false);
const message = ref('');
const messageTheme = ref<'success' | 'error'>('success');
const selectionMode = ref<'target' | 'pool'>('target');
const sourceMode = ref<'upload' | 'reuse'>('upload');
const archiveFile = ref<File>();
type BuildJobForm = Parameters<typeof createBuildJob>[0];
const form = ref<BuildJobForm>({
  input_snapshot_id: '',
  runtime_target_id: undefined,
  template_ref: BUILD_TEMPLATE_REF,
  driver: BUILD_DRIVER_REF,
  platforms: ['linux/amd64'],
  destination: { kind: 'oci_registry', connection_ref: '', repository_ref: '', reference: 'latest' },
});
type SelectorOption = { label: string; value: string | number };
type BuilderPoolOption = SelectorOption & { policy: BuildBuilderPool['scheduling_policy'] };
const driverOptions = computed(() => [
  {
    label: t('build.jobs.create.driverOptions.dockerEngine'),
    value: BUILD_DRIVER_REF,
    disabled: selectionMode.value === 'pool' && form.value.platforms?.includes('linux/arm64'),
  },
  { label: t('build.jobs.create.driverOptions.dockerBuildx'), value: BUILD_MULTI_PLATFORM_DRIVER_REF },
]);
const platformOptions = computed(() =>
  BUILD_PLATFORM_OPTIONS.map((platform) => ({
    label: platform,
    value: platform,
    disabled: selectionMode.value === 'target' && platform === 'linux/arm64',
  })),
);
const snapshotOptions = ref<SelectorOption[]>([]);
const runtimeTargetOptions = ref<SelectorOption[]>([]);
const builderPools = ref<BuildBuilderPool[]>([]);
const builderPoolOptions = computed<BuilderPoolOption[]>(() => {
  return builderPools.value.map((item) => ({
    label: `${item.display_name} (${builderPoolPolicyLabel(item.scheduling_policy)})`,
    value: item.pool_id,
    policy: item.scheduling_policy,
  }));
});
const snapshotLoading = ref(false);
const snapshotOffset = ref(0);
const snapshotTotal = ref<number>();
const runtimeTargetLoading = ref(false);
const builderPoolLoading = ref(false);
const snapshotError = ref('');
const snapshotHasMore = computed(() => snapshotTotal.value !== undefined && snapshotOffset.value < snapshotTotal.value);
const runtimeTargetError = ref('');
const builderPoolError = ref('');
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
onMounted(loadSelectorOptions);

async function loadSelectorOptions() {
  await Promise.all([loadSnapshots(), loadRuntimeTargets(), loadBuilderPools(), loadRegistryDestinations()]);
}

async function loadSnapshots(reset = false) {
  if (snapshotLoading.value || (!reset && !snapshotHasMore.value && snapshotOptions.value.length > 0)) return;
  if (reset) {
    snapshotOffset.value = 0;
    snapshotTotal.value = undefined;
    snapshotOptions.value = [];
  }
  snapshotLoading.value = true;
  snapshotError.value = '';
  const requestOffset = snapshotOffset.value;
  const requestLimit = 100;
  try {
    const snapshots = await getBuildInputSnapshots({ limit: requestLimit, offset: requestOffset });
    const unique = new Map(snapshotOptions.value.map((option) => [String(option.value), option]));
    for (const item of snapshots.items ?? []) {
      if (!item.snapshot_id || unique.has(item.snapshot_id)) continue;
      unique.set(item.snapshot_id, {
        value: item.snapshot_id,
        label: `${item.content_digest} (${item.source_kind})`,
      });
    }
    snapshotOptions.value = [...unique.values()];
    snapshotTotal.value = snapshots.total;
    snapshotOffset.value = (snapshots.offset ?? requestOffset) + (snapshots.limit ?? requestLimit);
  } catch (error) {
    snapshotError.value = resolveLocalizedErrorMessage(t, error, t('build.jobs.create.snapshotLoadFailed'));
  } finally {
    snapshotLoading.value = false;
  }
}

function onArchiveChange(event: Event) {
  archiveFile.value = (event.target as HTMLInputElement).files?.[0];
}

async function loadRuntimeTargets() {
  runtimeTargetLoading.value = true;
  runtimeTargetError.value = '';
  try {
    const targets = await getBuildRuntimeTargets();
    runtimeTargetOptions.value = (targets.items ?? []).map((item) => ({
      label: item.display_name,
      value: item.target_id,
    }));
  } catch (error) {
    runtimeTargetError.value = resolveLocalizedErrorMessage(t, error, t('build.jobs.create.runtimeTargetLoadFailed'));
  } finally {
    runtimeTargetLoading.value = false;
  }
}

async function loadBuilderPools() {
  builderPoolLoading.value = true;
  builderPoolError.value = '';
  try {
    const pools = await getBuildBuilderPools();
    builderPools.value = pools.items ?? [];
  } catch (error) {
    builderPoolError.value = resolveLocalizedErrorMessage(t, error, t('build.jobs.create.builderPoolLoadFailed'));
  } finally {
    builderPoolLoading.value = false;
  }
}

async function loadRegistryDestinations() {
  destinationLoading.value = true;
  destinationError.value = '';
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

// 构建资源模式只响应用户显式选择；切换时仅清理与新模式不兼容的派生字段。
watch(selectionMode, (mode) => {
  if (mode === 'pool') {
    form.value.runtime_target_id = undefined;
  } else {
    form.value.builder_pool_id = undefined;
    form.value.driver = BUILD_DRIVER_REF;
    form.value.platforms = ['linux/amd64'];
  }
});
watch(sourceMode, (mode) => {
  if (mode === 'upload') form.value.input_snapshot_id = '';
  else archiveFile.value = undefined;
  idempotencyKey = undefined;
  idempotencyPayload = undefined;
});
watch(
  () => form.value.platforms,
  (platforms) => {
    if (selectionMode.value === 'target') {
      if (platforms?.some((platform) => platform !== 'linux/amd64')) {
        form.value.platforms = ['linux/amd64'];
      }
      form.value.driver = BUILD_DRIVER_REF;
      return;
    }
    if ((platforms?.length ?? 0) > 1 || platforms?.includes('linux/arm64')) {
      form.value.driver = BUILD_MULTI_PLATFORM_DRIVER_REF;
    }
  },
  { deep: true },
);
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
const MAX_ARCHIVE_BYTES = 100 * 1024 * 1024;
const rules = computed(() => ({
  input_snapshot_id:
    sourceMode.value === 'reuse' ? [{ required: true, message: t('build.jobs.create.snapshotRequired') }] : [],
  runtime_target_id:
    selectionMode.value === 'target'
      ? [{ required: true, min: 1, message: t('build.jobs.create.runtimeTargetRequired') }]
      : [],
  builder_pool_id:
    selectionMode.value === 'pool' ? [{ required: true, message: t('build.jobs.create.builderPoolRequired') }] : [],
  platforms: [
    { required: true, min: 1, message: t('build.jobs.create.platformsRequired') },
    {
      validator: (platforms: unknown) =>
        !Array.isArray(platforms) ||
        (!platforms.includes('linux/arm64') && platforms.length < 2) ||
        form.value.driver === BUILD_MULTI_PLATFORM_DRIVER_REF,
      message: t('build.jobs.create.arm64BuildxRequired'),
    },
  ],
  'destination.connection_ref': [{ required: true, message: t('build.jobs.create.registryRequired') }],
  'destination.repository_ref': [{ required: true, message: t('build.jobs.create.repositoryRequired') }],
  'destination.reference': [{ required: true, message: t('build.jobs.create.tagRequired') }],
}));

function openRegistries() {
  void router.push(REGISTRY_ROUTE_PATH.LIST);
}
function returnToJobs() {
  void router.push(BUILD_ROUTE_PATH.JOBS);
}
async function submit({ validateResult }: SubmitContext) {
  if (validateResult !== true) return;
  if (sourceMode.value === 'upload' && !archiveFile.value) {
    messageTheme.value = 'error';
    message.value = t('build.jobs.create.archiveRequired');
    return;
  }
  if (archiveFile.value && archiveFile.value.size > MAX_ARCHIVE_BYTES) {
    messageTheme.value = 'error';
    message.value = t('build.jobs.create.archiveTooLarge');
    return;
  }
  submitting.value = true;
  message.value = '';
  try {
    if (sourceMode.value === 'upload' && archiveFile.value) {
      const snapshot = await uploadBuildInputSnapshot(archiveFile.value);
      form.value.input_snapshot_id = snapshot.snapshot_id;
    }
    const payload = { ...form.value };
    const payloadSnapshot = JSON.stringify(payload);
    if (idempotencyPayload !== payloadSnapshot) {
      idempotencyPayload = payloadSnapshot;
      idempotencyKey = createIdempotencyKey();
    }
    const currentIdempotencyKey = idempotencyKey ?? createIdempotencyKey();
    idempotencyKey = currentIdempotencyKey;
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
  min-width: 0;
  width: 100%;
}

.build-create-page__surface {
  margin-inline: auto;
  max-width: 1040px;
  width: 100%;
}

.build-create-page__form {
  display: flex;
  flex-direction: column;
  min-width: 0;
  width: 100%;
}

.build-create-page__section {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.build-create-page__section + .build-create-page__section {
  border-top: 1px solid var(--td-component-stroke);
  margin-top: var(--graft-density-gap-20);
  padding-top: var(--graft-density-gap-20);
}

.build-create-page__section-header {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
}

.build-create-page__field-hint {
  color: var(--td-text-color-secondary);
  margin: var(--graft-density-gap-8) 0 0;
}

.build-create-page__file-input {
  display: block;
  max-width: 100%;
}

.build-create-page__section-header h2,
.build-create-page__section-header p {
  margin: 0;
}

.build-create-page__section-header h2 {
  color: var(--td-text-color-primary);
  font-size: var(--td-font-size-title-medium);
  line-height: var(--td-line-height-title-medium);
}

.build-create-page__section-header p {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-medium);
  line-height: var(--td-line-height-body-medium);
}

.build-create-page__grid {
  display: grid;
  gap: var(--graft-density-gap-16) var(--graft-density-gap-20);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  min-width: 0;
}

.build-create-page__field--full {
  grid-column: 1 / -1;
}

.build-create-page__section-feedback {
  display: grid;
  gap: var(--graft-density-gap-8);
}

.build-create-page__section-feedback:empty {
  display: none;
}

.build-create-page__submission-feedback {
  margin-top: var(--graft-density-gap-20);
}

.build-create-page__actions {
  border-top: 1px solid var(--td-component-stroke);
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  justify-content: flex-end;
  margin-top: var(--graft-density-gap-20);
  padding-top: var(--graft-density-gap-20);
}

@media (width <= 768px) {
  .build-create-page__grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .build-create-page__field--full {
    grid-column: auto;
  }

  .build-create-page__actions {
    align-items: stretch;
    flex-direction: column-reverse;
  }

  .build-create-page__action {
    width: 100%;
  }
}
</style>
