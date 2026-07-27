<template>
  <div class="volume-removal-confirm">
    <t-alert theme="error" :message="t('container.volume.removal.risk')" />
    <div class="volume-removal-confirm__items graft-scrollbar">
      <dl v-for="candidate in candidates" :key="candidate.name" class="volume-removal-confirm__item">
        <div>
          <dt>{{ t('container.volume.columns.name') }}</dt>
          <dd>{{ candidate.name }}</dd>
        </div>
        <div>
          <dt>{{ t('container.volume.columns.size') }}</dt>
          <dd>{{ formatBytes(candidate.sizeBytes, t('container.volume.notCollected')) }}</dd>
        </div>
        <div>
          <dt>{{ t('container.volume.columns.mountedContainers') }}</dt>
          <dd>
            {{
              candidate.containerNames.length
                ? candidate.containerNames.join(', ')
                : t('container.volume.removal.noContainers')
            }}
          </dd>
        </div>
      </dl>
    </div>
    <t-input
      v-if="confirmationName"
      :value="typedName"
      :placeholder="confirmationName"
      @change="emit('update:typedName', String($event))"
    />
    <t-checkbox v-if="forceRequired" :checked="force" @change="emit('update:force', Boolean($event))">
      {{ t('container.volume.actions.force') }}
    </t-checkbox>
  </div>
</template>
<script setup lang="ts">
// 删除确认内容统一呈现资源事实和不可恢复风险，供单个、批量与清理流程复用。
import { formatBytes } from '@/shared/observability';

import type { VolumeRemovalCandidate } from '../shared/volume-removal';

type Translate = (key: string, named?: Record<string, unknown>) => string;

defineProps<{
  candidates: VolumeRemovalCandidate[];
  confirmationName?: string;
  force: boolean;
  forceRequired: boolean;
  t: Translate;
  typedName: string;
}>();

const emit = defineEmits<{
  'update:force': [value: boolean];
  'update:typedName': [value: string];
}>();
</script>
<style scoped lang="less">
.volume-removal-confirm {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.volume-removal-confirm__items {
  display: grid;
  gap: var(--graft-density-gap-8);
  max-block-size: 16rem;
  overflow-y: auto;
}

.volume-removal-confirm__item {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-small);
  display: grid;
  gap: var(--graft-density-gap-6);
  margin: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

.volume-removal-confirm__item > div {
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: minmax(5rem, auto) minmax(0, 1fr);
}

.volume-removal-confirm dt {
  color: var(--td-text-color-secondary);
}

.volume-removal-confirm dd {
  margin: 0;
  overflow-wrap: anywhere;
}
</style>
