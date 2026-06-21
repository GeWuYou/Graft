<template>
  <result
    :title="t('app.result.success.title')"
    :tip="t('app.result.success.subtitle')"
    status="success"
    status-theme="success"
  >
    <t-button @click="() => router.push(ROOT_ENTRY_PATH)">{{ t('app.result.success.back') }}</t-button>
  </result>
</template>
<script setup lang="ts">
import { useRouter } from 'vue-router';

import { ROOT_ENTRY_PATH } from '@/contracts/app/routes';
import { t } from '@/locales';
import Result from '@/shared/components/ResultView.vue';

defineOptions({
  name: 'ResultSuccess',
});

const router = useRouter();
</script>
