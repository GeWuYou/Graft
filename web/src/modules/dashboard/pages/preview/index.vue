<template>
  <dashboard-workbench
    preview
    :generated-at="generatedAt"
    :navigation-links="navigationLinks"
    :presentation="DASHBOARD_PREVIEW_PRESENTATION"
    :refreshing="refreshing"
    :retrying-id="retryingId"
    @navigate="navigate"
    @refresh="refresh"
    @retry-item="retryItem"
  />
</template>
<script setup lang="ts">
// 开发预览只驱动固定场景，不读取生产 Dashboard API，也不持有长期 presentation authority。
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';

import type { SupportedLocale } from '@/contracts/i18n/locales';
import { currentLocale } from '@/locales';
import { usePermissionStore } from '@/store/modules/permission';

import DashboardWorkbench from '../../components/workbench/DashboardWorkbench.vue';
import { buildDashboardQuickActionLinks } from '../../contract/sidebar-quick-actions';
import { DASHBOARD_PREVIEW_PRESENTATION } from '../../presentation/preview-workbench';
import type { PresentationItem } from '../../presentation/workbench';

defineOptions({ name: 'DashboardWorkbenchPreviewPage' });

const router = useRouter();
const permissionStore = usePermissionStore();
const generatedAt = ref(DASHBOARD_PREVIEW_PRESENTATION.generatedAt);
const refreshing = ref(false);
const retryingId = ref('');
const navigationLinks = computed(() =>
  buildDashboardQuickActionLinks(permissionStore.routers, currentLocale.value as SupportedLocale),
);

function navigate(route: string) {
  void router.push(route);
}

function refresh() {
  if (refreshing.value) {
    return;
  }
  refreshing.value = true;
  window.setTimeout(() => {
    generatedAt.value = new Date().toISOString();
    refreshing.value = false;
  }, 350);
}

function retryItem(item: PresentationItem) {
  if (retryingId.value) {
    return;
  }
  retryingId.value = item.id;
  window.setTimeout(() => {
    retryingId.value = '';
  }, 500);
}
</script>
