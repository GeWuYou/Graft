<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <result :title="t('app.result.networkError.title')" :tip="t('app.result.networkError.subtitle')" type="wifi">
    <t-button theme="default" @click="() => router.push(ROOT_ENTRY_PATH)">
      {{ t('app.result.networkError.back') }}
    </t-button>
    <t-button @click="reloadPage">{{ t('app.result.networkError.reload') }}</t-button>
  </result>
</template>
<script setup lang="ts">
import { useRouter } from 'vue-router';

import { ROOT_ENTRY_PATH } from '@/contracts/app/routes';
import { t } from '@/locales';
import Result from '@/shared/components/ResultView.vue';

defineOptions({
  name: 'ResultNetworkError',
});

const router = useRouter();

const reloadPage = () => {
  window.location.reload();
};
</script>
