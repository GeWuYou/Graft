<template>
  <div class="auth-layout">
    <div class="auth-layout__panel">
      <div class="auth-layout__brand">
        <span class="auth-layout__badge">{{ t('app.name') }}</span>
        <h1>{{ t('layouts.auth.title') }}</h1>
        <p>{{ t('layouts.auth.description') }}</p>
      </div>
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import { RouterView } from 'vue-router';

import { useI18n } from '@/app/i18n';

const { t } = useI18n();
</script>

<style scoped>
.auth-layout {
  background:
    radial-gradient(circle at top left, rgb(0 168 112 / 14%), transparent 32%),
    linear-gradient(180deg, #f4f7fb 0%, #eef3f8 100%);
  display: grid;
  min-height: 100vh;
  padding: 24px;
  place-items: center;
}

.auth-layout__panel {
  align-items: center;
  display: grid;
  gap: 40px;
  grid-template-columns: minmax(280px, 420px) minmax(320px, 420px);
  width: min(100%, 1120px);
}

.auth-layout__brand {
  color: #12324d;
}

.auth-layout__badge {
  align-items: center;
  background: rgb(0 168 112 / 12%);
  border-radius: 999px;
  color: #005a46;
  display: inline-flex;
  font-size: 14px;
  font-weight: 600;
  height: 32px;
  padding: 0 12px;
}

.auth-layout__brand h1 {
  font-size: clamp(32px, 4vw, 48px);
  line-height: 1.1;
  margin: 20px 0 12px;
}

.auth-layout__brand p {
  color: #4f6476;
  font-size: 16px;
  line-height: 1.7;
  margin: 0;
  max-width: 420px;
}

@media (width <= 960px) {
  .auth-layout__panel {
    gap: 24px;
    grid-template-columns: 1fr;
  }
}
</style>
