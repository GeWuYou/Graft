<template>
  <section class="project-import-lifecycle-review">
    <project-import-section-heading
      :description="t('project.import.lifecycle.description')"
      :title="t('project.import.lifecycle.title')"
    />

    <t-alert theme="info" :message="t('project.import.lifecycle.authorityHint')" />

    <div class="project-import-lifecycle-review__grid">
      <t-card :title="t('project.import.lifecycle.configurationTitle')" :bordered="true">
        <div class="project-import-lifecycle-review__content">
          <div class="project-import-lifecycle-review__field-grid">
            <label class="project-import-lifecycle-review__field">
              <span>{{ t('project.detail.lifecycle.profiles') }}</span>
              <t-input v-model="profilesInput" :placeholder="t('project.detail.lifecycle.profilesPlaceholder')" />
            </label>
            <label class="project-import-lifecycle-review__field">
              <span>{{ t('project.detail.lifecycle.additionalArgs') }}</span>
              <t-input
                v-model="draft.additional_args"
                :placeholder="t('project.detail.lifecycle.additionalArgsPlaceholder')"
              />
            </label>
          </div>

          <template v-for="definition in lifecycleSwitchHelpDefinitions" :key="definition.key">
            <div class="project-import-lifecycle-review__option">
              <div>
                <div class="project-import-lifecycle-review__option-title">
                  <span>{{ t(definition.titleKey) }}</span>
                  <lifecycle-help-trigger :definition="definition" :draft="draft" />
                </div>
                <p>{{ t(definition.summaryKey) }}</p>
              </div>
              <t-switch v-model="draft[definition.field]" :aria-label="t(definition.titleKey)" />
            </div>

            <label
              v-if="definition.key === 'waitAfterUp' && waitTimeoutDefinition.visible?.(draft)"
              class="project-import-lifecycle-review__field"
            >
              <span>{{ t(waitTimeoutDefinition.titleKey) }}</span>
              <t-input-number v-model="draft.wait_timeout_seconds" :min="1" :max="3600" :step="1" />
              <small>{{ t(waitTimeoutDefinition.summaryKey) }}</small>
            </label>
          </template>

          <t-alert
            v-if="draft.renew_anon_volumes"
            theme="warning"
            :message="t('project.detail.lifecycle.renewAnonVolumesWarning')"
          />
        </div>
      </t-card>

      <t-card :title="t('project.import.lifecycle.commandPreviewTitle')" :bordered="true">
        <div class="project-import-lifecycle-review__commands">
          <section v-for="command in commandPreviews" :key="command.key" class="project-code-panel">
            <strong>{{ command.title }}</strong>
            <code-block :code="command.preview" lang="shell" wrap />
          </section>
        </div>
      </t-card>
    </div>

    <div class="project-import-step-actions">
      <t-button theme="default" variant="outline" @click="$emit('back')">
        {{ t('project.import.actions.backToInspect') }}
      </t-button>
      <t-button theme="primary" @click="$emit('confirm')">
        {{ t('project.import.actions.confirmLifecycle') }}
      </t-button>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import CodeBlock from '@/shared/components/code/CodeBlock.vue';

import {
  lifecycleDraftProfilesText,
  resolveLifecycleCommandSteps,
  updateLifecycleDraftProfiles,
} from '../shared/lifecycle';
import { lifecycleSwitchHelpDefinitions, lifecycleWaitTimeoutHelpDefinition } from '../shared/lifecycle-help';
import { useProjectPageContext } from '../shared/page-context';
import type { ProjectLifecycleConfigurationDraft } from '../types/project';
import LifecycleHelpTrigger from './LifecycleHelpTrigger.vue';
import ProjectImportSectionHeading from './ProjectImportSectionHeading.vue';

defineOptions({ name: 'ProjectImportLifecycleReview' });

const draft = defineModel<ProjectLifecycleConfigurationDraft>('draft', { required: true });
defineEmits<{ (event: 'back'): void; (event: 'confirm'): void }>();

const { t } = useProjectPageContext();
const waitTimeoutDefinition = lifecycleWaitTimeoutHelpDefinition;
const profilesInput = computed({
  get: () => lifecycleDraftProfilesText(draft.value),
  set: (value: string) => updateLifecycleDraftProfiles(draft.value, value),
});
const commandPreviews = computed(() =>
  (['up', 'stop', 'restart', 'redeploy'] as const).map((key) => ({
    key,
    title: t(`project.detail.lifecycle.generatedCommands.${key}`),
    preview: resolveLifecycleCommandSteps(draft.value, key, { preferClientGenerated: true })
      .map((step) => step.command)
      .join('\n'),
  })),
);
</script>
<style scoped>
.project-import-lifecycle-review,
.project-import-lifecycle-review__content,
.project-import-lifecycle-review__commands,
.project-import-lifecycle-review__field,
.project-import-lifecycle-review__option > div {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.project-import-lifecycle-review {
  gap: var(--graft-density-gap-16);
}

.project-import-lifecycle-review__grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-import-lifecycle-review__field-grid {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-import-lifecycle-review__option {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.project-import-lifecycle-review__option-title {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-6);
}

.project-import-lifecycle-review__option p,
.project-import-lifecycle-review__field small {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.project-import-lifecycle-review__commands section {
  display: grid;
  gap: var(--graft-density-gap-6);
}

.project-import-lifecycle-review__commands .project-code-panel {
  gap: var(--graft-density-gap-8);
}

@media (width <= 900px) {
  .project-import-lifecycle-review__grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .project-import-lifecycle-review__field-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
