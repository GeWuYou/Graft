<template>
  <section class="project-lifecycle-configuration-step">
    <project-lifecycle-configuration-review
      v-model:draft="draft"
      :title="title"
      :description="description"
      :authority-message="authorityMessage"
      :configuration-title="configurationTitle"
    />

    <div class="project-lifecycle-step-actions">
      <t-button variant="outline" @click="$emit('back')">{{ backLabel }}</t-button>
      <t-button v-if="refreshLabel" variant="outline" :loading="refreshLoading" @click="$emit('refresh')">
        {{ refreshLabel }}
      </t-button>
      <t-button theme="primary" :disabled="continueDisabled" @click="$emit('continue')">{{ continueLabel }}</t-button>
    </div>
  </section>
</template>
<script setup lang="ts">
import type { ApplicationLifecycleConfigurationDraft } from '../types/project';
import ProjectLifecycleConfigurationReview from './ProjectLifecycleConfigurationReview.vue';

defineOptions({ name: 'ApplicationLifecycleConfigurationStep' });

// 生命周期步骤拥有配置草稿的编辑面和导航事件，调用方继续持有最终提交与路由状态。
const draft = defineModel<ApplicationLifecycleConfigurationDraft>('draft', { required: true });

withDefaults(
  defineProps<{
    title: string;
    description: string;
    authorityMessage: string;
    configurationTitle: string;
    backLabel: string;
    continueLabel: string;
    refreshLabel?: string;
    refreshLoading?: boolean;
    continueDisabled?: boolean;
  }>(),
  { refreshLabel: '', refreshLoading: false, continueDisabled: false },
);

defineEmits<{ (event: 'back'): void; (event: 'continue'): void; (event: 'refresh'): void }>();
</script>
<style scoped>
.project-lifecycle-configuration-step {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.project-lifecycle-step-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}
</style>
