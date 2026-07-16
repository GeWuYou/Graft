<template>
  <t-popup
    v-model:visible="detailVisible"
    attach="body"
    destroy-on-close
    placement="left-top"
    show-arrow
    trigger="click"
    overlay-class-name="project-lifecycle-help-popup"
  >
    <template #content>
      <div class="project-lifecycle-help">
        <div class="project-lifecycle-help__header">
          <strong>{{ title }}</strong>
          <t-tag size="small" theme="default" variant="light-outline">
            {{ recommendationLabel }}
          </t-tag>
        </div>
        <section class="project-lifecycle-help__section">
          <label>{{ t('project.detail.lifecycle.help.common.sections.effect') }}</label>
          <p>{{ t(`${definition.detailKeyPrefix}.effect`) }}</p>
        </section>
        <section class="project-lifecycle-help__section">
          <label>{{ t('project.detail.lifecycle.help.common.sections.command') }}</label>
          <div class="project-lifecycle-help__command-list">
            <code v-for="command in commandExamples" :key="command">{{ command }}</code>
          </div>
        </section>
        <section class="project-lifecycle-help__section">
          <label>{{ t('project.detail.lifecycle.help.common.sections.scenarios') }}</label>
          <p>{{ t(`${definition.detailKeyPrefix}.scenarios`) }}</p>
        </section>
        <section class="project-lifecycle-help__section">
          <label>{{ t('project.detail.lifecycle.help.common.sections.risks') }}</label>
          <p>{{ t(`${definition.detailKeyPrefix}.risks`) }}</p>
        </section>
        <section class="project-lifecycle-help__section">
          <label>{{ t('project.detail.lifecycle.help.common.sections.recommendation') }}</label>
          <p>{{ t(`${definition.detailKeyPrefix}.recommendation`) }}</p>
        </section>
      </div>
    </template>
    <t-tooltip
      attach="body"
      :content="tooltipContent"
      :disabled="detailVisible"
      placement="top"
      show-arrow
      :visible="tooltipVisible"
    >
      <button
        type="button"
        class="project-lifecycle-help-trigger"
        :aria-expanded="detailVisible ? 'true' : 'false'"
        :aria-label="ariaLabel"
        @mouseenter="setTooltipVisible(true)"
        @mouseleave="setTooltipVisible(false)"
        @focus="setTooltipVisible(true)"
        @blur="setTooltipVisible(false)"
        @click.stop
        @keydown.stop
      >
        <info-circle-icon class="project-lifecycle-help-trigger__icon" />
      </button>
    </t-tooltip>
  </t-popup>
</template>
<script setup lang="ts">
import { InfoCircleIcon } from 'tdesign-icons-vue-next';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import type { LifecycleHelpDefinition } from '../shared/lifecycle-help';
import type { ApplicationLifecycleConfigurationDraft } from '../types/project';

const props = defineProps<{
  definition: LifecycleHelpDefinition;
  draft: ApplicationLifecycleConfigurationDraft;
}>();

const { t } = useI18n();

const detailVisible = ref(false);
const tooltipVisible = ref(false);

const title = computed(() => t(props.definition.titleKey));
const tooltipContent = computed(() => t(props.definition.tooltipKey));
const ariaLabel = computed(() =>
  t('project.detail.lifecycle.help.common.ariaLabel', {
    item: title.value,
  }),
);
const recommendationLabel = computed(() =>
  t(`project.detail.lifecycle.help.common.tags.${props.definition.recommendation}`),
);
const commandExamples = computed(() =>
  typeof props.definition.commandExample === 'function'
    ? props.definition.commandExample(props.draft)
    : props.definition.commandExample,
);

watch(detailVisible, (visible) => {
  if (visible) {
    tooltipVisible.value = false;
  }
});

function setTooltipVisible(visible: boolean) {
  if (detailVisible.value) {
    tooltipVisible.value = false;
    return;
  }
  tooltipVisible.value = visible;
}
</script>
<style scoped lang="less">
.project-lifecycle-help-trigger {
  align-items: center;
  appearance: none;
  background: transparent;
  border: 0;
  color: var(--td-text-color-placeholder);
  cursor: help;
  display: inline-flex;
  flex: 0 0 auto;
  height: 18px;
  justify-content: center;
  padding: 0;
  transition: color 0.2s ease;
  width: 18px;
}

.project-lifecycle-help-trigger:hover,
.project-lifecycle-help-trigger:focus-visible {
  color: var(--td-text-color-secondary);
  outline: none;
}

.project-lifecycle-help-trigger__icon {
  font-size: var(--td-font-size-body-medium);
}

.project-lifecycle-help {
  color: var(--td-text-color-primary);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  max-width: min(420px, calc(100vw - 32px));
  padding: var(--graft-density-gap-4) 0;
}

.project-lifecycle-help__header {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
}

.project-lifecycle-help__header strong {
  font: var(--td-font-body-medium);
}

.project-lifecycle-help__section {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
}

.project-lifecycle-help__section label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-lifecycle-help__section p {
  font: var(--td-font-body-small);
  line-height: 1.6;
  margin: 0;
}

.project-lifecycle-help__command-list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
}

.project-lifecycle-help__command-list code {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-small);
  color: var(--td-text-color-primary);
  display: block;
  font: var(--td-font-body-small);
  overflow-wrap: anywhere;
  padding: var(--graft-density-gap-6) var(--graft-density-gap-8);
}
</style>
