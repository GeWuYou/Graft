<template>
  <t-card size="small">
    <template #title>{{ title }}</template>
    <template #actions>
      <t-tooltip :content="copyLabel">
        <t-button shape="square" size="small" variant="text" :aria-label="copyLabel" @click="$emit('copy', sha256)">
          <template #icon><copy-icon /></template>
        </t-button>
      </t-tooltip>
    </template>
    <strong class="backup-detail__artifact-size">{{ formatBytes(sizeBytes) }}</strong>
    <p class="backup-detail__artifact-label">SHA-256</p>
    <t-tooltip :content="sha256">
      <code class="backup-detail__checksum">{{ truncateChecksum(sha256) }}</code>
    </t-tooltip>
  </t-card>
</template>
<script setup lang="ts">
import { CopyIcon } from 'tdesign-icons-vue-next';

import { formatBytes } from '@/shared/observability';

defineOptions({ name: 'BackupArtifactCard' });

defineProps<{
  copyLabel: string;
  sha256: string;
  sizeBytes: number;
  title: string;
}>();

defineEmits<{
  copy: [checksum: string];
}>();

function truncateChecksum(value: string) {
  return value.length > 24 ? `${value.slice(0, 12)}...${value.slice(-8)}` : value;
}
</script>
