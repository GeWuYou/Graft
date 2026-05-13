<template>
  <div class="dashboard-page">
    <section class="dashboard-page__hero">
      <t-card class="dashboard-page__hero-card" :bordered="false">
        <div class="dashboard-page__hero-copy">
          <div>
            <p class="dashboard-page__eyebrow">Graft Platform</p>
            <h3>欢迎回来，{{ authStore.userName || '管理员' }}</h3>
            <p class="dashboard-page__summary">
              当前前端壳已经具备登录、静态路由、基础导航和权限占位，后续模块可以沿着
              `menu + route + page + api + permission` 的路径接入。
            </p>
          </div>

          <div class="dashboard-page__hero-tags">
            <t-tag theme="primary">Vue 3 + TypeScript</t-tag>
            <t-tag theme="success">TDesign Vue Next</t-tag>
            <t-tag theme="warning">Static Permission Stub</t-tag>
          </div>
        </div>
      </t-card>
    </section>

    <section class="dashboard-page__metrics">
      <t-card
        v-for="metric in metrics"
        :key="metric.title"
        class="dashboard-page__metric-card"
        :bordered="false"
      >
        <span class="dashboard-page__metric-label">{{ metric.title }}</span>
        <strong class="dashboard-page__metric-value">{{ metric.value }}</strong>
        <p class="dashboard-page__metric-note">{{ metric.note }}</p>
      </t-card>
    </section>

    <section class="dashboard-page__grid">
      <t-card title="当前壳能力" :bordered="false">
        <ul class="dashboard-page__list">
          <li>登录页使用 `AuthLayout`，后台页面统一挂在 `BasicLayout`。</li>
          <li>静态路由已经接入全局守卫，未登录访问会被重定向回登录页。</li>
          <li>导航 store 保留 `plugin` 和 `permissionCode` 字段，为后端动态菜单留出契约位。</li>
        </ul>
      </t-card>

      <t-card title="下一步接入建议" :bordered="false">
        <ul class="dashboard-page__list">
          <li>登录成功后改为拉取用户信息、权限集合和菜单树。</li>
          <li>将动态菜单结果装配到 `router` 与 `navigation` store。</li>
          <li>在 `web/src/modules` 下按插件维度接入用户、角色和审计模块。</li>
        </ul>
      </t-card>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth';

const authStore = useAuthStore();

const metrics = [
  {
    title: '静态路由',
    value: '3',
    note: '登录页、仪表盘和 404 已接通基础守卫。',
  },
  {
    title: '导航项',
    value: '1',
    note: '当前只保留 MVP 所需的仪表盘入口。',
  },
  {
    title: '权限码',
    value: '1',
    note: '通过 route meta 与 auth store 建立最小约束。',
  },
];
</script>

<style scoped>
.dashboard-page {
  display: grid;
  gap: 20px;
}

.dashboard-page__hero-card {
  overflow: hidden;
  border-radius: 24px;
  background:
    radial-gradient(circle at top right, rgba(0, 82, 217, 0.16), transparent 24%),
    linear-gradient(135deg, #f8fbff 0%, #edf5ff 100%);
}

.dashboard-page__hero-copy {
  display: flex;
  justify-content: space-between;
  gap: 24px;
}

.dashboard-page__eyebrow {
  margin: 0 0 12px;
  color: #0052d9;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.dashboard-page__hero-copy h3 {
  margin: 0;
  color: #1a2433;
  font-size: clamp(28px, 4vw, 36px);
}

.dashboard-page__summary {
  margin: 16px 0 0;
  max-width: 640px;
  color: #607086;
  line-height: 1.8;
}

.dashboard-page__hero-tags {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  flex-wrap: wrap;
}

.dashboard-page__metrics {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.dashboard-page__metric-card {
  border-radius: 20px;
}

.dashboard-page__metric-label {
  display: block;
  color: #6c7d93;
  font-size: 13px;
}

.dashboard-page__metric-value {
  display: block;
  margin-top: 12px;
  color: #1a2433;
  font-size: 40px;
  line-height: 1;
}

.dashboard-page__metric-note {
  margin: 12px 0 0;
  color: #607086;
  line-height: 1.7;
}

.dashboard-page__grid {
  display: grid;
  gap: 20px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.dashboard-page__list {
  margin: 0;
  padding-left: 20px;
  color: #44556b;
  line-height: 1.8;
}

@media (max-width: 960px) {
  .dashboard-page__hero-copy,
  .dashboard-page__metrics,
  .dashboard-page__grid {
    grid-template-columns: 1fr;
  }

  .dashboard-page__hero-copy {
    flex-direction: column;
  }
}
</style>
