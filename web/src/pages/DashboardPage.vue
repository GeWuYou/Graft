<template>
  <div class="dashboard-page">
    <section class="dashboard-page__hero">
      <t-card class="dashboard-page__hero-card" :bordered="false">
        <div class="dashboard-page__hero-copy">
          <div>
            <p class="dashboard-page__eyebrow">
              {{ t('dashboard.hero.eyebrow') }}
            </p>
            <h3>{{ welcomeTitle }}</h3>
            <p class="dashboard-page__summary">
              {{ t('dashboard.hero.summary') }}
            </p>
          </div>

          <div class="dashboard-page__hero-tags">
            <t-tag theme="primary">{{ t('dashboard.tags.stack') }}</t-tag>
            <t-tag theme="success">{{ t('dashboard.tags.ui') }}</t-tag>
            <t-tag theme="warning">{{ t('dashboard.tags.permission') }}</t-tag>
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
      <t-card
        :title="t('dashboard.sections.capabilities.title')"
        :bordered="false"
      >
        <ul class="dashboard-page__list">
          <li v-for="item in capabilityItems" :key="item">
            {{ item }}
          </li>
        </ul>
      </t-card>

      <t-card
        :title="t('dashboard.sections.nextSteps.title')"
        :bordered="false"
      >
        <ul class="dashboard-page__list">
          <li v-for="item in nextStepItems" :key="item">
            {{ item }}
          </li>
        </ul>
      </t-card>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

import { useI18n } from '@/app/i18n';
import { useAuthStore } from '@/stores/auth';

const authStore = useAuthStore();
const { t } = useI18n();

const welcomeTitle = computed(() =>
  t('dashboard.hero.title', {
    values: {
      userName: authStore.userName || t('dashboard.hero.defaultUser'),
    },
  }),
);

const metrics = computed(() => [
  {
    title: t('dashboard.metrics.routes.title'),
    value: '3',
    note: t('dashboard.metrics.routes.note'),
  },
  {
    title: t('dashboard.metrics.navigation.title'),
    value: '1',
    note: t('dashboard.metrics.navigation.note'),
  },
  {
    title: t('dashboard.metrics.permissions.title'),
    value: '1',
    note: t('dashboard.metrics.permissions.note'),
  },
]);

const capabilityItems = computed(() => [
  t('dashboard.sections.capabilities.items.layout'),
  t('dashboard.sections.capabilities.items.guard'),
  t('dashboard.sections.capabilities.items.menu'),
]);

const nextStepItems = computed(() => [
  t('dashboard.sections.nextSteps.items.session'),
  t('dashboard.sections.nextSteps.items.router'),
  t('dashboard.sections.nextSteps.items.modules'),
]);
</script>

<style scoped>
.dashboard-page {
  display: grid;
  gap: 20px;
}

.dashboard-page__hero-card {
  background:
    radial-gradient(circle at top right, rgb(0 82 217 / 16%), transparent 24%),
    linear-gradient(135deg, #f8fbff 0%, #edf5ff 100%);
  border-radius: 24px;
  overflow: hidden;
}

.dashboard-page__hero-copy {
  display: flex;
  gap: 24px;
  justify-content: space-between;
}

.dashboard-page__eyebrow {
  color: #0052d9;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.08em;
  margin: 0 0 12px;
  text-transform: uppercase;
}

.dashboard-page__hero-copy h3 {
  color: #1a2433;
  font-size: clamp(28px, 4vw, 36px);
  margin: 0;
}

.dashboard-page__summary {
  color: #607086;
  line-height: 1.8;
  margin: 16px 0 0;
  max-width: 640px;
}

.dashboard-page__hero-tags {
  align-items: flex-start;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
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
  color: #6c7d93;
  display: block;
  font-size: 13px;
}

.dashboard-page__metric-value {
  color: #1a2433;
  display: block;
  font-size: 40px;
  line-height: 1;
  margin-top: 12px;
}

.dashboard-page__metric-note {
  color: #607086;
  line-height: 1.7;
  margin: 12px 0 0;
}

.dashboard-page__grid {
  display: grid;
  gap: 20px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.dashboard-page__list {
  color: #44556b;
  line-height: 1.8;
  margin: 0;
  padding-left: 20px;
}

@media (width <= 960px) {
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
