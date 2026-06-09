<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

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
