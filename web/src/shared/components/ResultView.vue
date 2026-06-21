<template>
  <div class="result-page" :data-page-type="pageType">
    <div class="result-container">
      <div v-if="props.status" class="result-status-icon" :class="`result-status-icon--${props.statusTheme}`">
        <t-icon :name="statusIconName" />
      </div>
      <div v-else class="result-bg-img">
        <component :is="dynamicComponent"></component>
      </div>
      <div class="result-title">{{ props.title }}</div>
      <div class="result-tip">{{ props.tip }}</div>
      <div class="result-actions">
        <slot />
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import Result403Icon from '@/assets/assets-result-403.svg?component';
import Result404Icon from '@/assets/assets-result-404.svg?component';
import Result500Icon from '@/assets/assets-result-500.svg?component';
import ResultIeIcon from '@/assets/assets-result-ie.svg?component';
import ResultMaintenanceIcon from '@/assets/assets-result-maintenance.svg?component';
import ResultWifiIcon from '@/assets/assets-result-wifi.svg?component';

type ResultIllustrationType = '' | '403' | '404' | '500' | 'ie' | 'wifi' | 'maintenance';
type ResultStatus = '' | 'success' | 'fail';
type ResultStatusTheme = 'default' | 'success' | 'danger';

const props = withDefaults(
  defineProps<{
    title?: string;
    tip?: string;
    type?: ResultIllustrationType;
    status?: ResultStatus;
    statusTheme?: ResultStatusTheme;
  }>(),
  {
    title: '',
    tip: '',
    type: '',
    status: '',
    statusTheme: 'default',
  },
);

const dynamicComponent = computed(() => {
  switch (props.type) {
    case '403':
      return Result403Icon;
    case '404':
      return Result404Icon;
    case '500':
      return Result500Icon;
    case 'ie':
      return ResultIeIcon;
    case 'wifi':
      return ResultWifiIcon;
    case 'maintenance':
      return ResultMaintenanceIcon;
    default:
      return Result403Icon;
  }
});

const pageType = computed(() => (props.status ? 'operation-result' : 'error-result'));

const statusIconName = computed(() => {
  if (props.status === 'success') {
    return 'check-circle';
  }

  return 'error-circle';
});
</script>
<style lang="less" scoped>
.result {
  &-link {
    color: var(--td-brand-color);
    cursor: pointer;
    text-decoration: none;

    &:hover {
      color: var(--td-brand-color);
    }

    &:active {
      color: var(--td-brand-color);
    }

    &--active {
      color: var(--td-brand-color);
    }

    &:focus {
      text-decoration: none;
    }
  }

  &-container {
    align-items: center;
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-height: min(720px, 76vh);
    padding: var(--td-comp-paddingTB-xxxl) var(--td-comp-paddingLR-xl) var(--graft-page-bottom-safe-area);
  }

  &-page {
    align-items: center;
    background: var(--td-bg-color-page);
    color: var(--td-text-color-primary);
    display: flex;
    justify-content: center;
    min-height: 100%;
  }

  &-bg-img {
    color: var(--td-brand-color);
    width: 200px;
  }

  &-status-icon {
    align-items: center;
    color: var(--td-text-color-secondary);
    display: flex;
    font-size: var(--td-comp-size-xxxxl);
    justify-content: center;
    line-height: 1;

    &--success {
      color: var(--td-success-color);
    }

    &--danger {
      color: var(--td-error-color);
    }
  }

  &-title {
    color: var(--td-text-color-primary);
    font: var(--td-font-title-large);
    font-style: normal;
    margin-top: var(--td-comp-margin-l);
  }

  &-tip {
    color: var(--td-text-color-secondary);
    font: var(--td-font-body-medium);
    margin: var(--td-comp-margin-s) 0 var(--td-comp-margin-xxxl);
    max-width: 520px;
    text-align: center;
  }

  &-actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--td-comp-margin-s);
    justify-content: center;
  }
}
</style>
