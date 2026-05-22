<template>
  <div class="header-menu-search-left">
    <t-button
      :class="{ 'search-icon-hide': isSearchFocus }"
      theme="default"
      shape="square"
      variant="text"
      @click="changeSearchFocus(true)"
    >
      <t-icon name="search" />
    </t-button>
    <t-input
      ref="searchInput"
      v-model="searchData"
      class="header-search"
      :class="[{ 'width-zero': !isSearchFocus }]"
      :placeholder="t('layout.search.placeholder')"
      :autofocus="isSearchFocus"
      @blur="changeSearchFocus(false)"
    >
      <template #prefix-icon>
        <t-icon name="search" size="16" />
      </template>
    </t-input>
  </div>
</template>
<script setup lang="ts">
import { nextTick, ref } from 'vue';

import { t } from '@/locales';

const isSearchFocus = ref(false);
const searchData = ref('');
const searchInput = ref<{ focus?: () => void } | null>(null);

const changeSearchFocus = (value: boolean) => {
  if (!value) {
    searchData.value = '';
  }
  isSearchFocus.value = value;
  if (value) {
    nextTick(() => {
      searchInput.value?.focus?.();
    });
  }
};
</script>
<style lang="less" scoped>
.t-button {
  margin: 0 8px;
  transition: opacity @anim-duration-base @anim-time-fn-easing;

  .t-icon {
    font-size: 20px;

    &.general {
      margin-right: 16px;
    }
  }
}

.search-icon-hide {
  opacity: 0;
}

.header-menu-search-left {
  align-items: center;
  display: flex;

  .header-search {
    transition: width @anim-duration-base @anim-time-fn-easing;
    width: 200px;

    :deep(.t-input) {
      border: 0;

      &:focus {
        box-shadow: none;
      }
    }

    &.width-zero {
      opacity: 0;
      width: 0;
    }
  }
}
</style>
