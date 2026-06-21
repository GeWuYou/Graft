<template>
  <t-drawer
    :visible="visible"
    :footer="false"
    destroy-on-close
    placement="right"
    size="720px"
    @update:visible="$emit('update:visible', $event)"
  >
    <template #header>
      <div class="notification-detail__drawer-header">
        <h2>{{ t('notification.detail.title') }}</h2>
        <div v-if="item" class="notification-detail__drawer-actions">
          <t-button
            v-if="item.status === 'unread'"
            size="small"
            theme="default"
            variant="outline"
            :loading="markingRead"
            @click="$emit('mark-read', item)"
          >
            {{ t('notification.action.markRead') }}
          </t-button>
          <t-tag v-else theme="default" variant="light" size="small">
            {{ notificationView(item).statusLabel }}
          </t-tag>
        </div>
      </div>
    </template>

    <div v-if="item" class="notification-detail">
      <section class="notification-detail__section">
        <div class="notification-detail__title-row">
          <div>
            <h3>{{ notificationView(item).title }}</h3>
            <p>{{ notificationView(item).message }}</p>
          </div>
          <div class="notification-detail__level">
            <t-tag :theme="notificationSeverityTheme(item.severity)" variant="light-outline">
              {{ notificationView(item).levelLabel }}
            </t-tag>
          </div>
        </div>
      </section>

      <section class="notification-detail__section">
        <h4>{{ t('notification.detail.basic') }}</h4>
        <dl class="notification-detail__grid">
          <dt>{{ t('notification.columns.status') }}</dt>
          <dd>
            <t-tag :theme="notificationStatusTheme(item.status)" variant="light" size="small">
              {{ notificationView(item).statusLabel }}
            </t-tag>
          </dd>
          <dt>{{ t('notification.columns.severity') }}</dt>
          <dd>{{ notificationView(item).levelLabel }}</dd>
          <dt>{{ t('notification.columns.category') }}</dt>
          <dd>{{ notificationView(item).categoryLabel }}</dd>
          <dt>{{ t('notification.columns.sourceModule') }}</dt>
          <dd>{{ notificationView(item).sourceLabel }}</dd>
          <dt>{{ t('notification.columns.occurredAt') }}</dt>
          <dd>{{ notificationView(item).occurredAtLabel }}</dd>
          <dt>{{ t('notification.detail.readAt') }}</dt>
          <dd>{{ notificationView(item).readAtLabel }}</dd>
        </dl>
      </section>

      <section class="notification-detail__section">
        <h4>{{ t('notification.detail.resource') }}</h4>
        <dl class="notification-detail__grid">
          <dt>{{ t('notification.detail.resourceName') }}</dt>
          <dd>{{ notificationView(item).resourceName }}</dd>
          <dt>{{ t('notification.detail.resourceType') }}</dt>
          <dd>{{ notificationView(item).resourceTypeLabel }}</dd>
          <dt>{{ t('notification.detail.resourceId') }}</dt>
          <dd>{{ notificationView(item).resourceId }}</dd>
          <dt>{{ t('notification.detail.resultSummary') }}</dt>
          <dd>{{ resolveNotificationResultSummary(item, t) }}</dd>
        </dl>
      </section>

      <section class="notification-detail__section">
        <h4>{{ t('notification.detail.navigation') }}</h4>
        <div class="notification-detail__navigation">
          <t-tag variant="light">{{ navigationKindLabel }}</t-tag>
          <t-button v-if="canNavigate" theme="primary" @click="$emit('navigate', item)">
            {{ notificationView(item).actionLabel }}
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

import { NOTIFICATION_NAVIGATION_KIND, resolveNotificationNavigationLocation } from '../contract/navigation';
import {
  notificationSeverityTheme,
  notificationStatusTheme,
  presentNotification,
  resolveNotificationResultSummary,
} from '../shared/presentation';
import type { NotificationItem } from '../types/notification';

const props = defineProps<{
  item: NotificationItem | null;
  markingRead?: boolean;
  visible: boolean;
}>();

defineEmits<{
  (e: 'mark-read', row: NotificationItem): void;
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

function notificationView(item: NotificationItem) {
  return presentNotification(item, t, locale.value);
}
</script>
<style scoped lang="less">
.notification-detail {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.notification-detail__drawer-header {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
  min-width: 0;
  width: 100%;
}

.notification-detail__drawer-header h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
  min-width: 0;
}

.notification-detail__drawer-actions {
  align-items: center;
  display: flex;
  flex: 0 0 auto;
  gap: var(--graft-density-gap-8);
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

.notification-detail__level {
  align-items: flex-end;
  display: flex;
  flex: 0 0 auto;
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
