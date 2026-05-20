<template>
  <div class="monitor-page">
    <t-row :gutter="[24, 24]">
      <t-col :span="12">
        <t-card class="summary-card" :bordered="false" :title="t('monitor.serverStatus.summaryTitle')">
          <div class="summary-metric">
            <span class="summary-metric__label">{{ t('monitor.serverStatus.statusLabel') }}</span>
            <span class="summary-metric__value">{{ statusLabel(serverStatus?.status) }}</span>
          </div>
          <div class="summary-hint">{{ t('monitor.serverStatus.summaryHint') }}</div>
        </t-card>
      </t-col>
      <t-col :span="12">
        <t-card class="summary-card" :bordered="false" :title="t('monitor.serverStatus.endpointTitle')">
          <div class="summary-meta">
            <span
              >{{ t('monitor.serverStatus.endpointLabel') }}<code>{{ apiPath }}</code></span
            >
            <span
              >{{ t('monitor.serverStatus.fieldsLabel') }}<code>{{ t('monitor.serverStatus.fieldsValue') }}</code></span
            >
            <span v-if="serverStatus">{{
              t('monitor.serverStatus.lastObserved', { time: serverStatus.observed_at })
            }}</span>
          </div>
          <div class="summary-actions">
            <t-button theme="primary" variant="outline" :loading="loading" @click="fetchServerStatus">
              {{ t('monitor.serverStatus.refresh') }}
            </t-button>
          </div>
        </t-card>
      </t-col>
    </t-row>

    <t-row :gutter="[24, 24]">
      <t-col :span="8">
        <t-card class="detail-card" :bordered="false" :title="t('monitor.serverStatus.serverCardTitle')">
          <div v-if="serverStatus" class="detail-grid">
            <span class="detail-grid__label"
              >{{ t('monitor.serverStatus.versionLabel') }}:
              <strong class="detail-grid__value">{{ serverStatus.server.version }}</strong></span
            >
            <span class="detail-grid__label"
              >{{ t('monitor.serverStatus.startedAtLabel') }}:
              <strong class="detail-grid__value">{{ serverStatus.server.started_at }}</strong></span
            >
            <span class="detail-grid__label"
              >{{ t('monitor.serverStatus.uptimeLabel') }}:
              <strong class="detail-grid__value">{{ formatUptime(serverStatus.server.uptime_seconds) }}</strong></span
            >
            <span class="detail-grid__label"
              >{{ t('monitor.serverStatus.goVersionLabel') }}:
              <strong class="detail-grid__value">{{ serverStatus.server.go_version }}</strong></span
            >
            <span class="detail-grid__label"
              >{{ t('monitor.serverStatus.appLabel') }}:
              <strong class="detail-grid__value">{{ serverStatus.server.app_name || '-' }}</strong></span
            >
            <span class="detail-grid__label"
              >{{ t('monitor.serverStatus.envLabel') }}:
              <strong class="detail-grid__value">{{ serverStatus.server.app_env || '-' }}</strong></span
            >
          </div>
          <t-empty v-else :description="t('monitor.serverStatus.empty')" />
        </t-card>
      </t-col>
      <t-col :span="8">
        <t-card class="detail-card" :bordered="false" :title="t('monitor.serverStatus.dependencyCardTitle')">
          <div v-if="serverStatus" class="dependency-list">
            <div class="dependency-item">
              <span>{{ t('monitor.serverStatus.databaseLabel') }}</span>
              <t-tag :theme="statusTheme(serverStatus.dependencies.database.status)" variant="light">{{
                statusLabel(serverStatus.dependencies.database.status)
              }}</t-tag>
            </div>
            <div class="dependency-item">
              <span>{{ t('monitor.serverStatus.redisLabel') }}</span>
              <t-tag :theme="statusTheme(serverStatus.dependencies.redis.status)" variant="light">
                {{ statusLabel(serverStatus.dependencies.redis.status) }}
              </t-tag>
            </div>
          </div>
          <t-empty v-else :description="t('monitor.serverStatus.empty')" />
        </t-card>
      </t-col>
      <t-col :span="8">
        <t-card class="table-card" :bordered="false" :title="t('monitor.serverStatus.pluginCardTitle')">
          <t-table
            row-key="name"
            :data="pluginRows"
            :columns="columns"
            :loading="loading"
            size="medium"
            table-layout="fixed"
          >
            <template #status="{ row }">
              <t-tag :theme="statusTheme(row.status)" variant="light">{{ statusLabel(row.status) }}</t-tag>
            </template>
            <template #empty>
              <t-empty :description="t('monitor.serverStatus.empty')" />
            </template>
          </t-table>
        </t-card>
      </t-col>
    </t-row>
  </div>
</template>
<script setup lang="ts">
import type { TdBaseTableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { getServerStatus } from '../api/server-status';
import { MONITOR_API_PATH } from '../contract/paths';
import type { ServerStatusPlugin, ServerStatusResponse } from '../types/server-status';

defineOptions({
  name: 'MonitorServerStatusIndex',
});

const { t, locale } = useI18n();
const loading = ref(false);
const serverStatus = ref<ServerStatusResponse | null>(null);
const apiPath = MONITOR_API_PATH.SERVER_STATUS;

const pluginRows = computed<ServerStatusPlugin[]>(() => serverStatus.value?.plugins ?? []);

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;

  return [
    {
      title: t('monitor.serverStatus.pluginName'),
      colKey: 'name',
    },
    {
      title: t('monitor.serverStatus.pluginVersion'),
      colKey: 'version',
    },
    {
      title: t('monitor.serverStatus.pluginStatus'),
      colKey: 'status',
    },
  ];
});

async function fetchServerStatus() {
  loading.value = true;
  try {
    serverStatus.value = await getServerStatus();
  } catch (error) {
    serverStatus.value = null;
    const fallbackMessage = t('monitor.serverStatus.loadFailed');
    const message = error instanceof Error && error.message.trim() ? error.message : fallbackMessage;
    MessagePlugin.error(message);
  } finally {
    loading.value = false;
  }
}

function statusLabel(status?: string) {
  switch (status) {
    case 'healthy':
      return t('monitor.serverStatus.statusHealthy');
    case 'degraded':
      return t('monitor.serverStatus.statusDegraded');
    case 'disabled':
      return t('monitor.serverStatus.statusDisabled');
    default:
      return t('monitor.serverStatus.statusUnknown');
  }
}

function statusTheme(status?: string) {
  switch (status) {
    case 'healthy':
      return 'success';
    case 'degraded':
      return 'warning';
    case 'disabled':
      return 'danger';
    default:
      return 'default';
  }
}

function formatUptime(totalSeconds: number) {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  return `${hours}h ${minutes}m ${seconds}s`;
}

onMounted(() => {
  fetchServerStatus();
});
</script>
<style lang="less" scoped>
@import './index.less';
</style>
