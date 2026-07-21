<template>
  <section class="docker-resource-context-card">
    <h3>{{ t('container.resourceContext.title') }}</h3>
    <dl>
      <div>
        <dt>{{ t('container.resourceContext.runtime') }}</dt>
        <dd>{{ runtimeLabel }}</dd>
      </div>
      <div>
        <dt>{{ t('container.resourceContext.source') }}</dt>
        <dd>
          <t-tag size="small" variant="light-outline">{{ sourceLabel }}</t-tag>
        </dd>
      </div>
      <div v-if="context.runtime_target">
        <dt>{{ t('container.resourceContext.runtimeTarget') }}</dt>
        <dd>{{ context.runtime_target }}</dd>
      </div>
      <div v-if="context.compose_project">
        <dt>{{ t('container.resourceContext.project') }}</dt>
        <dd>{{ context.compose_project }}</dd>
      </div>
      <div v-if="context.compose_resource">
        <dt>{{ resourceLabel }}</dt>
        <dd>{{ context.compose_resource }}</dd>
      </div>
      <div v-if="context.managed_by">
        <dt>{{ t('container.resourceContext.managedBy') }}</dt>
        <dd>{{ context.managed_by }}</dd>
      </div>
    </dl>
  </section>
</template>
<script setup lang="ts">
// Context Card 只呈现服务端规范化的业务上下文，禁止从 labels 或 inspect 数据推导归属信息。
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { components } from '@/contracts/openapi/generated/schema';

import { getDockerResourceSourceLabel } from '../shared/resource-presentation';

const props = withDefaults(
  defineProps<{
    context: components['schemas']['docker-resource-context'];
    resourceKind: 'network' | 'volume';
  }>(),
  {},
);

const { t } = useI18n();
const runtimeLabel = computed(() =>
  t(`container.resourceContext.runtimeValues.${props.context.runtime}`, props.context.runtime),
);
const sourceLabel = computed(() => getDockerResourceSourceLabel(t, props.context.source));
const resourceLabel = computed(() =>
  t(props.resourceKind === 'network' ? 'container.resourceContext.network' : 'container.resourceContext.volume'),
);
</script>
<style scoped lang="less">
.docker-resource-context-card {
  background: var(--graft-card-elevated-bg);
  border: 1px solid var(--graft-card-border-color);
  border-radius: var(--td-radius-medium);
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
}

.docker-resource-context-card h3 {
  font-size: var(--td-font-size-body-large);
  margin: 0 0 var(--td-comp-margin-m);
}

.docker-resource-context-card dl {
  display: grid;
  gap: var(--td-comp-margin-m);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.docker-resource-context-card dt {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-small);
  margin-bottom: var(--td-comp-margin-xs);
}

.docker-resource-context-card dd {
  margin: 0;
  overflow-wrap: anywhere;
}

@media (width <= 560px) {
  .docker-resource-context-card dl {
    grid-template-columns: 1fr;
  }
}
</style>
