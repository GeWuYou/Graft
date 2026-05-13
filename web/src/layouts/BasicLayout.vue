<template>
  <t-layout class="basic-layout">
    <t-aside class="basic-layout__aside">
      <div class="basic-layout__logo">
        <span class="basic-layout__logo-mark">G</span>
        <div>
          <strong>{{ t('app.name') }}</strong>
          <p>{{ t('app.shellName') }}</p>
        </div>
      </div>

      <t-menu
        class="basic-layout__menu"
        :value="navigationStore.activePath"
        theme="light"
        @change="handleMenuChange"
      >
        <t-menu-item
          v-for="item in visibleItems"
          :key="item.path"
          :value="item.path"
        >
          <template #icon>
            <component :is="resolveIcon(item.icon)" />
          </template>
          {{ resolveNavigationTitle(item.titleKey, item.title) }}
        </t-menu-item>
      </t-menu>
    </t-aside>

    <t-layout>
      <t-header class="basic-layout__header">
        <div>
          <t-breadcrumb>
            <t-breadcrumb-item
              v-for="crumb in breadcrumbs"
              :key="crumb.path"
            >
              {{ crumb.title }}
            </t-breadcrumb-item>
          </t-breadcrumb>
          <h2 class="basic-layout__page-title">{{ currentTitle }}</h2>
        </div>

        <div class="basic-layout__actions">
          <t-tag theme="success" variant="light-outline">{{ t('common.status.mvpShell') }}</t-tag>
          <t-avatar>{{ userInitial }}</t-avatar>
          <div class="basic-layout__user">
            <strong>{{ authStore.userName }}</strong>
            <span>{{ t('layouts.basic.permissionHint') }}</span>
          </div>
          <t-button variant="text" theme="default" @click="handleLogout">
            {{ t('common.actions.logout') }}
          </t-button>
        </div>
      </t-header>

      <t-content class="basic-layout__content">
        <RouterView />
      </t-content>
    </t-layout>
  </t-layout>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { RouteRecordNormalized } from 'vue-router';
import { useRoute, useRouter, RouterView } from 'vue-router';
import { ChartBarIcon, DashboardIcon } from 'tdesign-icons-vue-next';

import { useI18n } from '@/app/i18n';
import { useAuthStore } from '@/stores/auth';
import { useNavigationStore } from '@/stores/navigation';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const navigationStore = useNavigationStore();
const { t } = useI18n();

const iconMap = {
  dashboard: DashboardIcon,
  chart: ChartBarIcon,
};

const visibleItems = computed(() =>
  navigationStore.items.filter((item) => authStore.hasPermission(item.permissionCode)),
);

const breadcrumbs = computed(() =>
  route.matched
    .filter(
      (record: RouteRecordNormalized) =>
        (record.meta.title || record.meta.titleKey) && !record.meta.hideInMenu,
    )
    .map((record: RouteRecordNormalized) => ({
      path: record.path,
      title: resolveRouteTitle(record.meta.titleKey, record.meta.title),
    })),
);

const currentTitle = computed(() => resolveRouteTitle(route.meta.titleKey, route.meta.title));
const userInitial = computed(() => authStore.userName.slice(0, 1).toUpperCase() || 'G');

function resolveIcon(icon?: string) {
  return icon && icon in iconMap ? iconMap[icon as keyof typeof iconMap] : DashboardIcon;
}

function resolveNavigationTitle(titleKey: string, fallback: string) {
  return t(titleKey, {
    fallback,
  });
}

function resolveRouteTitle(titleKey: unknown, fallback: unknown) {
  if (typeof titleKey === 'string') {
    return t(titleKey, {
      fallback: typeof fallback === 'string' ? fallback : t('app.name'),
    });
  }

  return typeof fallback === 'string' ? fallback : t('app.name');
}

function handleMenuChange(path: string) {
  void router.push(path);
}

function handleLogout() {
  authStore.logout();
  void router.push({ name: 'login' });
}
</script>

<style scoped>
.basic-layout {
  min-height: 100vh;
  background: linear-gradient(180deg, #f5f7fb 0%, #edf2f8 100%);
}

.basic-layout__aside {
  width: 240px;
  padding: 20px 16px;
  border-right: 1px solid #e7edf5;
  background: rgba(255, 255, 255, 0.88);
  backdrop-filter: blur(14px);
}

.basic-layout__logo {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 28px;
  padding: 0 8px;
}

.basic-layout__logo-mark {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  border-radius: 12px;
  background: linear-gradient(135deg, #0052d9 0%, #00a870 100%);
  color: #fff;
  font-weight: 700;
}

.basic-layout__logo strong {
  display: block;
  color: #1a2433;
  font-size: 16px;
}

.basic-layout__logo p {
  margin: 2px 0 0;
  color: #6b7a90;
  font-size: 12px;
}

.basic-layout__menu {
  border: 0;
  background: transparent;
}

.basic-layout__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  height: auto;
  padding: 20px 24px 0;
  background: transparent;
}

.basic-layout__page-title {
  margin: 12px 0 0;
  color: #1a2433;
  font-size: 28px;
  line-height: 1.1;
}

.basic-layout__actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.basic-layout__user {
  display: flex;
  flex-direction: column;
  min-width: 140px;
}

.basic-layout__user strong {
  color: #1a2433;
  font-size: 14px;
}

.basic-layout__user span {
  color: #7b889c;
  font-size: 12px;
}

.basic-layout__content {
  padding: 24px;
}

@media (max-width: 960px) {
  .basic-layout__aside {
    width: 88px;
  }

  .basic-layout__logo div,
  .basic-layout__user {
    display: none;
  }

  .basic-layout__header {
    flex-wrap: wrap;
    align-items: flex-start;
  }
}
</style>
