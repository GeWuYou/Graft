<template>
  <section class="project-import-lifecycle-review">
    <section class="project-import-lifecycle-review__scope">
      <div>
        <h2>{{ t('project.import.lifecycle.managedServicesTitle') }}</h2>
        <p>{{ t('project.import.lifecycle.managedServicesDescription') }}</p>
      </div>
      <div class="project-import-lifecycle-review__scope-actions">
        <t-button size="small" variant="outline" @click="selectAllServices">
          {{ t('project.import.lifecycle.selectAllServices') }}
        </t-button>
        <t-button size="small" variant="text" @click="clearServices">
          {{ t('project.import.lifecycle.clearServices') }}
        </t-button>
      </div>
      <t-checkbox-group v-model="draft.managed_service_names">
        <div class="project-import-lifecycle-review__services">
          <t-checkbox v-for="service in serviceOptions" :key="service.name" :value="service.name">
            <span>{{ service.name }}</span>
            <small v-if="service.depends_on.length">
              {{ t('project.import.lifecycle.dependsOn', { services: service.depends_on.join(', ') }) }}
            </small>
          </t-checkbox>
        </div>
      </t-checkbox-group>
      <t-alert
        v-if="missingDependencies.length"
        theme="warning"
        :message="t('project.import.lifecycle.dependencyWarning', { dependencies: missingDependencies.join(', ') })"
      />
      <t-alert
        v-if="!draft.managed_service_names.length"
        theme="error"
        :message="t('project.import.lifecycle.serviceRequired')"
      />
    </section>
    <project-lifecycle-configuration-step
      v-model:draft="draft"
      :title="t('project.import.lifecycle.title')"
      :description="t('project.import.lifecycle.description')"
      :authority-message="t('project.import.lifecycle.authorityHint')"
      :configuration-title="t('project.import.lifecycle.configurationTitle')"
      :back-label="t('project.import.actions.backToInspect')"
      :refresh-label="t('project.import.actions.refreshInspect')"
      :refresh-loading="inspectionRefreshLoading"
      :continue-disabled="!draft.managed_service_names.length"
      :continue-label="t('project.import.actions.confirmLifecycle')"
      @back="$emit('back')"
      @refresh="$emit('refresh')"
      @continue="$emit('confirm')"
    />
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { useApplicationPageContext } from '../shared/page-context';
import type { ApplicationImportServiceOption } from '../types/import';
import type { ApplicationLifecycleConfigurationDraft } from '../types/project';
import ProjectLifecycleConfigurationStep from './ProjectLifecycleConfigurationStep.vue';

defineOptions({ name: 'ApplicationImportLifecycleReview' });

// 导入页适配通用生命周期步骤的文案和确认事件，最终请求参数仍由导入流程持有。

const draft = defineModel<ApplicationLifecycleConfigurationDraft>('draft', { required: true });
const props = defineProps<{ inspectionRefreshLoading?: boolean; serviceOptions: ApplicationImportServiceOption[] }>();
defineEmits<{ (event: 'back'): void; (event: 'confirm'): void; (event: 'refresh'): void }>();

const { t } = useApplicationPageContext();
const missingDependencies = computed(() => {
  const selected = new Set(draft.value.managed_service_names);
  return Array.from(
    new Set(
      props.serviceOptions
        .filter((service) => selected.has(service.name))
        .flatMap((service) => service.depends_on)
        .filter((dependency) => !selected.has(dependency)),
    ),
  ).sort();
});

function selectAllServices() {
  draft.value.managed_service_names = props.serviceOptions.map((service) => service.name);
}

function clearServices() {
  draft.value.managed_service_names = [];
}
</script>
<style scoped>
.project-import-lifecycle-review,
.project-import-lifecycle-review__scope {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.project-import-lifecycle-review__scope h2,
.project-import-lifecycle-review__scope p {
  margin: 0;
}

.project-import-lifecycle-review__scope p,
.project-import-lifecycle-review__services small {
  color: var(--td-text-color-secondary);
}

.project-import-lifecycle-review__scope-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.project-import-lifecycle-review__services {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
}

.project-import-lifecycle-review__services :deep(.t-checkbox__label) {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
</style>
