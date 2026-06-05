<template>
  <result :title="t('app.result.maintenance.title')" :tip="t('app.result.maintenance.subtitle')" type="maintenance">
    <t-button @click="() => router.push(ROOT_ENTRY_PATH)">{{ t('app.result.maintenance.back') }}</t-button>
  </result>
</template>
<script setup lang="ts">
import { useRouter } from 'vue-router';

import { ROOT_ENTRY_PATH } from '@/contracts/app/routes';
import { t } from '@/locales';
import Result from '@/shared/components/ResultView.vue';

defineOptions({
  name: 'ResultMaintenance',
});

const router = useRouter();
</script>
