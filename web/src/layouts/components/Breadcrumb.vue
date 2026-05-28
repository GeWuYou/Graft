<template>
  <t-breadcrumb max-item-width="150" class="tdesign-breadcrumb">
    <t-breadcrumbItem v-for="item in crumbs" :key="item.to" :to="item.to">
      {{ item.title }}
    </t-breadcrumbItem>
  </t-breadcrumb>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';

import { useLocale } from '@/locales/useLocale';
import { renderLocalizedTitle, resolveRouteLocalizedTitle } from '@/utils/route/meta';

const { locale } = useLocale();
const route = useRoute();

interface BreadcrumbItem {
  to: string;
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

    breadcrumbArray.push({
      to: path,
      title,
    });

    return breadcrumbArray;
  }, []);
});
</script>
<style scoped>
.tdesign-breadcrumb {
  margin-bottom: 8px;
}
</style>
