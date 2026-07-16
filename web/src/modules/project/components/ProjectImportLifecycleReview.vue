<template>
  <section class="project-import-lifecycle-review">
    <project-lifecycle-configuration-review
      v-model:draft="draft"
      :title="t('project.import.lifecycle.title')"
      :description="t('project.import.lifecycle.description')"
      :authority-message="t('project.import.lifecycle.authorityHint')"
      :configuration-title="t('project.import.lifecycle.configurationTitle')"
      :command-preview-title="t('project.import.lifecycle.commandPreviewTitle')"
    />

    <div class="project-import-step-actions">
      <t-button theme="default" variant="outline" @click="$emit('back')">
        {{ t('project.import.actions.backToInspect') }}
      </t-button>
      <t-button theme="default" variant="outline" :loading="inspectionRefreshLoading" @click="$emit('refresh')">
        {{ t('project.import.actions.refreshInspect') }}
      </t-button>
      <t-button theme="primary" @click="$emit('confirm')">
        {{ t('project.import.actions.confirmLifecycle') }}
      </t-button>
    </div>
  </section>
</template>
<script setup lang="ts">
import { useProjectPageContext } from '../shared/page-context';
import type { ProjectLifecycleConfigurationDraft } from '../types/project';
import ProjectLifecycleConfigurationReview from './ProjectLifecycleConfigurationReview.vue';

defineOptions({ name: 'ProjectImportLifecycleReview' });

// 生命周期审核区编辑导入草稿并展示命令预览，最终请求参数由父级在提交时生成。

const draft = defineModel<ProjectLifecycleConfigurationDraft>('draft', { required: true });
defineProps<{ inspectionRefreshLoading?: boolean }>();
defineEmits<{ (event: 'back'): void; (event: 'confirm'): void; (event: 'refresh'): void }>();

const { t } = useProjectPageContext();
</script>
