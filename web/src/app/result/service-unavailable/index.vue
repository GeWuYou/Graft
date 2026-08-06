<template>
  <result :title="t('app.result.maintenance.title')" :tip="t('app.result.maintenance.subtitle')" type="maintenance">
    <t-button :loading="checking" @click="retry">{{ t('app.result.networkError.reload') }}</t-button>
    <t-button theme="default" @click="goHome">{{ t('app.result.maintenance.back') }}</t-button>
  </result>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { ROOT_ENTRY_PATH } from '@/contracts/app/routes';
import { t } from '@/locales';
import Result from '@/shared/components/ResultView.vue';
import { usePlatformAvailabilityStore } from '@/store';

defineOptions({ name: 'ResultServiceUnavailable' });

// 此壳层页只恢复原始导航，不自行恢复会话或重放业务请求。

const route = useRoute();
const router = useRouter();
const availability = usePlatformAvailabilityStore();
const checking = computed(() => availability.status === 'recovering');

async function retry() {
  if (!(await availability.checkHealth())) return;
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : availability.consumePendingPath();
  await router.replace(redirect || ROOT_ENTRY_PATH);
}

function goHome() {
  availability.pendingPath = null;
  void router.replace(ROOT_ENTRY_PATH);
}
</script>
