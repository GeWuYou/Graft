<template>
  <div class="auth-layout">
    <div class="auth-layout__panel">
      <div class="auth-layout__brand">
        <span class="auth-layout__badge">Graft</span>
        <h1>插件式后台平台</h1>
        <p>先提供稳定后台壳，再让业务模块沿着菜单、路由、权限和 API 的固定路径接入。</p>
      </div>
      <RouterView />
    </div>
  </div>
</template>

<script setup lang="ts">
import { RouterView } from 'vue-router';
</script>

<style scoped>
.auth-layout {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 24px;
  background:
    radial-gradient(circle at top left, rgba(0, 168, 112, 0.14), transparent 32%),
    linear-gradient(180deg, #f4f7fb 0%, #eef3f8 100%);
}

.auth-layout__panel {
  width: min(100%, 1120px);
  display: grid;
  grid-template-columns: minmax(280px, 420px) minmax(320px, 420px);
  gap: 40px;
  align-items: center;
}

.auth-layout__brand {
  color: #12324d;
}

.auth-layout__badge {
  display: inline-flex;
  align-items: center;
  height: 32px;
  padding: 0 12px;
  border-radius: 999px;
  background: rgba(0, 168, 112, 0.12);
  color: #005a46;
  font-size: 14px;
  font-weight: 600;
}

.auth-layout__brand h1 {
  margin: 20px 0 12px;
  font-size: clamp(32px, 4vw, 48px);
  line-height: 1.1;
}

.auth-layout__brand p {
  margin: 0;
  max-width: 420px;
  color: #4f6476;
  font-size: 16px;
  line-height: 1.7;
}

@media (max-width: 960px) {
  .auth-layout__panel {
    grid-template-columns: 1fr;
    gap: 24px;
  }
}
</style>
