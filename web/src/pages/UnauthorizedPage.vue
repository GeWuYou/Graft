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
    typeof value === 'string'
    && value.length > 0
    && value.startsWith('/')
    && !value.startsWith('//')
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
  min-height: 100%;
  display: grid;
  align-items: center;
}

.unauthorized-page__card {
  width: min(100%, 560px);
  margin: 0 auto;
  border-radius: 24px;
  text-align: center;
  box-shadow: 0 24px 72px rgba(15, 23, 42, 0.1);
}

.unauthorized-page__code {
  display: inline-block;
  color: #d54941;
  font-size: 64px;
  font-weight: 700;
  line-height: 1;
}

.unauthorized-page__card h1 {
  margin: 20px 0 12px;
  color: #1a2433;
}

.unauthorized-page__card p {
  margin: 0 0 24px;
  color: #66788f;
  line-height: 1.7;
}
</style>
