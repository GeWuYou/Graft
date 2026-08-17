<template>
  <div class="context-links" role="list">
    <context-link-row
      v-for="entry in visibleLinks"
      :key="entry.key"
      :link="entry"
      @navigate="emit('navigate', $event)"
    />
  </div>
  <t-collapse
    v-if="remainingLinks.length"
    v-model="expandedPanels"
    v-bind="collapseBehavior"
    class="workbench-collapse"
  >
    <t-collapse-panel :value="panelValue">
      <template #header>
        <t-button
          v-bind="overflowTriggerProps"
          @click.stop="toggleOverflow"
          @keydown.enter.prevent.stop="toggleOverflow"
          @keydown.space.prevent.stop="toggleOverflow"
        >
          {{ t('dashboard.workbench.expand.contextLinks', { count: remainingLinks.length }) }}
          <template #suffix>
            <chevron-down-icon :style="{ transform: overflowExpanded ? 'rotate(180deg)' : undefined }" />
          </template>
        </t-button>
      </template>
      <div
        :id="panelContentId"
        class="context-links workbench-collapse__content"
        role="list"
        :data-collapse-content="panelValue"
      >
        <context-link-row
          v-for="entry in remainingLinks"
          :key="entry.key"
          :link="entry"
          @navigate="emit('navigate', $event)"
        />
      </div>
    </t-collapse-panel>
  </t-collapse>
</template>
<script setup lang="ts">
import { ChevronDownIcon, ChevronRightIcon } from 'tdesign-icons-vue-next';
import type { PropType } from 'vue';
import { computed, defineComponent, h, ref, resolveComponent } from 'vue';

import { t } from '@/locales';
import GraftMenuIcon from '@/shared/icons/MenuIcon.vue';

import type { WorkbenchContextLink, WorkbenchContextLinkGroup } from '../../presentation/workbench';
import { resolveDashboardText } from '../widgets/widget-i18n';

// 上下文入口列表只承载贡献源下钻，不参与首页快捷入口的本地排序与访问计数。
const props = withDefaults(
  defineProps<{
    group: WorkbenchContextLinkGroup;
    visibleLimit?: number;
  }>(),
  { visibleLimit: 6 },
);

const emit = defineEmits<{
  navigate: [route: string];
}>();

const expandedPanels = ref<string[]>([]);
const collapseBehavior = {
  borderless: true,
  expandIcon: false,
  expandOnRowClick: false,
} as const;
const visibleLinks = computed(() => props.group.links.slice(0, props.visibleLimit));
const remainingLinks = computed(() => props.group.links.slice(props.visibleLimit));
const panelValue = computed(() => `context-more:${props.group.id}`);
const panelContentId = computed(() => `workbench-context-more-${props.group.id}-content`);
const overflowExpanded = computed(() => expandedPanels.value.includes(panelValue.value));
const overflowTriggerProps = computed(
  () =>
    ({
      block: true,
      theme: 'primary',
      type: 'button',
      variant: 'text',
      'aria-controls': panelContentId.value,
      'aria-expanded': overflowExpanded.value,
      'data-collapse-trigger': panelValue.value,
    }) as const,
);

function toggleOverflow() {
  expandedPanels.value = overflowExpanded.value ? [] : [panelValue.value];
}

function dashboardText(key?: string, fallback?: string, defaultText = '-') {
  return resolveDashboardText(key, fallback, defaultText);
}

const ContextLinkRow = defineComponent({
  name: 'ContextLinkRow',
  props: {
    link: {
      type: Object as PropType<WorkbenchContextLink>,
      required: true,
    },
  },
  emits: {
    navigate: (route: string) => Boolean(route),
  },
  setup(rowProps, { emit: emitRow }) {
    const Tag = resolveComponent('t-tag');
    return () => {
      const link = rowProps.link;
      const description =
        link.descriptionKey || link.descriptionFallback
          ? h('span', dashboardText(link.descriptionKey, link.descriptionFallback, ''))
          : null;
      const badge =
        link.badgeKey || link.badgeFallback
          ? h(
              Tag,
              { theme: 'default', variant: 'light' },
              { default: () => dashboardText(link.badgeKey, link.badgeFallback, '') },
            )
          : null;

      return h(
        'button',
        {
          class: 'context-link',
          type: 'button',
          role: 'listitem',
          disabled: link.disabled,
          'data-context-link-key': link.key,
          onClick: () => emitRow('navigate', link.route),
        },
        [
          h('span', { class: 'context-link__icon' }, [h(GraftMenuIcon, { iconKey: link.iconKey })]),
          h('span', { class: 'context-link__copy' }, [
            h('strong', dashboardText(link.labelKey, link.labelFallback)),
            description,
          ]),
          badge,
          h(ChevronRightIcon),
        ],
      );
    };
  },
});
</script>
<style scoped lang="less">
.workbench-collapse {
  background-color: transparent;
  margin: var(--graft-density-gap-8) 0 0;
}

.workbench-collapse :deep(.t-collapse-panel__body),
.workbench-collapse :deep(.t-collapse-panel__content) {
  background-color: transparent;
}

.workbench-collapse :deep(.t-collapse-panel__content) {
  color: inherit;
  padding: 0;
}

.context-links {
  display: grid;
  gap: var(--graft-density-gap-8);
}

.context-link {
  align-items: center;
  background: transparent;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  box-sizing: border-box;
  color: inherit;
  cursor: pointer;
  display: grid;
  font: inherit;
  gap: var(--graft-density-gap-10);
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  min-height: 52px;
  padding: var(--graft-density-gap-10);
  text-align: left;
  width: 100%;
}

.context-link:hover {
  background: var(--td-bg-color-container-hover);
  border-color: var(--td-border-level-2-color);
}

.context-link:focus-visible {
  border-color: var(--td-brand-color);
  box-shadow: inset 0 0 0 1px var(--td-brand-color);
  outline: none;
}

.context-link:disabled {
  color: var(--td-text-color-disabled);
  cursor: not-allowed;
}

.context-link:disabled:hover {
  background: transparent;
  border-color: var(--td-border-level-1-color);
}

.context-link :deep(.context-link__icon) {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  justify-content: center;
}

.context-link :deep(.context-link__copy) {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
  min-width: 0;
}

.context-link :deep(.context-link__copy strong) {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.context-link :deep(.context-link__copy > span) {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}
</style>
