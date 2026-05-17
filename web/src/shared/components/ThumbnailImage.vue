<template>
  <img :class="className" :src="src" />
</template>
<script setup lang="ts">
import { computed } from 'vue';

const { type } = defineProps({
  src: {
    type: String,
    default: '',
  },
  type: {
    type: String,
    default: 'layout',
  },
});

const className = computed(() => {
  return [
    'thumbnail-container',
    {
      'thumbnail-circle': type === 'circle',
      'thumbnail-layout': type === 'layout',
    },
  ];
});
</script>
<style lang="less" scoped>
@import '@/style/index.less';

.thumbnail {
  &-container {
    display: inline-block;
  }

  &-circle {
    border-radius: var(--td-radius-circle);
  }

  &-layout {
    height: 48px;
    width: 88px;
  }
}
</style>
