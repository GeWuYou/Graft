<template>
  <t-alert :theme="theme" :message="message">
    <template #operation>
      <t-button size="small" theme="default" variant="text" :loading="loading" @click="$emit('refresh')">
        {{ t('project.import.actions.refreshInspect') }}
      </t-button>
    </template>
  </t-alert>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { useApplicationPageContext } from '../shared/page-context';

defineOptions({ name: 'ApplicationImportInspectionSessionAlert' });

const props = defineProps<{
  errorMessage?: string;
  loading: boolean;
  valid: boolean;
}>();

defineEmits<{ (event: 'refresh'): void }>();

const { t } = useApplicationPageContext();
const theme = computed(() =>
  props.errorMessage ? 'error' : props.loading ? 'info' : props.valid ? 'success' : 'warning',
);
const message = computed(() => {
  if (props.errorMessage) {
    return props.errorMessage;
  }
  if (props.loading) {
    return t('project.import.inspectionSession.refreshing');
  }
  return props.valid ? t('project.import.inspectionSession.ready') : t('project.import.inspectionSession.expired');
});
</script>
