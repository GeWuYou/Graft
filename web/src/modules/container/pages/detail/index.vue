<template>
  <div class="container-detail-page" data-page-type="operations-detail">
    <management-page-header
      :title="pageTitle"
      :description="safeDetail ? safeDetail.image : t('container.detail.description')"
      :source="{ labelKey: 'container.list.eyebrow', fallback: t('container.list.eyebrow') }"
    >
      <template #actions>
        <t-button
          v-if="safeDetail && permissionStore.hasPermission(auditPermissionCodes.READ)"
          data-testid="container-detail-view-audit"
          theme="default"
          variant="outline"
          size="small"
          @click="openAuditLogs"
        >
          {{ t('container.detail.viewAudit') }}
        </t-button>
      </template>
      <template #meta>
        <div class="container-detail-header-meta" data-testid="container-detail-header-meta">
          <t-space class="container-detail-header-meta__tags" break-line size="small">
            <span v-if="safeDetail" class="container-detail-header-id">{{ shortContainerId(safeDetail) }}</span>
            <t-tag v-if="safeDetail" :theme="stateTheme(safeDetail.state)" variant="light-outline">
              {{ stateLabel(safeDetail.state) }}
            </t-tag>
            <t-tag v-if="safeDetail" :theme="healthTheme(safeDetail.health)" variant="light-outline">
              {{ healthLabel(safeDetail.health) }}
            </t-tag>
            <t-tag v-if="safeDetail?.runtime" theme="default" variant="light-outline">
              {{ safeDetail.runtime }}
            </t-tag>
          </t-space>
          <div v-if="safeDetail?.inspect_updated_at" class="container-detail-header-meta__updated-at">
            {{ t('container.detail.inspectUpdatedAt') }}: {{ formatTime(safeDetail.inspect_updated_at) }}
          </div>
        </div>
      </template>
    </management-page-header>

    <section class="container-detail-body">
      <t-loading v-if="detailRefreshing && !safeDetail && !error" class="container-detail-state" :loading="true">
        <t-skeleton animation="gradient" theme="article" />
      </t-loading>

      <t-alert v-else-if="error" class="container-detail-state-alert" theme="error" :title="error">
        <template #operation>
          <t-button theme="danger" variant="text" @click="refreshContainerDetail">
            {{ t('container.list.retry') }}
          </t-button>
        </template>
      </t-alert>

      <template v-else-if="safeDetail">
        <section class="container-detail-summary">
          <t-card
            class="container-detail-summary-card container-detail-summary-card--identity"
            size="small"
            :bordered="false"
            :title="t('container.detail.summary.identity')"
          >
            <div class="container-detail-summary-list">
              <div class="container-detail-kv">
                <span>{{ t('container.list.fields.name') }}</span>
                <strong>{{ displayName(safeDetail) }}</strong>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.list.fields.image') }}</span>
                <copyable-detail-value
                  :copy-label="t('container.detail.copy')"
                  :value="safeDetail.image"
                  :display-value="safeDetail.image"
                  @copy="copyDetailText"
                />
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.list.fields.id') }}</span>
                <copyable-detail-value
                  :value="safeDetail.id"
                  :display-value="shortContainerId(safeDetail)"
                  :copy-label="t('container.detail.copy')"
                  code
                  data-testid="summary-container-id-copy"
                  @copy="copyDetailText"
                />
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.list.fields.runtime') }}</span>
                <strong>{{ runtimeLabel(safeDetail) }}</strong>
              </div>
            </div>
          </t-card>
          <t-card
            class="container-detail-summary-card container-detail-summary-card--runtime"
            size="small"
            :bordered="false"
            :title="t('container.detail.summary.runtime')"
          >
            <div class="container-detail-summary-list">
              <div class="container-detail-kv container-detail-kv--inline">
                <span>{{ t('container.list.fields.status') }}</span>
                <t-tag :theme="stateTheme(safeDetail.state)" variant="light-outline">
                  {{ stateLabel(safeDetail.state) }}
                </t-tag>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.list.fields.state') }}</span>
                <code>{{ safeDetail.state || '-' }}</code>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.list.fields.startedAt') }}</span>
                <strong>{{ formatTime(safeDetail.started_at) }}</strong>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.detail.health.status') }}</span>
                <strong>{{ healthLabel(safeDetail.health) }}</strong>
              </div>
            </div>
          </t-card>
          <t-card
            class="container-detail-summary-card container-detail-summary-card--source"
            size="small"
            :bordered="false"
            :title="t('container.detail.summary.source')"
          >
            <div class="container-detail-summary-list">
              <div class="container-detail-kv container-detail-kv--inline">
                <span>{{ t('container.detail.source.type') }}</span>
                <t-tag :theme="orchestratorTheme(safeDetail)" variant="light-outline">
                  {{ orchestratorLabel(readOrchestratorType(safeDetail)) }}
                </t-tag>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.detail.source.summary') }}</span>
                <strong>{{ orchestratorSummary(safeDetail) }}</strong>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.detail.source.actionPolicy') }}</span>
                <strong>{{ actionLevelLabel(safeDetail) }}</strong>
              </div>
              <div v-if="orchestratorRiskHint(safeDetail)" class="container-detail-kv container-detail-kv--full">
                <t-alert theme="warning" :message="orchestratorRiskHint(safeDetail)" />
              </div>
            </div>
          </t-card>
          <t-card
            class="container-detail-summary-card container-detail-summary-card--resources"
            size="small"
            :bordered="false"
            :title="t('container.detail.summary.resources')"
          >
            <div class="container-detail-summary__resource">
              <div
                class="container-detail-resource-meter container-detail-resource-meter--cpu"
                :class="resourceChange.cpuClass"
                data-testid="container-detail-summary-cpu"
              >
                <div class="container-detail-resource-meter__content">
                  <span>{{ t('container.detail.resources.cpu') }}</span>
                  <strong>{{ formatPercent(safeDetail.resource?.cpu_percent) }}</strong>
                </div>
                <t-progress
                  theme="circle"
                  size="small"
                  :label="formatPercent(safeDetail.resource?.cpu_percent)"
                  :percentage="toProgressPercent(safeDetail.resource?.cpu_percent)"
                  :status="resourceChange.cpuStatus"
                />
              </div>
              <div
                class="container-detail-resource-meter container-detail-resource-meter--memory"
                :class="resourceChange.memoryClass"
                data-testid="container-detail-summary-memory"
              >
                <div class="container-detail-resource-meter__content">
                  <span>{{ t('container.detail.resources.memory') }}</span>
                  <strong>{{ memorySummary(safeDetail) }}</strong>
                  <em>{{ formatPercent(safeDetail.resource?.memory_percent) }}</em>
                </div>
                <t-progress
                  theme="line"
                  size="small"
                  :label="false"
                  :percentage="toProgressPercent(safeDetail.resource?.memory_percent)"
                  :status="resourceChange.memoryStatus"
                />
              </div>
            </div>
          </t-card>
          <t-card
            class="container-detail-summary-card container-detail-summary-card--network"
            size="small"
            :bordered="false"
            :title="t('container.detail.summary.network')"
          >
            <div class="container-detail-summary-list">
              <div class="container-detail-kv">
                <span>{{ t('container.detail.network.primaryIp') }}</span>
                <strong>{{ safeDetail.primary_ip || '-' }}</strong>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.detail.network.summary') }}</span>
                <strong>{{ networkSummary(safeDetail) }}</strong>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.detail.network.ports') }}</span>
                <strong v-if="safeDetail.ports.length">
                  {{ t('container.detail.network.portCount', { count: safeDetail.ports.length }) }}
                </strong>
                <strong v-else>{{ t('container.detail.network.noPublicPorts') }}</strong>
              </div>
            </div>
          </t-card>
        </section>

        <div
          v-if="activeTab !== 'logs'"
          class="container-detail-realtime-row"
          data-testid="container-detail-realtime-row"
        >
          <t-tooltip :content="t('container.detail.refreshTooltip')">
            <div class="container-detail-realtime-bar" data-testid="container-detail-realtime-bar">
              <span class="container-detail-realtime-bar__label">{{ t('container.detail.realtime.label') }}</span>
              <t-tag
                :theme="realtimeStatusTheme"
                variant="light-outline"
                data-testid="container-detail-realtime-status"
              >
                {{ realtimeStatusLabel }}
              </t-tag>
              <span class="container-detail-realtime-bar__hint">{{ realtimeStatusHint }}</span>
              <div class="container-detail-realtime-bar__actions">
                <t-button
                  size="small"
                  theme="default"
                  variant="outline"
                  data-testid="container-detail-realtime-toggle"
                  @click="toggleRealtimeSubscription"
                >
                  {{ realtimeToggleLabel }}
                </t-button>
                <t-button
                  size="small"
                  theme="primary"
                  variant="outline"
                  :loading="detailRefreshing"
                  data-refresh-now="true"
                  @click="handleManualRefresh"
                >
                  {{ t('container.detail.refreshNow') }}
                </t-button>
              </div>
            </div>
          </t-tooltip>
        </div>

        <t-card class="container-detail-tabs-card" :bordered="true">
          <t-tabs v-model:value="activeTab" theme="card" @change="handleTabChange">
            <t-tab-panel value="overview" :label="t('container.detail.tabs.overview')" :destroy-on-hide="false">
              <section class="container-detail-section container-detail-section--overview">
                <container-overview-panel
                  :copy-label="t('container.detail.copy')"
                  :sections="overviewSections"
                  @copy="copyDetailText"
                />
              </section>
            </t-tab-panel>

            <t-tab-panel value="resources" :label="t('container.detail.tabs.resources')" :destroy-on-hide="false">
              <section class="container-detail-section">
                <div class="container-detail-resource-grid">
                  <metric-card
                    :title="t('container.detail.resources.cpu')"
                    :value="resourceMetrics.cpu.value"
                    :description="resourceMetrics.cpu.description"
                    :progress="resourceMetrics.cpu.progress"
                    :progress-label="resourceMetrics.cpu.progressLabel"
                  />
                  <metric-card
                    :title="t('container.detail.resources.memory')"
                    :value="resourceMetrics.memory.value"
                    :description="resourceMetrics.memory.description"
                    :progress="resourceMetrics.memory.progress"
                    :progress-label="resourceMetrics.memory.progressLabel"
                  />
                  <article class="container-detail-resource-status-card">
                    <div class="container-detail-resource-status-card__content">
                      <span class="container-detail-resource-status-card__title">
                        {{ t('container.detail.resources.status') }}
                      </span>
                      <strong>{{ resourceMetrics.status.value }}</strong>
                      <span>{{ resourceMetrics.status.description }}</span>
                    </div>
                    <t-tag :theme="resourceMetrics.status.theme" variant="light-outline">
                      {{ resourceMetrics.status.value }}
                    </t-tag>
                  </article>
                </div>

                <section class="container-resource-dashboard-section">
                  <div class="container-resource-dashboard-section__title">
                    {{ t('container.detail.resources.dashboard') }}
                  </div>
                  <div class="container-resource-dashboard-grid">
                    <article
                      class="container-resource-dashboard-panel"
                      :class="resourceChange.cpuClass"
                      data-testid="container-detail-dashboard-cpu"
                    >
                      <div class="container-resource-dashboard-panel__heading">
                        <span>{{ t('container.detail.resources.cpuUsageRate') }}</span>
                        <strong>{{ resourceMetrics.cpu.value }}</strong>
                      </div>
                      <t-progress
                        theme="line"
                        size="small"
                        :label="false"
                        :percentage="resourceMetrics.cpu.progress"
                        :status="resourceChange.cpuStatus"
                      />
                      <div class="container-resource-dashboard-panel__meta">
                        <span>
                          {{ t('container.detail.resources.cpuLimit') }}
                          <strong>{{ notCollectedLabel() }}</strong>
                        </span>
                        <span>
                          {{ t('container.detail.resources.onlineCpus') }}
                          <strong>{{ readCpuCountText() }}</strong>
                        </span>
                        <span>
                          {{ t('container.detail.resources.systemCpuUsage') }}
                          <strong>{{ readCpuSystemTimeText() }}</strong>
                        </span>
                      </div>
                    </article>

                    <article
                      class="container-resource-dashboard-panel"
                      :class="resourceChange.memoryClass"
                      data-testid="container-detail-dashboard-memory"
                    >
                      <div class="container-resource-dashboard-panel__heading">
                        <span>{{ t('container.detail.resources.memoryUsageRate') }}</span>
                        <strong>{{ resourceMetrics.memory.description }}</strong>
                      </div>
                      <t-progress
                        theme="line"
                        size="small"
                        :label="false"
                        :percentage="resourceMetrics.memory.progress"
                        :status="resourceChange.memoryStatus"
                      />
                      <div class="container-resource-dashboard-panel__usage">
                        {{ resourceMetrics.memory.value }}
                      </div>
                      <div class="container-resource-dashboard-panel__meta">
                        <span>
                          {{ t('container.detail.resources.memoryCache') }}
                          <strong>{{ readMetricText('memory_cache', 'bytes') }}</strong>
                        </span>
                        <span>
                          {{ t('container.detail.resources.memoryRss') }}
                          <strong>{{ readMetricText('memory_rss', 'bytes') }}</strong>
                        </span>
                        <span>
                          {{ t('container.detail.resources.memoryActiveFile') }}
                          <strong>{{ readMetricText('memory_active_file', 'bytes') }}</strong>
                        </span>
                        <span>
                          {{ t('container.detail.resources.memoryInactiveFile') }}
                          <strong>{{ readMetricText('memory_inactive_file', 'bytes') }}</strong>
                        </span>
                      </div>
                    </article>
                  </div>
                </section>

                <section class="container-resource-history-section">
                  <div class="container-resource-history-section__title">
                    {{ t('container.detail.resources.history') }}
                  </div>
                  <div v-if="resourceHistoryPoints.length" class="container-resource-history-grid">
                    <article
                      v-for="point in resourceHistoryPoints"
                      :key="point.key"
                      class="container-resource-history-card"
                    >
                      <span>{{ point.collectedAt }}</span>
                      <strong>{{ point.cpuPercent }}</strong>
                      <em>{{ point.memoryPercent }}</em>
                    </article>
                  </div>
                  <t-empty v-else size="small" :description="t('container.detail.resources.historyEmpty')" />
                </section>

                <section class="container-resource-detail-section">
                  <div class="container-resource-detail-section__title">
                    {{ t('container.detail.resources.detailedMetrics') }}
                  </div>
                  <div class="container-resource-detail-grid">
                    <article
                      v-for="group in resourceDetailGroups"
                      :key="group.key"
                      class="container-resource-detail-card"
                      :class="{
                        'container-resource-detail-card--cpu': group.key === 'cpu',
                        'container-resource-detail-card--memory': group.key === 'memory',
                      }"
                    >
                      <h3>{{ group.title }}</h3>
                      <div v-if="group.key === 'cpu'" class="container-resource-cpu-metric-grid">
                        <div
                          v-for="metric in cpuDetailMetrics"
                          :key="metric.key"
                          class="container-resource-cpu-metric"
                          :class="{
                            'container-resource-cpu-metric--muted': metric.muted,
                            'container-resource-cpu-metric--warning': metric.emphasized,
                          }"
                        >
                          <span class="container-resource-cpu-metric__label">{{ metric.label }}</span>
                          <strong class="container-resource-cpu-metric__value">{{ metric.value }}</strong>
                          <span v-if="metric.hint" class="container-resource-cpu-metric__hint">
                            {{ metric.hint }}
                          </span>
                        </div>
                      </div>
                      <div
                        v-else
                        class="container-resource-detail-card__body"
                        :class="{ 'container-resource-detail-card__body--memory': group.key === 'memory' }"
                      >
                        <div
                          v-for="row in group.rows"
                          :key="row.key"
                          class="container-resource-detail-row"
                          :class="{ 'container-resource-detail-row--placeholder': row.type === 'placeholder' }"
                        >
                          <span class="container-resource-detail-row__label">{{ row.label }}</span>
                          <span class="container-resource-detail-row__value">
                            <t-tag v-if="row.type === 'tag'" :theme="row.theme" variant="light-outline">
                              {{ row.value }}
                            </t-tag>
                            <span v-else>{{ row.value }}</span>
                          </span>
                        </div>
                      </div>
                    </article>
                  </div>
                </section>
              </section>
            </t-tab-panel>

            <t-tab-panel value="logs" :label="t('container.detail.tabs.logs')" :destroy-on-hide="false">
              <section class="container-detail-section container-detail-tab-body container-detail-tab-body--viewer">
                <div class="container-detail-logs-header" data-testid="container-detail-logs-header">
                  <div class="container-detail-logs-header__title-block">
                    <h3 class="container-detail-logs-header__title">{{ t('container.detail.logs.title') }}</h3>
                  </div>
                  <t-tag :theme="logsLiveStateTheme" variant="light-outline" data-testid="container-detail-logs-status">
                    {{ logsLiveStateLabel }}
                  </t-tag>
                </div>
                <log-viewer
                  v-model:line-limit="logLineLimit"
                  :entries="[...logEntries]"
                  :content-version="logContentVersion"
                  :loading="logsLoading"
                  :error="logsError"
                  :truncated="logsTruncated"
                  :clear-label="t('container.detail.logs.clear')"
                  :copy-label="t('container.detail.copy')"
                  :download-label="t('container.detail.logs.download')"
                  :retry-label="t('container.list.retry')"
                  :search-placeholder="t('container.detail.logs.searchPlaceholder')"
                  :wrap-label="t('container.detail.logs.wrap')"
                  :auto-scroll-label="t('container.detail.logs.autoScroll')"
                  :auto-scroll-tooltip-label="t('container.detail.logs.autoScrollTooltip')"
                  :pause-label="t('container.detail.logs.pause')"
                  :resume-label="t('container.detail.logs.resume')"
                  :reconnect-label="t('container.detail.logs.reconnect')"
                  :jump-bottom-label="t('container.detail.logs.jumpBottom')"
                  :level-filter-label="t('container.detail.logs.levelFilter')"
                  :all-levels-label="t('container.detail.logs.allLevels')"
                  :match-count-label="t('container.detail.logs.matchCount')"
                  :empty-label="t('container.detail.logs.empty')"
                  :truncated-label="t('container.detail.logs.truncated')"
                  :detail-title-label="t('container.detail.logs.detailTitle')"
                  :important-fields-label="t('container.detail.logs.importantFields')"
                  :basic-info-label="t('container.detail.logs.basicInfo')"
                  :time-label="t('container.detail.logs.time')"
                  :level-label="t('container.detail.logs.level')"
                  :stream-label="t('container.detail.logs.stream')"
                  :source-label="t('container.detail.logs.source')"
                  :operation-label="t('container.detail.operation')"
                  :stdout-label="t('container.list.logs.stdout')"
                  :stderr-label="t('container.list.logs.stderr')"
                  :view-detail-label="t('container.detail.logs.viewDetail')"
                  :collapse-detail-label="t('container.detail.logs.collapseDetail')"
                  :metadata-label="t('container.detail.logs.metadata')"
                  :message-label="t('container.detail.logs.message')"
                  :raw-label="t('container.detail.logs.raw')"
                  :copy-message-label="t('container.detail.logs.copyMessage')"
                  :copy-line-label="t('container.detail.logs.copyLine')"
                  :copy-json-label="t('container.detail.logs.copyJson')"
                  :copy-success-label="t('container.detail.copySuccess')"
                  :copy-error-label="t('container.detail.copyError')"
                  :paused="logsPaused"
                  :show-reconnect="logsCanReconnect"
                  :viewer-mode="true"
                  viewer-storage-key="graft.container-detail.viewer.logs.height"
                  :fullscreen-label="t('container.detail.viewer.enterFullscreen')"
                  :exit-fullscreen-label="t('container.detail.viewer.exitFullscreen')"
                  :resize-handle-label="t('container.detail.viewer.resizeHandle')"
                  @refresh="loadLogs"
                  @clear="clearLogsScreen"
                  @pause="toggleLogsPause"
                  @resume="toggleLogsPause"
                  @reconnect="reconnectLogsRealtime"
                />
              </section>
            </t-tab-panel>

            <t-tab-panel value="shell" :label="t('container.detail.tabs.shell')" :destroy-on-hide="false">
              <section
                class="container-detail-section container-detail-section--shell container-detail-tab-body container-detail-tab-body--viewer"
              >
                <container-shell-panel
                  :active="activeTab === 'shell'"
                  :container-id="containerId"
                  :container-state="safeDetail.state"
                />
              </section>
            </t-tab-panel>

            <t-tab-panel value="health" :label="t('container.detail.tabs.health')" :destroy-on-hide="false">
              <section class="container-detail-section container-detail-section--health">
                <div class="container-health-diagnostics">
                  <header class="container-health-diagnostics__header">
                    <h3>{{ t('container.detail.health.diagnosisTitle') }}</h3>
                  </header>

                  <div class="container-health-summary-grid">
                    <t-card
                      class="container-health-summary-card"
                      size="small"
                      :bordered="false"
                      data-testid="health-summary-status"
                    >
                      <span class="container-health-summary-card__label">
                        {{ t('container.detail.health.currentStatus') }}
                      </span>
                      <div class="container-health-summary-card__value">
                        <t-tag :theme="healthDiagnosis.theme" variant="light-outline">
                          {{ healthDiagnosis.label }}
                        </t-tag>
                      </div>
                      <p>{{ healthDiagnosis.description }}</p>
                    </t-card>

                    <t-card
                      class="container-health-summary-card"
                      size="small"
                      :bordered="false"
                      data-testid="health-summary-restarts"
                    >
                      <span class="container-health-summary-card__label">
                        {{ t('container.detail.health.restartCount') }}
                      </span>
                      <strong class="container-health-summary-card__value">
                        {{ restartCountLabel(safeDetail.restart_count) }}
                      </strong>
                      <p>{{ restartSummaryLabel(safeDetail.restart_count) }}</p>
                    </t-card>

                    <t-card
                      class="container-health-summary-card"
                      size="small"
                      :bordered="false"
                      data-testid="health-summary-last-check"
                    >
                      <span class="container-health-summary-card__label">
                        {{ t('container.detail.health.recentCheck') }}
                      </span>
                      <strong class="container-health-summary-card__value">
                        {{ healthcheckDetails.lastCheckedAt || formatTime(safeDetail.inspect_updated_at) }}
                      </strong>
                      <p>{{ healthcheckDetails.relativeCheckedAt || inspectUpdatedRelative }}</p>
                    </t-card>
                  </div>

                  <t-card
                    class="container-health-info-card"
                    size="small"
                    :title="t('container.detail.health.checkResult')"
                    data-testid="healthcheck-result-card"
                  >
                    <template #actions>
                      <div v-if="healthcheckDetails.configured" class="container-health-card-actions">
                        <t-tag :theme="healthcheckDetails.theme" variant="light-outline" size="small">
                          {{ healthcheckDetails.statusLabel }}
                        </t-tag>
                        <span>{{
                          t('container.detail.health.exitCodeValue', { code: healthcheckDetails.exitCode })
                        }}</span>
                      </div>
                    </template>
                    <template v-if="healthcheckDetails.configured">
                      <t-descriptions
                        class="container-health-inline-descriptions"
                        :column="2"
                        size="small"
                        table-layout="auto"
                      >
                        <t-descriptions-item :label="t('container.detail.health.healthcheck')">
                          <t-tag :theme="healthcheckDetails.theme" variant="light-outline">
                            {{ healthcheckDetails.statusLabel }}
                          </t-tag>
                        </t-descriptions-item>
                        <t-descriptions-item :label="t('container.detail.health.exitCode')">
                          {{ healthcheckDetails.exitCode }}
                        </t-descriptions-item>
                      </t-descriptions>
                      <div class="container-health-block">
                        <div class="container-health-block__label">
                          <span>{{ t('container.detail.health.checkCommand') }}</span>
                          <t-tooltip :content="t('container.detail.health.copyCommand')">
                            <t-button
                              size="small"
                              variant="text"
                              :disabled="healthcheckDetails.command === '-'"
                              @click="copyHealthcheckCommand"
                            >
                              {{ t('container.detail.copy') }}
                            </t-button>
                          </t-tooltip>
                        </div>
                        <code class="container-health-code graft-scrollbar">{{ healthcheckDetails.command }}</code>
                      </div>
                      <div class="container-health-block">
                        <span class="container-health-block__label">{{ t('container.detail.health.lastOutput') }}</span>
                        <pre
                          class="container-health-output graft-scrollbar"
                          :class="{ 'container-health-output--error': healthcheckDetails.hasFailure }"
                          >{{ healthcheckDetails.output }}</pre
                        >
                      </div>
                      <p class="container-health-last-check">
                        {{
                          t('container.detail.health.lastCheckValue', { time: healthcheckDetails.lastCheckedAt || '-' })
                        }}
                      </p>
                    </template>
                    <div v-else class="container-health-empty">
                      <t-alert theme="info">
                        {{ t('container.detail.health.healthcheckUnavailableAlert') }}
                      </t-alert>
                      <t-empty size="small" :description="t('container.detail.health.healthcheckUnavailableEmpty')" />
                    </div>
                  </t-card>

                  <t-card
                    class="container-health-info-card"
                    size="small"
                    :title="t('container.detail.health.stability')"
                    data-testid="runtime-stability-card"
                  >
                    <template #actions>
                      <t-tag :theme="runtimeStability.theme" variant="light-outline" size="small">
                        {{ runtimeStability.label }}
                      </t-tag>
                    </template>
                    <t-descriptions
                      class="container-health-stability-grid"
                      :column="2"
                      size="small"
                      table-layout="auto"
                    >
                      <t-descriptions-item :label="t('container.list.fields.startedAt')">
                        {{ formatTime(safeDetail.started_at) }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('container.detail.health.uptime')">
                        {{ uptimeLabel(safeDetail.started_at) }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('container.list.fields.restartPolicy')">
                        {{ safeDetail.restart_policy || '-' }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('container.detail.health.restartCount')">
                        {{ safeDetail.restart_count ?? '-' }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('container.detail.health.lastExitCode')">
                        {{ runtimeStability.lastExitCode }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('container.detail.health.oomKilled')">
                        {{ runtimeStability.oomKilled }}
                      </t-descriptions-item>
                    </t-descriptions>
                  </t-card>
                </div>
              </section>
            </t-tab-panel>

            <t-tab-panel value="config" :label="t('container.detail.tabs.config')" :destroy-on-hide="false">
              <section
                class="container-detail-section container-detail-section--config container-detail-tab-body container-detail-tab-body--long"
              >
                <div class="container-config-section">
                  <h3>{{ t('container.detail.config.runtimeTitle') }}</h3>
                  <t-card class="container-runtime-config-card" size="small" :bordered="false">
                    <div class="container-runtime-config-list">
                      <div v-for="item in runtimeConfigItems" :key="item.key" class="container-runtime-config-row">
                        <span class="container-runtime-config-row__label">{{ item.label }}</span>
                        <t-tooltip :content="item.value">
                          <code class="container-runtime-config-row__value">{{ item.value }}</code>
                        </t-tooltip>
                        <t-tooltip :content="t('container.detail.config.copyRuntimeValue')">
                          <t-button
                            class="container-runtime-config-row__copy"
                            shape="square"
                            size="small"
                            theme="default"
                            variant="text"
                            :disabled="!item.copyable"
                            @click="copyRuntimeConfigValue(item)"
                          >
                            <template #icon>
                              <t-icon name="copy" />
                            </template>
                          </t-button>
                        </t-tooltip>
                      </div>
                    </div>
                  </t-card>
                </div>

                <div class="container-config-section">
                  <div class="container-config-section__header">
                    <h3>{{ t('container.detail.config.environment') }}</h3>
                    <span>{{ t('container.detail.config.environmentCount', { count: environmentRows.length }) }}</span>
                  </div>
                  <t-alert
                    class="container-config-section__policy-alert"
                    :theme="environmentPolicyNoticeTheme"
                    :message="environmentPolicyNotice"
                  />
                  <div class="container-env-toolbar">
                    <t-input
                      v-model="environmentKeyword"
                      class="container-env-toolbar__search"
                      clearable
                      :placeholder="t('container.detail.config.searchPlaceholder')"
                    >
                      <template #prefix-icon>
                        <t-icon name="search" />
                      </template>
                    </t-input>
                    <t-select
                      v-model="environmentPolicyFilter"
                      class="container-env-toolbar__policy"
                      :options="environmentPolicyFilterOptions"
                    />
                    <t-tooltip :content="environmentCopyTooltip">
                      <span>
                        <t-button
                          theme="default"
                          variant="outline"
                          :disabled="environmentCopyDisabled"
                          @click="copyFilteredEnvironmentAsEnv"
                        >
                          <template #icon>
                            <t-icon name="file-copy" />
                          </template>
                          {{ t('container.detail.config.copyEnvFile') }}
                        </t-button>
                      </span>
                    </t-tooltip>
                    <t-button
                      theme="default"
                      variant="outline"
                      :loading="detailRefreshing"
                      @click="refreshContainerDetail"
                    >
                      <template #icon>
                        <t-icon name="refresh" />
                      </template>
                      {{ t('container.detail.refresh') }}
                    </t-button>
                  </div>
                  <t-table
                    v-if="environmentRows.length"
                    class="container-env-table"
                    row-key="name"
                    size="small"
                    :columns="environmentColumns"
                    :data="filteredEnvironmentRows"
                    :pagination="undefined"
                    table-layout="fixed"
                    cell-empty-content="-"
                  >
                    <template #empty>
                      <t-empty
                        size="small"
                        :title="environmentEmptyState.title"
                        :description="environmentEmptyState.description"
                      />
                    </template>
                    <template #name="{ row }">
                      <t-tooltip :content="row.name">
                        <code class="container-env-cell container-env-cell--name">{{ row.name }}</code>
                      </t-tooltip>
                    </template>
                    <template #value="{ row }">
                      <t-tag v-if="row.policy === 'hidden'" theme="danger" variant="light-outline" size="small">
                        {{ row.value }}
                      </t-tag>
                      <t-tag
                        v-else-if="row.policy !== 'masked' && isBooleanEnvironmentValue(row.value)"
                        theme="primary"
                        variant="light-outline"
                        size="small"
                      >
                        {{ row.value }}
                      </t-tag>
                      <t-tooltip v-else :content="row.rawValue || row.value">
                        <code class="container-env-cell container-env-cell--value">{{ row.value || '-' }}</code>
                      </t-tooltip>
                    </template>
                    <template #policy="{ row }">
                      <t-tag :theme="policyTheme(row.policy)" variant="light-outline" size="small">
                        {{ policyLabel(row.policy) }}
                      </t-tag>
                    </template>
                    <template #operation="{ row }">
                      <t-tooltip v-if="row.copyable" :content="row.copyTooltip">
                        <span>
                          <t-button
                            class="container-env-copy-button"
                            data-testid="env-copy"
                            shape="square"
                            size="small"
                            theme="default"
                            variant="text"
                            :disabled="row.copyDisabled"
                            @click="copyEnvironmentValue(row)"
                          >
                            <template #icon>
                              <t-icon name="copy" />
                            </template>
                          </t-button>
                        </span>
                      </t-tooltip>
                    </template>
                  </t-table>
                  <div
                    v-else
                    class="container-detail-empty-state container-detail-empty-state--compact container-detail-empty-state--inline"
                  >
                    <t-empty
                      size="small"
                      :title="t('container.detail.config.environmentEmptyTitle')"
                      :description="t('container.detail.config.environmentUnavailable')"
                    />
                  </div>
                </div>
              </section>
            </t-tab-panel>

            <t-tab-panel value="network" :label="t('container.detail.tabs.network')" :destroy-on-hide="false">
              <section
                class="container-detail-section container-detail-section--network container-detail-tab-body container-detail-tab-body--long"
              >
                <section class="container-network-panel">
                  <header class="container-network-panel__header">
                    <h3>{{ t('container.detail.network.connections') }}</h3>
                  </header>
                  <template v-if="networkConnections.length">
                    <article v-if="networkConnections.length === 1" class="container-network-connection-card">
                      <header class="container-network-connection-card__header">
                        <copyable-detail-value
                          class="container-network-connection-card__name"
                          :copy-label="t('container.detail.copy')"
                          :value="networkConnections[0].name"
                          :display-value="displayOptionalValue(networkConnections[0].name)"
                          code
                          data-testid="network-name-copy-0"
                          @copy="copyDetailText"
                        />
                      </header>
                      <div class="container-network-field-grid">
                        <div
                          v-for="field in singleNetworkFields"
                          :key="field.key"
                          class="container-network-field"
                          :class="{ 'container-network-field--technical': field.technical }"
                        >
                          <span>{{ field.label }}</span>
                          <copyable-detail-value
                            v-if="field.copyable"
                            :copy-label="t('container.detail.copy')"
                            :value="field.copyValue"
                            :display-value="field.displayValue"
                            code
                            :data-testid="field.testId"
                            @copy="copyDetailText"
                          />
                          <strong v-else>{{ field.displayValue }}</strong>
                        </div>
                      </div>
                    </article>
                    <t-table
                      v-else
                      row-key="name"
                      size="small"
                      :columns="networkColumns"
                      :data="networkConnections"
                      :pagination="undefined"
                      table-layout="fixed"
                      :cell-empty-content="t('container.detail.network.noData')"
                    >
                      <template #name="{ row, rowIndex }">
                        <copyable-detail-value
                          :copy-label="t('container.detail.copy')"
                          :value="row.name"
                          :display-value="displayOptionalValue(row.name)"
                          code
                          :data-testid="`network-name-copy-${rowIndex}`"
                          @copy="copyDetailText"
                        />
                      </template>
                      <template #ip_address="{ row, rowIndex }">
                        <copyable-detail-value
                          :copy-label="t('container.detail.copy')"
                          :value="row.ip_address || ''"
                          :display-value="displayOptionalValue(row.ip_address)"
                          code
                          :data-testid="`network-ip-copy-${rowIndex}`"
                          @copy="copyDetailText"
                        />
                      </template>
                      <template #gateway="{ row }">
                        {{ displayOptionalValue(row.gateway) }}
                      </template>
                      <template #mac_address="{ row, rowIndex }">
                        <copyable-detail-value
                          :copy-label="t('container.detail.copy')"
                          :value="row.mac_address || ''"
                          :display-value="displayOptionalValue(row.mac_address)"
                          code
                          :data-testid="`network-mac-copy-${rowIndex}`"
                          @copy="copyDetailText"
                        />
                      </template>
                      <template #network_id="{ row, rowIndex }">
                        <copyable-detail-value
                          :copy-label="t('container.detail.copy')"
                          :value="row.network_id || ''"
                          :display-value="displayTechnicalIdentifier(row.network_id)"
                          code
                          :data-testid="`network-id-copy-${rowIndex}`"
                          @copy="copyDetailText"
                        />
                      </template>
                      <template #endpoint_id="{ row, rowIndex }">
                        <copyable-detail-value
                          :copy-label="t('container.detail.copy')"
                          :value="row.endpoint_id || ''"
                          :display-value="displayTechnicalIdentifier(row.endpoint_id)"
                          code
                          :data-testid="`network-endpoint-copy-${rowIndex}`"
                          @copy="copyDetailText"
                        />
                      </template>
                    </t-table>
                  </template>
                  <div v-else class="container-detail-empty-state container-detail-empty-state--inline">
                    <t-empty size="small" :description="t('container.list.detail.networkEmpty')" />
                  </div>
                </section>

                <section class="container-network-panel">
                  <header class="container-network-panel__header">
                    <h3>{{ t('container.detail.network.ports') }}</h3>
                  </header>
                  <article
                    v-if="portMappingRows.length === 1"
                    class="container-port-mapping-card"
                    :class="{ 'container-port-mapping-card--internal': !portMappingRows[0].hasHostBinding }"
                  >
                    <div class="container-port-mapping-card__main">
                      <span class="container-port-mapping-card__label">
                        {{ t('container.detail.network.mapping') }}
                      </span>
                      <copyable-detail-value
                        :copy-label="t('container.detail.copy')"
                        :value="portMappingRows[0].copyValue"
                        :display-value="portMappingRows[0].mapping"
                        code
                        data-testid="port-mapping-copy-0"
                        @copy="copyDetailText"
                      />
                    </div>
                    <div v-if="portMappingRows[0].hasHostBinding" class="container-port-mapping-card__listen">
                      <span>{{ t('container.detail.network.listenAddress') }}</span>
                      <code class="container-network-code">{{ portMappingRows[0].listenAddress }}</code>
                    </div>
                    <span class="container-port-mapping-card__description">
                      {{ portMappingRows[0].description }}
                    </span>
                  </article>
                  <t-table
                    v-else-if="portMappingRows.length > 1"
                    row-key="key"
                    size="small"
                    :columns="portColumns"
                    :data="portMappingRows"
                    :pagination="undefined"
                    table-layout="fixed"
                    :cell-empty-content="t('container.detail.network.noData')"
                  >
                    <template #privatePort="{ row }">
                      <code class="container-network-code">{{ row.privatePort }}</code>
                    </template>
                    <template #publicPort="{ row }">
                      <code class="container-network-code">{{ row.publicPort }}</code>
                    </template>
                    <template #protocol="{ row }">
                      <code class="container-network-code">{{ row.protocol }}</code>
                    </template>
                    <template #listenAddress="{ row }">
                      <code class="container-network-code">{{ row.listenAddress }}</code>
                    </template>
                    <template #mapping="{ row, rowIndex }">
                      <copyable-detail-value
                        :copy-label="t('container.detail.copy')"
                        :value="row.copyValue"
                        :display-value="row.mapping"
                        code
                        :data-testid="`port-mapping-copy-${rowIndex}`"
                        @copy="copyDetailText"
                      />
                    </template>
                  </t-table>
                  <div v-else class="container-detail-empty-state container-detail-empty-state--inline">
                    <t-empty size="small" :description="t('container.list.detail.portEmpty')" />
                  </div>
                </section>

                <section class="container-network-panel">
                  <header class="container-network-panel__header">
                    <h3>{{ t('container.detail.network.aliasDns') }}</h3>
                  </header>
                  <div
                    v-if="networkMetadataFields.length"
                    class="container-network-field-grid container-network-field-grid--metadata"
                  >
                    <div
                      v-for="field in networkMetadataFields"
                      :key="field.key"
                      class="container-network-field"
                      :class="{ 'container-network-field--technical': field.technical }"
                    >
                      <span>{{ field.label }}</span>
                      <strong>{{ field.displayValue }}</strong>
                    </div>
                  </div>
                  <t-empty
                    v-if="!hasAdditionalNetworkMetadata"
                    class="container-network-metadata-empty"
                    size="small"
                    :description="t('container.detail.network.aliasDnsEmpty')"
                  />
                </section>
              </section>
            </t-tab-panel>

            <t-tab-panel value="storage" :label="t('container.detail.tabs.storage')" :destroy-on-hide="false">
              <section
                class="container-detail-section container-detail-section--storage container-detail-tab-body container-detail-tab-body--long"
              >
                <div v-if="mountCards.length" class="container-mount-card-grid">
                  <article
                    v-for="mount in mountCards"
                    :key="mount.key"
                    class="container-mount-card"
                    :data-testid="`mount-card-${mount.index}`"
                  >
                    <header class="container-mount-card__header">
                      <copyable-detail-value
                        class="container-mount-card__destination"
                        :copy-label="t('container.detail.copy')"
                        :value="mount.destination"
                        :display-value="mount.destinationDisplay"
                        code
                        :data-testid="`mount-destination-copy-${mount.index}`"
                        @copy="copyDetailText"
                      />
                      <div class="container-mount-card__actions">
                        <t-tag :theme="mount.typeTheme" variant="light-outline" size="small">
                          {{ mount.typeLabel }}
                        </t-tag>
                        <t-tag :theme="mount.accessTheme" variant="light-outline" size="small">
                          {{ mount.accessLabel }}
                        </t-tag>
                      </div>
                    </header>

                    <div class="container-mount-card__body">
                      <section class="container-mount-info">
                        <h3>{{ t('container.detail.storage.basicInfo') }}</h3>
                        <div class="container-mount-field">
                          <span>{{ t('container.detail.storage.source') }}</span>
                          <copyable-detail-value
                            v-if="mount.source"
                            :copy-label="t('container.detail.copy')"
                            :value="mount.source"
                            :display-value="mount.sourceDisplay"
                            code
                            :data-testid="`mount-source-copy-${mount.index}`"
                            @copy="copyDetailText"
                          />
                          <strong v-else>{{ t('container.detail.storage.sourceUnavailable') }}</strong>
                        </div>
                        <div class="container-mount-field">
                          <span>{{ t('container.detail.storage.destination') }}</span>
                          <copyable-detail-value
                            :copy-label="t('container.detail.copy')"
                            :value="mount.destination"
                            :display-value="mount.destinationDisplay"
                            code
                            :data-testid="`mount-destination-field-copy-${mount.index}`"
                            @copy="copyDetailText"
                          />
                        </div>
                        <div class="container-mount-field">
                          <span>{{ t('container.detail.storage.mode') }}</span>
                          <code>{{ mount.modeLabel }}</code>
                        </div>
                      </section>
                      <div class="container-mount-usage" :class="`container-mount-usage--${mount.usageTone}`">
                        <div class="container-mount-usage__header">
                          <span>{{ t('container.detail.storage.usage') }}</span>
                          <t-button
                            v-if="!mount.usageRefreshDisabled"
                            size="small"
                            theme="default"
                            variant="outline"
                            :loading="mount.usageRefreshing"
                            :disabled="mount.usageRefreshing"
                            :data-testid="`mount-refresh-${mount.index}`"
                            @click="refreshMountUsage(mount)"
                          >
                            {{ mount.usageRefreshLabel }}
                          </t-button>
                          <t-tooltip v-else :content="mount.usageRefreshTooltip">
                            <t-button
                              size="small"
                              theme="default"
                              variant="outline"
                              disabled
                              :data-testid="`mount-refresh-unsupported-${mount.index}`"
                            >
                              {{ mount.usageRefreshLabel }}
                            </t-button>
                          </t-tooltip>
                        </div>
                        <strong>{{ mount.usageSize }}</strong>
                        <span v-if="mount.measuredAt" class="container-mount-usage__time">
                          {{ t('container.detail.storage.measuredAt', { time: mount.measuredAt }) }}
                        </span>
                        <span v-if="mount.message" class="container-mount-usage__message">
                          {{ mount.message }}
                        </span>
                        <span v-if="mount.sharedHint" class="container-mount-usage__message">
                          {{ mount.sharedHint }}
                        </span>
                      </div>
                    </div>
                  </article>
                </div>
                <div v-else class="container-detail-empty-state" role="status" aria-live="polite">
                  <t-empty
                    size="small"
                    :title="t('container.detail.storage.emptyTitle')"
                    :description="t('container.detail.storage.emptyDescription')"
                  />
                </div>
              </section>
            </t-tab-panel>

            <t-tab-panel value="raw" :label="t('container.detail.tabs.raw')" :destroy-on-hide="false">
              <section
                class="container-detail-section container-detail-section--raw container-detail-tab-body container-detail-tab-body--viewer"
              >
                <container-raw-json-panel
                  :copy-value="rawJsonCopyValue"
                  :value="rawJsonDisplayValue"
                  :title="t('container.detail.raw.title')"
                  :description="t('container.detail.raw.description')"
                  :policy-message="rawJsonPolicyMessage"
                  :policy-alert-theme="rawJsonPolicyTheme"
                  :search-placeholder="t('container.detail.raw.searchPlaceholder')"
                  :root-label="t('container.detail.raw.root')"
                  :source-label="t('container.detail.raw.source')"
                  :tree-label="t('container.detail.raw.tree')"
                  :copy-label="t('container.detail.copy')"
                  :copy-tooltip="rawJsonCopyTooltip"
                  :copy-masked-tooltip="rawJsonCopyTooltip"
                  :copy-disabled-message="t('container.detail.raw.copyDisabledMessage')"
                  :raw-copy-enabled="rawJsonCopyEnabled"
                  :copy-success-label="t('container.detail.copySuccess')"
                  :copy-error-label="t('container.detail.copyError')"
                  :fullscreen-label="t('container.detail.viewer.enterFullscreen')"
                  :exit-fullscreen-label="t('container.detail.viewer.exitFullscreen')"
                  :resize-handle-label="t('container.detail.viewer.resizeHandle')"
                  :expand-all-label="t('container.detail.raw.expandAll')"
                  :collapse-all-label="t('container.detail.raw.collapseAll')"
                  :format-label="t('container.detail.raw.format')"
                  :field-count-label="t('container.detail.raw.fieldCount')"
                  :sensitive-field-label="t('container.detail.raw.sensitiveFieldCount')"
                  :masked-count-label="t('container.detail.raw.maskedCount')"
                  :environment-label="t('container.detail.raw.environmentCount')"
                  :port-label="t('container.detail.raw.portCount')"
                  :mounted-label="t('container.detail.raw.mountCount')"
                  :network-label="t('container.detail.raw.networkCount')"
                  :updated-at-label="t('container.detail.raw.updatedAt')"
                  :collapse-tree-node-label="t('container.detail.raw.collapseNode')"
                  :expand-tree-node-label="t('container.detail.raw.expandNode')"
                  :search-empty-label="t('container.detail.raw.noMatches')"
                  :sensitive-label="t('container.detail.raw.sensitive')"
                  :empty-label="t('container.detail.raw.empty')"
                  :error-label="t('container.detail.raw.error')"
                />
              </section>
            </t-tab-panel>
          </t-tabs>
        </t-card>
      </template>

      <t-empty v-else class="container-detail-state" size="small" :description="t('container.detail.empty')" />
    </section>
  </div>
</template>
<script setup lang="ts">
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onActivated, onDeactivated, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { LOCALE, type LocalizedTitle } from '@/contracts/i18n/locales';
import { buildAuditResourceLocation } from '@/modules/audit/contract/deep-link';
import { AUDIT_PERMISSION_CODE } from '@/modules/audit/contract/permissions';
import { ManagementPageHeader } from '@/shared/components/management';
import { MetricCard } from '@/shared/components/metrics';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import {
  copyText as copyTextToClipboard,
  formatBytes,
  formatLocaleDateTime,
  formatNanosecondsAsDuration,
  formatPercent,
  LogViewer,
  normalizeStructuredLogEntry,
  toProgressPercent,
} from '@/shared/observability';
import { openRealtimeTopicSocket, type RealtimeTopicSocketController } from '@/shared/realtime';
import { usePermissionStore, useTabsRouterStore } from '@/store';
import { createLogger } from '@/utils/logger';
import { localizeRouteTitleKey } from '@/utils/route/title';

import {
  getContainer,
  getContainerLogs,
  getContainerMountUsage,
  postContainerMountUsageRefresh,
} from '../../api/container';
import ContainerRawJsonPanel from '../../components/ContainerRawJsonPanel.vue';
import ContainerShellPanel from '../../components/ContainerShellPanel.vue';
import {
  buildContainerLogsTopicName,
  isContainerLogsTopicForContainer,
  parseContainerLogsTopicContainerId,
} from '../../contract/realtime';
import {
  acquireContainerStatsSubscription,
  clearContainerDetail,
  releaseContainerStatsSubscription,
  seedContainerDetail,
  selectContainerDetailView,
  selectContainerStatsChangeState,
  selectContainerStatsHistory,
  selectContainerStatsRealtimeState,
} from '../../shared/stats-manager';
import { metricChangedClass, metricProgressStatus } from '../../shared/stats-visual-state';
import type {
  ContainerActionLevel,
  ContainerDetail,
  ContainerDetailRecord,
  ContainerHealth,
  ContainerHealthcheck,
  ContainerLogEntry,
  ContainerMount,
  ContainerMountUsage,
  ContainerOrchestratorType,
  ContainerState,
} from '../../types/container';
import ContainerOverviewPanel from './components/ContainerOverviewPanel.vue';
import CopyableDetailValue from './components/CopyableDetailValue.vue';
import type { ContainerOverviewInfoSection } from './components/overview';
import { buildCpuDetailMetrics, formatCpuCountText } from './components/resource-cpu-presenter';
import { ContainerLogRealtimeBatcher } from './log-realtime-batcher';
import { createContainerDetailLogViewStore } from './log-view-store';

defineOptions({
  name: 'ContainerDetailIndex',
});

type DetailTab = 'overview' | 'resources' | 'logs' | 'shell' | 'health' | 'config' | 'network' | 'storage' | 'raw';
type EnvironmentPolicy = 'plain' | 'masked' | 'hidden' | 'unknown';
type EnvironmentPolicyFilter = EnvironmentPolicy | 'all' | 'sensitive';
type EnvironmentRow = {
  copyValue: string;
  copyable: boolean;
  copyDisabled: boolean;
  copyTooltip: string;
  displayValue: string;
  hasSensitiveValue: boolean;
  name: string;
  policy: EnvironmentPolicy;
  rawJsonCopyValue: string;
  rawJsonValue: string;
  rawValue: string;
  value: string;
};
type RuntimeConfigItem = {
  copyable: boolean;
  key: 'command' | 'entrypoint' | 'workingDir';
  label: string;
  rawValue: string;
  value: string;
};
type NetworkField = {
  copyValue: string;
  copyable: boolean;
  displayValue: string;
  key: string;
  label: string;
  testId?: string;
  technical?: boolean;
};
type PortMappingRow = {
  copyValue: string;
  description: string;
  hasHostBinding: boolean;
  key: string;
  listenAddresses: string[];
  listenAddress: string;
  mapping: string;
  privatePort: string;
  protocol: string;
  publicPort: string;
};
type MountCardUsageTone = 'success' | 'weak' | 'warning';
type MountTagTheme = 'success' | 'primary' | 'warning' | 'danger' | 'default';
type MountCard = {
  accessLabel: string;
  accessTheme: MountTagTheme;
  destination: string;
  destinationDisplay: string;
  index: number;
  key: string;
  mountId: string;
  measuredAt: string;
  message: string;
  mode: string;
  modeLabel: string;
  raw: ContainerMount;
  sharedHint: string;
  source: string;
  sourceDisplay: string;
  typeLabel: string;
  typeTheme: MountTagTheme;
  usageLabel: string;
  usageRefreshDisabled: boolean;
  usageRefreshLabel: string;
  usageRefreshTooltip: string;
  usageRefreshing: boolean;
  usageSize: string;
  usageTheme: MountTagTheme;
  usageTone: MountCardUsageTone;
};
type SafeContainerDetail = ContainerDetailRecord & {
  command: string[];
  entrypoint: string[];
  environment: NonNullable<ContainerDetail['environment']>;
  mounts: ContainerMount[];
  names: NonNullable<ContainerDetail['names']>;
  networks: NonNullable<ContainerDetail['networks']>;
  ports: NonNullable<ContainerDetail['ports']>;
};
type ResourceStatusTheme = 'success' | 'warning' | 'default';
type HealthStatusTheme = 'success' | 'warning' | 'danger' | 'default';
type ResourceMetricFormat = 'bytes' | 'number' | 'percent' | 'text';
type ContainerResourceSummary = NonNullable<ContainerDetail['resource']>;
type ContainerStatsHistoryPoint = {
  collectedAt: string;
  cpuPercent: string;
  key: string;
  memoryPercent: string;
};
type ResourceMetricKey = Extract<
  keyof ContainerResourceSummary,
  | 'cpu_percent'
  | 'cpu_usage_in_kernelmode'
  | 'cpu_usage_in_usermode'
  | 'memory_active_file'
  | 'memory_cache'
  | 'memory_inactive_file'
  | 'memory_limit_bytes'
  | 'memory_percent'
  | 'memory_pgfault'
  | 'memory_pgmajfault'
  | 'memory_rss'
  | 'memory_usage_bytes'
  | 'online_cpus'
  | 'pids_current'
  | 'pids_limit'
  | 'rx_bytes'
  | 'rx_dropped'
  | 'rx_errors'
  | 'rx_packets'
  | 'system_cpu_usage'
  | 'throttling_periods'
  | 'throttling_throttled_periods'
  | 'throttling_throttled_time'
  | 'total_cpu_usage'
  | 'tx_bytes'
  | 'tx_dropped'
  | 'tx_errors'
  | 'tx_packets'
>;
type ResourceDetailRow =
  | {
      key: string;
      label: string;
      type: 'text';
      value: string;
    }
  | {
      key: string;
      label: string;
      type: 'placeholder';
      value: string;
    }
  | {
      key: string;
      label: string;
      theme: ResourceStatusTheme;
      type: 'tag';
      value: string;
    };
type ResourceDetailGroup = {
  key: string;
  rows: ResourceDetailRow[];
  title: string;
};
type ResourceMetricDefinition = [ResourceMetricKey, string, ResourceMetricFormat];
const DETAIL_TABS: DetailTab[] = [
  'overview',
  'resources',
  'logs',
  'shell',
  'health',
  'config',
  'network',
  'storage',
  'raw',
];
const DEFAULT_LOG_QUERY = {
  tail: 200,
  since: undefined,
  timestamps: true,
  stdout: true,
  stderr: true,
};

const { locale, t } = useI18n();
const route = useRoute();
const router = useRouter();
const permissionStore = usePermissionStore();
const tabsRouterStore = useTabsRouterStore();
const logger = createLogger('container.detail');
const auditPermissionCodes = AUDIT_PERMISSION_CODE;

const detail = ref<ContainerDetailRecord | null>(null);
const detailRefreshing = ref(false);
const error = ref('');
const logLineLimit = ref(DEFAULT_LOG_QUERY.tail);
const activeTab = ref<DetailTab>(normalizeTab(route.query.tab));
const environmentKeyword = ref('');
const environmentPolicyFilter = ref<EnvironmentPolicyFilter>('all');
const refreshingMountKeys = ref<Set<string>>(new Set());
const realtimeEnabled = ref(true);
const activeRealtimeSubscriptionId = ref('');
const detailPageActive = ref(false);
const logViewStore = createContainerDetailLogViewStore();
const logEntries = computed(() =>
  logViewStore.entries.value
    .map((entry) => normalizeStructuredLogEntry(entry))
    .filter((entry): entry is NonNullable<ReturnType<typeof normalizeStructuredLogEntry>> => entry !== null),
);
const logContentVersion = computed(() => logViewStore.version.value);
const logsLoading = computed(() => logViewStore.state.value.loading);
const logsError = computed(() => logViewStore.state.value.error);
const logsTruncated = computed(() => logViewStore.truncated.value);
const logsPaused = computed(() => logViewStore.paused.value);
const logsSocketState = ref<'idle' | 'connecting' | 'open' | 'closed' | 'error'>('idle');
let detailRefreshSeq = 0;
let logsRefreshSeq = 0;
let logsRealtimeController: RealtimeTopicSocketController | null = null;
let logsRealtimeTopic = '';
let logsRealtimeSeeded = false;
let logsInitialLoadAttempted = false;
let logsRealtimeBatcher = new ContainerLogRealtimeBatcher({
  lineLimit: DEFAULT_LOG_QUERY.tail,
  onCommit: (snapshot) => {
    logViewStore.commit(snapshot);
  },
});
const logsLiveState = computed<'paused' | 'live' | 'reconnecting' | 'disconnected'>(() => {
  if (logsPaused.value) {
    return 'paused';
  }
  if (logsSocketState.value === 'open') {
    return 'live';
  }
  if (logsSocketState.value === 'connecting' || logsSocketState.value === 'closed') {
    return 'reconnecting';
  }
  return 'disconnected';
});
const logsLiveStateTheme = computed(() => {
  if (logsLiveState.value === 'paused') {
    return 'warning';
  }
  if (logsLiveState.value === 'live') {
    return 'success';
  }
  if (logsLiveState.value === 'reconnecting') {
    return 'primary';
  }
  return 'danger';
});
const logsLiveStateLabel = computed(() => {
  if (logsLiveState.value === 'paused') {
    return t('container.detail.logs.states.paused');
  }
  if (logsLiveState.value === 'live') {
    return t('container.detail.logs.states.live');
  }
  if (logsLiveState.value === 'reconnecting') {
    return t('container.detail.logs.states.reconnecting');
  }
  return t('container.detail.logs.states.disconnected');
});
const logsCanReconnect = computed(
  () => logsLiveState.value === 'reconnecting' || logsLiveState.value === 'disconnected',
);

const containerId = computed(() => String(route.params.id ?? '').trim());
const shortContainerIdFallback = computed(() => shortIdentifier(containerId.value, undefined, 12));
const currentDetailStatsKey = computed(() => detail.value?.id || containerId.value);
const safeDetail = computed(() =>
  normalizeDetail(materializeDetailWithManagedStats(detail.value, currentDetailStatsKey.value)),
);
const activeTabRoute = computed(() =>
  tabsRouterStore.tabRouterList.find(
    (tab) => tab.tabKey === route.path || tab.path === route.path || tab.fullPath === route.fullPath,
  ),
);
const fallbackDisplayName = computed(() => {
  const tabTitle = readNameFromTabTitle(activeTabRoute.value?.title);
  if (tabTitle) {
    return tabTitle;
  }

  const queryName = readQueryString(route.query.name);
  if (queryName) {
    return queryName;
  }

  if (containerId.value) {
    return shortContainerIdFallback.value;
  }

  return '';
});
const fallbackTitle = computed(() => buildDetailTitle(fallbackDisplayName.value));
const pageTitle = computed(() => {
  if (safeDetail.value) {
    return displayName(safeDetail.value);
  }
  return fallbackDisplayName.value || t('container.detail.title');
});
const environmentRows = computed(() => normalizeEnvironmentRows(safeDetail.value));
const environmentHasSensitiveRows = computed(() => environmentRows.value.some((row) => row.hasSensitiveValue));
const rawJsonCopyValue = computed(() => buildRawJsonCopyValue(safeDetail.value));
const rawJsonCopyEnabled = computed(() => Boolean(rawJsonCopyValue.value));
const rawJsonCopyTooltip = computed(() =>
  rawJsonCopyEnabled.value
    ? t('container.detail.raw.copyRealValueTooltip')
    : t('container.detail.raw.copyDisabledTooltip'),
);
const rawJsonPolicyTheme = computed(() => {
  if (!environmentHasSensitiveRows.value) {
    return 'info';
  }
  return rawJsonCopyEnabled.value ? 'warning' : 'error';
});
const rawJsonPolicyMessage = computed(() => {
  const policy = readEnvironmentPolicy(safeDetail.value);
  if (!environmentHasSensitiveRows.value) {
    return t('container.detail.raw.policy.noSensitive');
  }
  if (rawJsonCopyEnabled.value) {
    return t('container.detail.raw.policy.maskedCopyEnabled', { strategy: policyLabel(policy) });
  }
  return t('container.detail.raw.policy.maskedCopyDisabled', { strategy: policyLabel(policy) });
});
const environmentCopyDisabled = computed(() => {
  if (!filteredEnvironmentRows.value.length) {
    return true;
  }
  return filteredEnvironmentRows.value.some((row) => row.copyDisabled);
});
const environmentCopyTooltip = computed(() =>
  environmentCopyDisabled.value
    ? t('container.detail.config.copyPolicyDisabled')
    : t('container.detail.config.copyRealValueTooltip'),
);
const environmentPolicyNoticeTheme = computed(() => {
  if (!environmentHasSensitiveRows.value) {
    return 'info';
  }
  return environmentCopyDisabled.value ? 'warning' : 'info';
});
const environmentPolicyNotice = computed(() => {
  const policy = readEnvironmentPolicy(safeDetail.value);
  if (!environmentHasSensitiveRows.value) {
    return t('container.detail.config.policyNoticeNoSensitive', { strategy: policyLabel(policy) });
  }
  if (environmentCopyDisabled.value) {
    return t('container.detail.config.policyNoticeCopyDisabled', { strategy: policyLabel(policy) });
  }
  return t('container.detail.config.policyNoticeMaskedCopy', { strategy: policyLabel(policy) });
});
const rawJsonDisplayValue = computed(() => buildRawJsonDisplayValue(safeDetail.value));
const networkConnections = computed(() => safeDetail.value?.networks ?? []);
const singleNetworkFields = computed<NetworkField[]>(() => {
  const network = networkConnections.value[0];
  if (!network) {
    return [];
  }

  return [
    createNetworkField(
      'ip-address',
      t('container.detail.network.ipAddress'),
      network.ip_address,
      true,
      'network-ip-copy-0',
    ),
    createNetworkField('gateway', t('container.detail.network.gateway'), network.gateway),
    createNetworkField(
      'mac-address',
      t('container.detail.network.macAddress'),
      network.mac_address,
      true,
      'network-mac-copy-0',
    ),
    createNetworkField('subnet', t('container.detail.network.subnet')),
    createNetworkField(
      'network-id',
      t('container.detail.network.networkId'),
      network.network_id,
      true,
      'network-id-copy-0',
      true,
    ),
    createNetworkField(
      'endpoint-id',
      t('container.detail.network.endpointId'),
      network.endpoint_id,
      true,
      'network-endpoint-copy-0',
      true,
    ),
  ];
});
const portMappingRows = computed<PortMappingRow[]>(() => buildPortMappingRows(safeDetail.value?.ports ?? []));
const mountCards = computed<MountCard[]>(() => buildMountCards(safeDetail.value?.mounts ?? []));
const networkMetadataFields = computed<NetworkField[]>(() => {
  const current = safeDetail.value;
  if (!current) {
    return [];
  }

  const hostname = readContainerHostname(current);
  const aliases = readStringListFromRecord(current, ['aliases', 'network_aliases']);
  const dns = readStringListFromRecord(current, ['dns', 'dns_servers', 'dns_nameservers']);
  const fields = [
    createNetworkField('hostname', t('container.detail.network.hostname'), hostname),
    createNetworkField('aliases', t('container.detail.network.aliases'), aliases.join(', ')),
    createNetworkField('dns', t('container.detail.network.dns'), dns.join(', ')),
  ];

  return fields.filter((field) => field.copyValue);
});
const hasAdditionalNetworkMetadata = computed(() =>
  networkMetadataFields.value.some((field) => field.key === 'aliases' || field.key === 'dns'),
);
const runtimeConfigItems = computed<RuntimeConfigItem[]>(() => {
  const current = safeDetail.value;
  return [
    {
      copyable: Boolean(joinList(current?.command).trim() && joinList(current?.command) !== '-'),
      key: 'command',
      label: t('container.list.detail.command'),
      rawValue: joinList(current?.command),
      value: joinList(current?.command),
    },
    {
      copyable: Boolean(joinList(current?.entrypoint).trim() && joinList(current?.entrypoint) !== '-'),
      key: 'entrypoint',
      label: t('container.list.detail.entrypoint'),
      rawValue: joinList(current?.entrypoint),
      value: joinList(current?.entrypoint),
    },
    {
      copyable: Boolean(current?.working_dir?.trim()),
      key: 'workingDir',
      label: t('container.list.detail.workingDir'),
      rawValue: current?.working_dir?.trim() || '',
      value: current?.working_dir?.trim() || '-',
    },
  ];
});
const environmentPolicyFilterOptions = computed(() => [
  { label: t('container.detail.config.policyFilter.all'), value: 'all' },
  { label: t('container.detail.config.policy.plain'), value: 'plain' },
  { label: t('container.detail.config.policy.masked'), value: 'masked' },
  { label: t('container.detail.config.policy.hidden'), value: 'hidden' },
  { label: t('container.detail.config.policy.sensitive'), value: 'sensitive' },
]);
const filteredEnvironmentRows = computed(() => {
  const keyword = environmentKeyword.value.trim().toLowerCase();
  const policy = environmentPolicyFilter.value;

  return environmentRows.value.filter((row) => {
    const matchesPolicy =
      policy === 'all' || row.policy === policy || (policy === 'sensitive' && isSensitiveEnvironmentRow(row));
    if (!matchesPolicy) {
      return false;
    }
    if (!keyword) {
      return true;
    }
    return row.name.toLowerCase().includes(keyword) || row.value.toLowerCase().includes(keyword);
  });
});
const environmentEmptyState = computed(() => {
  if (!environmentRows.value.length) {
    return {
      description: t('container.detail.config.environmentUnavailable'),
      title: t('container.detail.config.environmentEmptyTitle'),
    };
  }
  return {
    description: t('container.detail.config.environmentFilterEmptyDescription'),
    title: t('container.detail.config.environmentFilterEmptyTitle'),
  };
});
const healthDiagnosis = computed(() => {
  const current = safeDetail.value;
  if (!current) {
    return {
      description: '-',
      label: '-',
      theme: 'default' as HealthStatusTheme,
    };
  }
  return resolveHealthDiagnosis(current);
});
const healthcheckDetails = computed(() => resolveHealthcheckDetails(safeDetail.value?.healthcheck));
const inspectUpdatedRelative = computed(() => {
  const current = safeDetail.value;
  if (!current?.inspect_updated_at) {
    return t('container.detail.health.noRecentCheck');
  }
  return t('container.detail.health.updatedFromInspect');
});

function openAuditLogs() {
  if (!safeDetail.value) {
    return;
  }

  void router.push(buildAuditResourceLocation('container', safeDetail.value.id, displayName(safeDetail.value)));
}

const runtimeStability = computed(() => {
  const current = safeDetail.value;
  const risk = resolveRuntimeStability(current);
  return {
    ...risk,
    lastExitCode: formatNullableNumber(current?.last_exit_code),
    oomKilled: formatNullableBoolean(current?.oom_killed),
  };
});
const resourceMetrics = computed(() => {
  const current = safeDetail.value;
  const resource = current?.resource;
  const cpuValue = formatPercent(resource?.cpu_percent);
  const memoryPercent = formatPercent(resource?.memory_percent);
  const status = current ? resourceStatus(current) : emptyResourceStatus();

  return {
    cpu: {
      description:
        cpuValue === '-' ? t('container.detail.resources.noData') : t('container.detail.resources.currentSnapshot'),
      progress: toProgressPercent(resource?.cpu_percent),
      progressLabel: cpuValue,
      value: cpuValue,
    },
    memory: {
      description: memoryPercent,
      progress: toProgressPercent(resource?.memory_percent),
      progressLabel: memoryPercent,
      value: current ? memorySummary(current) : '-',
    },
    status,
  };
});
const realtimeSocketState = computed(() => selectContainerStatsRealtimeState(currentDetailStatsKey.value));
const resourceChange = computed(() => {
  const change = selectContainerStatsChangeState(currentDetailStatsKey.value);
  return {
    cpuClass: metricChangedClass(change, 'cpu'),
    cpuStatus: metricProgressStatus(change.cpu),
    memoryClass: metricChangedClass(change, 'memory'),
    memoryStatus: metricProgressStatus(change.memory),
  };
});
const realtimeStatusTheme = computed(() => {
  if (!realtimeEnabled.value) {
    return 'default';
  }
  if (realtimeSocketState.value === 'error') {
    return 'danger';
  }
  if (realtimeSocketState.value === 'connecting' || realtimeSocketState.value === 'closed') {
    return 'warning';
  }
  if (realtimeSocketState.value === 'open') {
    return 'success';
  }
  return 'default';
});
const realtimeStatusLabel = computed(() => {
  if (!realtimeEnabled.value) {
    return t('container.detail.realtime.paused');
  }
  if (realtimeSocketState.value === 'error') {
    return t('container.detail.realtime.degraded');
  }
  if (realtimeSocketState.value === 'connecting' || realtimeSocketState.value === 'closed') {
    return t('container.detail.realtime.connecting');
  }
  if (realtimeSocketState.value === 'open') {
    return t('container.detail.realtime.live');
  }
  return t('container.detail.realtime.idle');
});
const realtimeStatusHint = computed(() =>
  realtimeEnabled.value ? t('container.detail.realtime.hint') : t('container.detail.realtime.pausedHint'),
);
const realtimeToggleLabel = computed(() =>
  realtimeEnabled.value ? t('container.detail.realtime.pause') : t('container.detail.realtime.resume'),
);
const resourceHistoryPoints = computed<ContainerStatsHistoryPoint[]>(() => {
  const containerId = safeDetail.value?.id;
  if (!containerId) {
    return [];
  }

  return selectContainerStatsHistory(containerId)
    .filter((item) => item.resource.collected_at)
    .slice(-6)
    .map((item, index) => ({
      collectedAt: formatTime(item.resource.collected_at),
      cpuPercent: formatPercent(item.resource.cpu_percent),
      key: `${item.resource.collected_at || 'snapshot'}-${index}`,
      memoryPercent: formatPercent(item.resource.memory_percent),
    }));
});
const cpuDetailMetrics = computed(() =>
  buildCpuDetailMetrics(
    safeDetail.value?.resource,
    {
      cpuLimit: t('container.detail.resources.cpuLimitWithOnline'),
      cpuPercent: t('container.detail.resources.cpuPercent'),
      kernelTime: t('container.detail.resources.cpuKernelTime'),
      systemCpuTime: t('container.detail.resources.systemCpuTime'),
      throttlingCount: t('container.detail.resources.throttlingCount'),
      throttlingInactiveHint: t('container.detail.resources.throttlingInactiveHint'),
      throttlingSignalHint: t('container.detail.resources.throttlingSignalHint'),
      throttlingTime: t('container.detail.resources.throttlingTime'),
      totalCpuTime: t('container.detail.resources.totalCpuTime'),
      userTime: t('container.detail.resources.cpuUserTime'),
    },
    locale.value,
  ),
);
const resourceDetailGroups = computed<ResourceDetailGroup[]>(() => {
  const current = safeDetail.value;
  if (!current) {
    return [];
  }
  const status = resourceStatus(current);

  return [
    {
      key: 'memory',
      title: t('container.detail.resources.memoryDetails'),
      rows: [
        ...metricRows([
          ['memory_usage_bytes', t('container.detail.resources.memoryUsage'), 'bytes'],
          ['memory_cache', t('container.detail.resources.memoryCache'), 'bytes'],
          ['memory_limit_bytes', t('container.detail.resources.memoryLimit'), 'bytes'],
          ['memory_rss', t('container.detail.resources.memoryRss'), 'bytes'],
          ['memory_percent', t('container.detail.resources.memoryPercent'), 'percent'],
          ['memory_active_file', t('container.detail.resources.memoryActiveFile'), 'bytes'],
          ['memory_inactive_file', t('container.detail.resources.memoryInactiveFile'), 'bytes'],
          ['memory_pgfault', t('container.detail.resources.memoryPgfault'), 'number'],
          ['memory_pgmajfault', t('container.detail.resources.memoryPgmajfault'), 'number'],
        ]),
        {
          key: 'memory-placeholder',
          label: '',
          type: 'placeholder',
          value: '—',
        },
      ],
    },
    {
      key: 'cpu',
      title: t('container.detail.resources.cpuDetails'),
      rows: [],
    },
    {
      key: 'network',
      title: t('container.detail.resources.networkIo'),
      rows: metricRows([
        ['rx_bytes', t('container.detail.resources.rxBytes'), 'bytes'],
        ['tx_bytes', t('container.detail.resources.txBytes'), 'bytes'],
        ['rx_packets', t('container.detail.resources.rxPackets'), 'number'],
        ['tx_packets', t('container.detail.resources.txPackets'), 'number'],
        ['rx_errors', t('container.detail.resources.rxErrors'), 'number'],
        ['tx_errors', t('container.detail.resources.txErrors'), 'number'],
        ['rx_dropped', t('container.detail.resources.rxDropped'), 'number'],
        ['tx_dropped', t('container.detail.resources.txDropped'), 'number'],
      ]),
    },
    {
      key: 'process',
      title: t('container.detail.resources.processInfo'),
      rows: [
        ...metricRows([
          ['pids_current', t('container.detail.resources.pidsCurrent'), 'number'],
          ['pids_limit', t('container.detail.resources.pidsLimit'), 'number'],
        ]),
        {
          key: 'status',
          label: t('container.detail.resources.status'),
          theme: status.theme,
          type: 'tag',
          value: status.value,
        },
        {
          key: 'collected-at',
          label: t('container.detail.resources.collectedAt'),
          type: 'text',
          value: status.collectedAt === '-' ? notCollectedLabel() : status.collectedAt,
        },
      ],
    },
  ];
});
const overviewSections = computed<ContainerOverviewInfoSection[]>(() => {
  const current = safeDetail.value;
  if (!current) {
    return [];
  }

  const imageId = readableImageId(current.image_id);

  return [
    {
      key: 'basic',
      title: t('container.detail.overview.basicInfo'),
      rows: [
        {
          displayValue: displayName(current),
          key: 'name',
          label: t('container.detail.overview.fields.name'),
          type: 'text',
        },
        {
          code: true,
          copyValue: current.id,
          displayValue: shortContainerId(current),
          key: 'container-id',
          label: t('container.detail.overview.fields.containerId'),
          testId: 'container-id-copy',
          type: 'copy',
        },
        {
          copyValue: current.image,
          displayValue: current.image || '-',
          key: 'image',
          label: t('container.detail.overview.fields.image'),
          type: 'copy',
        },
        {
          code: true,
          copyValue: imageId,
          displayValue: shortIdentifier(imageId),
          key: 'image-id',
          label: t('container.detail.overview.fields.imageId'),
          testId: 'image-id-copy',
          type: 'copy',
        },
        {
          displayValue: runtimeLabel(current),
          key: 'runtime',
          label: t('container.detail.overview.fields.runtime'),
          type: 'text',
        },
      ],
    },
    {
      key: 'runtime',
      title: t('container.detail.overview.runtimeInfo'),
      rows: [
        {
          key: 'status',
          label: t('container.detail.overview.fields.status'),
          tagLabel: stateLabel(current.state),
          tagTheme: stateTheme(current.state),
          type: 'tag',
        },
        {
          displayValue: current.state || '-',
          key: 'state',
          label: t('container.detail.overview.fields.state'),
          type: 'text',
        },
        {
          key: 'health',
          label: t('container.detail.overview.fields.health'),
          tagLabel: healthLabel(current.health),
          tagTheme: healthTheme(current.health),
          type: 'tag',
        },
        {
          displayValue: formatTime(current.created_at),
          key: 'created-at',
          label: t('container.detail.overview.fields.createdAt'),
          type: 'text',
        },
        {
          displayValue: formatTime(current.started_at),
          key: 'started-at',
          label: t('container.detail.overview.fields.startedAt'),
          type: 'text',
        },
        {
          displayValue: formatTime(current.inspect_updated_at),
          key: 'updated-at',
          label: t('container.detail.overview.fields.updatedAt'),
          type: 'text',
        },
      ],
    },
    {
      key: 'resource-network',
      title: t('container.detail.overview.resourceNetwork'),
      rows: [
        {
          displayValue: formatPercent(current.resource?.cpu_percent),
          key: 'cpu',
          label: t('container.detail.resources.cpu'),
          type: 'text',
        },
        {
          displayValue: memorySummary(current),
          key: 'memory',
          label: t('container.detail.resources.memory'),
          type: 'text',
        },
        {
          displayValue: current.primary_ip || '-',
          key: 'primary-ip',
          label: t('container.detail.network.primaryIp'),
          type: 'text',
        },
        {
          displayValue: networkSummary(current),
          key: 'network-mode',
          label: t('container.detail.overview.fields.networkMode'),
          type: 'text',
        },
        {
          displayValue: primaryNetworkName(current),
          key: 'network-name',
          label: t('container.detail.overview.fields.networkName'),
          type: 'text',
        },
        {
          emptyLabel: t('container.detail.network.noPublicPorts'),
          key: 'ports',
          label: t('container.detail.network.ports'),
          ports: current.ports.map((port) => portLabel(port)),
          type: 'ports',
        },
      ],
    },
  ];
});
const environmentColumns = computed<TableProps['columns']>(() => [
  { colKey: 'name', title: t('container.detail.config.envName'), width: '40%', ellipsis: true },
  { colKey: 'value', title: t('container.detail.config.envValue'), width: '40%', ellipsis: true },
  { colKey: 'policy', title: t('container.detail.config.envPolicy'), width: 120, align: 'center' },
  { colKey: 'operation', title: t('container.detail.operation'), width: 72, align: 'center' },
]);
const networkColumns = computed<TableProps['columns']>(() => [
  { colKey: 'name', title: t('container.detail.network.name'), minWidth: 180, ellipsis: true },
  { colKey: 'ip_address', title: t('container.detail.network.ipAddress'), minWidth: 160, ellipsis: true },
  { colKey: 'gateway', title: t('container.detail.network.gateway'), minWidth: 160, ellipsis: true },
  { colKey: 'mac_address', title: t('container.detail.network.macAddress'), minWidth: 180, ellipsis: true },
  { colKey: 'network_id', title: t('container.detail.network.networkId'), minWidth: 180, ellipsis: true },
  { colKey: 'endpoint_id', title: t('container.detail.network.endpointId'), minWidth: 180, ellipsis: true },
]);
const portColumns = computed<TableProps['columns']>(() => [
  { colKey: 'publicPort', title: t('container.detail.network.hostPort'), minWidth: 140, ellipsis: true },
  { colKey: 'privatePort', title: t('container.detail.network.containerPort'), minWidth: 140, ellipsis: true },
  { colKey: 'protocol', title: t('container.detail.network.protocol'), width: 112, align: 'center' },
  { colKey: 'listenAddress', title: t('container.detail.network.listenAddress'), minWidth: 160, ellipsis: true },
  { colKey: 'mapping', title: t('container.detail.network.mapping'), minWidth: 260, ellipsis: true },
]);

onMounted(() => {
  detailPageActive.value = true;
  updateCurrentTabTitle(fallbackTitle.value);
  void refreshContainerDetail();
  syncLogsRealtimeSubscription();
});

onUnmounted(() => {
  detailPageActive.value = false;
  releaseCurrentRealtimeSubscription();
  releaseLogsRealtimeSubscription();
  logsRealtimeBatcher.destroy();
});

onActivated(() => {
  detailPageActive.value = true;
  if (safeDetail.value?.id) {
    syncRealtimeSubscription(safeDetail.value.id);
  }
  syncLogsRealtimeSubscription();
});

onDeactivated(() => {
  detailPageActive.value = false;
  releaseCurrentRealtimeSubscription();
  releaseLogsRealtimeSubscription();
});

watch(
  () => route.params.id,
  () => {
    resetDetailState();
    void refreshContainerDetail();
    syncLogsRealtimeSubscription();
  },
);

watch(
  () => route.query.tab,
  (tab) => {
    const normalized = normalizeTab(tab);
    activeTab.value = normalized;
    syncLogsRealtimeSubscription();
  },
);

watch(logLineLimit, () => {
  logsRealtimeBatcher.updateLineLimit(logLineLimit.value);
  if (activeTab.value === 'logs') {
    releaseLogsRealtimeSubscription();
    void loadLogs();
  }
});

async function handleManualRefresh() {
  await refreshContainerDetail();
  if (!error.value) {
    MessagePlugin.success(t('container.detail.refreshSuccess'));
  }
}

async function refreshContainerDetail() {
  const currentContainerId = containerId.value;
  if (!currentContainerId) {
    detail.value = null;
    logViewStore.reset();
    error.value = t('container.detail.missingId');
    return;
  }

  const requestSeq = ++detailRefreshSeq;
  detailRefreshing.value = true;
  error.value = '';
  try {
    const nextDetail = await getContainer(currentContainerId);
    if (requestSeq !== detailRefreshSeq || currentContainerId !== containerId.value) {
      return;
    }
    detail.value = nextDetail ? mergeDetailWithLocalMountUsage(nextDetail) : null;
    if (detail.value) {
      seedContainerDetail(detail.value);
    }
    const current = safeDetail.value;
    if (current) {
      updateCurrentTabTitle(buildDetailTitle(displayName(current)));
      syncRealtimeSubscription(current.id);
    }
    await fetchMountUsage(currentContainerId, requestSeq);
  } catch (loadError) {
    if (requestSeq !== detailRefreshSeq || currentContainerId !== containerId.value) {
      return;
    }
    clearContainerDetail(detail.value?.id);
    clearContainerDetail(currentContainerId);
    detail.value = null;
    releaseCurrentRealtimeSubscription();
    error.value = resolveLocalizedErrorMessage(t, loadError, t('container.list.detail.loadFailed'));
    logger.warn('failed to fetch container detail', loadError);
  } finally {
    if (requestSeq === detailRefreshSeq) {
      detailRefreshing.value = false;
    }
  }
}

async function fetchMountUsage(currentContainerId = containerId.value, requestSeq = detailRefreshSeq) {
  if (!currentContainerId || !detail.value) {
    return;
  }
  try {
    const usageList = await getContainerMountUsage(currentContainerId);
    if (requestSeq !== detailRefreshSeq || currentContainerId !== containerId.value) {
      return;
    }
    mergeMountUsage(usageList.items ?? []);
  } catch (loadError) {
    MessagePlugin.warning(t('container.detail.storage.syncFailed'));
    logger.warn('failed to fetch cached container mount usage', loadError);
  }
}

function mergeDetailWithLocalMountUsage(nextDetail: ContainerDetail): ContainerDetail {
  const currentMounts = safeDetail.value?.mounts ?? [];
  if (!Array.isArray(nextDetail.mounts)) {
    return nextDetail;
  }
  const orderedMounts = orderMountsByCurrentPosition(nextDetail.mounts, currentMounts);
  const orderedDetail = {
    ...nextDetail,
    mounts: orderedMounts,
  };
  if (currentMounts.length === 0) {
    return orderedDetail;
  }
  const currentUsageByKey = new Map<string, ContainerMountUsage>();
  currentMounts.forEach((mount, index) => {
    const key = stableMountIdentity(mount);
    if (key && shouldKeepLocalMountUsage(mount, index) && mount.usage) {
      currentUsageByKey.set(key, mount.usage);
    }
  });
  if (currentUsageByKey.size === 0) {
    return orderedDetail;
  }
  return {
    ...orderedDetail,
    mounts: orderedMounts.map((mount) => {
      const usage = currentUsageByKey.get(stableMountIdentity(mount));
      return usage ? { ...mount, usage } : mount;
    }),
  };
}

function orderMountsByCurrentPosition(nextMounts: ContainerMount[], currentMounts: ContainerMount[]) {
  if (!currentMounts.length) {
    return nextMounts;
  }

  const currentPosition = new Map<string, number>();
  currentMounts.forEach((mount, index) => {
    const key = stableMountIdentity(mount);
    if (key) {
      currentPosition.set(key, index);
    }
  });

  return [...nextMounts].sort((left, right) => {
    const leftPosition = currentPosition.get(stableMountIdentity(left));
    const rightPosition = currentPosition.get(stableMountIdentity(right));
    if (leftPosition !== undefined && rightPosition !== undefined) {
      return leftPosition - rightPosition;
    }
    if (leftPosition !== undefined) {
      return -1;
    }
    if (rightPosition !== undefined) {
      return 1;
    }
    return stableMountIdentity(left).localeCompare(stableMountIdentity(right));
  });
}

async function loadLogs() {
  const currentContainerId = containerId.value;
  if (!currentContainerId) {
    logViewStore.reset();
    logsRealtimeSeeded = false;
    logsInitialLoadAttempted = false;
    return;
  }
  const requestSeq = ++logsRefreshSeq;
  logsInitialLoadAttempted = true;
  logViewStore.setLoading(true);
  logViewStore.setError('');
  try {
    const nextLogs = await getContainerLogs(currentContainerId, {
      ...DEFAULT_LOG_QUERY,
      tail: logLineLimit.value,
    });
    if (requestSeq !== logsRefreshSeq || currentContainerId !== containerId.value) {
      return;
    }
    logsRealtimeBatcher.seed(nextLogs);
    logsRealtimeSeeded = true;
  } catch (loadError) {
    if (requestSeq !== logsRefreshSeq || currentContainerId !== containerId.value) {
      return;
    }
    logViewStore.setError(resolveLocalizedErrorMessage(t, loadError, t('container.list.logs.loadFailed')));
    logger.warn('failed to fetch container logs', loadError);
  } finally {
    if (requestSeq === logsRefreshSeq) {
      logViewStore.setLoading(false);
      syncLogsRealtimeSubscription();
    }
  }
}

async function copyHealthcheckCommand() {
  if (!healthcheckDetails.value.configured || healthcheckDetails.value.command === '-') {
    return;
  }
  try {
    await copyTextToClipboard(healthcheckDetails.value.command);
    MessagePlugin.success(t('container.detail.copySuccess'));
  } catch (copyError) {
    MessagePlugin.error(t('container.detail.copyError'));
    logger.warn('failed to copy healthcheck command', copyError);
  }
}

async function copyRuntimeConfigValue(item: RuntimeConfigItem) {
  if (!item.copyable) {
    return;
  }
  const copied = await copyTextToClipboard(item.rawValue);
  if (copied) {
    MessagePlugin.success(t('container.detail.config.copyRuntimeSuccess'));
    return;
  }
  MessagePlugin.error(t('container.detail.copyError'));
}

function resetDetailState() {
  clearContainerDetail(detail.value?.id);
  clearContainerDetail(containerId.value);
  detail.value = null;
  error.value = '';
  logViewStore.reset();
  logsRealtimeSeeded = false;
  logsInitialLoadAttempted = false;
  logsRefreshSeq += 1;
  logsRealtimeBatcher.clear();
  refreshingMountKeys.value = new Set();
  releaseCurrentRealtimeSubscription();
  releaseLogsRealtimeSubscription();
  updateCurrentTabTitle(fallbackTitle.value);
}

function toggleRealtimeSubscription() {
  realtimeEnabled.value = !realtimeEnabled.value;
  if (!realtimeEnabled.value) {
    releaseCurrentRealtimeSubscription();
    return;
  }
  if (safeDetail.value?.id) {
    syncRealtimeSubscription(safeDetail.value.id);
  }
}

function toggleLogsPause() {
  if (logsPaused.value) {
    logViewStore.resume();
    return;
  }

  logViewStore.pause();
}

function clearLogsScreen() {
  logsRealtimeBatcher.clearView();
}

function reconnectLogsRealtime() {
  logsRealtimeController?.reconnect();
}

function syncRealtimeSubscription(nextContainerId: string) {
  if (!detailPageActive.value || !realtimeEnabled.value) {
    releaseCurrentRealtimeSubscription();
    return;
  }
  if (activeRealtimeSubscriptionId.value === nextContainerId) {
    return;
  }
  if (activeRealtimeSubscriptionId.value && activeRealtimeSubscriptionId.value !== nextContainerId) {
    releaseContainerStatsSubscription(activeRealtimeSubscriptionId.value);
    activeRealtimeSubscriptionId.value = '';
  }
  acquireContainerStatsSubscription(nextContainerId);
  activeRealtimeSubscriptionId.value = nextContainerId;
}

function releaseCurrentRealtimeSubscription() {
  const currentSubscriptionId = activeRealtimeSubscriptionId.value;
  if (!currentSubscriptionId) {
    return;
  }
  releaseContainerStatsSubscription(currentSubscriptionId);
  activeRealtimeSubscriptionId.value = '';
}

function handleTabChange(value: string | number) {
  const tab = normalizeTab(value);
  activeTab.value = tab;
  void router.replace({
    params: route.params,
    query: {
      ...route.query,
      tab,
    },
  });
  syncLogsRealtimeSubscription();
}

function syncLogsRealtimeSubscription() {
  const nextTopic =
    detailPageActive.value && activeTab.value === 'logs' && containerId.value
      ? buildContainerLogsTopicName(containerId.value)
      : '';

  if (!nextTopic) {
    releaseLogsRealtimeSubscription();
    return;
  }

  if (!logsRealtimeSeeded && !logsLoading.value && !logsInitialLoadAttempted) {
    void loadLogs();
    return;
  }

  if (logsRealtimeTopic === nextTopic && logsRealtimeController) {
    return;
  }

  releaseLogsRealtimeSubscription();
  logsRealtimeTopic = nextTopic;
  logsRealtimeController = openRealtimeTopicSocket<ContainerLogsRealtimeMessage>({
    topic: nextTopic,
    parseMessage: parseContainerLogsRealtimeMessage,
    onMessage: applyLogsRealtimeMessage,
    onStateChange: (state) => {
      logsSocketState.value = state;
    },
    onError: (message) => {
      logger.warn('container log realtime subscription error', { message, topic: nextTopic });
    },
  });
}

function releaseLogsRealtimeSubscription() {
  logsRealtimeBatcher.flush();
  logsRealtimeTopic = '';
  logsRealtimeController?.close();
  logsRealtimeController = null;
  logsSocketState.value = 'idle';
}

type ContainerLogsRealtimeMessage = {
  containerId: string;
  entries: ContainerLogEntry[];
  topic: string;
};

function applyLogsRealtimeMessage(message: ContainerLogsRealtimeMessage) {
  if (!detailPageActive.value || activeTab.value !== 'logs' || !containerId.value) {
    return;
  }
  if (!isContainerLogsTopicForContainer(message.topic, containerId.value)) {
    return;
  }
  if (message.containerId && message.containerId !== containerId.value) {
    return;
  }

  logsRealtimeBatcher.enqueue(message.entries);
}

function parseContainerLogsRealtimeMessage(raw: unknown): ContainerLogsRealtimeMessage | null {
  const eventData = parseRealtimeEventData(raw);
  if (!eventData) {
    return null;
  }

  const candidateEntries = readRealtimeLogEntries(eventData);
  if (!candidateEntries.length) {
    return null;
  }

  const topic = readRealtimeLogTopic(eventData);
  const containerId = readRealtimeLogContainerId(eventData, topic);
  if (!topic || !containerId) {
    return null;
  }

  return {
    containerId,
    entries: candidateEntries,
    topic,
  };
}

function parseRealtimeEventData(raw: unknown): Record<string, unknown> | null {
  const parsed = typeof raw === 'string' ? safeParseJson(raw) : raw;
  if (!isPlainObject(parsed)) {
    return null;
  }
  if (isPlainObject(parsed.data)) {
    return parsed.data;
  }
  return parsed;
}

function readRealtimeLogEntries(value: Record<string, unknown>) {
  const directEntries = normalizeRealtimeLogEntries(value.entries);
  if (directEntries.length) {
    return directEntries;
  }

  const directEntry = normalizeStructuredLogEntry(value.entry);
  if (directEntry) {
    return [toContainerLogEntry(directEntry)];
  }

  const nestedPayloads = [value.payload, value.snapshot, value.append, value.delta];
  for (const candidate of nestedPayloads) {
    if (!isPlainObject(candidate)) {
      continue;
    }
    const nestedEntries = normalizeRealtimeLogEntries(candidate.entries);
    if (nestedEntries.length) {
      return nestedEntries;
    }
    const nestedEntry = normalizeStructuredLogEntry(candidate.entry);
    if (nestedEntry) {
      return [toContainerLogEntry(nestedEntry)];
    }
  }

  return [];
}

function normalizeRealtimeLogEntries(value: unknown) {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map((entry) => normalizeStructuredLogEntry(entry))
    .filter((entry): entry is NonNullable<ReturnType<typeof normalizeStructuredLogEntry>> => entry !== null)
    .map(toContainerLogEntry);
}

function readRealtimeLogTopic(value: Record<string, unknown>) {
  const directTopic = typeof value.topic === 'string' ? value.topic.trim() : '';
  if (directTopic) {
    return directTopic;
  }

  const nestedPayloads = [value.payload, value.snapshot, value.append, value.delta];
  for (const candidate of nestedPayloads) {
    if (!isPlainObject(candidate) || typeof candidate.topic !== 'string') {
      continue;
    }
    const nestedTopic = candidate.topic.trim();
    if (nestedTopic) {
      return nestedTopic;
    }
  }

  return '';
}

function readRealtimeLogContainerId(value: Record<string, unknown>, topic: string) {
  const directId = typeof value.id === 'string' ? value.id.trim() : '';
  if (directId) {
    return directId;
  }

  const nestedPayloads = [value.payload, value.snapshot, value.append, value.delta];
  for (const candidate of nestedPayloads) {
    if (!isPlainObject(candidate) || typeof candidate.id !== 'string') {
      continue;
    }
    const nestedId = candidate.id.trim();
    if (nestedId) {
      return nestedId;
    }
  }

  return parseContainerLogsTopicContainerId(topic);
}

function safeParseJson(raw: string) {
  try {
    return JSON.parse(raw) as unknown;
  } catch {
    return null;
  }
}

function toContainerLogEntry(entry: NonNullable<ReturnType<typeof normalizeStructuredLogEntry>>): ContainerLogEntry {
  return {
    line: entry.line,
    occurred_at: entry.occurredAt,
    stream: entry.stream,
  };
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value));
}

async function copyEnvironmentValue(row: EnvironmentRow) {
  if (!row.copyable || row.copyDisabled) {
    return;
  }
  const copied = await copyTextToClipboard(row.copyValue);
  if (copied) {
    MessagePlugin.success(t('container.detail.config.copyVariableValueSuccess'));
    return;
  }
  MessagePlugin.error(t('container.detail.copyError'));
}

async function copyFilteredEnvironmentAsEnv() {
  if (environmentCopyDisabled.value) {
    MessagePlugin.error(t('container.detail.config.copyPolicyDisabled'));
    return;
  }
  const content = filteredEnvironmentRows.value.map(formatEnvironmentLine).join('\n');
  const copied = await copyTextToClipboard(content);
  if (copied) {
    MessagePlugin.success(t('container.detail.config.copyEnvSuccess'));
    return;
  }
  MessagePlugin.error(t('container.detail.copyError'));
}

async function copyDetailText(text: string) {
  if (!text) return;
  const copied = await copyTextToClipboard(text);
  if (copied) {
    MessagePlugin.success(t('container.detail.copySuccess'));
    return;
  }
  MessagePlugin.error(t('container.detail.copyError'));
}

async function refreshMountUsage(mount: MountCard) {
  if (!containerId.value || mount.usageRefreshDisabled || refreshingMountKeys.value.has(mount.key)) {
    return;
  }

  setMountRefreshing(mount.key, true);
  mergeOneMountUsage(
    mount.key,
    {
      container_id: containerId.value,
      destination: mount.destination,
      mount_id: mount.mountId,
      source: mount.source,
      status: 'pending',
      type: mount.raw.type,
      message: t('container.detail.storage.pendingMessage'),
    },
    { force: true },
  );
  try {
    const refreshedUsage = await postContainerMountUsageRefresh(containerId.value, mount.mountId);
    mergeOneMountUsage(mount.key, refreshedUsage, { force: true });
    MessagePlugin.success(t('container.detail.storage.refreshSuccess'));
  } catch (refreshError) {
    logger.warn('failed to refresh container mount usage', refreshError);
    MessagePlugin.error(resolveLocalizedErrorMessage(t, refreshError, t('container.detail.storage.refreshError')));
  } finally {
    setMountRefreshing(mount.key, false);
  }
}

function setMountRefreshing(key: string, refreshing: boolean) {
  const nextKeys = new Set(refreshingMountKeys.value);
  if (refreshing) {
    nextKeys.add(key);
  } else {
    nextKeys.delete(key);
  }
  refreshingMountKeys.value = nextKeys;
}

function mergeMountUsage(usages: ContainerMountUsage[]) {
  if (!detail.value || usages.length === 0) {
    return;
  }
  const byMountID = new Map<string, ContainerMountUsage>();
  usages.forEach((usage) => {
    const record = readUnknownRecord(usage);
    const mountID = readString(record?.mount_id);
    if (mountID) {
      byMountID.set(mountID, usage);
    }
  });
  detail.value = {
    ...detail.value,
    mounts: (safeDetail.value?.mounts ?? []).map((mount, index) => {
      const usage = byMountID.get(mount.mount_id);
      if (!usage || shouldKeepLocalMountUsage(mount, index)) {
        return mount;
      }
      return { ...mount, usage };
    }),
  };
}

function mergeOneMountUsage(key: string, usage: ContainerMountUsage, options: { force?: boolean } = {}) {
  if (!detail.value) {
    return;
  }
  const usageMountKey = usage.mount_id?.trim() || '';
  detail.value = {
    ...detail.value,
    mounts: (safeDetail.value?.mounts ?? []).map((item, index) => {
      if (mountKey(item, index) !== key && (!usageMountKey || stableMountIdentity(item) !== usageMountKey)) {
        return item;
      }
      if (!options.force && shouldKeepLocalMountUsage(item, index)) {
        return item;
      }
      return { ...item, usage };
    }),
  };
}

function shouldKeepLocalMountUsage(mount: ContainerMount, index: number) {
  const key = mountKey(mount, index);
  if (refreshingMountKeys.value.has(key)) {
    return true;
  }
  const status = readMountUsage(mount).status;
  return status === 'pending';
}

function normalizeTab(value: unknown): DetailTab {
  const raw = Array.isArray(value) ? value[0] : value;
  return typeof raw === 'string' && DETAIL_TABS.includes(raw as DetailTab) ? (raw as DetailTab) : 'overview';
}

function normalizeEnvironmentRows(nextDetail: ContainerDetail | null): EnvironmentRow[] {
  const detailRecord = readUnknownRecord(nextDetail);
  const source = detailRecord?.environment;
  if (!Array.isArray(source)) {
    return [];
  }

  return source.flatMap((item) => {
    const record = readUnknownRecord(item);
    const name = readString(record?.name ?? record?.key);
    if (!name) {
      return [];
    }

    const rawPolicy = readString(record?.policy ?? record?.visibility ?? record?.state);
    const masked = record?.masked === true;
    const rawValue = readRawString(record?.value);
    const displayValue = readRawString(record?.display_value);
    const copyValue = readRawString(record?.copy_value);
    const valueMasked = record?.value_masked === true;
    const valueHidden = record?.value_hidden === true;
    const policy = normalizeEnvironmentPolicy(
      rawPolicy,
      readString(detailRecord?.environment_policy),
      masked,
      rawValue,
      valueMasked,
      valueHidden,
    );
    const value = resolveEnvironmentDisplayValue(rawValue, displayValue, policy, valueMasked, valueHidden);
    const hasSensitiveValue = masked || valueMasked || valueHidden || record?.sensitive === true;
    const maskedCopyAllowed = containerEnvironmentMaskedCopyEnabled(nextDetail);
    const resolvedCopyValue =
      hasSensitiveValue && policy !== 'plain' && !maskedCopyAllowed
        ? ''
        : resolveEnvironmentCopyValue(rawValue, copyValue, policy, hasSensitiveValue);
    const copyDisabled = !resolvedCopyValue;

    return [
      {
        copyValue: resolvedCopyValue,
        copyable: Boolean(resolvedCopyValue),
        copyDisabled,
        copyTooltip: copyDisabled
          ? t('container.detail.config.copyPolicyDisabled')
          : t('container.detail.config.copyRealValueTooltip'),
        displayValue: value,
        hasSensitiveValue,
        name,
        policy,
        rawJsonCopyValue: resolvedCopyValue,
        rawJsonValue:
          valueHidden || policy === 'hidden'
            ? '[HIDDEN]'
            : valueMasked || policy === 'masked'
              ? value
              : displayValue || rawValue,
        rawValue,
        value: value || environmentValueFallback(policy),
      },
    ];
  });
}

function readUnknownRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' && !Array.isArray(value) ? (value as Record<string, unknown>) : null;
}

function readString(value: unknown) {
  return typeof value === 'string' ? value.trim() : '';
}

function readRawString(value: unknown) {
  return typeof value === 'string' ? value : '';
}

function normalizeEnvironmentPolicy(
  value: string,
  detailPolicy = '',
  masked = false,
  rawValue = '',
  valueMasked = false,
  valueHidden = false,
): EnvironmentPolicy {
  if (value === 'plain' || value === 'masked' || value === 'hidden') {
    return value;
  }
  if (valueHidden) {
    return 'hidden';
  }
  if (valueMasked) {
    return 'masked';
  }
  if (rawValue && !masked) {
    return 'plain';
  }
  if (masked) {
    return 'masked';
  }
  if (detailPolicy === 'plain' || detailPolicy === 'masked' || detailPolicy === 'hidden') {
    return detailPolicy;
  }
  return 'unknown';
}

function environmentValueFallback(policy: EnvironmentPolicy) {
  if (policy === 'masked') return t('container.detail.config.maskedValue');
  if (policy === 'hidden') return t('container.detail.config.hiddenValue');
  return '-';
}

function displayEnvironmentValue(value: string, policy: EnvironmentPolicy) {
  if (policy === 'masked') {
    return t('container.detail.config.maskedValue');
  }
  return abbreviateMiddle(value);
}

function abbreviateMiddle(value: string, maxLength = 32) {
  if (value.length <= maxLength) {
    return value;
  }
  const edgeLength = Math.max(4, Math.floor((maxLength - 3) / 2));
  return `${value.slice(0, edgeLength)}...${value.slice(-edgeLength)}`;
}

function policyLabel(policy: EnvironmentPolicy) {
  return t(`container.detail.config.policy.${policy}`);
}

function policyTheme(policy: EnvironmentPolicy) {
  if (policy === 'plain') return 'success';
  if (policy === 'masked') return 'warning';
  if (policy === 'hidden') return 'danger';
  return 'default';
}

function isBooleanEnvironmentValue(value: string) {
  const normalized = value.trim().toLowerCase();
  return normalized === 'true' || normalized === 'false';
}

function isSensitiveEnvironmentRow(row: EnvironmentRow) {
  if (row.policy === 'masked' || row.policy === 'hidden') {
    return true;
  }
  return /(?:password|passwd|pwd|secret|token|key|credential|auth)/i.test(row.name);
}

function formatEnvironmentLine(row: EnvironmentRow) {
  return `${row.name}=${row.copyValue}`;
}

function resolveEnvironmentCopyValue(
  rawValue: string,
  copyValue: string,
  policy: EnvironmentPolicy,
  hasSensitiveValue: boolean,
) {
  if (policy === 'plain') {
    return rawValue;
  }
  if (!hasSensitiveValue) {
    return rawValue;
  }
  return copyValue;
}

function resolveEnvironmentDisplayValue(
  rawValue: string,
  displayValue: string,
  policy: EnvironmentPolicy,
  valueMasked: boolean,
  valueHidden: boolean,
) {
  if (isMaskedEnvironmentDisplayValue(displayValue)) {
    return t('container.detail.config.maskedValue');
  }
  if (isHiddenEnvironmentDisplayValue(displayValue)) {
    return t('container.detail.config.hiddenValue');
  }
  if (valueHidden || policy === 'hidden') {
    return t('container.detail.config.hiddenValue');
  }
  if (valueMasked || policy === 'masked') {
    return t('container.detail.config.maskedValue');
  }
  if (displayValue) {
    return displayEnvironmentValue(displayValue, 'plain');
  }
  return displayEnvironmentValue(rawValue, policy);
}

function isMaskedEnvironmentDisplayValue(value: string) {
  return value === '[MASKED]' || value === t('container.detail.config.maskedValue');
}

function isHiddenEnvironmentDisplayValue(value: string) {
  return value === '[HIDDEN]' || value === t('container.detail.config.hiddenValue');
}

function readEnvironmentPolicy(detail: ContainerDetail | null) {
  const value = readString(readUnknownRecord(detail)?.environment_policy);
  return value === 'plain' || value === 'masked' || value === 'hidden' ? value : 'unknown';
}

function containerEnvironmentMaskedCopyEnabled(detail: ContainerDetail | null) {
  const record = readUnknownRecord(detail);
  return record?.environment_masked_copy_enabled === true;
}

function buildRawJsonDisplayValue(detail: SafeContainerDetail | null) {
  if (!detail) {
    return detail;
  }
  const sourceEnvironment = Array.isArray(detail.environment) ? detail.environment : [];
  return {
    ...detail,
    environment: environmentRows.value.map((row, index) => ({
      ...(readUnknownRecord(sourceEnvironment[index]) ?? {}),
      key: row.name,
      value: row.rawJsonValue,
      display_value: row.rawJsonValue,
      masked: row.policy !== 'plain',
      sensitive: row.hasSensitiveValue,
      value_masked: row.policy === 'masked',
      value_hidden: row.policy === 'hidden',
    })),
  };
}

function buildRawJsonCopyValue(detail: SafeContainerDetail | null) {
  if (!detail) {
    return detail;
  }
  if (environmentRows.value.some((row) => row.hasSensitiveValue && row.copyDisabled)) {
    return null;
  }
  const sourceEnvironment = Array.isArray(detail.environment) ? detail.environment : [];
  return {
    ...detail,
    environment: environmentRows.value.map((row, index) => ({
      ...(readUnknownRecord(sourceEnvironment[index]) ?? {}),
      key: row.name,
      value: row.rawJsonCopyValue || row.rawJsonValue,
      copy_value: row.rawJsonCopyValue || undefined,
      display_value: row.rawJsonValue,
      masked: row.policy !== 'plain',
      sensitive: row.hasSensitiveValue,
      value_masked: row.policy === 'masked',
      value_hidden: row.policy === 'hidden',
    })),
  };
}

function normalizeDetail(current: ContainerDetail | null): SafeContainerDetail | null {
  if (!current || typeof current !== 'object') {
    return null;
  }

  return {
    ...current,
    command: Array.isArray(current.command) ? current.command : [],
    entrypoint: Array.isArray(current.entrypoint) ? current.entrypoint : [],
    environment: Array.isArray(current.environment) ? current.environment : [],
    mounts: Array.isArray(current.mounts) ? current.mounts : [],
    names: Array.isArray(current.names) ? current.names : [],
    networks: Array.isArray(current.networks) ? current.networks : [],
    ports: Array.isArray(current.ports) ? current.ports : [],
  };
}

function materializeDetailWithManagedStats(
  current: ContainerDetailRecord | null,
  currentContainerId: string,
): ContainerDetailRecord | null {
  if (!current || !currentContainerId) {
    return current;
  }

  const managed = selectContainerDetailView(currentContainerId);
  if (!managed) {
    return current;
  }

  return {
    ...current,
    resource: managed.resource,
  };
}

function updateCurrentTabTitle(title: LocalizedTitle) {
  if (!hasMeaningfulTitle(title)) {
    return;
  }
  const routePath = route.path;
  const routeFullPath = route.fullPath;
  tabsRouterStore.tabRouterList = tabsRouterStore.tabRouterList.map((tab) =>
    tab.tabKey === routePath || tab.path === routePath || tab.fullPath === routeFullPath ? { ...tab, title } : tab,
  );
}

function hasMeaningfulTitle(title: LocalizedTitle) {
  return Boolean(title[LOCALE.ZH_CN]?.trim() || title[LOCALE.EN_US]?.trim());
}

function buildDetailTitle(name: string): LocalizedTitle {
  const normalizedName = name.trim();
  const baseTitle = localizeRouteTitleKey('container.detail.title');
  if (!normalizedName || normalizedName === baseTitle[LOCALE.ZH_CN] || normalizedName === baseTitle[LOCALE.EN_US]) {
    return baseTitle;
  }

  return {
    [LOCALE.ZH_CN]: `${baseTitle[LOCALE.ZH_CN]} - ${normalizedName}`,
    [LOCALE.EN_US]: `${baseTitle[LOCALE.EN_US]} - ${normalizedName}`,
  };
}

function readNameFromTabTitle(title?: LocalizedTitle) {
  if (!title) {
    return '';
  }

  return extractNameFromTitle(title[LOCALE.ZH_CN]) || extractNameFromTitle(title[LOCALE.EN_US]);
}

function extractNameFromTitle(value?: string) {
  const normalized = value?.trim() ?? '';
  if (!normalized) {
    return '';
  }

  const separator = ' - ';
  const index = normalized.indexOf(separator);
  if (index === -1) {
    return '';
  }

  return normalized.slice(index + separator.length).trim();
}

function readQueryString(value: unknown) {
  const raw = Array.isArray(value) ? value[0] : value;
  return typeof raw === 'string' ? raw.trim() : '';
}

function displayName(row: SafeContainerDetail) {
  return row.name || row.names[0] || shortContainerId(row);
}

function stateLabel(state: ContainerState) {
  return t(`container.list.states.${state}`);
}

function readOrchestratorType(detailRecord?: SafeContainerDetail | null): ContainerOrchestratorType {
  return detailRecord?.orchestrator?.type || 'standalone';
}

function orchestratorLabel(type: ContainerOrchestratorType) {
  return t(`container.list.orchestrators.${type}`);
}

function orchestratorTheme(detailRecord?: SafeContainerDetail | null) {
  const type = readOrchestratorType(detailRecord);
  if (type === 'standalone') return 'success';
  if (type === 'compose') return 'warning';
  if (type === 'unknown') return 'danger';
  return 'default';
}

function orchestratorSummary(detailRecord?: SafeContainerDetail | null) {
  const orchestrator = detailRecord?.orchestrator;
  if (!orchestrator) {
    return orchestratorLabel(readOrchestratorType(detailRecord));
  }

  return (
    orchestrator.service ||
    orchestrator.project ||
    orchestrator.stack ||
    orchestrator.namespace ||
    orchestrator.pod ||
    orchestrator.display_name ||
    t('container.detail.source.summaryUnknown')
  );
}

function readActionLevel(detailRecord?: SafeContainerDetail | null): ContainerActionLevel {
  if (detailRecord?.orchestrator?.action_level) {
    return detailRecord.orchestrator.action_level;
  }

  return detailRecord?.can_start || detailRecord?.can_stop || detailRecord?.can_restart || detailRecord?.can_remove
    ? 'allow'
    : 'readonly';
}

function actionLevelLabel(detailRecord?: SafeContainerDetail | null) {
  return t(`container.detail.source.actionLevels.${readActionLevel(detailRecord)}`);
}

function orchestratorRiskHint(detailRecord?: SafeContainerDetail | null) {
  const level = readActionLevel(detailRecord);
  if (level === 'warn') {
    return t('container.detail.source.riskWarn', { source: orchestratorLabel(readOrchestratorType(detailRecord)) });
  }
  if (level === 'readonly') {
    return t('container.detail.source.riskReadonly', {
      source: orchestratorLabel(readOrchestratorType(detailRecord)),
    });
  }
  return '';
}

function resolveHealthDiagnosis(nextDetail: ContainerDetail) {
  if (nextDetail.state === 'exited' || nextDetail.state === 'dead') {
    return {
      description: t('container.detail.health.description.notRunning'),
      label: t('container.detail.health.diagnosis.notRunning'),
      theme: 'default' as const,
    };
  }
  if (nextDetail.state === 'running' && nextDetail.health === 'healthy') {
    return {
      description: t('container.detail.health.description.healthy'),
      label: t('container.detail.health.diagnosis.healthy'),
      theme: 'success' as const,
    };
  }
  if (nextDetail.state === 'running' && nextDetail.health === 'unhealthy') {
    return {
      description: t('container.detail.health.description.unhealthy'),
      label: t('container.detail.health.diagnosis.unhealthy'),
      theme: 'danger' as const,
    };
  }
  if (nextDetail.state === 'running' && (!nextDetail.health || nextDetail.health === 'none')) {
    return {
      description: t('container.detail.health.description.noHealthcheck'),
      label: t('container.detail.health.diagnosis.noHealthcheck'),
      theme: 'warning' as const,
    };
  }
  if (nextDetail.health === 'starting') {
    return {
      description: t('container.detail.health.description.starting'),
      label: t('container.detail.health.diagnosis.starting'),
      theme: 'warning' as const,
    };
  }
  return {
    description: t('container.detail.health.description.unavailable'),
    label: healthLabel(nextDetail.health),
    theme: healthTheme(nextDetail.health),
  };
}

function resolveHealthcheckDetails(healthcheck?: ContainerHealthcheck) {
  if (!healthcheck?.configured) {
    return {
      command: '-',
      configured: false,
      exitCode: '-',
      hasFailure: false,
      lastCheckedAt: '',
      output: '-',
      relativeCheckedAt: '',
      statusLabel: t('container.detail.health.healthcheckStatus.unconfigured'),
      theme: 'default' as HealthStatusTheme,
    };
  }
  return {
    command: joinList(healthcheck.command),
    configured: true,
    exitCode: formatNullableNumber(healthcheck.exit_code),
    hasFailure:
      healthcheck.status === 'unhealthy' || (typeof healthcheck.exit_code === 'number' && healthcheck.exit_code !== 0),
    lastCheckedAt: formatTime(healthcheck.checked_at),
    output: healthcheck.output || healthcheck.failure_message || t('container.detail.health.noOutput'),
    relativeCheckedAt: healthcheck.checked_at
      ? t('container.detail.health.updatedFromHealthcheck')
      : t('container.detail.health.noRecentCheck'),
    statusLabel: healthcheckStatusLabel(healthcheck.status),
    theme: healthTheme(healthcheck.status),
  };
}

function resolveRuntimeStability(current?: ContainerDetail | null) {
  if (current?.oom_killed) {
    return {
      label: t('container.detail.health.stabilityStatus.oom'),
      theme: 'danger' as const,
    };
  }
  if (typeof current?.last_exit_code === 'number' && current.last_exit_code !== 0) {
    return {
      label: t('container.detail.health.stabilityStatus.exit'),
      theme: 'danger' as const,
    };
  }
  if (typeof current?.restart_count === 'number' && current.restart_count > 0) {
    return {
      label: t('container.detail.health.stabilityStatus.restart'),
      theme: 'warning' as const,
    };
  }
  return {
    label: t('container.detail.health.stabilityStatus.stable'),
    theme: 'success' as const,
  };
}

function healthcheckStatusLabel(status: ContainerHealth) {
  if (status === 'healthy') return t('container.detail.health.healthcheckStatus.passed');
  if (status === 'unhealthy') return t('container.detail.health.healthcheckStatus.failed');
  if (status === 'starting') return t('container.detail.health.healthcheckStatus.starting');
  if (status === 'none') return t('container.detail.health.healthcheckStatus.unconfigured');
  return t('container.detail.health.healthcheckStatus.unavailable');
}

function restartCountLabel(value?: number | null) {
  if (typeof value !== 'number') {
    return '-';
  }
  return t('container.detail.health.restartCountValue', { count: value });
}

function restartSummaryLabel(value?: number | null) {
  if (typeof value !== 'number') {
    return t('container.detail.health.restartUnknown');
  }
  return value > 0
    ? t('container.detail.health.restartAbnormal', { count: value })
    : t('container.detail.health.restartNormal');
}

function healthLabel(health?: ContainerHealth | null) {
  return t(`container.list.health.${health || 'none'}`);
}

function healthTheme(health?: ContainerHealth | null): HealthStatusTheme {
  if (health === 'healthy') return 'success';
  if (health === 'unhealthy') return 'danger';
  if (health === 'starting') return 'warning';
  return 'default';
}

function stateTheme(state: ContainerState) {
  if (state === 'running') return 'success';
  if (state === 'created' || state === 'paused' || state === 'restarting') return 'warning';
  if (state === 'dead') return 'danger';
  return 'default';
}

function formatTime(value?: string | null) {
  return formatLocaleDateTime(value, locale);
}

function joinList(values?: string[]) {
  return values?.length ? values.join(' ') : '-';
}

function uptimeLabel(value?: string | null) {
  if (!value) {
    return '-';
  }
  const startedAt = new Date(value);
  if (Number.isNaN(startedAt.getTime())) {
    return '-';
  }
  const elapsedMs = Date.now() - startedAt.getTime();
  if (elapsedMs < 0) {
    return '-';
  }
  const totalMinutes = Math.floor(elapsedMs / 60000);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (hours > 0) {
    return t('container.detail.health.uptimeHoursMinutes', { hours, minutes });
  }
  return t('container.detail.health.uptimeMinutes', { minutes });
}

function formatNullableNumber(value?: number | null) {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return '-';
  }
  const currentLocale = typeof locale === 'string' ? locale : locale.value;
  return new Intl.NumberFormat(currentLocale).format(value);
}

function formatNullableBoolean(value?: boolean | null) {
  if (typeof value !== 'boolean') {
    return '-';
  }
  return value ? t('container.detail.health.boolean.yes') : t('container.detail.health.boolean.no');
}

function runtimeLabel(nextDetail: ContainerDetail) {
  return nextDetail.runtime || nextDetail.runtime_info?.runtime || '-';
}

function shortContainerId(nextDetail: ContainerDetail) {
  return shortIdentifier(nextDetail.id, nextDetail.short_id, 12);
}

function networkSummary(nextDetail: ContainerDetail) {
  return nextDetail.network_summary || primaryNetworkName(nextDetail);
}

function primaryNetworkName(nextDetail: ContainerDetail) {
  return nextDetail.networks[0]?.name || '-';
}

function createNetworkField(
  key: string,
  label: string,
  value = '',
  copyable = false,
  testId?: string,
  technical = false,
): NetworkField {
  const displayValue = technical ? displayTechnicalIdentifier(value) : displayOptionalValue(value);
  return {
    copyValue: displayValue === t('container.detail.network.noData') ? '' : value.trim(),
    copyable: copyable && Boolean(value.trim()),
    displayValue,
    key,
    label,
    testId,
    technical,
  };
}

function displayOptionalValue(value?: string | number | null) {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value);
  }
  const normalized = typeof value === 'string' ? value.trim() : '';
  return normalized || t('container.detail.network.noData');
}

function displayTechnicalIdentifier(value?: string | null) {
  const normalized = value?.trim() ?? '';
  if (!normalized) {
    return t('container.detail.network.noData');
  }
  return `${normalized.slice(0, 12)}...${normalized.slice(-6)}`;
}

function formatPortNumber(value?: number | null) {
  return typeof value === 'number' && Number.isFinite(value) ? String(value) : t('container.detail.network.noData');
}

function hasPublishedHostBinding(port: ContainerDetail['ports'][number]) {
  return typeof port.public_port === 'number' && Number.isFinite(port.public_port);
}

function buildPortMappingRows(ports: ContainerDetail['ports']): PortMappingRow[] {
  const groups = new Map<
    string,
    {
      hasHostBinding: boolean;
      listenAddresses: string[];
      privatePort: string;
      protocol: string;
      publicPort: string;
      rawBindings: string[];
    }
  >();

  for (const port of ports) {
    const privatePort = formatPortNumber(port.private_port);
    const publicPort = formatPortNumber(port.public_port);
    const protocol = port.type || '-';
    const hasHostBinding = hasPublishedHostBinding(port);
    const groupKey = `${privatePort}:${publicPort}:${protocol}`;
    const existing = groups.get(groupKey) ?? {
      hasHostBinding,
      listenAddresses: [],
      privatePort,
      protocol,
      publicPort,
      rawBindings: [],
    };

    if (hasHostBinding) {
      const listenAddress = port.ip?.trim() || t('container.detail.network.allInterfaces');
      if (!existing.listenAddresses.includes(listenAddress)) {
        existing.listenAddresses.push(listenAddress);
      }
    }
    existing.hasHostBinding = existing.hasHostBinding || hasHostBinding;
    existing.rawBindings.push(portMappingRawLabel(port));
    groups.set(groupKey, existing);
  }

  return Array.from(groups.values()).map((group) => {
    const listenAddress = group.hasHostBinding
      ? formatListenAddresses(group.listenAddresses)
      : t('container.detail.network.notPublished');
    const mapping = group.hasHostBinding
      ? t('container.detail.network.publishedMapping', {
          hostPort: group.publicPort,
          privatePort: group.privatePort,
          protocol: group.protocol,
        })
      : t('container.detail.network.internalOnlyFull');

    return {
      copyValue: group.rawBindings.join('\n'),
      description: group.hasHostBinding
        ? t('container.detail.network.publishedToHost')
        : t('container.detail.network.internalOnly'),
      hasHostBinding: group.hasHostBinding,
      key: `${group.privatePort}:${group.publicPort}:${group.protocol}:${listenAddress}`,
      listenAddresses: group.listenAddresses,
      listenAddress,
      mapping,
      privatePort: group.privatePort,
      protocol: group.protocol,
      publicPort: group.publicPort,
    };
  });
}

function formatListenAddresses(addresses: string[]) {
  const unique = addresses.filter(Boolean);
  if (!unique.length) {
    return t('container.detail.network.allInterfaces');
  }
  if (unique.includes('0.0.0.0') && unique.includes('::') && unique.length === 2) {
    return '0.0.0.0, ::';
  }
  return unique.join(', ');
}

function portMappingRawLabel(port: ContainerDetail['ports'][number]) {
  const privatePort = formatPortNumber(port.private_port);
  const containerEndpoint = `${privatePort}/${port.type || '-'}`;
  if (!hasPublishedHostBinding(port)) {
    return `${containerEndpoint} / ${t('container.detail.network.notPublished')}`;
  }
  const publicPort = formatPortNumber(port.public_port);
  const listenAddress = port.ip?.trim();
  const hostPart = listenAddress ? `${listenAddress}:${publicPort}` : publicPort;
  return `${hostPart}->${containerEndpoint}`;
}

function buildMountCards(mounts: ContainerMount[]): MountCard[] {
  return stableSortMounts(mounts).map((mount, index) => {
    const usage = readMountUsage(mount);
    const status = usage.status || 'not_measured';
    const message = usage.message || mountUsageMessage(status);
    const key = mountKey(mount, index);
    const refreshing = refreshingMountKeys.value.has(key) || status === 'pending';
    return {
      accessLabel: mountAccessLabel(mount),
      accessTheme: mount.read_only ? 'warning' : 'success',
      destination: mount.destination || '',
      destinationDisplay: displayMountPath(mount.destination),
      index,
      key,
      mountId: mount.mount_id,
      measuredAt: formatMountMeasuredAt(usage.measuredAt),
      message,
      mode: mount.mode || '-',
      modeLabel: mountModeLabel(mount),
      raw: mount,
      sharedHint: usage.sharedHint,
      source: mount.source || mount.name || '',
      sourceDisplay: displayMountPath(mount.source || mount.name || ''),
      typeLabel: mountTypeLabel(mount.type),
      typeTheme: mountTypeTheme(mount.type),
      usageLabel: mountUsageStatusLabel(status),
      usageRefreshDisabled: status === 'unsupported',
      usageRefreshLabel: mountUsageRefreshLabel(status),
      usageRefreshTooltip: mountUsageRefreshTooltip(status),
      usageRefreshing: refreshing,
      usageSize: usage.sizeBytes === null ? mountUsageSizeFallback(status) : formatBytes(usage.sizeBytes),
      usageTheme: mountUsageTheme(status),
      usageTone: mountUsageTone(status),
    };
  });
}

function stableSortMounts(mounts: ContainerMount[]) {
  return [...mounts].sort(compareMountsForDisplay);
}

function compareMountsForDisplay(left: ContainerMount, right: ContainerMount) {
  return (
    compareStableString(left.destination, right.destination) ||
    compareStableString(left.source || left.name, right.source || right.name) ||
    compareStableString(left.type, right.type) ||
    stableMountIdentity(left).localeCompare(stableMountIdentity(right))
  );
}

function compareStableString(left?: string | null, right?: string | null) {
  return (left?.trim() ?? '').localeCompare(right?.trim() ?? '');
}

function readMountUsage(mount: ContainerMount) {
  const usageRecord = readUnknownRecord(mount.usage);
  return {
    measuredAt: readRawString(usageRecord?.measured_at),
    message: readRawString(usageRecord?.message),
    sharedHint: readRawString(usageRecord?.shared_hint),
    sizeBytes: readNullableNumber(usageRecord?.size_bytes),
    status: readString(usageRecord?.status),
  };
}

function readNullableNumber(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

function mountKey(mount: ContainerMount, index: number) {
  return stableMountIdentity(mount) || `mount-${index}`;
}

function stableMountIdentity(mount: ContainerMount) {
  const mountID = mount.mount_id?.trim();
  if (mountID) {
    return mountID;
  }
  const parts = [mount.destination || '', mount.source || mount.name || '', mount.type || ''].map((part) =>
    part.trim(),
  );
  if (!parts.some(Boolean)) {
    return '';
  }
  return parts.join('::').toLowerCase();
}

function mountTypeLabel(type?: string | null) {
  const normalized = normalizeMountType(type);
  return t(`container.detail.storage.typeLabels.${normalized}`);
}

function mountTypeTheme(type?: string | null): MountTagTheme {
  const normalized = normalizeMountType(type);
  if (normalized === 'bind') return 'primary';
  if (normalized === 'volume') return 'success';
  if (normalized === 'tmpfs') return 'warning';
  return 'default';
}

function normalizeMountType(type?: string | null) {
  const normalized = type?.trim().toLowerCase();
  if (normalized === 'bind' || normalized === 'volume' || normalized === 'tmpfs') {
    return normalized;
  }
  return 'unknown';
}

function mountAccessLabel(mount: ContainerMount) {
  return mount.read_only
    ? t('container.detail.storage.accessLabels.readOnly')
    : t('container.detail.storage.accessLabels.readWrite');
}

function mountModeLabel(mount: ContainerMount) {
  const accessLabel = mountAccessLabel(mount);
  const mode = mount.mode || (mount.read_only ? 'ro' : 'rw');
  return `${accessLabel} ${mode}`;
}

function mountUsageStatusLabel(status: string) {
  if (
    status === 'measured' ||
    status === 'not_measured' ||
    status === 'pending' ||
    status === 'unsupported' ||
    status === 'permission_denied' ||
    status === 'not_found' ||
    status === 'timeout' ||
    status === 'error'
  ) {
    return t(`container.detail.storage.usageStatus.${status}`);
  }
  return t('container.detail.storage.usageStatus.not_measured');
}

function mountUsageTheme(status: string): MountTagTheme {
  if (status === 'measured') return 'success';
  if (status === 'unsupported' || status === 'pending' || status === 'timeout') return 'warning';
  if (status === 'permission_denied' || status === 'not_found' || status === 'error') return 'danger';
  return 'default';
}

function mountUsageTone(status: string): MountCardUsageTone {
  if (status === 'measured') return 'success';
  if (status === 'unsupported' || status === 'pending' || status === 'timeout') return 'warning';
  return 'weak';
}

function mountUsageMessage(status: string) {
  if (status === 'pending') return t('container.detail.storage.pendingMessage');
  if (status === 'unsupported') return t('container.detail.storage.unsupportedMessage');
  if (status === 'permission_denied' || status === 'not_found' || status === 'timeout' || status === 'error') {
    return t('container.detail.storage.errorMessage');
  }
  if (status === 'not_measured') return t('container.detail.storage.notMeasuredMessage');
  return '';
}

function mountUsageSizeFallback(status: string) {
  if (status === 'pending') return t('container.detail.storage.pendingSize');
  if (status === 'measured') return '0 B';
  return '-';
}

function mountUsageRefreshLabel(status: string) {
  if (status === 'pending') return t('container.detail.storage.refreshPending');
  if (status === 'permission_denied' || status === 'not_found' || status === 'timeout' || status === 'error') {
    return t('container.detail.storage.retryUsage');
  }
  return t('container.detail.storage.refreshMount');
}

function mountUsageRefreshTooltip(status: string) {
  if (status === 'unsupported') return t('container.detail.storage.unsupportedTooltip');
  return t('container.detail.storage.refreshMountTooltip');
}

function formatMountMeasuredAt(value: string) {
  return value ? formatTime(value) : '';
}

function displayMountPath(value?: string | null) {
  const normalized = value?.trim() ?? '';
  if (!normalized) {
    return '-';
  }
  return abbreviateMiddle(normalized, 42);
}

function readContainerHostname(current: SafeContainerDetail) {
  const record = readUnknownRecord(current);
  const hostname = readString(record?.hostname);
  return hostname || displayName(current);
}

function readStringListFromRecord(current: SafeContainerDetail, keys: string[]) {
  const record = readUnknownRecord(current);
  if (!record) {
    return [];
  }

  for (const key of keys) {
    const value = record[key];
    if (Array.isArray(value)) {
      return value.map((item) => readString(item)).filter(Boolean);
    }
    if (typeof value === 'string' && value.trim()) {
      return value
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean);
    }
  }
  return [];
}

function resourceStatus(nextDetail: ContainerDetail) {
  const resource = nextDetail.resource;
  const collectedAt = formatTime(resource?.collected_at);
  if (resource?.stats_available || resource?.available) {
    return {
      collectedAt,
      description: collectedAt,
      theme: 'success' as const,
      value: t('container.detail.resources.available'),
    };
  }
  if (resource?.stats_error_message || resource?.stats_error_key || resource?.unavailable_reason) {
    return {
      collectedAt: '-',
      description: resource.stats_error_message || resource.stats_error_key || resource.unavailable_reason || '-',
      theme: 'warning' as const,
      value: t('container.detail.resources.unavailable'),
    };
  }
  return emptyResourceStatus();
}

function emptyResourceStatus() {
  return {
    collectedAt: '-',
    description: '-',
    theme: 'default' as const,
    value: t('container.detail.resources.noData'),
  };
}

function metricRows(definitions: ResourceMetricDefinition[]): ResourceDetailRow[] {
  return definitions.map(([key, label, format]) => ({
    key,
    label,
    type: 'text',
    value: readMetricText(key, format),
  }));
}

function readMetricText(key: ResourceMetricKey, format: ResourceMetricFormat) {
  const value = readResourceMetric(key);
  if (format === 'bytes') {
    return formatResourceValue(formatBytes(value));
  }
  if (format === 'percent') {
    return formatResourceValue(formatPercent(value));
  }
  if (format === 'number') {
    return formatResourceValue(formatNumberMetric(value));
  }
  return formatResourceValue(readRawString(value));
}

function readCpuSystemTimeText() {
  return formatResourceValue(formatNanosecondsAsDuration(readResourceMetric('system_cpu_usage'), '-', locale.value));
}

function readCpuCountText() {
  return formatResourceValue(formatCpuCountText(readResourceMetric('online_cpus'), locale.value));
}

function readResourceMetric(key: ResourceMetricKey) {
  return safeDetail.value?.resource?.[key];
}

function formatNumberMetric(value: unknown) {
  if (typeof value === 'number' && Number.isFinite(value)) {
    const currentLocale = typeof locale === 'string' ? locale : locale.value;
    return new Intl.NumberFormat(currentLocale).format(value);
  }
  if (typeof value === 'string' && value.trim()) {
    return value.trim();
  }
  return '-';
}

function formatResourceValue(value: string) {
  return value && value !== '-' ? value : notCollectedLabel();
}

function notCollectedLabel() {
  return t('container.detail.resources.notCollected');
}

function memorySummary(nextDetail: ContainerDetail) {
  const resource = nextDetail.resource;
  const usage = formatBytes(resource?.memory_usage_bytes);
  const limit = formatBytes(resource?.memory_limit_bytes);
  if (usage === '-' && limit === '-') {
    return '-';
  }
  return `${usage} / ${limit}`;
}

function shortIdentifier(value?: string | null, preferred?: string | null, maxLength = 28) {
  const normalized = value?.trim();
  if (!normalized) return '-';
  if (preferred?.trim()) return preferred.trim();
  if (normalized.length <= maxLength) return normalized;
  if (maxLength <= 12) return normalized.slice(0, maxLength);
  return `${normalized.slice(0, 18)}...${normalized.slice(-10)}`;
}

function readableImageId(value?: string | null) {
  const normalized = value?.trim();
  if (!normalized) return '-';
  return normalized.startsWith('sha256:') ? normalized.slice('sha256:'.length) : normalized;
}

function portLabel(port: ContainerDetail['ports'][number]) {
  const privatePort = port.private_port ? `${port.private_port}` : '-';
  const publicPort = port.public_port ? `${port.public_port}` : '';
  return publicPort ? `${publicPort}:${privatePort}/${port.type}` : `${privatePort}/${port.type}`;
}
</script>
<style scoped lang="less">
.container-detail-page {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-height: 0;
  min-width: 0;
}

.container-detail-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-height: 420px;
  min-width: 0;
}

.container-detail-state {
  align-items: center;
  display: flex;
  flex: 1;
  justify-content: center;
  min-height: 420px;
  min-width: 0;
}

.container-detail-state :deep(.t-loading__parent) {
  width: min(100%, 760px);
}

.container-detail-state-alert {
  flex: 0 0 auto;
}

.container-detail-summary {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.container-detail-summary-card {
  background: color-mix(in srgb, var(--td-bg-color-container) 92%, transparent);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 64%, transparent);
  height: 100%;
  min-width: 0;
}

.container-detail-summary-card :deep(.t-card__body) {
  height: calc(100% - var(--td-comp-size-xxxl));
}

.container-detail-metric,
.container-detail-section,
.container-detail-subsection {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.container-detail-summary-list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
  height: 100%;
  min-width: 0;
}

.container-detail-kv {
  display: grid;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-detail-kv--inline {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
}

.container-detail-kv > span {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
}

.container-detail-kv strong,
.container-detail-kv code {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-detail-kv code,
.container-detail-header-id {
  font-family: var(
    --td-font-family-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    monospace
  );
}

.container-detail-header-id {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  font: var(--td-font-body-small);
  min-height: var(--td-comp-size-xs);
}

.container-detail-header-meta {
  display: grid;
  gap: var(--graft-density-gap-6);
  max-width: min(100%, 680px);
  min-width: 0;
}

.container-detail-header-meta__tags {
  min-width: 0;
}

.container-detail-header-meta__updated-at {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.container-detail-header-meta__tags :deep(.t-space-item) {
  min-width: 0;
}

.container-detail-header-meta__tags :deep(.t-tag) {
  max-width: 100%;
}

.container-detail-summary__resource,
.container-detail-resource-grid {
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  min-width: 0;
}

.container-detail-resource-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.container-detail-summary__resource {
  grid-template-columns: 1fr;
}

.container-detail-metric strong,
.container-detail-subsection h3,
.container-detail-overview-group h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0;
}

.container-detail-metric span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  overflow-wrap: anywhere;
}

.container-detail-tag-row,
.container-detail-port-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.container-detail-resource-meter {
  border-radius: var(--td-radius-medium);
  display: flex;
  gap: var(--graft-density-gap-10);
  min-width: 0;
  padding: var(--graft-density-gap-8);
  transition:
    background-color 180ms ease,
    box-shadow 180ms ease,
    transform 180ms ease;
}

.container-detail-resource-meter--cpu {
  align-items: center;
  justify-content: space-between;
}

.container-detail-resource-meter--memory {
  flex-direction: column;
  justify-content: center;
}

.container-detail-resource-meter__content {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-detail-resource-meter__content span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.container-detail-resource-meter__content strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  min-width: 0;
  overflow-wrap: anywhere;
}

.container-detail-resource-meter__content em {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  font-style: normal;
}

.container-detail-resource-meter--memory :deep(.t-progress) {
  width: 100%;
}

.container-detail-resource-meter.container-metric-change--up,
.container-resource-dashboard-panel.container-metric-change--up {
  background: color-mix(in srgb, var(--td-warning-color-1) 74%, transparent);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--td-warning-color-5) 34%, transparent);
  transform: translateY(-1px);
}

.container-detail-resource-meter.container-metric-change--down,
.container-resource-dashboard-panel.container-metric-change--down {
  background: color-mix(in srgb, var(--td-success-color-1) 80%, transparent);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--td-success-color-5) 30%, transparent);
}

.container-detail-port-chip {
  font-family: var(
    --td-font-family-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    monospace
  );
  max-width: 100%;
}

.container-detail-tabs-card {
  min-height: 0;
  min-width: 0;
}

.container-detail-tabs-card :deep(.t-card__body) {
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 0;
}

.container-detail-realtime-row {
  align-items: center;
  display: flex;
  justify-content: flex-end;
  margin: var(--graft-density-gap-8) 0 var(--graft-density-gap-10);
  min-height: 40px;
  min-width: 0;
  padding-inline: var(--graft-density-gap-12);
}

.container-detail-realtime-bar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6) var(--graft-density-gap-8);
  justify-content: flex-end;
  min-width: 0;
}

.container-detail-realtime-bar__label {
  color: var(--td-text-color-primary);
  font-size: var(--td-font-size-body-small);
  font-weight: 600;
  line-height: 20px;
}

.container-detail-realtime-bar__hint {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-small);
  line-height: 20px;
  max-width: min(560px, 100%);
  min-width: 0;
  text-align: right;
}

.container-detail-realtime-bar__actions {
  align-items: center;
  display: inline-flex;
  flex-shrink: 0;
  gap: var(--graft-density-gap-8);
}

.container-detail-tabs-card :deep(.t-tabs__content) {
  min-height: 0;
  padding-top: var(--graft-density-gap-12);
}

.container-detail-tabs-card :deep(.t-tabs) {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.container-detail-tabs-card :deep(.t-tabs__panel) {
  min-height: 0;
}

.container-detail-logs-header {
  align-items: center;
  border-bottom: 1px solid var(--td-component-stroke);
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  margin: 0 var(--graft-density-gap-16) var(--graft-density-gap-12);
  padding-bottom: var(--graft-density-gap-10);
}

.container-detail-logs-header__title-block {
  min-width: 0;
}

.container-detail-logs-header__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
}

.container-detail-tab-body {
  height: auto;
  min-height: 0;
}

.container-detail-tab-body--viewer {
  min-height: 0;
}

.container-detail-section {
  padding: 0;
}

.container-detail-section--overview {
  min-height: 0;
  padding: 0 var(--graft-density-gap-16) var(--graft-density-gap-16);
}

.container-detail-section--health {
  padding: 0 var(--graft-density-gap-16) var(--graft-density-gap-16);
}

.container-detail-section--shell {
  min-height: 0;
}

.container-detail-section--config {
  gap: var(--graft-density-gap-16);
  padding: 0 var(--graft-density-gap-16) var(--graft-density-gap-16);
}

.container-detail-section--network {
  gap: var(--graft-density-gap-16);
  padding: 0 var(--graft-density-gap-16) var(--graft-density-gap-16);
}

.container-detail-section--storage {
  padding: 0 var(--graft-density-gap-16) var(--graft-density-gap-16);
}

.container-detail-section--raw {
  min-height: var(--container-detail-tab-body-min-height);
  padding: 0;
}

.container-detail-empty-state {
  align-items: center;
  display: flex;
  flex: 1 1 auto;
  justify-content: center;
  min-height: var(--container-detail-tab-body-min-height, 100%);
  min-width: 0;
  padding: var(--graft-density-gap-24) var(--graft-density-gap-16);
  text-align: center;
}

.container-detail-empty-state :deep(.t-empty) {
  max-width: 420px;
}

.container-detail-empty-state :deep(.t-empty__description) {
  overflow-wrap: anywhere;
}

.container-detail-empty-state--compact {
  padding-block: var(--graft-density-gap-16);
}

.container-detail-empty-state--inline {
  min-height: 180px;
  padding-block: var(--graft-density-gap-16);
}

.container-mount-card-grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  min-width: 0;
}

.container-mount-card {
  background: var(--td-bg-color-container);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
  padding: var(--graft-density-gap-16);
}

.container-mount-card__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-10);
  justify-content: space-between;
  min-width: 0;
}

.container-mount-card__destination {
  color: var(--td-text-color-primary);
  flex: 1 1 auto;
  font: var(--td-font-title-small);
  font-weight: 600;
  max-width: min(100%, 520px);
  min-width: 0;
}

.container-mount-card__actions {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6);
  justify-content: flex-end;
  min-width: 0;
}

.container-mount-card__body {
  display: grid;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.container-mount-info {
  display: grid;
  gap: var(--graft-density-gap-10);
  min-width: 0;
}

.container-mount-info h3 {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  font-weight: 600;
  margin: 0;
}

.container-mount-field {
  align-items: center;
  column-gap: var(--graft-density-gap-12);
  display: grid;
  grid-template-columns: minmax(72px, 96px) minmax(0, 1fr);
  min-width: 0;
}

.container-mount-field > span {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
}

.container-mount-field strong,
.container-mount-field code {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-mount-field code {
  font-family: var(
    --td-font-family-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    monospace
  );
}

.container-mount-usage {
  background: color-mix(in srgb, var(--td-bg-color-page) 72%, var(--td-bg-color-container));
  border: 0;
  border-radius: var(--td-radius-default);
  border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  display: grid;
  gap: var(--graft-density-gap-8);
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.container-mount-usage--success {
  background: color-mix(in srgb, var(--td-success-color-1) 28%, var(--td-bg-color-container));
  border-color: color-mix(in srgb, var(--td-success-color) 20%, var(--td-component-stroke));
}

.container-mount-usage--warning {
  background: color-mix(in srgb, var(--td-warning-color-1) 28%, var(--td-bg-color-container));
  border-color: color-mix(in srgb, var(--td-warning-color) 22%, var(--td-component-stroke));
}

.container-mount-usage__header {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
  min-width: 0;
}

.container-mount-usage__header > :deep(.t-button) {
  flex: 0 0 auto;
  min-width: 92px;
}

.container-mount-usage__header > span,
.container-mount-usage__time,
.container-mount-usage__message {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  min-width: 0;
  overflow-wrap: anywhere;
}

.container-mount-usage strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.container-network-panel {
  background: var(--td-bg-color-container);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
  padding: var(--graft-density-gap-14) var(--graft-density-gap-16);
}

.container-network-panel__header {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
  min-width: 0;
}

.container-network-panel__header h3,
.container-network-connection-card__header h4 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-weight: 600;
  margin: 0;
}

.container-network-connection-card {
  background: color-mix(in srgb, var(--td-bg-color-container) 92%, var(--td-bg-color-page));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 68%, transparent);
  border-radius: var(--td-radius-default);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.container-network-connection-card__header {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
  min-width: 0;
}

.container-network-connection-card__label,
.container-port-mapping-card__label {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
}

.container-network-connection-card__name {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-weight: 600;
  min-width: 0;
}

.container-network-field-grid {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  max-width: 880px;
  min-width: 0;
}

.container-network-field-grid--metadata {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  max-width: 980px;
}

.container-network-field {
  display: grid;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-network-field > span {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
}

.container-network-field strong,
.container-network-code {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-network-field--technical strong,
.container-network-field--technical :deep(.container-detail-copyable-value__text) {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-small);
}

.container-network-code {
  display: inline-block;
  font-family: var(
    --td-font-family-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    monospace
  );
  max-width: 100%;
}

.container-port-mapping-card {
  align-items: center;
  background: color-mix(in srgb, var(--td-bg-color-container) 94%, var(--td-bg-color-page));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 62%, transparent);
  border-radius: var(--td-radius-default);
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-10) var(--graft-density-gap-16);
  justify-content: space-between;
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.container-port-mapping-card--internal {
  background: color-mix(in srgb, var(--td-warning-color-1) 24%, var(--td-bg-color-container));
  border-color: color-mix(in srgb, var(--td-warning-color) 22%, var(--td-component-stroke));
}

.container-port-mapping-card__main {
  align-items: center;
  display: flex;
  flex: 1 1 320px;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.container-port-mapping-card__description {
  color: var(--td-text-color-secondary);
  flex: 0 1 auto;
  font: var(--td-font-body-small);
  min-width: 0;
}

.container-port-mapping-card__listen {
  align-items: center;
  display: inline-flex;
  flex: 0 1 auto;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.container-port-mapping-card__listen > span {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
}

.container-network-metadata-empty {
  align-items: flex-start;
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  min-height: auto;
  padding: 0;
  text-align: left;
}

.container-network-metadata-empty :deep(.t-empty__image) {
  display: none;
}

.container-config-section {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
  min-width: 0;
}

.container-config-section h3,
.container-config-section__header h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-weight: 600;
  margin: 0;
}

.container-config-section__header {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
  min-width: 0;
}

.container-config-section__header > span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.container-runtime-config-card {
  background: var(--td-bg-color-container);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  min-width: 0;
}

.container-runtime-config-card :deep(.t-card__body) {
  padding: var(--graft-density-gap-14) var(--graft-density-gap-16);
}

.container-runtime-config-list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-runtime-config-row {
  align-items: center;
  border-radius: var(--td-radius-default);
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: 108px minmax(0, 1fr) var(--td-comp-size-xs);
  min-height: var(--td-comp-size-xl);
  min-width: 0;
  padding: var(--graft-density-gap-6) var(--graft-density-gap-8);
}

.container-runtime-config-row:hover {
  background: var(--td-bg-color-container-hover);
}

.container-runtime-config-row__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  white-space: nowrap;
}

.container-runtime-config-row__value,
.container-env-cell {
  color: var(--td-text-color-primary);
  display: block;
  font-family: var(
    --td-font-family-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    monospace
  );
  font-size: var(--td-font-size-body-small);
  line-height: 20px;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-runtime-config-row__copy {
  opacity: 0.56;
}

.container-runtime-config-row:hover .container-runtime-config-row__copy,
.container-runtime-config-row__copy:focus-visible {
  opacity: 1;
}

.container-env-toolbar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
  min-width: 0;
}

.container-env-toolbar__search {
  flex: 1 1 280px;
  min-width: 220px;
}

.container-env-toolbar__policy {
  flex: 0 0 176px;
}

.container-env-table {
  min-width: 0;
  width: 100%;
}

.container-env-table :deep(.t-table__content) {
  max-width: 100%;
}

.container-env-table :deep(.t-table--layout-fixed) {
  width: 100%;
}

.container-env-table :deep(.t-table__th-cell-inner) {
  color: var(--td-text-color-secondary);
  font-weight: 500;
}

.container-env-table :deep(.t-table__body tr .container-env-copy-button) {
  opacity: 0.42;
}

.container-env-table :deep(.t-table__body tr:hover .container-env-copy-button),
.container-env-table :deep(.container-env-copy-button:focus-visible) {
  opacity: 1;
}

.container-env-cell--value {
  color: var(--td-text-color-secondary);
}

.container-health-diagnostics {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
  min-width: 0;
}

.container-health-diagnostics__header h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-weight: 600;
  margin: 0;
}

.container-health-summary-grid {
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: repeat(3, minmax(0, 1fr));
  min-width: 0;
}

.container-health-summary-card,
.container-health-info-card {
  background: var(--td-bg-color-container);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 64%, transparent);
  min-width: 0;
}

.container-health-summary-card :deep(.t-card__body) {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-height: 86px;
  padding: var(--graft-density-gap-12);
}

.container-health-summary-card__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.container-health-summary-card__value {
  align-items: center;
  color: var(--td-text-color-primary);
  display: flex;
  font: var(--td-font-title-small);
  font-weight: 600;
  min-height: 24px;
  min-width: 0;
  overflow-wrap: anywhere;
}

.container-health-summary-card p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  line-height: 20px;
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

.container-health-info-card :deep(.t-card__body) {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
  min-width: 0;
  padding: var(--graft-density-gap-14) var(--graft-density-gap-16);
}

.container-health-info-card :deep(.t-card__header) {
  align-items: center;
  min-height: var(--td-comp-size-xxl);
  padding: 0 var(--graft-density-gap-16);
}

.container-health-info-card :deep(.t-descriptions__body) {
  table-layout: auto;
}

.container-health-card-actions {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  flex-wrap: wrap;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-8);
  justify-content: flex-end;
  min-width: 0;
}

.container-health-inline-descriptions,
.container-health-stability-grid {
  max-width: 920px;
}

.container-health-inline-descriptions :deep(.t-descriptions__label),
.container-health-stability-grid :deep(.t-descriptions__label) {
  color: var(--td-text-color-placeholder);
  white-space: nowrap;
}

.container-health-block {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.container-health-block__label {
  align-items: center;
  color: var(--td-text-color-placeholder);
  display: flex;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
  min-height: var(--td-comp-size-xs);
  min-width: 0;
}

.container-health-code,
.container-health-output {
  background: color-mix(in srgb, var(--td-bg-color-page) 82%, var(--td-bg-color-container));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 62%, transparent);
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-primary);
  display: block;
  font-family: var(
    --td-font-family-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    monospace
  );
  font-size: var(--td-font-size-body-small);
  font-weight: 400;
  line-height: 20px;
  margin: 0;
  max-width: 100%;
  min-width: 0;
  overflow: auto;
  padding: var(--graft-density-gap-8) var(--graft-density-gap-10);
}

.container-health-code {
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.container-health-output {
  max-height: 144px;
  white-space: pre-wrap;
}

.container-health-output--error {
  background: color-mix(in srgb, var(--td-error-color-1) 36%, var(--td-bg-color-container));
  border-color: color-mix(in srgb, var(--td-error-color) 34%, var(--td-component-stroke));
}

.container-health-last-check {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.container-health-empty {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.container-detail-resource-status-card {
  align-items: center;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
  padding: var(--graft-density-gap-14);
}

.container-detail-resource-status-card__content {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.container-detail-resource-status-card__content strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  min-width: 0;
  overflow-wrap: anywhere;
}

.container-detail-resource-status-card__content span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  min-width: 0;
  overflow-wrap: anywhere;
}

.container-detail-resource-status-card__title {
  color: var(--td-text-color-secondary);
}

.container-resource-dashboard-section,
.container-resource-history-section,
.container-resource-detail-section {
  background: var(--td-bg-color-container);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  margin-top: var(--graft-density-gap-12);
  min-width: 0;
  overflow: hidden;
  width: 100%;
}

.container-resource-dashboard-section {
  gap: var(--graft-density-gap-14);
  padding: var(--graft-density-gap-14) var(--graft-density-gap-16) var(--graft-density-gap-16);
}

.container-resource-history-section {
  gap: var(--graft-density-gap-12);
  padding: var(--graft-density-gap-14) var(--graft-density-gap-16);
}

.container-resource-dashboard-section__title,
.container-resource-history-section__title,
.container-resource-detail-section__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-weight: 600;
  line-height: 22px;
}

.container-resource-detail-section__title {
  background: color-mix(in srgb, var(--td-bg-color-container) 86%, var(--td-bg-color-page));
  border-bottom: 1px solid color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  padding: var(--graft-density-gap-12) var(--graft-density-gap-16);
}

.container-resource-dashboard-grid,
.container-resource-detail-grid {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  min-width: 0;
}

.container-resource-history-grid {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  min-width: 0;
}

.container-resource-dashboard-panel,
.container-resource-history-card,
.container-resource-detail-card {
  background: color-mix(in srgb, var(--td-bg-color-container) 96%, var(--td-bg-color-page));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 56%, transparent);
  border-radius: var(--td-radius-medium);
  min-width: 0;
}

.container-resource-dashboard-panel {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  padding: var(--graft-density-gap-14);
}

.container-resource-history-card {
  display: grid;
  gap: var(--graft-density-gap-6);
  padding: var(--graft-density-gap-12);
}

.container-resource-dashboard-panel__heading {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
}

.container-resource-dashboard-panel__heading span {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
  min-width: 0;
}

.container-resource-dashboard-panel__heading strong,
.container-resource-dashboard-panel__usage {
  color: var(--td-brand-color);
  font: var(--td-font-title-small);
  font-weight: 600;
  min-width: 0;
  overflow-wrap: anywhere;
}

.container-resource-dashboard-panel__usage {
  color: var(--td-text-color-primary);
}

.container-resource-history-card span,
.container-resource-history-card em {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  font-style: normal;
}

.container-resource-history-card strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-weight: 600;
}

.container-resource-dashboard-panel :deep(.t-progress) {
  width: 100%;
}

.container-resource-dashboard-panel__meta {
  display: grid;
  gap: var(--graft-density-gap-8) var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  min-width: 0;
}

.container-resource-dashboard-panel__meta span {
  color: var(--td-text-color-secondary);
  display: flex;
  flex-direction: column;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-2);
  min-width: 0;
}

.container-resource-dashboard-panel__meta strong {
  color: var(--td-text-color-primary);
  font-weight: 500;
  min-width: 0;
  overflow-wrap: anywhere;
}

.container-resource-detail-grid {
  padding: var(--graft-density-gap-12) var(--graft-density-gap-16) var(--graft-density-gap-16);
}

.container-resource-detail-card {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.container-resource-detail-card--cpu,
.container-resource-detail-card--memory {
  grid-column: 1 / -1;
}

.container-resource-detail-card h3 {
  border-bottom: 1px solid color-mix(in srgb, var(--td-component-stroke) 44%, transparent);
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
  line-height: 22px;
  margin: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

.container-resource-detail-card__body {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: var(--graft-density-gap-6) var(--graft-density-gap-12);
}

.container-resource-detail-card__body--memory {
  display: grid;
  gap: 0 var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  padding: var(--graft-density-gap-6) var(--graft-density-gap-12);
}

.container-resource-cpu-metric-grid {
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: repeat(4, minmax(0, 1fr));
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.container-resource-cpu-metric {
  background: color-mix(in srgb, var(--td-bg-color-container) 94%, var(--td-bg-color-page));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 36%, transparent);
  border-radius: var(--td-radius-medium);
  display: grid;
  gap: var(--graft-density-gap-4);
  min-height: 78px;
  min-width: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

.container-resource-cpu-metric--warning {
  background: color-mix(in srgb, var(--td-warning-color-1) 42%, var(--td-bg-color-container));
  border-color: color-mix(in srgb, var(--td-warning-color) 52%, var(--td-component-stroke));
}

.container-resource-cpu-metric--muted .container-resource-cpu-metric__value,
.container-resource-cpu-metric--muted .container-resource-cpu-metric__hint {
  color: var(--td-text-color-placeholder);
}

.container-resource-cpu-metric__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  line-height: 20px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-resource-cpu-metric__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-weight: 600;
  justify-self: end;
  line-height: 24px;
  max-width: 100%;
  min-width: 0;
  overflow-wrap: anywhere;
  text-align: right;
}

.container-resource-cpu-metric--warning .container-resource-cpu-metric__value {
  color: var(--td-warning-color);
}

.container-resource-cpu-metric__hint {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  line-height: 18px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-resource-detail-row {
  align-items: center;
  column-gap: var(--graft-density-gap-12);
  display: grid;
  grid-template-columns: minmax(112px, 34%) minmax(0, 1fr);
  min-height: 34px;
  min-width: 0;
  width: 100%;
}

.container-resource-detail-row + .container-resource-detail-row {
  border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 30%, transparent);
}

.container-resource-detail-card__body--memory .container-resource-detail-row + .container-resource-detail-row {
  border-top: 0;
}

.container-resource-detail-card__body--memory .container-resource-detail-row {
  border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 30%, transparent);
  grid-template-columns: minmax(112px, 42%) minmax(0, 1fr);
}

.container-resource-detail-card__body--memory .container-resource-detail-row:nth-child(-n + 2) {
  border-top: 0;
}

.container-resource-detail-row--placeholder .container-resource-detail-row__label,
.container-resource-detail-row--placeholder .container-resource-detail-row__value {
  color: var(--td-text-color-placeholder);
}

.container-resource-detail-row__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  line-height: 20px;
  min-width: 0;
}

.container-resource-detail-row__value {
  align-items: center;
  color: var(--td-text-color-primary);
  display: inline-flex;
  font: var(--td-font-body-small);
  font-weight: 500;
  gap: var(--graft-density-gap-6);
  justify-content: flex-end;
  line-height: 22px;
  min-width: 0;
  overflow: hidden;
  text-align: right;
}

.container-resource-detail-row__value > span:not(.t-tag) {
  display: inline-block;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (width <= 1360px) {
  .container-detail-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 960px) {
  .container-detail-resource-grid,
  .container-health-summary-grid,
  .container-mount-card-grid,
  .container-resource-dashboard-grid,
  .container-resource-detail-grid,
  .container-resource-detail-card__body--memory,
  .container-resource-cpu-metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 720px) {
  .container-runtime-config-row {
    align-items: flex-start;
    grid-template-columns: minmax(0, 1fr) var(--td-comp-size-xs);
  }

  .container-runtime-config-row__label {
    grid-column: 1 / -1;
  }

  .container-env-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .container-env-toolbar__search,
  .container-env-toolbar__policy {
    flex-basis: auto;
    min-width: 0;
    width: 100%;
  }

  .container-env-toolbar :deep(.t-button) {
    width: 100%;
  }

  .container-detail-summary,
  .container-detail-summary__resource {
    grid-template-columns: 1fr;
  }

  .container-detail-resource-grid,
  .container-health-summary-grid,
  .container-resource-dashboard-grid,
  .container-resource-detail-grid,
  .container-resource-detail-card__body--memory,
  .container-resource-cpu-metric-grid,
  .container-resource-dashboard-panel__meta {
    grid-template-columns: 1fr;
  }

  .container-resource-detail-card__body--memory .container-resource-detail-row:nth-child(-n + 2) {
    border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 30%, transparent);
  }

  .container-resource-detail-card__body--memory .container-resource-detail-row:first-child {
    border-top: 0;
  }

  .container-resource-detail-row {
    align-items: flex-start;
    gap: var(--graft-density-gap-4);
    grid-template-columns: 1fr;
    padding: var(--graft-density-gap-8) 0;
  }

  .container-resource-detail-row__value {
    justify-content: flex-start;
    text-align: left;
    width: 100%;
  }

  .container-mount-card__header {
    flex-direction: column;
  }

  .container-mount-card__actions {
    justify-content: flex-start;
  }

  .container-mount-field {
    align-items: flex-start;
    gap: var(--graft-density-gap-4);
    grid-template-columns: 1fr;
  }

  .container-detail-header-meta {
    max-width: 100%;
  }

  .container-detail-header-meta__tags :deep(.t-space) {
    width: 100%;
  }

  .container-detail-header-meta__tags :deep(.t-space-item) {
    max-width: 100%;
  }

  .container-detail-realtime-row {
    margin-block: var(--graft-density-gap-8);
  }
}

@media (width <= 767px) {
  .container-detail-realtime-row {
    padding-inline: var(--graft-density-gap-12);
  }

  .container-detail-logs-header {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
