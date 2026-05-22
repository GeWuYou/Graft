<template>
  <div class="result-page" data-page-type="error-result">
    <div class="result-container">
      <div class="result-bg-img">
        <component :is="dynamicComponent"></component>
      </div>
      <div class="result-title">{{ title }}</div>
      <div class="result-tip">{{ tip }}</div>
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

const { type } = defineProps({
  bgUrl: {
    type: String,
    default: '',
  },
  title: {
    type: String,
    default: '',
  },
  tip: {
    type: String,
    default: '',
  },
  type: {
    type: String,
    default: '',
  },
});

const dynamicComponent = computed(() => {
  switch (type) {
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
    padding: var(--td-comp-paddingTB-xxxl) var(--td-comp-paddingLR-xl) var(--graft-page-bottom-safe-space);
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
