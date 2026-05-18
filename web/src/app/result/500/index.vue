<template>
  <result title="500 Internal Server Error" type="500" :tip="t('app.result.500.subtitle')">
    <t-button @click="() => router.push(ROOT_ENTRY_PATH)">{{ t('app.result.500.back') }}</t-button>
  </result>
</template>
<script setup lang="ts">
import { useRouter } from 'vue-router';

import { ROOT_ENTRY_PATH } from '@/contracts/app/routes';
import { t } from '@/locales';
import Result from '@/shared/components/ResultView.vue';

defineOptions({
  name: 'Result500',
});

const router = useRouter();
</script>
