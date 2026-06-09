<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <result :title="t('app.result.fail.title')" :tip="t('app.result.fail.subtitle')" status="fail" status-theme="danger">
    <t-button theme="default" @click="() => router.push(ROOT_ENTRY_PATH)">{{ t('app.result.fail.back') }}</t-button>
    <t-button @click="reloadPage">{{ t('app.result.fail.retry') }}</t-button>
  </result>
</template>
<script setup lang="ts">
import { useRouter } from 'vue-router';

import { ROOT_ENTRY_PATH } from '@/contracts/app/routes';
import { t } from '@/locales';
import Result from '@/shared/components/ResultView.vue';

defineOptions({
  name: 'ResultFail',
});

const router = useRouter();

const reloadPage = () => {
  window.location.reload();
};
</script>
