<template>
  <div v-if="!isRefreshing">
    <router-view v-if="!isFramePage" v-slot="{ Component }">
      <transition name="fade" mode="out-in">
        <keep-alive>
          <component :is="Component" v-if="shouldKeepActiveViewAlive" :key="activeViewKey" />
        </keep-alive>
      </transition>
      <component :is="Component" v-if="!shouldKeepActiveViewAlive" :key="activeViewKey" />
    </router-view>
    <frame-page v-else />
  </div>

  <t-loading v-else />
</template>
<script setup lang="ts">
import isBoolean from 'lodash/isBoolean';
import isUndefined from 'lodash/isUndefined';
import { computed } from 'vue';
import { useRoute } from 'vue-router';

import FramePage from '@/layouts/frame/index.vue';
import { useTabsRouterStore } from '@/store';

// <suspense>标签属于实验性功能，请谨慎使用
// 如果存在需解决/page/1=> /page/2 刷新数据问题 请修改代码 使用activeRouteFullPath 作为key
// <suspense>
//  <component :is="Component" :key="activeRouteFullPath" />
// </suspense>

// import { useRouter } from 'vue-router';
// const activeRouteFullPath = computed(() => {
//   const router = useRouter();
//   return router.currentRoute.value.fullPath;
// });

const activeTabRoute = computed(() => {
  const tabsRouterStore = useTabsRouterStore();
  return tabsRouterStore.tabRouters.find((tabRoute) => tabRoute.tabKey === tabsRouterStore.activeTabKey);
});

const shouldKeepActiveViewAlive = computed(() => {
  const tabRoute = activeTabRoute.value;
  const keepAliveConfig = tabRoute?.meta?.keepAlive ?? route.meta?.keepAlive;
  const isRouteKeepAlive = isUndefined(keepAliveConfig) || (isBoolean(keepAliveConfig) && keepAliveConfig); // 默认开启keepalive
  return Boolean(tabRoute?.isAlive) && isRouteKeepAlive;
});

const isRefreshing = computed(() => {
  const tabsRouterStore = useTabsRouterStore();
  const { refreshing } = tabsRouterStore;
  return refreshing;
});

const activeViewKey = computed(() => {
  const tabsRouterStore = useTabsRouterStore();
  return tabsRouterStore.activeTabKey || route.fullPath || route.path;
});

const route = useRoute(); // 这个不能放到computed中，切换页面时会导致被缓存
const isFramePage = computed(() => {
  return !!route.meta?.frameSrc;
});
</script>
<style lang="less" scoped>
.fade-leave-active,
.fade-enter-active {
  transition: opacity @anim-duration-slow @anim-time-fn-easing;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
