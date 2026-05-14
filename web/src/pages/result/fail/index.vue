<template>
  <div class="result-success">
    <t-icon class="result-success-icon" name="error-circle" />
    <div class="result-success-title">{{ t('pages.result.fail.title') }}</div>
    <div class="result-success-describe">{{ t('pages.result.fail.subtitle') }}</div>
    <div>
      <t-button theme="default" @click="() => router.push('/dashboard/base')">{{
        t('pages.result.fail.back')
      }}</t-button>
      <t-button @click="() => router.push('/form/base')">{{ t('pages.result.fail.modify') }} </t-button>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { useRouter } from 'vue-router';

import { t } from '@/locales';

defineOptions({
  name: 'ResultFail',
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
    color: var(--td-text-color-secondary);
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
