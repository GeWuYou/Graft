<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-breadcrumb
    v-if="showBreadcrumb && crumbs.length > 0"
    max-item-width="150"
    class="tdesign-breadcrumb shell-breadcrumb"
  >
    <t-breadcrumbItem v-for="item in crumbs" :key="item.key" :to="item.to">
      {{ item.title }}
    </t-breadcrumbItem>
  </t-breadcrumb>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { useLocale } from '@/locales/useLocale';
import { useSettingStore } from '@/store';
import { renderLocalizedTitle, resolveRouteLocalizedTitle } from '@/utils/route/meta';

const { locale } = useLocale();
const route = useRoute();
const router = useRouter();
const settingStore = useSettingStore();
const showBreadcrumb = computed(() => settingStore.showBreadcrumb);

interface BreadcrumbItem {
  key: string;
  to?: string;
  title: string;
}

const crumbs = computed(() => {
  return route.matched.reduce<BreadcrumbItem[]>((breadcrumbArray, matchedRoute) => {
    const { meta, path } = matchedRoute;
    if (meta?.hiddenBreadcrumb) {
      return breadcrumbArray;
    }

    const title = renderLocalizedTitle(resolveRouteLocalizedTitle(meta, 'breadcrumb'), locale.value, '');
    if (!title) {
      return breadcrumbArray;
    }

    const isLast = matchedRoute === route.matched.at(-1);
    const resolvedPath = resolveMatchedRoutePath(matchedRoute.name, matchedRoute.path);

    breadcrumbArray.push({
      key: matchedRoute.name ? String(matchedRoute.name) : `${path}:${breadcrumbArray.length}`,
      to: !isLast ? resolvedPath : undefined,
      title,
    });

    return breadcrumbArray;
  }, []);
});

function resolveMatchedRoutePath(name: string | symbol | undefined | null, path: string) {
  try {
    if (name) {
      return router.resolve({
        name,
        params: route.params,
      }).fullPath;
    }
    return router.resolve({
      path,
    }).fullPath;
  } catch {
    return path;
  }
}
</script>
<style scoped>
.tdesign-breadcrumb {
  margin-bottom: var(--graft-density-gap-8);
}
</style>
