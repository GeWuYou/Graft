<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-drawer
    :visible="visible"
    :header="t('notification.detail.title')"
    :footer="false"
    destroy-on-close
    placement="right"
    size="720px"
    @update:visible="$emit('update:visible', $event)"
  >
    <div v-if="item" class="notification-detail">
      <section class="notification-detail__section">
        <div class="notification-detail__title-row">
          <div>
            <h3>{{ notificationTitle(item, t) }}</h3>
            <p>{{ notificationMessage(item, t) }}</p>
          </div>
          <t-tag :theme="notificationSeverityTheme(item.severity)" variant="light-outline">
            {{ t(`notification.severity.${item.severity}`) }}
          </t-tag>
        </div>
      </section>

      <section class="notification-detail__section">
        <h4>{{ t('notification.detail.basic') }}</h4>
        <dl class="notification-detail__grid">
          <dt>{{ t('notification.columns.status') }}</dt>
          <dd>
            <t-tag :theme="notificationStatusTheme(item.status)" variant="light" size="small">
              {{ t(`notification.status.${item.status}`) }}
            </t-tag>
          </dd>
          <dt>{{ t('notification.columns.category') }}</dt>
          <dd>{{ t(`notification.category.${item.category}`) }}</dd>
          <dt>{{ t('notification.columns.sourceModule') }}</dt>
          <dd>{{ notificationSourceLabel(item.source_module, t) }}</dd>
          <dt>{{ t('notification.detail.eventType') }}</dt>
          <dd>{{ item.event_type }}</dd>
          <dt>{{ t('notification.columns.occurredAt') }}</dt>
          <dd>{{ formatCompactDateTime(item.occurred_at, locale) }}</dd>
          <dt>{{ t('notification.detail.readAt') }}</dt>
          <dd>{{ item.read_at ? formatCompactDateTime(item.read_at, locale) : t('notification.values.notRead') }}</dd>
        </dl>
      </section>

      <section class="notification-detail__section">
        <h4>{{ t('notification.detail.resource') }}</h4>
        <dl class="notification-detail__grid">
          <dt>{{ t('notification.detail.resourceType') }}</dt>
          <dd>{{ item.resource_type || t('notification.values.emptyField') }}</dd>
          <dt>{{ t('notification.detail.resourceId') }}</dt>
          <dd>{{ item.resource_id || t('notification.values.emptyField') }}</dd>
          <dt>{{ t('notification.detail.resourceName') }}</dt>
          <dd>{{ item.resource_name || t('notification.values.emptyField') }}</dd>
          <dt>{{ t('notification.detail.targetType') }}</dt>
          <dd>{{ item.target_type }}</dd>
          <dt>{{ t('notification.detail.targetRef') }}</dt>
          <dd>{{ item.target_ref || t('notification.values.emptyField') }}</dd>
        </dl>
      </section>

      <section class="notification-detail__section">
        <h4>{{ t('notification.detail.navigation') }}</h4>
        <div class="notification-detail__navigation">
          <t-tag variant="light">{{ navigationKindLabel }}</t-tag>
          <t-button v-if="canNavigate" theme="primary" @click="$emit('navigate', item)">
            {{ t('notification.actions.openTarget') }}
          </t-button>
          <span v-else>{{ t('notification.detail.unsupportedNavigation') }}</span>
        </div>
      </section>
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { formatCompactDateTime } from '@/shared/components/management';

import { NOTIFICATION_NAVIGATION_KIND, resolveNotificationNavigationLocation } from '../contract/navigation';
import {
  notificationMessage,
  notificationSeverityTheme,
  notificationSourceLabel,
  notificationStatusTheme,
  notificationTitle,
} from '../shared/presentation';
import type { NotificationItem } from '../types/notification';

const props = defineProps<{
  item: NotificationItem | null;
  visible: boolean;
}>();

defineEmits<{
  (e: 'navigate', row: NotificationItem): void;
  (e: 'update:visible', value: boolean): void;
}>();

const { t, locale } = useI18n();
const canNavigate = computed(() => Boolean(props.item && resolveNotificationNavigationLocation(props.item.navigation)));
const navigationKindLabel = computed(() => {
  const kind = props.item?.navigation.kind;
  if (!kind) {
    return t('notification.navigation.unknown');
  }

  const key = NOTIFICATION_NAVIGATION_LABEL_KEYS[kind];
  return key ? t(key) : t('notification.navigation.unknown');
});

const NOTIFICATION_NAVIGATION_LABEL_KEYS = {
  [NOTIFICATION_NAVIGATION_KIND.AUDIT_INCIDENT]: 'notification.navigation.auditIncident',
  [NOTIFICATION_NAVIGATION_KIND.AUDIT_LOG]: 'notification.navigation.auditLog',
  [NOTIFICATION_NAVIGATION_KIND.SCHEDULER_RUN]: 'notification.navigation.schedulerRun',
  [NOTIFICATION_NAVIGATION_KIND.SYSTEM_CONFIG_ITEM]: 'notification.navigation.systemConfigItem',
  [NOTIFICATION_NAVIGATION_KIND.MODULE_RUNTIME_ITEM]: 'notification.navigation.moduleRuntimeItem',
} as const;
</script>
<style scoped lang="less">
.notification-detail {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.notification-detail__section {
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-default);
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
}

.notification-detail__section h4,
.notification-detail__title-row h3,
.notification-detail__title-row p {
  margin: 0;
}

.notification-detail__section h4 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin-bottom: var(--graft-density-gap-12);
}

.notification-detail__title-row {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
}

.notification-detail__title-row h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.notification-detail__title-row p {
  color: var(--td-text-color-secondary);
  line-height: 1.7;
  margin-top: var(--graft-density-gap-8);
}

.notification-detail__grid {
  display: grid;
  gap: var(--graft-density-gap-10) var(--graft-density-gap-16);
  grid-template-columns: 140px minmax(0, 1fr);
  margin: 0;
}

.notification-detail__grid dt {
  color: var(--td-text-color-secondary);
}

.notification-detail__grid dd {
  color: var(--td-text-color-primary);
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

.notification-detail__navigation {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
}

.notification-detail__navigation span {
  color: var(--td-text-color-secondary);
}
</style>
