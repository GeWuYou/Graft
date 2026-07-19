<template>
  <div class="docker-volume-detail" data-page-type="list-form-detail">
    <management-page-header
      :title="volume?.name || t('container.volume.detail.title')"
      :description="t('container.volume.detail.description')"
    >
      <template #meta
        ><t-space v-if="volume" size="small"
          ><t-tag :theme="usageTheme" variant="light-outline">{{ usageLabel }}</t-tag
          ><t-tag variant="light-outline">{{ volume.driver }}</t-tag
          ><t-tag variant="light-outline">{{ volume.scope }}</t-tag></t-space
        ></template
      >
    </management-page-header>
    <t-loading :loading="loading"
      ><t-alert v-if="error" theme="error" :message="error" /><t-descriptions v-else-if="volume" bordered :column="2">
        <t-descriptions-item :label="t('container.volume.columns.name')">{{ volume.name }}</t-descriptions-item
        ><t-descriptions-item :label="t('container.volume.columns.driver')">{{ volume.driver }}</t-descriptions-item
        ><t-descriptions-item :label="t('container.volume.columns.scope')">{{ volume.scope }}</t-descriptions-item
        ><t-descriptions-item :label="t('container.volume.columns.usage')">{{ usageLabel }}</t-descriptions-item
        ><t-descriptions-item :label="t('container.volume.columns.size')">{{
          formatBytes(volume.size_bytes, t('container.volume.unavailable'))
        }}</t-descriptions-item
        ><t-descriptions-item :label="t('container.volume.columns.createdAt')">{{
          formatTime(volume.created_at)
        }}</t-descriptions-item
        ><t-descriptions-item :label="t('container.volume.detail.labels')" :span="2"
          ><t-space break-line
            ><t-tag v-for="(value, key) in volume.labels || {}" :key="key" variant="light-outline"
              >{{ key }}={{ value }}</t-tag
            ><span v-if="!Object.keys(volume.labels || {}).length">{{
              t('container.volume.detail.noLabels')
            }}</span></t-space
          ></t-descriptions-item
        >
      </t-descriptions></t-loading
    >
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import { ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatBytes, formatLocaleDateTime } from '@/shared/observability';

import { type DockerVolumeDetail, getDockerVolume } from '../../api/container';
const { locale, t } = useI18n();
const route = useRoute();
const volume = ref<DockerVolumeDetail | null>(null);
const loading = ref(false);
const error = ref('');
const volumeId = computed(() => String(route.params.id || ''));
const usageLabel = computed(() => {
  const count = volume.value?.reference_count;
  return count === null || count === undefined
    ? t('container.volume.usage.unknown')
    : count > 0
      ? t('container.volume.usage.used', { count })
      : t('container.volume.usage.unused');
});
const usageTheme = computed(() => (volume.value?.reference_count ? 'primary' : 'default'));
onMounted(() => void load());
watch(volumeId, () => void load());
async function load() {
  if (!volumeId.value) return;
  loading.value = true;
  error.value = '';
  try {
    volume.value = await getDockerVolume(volumeId.value);
  } catch (cause) {
    volume.value = null;
    error.value = resolveLocalizedErrorMessage(t, cause, t('container.volume.detail.loadFailed'));
  } finally {
    loading.value = false;
  }
}
function formatTime(value?: string) {
  return value ? formatLocaleDateTime(value, locale) : t('container.volume.unavailable');
}
</script>
<style scoped lang="less">
.docker-volume-detail {
  display: grid;
  gap: var(--td-comp-margin-xl);
}
</style>
