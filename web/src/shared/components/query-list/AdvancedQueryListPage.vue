<template>
  <div :class="rootClass" :data-page-type="pageType">
    <management-page-content>
      <management-page-header :title="title" :description="description">
        <template v-if="$slots.eyebrow" #eyebrow>
          <slot name="eyebrow" />
        </template>
        <template #actions>
          <slot name="actions" />
          <t-button theme="default" variant="outline" :loading="loading" @click="$emit('reload')">
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

withDefaults(
  defineProps<{
    description: string;
    errorMessage?: string;
    errorTitle: string;
    loading?: boolean;
    pageType?: string;
    reloadLabel: string;
    retryLabel: string;
    rootClass?: string;
    title: string;
  }>(),
  {
    errorMessage: '',
    loading: false,
    pageType: 'query-builder-list-detail',
    rootClass: '',
  },
);

defineEmits<{
  (e: 'reload'): void;
}>();
</script>
