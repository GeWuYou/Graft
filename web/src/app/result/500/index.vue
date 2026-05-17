<template>
  <result title="500 Internal Server Error" type="500" :tip="t('app.result.500.subtitle')">
    <t-button @click="() => router.push('/')">{{ t('app.result.500.back') }}</t-button>
  </result>
</template>
<script setup lang="ts">
import { useRouter } from 'vue-router';

import { t } from '@/locales';
import Result from '@/shared/components/ResultView.vue';

defineOptions({
  name: 'Result500',
});

const router = useRouter();
</script>
