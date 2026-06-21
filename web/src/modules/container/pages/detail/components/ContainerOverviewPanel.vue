<template>
  <div class="container-overview-panel">
    <div class="container-overview-detail">
      <section v-for="section in sections" :key="section.key" class="container-info-section">
        <div class="container-info-section__title">{{ section.title }}</div>
        <div class="container-info-section__body">
          <div v-for="row in section.rows" :key="row.key" class="container-info-row">
            <span class="container-info-row__label">{{ row.label }}</span>
            <span class="container-info-row__value">
              <copyable-detail-value
                v-if="row.type === 'copy'"
                :code="row.code"
                :copy-label="copyLabel"
                :data-testid="row.testId"
                :display-value="row.displayValue"
                :value="row.copyValue ?? row.displayValue"
                @copy="emit('copy', $event)"
              />
              <t-tag v-else-if="row.type === 'tag'" :theme="row.tagTheme" variant="light-outline">
                {{ row.tagLabel }}
              </t-tag>
              <span v-else-if="row.type === 'ports'">
                <span v-if="row.ports.length" class="container-detail-port-list">
                  <t-tag
                    v-for="port in row.ports"
                    :key="port"
                    class="container-info-row__tag"
                    theme="default"
                    :title="port"
                    variant="light-outline"
                  >
                    {{ port }}
                  </t-tag>
                </span>
                <span v-else class="container-info-row__text" :title="row.emptyLabel">
                  {{ row.emptyLabel }}
                </span>
              </span>
              <t-tooltip v-else :content="row.displayValue" placement="top-left">
                <span class="container-info-row__text" :title="row.displayValue">
                  {{ row.displayValue }}
                </span>
              </t-tooltip>
            </span>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>
<script setup lang="ts">
import CopyableDetailValue from './CopyableDetailValue.vue';
import type { ContainerOverviewInfoSection } from './overview';

withDefaults(
  defineProps<{
    copyLabel: string;
    sections: ContainerOverviewInfoSection[];
  }>(),
  {
    sections: () => [],
  },
);

const emit = defineEmits<{
  copy: [value: string];
}>();
</script>
<style scoped lang="less">
.container-overview-panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
  width: 100%;
}

.container-overview-detail {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
  padding: var(--graft-density-gap-16);
  width: 100%;
}

.container-info-section {
  background: var(--td-bg-color-container);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
  width: 100%;
}

.container-info-section__title {
  background: color-mix(in srgb, var(--td-bg-color-container) 86%, var(--td-bg-color-page));
  border-bottom: 1px solid color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-weight: 600;
  line-height: 22px;
  padding: var(--graft-density-gap-12) var(--graft-density-gap-16);
}

.container-info-section__body {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: var(--graft-density-gap-8) var(--graft-density-gap-16);
}

.container-info-row {
  align-items: center;
  column-gap: var(--graft-density-gap-16);
  display: grid;
  grid-template-columns: 112px minmax(0, 1fr);
  min-height: 36px;
  min-width: 0;
  width: 100%;
}

.container-info-row + .container-info-row {
  border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 30%, transparent);
}

.container-info-row__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  line-height: 20px;
  min-width: 0;
}

.container-info-row__value {
  align-items: center;
  color: var(--td-text-color-primary);
  display: inline-flex;
  font: var(--td-font-body-small);
  font-weight: 500;
  gap: var(--graft-density-gap-6);
  line-height: 22px;
  min-width: 0;
  overflow: hidden;
}

.container-info-row__text {
  color: var(--td-text-color-primary);
  display: inline-block;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-detail-port-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.container-info-row__tag {
  font-family: var(
    --td-font-family-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    monospace
  );
  max-width: 100%;
}

@media (width <= 560px) {
  .container-info-row {
    align-items: flex-start;
    gap: var(--graft-density-gap-4);
    grid-template-columns: 1fr;
    padding: var(--graft-density-gap-8) 0;
  }

  .container-info-row__value {
    width: 100%;
  }
}
</style>
