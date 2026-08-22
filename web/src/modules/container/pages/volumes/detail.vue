<template>
  <resource-detail-layout
    :title="volume?.name || t('container.volume.detail.title')"
    :back-label="t('container.detail.back')"
    presentation="page"
    @update:visible="returnToList"
  >
    <template v-if="volume" #actions>
      <t-tag :theme="statusPresentation.theme" size="small" variant="light">{{ statusPresentation.label }}</t-tag>
    </template>
    <div class="docker-volume-detail-page">
      <div v-if="loading" class="docker-volume-detail-page__state"><t-loading :loading="true" size="large" /></div>
      <t-alert v-else-if="error" theme="error" :message="error" />
      <volume-detail-content
        v-else-if="volume"
        :can-remove="canRemove"
        surface="page"
        :volume="volume"
        @open-container="openContainerReference"
        @remove="confirmRemove"
      />
      <div v-else class="docker-volume-detail-page__state">
        <t-empty
          :title="t('container.volume.detail.emptyTitle')"
          :description="t('container.volume.detail.emptyDescription')"
        />
      </div>
    </div>
  </resource-detail-layout>
</template>
<script setup lang="ts">
// 独立详情路由为窄屏提供完整页面，同时保持桌面直接访问时可用。
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { CONTAINER_PERMISSION_CODE } from '@/contracts/generated/modules/container';
import ResourceDetailLayout from '@/shared/components/responsive/ResourceDetailLayout.vue';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { usePermissionStore } from '@/store';

import { type DockerVolumeDetail, getDockerVolume, removeDockerVolume } from '../../api/container';
import VolumeDetailContent from '../../components/VolumeDetailContent.vue';
import { CONTAINER_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { getDockerVolumeStatusPresentation } from '../../shared/volume-presentation';
import { openVolumeRemovalConfirmation } from '../../shared/volume-removal';

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const permissionStore = usePermissionStore();
const volume = ref<DockerVolumeDetail | null>(null);
const loading = ref(true);
const error = ref('');
const canRemove = computed(() => permissionStore.hasPermission(CONTAINER_PERMISSION_CODE.VOLUME_REMOVE));
const statusPresentation = computed(() =>
  volume.value
    ? getDockerVolumeStatusPresentation(t, volume.value.relationship_status)
    : { label: '', theme: 'default' as const },
);

watch(
  () => String(route.params.name ?? ''),
  (volumeName) => void loadVolume(volumeName),
  { immediate: true },
);

async function loadVolume(volumeName: string) {
  loading.value = true;
  error.value = '';
  volume.value = null;
  try {
    const result = await getDockerVolume(volumeName);
    if (volumeName === String(route.params.name ?? '')) {
      volume.value = result;
    }
  } catch (cause) {
    if (volumeName === String(route.params.name ?? '')) {
      error.value = resolveLocalizedErrorMessage(t, cause, t('container.volume.detail.loadFailed'));
    }
  } finally {
    if (volumeName === String(route.params.name ?? '')) {
      loading.value = false;
    }
  }
}

function returnToList() {
  void router.push({ name: CONTAINER_BOOTSTRAP_ROUTE.VOLUMES.pageRouteName });
}

function openContainerReference(id: string) {
  void router.push({ name: CONTAINER_BOOTSTRAP_ROUTE.DETAIL.pageRouteName, params: { id }, query: { tab: 'storage' } });
}

function confirmRemove() {
  if (!volume.value || !canRemove.value) return;
  const candidate = volume.value;
  openVolumeRemovalConfirmation({
    candidates: [
      {
        containerNames: candidate.container_references.map((reference) => reference.name || reference.id),
        name: candidate.name,
        sizeBytes: candidate.size_bytes,
      },
    ],
    confirmationName: candidate.name,
    forceRequired: candidate.relationship_status !== 'unused',
    header: t('container.volume.actions.confirmTitle'),
    confirmLabel: t('container.volume.actions.remove'),
    t,
    onConfirm: async (force) => {
      try {
        const receipt = await removeDockerVolume(candidate.name, { force });
        void receipt.task_id;
        returnToList();
        return true;
      } catch (cause) {
        error.value = resolveLocalizedErrorMessage(t, cause, t('container.volume.actions.removeFailed'));
        return false;
      }
    },
  });
}
</script>
<style scoped lang="less">
.docker-volume-detail-page {
  min-height: 0;
}

.docker-volume-detail-page__state {
  display: grid;
  min-height: 16rem;
  place-items: center;
}
</style>
