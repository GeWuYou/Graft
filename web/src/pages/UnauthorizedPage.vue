<template>
  <div class="unauthorized-page">
    <t-card class="unauthorized-page__card" :bordered="false">
      <span class="unauthorized-page__code">403</span>
      <h1>当前账号无权访问此页面</h1>
      <p>
        登录态仍然有效，但目标路由要求的权限不在当前会话内。后续接入后端菜单与权限数据后，这里会继续作为显式授权兜底页。
      </p>
      <t-space wrap>
        <t-button theme="primary" @click="goFallback">前往可访问页面</t-button>
        <t-button variant="outline" theme="default" @click="goLogin">切换账号</t-button>
      </t-space>
    </t-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { useAuthStore } from '@/stores/auth';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

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
