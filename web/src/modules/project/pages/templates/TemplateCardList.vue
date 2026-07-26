<template>
  <section class="template-card-list" :aria-busy="loading">
    <div v-if="loading" class="template-card-list__items" data-testid="template-card-skeletons">
      <article v-for="item in skeletonCount" :key="item" class="template-card-list__skeleton">
        <t-skeleton animation="gradient" :row-col="skeletonRows" />
      </article>
    </div>

    <div v-else-if="templates.length" class="template-card-list__items" data-testid="template-card-items">
      <article v-for="template in templates" :key="template.template_id" class="template-card-list__card">
        <button class="template-card-list__name" type="button" @click="emit('open', template)">
          {{ template.display_name }}
        </button>
        <p class="template-card-list__description">{{ template.description || noDescriptionLabel }}</p>

        <div class="template-card-list__metadata">
          <t-tag variant="light-outline">{{ adapterLabel }}</t-tag>
          <t-tag :theme="statusTheme(template)" variant="light-outline">{{ statusLabel(template) }}</t-tag>
        </div>
        <p class="template-card-list__updated">{{ updatedAtLabel(template) }}</p>

        <div class="template-card-list__actions">
          <table-action-menu :actions="actionsFor(template)" @action="emit('action', $event, template)" />
        </div>
      </article>
    </div>

    <t-empty v-else :title="emptyTitle" :description="emptyDescription" />
  </section>
</template>
<script setup lang="ts">
import type { TagProps } from 'tdesign-vue-next';

import { TableActionMenu } from '@/shared/components/management';

import type { ApplicationTemplate } from '../../types/project';

type TemplateAction = {
  label: string;
  value: string;
};

/** 模板卡片列表只负责窄宽度呈现；查询、权限动作和导航仍由模板管理页统一拥有。 */
defineProps<{
  actionsFor: (template: ApplicationTemplate) => TemplateAction[];
  adapterLabel: string;
  emptyDescription: string;
  emptyTitle: string;
  noDescriptionLabel: string;
  statusLabel: (template: ApplicationTemplate) => string;
  statusTheme: (template: ApplicationTemplate) => TagProps['theme'];
  templates: ApplicationTemplate[];
  updatedAtLabel: (template: ApplicationTemplate) => string;
  loading: boolean;
}>();

const emit = defineEmits<{
  action: [action: string, template: ApplicationTemplate];
  open: [template: ApplicationTemplate];
}>();

const skeletonCount = 3;
const skeletonRows = [
  { width: '42%', height: '20px' },
  { width: '78%', height: '16px', marginTop: '12px' },
  [{ width: '88px', height: '24px', marginTop: '18px' }, { width: '76px', height: '24px', marginLeft: '8px' }],
  { width: '46%', height: '16px', marginTop: '14px' },
  [{ width: '88px', height: '44px', marginTop: '18px' }, { width: '44px', height: '44px', marginLeft: '8px' }],
];
</script>
<style scoped lang="less">
.template-card-list {
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.template-card-list__items {
  display: grid;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.template-card-list__card,
.template-card-list__skeleton {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  box-shadow: var(--td-shadow-1);
  min-width: 0;
  padding: var(--graft-density-gap-16);
}

.template-card-list__name {
  background: none;
  border: 0;
  color: var(--td-brand-color);
  cursor: pointer;
  font: var(--td-font-title-medium);
  font-weight: 600;
  max-width: 100%;
  overflow-wrap: anywhere;
  padding: 0;
  text-align: left;
}

.template-card-list__description,
.template-card-list__updated {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  margin: var(--graft-density-gap-8) 0 0;
  overflow-wrap: anywhere;
}

.template-card-list__metadata {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  margin-top: var(--graft-density-gap-16);
}

.template-card-list__actions {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--graft-density-gap-16);
}

.template-card-list__actions :deep(.table-action-menu) {
  justify-content: flex-end;
  width: auto;
}

.template-card-list__actions :deep(.t-button) {
  min-height: 44px;
}

@container (width >= 40rem) {
  .template-card-list__items {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
