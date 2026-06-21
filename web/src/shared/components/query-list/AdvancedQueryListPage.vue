<template>
  <div :class="rootClass" :data-page-type="pageType">
    <management-page-content>
      <management-page-header
        :title="title"
        :description="description"
        :title-key="titleKey"
        :description-key="descriptionKey"
        :compact="compactHeader"
        :source="source"
      >
        <template #actions>
          <slot name="actions" />
          <t-button
            v-if="showHeaderReload"
            theme="default"
            variant="outline"
            :loading="loading"
            @click="$emit('reload')"
          >
            {{ reloadLabel }}
          </t-button>
        </template>
      </management-page-header>

      <slot name="feedback-extra" />

      <slot name="filters" />

      <management-empty-state
        v-if="errorMessage && !loading"
        tone="error"
        :title="errorTitle"
        :description="errorMessage"
      >
        <template #actions>
          <t-button theme="primary" variant="outline" @click="$emit('reload')">
            {{ retryLabel }}
          </t-button>
        </template>
      </management-empty-state>

      <slot v-else name="table" />
    </management-page-content>

    <slot name="detail" />
  </div>
</template>
<script setup lang="ts">
import { ManagementEmptyState, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import type { PageHeaderSource } from '@/shared/components/page';

withDefaults(
  defineProps<{
    description?: string;
    descriptionKey?: string;
    compactHeader?: boolean;
    errorMessage?: string;
    errorTitle: string;
    loading?: boolean;
    pageType?: string;
    reloadLabel: string;
    retryLabel: string;
    rootClass?: string;
    showHeaderReload?: boolean;
    source?: PageHeaderSource;
    title?: string;
    titleKey?: string;
  }>(),
  {
    compactHeader: false,
    description: '',
    descriptionKey: '',
    errorMessage: '',
    loading: false,
    pageType: 'query-builder-list-detail',
    rootClass: '',
    showHeaderReload: true,
    source: undefined,
    title: '',
    titleKey: '',
  },
);

defineEmits<{
  (e: 'reload'): void;
}>();
</script>
