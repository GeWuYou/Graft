<template>
  <t-select
    v-model="selectedSource"
    class="management-toolbar__select"
    clearable
    :placeholder="t('container.resourceContext.source')"
  >
    <t-option
      v-for="sourceOption in resourceSources"
      :key="sourceOption"
      :value="sourceOption"
      :label="sourceLabel(sourceOption)"
    />
  </t-select>
  <t-input
    v-model="project"
    class="management-toolbar__select"
    clearable
    :placeholder="t('container.resourceContext.project')"
    @enter="emit('apply')"
  />
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { components } from '@/contracts/openapi/generated/schema';

import { getDockerResourceSourceLabel } from '../shared/resource-presentation';

type DockerResourceSource = components['schemas']['docker-resource-source'];

// 统一网络和卷列表的来源/项目筛选控件，页面仍负责各自的查询提交时机和其他资源专属条件。
const props = defineProps<{
  composeProject: string;
  source: DockerResourceSource | '';
}>();
const emit = defineEmits<{
  apply: [];
  'update:composeProject': [value: string];
  'update:source': [value: DockerResourceSource | ''];
}>();
const { t } = useI18n();
const resourceSources = ['compose', 'docker_default', 'docker', 'managed', 'imported', 'unknown'] as const;
const selectedSource = computed({
  get: () => props.source,
  set: (value: DockerResourceSource | '') => emit('update:source', value),
});
const project = computed({
  get: () => props.composeProject,
  set: (value: string) => emit('update:composeProject', value),
});

function sourceLabel(source: DockerResourceSource) {
  return getDockerResourceSourceLabel(t, source);
}
</script>
