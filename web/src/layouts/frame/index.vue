<template>
  <div v-if="showFrame">
    <template v-for="frame in getFramePages" :key="frame.path">
      <frame-content
        v-if="frame.name && hasRenderFrame(String(frame.name))"
        v-show="showIframe(frame)"
        :frame-src="frame.meta?.frameSrc"
      />
    </template>
  </div>
</template>
<script lang="ts">
import { computed, defineComponent, unref } from 'vue';

import FrameContent from '../components/FrameContent.vue';
import { useFrameKeepAlive } from './useFrameKeepAlive';

// iframe 布局只负责筛选并切换已注册的外部页面实例，具体尺寸和加载态由 FrameContent 管理。
export default defineComponent({
  name: 'FrameLayout',
  components: { FrameContent },
  setup() {
    const { getFramePages, hasRenderFrame, showIframe } = useFrameKeepAlive();

    const showFrame = computed(() => unref(getFramePages).length > 0);

    return { getFramePages, hasRenderFrame, showIframe, showFrame };
  },
});
</script>
