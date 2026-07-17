<template>
  <section class="project-lifecycle-configuration-review">
    <div class="project-lifecycle-configuration-review__heading">
      <h2>{{ title }}</h2>
      <p>{{ description }}</p>
    </div>
    <t-alert theme="info" :message="authorityMessage" />
    <div class="project-lifecycle-configuration-review__grid">
      <t-card :title="configurationTitle" bordered>
        <div class="project-lifecycle-configuration-review__content">
          <div class="project-lifecycle-configuration-review__field-grid">
            <label class="project-lifecycle-configuration-review__field">
              <span>{{ t('project.detail.lifecycle.profiles') }}</span>
              <t-input
                v-model="profilesInput"
                :disabled="disabled"
                :placeholder="t('project.detail.lifecycle.profilesPlaceholder')"
              />
            </label>
            <label class="project-lifecycle-configuration-review__field">
              <span>{{ t('project.detail.lifecycle.additionalArgs') }}</span>
              <t-input
                v-model="draft.additional_args"
                :disabled="disabled"
                :placeholder="t('project.detail.lifecycle.additionalArgsPlaceholder')"
              />
            </label>
          </div>
          <template v-for="definition in lifecycleSwitchHelpDefinitions" :key="definition.key">
            <div class="project-lifecycle-configuration-review__option">
              <div class="project-lifecycle-configuration-review__option-content">
                <div class="project-lifecycle-configuration-review__option-title">
                  <span>{{ t(definition.titleKey) }}</span>
                  <lifecycle-help-trigger :definition="definition" :draft="draft" />
                </div>
                <p>{{ t(definition.summaryKey) }}</p>
              </div>
              <div class="project-lifecycle-configuration-review__option-control">
                <t-switch v-model="draft[definition.field]" :aria-label="t(definition.titleKey)" :disabled="disabled" />
              </div>
            </div>
            <label
              v-if="definition.key === 'waitAfterUp' && waitTimeoutDefinition.visible?.(draft)"
              class="project-lifecycle-configuration-review__field"
            >
              <span>{{ t(waitTimeoutDefinition.titleKey) }}</span>
              <t-input-number
                v-model="draft.wait_timeout_seconds"
                :disabled="disabled"
                :min="1"
                :max="3600"
                :step="1"
              />
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
      <t-card :title="commandPreviewTitle" bordered>
        <div class="project-lifecycle-configuration-review__commands">
          <section v-for="command in commandPreviews" :key="command.key" class="project-code-panel">
            <strong>{{ command.title }}</strong>
            <code-block :code="command.preview" lang="shell" wrap />
          </section>
        </div>
      </t-card>
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
import { useApplicationPageContext } from '../shared/page-context';
import type { ApplicationLifecycleConfigurationDraft } from '../types/project';
import LifecycleHelpTrigger from './LifecycleHelpTrigger.vue';

defineOptions({ name: 'ApplicationLifecycleConfigurationReview' });
// 配置审核区直接编辑调用方持有的草稿，并将生成命令作为当前草稿的可视化结果展示。
const draft = defineModel<ApplicationLifecycleConfigurationDraft>('draft', { required: true });
defineProps<{
  title: string;
  description: string;
  authorityMessage: string;
  configurationTitle: string;
  commandPreviewTitle: string;
  disabled?: boolean;
}>();
const { t } = useApplicationPageContext();
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
.project-lifecycle-configuration-review,
.project-lifecycle-configuration-review__content,
.project-lifecycle-configuration-review__commands,
.project-lifecycle-configuration-review__field,
.project-lifecycle-configuration-review__option-content {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.project-lifecycle-configuration-review {
  gap: var(--graft-density-gap-16);
}

.project-lifecycle-configuration-review__heading p,
.project-lifecycle-configuration-review__option p,
.project-lifecycle-configuration-review__field small {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.project-lifecycle-configuration-review__heading h2 {
  margin: 0;
}

.project-lifecycle-configuration-review__grid,
.project-lifecycle-configuration-review__field-grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-lifecycle-configuration-review__field-grid {
  gap: var(--graft-density-gap-12);
}

.project-lifecycle-configuration-review__option {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
}

.project-lifecycle-configuration-review__option-control {
  align-items: center;
  display: flex;
  flex: none;
}

.project-lifecycle-configuration-review__option-control :deep(.t-switch) {
  flex: none;
  width: auto;
}

.project-lifecycle-configuration-review__option-control :deep(.t-switch:disabled) {
  cursor: not-allowed;
}

.project-lifecycle-configuration-review__option-title {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-6);
}

.project-lifecycle-configuration-review__commands section {
  display: grid;
  gap: var(--graft-density-gap-6);
}

@media (width <= 900px) {
  .project-lifecycle-configuration-review__grid,
  .project-lifecycle-configuration-review__field-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (width <= 720px) {
  .project-lifecycle-configuration-review__option {
    align-items: flex-start;
    flex-direction: column;
  }

  .project-lifecycle-configuration-review__option-control {
    align-self: flex-end;
  }
}
</style>
