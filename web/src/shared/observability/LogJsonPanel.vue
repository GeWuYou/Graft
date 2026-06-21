<template>
  <section class="log-json-panel">
    <t-card :bordered="true" size="small">
      <t-collapse v-model:value="expandedPanels" borderless expand-icon-placement="right">
        <t-collapse-panel value="json">
          <template #header>
            <div class="log-json-panel__header">
              <strong class="log-json-panel__title">{{ title }}</strong>
            </div>
          </template>
          <template #headerRightContent>
            <t-space size="var(--graft-density-gap-8)" align="center">
              <t-button v-if="!isEmpty" size="small" theme="default" variant="text" @click.stop="copyJson">
                {{ copyLabel }}
              </t-button>
              <t-button size="small" theme="default" variant="text" @click.stop="toggleExpanded">
                {{ currentToggleLabel }}
              </t-button>
            </t-space>
          </template>
          <div v-if="isEmpty" class="log-json-panel__empty">{{ emptyText }}</div>
          <pre v-else class="log-json-panel__code">{{ formattedJson }}</pre>
        </t-collapse-panel>
      </t-collapse>
    </t-card>
  </section>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, ref } from 'vue';

import { copyText } from './copy';

const props = defineProps<{
  title: string;
  expandLabel: string;
  collapseLabel: string;
  copyLabel: string;
  copySuccessLabel: string;
  copyFailLabel: string;
  emptyText: string;
  value: unknown;
  defaultExpanded?: boolean;
}>();

const expandedPanels = ref<Array<string | number>>(props.defaultExpanded === false ? [] : ['json']);

const isEmpty = computed(() => {
  const value = props.value;
  if (value === null || value === undefined || value === '') {
    return true;
  }

  if (typeof value === 'object') {
    return Object.keys(value as Record<string, unknown>).length === 0;
  }

  return false;
});

const formattedJson = computed(() => {
  if (isEmpty.value) {
    return '';
  }

  try {
    return JSON.stringify(props.value, null, 2);
  } catch {
    return String(props.value);
  }
});

const isExpanded = computed(() => expandedPanels.value.includes('json'));
const currentToggleLabel = computed(() => (isExpanded.value ? props.collapseLabel : props.expandLabel));

function toggleExpanded() {
  expandedPanels.value = isExpanded.value ? [] : ['json'];
}

async function copyJson() {
  try {
    const copied = await copyText(formattedJson.value);
    if (!copied) {
      MessagePlugin.error(props.copyFailLabel);
      return;
    }
    MessagePlugin.success(props.copySuccessLabel);
  } catch {
    MessagePlugin.error(props.copyFailLabel);
  }
}
</script>
<style scoped lang="less">
.log-json-panel {
  display: flex;
  flex-direction: column;
}

.log-json-panel__title {
  margin: 0;
}

.log-json-panel__header {
  align-items: center;
  display: flex;
  min-width: 0;
  width: 100%;
}

.log-json-panel__empty,
.log-json-panel__code {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  margin: 0;
  max-height: min(56vh, 560px);
  overflow: auto;
  overflow-wrap: anywhere;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
  white-space: pre-wrap;
  width: 100%;
}

.log-json-panel__empty {
  color: var(--td-text-color-secondary);
}
</style>
