<template>
  <div class="unauthorized-page">
    <t-card class="unauthorized-page__card" :bordered="false">
      <span class="unauthorized-page__code">403</span>
      <h1>{{ t('unauthorized.title') }}</h1>
      <p>{{ t('unauthorized.description') }}</p>
      <t-space wrap>
        <t-button theme="primary" @click="goFallback">
          {{ t('common.actions.goToAccessiblePage') }}
        </t-button>
        <t-button variant="outline" theme="default" @click="goLogin">
          {{ t('common.actions.switchAccount') }}
        </t-button>
      </t-space>
    </t-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { useI18n } from '@/app/i18n';
import { useAuthStore } from '@/stores/auth';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const { t } = useI18n();

function isSafeFallbackPath(value: unknown): value is string {
  return (
    typeof value === 'string' &&
    value.length > 0 &&
    value.startsWith('/') &&
    !value.startsWith('//')
  );
}

const fallbackPath = computed(() => {
  const fallback = route.query.fallback;

  return isSafeFallbackPath(fallback) ? fallback : '/dashboard';
});

function goFallback() {
  void router.replace(fallbackPath.value);
}

function goLogin() {
  authStore.logout();
  void router.push('/login');
}
</script>

<style scoped>
.unauthorized-page {
  align-items: center;
  display: grid;
  min-height: 100%;
}

.unauthorized-page__card {
  border-radius: 24px;
  box-shadow: 0 24px 72px rgb(15 23 42 / 10%);
  margin: 0 auto;
  text-align: center;
  width: min(100%, 560px);
}

.unauthorized-page__code {
  color: #d54941;
  display: inline-block;
  font-size: 64px;
  font-weight: 700;
  line-height: 1;
}

.unauthorized-page__card h1 {
  color: #1a2433;
  margin: 20px 0 12px;
}

.unauthorized-page__card p {
  color: #66788f;
  line-height: 1.7;
  margin: 0 0 24px;
}
</style>
