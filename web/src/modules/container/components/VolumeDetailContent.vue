<template>
  <div class="volume-detail-content" :data-surface="surface">
    <section class="volume-detail-content__section">
      <h3>{{ t('container.resourceContext.overview') }}</h3>
      <t-card size="small" :bordered="true">
        <dl class="volume-detail-content__overview">
          <div v-if="surface === 'page'">
            <dt>{{ t('container.volume.columns.status') }}</dt>
            <dd>
              <t-tag :theme="statusPresentation.theme" size="small" variant="light">
                {{ statusPresentation.label }}
              </t-tag>
            </dd>
          </div>
          <div>
            <dt>{{ t('container.volume.columns.size') }}</dt>
            <dd>{{ displaySize }}</dd>
          </div>
          <div>
            <dt>{{ t('container.volume.columns.driver') }}</dt>
            <dd>{{ volume.driver || t('container.volume.notCollected') }}</dd>
          </div>
          <div>
            <dt>{{ t('container.volume.columns.createdAt') }}</dt>
            <dd>{{ displayCreatedAt }}</dd>
          </div>
        </dl>
      </t-card>
    </section>

    <section class="volume-detail-content__section">
      <h3>{{ t('container.resourceContext.relations') }}</h3>
      <div v-if="volume.container_references.length" class="volume-detail-content__relations">
        <t-link
          v-for="reference in volume.container_references"
          :key="reference.id"
          theme="primary"
          @click="emit('open-container', reference.id)"
        >
          {{ reference.name || reference.id }}
        </t-link>
      </div>
      <div v-else class="volume-detail-content__empty-relation">
        <span>{{ t('container.volume.detail.noContainers') }}</span>
        <p v-if="safeCleanupCandidate">{{ t('container.volume.detail.safeCleanupHint') }}</p>
      </div>
    </section>

    <section v-if="surface === 'page'" class="volume-detail-content__section">
      <h3>{{ t('container.volume.detail.storage') }}</h3>
      <t-descriptions :column="1" size="small">
        <t-descriptions-item :label="t('container.volume.detail.mountpoint')">
          <span class="volume-detail-content__breakable">
            {{ volume.mountpoint || t('container.volume.notCollected') }}
          </span>
        </t-descriptions-item>
        <t-descriptions-item :label="t('container.volume.detail.actualUsage')">
          {{ displaySize }}
        </t-descriptions-item>
      </t-descriptions>
    </section>

    <section v-else class="volume-detail-content__section">
      <h3>{{ t('container.resourceContext.configuration') }}</h3>
      <t-descriptions :column="1" size="small">
        <t-descriptions-item :label="t('container.volume.columns.driver')">{{ volume.driver }}</t-descriptions-item>
        <t-descriptions-item :label="t('container.volume.columns.scope')">{{ volume.scope }}</t-descriptions-item>
        <t-descriptions-item :label="t('container.resourceContext.metadata')">
          <div v-if="Object.keys(volume.labels || {}).length" class="volume-detail-content__metadata">
            <t-tag v-for="(value, key) in volume.labels" :key="key" size="small" variant="light-outline">
              {{ key }}={{ value }}
            </t-tag>
          </div>
          <span v-else class="volume-detail-content__muted">{{ t('container.volume.detail.noLabels') }}</span>
        </t-descriptions-item>
      </t-descriptions>
    </section>

    <container-danger-zone
      v-if="canRemove"
      :action-label="t('container.volume.actions.remove')"
      :description="t('container.volume.removal.risk')"
      @action="emit('remove')"
    />
  </div>
</template>
<script setup lang="ts">
// 数据卷详情内容只消费 canonical DTO，并按 page/drawer surface 重排同一组资产事实。
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { formatBytes, formatLocaleDateTime } from '@/shared/observability';

import type { DockerVolumeDetail } from '../api/container';
import { getDockerVolumeStatusPresentation, isDockerVolumeSafeCleanupCandidate } from '../shared/volume-presentation';
import ContainerDangerZone from './ContainerDangerZone.vue';

const props = defineProps<{
  canRemove: boolean;
  surface: 'drawer' | 'page';
  volume: DockerVolumeDetail;
}>();
const emit = defineEmits<{ 'open-container': [id: string]; remove: [] }>();
const { locale, t } = useI18n();
const statusPresentation = computed(() => getDockerVolumeStatusPresentation(t, props.volume.relationship_status));
const safeCleanupCandidate = computed(() => isDockerVolumeSafeCleanupCandidate(props.volume.relationship_status));
const displaySize = computed(() => formatBytes(props.volume.size_bytes, t('container.volume.notCollected')));
const displayCreatedAt = computed(() =>
  props.volume.created_at ? formatLocaleDateTime(props.volume.created_at, locale) : t('container.volume.notCollected'),
);
</script>
<style scoped lang="less">
.volume-detail-content {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.volume-detail-content__section h3 {
  font-size: var(--td-font-size-body-large);
  margin: 0 0 var(--graft-density-gap-10);
}

.volume-detail-content__overview {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.volume-detail-content dt {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-small);
}

.volume-detail-content dd {
  font-variant-numeric: tabular-nums;
  margin: var(--graft-density-gap-4) 0 0;
  overflow-wrap: anywhere;
}

.volume-detail-content__relations,
.volume-detail-content__metadata {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.volume-detail-content__empty-relation,
.volume-detail-content__muted {
  color: var(--td-text-color-secondary);
}

.volume-detail-content__empty-relation p {
  color: var(--td-success-color);
  margin: var(--graft-density-gap-6) 0 0;
}

.volume-detail-content__breakable {
  overflow-wrap: anywhere;
}

.volume-detail-content[data-surface='page'] {
  margin: 0 auto;
  max-width: 48rem;
  width: 100%;
}

.volume-detail-content[data-surface='drawer'] .volume-detail-content__overview {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
</style>
