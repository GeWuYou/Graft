<template>
  <section class="assignment-header">
    <div class="assignment-header__identity">
      <div class="assignment-header__avatar">{{ avatarText }}</div>
      <div class="assignment-header__copy">
        <p v-if="eyebrow" class="assignment-header__eyebrow">{{ eyebrow }}</p>
        <h2 class="assignment-header__title">{{ title }}</h2>
        <p v-if="subtitle" class="assignment-header__subtitle">{{ subtitle }}</p>
        <p v-if="description" class="assignment-header__description">{{ description }}</p>
      </div>
    </div>

    <div class="assignment-header__aside">
      <div v-if="badges.length > 0" class="assignment-header__badges">
        <t-tag
          v-for="badge in badges"
          :key="`${badge.label}-${badge.theme ?? 'default'}`"
          :theme="badge.theme ?? 'default'"
          :variant="badge.variant ?? 'light'"
          size="small"
        >
          {{ badge.label }}
        </t-tag>
      </div>

      <div v-if="stats.length > 0" class="assignment-header__stats">
        <div v-for="stat in stats" :key="stat.label" class="assignment-header__stat">
          <span class="assignment-header__stat-value">{{ stat.value }}</span>
          <span class="assignment-header__stat-label">{{ stat.label }}</span>
        </div>
      </div>
    </div>
  </section>
</template>
<script setup lang="ts">
export type AssignmentHeaderBadge = {
  label: string;
  theme?: 'danger' | 'default' | 'primary' | 'success' | 'warning';
  variant?: 'dark' | 'light' | 'light-outline' | 'outline';
};

export type AssignmentHeaderStat = {
  label: string;
  value: number | string;
};

withDefaults(
  defineProps<{
    avatarText: string;
    badges?: AssignmentHeaderBadge[];
    description?: string;
    eyebrow?: string;
    stats?: AssignmentHeaderStat[];
    subtitle?: string;
    title: string;
  }>(),
  {
    badges: () => [],
    description: '',
    eyebrow: '',
    stats: () => [],
    subtitle: '',
  },
);
</script>
<style scoped lang="less">
.assignment-header,
.assignment-header__identity,
.assignment-header__copy,
.assignment-header__aside,
.assignment-header__badges,
.assignment-header__stats,
.assignment-header__stat {
  display: flex;
}

.assignment-header {
  align-items: flex-start;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--td-brand-color) 10%, var(--td-bg-color-container)) 0%,
    var(--td-bg-color-container) 100%
  );
  border: 1px solid color-mix(in srgb, var(--td-brand-color) 14%, var(--td-component-stroke));
  border-radius: var(--td-radius-large);
  gap: var(--td-comp-margin-xl);
  justify-content: space-between;
  padding: var(--td-comp-paddingTB-xl) var(--td-comp-paddingLR-xl);
}

.assignment-header__identity {
  align-items: flex-start;
  gap: var(--td-comp-margin-l);
  min-width: 0;
}

.assignment-header__avatar {
  align-items: center;
  background: color-mix(in srgb, var(--td-brand-color) 16%, var(--td-bg-color-container));
  border: 1px solid color-mix(in srgb, var(--td-brand-color) 24%, var(--td-component-stroke));
  border-radius: 999px;
  color: var(--td-brand-color);
  display: inline-flex;
  flex: 0 0 52px;
  font: var(--td-font-title-large);
  height: 52px;
  justify-content: center;
  width: 52px;
}

.assignment-header__copy {
  flex: 1;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.assignment-header__eyebrow,
.assignment-header__subtitle,
.assignment-header__description,
.assignment-header__stat-label {
  color: var(--td-text-color-secondary);
}

.assignment-header__eyebrow,
.assignment-header__subtitle,
.assignment-header__description {
  margin: 0;
}

.assignment-header__eyebrow {
  font: var(--td-font-body-small);
}

.assignment-header__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  margin: 0;
}

.assignment-header__subtitle,
.assignment-header__description {
  font: var(--td-font-body-medium);
}

.assignment-header__aside {
  align-items: flex-end;
  flex-direction: column;
  gap: var(--td-comp-margin-l);
}

.assignment-header__badges {
  flex-wrap: wrap;
  gap: var(--td-comp-margin-s);
  justify-content: flex-end;
}

.assignment-header__stats {
  gap: var(--td-comp-margin-l);
}

.assignment-header__stat {
  align-items: flex-end;
  flex-direction: column;
  gap: 2px;
}

.assignment-header__stat-value {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  white-space: nowrap;
}

.assignment-header__stat-label {
  font: var(--td-font-body-small);
  white-space: nowrap;
}

@media (width <= 768px) {
  .assignment-header,
  .assignment-header__aside {
    align-items: stretch;
    flex-direction: column;
  }

  .assignment-header__badges {
    justify-content: flex-start;
  }

  .assignment-header__stats {
    flex-wrap: wrap;
  }

  .assignment-header__stat {
    align-items: flex-start;
  }
}
</style>
