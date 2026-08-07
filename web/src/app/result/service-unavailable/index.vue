<template>
  <result
    :title="t('app.result.serviceUnavailable.title')"
    :tip="t('app.result.serviceUnavailable.subtitle')"
    type="maintenance"
  >
    <t-button :loading="checking" @click="retry">{{ t('app.result.networkError.reload') }}</t-button>
    <t-button theme="default" @click="copyDiagnostics">{{
      t('app.result.serviceUnavailable.copyDiagnostics')
    }}</t-button>
  </result>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { resolveRecoveryRoutePath, ROOT_ENTRY_PATH } from '@/contracts/app/routes';
import { t } from '@/locales';
import Result from '@/shared/components/ResultView.vue';
import { copyText } from '@/shared/observability/copy';
import { usePlatformAvailabilityStore } from '@/store';

defineOptions({ name: 'ResultServiceUnavailable' });

// 此壳层页只恢复原始导航，不自行恢复会话或重放业务请求。

const route = useRoute();
const router = useRouter();
const availability = usePlatformAvailabilityStore();
const checking = ref(false);

async function retry() {
  if (checking.value) {
    return;
  }

  checking.value = true;
  try {
    if (!(await availability.checkHealth())) return;
    const requestedRedirect = typeof route.query.redirect === 'string' ? route.query.redirect : null;
    const pendingPath = availability.consumePendingPath();
    const redirect = resolveRecoveryRoutePath(requestedRedirect, pendingPath);
    await router.replace(redirect);
  } finally {
    checking.value = false;
  }
}

async function copyDiagnostics() {
  const copied = await copyText(
    JSON.stringify(
      {
        status: availability.status,
        consecutiveFailures: availability.consecutiveFailures,
        lastCheckedAt: availability.lastCheckedAt ? new Date(availability.lastCheckedAt).toISOString() : null,
        path: route.query.redirect || availability.pendingPath || ROOT_ENTRY_PATH,
        userAgent: typeof navigator === 'undefined' ? null : navigator.userAgent,
      },
      null,
      2,
    ),
  );
  await (copied
    ? MessagePlugin.success(t('app.result.serviceUnavailable.copySuccess'))
    : MessagePlugin.error(t('app.result.serviceUnavailable.copyFailed')));
}
</script>
