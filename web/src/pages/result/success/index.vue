<template>
  <div class="result-success">
    <t-icon class="result-success-icon" name="check-circle" />
    <div class="result-success-title">{{ t('pages.result.success.title') }}</div>
    <div class="result-success-describe">{{ t('pages.result.success.subtitle') }}</div>
    <div>
      <t-button theme="default" @click="() => router.push('/detail/advanced')">
        {{ t('pages.result.success.progress') }}
      </t-button>
      <t-button @click="() => router.push('/dashboard/base')"> {{ t('pages.result.success.back') }} </t-button>
    </div>
  </div>
</template>
<script setup lang="ts">
import { useRouter } from 'vue-router';

import { t } from '@/locales';

defineOptions({
  name: 'ResultSuccess',
});

const router = useRouter();
</script>
<style lang="less" scoped>
.result-success {
  align-items: center;
  display: flex;
  flex-direction: column;
  height: 75vh;
  justify-content: center;

  &-icon {
    color: var(--td-success-color);
    font-size: var(--td-comp-size-xxxxl);
  }

  &-title {
    color: var(--td-text-color-primary);
    font: var(--td-font-title-large);
    margin-top: var(--td-comp-margin-xxl);
    text-align: center;
  }

  &-describe {
    color: var(--td-text-color-secondary);
    font: var(--td-font-body-medium);
    margin: var(--td-comp-margin-s) 0 var(--td-comp-margin-xxxl);
  }
}
</style>
