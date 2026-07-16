<template>
  <section class="project-import-lifecycle-review">
    <project-lifecycle-configuration-step
      v-model:draft="draft"
      :title="t('project.import.lifecycle.title')"
      :description="t('project.import.lifecycle.description')"
      :authority-message="t('project.import.lifecycle.authorityHint')"
      :configuration-title="t('project.import.lifecycle.configurationTitle')"
      :command-preview-title="t('project.import.lifecycle.commandPreviewTitle')"
      :back-label="t('project.import.actions.backToInspect')"
      :refresh-label="t('project.import.actions.refreshInspect')"
      :refresh-loading="inspectionRefreshLoading"
      :continue-label="t('project.import.actions.confirmLifecycle')"
      @back="$emit('back')"
      @refresh="$emit('refresh')"
      @continue="$emit('confirm')"
    />
  </section>
</template>
<script setup lang="ts">
import { useApplicationPageContext } from '../shared/page-context';
import type { ApplicationLifecycleConfigurationDraft } from '../types/project';
import ProjectLifecycleConfigurationStep from './ProjectLifecycleConfigurationStep.vue';

defineOptions({ name: 'ApplicationImportLifecycleReview' });

// 导入页适配通用生命周期步骤的文案和确认事件，最终请求参数仍由导入流程持有。

const draft = defineModel<ApplicationLifecycleConfigurationDraft>('draft', { required: true });
defineProps<{ inspectionRefreshLoading?: boolean }>();
defineEmits<{ (event: 'back'): void; (event: 'confirm'): void; (event: 'refresh'): void }>();

const { t } = useApplicationPageContext();
</script>
