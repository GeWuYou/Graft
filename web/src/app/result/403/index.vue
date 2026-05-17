<template>
  <result title="403 Forbidden" :tip="t('app.result.403.tips')">
    <t-button @click="() => router.push('/')">{{ t('app.result.403.back') }}</t-button>
  </result>
</template>
<script setup lang="ts">
import { useRouter } from 'vue-router';

import { t } from '@/locales';
import Result from '@/shared/components/ResultView.vue';

defineOptions({
  name: 'Result403',
});

const router = useRouter();
</script>
