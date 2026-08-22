<template>
  <div class="project-detail-page" data-page-type="list-form-detail">
    <management-page-header
      :title="pageTitle"
      description-key="project.detail.description"
      :description="t('project.detail.description')"
      :source="{ labelKey: 'project.detail.eyebrow', fallback: t('project.detail.eyebrow') }"
    >
      <template #actions>
        <t-space class="project-detail-actions--wide" break-line size="small">
          <t-button
            v-if="lifecycleActionVisibility.up"
            data-testid="project-detail-action-up"
            theme="primary"
            variant="outline"
            :loading="actionLoading === 'up'"
            :disabled="lifecycleReviewRequired || !runtimeTargetCapabilityAvailable('compose_execution')"
            @click="runLifecycleAction('up')"
          >
            {{ t('project.detail.actions.up') }}
          </t-button>
          <t-button
            v-if="lifecycleActionVisibility.stop"
            data-testid="project-detail-action-stop"
            theme="warning"
            variant="outline"
            :loading="actionLoading === 'stop'"
            :disabled="lifecycleReviewRequired || !runtimeTargetCapabilityAvailable('compose_execution')"
            @click="runLifecycleAction('stop')"
          >
            {{ t('project.detail.actions.stop') }}
          </t-button>
          <t-button
            v-if="lifecycleActionVisibility.restart"
            data-testid="project-detail-action-restart"
            theme="warning"
            variant="outline"
            :loading="actionLoading === 'restart'"
            :disabled="lifecycleReviewRequired || !runtimeTargetCapabilityAvailable('compose_execution')"
            @click="runLifecycleAction('restart')"
          >
            {{ t('project.detail.actions.restart') }}
          </t-button>
          <t-button
            v-if="lifecycleActionVisibility.redeploy"
            data-testid="project-detail-action-redeploy"
            theme="default"
            variant="outline"
            :loading="actionLoading === 'redeploy'"
            :disabled="lifecycleReviewRequired || !runtimeTargetCapabilityAvailable('compose_execution')"
            @click="runLifecycleAction('redeploy')"
          >
            {{ t('project.detail.actions.redeploy') }}
          </t-button>
          <t-button
            data-testid="project-detail-action-open-configuration-workspace"
            theme="default"
            variant="outline"
            @click="openConfigurationWorkspace"
          >
            {{ t('project.detail.actions.openConfigurationWorkspace') }}
          </t-button>
          <t-button
            v-if="lifecycleActionVisibility.unregister"
            data-testid="project-detail-action-unregister"
            theme="danger"
            variant="outline"
            :loading="actionLoading === 'unregister'"
            @click="runLifecycleAction('unregister')"
          >
            {{ t('project.detail.actions.unregister') }}
          </t-button>
          <t-button
            data-testid="project-detail-action-destroy"
            theme="danger"
            :loading="actionLoading === 'destroy'"
            :disabled="!runtimeTargetCapabilityAvailable('compose_execution')"
            @click="runDestroyAction"
          >
            {{ t('project.detail.actions.destroy') }}
          </t-button>
        </t-space>
        <t-dropdown
          class="project-detail-actions--compact"
          :options="headerActionOptions"
          placement="bottom-right"
          trigger="click"
          @click="handleHeaderAction"
        >
          <t-button variant="outline" data-testid="project-detail-action-overflow">
            {{ t('project.detail.actions.more') }}
          </t-button>
        </t-dropdown>
      </template>
      <template #meta>
        <t-space break-line size="small">
          <t-tag :theme="driftStatusTheme(detailRecord?.drift_status)" variant="light-outline">
            {{ detailRecord ? driftStatusLabel(detailRecord.drift_status) : '-' }}
          </t-tag>
          <t-tag :theme="runtimeStatusTheme(detailRecord?.runtime_status)" variant="light-outline">
            {{ detailRecord ? runtimeStatusLabel(detailRecord.runtime_status) : '-' }}
          </t-tag>
          <t-tag theme="default" variant="light-outline">
            {{ detailRecord?.compose_project_name || fallbackCanonicalName }}
          </t-tag>
        </t-space>
      </template>
    </management-page-header>

    <section class="project-detail-body">
      <t-loading v-if="detailLoading && !detailRecord && !detailError" :loading="true" class="project-detail-state" />

      <t-alert v-else-if="detailError" theme="error" :message="detailError" class="project-detail-state">
        <template #operation>
          <t-button theme="danger" variant="text" @click="refreshDetail">{{ t('project.list.retry') }}</t-button>
        </template>
      </t-alert>

      <template v-else-if="detailRecord">
        <t-tabs v-model:value="activeDetailTab" class="project-detail-tabs" theme="normal">
          <t-tab-panel value="overview" :destroy-on-hide="false" :label="t('project.detail.tabs.overview')">
            <section class="project-section project-tab-panel">
              <div class="project-section-heading">
                <div>
                  <h2>{{ t('project.detail.sections.overview.title') }}</h2>
                  <p>{{ t('project.detail.sections.overview.description') }}</p>
                </div>
              </div>

              <div class="project-overview-grid">
                <t-card size="small">
                  <div class="project-runtime-target-card" data-testid="project-detail-runtime-target">
                    <header class="project-runtime-target-card__header">
                      <img
                        v-if="runtimeTargetUsesDocker"
                        alt=""
                        class="project-runtime-target-card__icon"
                        :src="dockerRuntimeIcon"
                      />
                      <div class="project-runtime-target-card__identity">
                        <div>
                          <strong>{{ runtimeTargetName }}</strong>
                          <p>{{ runtimeTargetDescription }}</p>
                        </div>
                        <t-tag
                          v-if="runtimeTargetHealthStatus"
                          :theme="runtimeTargetHealthTheme"
                          size="small"
                          variant="light"
                        >
                          {{ runtimeTargetHealthLabel }}
                        </t-tag>
                      </div>
                    </header>
                    <div class="project-runtime-target-card__content">
                      <dl v-if="projectRuntimeTarget" class="project-runtime-target-card__facts">
                        <div>
                          <dt>{{ t('project.detail.overview.runtimeTargetType') }}</dt>
                          <dd>{{ runtimeTargetType }}</dd>
                        </div>
                        <div>
                          <dt>{{ t('project.detail.overview.runtimeTargetVersion') }}</dt>
                          <dd>{{ runtimeTargetVersion }}</dd>
                        </div>
                        <div>
                          <dt>{{ t('project.detail.overview.runtimeTargetOperatingSystem') }}</dt>
                          <dd>{{ runtimeTargetOperatingSystem }}</dd>
                        </div>
                        <div>
                          <dt>{{ t('project.detail.overview.runtimeTargetHost') }}</dt>
                          <dd>{{ runtimeTargetHost }}</dd>
                        </div>
                      </dl>
                      <div v-if="projectRuntimeTarget" class="project-runtime-target-card__endpoint">
                        <span>{{ t('project.detail.overview.runtimeTargetEndpoint') }}</span>
                        <t-tooltip v-if="runtimeTargetDetail?.endpoint" :content="runtimeTargetDetail.endpoint">
                          <strong :title="runtimeTargetDetail.endpoint">{{ runtimeTargetEndpoint }}</strong>
                        </t-tooltip>
                        <strong v-else>{{ runtimeTargetEndpoint }}</strong>
                      </div>
                    </div>
                  </div>
                </t-card>
              </div>

              <div class="project-overview-grid">
                <t-card
                  size="small"
                  :title="t('project.detail.overview.containerSnapshotTitle')"
                  class="project-overview-grid__wide"
                >
                  <div v-if="serviceSnapshotCards.length" class="project-service-snapshot-grid">
                    <article v-for="service in serviceSnapshotCards" :key="service.key" class="project-service-card">
                      <header class="project-service-card__head">
                        <div>
                          <strong>{{ service.name }}</strong>
                          <p>{{ service.meta }}</p>
                        </div>
                        <div class="project-service-card__tags">
                          <t-tag :theme="service.healthTheme" variant="light-outline">{{ service.healthLabel }}</t-tag>
                        </div>
                      </header>
                      <dl class="project-service-card__metrics">
                        <div>
                          <dt>{{ t('project.detail.overview.runtimeStatus') }}</dt>
                          <dd>{{ service.statusLabel }}</dd>
                        </div>
                        <div>
                          <dt>{{ t('project.detail.overview.actionLabel') }}</dt>
                          <dd>
                            <t-button
                              size="small"
                              theme="default"
                              variant="text"
                              :disabled="!service.canOpen"
                              @click="openFirstServiceContainer(service.raw)"
                            >
                              {{ t('project.detail.overview.viewService') }}
                            </t-button>
                          </dd>
                        </div>
                      </dl>
                    </article>
                  </div>
                  <t-empty v-else :description="t('project.detail.services.emptyDescription')" />
                </t-card>
              </div>

              <div class="project-overview-grid">
                <t-card
                  size="small"
                  :title="t('project.detail.overview.diagnosticsTitle')"
                  class="project-overview-grid__wide"
                >
                  <div class="project-diagnostics-list">
                    <t-alert
                      v-for="item in overviewDiagnostics"
                      :key="item.key"
                      :theme="item.theme"
                      :message="item.message"
                    />
                  </div>
                </t-card>
              </div>
            </section>
          </t-tab-panel>

          <t-tab-panel value="services" :destroy-on-hide="false" :label="t('project.detail.tabs.services')">
            <section class="project-section project-tab-panel">
              <div class="project-section-heading">
                <div>
                  <h2>{{ t('project.detail.services.title') }}</h2>
                  <p>{{ t('project.detail.services.description') }}</p>
                </div>
              </div>
              <t-alert
                v-if="!runtimeTargetCapabilityAvailable('container_execution')"
                theme="warning"
                :message="runtimeTargetCapabilityUnavailableReason"
                data-testid="project-detail-container-capability-warning"
              />

              <management-paged-table
                v-model:current="serviceTableCurrent"
                v-model:page-size="serviceTablePageSize"
                :cell-slot-names="['name', 'status', 'health', 'ports', 'operation']"
                cards-visible
                :columns="serviceColumns"
                :description="t('project.detail.services.description')"
                :empty-description="t('project.detail.services.emptyDescription')"
                :empty-title="t('project.detail.services.emptyTitle')"
                :footer-summary="t('project.detail.services.summary', { count: serviceTableRows.length })"
                head-label="project-detail-services-table"
                :loading="serviceLoading || serviceActionLoading"
                :pagination-visible="true"
                row-key="service_name"
                :rows="pagedServiceTableRows"
                :selected-row-keys="selectedServiceRowKeys"
                :summary="t('project.detail.services.summary', { count: serviceTableRows.length })"
                :total="serviceTableRows.length"
                @select-change="handleServiceSelectChange"
              >
                <template #toolbar>
                  <t-button
                    theme="primary"
                    variant="outline"
                    :loading="serviceLoading"
                    :disabled="serviceActionLoading"
                    @click="refreshApplicationServices"
                  >
                    {{ t('project.detail.services.refresh') }}
                  </t-button>
                </template>
                <template v-if="selectedServiceRows.length > 0" #batch>
                  <div class="project-service-selection-toolbar">
                    <span class="project-service-selection-toolbar__summary">
                      {{ t('project.detail.services.batch.selected', { count: selectedServiceRows.length }) }}
                    </span>
                    <t-dropdown
                      :options="serviceBatchActionOptions"
                      placement="bottom-right"
                      trigger="click"
                      @click="handleServiceBatchMenuAction"
                    >
                      <t-button
                        data-testid="project-service-batch-actions"
                        size="small"
                        theme="default"
                        variant="outline"
                      >
                        {{ t('project.detail.services.batch.actionMenu') }}
                      </t-button>
                    </t-dropdown>
                  </div>
                </template>

                <template #cards>
                  <article
                    v-for="row in pagedServiceTableRows"
                    :key="row.service_name"
                    class="project-service-mobile-card"
                    :data-testid="`project-service-card-${row.service_name}`"
                  >
                    <header class="project-service-mobile-card__header">
                      <t-checkbox
                        :aria-label="`${t('project.detail.services.columns.service')} ${row.name}`"
                        :checked="selectedServiceRowKeys.includes(row.service_name)"
                        :disabled="row.raw.managed === false"
                        :data-testid="`project-service-card-select-${row.service_name}`"
                        @change="toggleServiceSelection(row.service_name, $event)"
                      />
                      <div class="project-service-mobile-card__identity">
                        <strong>{{ row.name }}</strong>
                        <t-tag v-if="row.raw.managed === false" theme="default" variant="light-outline">
                          {{ t('project.detail.services.unmanaged') }}
                        </t-tag>
                        <span :title="row.image">{{ row.image }}</span>
                      </div>
                    </header>
                    <dl class="project-service-mobile-card__facts">
                      <div>
                        <dt>{{ t('project.detail.services.columns.status') }}</dt>
                        <dd>
                          <t-tag :theme="row.statusTheme" variant="light-outline">{{ row.statusLabel }}</t-tag>
                        </dd>
                      </div>
                      <div>
                        <dt>{{ t('project.detail.services.columns.health') }}</dt>
                        <dd>
                          <t-tag :theme="row.healthTheme" variant="light-outline">{{ row.healthLabel }}</t-tag>
                        </dd>
                      </div>
                      <div class="project-service-mobile-card__ports">
                        <dt>{{ t('project.detail.services.columns.ports') }}</dt>
                        <dd>{{ row.portsSummary }}</dd>
                      </div>
                    </dl>
                    <table-action-menu
                      class="project-service-mobile-card__actions"
                      :actions="serviceActionOptions(row)"
                      @action="handleServiceAction($event, row)"
                    />
                  </article>
                </template>

                <template #name="{ row }">
                  <div class="project-service-name">
                    <strong>{{ row.name }}</strong>
                    <t-tag v-if="row.raw.managed === false" theme="default" variant="light-outline">
                      {{ t('project.detail.services.unmanaged') }}
                    </t-tag>
                    <span>{{ row.image }}</span>
                  </div>
                </template>

                <template #status="{ row }">
                  <t-tag :theme="row.statusTheme" variant="light-outline">
                    {{ row.statusLabel }}
                  </t-tag>
                </template>

                <template #health="{ row }">
                  <t-tag :theme="row.healthTheme" variant="light-outline">
                    {{ row.healthLabel }}
                  </t-tag>
                </template>

                <template #ports="{ row }">
                  <span>{{ row.portsSummary }}</span>
                </template>

                <template #operation="{ row }">
                  <table-action-menu :actions="serviceActionOptions(row)" @action="handleServiceAction($event, row)" />
                </template>
              </management-paged-table>
            </section>
          </t-tab-panel>

          <t-tab-panel value="logs" :destroy-on-hide="false" :label="t('project.detail.tabs.logs')">
            <section class="project-section project-tab-panel">
              <t-card size="small" :title="t('project.detail.logs.title')">
                <template #actions>
                  <t-space size="small" align="center">
                    <t-input
                      v-model="projectLogSince"
                      class="project-activity-toolbar__since"
                      :placeholder="t('project.detail.logs.sinceLabel')"
                    />
                    <t-button
                      theme="primary"
                      variant="outline"
                      :loading="projectLogLoading"
                      @click="loadApplicationLogs()"
                    >
                      {{ t('project.list.refresh') }}
                    </t-button>
                  </t-space>
                </template>
                <p class="project-inline-head__hint">{{ t('project.detail.logs.summary') }}</p>
                <t-alert theme="info" :message="projectLogAuthorityNotice" class="project-activity-alert" />
                <div class="project-logs-toolbar">
                  <div class="project-logs-toolbar__summary">
                    <t-tag theme="default" variant="light-outline">
                      {{ t('project.detail.logs.memberCount', { count: projectLogSourceCount }) }}
                    </t-tag>
                    <t-tag theme="default" variant="light-outline">
                      {{ t('project.detail.logs.logCount', { count: projectLogEntries.length }) }}
                    </t-tag>
                  </div>
                </div>
                <log-viewer
                  :entries="projectLogEntries"
                  :content-version="projectLogContentVersion"
                  :loading="projectLogLoading"
                  :error="projectLogError"
                  :truncated="projectLogResponse?.truncated"
                  :line-limit="projectLogTail"
                  :clear-label="t('project.detail.logs.clear')"
                  :copy-label="t('project.detail.logs.copy')"
                  :download-label="t('project.detail.logs.download')"
                  :retry-label="t('project.list.retry')"
                  :search-placeholder="t('project.detail.logs.searchPlaceholder')"
                  :wrap-label="t('project.detail.logs.wrap')"
                  :auto-scroll-label="t('project.detail.logs.autoScroll')"
                  :auto-scroll-tooltip-label="t('project.detail.logs.autoScrollTooltip')"
                  :pause-label="t('project.detail.logs.pause')"
                  :resume-label="t('project.detail.logs.resume')"
                  :reconnect-label="t('project.detail.logs.refreshAction')"
                  :jump-bottom-label="t('project.detail.logs.jumpBottom')"
                  :jump-top-label="t('project.detail.logs.jumpTop')"
                  :level-filter-label="t('project.detail.logs.levelFilter')"
                  :all-levels-label="t('project.detail.logs.allLevels')"
                  :match-count-label="t('project.detail.logs.matchCount')"
                  :empty-label="t('project.detail.logs.empty')"
                  :empty-description-label="t('project.detail.logs.emptyDescription')"
                  :truncated-label="t('project.detail.logs.truncated')"
                  :detail-title-label="t('project.detail.logs.detailTitle')"
                  :important-fields-label="t('project.detail.logs.importantFields')"
                  :basic-info-label="t('project.detail.logs.basicInfo')"
                  :time-label="t('project.detail.logs.time')"
                  :level-label="t('project.detail.logs.level')"
                  :stream-label="t('project.detail.logs.stream')"
                  :source-label="t('project.detail.logs.source')"
                  :operation-label="t('project.detail.logs.operation')"
                  :stdout-label="t('project.detail.logs.stdout')"
                  :stderr-label="t('project.detail.logs.stderr')"
                  :view-detail-label="t('project.detail.logs.viewDetail')"
                  :collapse-detail-label="t('project.detail.logs.collapseDetail')"
                  :metadata-label="t('project.detail.logs.metadata')"
                  :message-label="t('project.detail.logs.message')"
                  :raw-label="t('project.detail.logs.raw')"
                  :copy-message-label="t('project.detail.logs.copyMessage')"
                  :copy-line-label="t('project.detail.logs.copyLine')"
                  :copy-json-label="t('project.detail.logs.copyJson')"
                  :copy-success-label="t('project.detail.logs.copySuccess')"
                  :copy-error-label="t('project.detail.logs.copyError')"
                  :more-actions-label="t('project.detail.logs.moreActions')"
                  :expand-log-label="t('project.detail.logs.expandLog')"
                  :collapse-log-label="t('project.detail.logs.collapseLog')"
                  :download-log-fragment-label="t('project.detail.logs.downloadLogFragment')"
                  :detail-wrap-label="t('project.detail.logs.detailWrap')"
                  :font-size-label="t('project.detail.logs.fontSize')"
                  :font-size-small-label="t('project.detail.logs.fontSizeSmall')"
                  :font-size-medium-label="t('project.detail.logs.fontSizeMedium')"
                  :font-size-large-label="t('project.detail.logs.fontSizeLarge')"
                  :paused="projectLogPaused"
                  :viewer-mode="true"
                  viewer-storage-key="graft.project.logs.height"
                  :fullscreen-label="t('project.detail.logs.fullscreen')"
                  :exit-fullscreen-label="t('project.detail.logs.exitFullscreen')"
                  :resize-handle-label="t('project.detail.logs.resize')"
                  @clear="clearApplicationLogs"
                  @pause="pauseApplicationLogs"
                  @refresh="loadApplicationLogs()"
                  @resume="resumeApplicationLogs"
                  @update:line-limit="updateApplicationLogTail"
                />
              </t-card>
            </section>
          </t-tab-panel>

          <t-tab-panel value="lifecycle" :destroy-on-hide="false" :label="t('project.detail.tabs.lifecycle')">
            <section class="project-section project-tab-panel">
              <div class="project-section-heading">
                <div>
                  <h2>{{ t('project.detail.lifecycle.title') }}</h2>
                  <p>{{ t('project.detail.lifecycle.description') }}</p>
                </div>
              </div>

              <t-alert
                v-if="lifecycleReviewRequired"
                class="project-lifecycle-alert"
                theme="warning"
                :title="t('project.detail.lifecycle.reviewRequiredTitle')"
                :message="t('project.detail.lifecycle.reviewRequiredDescription')"
              />
              <t-alert
                v-if="lifecycleRemoteStale"
                class="project-lifecycle-alert"
                data-testid="project-lifecycle-remote-stale-alert"
                theme="info"
                :title="t('project.detail.lifecycle.remoteStaleTitle')"
                :message="t('project.detail.lifecycle.remoteStaleDescription')"
              />
              <div class="project-lifecycle-layout">
                <t-card
                  size="small"
                  :title="t('project.detail.lifecycle.statusTitle')"
                  data-testid="project-lifecycle-summary-card"
                >
                  <div class="project-lifecycle-summary">
                    <p class="project-inline-head__hint">{{ t('project.detail.lifecycle.statusDescription') }}</p>
                    <t-descriptions size="small" :column="1" table-layout="auto">
                      <t-descriptions-item :label="t('project.detail.lifecycle.reviewStatus')">
                        <t-tag
                          :theme="projectLifecycleReviewStatusTheme(lifecycleReviewStatus)"
                          variant="light-outline"
                        >
                          {{ projectLifecycleReviewStatusLabel(t, lifecycleReviewStatus) }}
                        </t-tag>
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.lifecycle.mode')">
                        {{ t('project.detail.lifecycle.modeStandard') }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.lifecycle.workspacePath')">
                        <div class="project-detail-copy-row">
                          <code>{{ lifecycleDraft.workspace_path }}</code>
                          <t-button
                            size="small"
                            theme="default"
                            variant="text"
                            @click="copyPath(lifecycleDraft.workspace_path)"
                          >
                            {{ t('project.detail.actions.copyPath') }}
                          </t-button>
                        </div>
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.lifecycle.projectName')">
                        <code>{{ lifecycleDraft.compose_project_name }}</code>
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.lifecycle.composeFiles')">
                        <div class="project-lifecycle-file-list">
                          <code v-for="file in lifecycleDraft.compose_files" :key="file">{{ file }}</code>
                        </div>
                      </t-descriptions-item>
                    </t-descriptions>
                  </div>
                </t-card>

                <t-card
                  size="small"
                  :title="t('project.detail.lifecycle.configurationTitle')"
                  data-testid="project-lifecycle-configuration-card"
                >
                  <div class="project-lifecycle-config">
                    <section class="project-lifecycle-config-group" data-testid="project-lifecycle-config-group-base">
                      <header class="project-lifecycle-config-group__header">
                        <div>
                          <strong>{{ t('project.detail.lifecycle.groups.base.title') }}</strong>
                          <p>{{ t('project.detail.lifecycle.groups.base.description') }}</p>
                        </div>
                      </header>
                      <div class="project-lifecycle-field-grid">
                        <label class="project-lifecycle-field">
                          <span class="project-lifecycle-field__label">{{
                            t('project.detail.lifecycle.profiles')
                          }}</span>
                          <t-input
                            v-model="lifecycleProfilesInput"
                            :placeholder="t('project.detail.lifecycle.profilesPlaceholder')"
                          />
                        </label>
                      </div>
                    </section>

                    <section
                      class="project-lifecycle-config-group"
                      data-testid="project-lifecycle-config-group-redeploy"
                    >
                      <header class="project-lifecycle-config-group__header">
                        <div>
                          <strong>{{ t('project.detail.lifecycle.groups.redeploy.title') }}</strong>
                          <p>{{ t('project.detail.lifecycle.groups.redeploy.description') }}</p>
                        </div>
                      </header>
                      <div class="project-lifecycle-option-list">
                        <div
                          v-for="definition in lifecycleSwitchOptionDefinitionsBeforeWaitTimeout"
                          :key="definition.key"
                          class="project-lifecycle-option"
                        >
                          <div class="project-lifecycle-option__content">
                            <div class="project-lifecycle-option__label">
                              <span>{{ t(definition.titleKey) }}</span>
                              <lifecycle-help-trigger :definition="definition" />
                            </div>
                            <p>{{ t(definition.summaryKey) }}</p>
                          </div>
                          <div class="project-lifecycle-option__control">
                            <t-switch
                              v-model="lifecycleDraft[definition.field]"
                              :aria-label="t(definition.titleKey)"
                              :data-testid="definition.switchTestId"
                            />
                          </div>
                        </div>
                        <label
                          v-if="lifecycleWaitTimeoutDefinition.visible?.(lifecycleDraft)"
                          class="project-lifecycle-field"
                          data-testid="project-lifecycle-wait-timeout-field"
                        >
                          <span class="project-lifecycle-field__label project-lifecycle-field__label--with-help">
                            <span>{{ t(lifecycleWaitTimeoutDefinition.titleKey) }}</span>
                            <lifecycle-help-trigger :definition="lifecycleWaitTimeoutDefinition" />
                          </span>
                          <t-input-number
                            v-model="lifecycleDraft.wait_timeout_seconds"
                            :min="1"
                            :max="3600"
                            :step="1"
                          />
                          <small class="project-lifecycle-field__hint">
                            {{ t(lifecycleWaitTimeoutDefinition.summaryKey) }}
                          </small>
                        </label>
                        <div
                          v-for="definition in lifecycleSwitchOptionDefinitionsAfterWaitTimeout"
                          :key="definition.key"
                          class="project-lifecycle-option"
                        >
                          <div class="project-lifecycle-option__content">
                            <div class="project-lifecycle-option__label">
                              <span>{{ t(definition.titleKey) }}</span>
                              <lifecycle-help-trigger :definition="definition" />
                            </div>
                            <p>{{ t(definition.summaryKey) }}</p>
                          </div>
                          <div class="project-lifecycle-option__control">
                            <t-switch
                              v-model="lifecycleDraft[definition.field]"
                              :aria-label="t(definition.titleKey)"
                              :data-testid="definition.switchTestId"
                            />
                          </div>
                        </div>
                        <t-alert
                          v-if="lifecycleDraft.renew_anon_volumes"
                          data-testid="project-lifecycle-renew-anon-volumes-warning"
                          theme="warning"
                          :message="t('project.detail.lifecycle.renewAnonVolumesWarning')"
                        />
                      </div>
                    </section>
                  </div>
                  <div class="project-lifecycle-actions" data-testid="project-lifecycle-actions">
                    <t-button
                      theme="default"
                      variant="outline"
                      :disabled="lifecycleSaveLoading || !lifecycleDraftDirty"
                      @click="resetLifecycleConfiguration"
                    >
                      {{ t('project.detail.lifecycle.reset') }}
                    </t-button>
                    <t-button
                      data-testid="project-lifecycle-save"
                      theme="primary"
                      :disabled="lifecycleSaveLoading || !lifecycleCanSave"
                      :loading="lifecycleSaveLoading"
                      @click="saveLifecycleConfiguration"
                    >
                      {{ t('project.detail.lifecycle.save') }}
                    </t-button>
                  </div>
                </t-card>
              </div>
            </section>
          </t-tab-panel>
          <t-tab-panel value="tasks" :destroy-on-hide="false" :label="t('project.detail.tabs.tasks')">
            <section class="project-section project-tab-panel">
              <task-history-table
                owner-type="application"
                :owner-id="projectTaskOwnerId"
                :resolve-task-type="(taskType) => projectTaskTypeLabel(t, taskType)"
                @open="openTaskDrawer($event.id)"
              />
            </section>
          </t-tab-panel>
        </t-tabs>
      </template>
    </section>
    <task-detail-drawer
      v-model:visible="taskDrawerVisible"
      :resolve-task-type="(taskType) => projectTaskTypeLabel(t, taskType)"
      :task-id="activeTaskId"
    />
  </div>
</template>
<script setup lang="ts">
import type { DropdownItemTheme, DropdownProps, TableProps } from 'tdesign-vue-next';
import { DialogPlugin } from 'tdesign-vue-next/es/dialog';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { NotifyPlugin } from 'tdesign-vue-next/es/notification';
import { computed, h, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { LOCALE, type LocalizedTitle } from '@/contracts/i18n/locales';
import { CONTAINER_BOOTSTRAP_ROUTE } from '@/modules/container/contract/bootstrap';
import {
  batchContainerActions,
  type ProjectContainerAction,
  type ProjectContainerActionResult,
  type ProjectContainerActionResultItem,
  type ProjectContainerActionSubmission,
  type ProjectContainerSummary,
} from '@/modules/container/contract/project';
import {
  type ApplicationRuntimeTargetDetail,
  getApplicationRuntimeTargetDetail,
} from '@/modules/runtime-target/contract/application-target-detail';
import { TaskDetailDrawer, TaskHistoryTable } from '@/modules/task/contract/task-ui';
import {
  createActionColumn,
  createMainTextColumn,
  createStatusColumn,
  createTextColumn,
  ManagementPageHeader,
  TableActionMenu,
} from '@/shared/components/management';
import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { copyText, LogViewer, normalizeStructuredLogEntry } from '@/shared/observability';
import {
  createRealtimeSnapshotGate,
  openRealtimeTopicSocket,
  type RealtimeTopicSocketController,
} from '@/shared/realtime';
import { useTabsRouterStore } from '@/store/modules/tabs-router';
import { createLogger } from '@/utils/logger';

import {
  getApplication,
  getApplicationConfiguration,
  getApplicationLogs,
  getApplicationOverview,
  getApplicationServices,
  postApplicationDestroy,
  postApplicationRedeploy,
  postApplicationRestart,
  postApplicationStop,
  postApplicationUnregister,
  postApplicationUp,
  putApplicationLifecycleConfiguration,
} from '../../api/project';
import dockerRuntimeIcon from '../../assets/runtime/docker.svg?url';
import LifecycleHelpTrigger from '../../components/LifecycleHelpTrigger.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import {
  buildApplicationLifecycleConfigTopicName,
  buildApplicationLogsTopicName,
  buildApplicationRuntimeTopicName,
  parseApplicationLifecycleConfigRealtimePayload,
  parseApplicationLogsRealtimePayload,
  parseApplicationRuntimeRealtimePayload,
} from '../../contract/realtime';
import { paginateApplicationResourceRows } from '../../shared/detail-resources';
import {
  projectDriftStatusLabel,
  projectDriftStatusTheme,
  projectLifecycleActionVisibility,
  projectRuntimeStatusLabel,
  projectRuntimeStatusTheme as runtimeStatusTheme,
  projectTaskTypeLabel,
} from '../../shared/display';
import {
  buildLifecycleConfigurationDraft,
  buildLifecycleConfigurationRequest,
  isLifecycleDraftDirty,
  lifecycleDraftProfilesText,
  projectLifecycleReviewStatusLabel,
  projectLifecycleReviewStatusTheme,
  updateLifecycleDraftProfiles,
} from '../../shared/lifecycle';
import { lifecycleSwitchHelpDefinitions, lifecycleWaitTimeoutHelpDefinition } from '../../shared/lifecycle-help';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../../shared/navigation';
import { emitApplicationLogDebug } from '../../shared/project-log-debug';
import { ApplicationLogRealtimeBatcher } from '../../shared/project-log-realtime-batcher';
import { fetchApplicationRuntimeContainers, readProjectContainerSourceMember } from '../../shared/runtime-containers';
import type {
  ApplicationActionResponse,
  ApplicationConfigurationMetadataResponse,
  ApplicationDestroyRequest,
  ApplicationDetailResponseWithLifecycle,
  ApplicationLifecycleConfigurationDraft,
  ApplicationLifecycleReviewStatus,
  ApplicationLogEntry,
  ApplicationLogResponse,
  ApplicationOverviewResponse,
  ApplicationOverviewServiceItem,
  ApplicationServiceContainerMember,
  ApplicationServiceItem,
} from '../../types/project';

// 项目详情页组合服务端项目数据、容器操作和实时日志；实时订阅只更新观察快照，不改变项目契约的拥有边界。
defineOptions({
  name: 'ApplicationDetailIndex',
});

type ApplicationDetailTab = 'overview' | 'services' | 'logs' | 'lifecycle' | 'tasks';
type OverviewDiagnostic = {
  key: string;
  message: string;
  theme: 'success' | 'warning' | 'error' | 'info';
};
type ServiceRowAction = 'detail' | ProjectContainerAction;
type ServiceTableRow = {
  healthLabel: string;
  healthTheme: 'success' | 'warning' | 'danger' | 'default';
  hasMembers: boolean;
  image: string;
  name: string;
  portsSummary: string;
  raw: ApplicationServiceItem;
  runningCount: number;
  service_name: string;
  statusLabel: string;
  statusTheme: 'success' | 'warning' | 'danger' | 'default';
};
type ServiceSnapshotCard = {
  key: string;
  name: string;
  image: string;
  meta: string;
  statusLabel: string;
  statusTheme: 'success' | 'warning' | 'danger' | 'default';
  healthLabel: string;
  healthTheme: 'success' | 'warning' | 'danger' | 'default';
  memberValue: string;
  canOpen: boolean;
  raw: ApplicationServiceItem;
};
const { locale, t } = useI18n();
const route = useRoute();
const router = useRouter();
const tabsRouterStore = useTabsRouterStore();
const logger = createLogger('project.detail');
const activeTaskId = ref<number | null>(null);
const taskDrawerVisible = ref(false);

const detailRecord = ref<ApplicationDetailResponseWithLifecycle | null>(null);
const detailLoading = ref(false);
const detailError = ref('');
const activeDetailTab = ref<ApplicationDetailTab>(normalizeDetailTab(route.query.tab));
const configurationMetadata = ref<ApplicationConfigurationMetadataResponse | null>(null);
const projectLogResponse = ref<ApplicationLogResponse | null>(null);
const projectLogLoading = ref(false);
const projectLogError = ref('');
const projectLogPaused = ref(false);
const projectLogContentVersion = ref(0);
const serviceRows = ref<ApplicationServiceItem[]>([]);
const projectOverview = ref<ApplicationOverviewResponse | null>(null);
const serviceActionKey = ref('');
const serviceBatchActionLoading = ref<ProjectContainerAction | ''>('');
const serviceLoading = ref(false);
const serviceRuntimePortSummaries = ref<Record<string, string>>({});
const runtimeTargetDetail = ref<ApplicationRuntimeTargetDetail | null>(null);
let runtimeTargetDetailRequestId = 0;
const serviceRuntimePortsRequestId = ref(0);
const serviceTableCurrent = ref(1);
const serviceTablePageSize = ref(20);
const selectedServiceRowKeys = ref<Array<string | number>>([]);
const servicesLoaded = ref(false);
const projectOverviewLoaded = ref(false);
const actionLoading = ref<ApplicationActionResponse['action'] | 'destroy' | ''>('');
const lifecycleSaveLoading = ref(false);
const lifecycleBaseline = ref<ApplicationLifecycleConfigurationDraft | null>(null);
const lifecycleRemoteStale = ref(false);
const projectLogSince = ref('1h');
const projectLogTail = ref(200);
const lifecycleDraft = reactive<ApplicationLifecycleConfigurationDraft>({
  strategy_kind: 'standard',
  workspace_path: '',
  compose_files: [],
  compose_project_name: '',
  profiles: [],
  down_before_redeploy: true,
  pull_before_redeploy: false,
  build_before_up: false,
  force_recreate: false,
  remove_orphans: true,
  wait_after_up: false,
  wait_timeout_seconds: 120,
  renew_anon_volumes: false,
  prune_images_after_redeploy: false,
  managed_service_names: [],
  declared_service_names: [],
});
const projectRuntimeSocketState = ref<'idle' | 'connecting' | 'open' | 'closed' | 'error'>('idle');
const projectLifecycleConfigSocketState = ref<'idle' | 'connecting' | 'open' | 'closed' | 'error'>('idle');
const projectLogsSocketState = ref<'idle' | 'connecting' | 'open' | 'closed' | 'error'>('idle');
const projectLogsHasSnapshot = ref(false);
const projectLogsBootstrapRequested = ref(false);
const projectLogsRecoveryLoadRequested = ref(false);
let projectRuntimeRealtimeController: RealtimeTopicSocketController | null = null;
let projectLifecycleConfigRealtimeController: RealtimeTopicSocketController | null = null;
let projectLogsRealtimeController: RealtimeTopicSocketController | null = null;
let projectRuntimeRealtimeTopic = '';
let projectLifecycleConfigRealtimeTopic = '';
let projectLogsRealtimeTopic = '';
let projectLogsSubscriptionSequence = 0;
let pendingApplicationLogSnapshot: ApplicationLogResponse | null = null;
let projectLogsLoadSequence = 0;
let serviceBatchIdempotencySequence = 0;
const projectLogRealtimeBatcher = new ApplicationLogRealtimeBatcher({
  lineLimit: projectLogTail.value,
  onCommit: (snapshot) => {
    emitApplicationLogDebug('view-snapshot-commit', {
      entryCount: snapshot.entries.length,
      paused: projectLogPaused.value,
      tail: snapshot.tail,
      truncated: snapshot.truncated,
    });
    if (projectLogPaused.value) {
      pendingApplicationLogSnapshot = snapshot;
      return;
    }
    projectLogResponse.value = snapshot;
    projectLogContentVersion.value += 1;
  },
});
const projectRuntimeRealtimeGate = createRealtimeSnapshotGate({
  apply: (message: {
    detail: ApplicationDetailResponseWithLifecycle;
    overview: ApplicationOverviewResponse;
    services: { items: ApplicationServiceItem[] };
  }) => {
    applyApplicationRuntimeRealtimeSnapshot(message);
  },
});
const projectLifecycleConfigRealtimeGate = createRealtimeSnapshotGate({
  apply: (message: { detail: ApplicationDetailResponseWithLifecycle }) => {
    applyApplicationLifecycleConfigRealtimeSnapshot(message);
  },
});

const applicationId = computed(() =>
  typeof route.params.applicationId === 'string' ? route.params.applicationId : '',
);
const projectTaskOwnerId = computed(() => applicationId.value);
const activeTabRoute = computed(() => {
  if (route.name !== PROJECT_BOOTSTRAP_ROUTE.DETAIL.pageRouteName) {
    return undefined;
  }

  const activeTab = tabsRouterStore.tabRouterList.find((tab) => tab.tabKey === tabsRouterStore.activeTabKey);
  return activeTab?.path === route.path && activeTab.name === route.name ? activeTab : undefined;
});
const fallbackDisplayName = computed(() => {
  const tabTitle = readNameFromTabTitle(activeTabRoute.value?.title);
  if (tabTitle) return tabTitle;
  const queryName = typeof route.query.name === 'string' ? route.query.name : '';
  return queryName.trim();
});
const fallbackCanonicalName = computed(() => fallbackDisplayName.value);
const pageTitle = computed(
  () => detailRecord.value?.display_name || fallbackDisplayName.value || t('project.detail.titleFallback'),
);
const projectRuntimeTarget = computed(() => detailRecord.value?.runtime_target ?? null);
const runtimeTargetName = computed(
  () =>
    runtimeTargetDetail.value?.displayName ||
    projectRuntimeTarget.value?.display_name ||
    t('project.detail.overview.runtimeTargetUnavailable'),
);
const runtimeTargetDescription = computed(() =>
  runtimeTargetUsesDocker.value
    ? t('project.detail.overview.runtimeTargetDockerDescription')
    : t('project.detail.overview.runtimeTargetDescription'),
);
const runtimeTargetUsesDocker = computed(() => {
  const provider = runtimeTargetDetail.value?.provider || projectRuntimeTarget.value?.provider || '';
  return provider.toLowerCase() === 'docker';
});
const runtimeTargetType = computed(() => {
  const provider = runtimeTargetDetail.value?.provider || projectRuntimeTarget.value?.provider || '';
  if (provider.toLowerCase() === 'docker') {
    return t('project.detail.overview.runtimeTargetDockerEngine');
  }
  return runtimeTargetDetail.value?.runtimeType || provider || runtimeTargetUnavailable.value;
});
const runtimeTargetEndpoint = computed(() => {
  const endpoint = runtimeTargetDetail.value?.endpoint;
  if (!endpoint) return runtimeTargetUnavailable.value;
  return endpoint.startsWith('unix://') ? endpoint.slice('unix://'.length) : endpoint;
});
const runtimeTargetVersion = computed(() => runtimeTargetDetail.value?.version || runtimeTargetUnavailable.value);
const runtimeTargetOperatingSystem = computed(
  () => runtimeTargetDetail.value?.operatingSystem || runtimeTargetUnavailable.value,
);
const runtimeTargetHost = computed(() => runtimeTargetDetail.value?.hostName || runtimeTargetUnavailable.value);
const runtimeTargetHealthStatus = computed(() => runtimeTargetDetail.value?.healthStatus || '');
const runtimeTargetHealthTheme = computed(() => (runtimeTargetHealthStatus.value === 'healthy' ? 'success' : 'danger'));
const runtimeTargetHealthLabel = computed(() =>
  runtimeTargetHealthStatus.value === 'healthy'
    ? t('project.detail.overview.runtimeTargetStatusHealthy')
    : t('project.detail.overview.runtimeTargetStatusUnavailable'),
);
const runtimeTargetUnavailable = computed(() => t('project.detail.overview.runtimeTargetValueUnavailable'));
const runtimeTargetCapabilityUnavailableReason = computed(() => t('project.detail.runtimeTargetCapabilityUnavailable'));
const lifecycleActionVisibility = computed(() => projectLifecycleActionVisibility(detailRecord.value?.runtime_status));
const lifecycleReviewStatus = computed<ApplicationLifecycleReviewStatus>(() => {
  if (!detailRecord.value) {
    return 'confirmed';
  }
  return buildLifecycleConfigurationDraft(detailRecord.value).review_status ?? 'confirmed';
});
const lifecycleReviewRequired = computed(() => lifecycleReviewStatus.value === 'review_required');
const headerActionOptions = computed<DropdownProps['options']>(() => {
  const options: DropdownProps['options'] = [];
  const add = (content: string, value: string, disabled = false, theme?: DropdownItemTheme) => {
    options.push({ content, disabled, theme, value });
  };
  const composeUnavailable = !runtimeTargetCapabilityAvailable('compose_execution');
  if (lifecycleActionVisibility.value.up)
    add(t('project.detail.actions.up'), 'up', lifecycleReviewRequired.value || composeUnavailable, 'success');
  if (lifecycleActionVisibility.value.stop)
    add(t('project.detail.actions.stop'), 'stop', lifecycleReviewRequired.value || composeUnavailable, 'warning');
  if (lifecycleActionVisibility.value.restart)
    add(t('project.detail.actions.restart'), 'restart', lifecycleReviewRequired.value || composeUnavailable, 'warning');
  if (lifecycleActionVisibility.value.redeploy)
    add(t('project.detail.actions.redeploy'), 'redeploy', lifecycleReviewRequired.value || composeUnavailable);
  add(t('project.detail.actions.openConfigurationWorkspace'), 'configuration');
  if (lifecycleActionVisibility.value.unregister)
    add(t('project.detail.actions.unregister'), 'unregister', false, 'error');
  add(t('project.detail.actions.destroy'), 'destroy', composeUnavailable, 'error');
  return options;
});
const handleHeaderAction: NonNullable<DropdownProps['onClick']> = (item) => {
  const value = typeof item === 'object' && item ? item.value : item;
  if (value === 'configuration') openConfigurationWorkspace();
  else if (value === 'destroy') void runDestroyAction();
  else if (value === 'up' || value === 'stop' || value === 'restart' || value === 'redeploy' || value === 'unregister')
    void runLifecycleAction(value);
};
const lifecycleDraftDirty = computed(() => {
  if (!lifecycleBaseline.value) {
    return false;
  }
  return isLifecycleDraftDirty(lifecycleDraft, lifecycleBaseline.value);
});
const lifecycleCanSave = computed(() => lifecycleReviewRequired.value || lifecycleDraftDirty.value);
const lifecycleProfilesInput = computed({
  get: () => lifecycleDraftProfilesText(lifecycleDraft),
  set: (value: string) => {
    updateLifecycleDraftProfiles(lifecycleDraft, value);
  },
});
const lifecycleWaitTimeoutDefinition = lifecycleWaitTimeoutHelpDefinition;
const waitAfterUpDefinitionIndex = lifecycleSwitchHelpDefinitions.findIndex((item) => item.key === 'waitAfterUp');
const lifecycleSwitchOptionDefinitionsBeforeWaitTimeout =
  waitAfterUpDefinitionIndex >= 0
    ? lifecycleSwitchHelpDefinitions.slice(0, waitAfterUpDefinitionIndex + 1)
    : lifecycleSwitchHelpDefinitions;
const lifecycleSwitchOptionDefinitionsAfterWaitTimeout =
  waitAfterUpDefinitionIndex >= 0 ? lifecycleSwitchHelpDefinitions.slice(waitAfterUpDefinitionIndex + 1) : [];
const overviewServiceMap = computed<Map<string, ApplicationOverviewServiceItem>>(
  () =>
    new Map(
      (projectOverview.value?.services || []).map((item: ApplicationOverviewServiceItem) => [item.service_name, item]),
    ),
);
const totalRestartCount = computed(() => projectOverview.value?.health.restart_count ?? 0);
const unhealthyContainerCount = computed(() => projectOverview.value?.health.unhealthy_container_count ?? 0);
const projectLogAuthorityNotice = computed(() => t('project.detail.logs.authorityApplicationOwned'));
const projectLogSourceCount = computed(() => {
  const entries = projectLogResponse.value?.entries ?? [];
  return new Set(entries.map((entry) => `${entry.container_id}:${entry.service_name}`)).size;
});
const serviceActionLoading = computed(() => serviceActionKey.value.length > 0);
const serviceActionBusy = computed(() => serviceActionLoading.value || serviceBatchActionLoading.value.length > 0);
const serviceTableRows = computed<ServiceTableRow[]>(() =>
  serviceRows.value.map((service) => {
    const overviewItem = overviewServiceMap.value.get(service.service_name);
    return {
      healthLabel: overviewServiceHealthLabel(overviewItem?.health),
      healthTheme: overviewServiceHealthTheme(overviewItem?.health),
      hasMembers: service.container_members.length > 0,
      image: service.image || '-',
      name: service.service_name,
      portsSummary: serviceRuntimePortSummaries.value[service.service_name] || '-',
      raw: service,
      runningCount: service.running_count,
      service_name: service.service_name,
      statusLabel: overviewServiceStatusLabel(overviewItem?.status),
      statusTheme: overviewServiceStatusTheme(overviewItem?.status),
    };
  }),
);
const pagedServiceTableRows = computed<ServiceTableRow[]>(() =>
  paginateApplicationResourceRows(serviceTableRows.value, serviceTableCurrent.value, serviceTablePageSize.value),
);
const serviceRowMap = computed(() => new Map(serviceTableRows.value.map((row) => [row.service_name, row])));
const selectedServiceRows = computed<ServiceTableRow[]>(() =>
  selectedServiceRowKeys.value
    .map((key) => serviceRowMap.value.get(String(key)))
    .filter((row): row is ServiceTableRow => Boolean(row)),
);
const serviceSnapshotCards = computed<ServiceSnapshotCard[]>(() =>
  serviceRows.value.map((service) => {
    const overviewItem = overviewServiceMap.value.get(service.service_name);
    return {
      key: service.service_name,
      name: service.service_name,
      image: service.image || '-',
      meta: service.image || '-',
      statusLabel: overviewServiceStatusLabel(overviewItem?.status),
      statusTheme: overviewServiceStatusTheme(overviewItem?.status),
      healthLabel: overviewServiceHealthLabel(overviewItem?.health),
      healthTheme: overviewServiceHealthTheme(overviewItem?.health),
      memberValue: `${overviewItem?.running_count ?? service.running_count}/${overviewItem?.container_count ?? service.container_members.length}`,
      canOpen: service.container_members.length > 0,
      raw: service,
    };
  }),
);
const projectLogEntries = computed(() => {
  const rawEntries = projectLogResponse.value?.entries ?? [];
  const normalizedEntries = rawEntries
    .map((entry) =>
      normalizeStructuredLogEntry({
        line: JSON.stringify({
          container: entry.container_name,
          container_id: entry.container_id,
          message: entry.line,
          occurred_at: entry.occurred_at,
          service: entry.service_name,
          source: `${entry.source.service_name} · ${entry.source.container_name}`,
          stream: entry.stream,
        }),
        occurred_at: entry.occurred_at,
        stream: entry.stream,
      }),
    )
    // 保留服务端与实时流的时间顺序，让 LogViewer 始终把最新日志显示在底部。
    .filter((entry): entry is NonNullable<ReturnType<typeof normalizeStructuredLogEntry>> => entry !== null);
  emitApplicationLogDebug('view-entries-normalized', {
    rawCount: rawEntries.length,
    visibleCount: normalizedEntries.length,
    truncated: Boolean(projectLogResponse.value?.truncated),
  });
  return normalizedEntries;
});
const overviewDiagnostics = computed<OverviewDiagnostic[]>(() => {
  if (configurationMetadata.value?.diagnostics_summary?.length) {
    return configurationMetadata.value.diagnostics_summary.map((item, index) => ({
      key: `config-${index}`,
      message: item,
      theme: 'warning',
    }));
  }
  return [
    {
      key: 'healthy',
      message:
        unhealthyContainerCount.value > 0
          ? t('project.detail.overview.diagnosticUnhealthy', { count: unhealthyContainerCount.value })
          : t('project.detail.overview.diagnosticHealthy'),
      theme: unhealthyContainerCount.value > 0 ? 'warning' : 'success',
    },
    {
      key: 'restart',
      message:
        totalRestartCount.value > 0
          ? t('project.detail.overview.diagnosticRestartWarning', { count: totalRestartCount.value })
          : t('project.detail.overview.diagnosticRestartClear'),
      theme: totalRestartCount.value > 0 ? 'warning' : 'success',
    },
    {
      key: 'drift',
      message:
        detailRecord.value?.drift_status === 'clean'
          ? t('project.detail.overview.diagnosticConfigSynced')
          : t('project.detail.overview.diagnosticConfigDrift', {
              status: driftStatusLabel(detailRecord.value?.drift_status ?? 'unknown'),
            }),
      theme: detailRecord.value?.drift_status === 'clean' ? 'success' : 'info',
    },
  ];
});
const serviceColumns = computed<TableProps['columns']>(() => [
  {
    align: 'center',
    colKey: 'row-select',
    fixed: 'left',
    title: t('project.list.columns.selection'),
    type: 'multiple',
    width: 48,
  },
  createMainTextColumn(t('project.detail.services.columns.service'), 'name', 220),
  createStatusColumn(t('project.detail.services.columns.status'), 'status', 120),
  createTextColumn(t('project.detail.services.columns.image'), 'image', { minWidth: 220 }),
  createStatusColumn(t('project.detail.services.columns.health'), 'health', 120),
  createTextColumn(t('project.detail.services.columns.ports'), 'ports', { minWidth: 220 }),
  createActionColumn(t('project.detail.services.columns.operation'), 136),
]);

onMounted(async () => {
  await refreshDetail();
  syncApplicationRuntimeRealtimeSubscription();
  syncApplicationLifecycleConfigRealtimeSubscription();
  syncApplicationLogsRealtimeSubscription();
});

onUnmounted(() => {
  releaseApplicationRuntimeRealtimeSubscription();
  releaseApplicationLifecycleConfigRealtimeSubscription();
  releaseApplicationLogsRealtimeSubscription();
  projectRuntimeRealtimeGate.dispose();
  projectLifecycleConfigRealtimeGate.dispose();
  projectLogRealtimeBatcher.destroy();
});

watch(
  () => route.query.tab,
  (value) => {
    const nextTab = normalizeDetailTab(value);
    if (activeDetailTab.value !== nextTab) {
      activeDetailTab.value = nextTab;
    }
  },
);

watch(activeDetailTab, (value) => {
  const currentTab = normalizeDetailTab(route.query.tab);
  if (currentTab !== value) {
    void router.replace({
      query: {
        ...route.query,
        tab: value,
      },
    });
  }

  syncApplicationLifecycleConfigRealtimeSubscription();
  syncApplicationLogsRealtimeSubscription();
});

watch(
  () => [serviceTableRows.value.length, serviceTablePageSize.value],
  () => {
    const maxPage = Math.max(1, Math.ceil(serviceTableRows.value.length / serviceTablePageSize.value));
    if (serviceTableCurrent.value > maxPage) {
      serviceTableCurrent.value = maxPage;
    }
  },
);

watch(serviceTableRows, (rows) => {
  const availableKeys = new Set(rows.map((row) => row.service_name));
  selectedServiceRowKeys.value = selectedServiceRowKeys.value.filter((key) => availableKeys.has(String(key)));
});

watch(applicationId, () => {
  resetApplicationLogsState();
  lifecycleBaseline.value = null;
  lifecycleRemoteStale.value = false;
  projectRuntimeRealtimeGate.clear();
  projectLifecycleConfigRealtimeGate.clear();
  releaseApplicationRuntimeRealtimeSubscription();
  releaseApplicationLifecycleConfigRealtimeSubscription();
  releaseApplicationLogsRealtimeSubscription();
  syncApplicationRuntimeRealtimeSubscription();
  syncApplicationLifecycleConfigRealtimeSubscription();
  syncApplicationLogsRealtimeSubscription();
});

function driftStatusLabel(value: ApplicationDetailResponseWithLifecycle['drift_status']) {
  return projectDriftStatusLabel(t, value);
}

function driftStatusTheme(value?: ApplicationDetailResponseWithLifecycle['drift_status']) {
  return projectDriftStatusTheme(value);
}

function runtimeStatusLabel(value?: ApplicationDetailResponseWithLifecycle['runtime_status'] | null) {
  return projectRuntimeStatusLabel(t, value);
}

function overviewServiceStatusLabel(value?: ApplicationOverviewServiceItem['status']) {
  if (value === 'degraded') return t('project.detail.overview.serviceStatusDegraded');
  if (value === 'running') return t('project.detail.overview.serviceStatusRunning');
  return t('project.detail.overview.serviceStatusStopped');
}

function overviewServiceStatusTheme(
  value?: ApplicationOverviewServiceItem['status'],
): ServiceSnapshotCard['statusTheme'] {
  if (value === 'degraded') return 'warning';
  if (value === 'running') return 'success';
  return 'default';
}

function overviewServiceHealthLabel(value?: ApplicationOverviewServiceItem['health']) {
  if (value === 'attention') return t('project.detail.overview.serviceHealthAttention');
  if (value === 'healthy') return t('project.detail.overview.serviceHealthHealthy');
  return t('project.detail.overview.serviceHealthUnknown');
}

function overviewServiceHealthTheme(
  value?: ApplicationOverviewServiceItem['health'],
): ServiceSnapshotCard['healthTheme'] {
  if (value === 'attention') return 'warning';
  if (value === 'healthy') return 'success';
  return 'default';
}

function assignLifecycleDraft(
  target: ApplicationLifecycleConfigurationDraft,
  nextConfig: ApplicationLifecycleConfigurationDraft,
) {
  Object.assign(target, {
    ...nextConfig,
    compose_files: [...nextConfig.compose_files],
    profiles: [...nextConfig.profiles],
  });
}

function syncLifecycleState(
  detail: ApplicationDetailResponseWithLifecycle,
  options: {
    preserveDirtyDraft: boolean;
  } = { preserveDirtyDraft: false },
) {
  const preserveDirtyDraft =
    options.preserveDirtyDraft &&
    lifecycleBaseline.value !== null &&
    isLifecycleDraftDirty(lifecycleDraft, lifecycleBaseline.value);
  const nextConfig = buildLifecycleConfigurationDraft(detail);
  lifecycleBaseline.value = nextConfig;

  if (preserveDirtyDraft) {
    lifecycleRemoteStale.value = true;
    return;
  }

  assignLifecycleDraft(lifecycleDraft, nextConfig);
  lifecycleRemoteStale.value = false;
}

async function refreshDetail() {
  if (!applicationId.value) {
    detailError.value = t('project.list.retry');
    return;
  }
  detailLoading.value = true;
  detailError.value = '';
  try {
    detailRecord.value = await getApplication(applicationId.value);
    syncLifecycleState(detailRecord.value);
    updateCurrentTabTitle(buildDetailTitle(detailRecord.value.display_name));
    await Promise.all([
      loadApplicationRuntimeTarget(detailRecord.value.runtime_target?.id),
      loadConfigurationSummary(),
      loadApplicationServices(true),
      loadApplicationOverview(true),
    ]);
    if (activeDetailTab.value === 'logs' && projectLogsHasSnapshot.value) {
      await loadApplicationLogs();
    }
    syncApplicationRuntimeRealtimeSubscription();
    syncApplicationLifecycleConfigRealtimeSubscription();
    syncApplicationLogsRealtimeSubscription();
  } catch (error) {
    logger.error('failed to load project detail', error);
    detailRecord.value = null;
    runtimeTargetDetail.value = null;
    resetApplicationLogsState();
    projectOverview.value = null;
    projectOverviewLoaded.value = false;
    projectLogError.value = '';
    detailError.value = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
  } finally {
    detailLoading.value = false;
  }
}

async function loadConfigurationSummary() {
  if (!applicationId.value) return;
  try {
    configurationMetadata.value = await getApplicationConfiguration(applicationId.value);
  } catch (error) {
    logger.error('failed to load project configuration', error);
    configurationMetadata.value = null;
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.retry')));
  }
}

async function loadApplicationLogs() {
  if (!applicationId.value) return;
  const requestSequence = ++projectLogsLoadSequence;
  const currentApplicationId = applicationId.value;
  projectLogRealtimeBatcher.beginSnapshot(projectLogTail.value);
  emitApplicationLogDebug('snapshot-request-started', {
    applicationId: currentApplicationId,
    requestSequence,
    since: projectLogSince.value.trim() || '1h',
    tail: projectLogTail.value,
  });
  projectLogLoading.value = true;
  projectLogError.value = '';
  try {
    const response = await getApplicationLogs(applicationId.value, {
      tail: projectLogTail.value,
      since: projectLogSince.value.trim() || '1h',
      timestamps: true,
      stdout: true,
      stderr: true,
    });
    if (requestSequence !== projectLogsLoadSequence || currentApplicationId !== applicationId.value) {
      emitApplicationLogDebug('snapshot-response-discarded', { currentApplicationId, requestSequence });
      return;
    }
    emitApplicationLogDebug('snapshot-response-received', {
      applicationId: currentApplicationId,
      requestSequence,
      entryCount: response.entries.length,
      tail: response.tail,
      truncated: response.truncated,
    });
    commitApplicationLogsSnapshot(response);
    projectLogsHasSnapshot.value = true;
    projectLogsBootstrapRequested.value = false;
    projectLogsRecoveryLoadRequested.value = false;
  } catch (error) {
    if (requestSequence !== projectLogsLoadSequence || currentApplicationId !== applicationId.value) {
      return;
    }
    logger.error('failed to load project logs', error);
    emitApplicationLogDebug('snapshot-request-failed', {
      applicationId: currentApplicationId,
      requestSequence,
      error: error instanceof Error ? error.message : String(error),
    });
    projectLogError.value = resolveLocalizedErrorMessage(t, error, t('project.detail.logs.loadFailed'));
  } finally {
    if (requestSequence === projectLogsLoadSequence && currentApplicationId === applicationId.value) {
      projectLogLoading.value = false;
      if (activeDetailTab.value === 'logs') {
        syncApplicationLogsRealtimeSubscription();
      }
    }
  }
}

async function loadApplicationServices(forceRefresh = false) {
  if (!applicationId.value) {
    serviceRows.value = [];
    serviceRuntimePortSummaries.value = {};
    servicesLoaded.value = false;
    return [];
  }
  if (servicesLoaded.value && !forceRefresh) {
    return serviceRows.value;
  }

  serviceLoading.value = true;
  try {
    const response = await getApplicationServices(applicationId.value);
    serviceRows.value = response.items;
    serviceTableCurrent.value = 1;
    await syncServiceRuntimePortSummaries(response.items);
    servicesLoaded.value = true;
    return response.items;
  } catch (error) {
    logger.error('failed to load project services', error);
    serviceRows.value = [];
    serviceRuntimePortSummaries.value = {};
    servicesLoaded.value = false;
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.services.loadFailed')));
    return [];
  } finally {
    serviceLoading.value = false;
  }
}

async function loadApplicationRuntimeTarget(runtimeTargetId?: number) {
  const requestId = runtimeTargetDetailRequestId + 1;
  runtimeTargetDetailRequestId = requestId;
  runtimeTargetDetail.value = null;
  if (!runtimeTargetId) return;

  try {
    const detail = await getApplicationRuntimeTargetDetail(runtimeTargetId);
    if (requestId === runtimeTargetDetailRequestId) {
      runtimeTargetDetail.value = detail;
    }
  } catch (error) {
    logger.warn('failed to load application runtime target detail', error);
  }
}

async function loadApplicationOverview(forceRefresh = false) {
  if (!applicationId.value) {
    projectOverview.value = null;
    projectOverviewLoaded.value = false;
    return null;
  }
  if (projectOverviewLoaded.value && !forceRefresh) {
    return projectOverview.value;
  }
  try {
    const response = await getApplicationOverview(applicationId.value);
    projectOverview.value = response;
    projectOverviewLoaded.value = true;
    return response;
  } catch (error) {
    logger.error('failed to load project overview', error);
    projectOverview.value = null;
    projectOverviewLoaded.value = false;
    return null;
  }
}

function resetApplicationLogsState() {
  projectLogResponse.value = null;
  pendingApplicationLogSnapshot = null;
  projectLogError.value = '';
  projectLogPaused.value = false;
  projectLogsHasSnapshot.value = false;
  projectLogsBootstrapRequested.value = false;
  projectLogsRecoveryLoadRequested.value = false;
  projectLogsLoadSequence += 1;
  projectLogRealtimeBatcher.clear();
}

function commitApplicationLogsSnapshot(response: ApplicationLogResponse) {
  emitApplicationLogDebug('snapshot-seed', {
    entryCount: response.entries.length,
    tail: response.tail,
    truncated: response.truncated,
  });
  projectLogRealtimeBatcher.seed(response);
}

function appendApplicationLogEntry(entry: ApplicationLogEntry) {
  projectLogRealtimeBatcher.enqueue(entry);
}

function applyApplicationLogRealtimeEntry(entry: ApplicationLogEntry) {
  appendApplicationLogEntry(entry);
}

function applyApplicationRuntimeRealtimeSnapshot(payload: {
  detail: ApplicationDetailResponseWithLifecycle;
  overview: ApplicationOverviewResponse;
  services: { items: ApplicationServiceItem[] };
}) {
  detailRecord.value = payload.detail;
  updateCurrentTabTitle(buildDetailTitle(payload.detail.display_name));
  detailError.value = '';
  serviceRows.value = payload.services.items;
  void syncServiceRuntimePortSummaries(payload.services.items);
  servicesLoaded.value = true;
  projectOverview.value = payload.overview;
  projectOverviewLoaded.value = true;
}

function applyApplicationLifecycleConfigRealtimeSnapshot(payload: { detail: ApplicationDetailResponseWithLifecycle }) {
  const nextDetail = mergeLifecycleConfigurationRealtimeDetail(payload.detail);
  detailRecord.value = nextDetail;
  syncLifecycleState(nextDetail, { preserveDirtyDraft: true });
}

function mergeLifecycleConfigurationRealtimeDetail(
  lifecycleDetail: ApplicationDetailResponseWithLifecycle,
): ApplicationDetailResponseWithLifecycle {
  const currentDetail = detailRecord.value;
  if (!currentDetail) {
    return lifecycleDetail;
  }

  return {
    ...lifecycleDetail,
    container_counts: currentDetail.container_counts,
    runtime_status: currentDetail.runtime_status,
    service_count: currentDetail.service_count,
  };
}

function syncApplicationRuntimeRealtimeSubscription() {
  const nextTopic = applicationId.value ? buildApplicationRuntimeTopicName(applicationId.value) : '';
  if (!nextTopic) {
    releaseApplicationRuntimeRealtimeSubscription();
    return;
  }
  if (projectRuntimeRealtimeTopic === nextTopic && projectRuntimeRealtimeController) {
    return;
  }
  releaseApplicationRuntimeRealtimeSubscription();
  projectRuntimeRealtimeTopic = nextTopic;
  projectRuntimeRealtimeController = openRealtimeTopicSocket({
    topic: nextTopic,
    parseMessage: parseApplicationRuntimeRealtimePayload,
    onMessage: (message) => {
      projectRuntimeRealtimeGate.commit(message);
    },
    onStateChange: (state) => {
      projectRuntimeSocketState.value = state;
    },
    onError: (message) => {
      logger.warn('project runtime realtime subscription error', { message, topic: nextTopic });
    },
  });
}

function releaseApplicationRuntimeRealtimeSubscription() {
  projectRuntimeRealtimeTopic = '';
  projectRuntimeRealtimeController?.close();
  projectRuntimeRealtimeController = null;
  projectRuntimeSocketState.value = 'idle';
}

function syncApplicationLifecycleConfigRealtimeSubscription() {
  const nextTopic =
    applicationId.value && activeDetailTab.value === 'lifecycle'
      ? buildApplicationLifecycleConfigTopicName(applicationId.value)
      : '';
  if (!nextTopic) {
    releaseApplicationLifecycleConfigRealtimeSubscription();
    return;
  }
  if (projectLifecycleConfigRealtimeTopic === nextTopic && projectLifecycleConfigRealtimeController) {
    return;
  }
  releaseApplicationLifecycleConfigRealtimeSubscription();
  projectLifecycleConfigRealtimeTopic = nextTopic;
  projectLifecycleConfigRealtimeController = openRealtimeTopicSocket({
    topic: nextTopic,
    parseMessage: parseApplicationLifecycleConfigRealtimePayload,
    onMessage: (message) => {
      projectLifecycleConfigRealtimeGate.commit(message);
    },
    onStateChange: (state) => {
      projectLifecycleConfigSocketState.value = state;
    },
    onError: (message) => {
      logger.warn('project lifecycle config realtime subscription error', { message, topic: nextTopic });
    },
  });
}

function releaseApplicationLifecycleConfigRealtimeSubscription() {
  projectLifecycleConfigRealtimeTopic = '';
  projectLifecycleConfigRealtimeController?.close();
  projectLifecycleConfigRealtimeController = null;
  projectLifecycleConfigSocketState.value = 'idle';
}

function syncApplicationLogsRealtimeSubscription() {
  const nextTopic =
    applicationId.value && activeDetailTab.value === 'logs' ? buildApplicationLogsTopicName(applicationId.value) : '';
  if (!nextTopic) {
    emitApplicationLogDebug('subscription-not-required', { activeTab: activeDetailTab.value });
    releaseApplicationLogsRealtimeSubscription();
    return;
  }
  if (!projectLogsHasSnapshot.value && !projectLogLoading.value && !projectLogsBootstrapRequested.value) {
    projectLogsBootstrapRequested.value = true;
    void loadApplicationLogs();
  }
  if (projectLogsRealtimeTopic === nextTopic && projectLogsRealtimeController) {
    emitApplicationLogDebug('subscription-reused', { topic: nextTopic });
    return;
  }
  releaseApplicationLogsRealtimeSubscription();
  const subscriptionSequence = ++projectLogsSubscriptionSequence;
  projectLogsRealtimeTopic = nextTopic;
  emitApplicationLogDebug('subscription-opening', { subscriptionSequence, topic: nextTopic });
  projectLogsRealtimeController = openRealtimeTopicSocket({
    topic: nextTopic,
    parseMessage: parseApplicationLogsRealtimePayload,
    onMessage: (message) => {
      if (subscriptionSequence !== projectLogsSubscriptionSequence || projectLogsRealtimeTopic !== nextTopic) {
        return;
      }
      emitApplicationLogDebug('subscription-entry-received', {
        subscriptionSequence,
        topic: nextTopic,
        socketState: projectLogsSocketState.value,
      });
      applyApplicationLogRealtimeEntry(message.entry);
    },
    onStateChange: (state) => {
      if (subscriptionSequence !== projectLogsSubscriptionSequence || projectLogsRealtimeTopic !== nextTopic) {
        return;
      }
      projectLogsSocketState.value = state;
      emitApplicationLogDebug('subscription-state-changed', {
        hasSnapshot: projectLogsHasSnapshot.value,
        loading: projectLogLoading.value,
        state,
        subscriptionSequence,
        topic: nextTopic,
      });
      if (state === 'open') {
        projectLogsRecoveryLoadRequested.value = false;
      }
      if (
        activeDetailTab.value === 'logs' &&
        !projectLogsHasSnapshot.value &&
        !projectLogLoading.value &&
        (state === 'closed' || state === 'error') &&
        !projectLogsRecoveryLoadRequested.value
      ) {
        projectLogsRecoveryLoadRequested.value = true;
        void loadApplicationLogs();
      }
    },
    onError: (message) => {
      emitApplicationLogDebug('subscription-error', { message, topic: nextTopic });
      logger.warn('project log realtime subscription error', { message, topic: nextTopic });
    },
  });
}

function releaseApplicationLogsRealtimeSubscription() {
  emitApplicationLogDebug('subscription-releasing', { topic: projectLogsRealtimeTopic });
  projectLogsSubscriptionSequence += 1;
  projectLogRealtimeBatcher.flush();
  projectLogsRealtimeTopic = '';
  projectLogsRealtimeController?.close();
  projectLogsRealtimeController = null;
  projectLogsSocketState.value = 'idle';
}

async function refreshApplicationServices() {
  try {
    await loadApplicationServices(true);
  } catch {
    // loadApplicationServices 已负责提示失败；刷新入口只需保持详情页可继续交互。
  }
}

function openFirstServiceContainer(service: ApplicationServiceItem) {
  const member = resolveServiceDetailMember(service);
  if (!member) {
    return;
  }
  openContainerDetail(member);
}

function handleServiceSelectChange(rowKeys: Array<string | number>) {
  const currentPageKeys = new Set(pagedServiceTableRows.value.map((row) => row.service_name));
  const preservedKeys = selectedServiceRowKeys.value.filter((key) => !currentPageKeys.has(String(key)));
  const normalizedCurrentKeys = rowKeys.filter((key) => {
    const row = serviceRowMap.value.get(String(key));
    return currentPageKeys.has(String(key)) && row?.raw.managed !== false;
  });
  selectedServiceRowKeys.value = [...preservedKeys, ...normalizedCurrentKeys];
}

function toggleServiceSelection(serviceName: string, checked: boolean) {
  if (serviceRowMap.value.get(serviceName)?.raw.managed === false) {
    return;
  }
  const currentPageKeys = new Set(pagedServiceTableRows.value.map((row) => row.service_name));
  const nextCurrentKeys = new Set(
    selectedServiceRowKeys.value.filter((key) => currentPageKeys.has(String(key))).map(String),
  );

  if (checked) {
    nextCurrentKeys.add(serviceName);
  } else {
    nextCurrentKeys.delete(serviceName);
  }

  handleServiceSelectChange(Array.from(nextCurrentKeys));
}

function clearSelectedServices() {
  selectedServiceRowKeys.value = [];
}

function canRunServiceContainerAction(
  row: Pick<ServiceTableRow, 'hasMembers' | 'runningCount' | 'raw'>,
  action: ProjectContainerAction,
) {
  if (row.raw.managed === false || !row.hasMembers || !runtimeTargetCapabilityAvailable('container_execution')) {
    return false;
  }
  if (action === 'start') {
    return row.runningCount === 0;
  }
  return row.runningCount > 0;
}

function runtimeTargetCapabilityAvailable(capability: string) {
  return (
    runtimeTargetDetail.value?.agent.capabilities.some((item) => item.name === capability && item.status === 'ready') ??
    false
  );
}

function isServiceBatchActionEligible(row: ServiceTableRow, action: ProjectContainerAction) {
  return canRunServiceContainerAction(row, action);
}

function serviceBatchActionableRows(action: ProjectContainerAction) {
  return selectedServiceRows.value.filter((row) => isServiceBatchActionEligible(row, action));
}

function isServiceBatchActionDisabled(action: ProjectContainerAction) {
  return serviceActionBusy.value || serviceBatchActionableRows(action).length === 0;
}

const serviceBatchActionOptions = computed<DropdownProps['options']>(() => [
  {
    content: t('project.detail.services.batch.start'),
    disabled: isServiceBatchActionDisabled('start'),
    theme: 'success',
    value: 'start',
  },
  {
    content: t('project.detail.services.batch.stop'),
    disabled: isServiceBatchActionDisabled('stop'),
    theme: 'warning',
    value: 'stop',
  },
  {
    content: t('project.detail.services.batch.restart'),
    disabled: isServiceBatchActionDisabled('restart'),
    value: 'restart',
  },
  {
    content: t('project.detail.services.batch.cancelSelection'),
    divider: true,
    value: 'clear',
  },
]);

const handleServiceBatchMenuAction: NonNullable<DropdownProps['onClick']> = (item) => {
  const action = typeof item === 'object' && item ? item.value : item;
  if (action === 'clear') {
    clearSelectedServices();
    return;
  }

  if (action === 'start' || action === 'stop' || action === 'restart') {
    confirmServiceBatchAction(action);
  }
};

function serviceActionOptions(row: ServiceTableRow) {
  const rowLoading = serviceActionBusy.value || serviceActionKey.value.startsWith(`${row.service_name}:`);
  const capabilityReady = runtimeTargetCapabilityAvailable('container_execution');
  const actions: Array<{ disabled?: boolean; label: string; value: ServiceRowAction }> = [
    {
      disabled: rowLoading || !row.hasMembers,
      label: 'components.commonTable.detail',
      value: 'detail',
    },
  ];

  if (row.hasMembers && row.runningCount === 0) {
    actions.push({
      disabled: rowLoading || row.raw.managed === false || !capabilityReady,
      label: 'project.detail.services.actions.start',
      value: 'start',
    });
  }

  if (row.hasMembers && row.runningCount > 0) {
    actions.push(
      {
        disabled: rowLoading || row.raw.managed === false || !capabilityReady,
        label: 'project.detail.services.actions.stop',
        value: 'stop',
      },
      {
        disabled: rowLoading || row.raw.managed === false || !capabilityReady,
        label: 'project.detail.services.actions.restart',
        value: 'restart',
      },
    );
  }

  return actions;
}

async function handleServiceAction(action: string, row: ServiceTableRow) {
  if (action === 'detail') {
    openFirstServiceContainer(row.raw);
    return;
  }

  if (action !== 'start' && action !== 'stop' && action !== 'restart') {
    return;
  }

  await runServiceContainerAction(action, [row.raw], `${row.service_name}:${action}`);
}

function confirmServiceBatchAction(action: ProjectContainerAction) {
  if (isServiceBatchActionDisabled(action)) {
    MessagePlugin.warning(t('project.detail.services.batch.noSelection'));
    return;
  }

  const actionableRows = serviceBatchActionableRows(action);
  let idempotencyKey: string;
  try {
    idempotencyKey = createServiceBatchIdempotencyKey(action);
  } catch (error) {
    logger.warn(`failed to create ${action} service batch idempotency key`, error);
    MessagePlugin.error(t('project.detail.services.batch.failed'));
    return;
  }
  const dialog = DialogPlugin.confirm({
    header: t(`project.detail.services.batch.confirm${capitalizeAction(action)}Title`),
    body: t(`project.detail.services.batch.confirm${capitalizeAction(action)}`, {
      count: actionableRows.length,
    }),
    confirmBtn: t('project.list.actions.confirm'),
    cancelBtn: t('project.list.actions.cancel'),
    theme: action === 'start' ? 'warning' : 'danger',
    onConfirm: async () => {
      dialog.setConfirmLoading(true);
      try {
        const submitted = await runServiceContainerAction(
          action,
          actionableRows.map((row) => row.raw),
          `batch:${action}`,
          idempotencyKey,
        );
        if (submitted) {
          dialog.destroy();
        }
      } finally {
        dialog.setConfirmLoading(false);
      }
    },
  });
}

async function runServiceContainerAction(
  action: ProjectContainerAction,
  services: ApplicationServiceItem[],
  actionKey: string,
  idempotencyKey = createServiceBatchIdempotencyKey(action),
) {
  const ids = Array.from(
    new Set(
      services.flatMap((service) => service.container_members.map((member) => member.container_id).filter(Boolean)),
    ),
  );
  if (!ids.length) {
    return;
  }

  if (actionKey.startsWith('batch:')) {
    serviceBatchActionLoading.value = action;
  } else {
    serviceActionKey.value = actionKey;
  }
  try {
    const response = await batchContainerActions(
      {
        action,
        force: false,
        ids,
      } satisfies ProjectContainerActionSubmission,
      idempotencyKey,
    );
    handleServiceBatchActionResult(response);
    try {
      await refreshApplicationRuntimeSurface();
    } catch (error) {
      logger.warn('failed to refresh project runtime surface after service action', error);
      MessagePlugin.warning(t('project.detail.services.batch.refreshWarning'));
    }
    return true;
  } catch (error) {
    logger.warn(`failed to ${action} service containers`, error);
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.services.batch.failed')));
    return false;
  } finally {
    if (actionKey.startsWith('batch:')) {
      serviceBatchActionLoading.value = '';
    } else {
      serviceActionKey.value = '';
    }
  }
}

function createServiceBatchIdempotencyKey(action: ProjectContainerAction) {
  const crypto = globalThis.crypto;
  const uuid = crypto?.randomUUID?.();
  if (uuid) return uuid;

  if (!crypto?.getRandomValues) {
    throw new Error('Web Crypto API is unavailable');
  }
  const entropy = new Uint32Array(4);
  crypto.getRandomValues(entropy);
  serviceBatchIdempotencySequence += 1;
  return `project-service-batch-${action}-${Date.now()}-${serviceBatchIdempotencySequence}-${Array.from(
    entropy,
    (value) => value.toString(36),
  ).join('')}`;
}

function handleServiceBatchActionResult(response: ProjectContainerActionResult) {
  if (response.failed_count === 0) {
    MessagePlugin.success(t('project.detail.services.batch.success', { count: response.accepted_count }));
    return;
  }

  if (response.accepted_count > 0) {
    void NotifyPlugin.warning({
      closeBtn: true,
      content: batchFailureSummary(response.items),
      duration: 0,
      title: t('project.detail.services.batch.partialTitle'),
    });
    return;
  }

  MessagePlugin.error(t('project.detail.services.batch.failed'));
  DialogPlugin.alert({
    body: batchFailureSummary(response.items),
    confirmBtn: t('project.list.actions.confirm'),
    header: t('project.detail.services.batch.failureDetailTitle'),
    theme: 'danger',
  });
}

function batchFailureSummary(items: ProjectContainerActionResultItem[]) {
  const failedItems = items.filter((item) => !item.accepted);
  if (!failedItems.length) {
    return t('project.detail.services.batch.noFailureDetail');
  }

  return failedItems
    .slice(0, 5)
    .map((item) => `${item.name || item.id}: ${item.message_key ? t(item.message_key) : item.message || '-'}`)
    .join('\n');
}

function capitalizeAction(action: ProjectContainerAction) {
  return `${action.charAt(0).toUpperCase()}${action.slice(1)}`;
}

async function refreshApplicationRuntimeSurface() {
  if (!applicationId.value) {
    return;
  }

  const nextDetail = await getApplication(applicationId.value);
  detailRecord.value = nextDetail;
  syncLifecycleState(nextDetail, { preserveDirtyDraft: true });
  updateCurrentTabTitle(buildDetailTitle(nextDetail.display_name));
  await Promise.all([loadApplicationServices(true), loadApplicationOverview(true)]);
  syncApplicationRuntimeRealtimeSubscription();
  syncApplicationLifecycleConfigRealtimeSubscription();
  syncApplicationLogsRealtimeSubscription();
}

function pauseApplicationLogs() {
  projectLogPaused.value = true;
}

function resumeApplicationLogs() {
  projectLogPaused.value = false;
  if (pendingApplicationLogSnapshot) {
    projectLogResponse.value = pendingApplicationLogSnapshot;
    pendingApplicationLogSnapshot = null;
    projectLogContentVersion.value += 1;
  }
}

function clearApplicationLogs() {
  projectLogPaused.value = false;
  pendingApplicationLogSnapshot = null;
  projectLogRealtimeBatcher.clearView();
}

function updateApplicationLogTail(value: number) {
  if (![100, 200, 500, 1000].includes(value) || projectLogTail.value === value) {
    return;
  }
  projectLogTail.value = value;
  void loadApplicationLogs();
}

async function runLifecycleAction(action: 'up' | 'stop' | 'restart' | 'redeploy' | 'unregister') {
  if (!applicationId.value) return;
  if (action === 'unregister' && !(await confirmDangerousAction('unregister'))) {
    return;
  }
  if (action !== 'unregister' && lifecycleReviewRequired.value) {
    MessagePlugin.warning(t('project.detail.lifecycle.reviewRequiredActionBlocked'));
    return;
  }
  actionLoading.value = action;
  try {
    let receipt: { task_id: number } | null = null;
    if (action === 'up') {
      receipt = await postApplicationUp(applicationId.value);
    } else if (action === 'stop') {
      receipt = await postApplicationStop(applicationId.value);
    } else if (action === 'restart') {
      receipt = await postApplicationRestart(applicationId.value);
    } else if (action === 'redeploy') {
      receipt = await postApplicationRedeploy(applicationId.value);
    } else {
      await postApplicationUnregister(applicationId.value);
    }
    if (receipt) {
      openTaskDrawer(receipt.task_id);
      MessagePlugin.success(t('project.list.actions.taskAccepted'));
    } else {
      MessagePlugin.success(t('project.list.actions.actionSuccess'));
    }
    await refreshDetail();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.actions.actionFailed')));
  } finally {
    actionLoading.value = '';
  }
}

function openTaskDrawer(taskId: number) {
  activeTaskId.value = taskId;
  taskDrawerVisible.value = true;
}

function isDeleteWorkspacePathAllowed() {
  return detailRecord.value?.ownership_mode !== 'external';
}

function confirmDangerousAction(action: 'unregister' | 'destroy') {
  if (!detailRecord.value) {
    return Promise.resolve(false);
  }

  return new Promise<boolean>((resolve) => {
    const deleteWorkspacePath = ref(false);
    const autoUnregister = ref(action === 'destroy' ? false : true);
    const removeNamedVolumes = ref(false);
    const row = detailRecord.value!;
    let settled = false;

    const finish = (confirmed: boolean) => {
      if (settled) {
        return;
      }
      settled = true;
      dialog.destroy();
      resolve(confirmed);
    };

    const dialog = DialogPlugin.confirm({
      header: t(`project.list.actions.confirm${action.charAt(0).toUpperCase()}${action.slice(1)}Title`),
      body: () =>
        h('div', { class: 'project-action-confirm' }, [
          h(
            'p',
            t(`project.list.actions.confirm${action.charAt(0).toUpperCase()}${action.slice(1)}Description`, {
              name: row.display_name,
            }),
          ),
          action === 'destroy'
            ? h('div', { class: 'project-action-confirm__danger' }, [
                h('label', { class: 'project-action-confirm__option' }, [
                  h('input', {
                    checked: removeNamedVolumes.value,
                    type: 'checkbox',
                    onInput: (event: Event) => {
                      removeNamedVolumes.value = (event.target as HTMLInputElement).checked;
                    },
                  }),
                  h('span', t('project.list.actions.destroyDeleteVolumes')),
                ]),
                h('label', { class: 'project-action-confirm__option' }, [
                  h('input', {
                    checked: autoUnregister.value,
                    type: 'checkbox',
                    onInput: (event: Event) => {
                      autoUnregister.value = (event.target as HTMLInputElement).checked;
                    },
                  }),
                  h('span', t('project.list.actions.destroyAutoUnregister')),
                ]),
                h('label', { class: 'project-action-confirm__option' }, [
                  h('input', {
                    checked: deleteWorkspacePath.value,
                    disabled: !isDeleteWorkspacePathAllowed(),
                    type: 'checkbox',
                    onInput: (event: Event) => {
                      deleteWorkspacePath.value = (event.target as HTMLInputElement).checked;
                      if (deleteWorkspacePath.value) {
                        autoUnregister.value = true;
                      }
                    },
                  }),
                  h('span', t('project.list.actions.destroyDeleteApplicationFiles')),
                ]),
              ])
            : null,
        ]),
      theme: 'danger',
      confirmBtn: {
        content: t('project.list.actions.confirm'),
        theme: 'danger',
      },
      cancelBtn: t('project.list.actions.cancel'),
      onCancel: () => finish(false),
      onClose: () => finish(false),
      onConfirm: async () => {
        if (action === 'unregister') {
          finish(true);
          return;
        }

        await runDestroy({
          auto_unregister: autoUnregister.value || deleteWorkspacePath.value,
          confirm_application_id: row.application_id,
          delete_workspace: deleteWorkspacePath.value,
          image_prune: false,
          remove_named_volumes: removeNamedVolumes.value,
        });
        finish(true);
      },
    });
  });
}

async function runDestroy(payload: ApplicationDestroyRequest) {
  if (!applicationId.value) {
    return;
  }

  try {
    await postApplicationDestroy(applicationId.value, payload);
    MessagePlugin.success(t('project.list.actions.actionSuccess'));
    await refreshDetail();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.actions.actionFailed')));
  }
}

async function runDestroyAction() {
  if (!(await confirmDangerousAction('destroy'))) {
    return;
  }
}

async function saveLifecycleConfiguration() {
  if (!applicationId.value || !detailRecord.value) {
    return;
  }
  if (
    lifecycleDraft.wait_after_up &&
    (lifecycleDraft.wait_timeout_seconds < 1 || lifecycleDraft.wait_timeout_seconds > 3600)
  ) {
    MessagePlugin.warning(t('project.detail.lifecycle.waitTimeoutValidation'));
    return;
  }
  lifecycleSaveLoading.value = true;
  try {
    const response = await putApplicationLifecycleConfiguration(
      applicationId.value,
      buildLifecycleConfigurationRequest(lifecycleDraft),
    );
    detailRecord.value = {
      ...detailRecord.value,
      lifecycle_review_status: response.lifecycle_review_status,
      lifecycle_configuration: response.lifecycle_configuration,
      workspace_path: response.workspace_path,
      compose_project_name: response.compose_project_name,
      compose_files: response.compose_files,
    };
    syncLifecycleState(detailRecord.value, { preserveDirtyDraft: false });
    MessagePlugin.success(t('project.detail.lifecycle.saveSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.lifecycle.saveFailed')));
  } finally {
    lifecycleSaveLoading.value = false;
  }
}

function resetLifecycleConfiguration() {
  if (!lifecycleBaseline.value) {
    return;
  }
  assignLifecycleDraft(lifecycleDraft, lifecycleBaseline.value);
  lifecycleRemoteStale.value = false;
}

async function copyPath(path: string) {
  try {
    await copyText(path);
    MessagePlugin.success(t('project.detail.actions.copyPathSuccess'));
  } catch {
    MessagePlugin.error(t('project.detail.actions.copyPathError'));
  }
}

function joinList(items: string[]) {
  return items.length > 0 ? items.join(', ') : '-';
}

function formatRuntimePortLabel(port: ProjectContainerSummary['ports'][number]) {
  if (typeof port.public_port !== 'number') {
    return '';
  }
  const host = port.ip && port.ip !== '0.0.0.0' && port.ip !== '::' ? `${port.ip}:` : '';
  return `${host}${port.public_port}:${port.private_port} ${port.type.toUpperCase()}`;
}

function buildServiceRuntimePortSummaries(
  services: ApplicationServiceItem[],
  containers: ProjectContainerSummary[],
): Record<string, string> {
  const labelsByService = new Map<string, Set<string>>();

  for (const service of services) {
    labelsByService.set(service.service_name, new Set());
  }

  for (const container of containers) {
    const serviceName = readProjectContainerSourceMember(container);
    if (!serviceName || !labelsByService.has(serviceName)) {
      continue;
    }
    const portLabels = container.ports.map(formatRuntimePortLabel).filter(Boolean);
    const labelSet = labelsByService.get(serviceName);
    if (!labelSet) {
      continue;
    }
    for (const label of portLabels) {
      labelSet.add(label);
    }
  }

  return Object.fromEntries(
    Array.from(labelsByService.entries()).map(([serviceName, labels]) => [serviceName, joinList(Array.from(labels))]),
  );
}

async function syncServiceRuntimePortSummaries(services: ApplicationServiceItem[]) {
  const requestId = serviceRuntimePortsRequestId.value + 1;
  serviceRuntimePortsRequestId.value = requestId;

  const composeProjectName = (detailRecord.value?.compose_project_name || fallbackCanonicalName.value).trim();
  if (!composeProjectName || services.length === 0) {
    if (requestId === serviceRuntimePortsRequestId.value) {
      serviceRuntimePortSummaries.value = {};
    }
    return;
  }

  try {
    const containers = await fetchApplicationRuntimeContainers(composeProjectName);
    if (requestId !== serviceRuntimePortsRequestId.value) {
      return;
    }
    serviceRuntimePortSummaries.value = buildServiceRuntimePortSummaries(services, containers);
  } catch (error) {
    logger.warn('failed to load runtime ports for project services', error);
    if (requestId === serviceRuntimePortsRequestId.value) {
      serviceRuntimePortSummaries.value = {};
    }
  }
}

function buildDetailTitle(name: string): LocalizedTitle {
  return buildDetailTitleWithFallback('project.route.detail.title', name);
}

function buildConfigurationWorkspaceTitle(name: string): LocalizedTitle {
  return buildDetailTitleWithFallback('project.route.configurationWorkspace.title', name);
}

function normalizeDetailTab(value: unknown): ApplicationDetailTab {
  const raw = Array.isArray(value) ? value[0] : value;
  const tabs: ApplicationDetailTab[] = ['overview', 'services', 'logs', 'lifecycle', 'tasks'];
  return typeof raw === 'string' && tabs.includes(raw as ApplicationDetailTab)
    ? (raw as ApplicationDetailTab)
    : 'overview';
}

function readNameFromTabTitle(title?: LocalizedTitle) {
  if (!title) return '';
  const current = title[locale.value as keyof LocalizedTitle] || title[LOCALE.ZH_CN] || title[LOCALE.EN_US] || '';
  const parts = current.split(' - ');
  return parts.length > 1 ? parts.slice(1).join(' - ').trim() : '';
}

function updateCurrentTabTitle(title: LocalizedTitle) {
  tabsRouterStore.updateActiveTabTitle(PROJECT_BOOTSTRAP_ROUTE.DETAIL.pageRouteName, route, title);
}

function openConfigurationWorkspace() {
  const target = {
    name: PROJECT_BOOTSTRAP_ROUTE.CONFIGURATION_WORKSPACE.pageRouteName,
    params: { applicationId: applicationId.value },
    query: fallbackDisplayName.value ? { name: fallbackDisplayName.value } : undefined,
  };
  const resolved = router.resolve(target);
  appendResolvedTab(tabsRouterStore, resolved, buildConfigurationWorkspaceTitle(pageTitle.value));
  void router.push(target);
}

function resolveServiceDetailMember(service: ApplicationServiceItem) {
  return service.container_members.find((member) => member.state === 'running') ?? service.container_members[0];
}

function openContainerDetail(member: ApplicationServiceContainerMember) {
  const target = {
    name: CONTAINER_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    params: { id: member.container_id },
    query: { name: member.container_name },
  };
  const resolved = router.resolve(target);
  appendResolvedTab(
    tabsRouterStore,
    resolved,
    buildDetailTitleWithFallback('container.route.detail.title', member.container_name),
  );
  void router.push(target);
}
</script>
<style scoped lang="less">
.project-detail-page,
.project-detail-body,
.project-tab-panel,
.project-overview-grid,
.project-overview-hero__body,
.project-overview-hero__copy,
.project-overview-hero__actions,
.project-overview-hero__stats,
.project-overview-stat-list,
.project-overview-metric-list,
.project-overview-activity-list,
.project-runtime-grid,
.project-configuration-grid,
.project-file-groups,
.project-detail-copy-row,
.project-service-name,
.project-service-snapshot-grid,
.project-service-card,
.project-service-card__head,
.project-service-card__tags,
.project-activity-card,
.project-activity-card__head,
.project-activity-grid,
.project-activity-grid__title,
.project-activity-entries {
  display: flex;
}

.project-detail-page {
  container-type: inline-size;
}

.project-detail-page,
.project-detail-body,
.project-tab-panel,
.project-service-name,
.project-overview-hero__copy,
.project-overview-hero__actions,
.project-overview-stat-list,
.project-overview-metric-list,
.project-overview-activity-list,
.project-service-card {
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.project-overview-grid,
.project-runtime-grid,
.project-configuration-grid,
.project-activity-grid,
.project-service-snapshot-grid {
  gap: var(--graft-density-gap-16);
}

.project-overview-grid,
.project-runtime-grid,
.project-configuration-grid,
.project-service-snapshot-grid {
  align-items: stretch;
}

.project-overview-grid > .t-card,
.project-runtime-grid > .t-card,
.project-configuration-grid > .t-card {
  flex: 1 1 0;
  min-width: 0;
}

.project-overview-grid__wide {
  flex-basis: 100%;
}

.project-detail-tabs :deep(.t-tabs__content) {
  padding: var(--graft-density-gap-16);
}

.project-detail-actions--compact {
  display: none;
}

.project-section,
.project-lifecycle-bar {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.project-section-heading {
  align-items: flex-start;
  display: flex;
  justify-content: space-between;
}

.project-section-heading h2,
.project-section-heading p {
  margin: 0;
}

.project-section-heading h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.project-section-heading p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin-top: var(--graft-density-gap-4);
}

.project-detail-action-bar,
.project-detail-copy-row,
.project-activity-grid__title,
.project-overview-hero__heading,
.project-overview-hero__body,
.project-service-card__head,
.project-service-card__tags,
.project-overview-hero {
  border-color: color-mix(in srgb, var(--td-brand-color-6) 28%, var(--td-border-level-1-color));
}

.project-overview-hero__body {
  align-items: flex-start;
  justify-content: space-between;
}

.project-overview-hero__heading h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-headline-medium);
  margin: 0;
}

.project-overview-hero__copy p,
.project-overview-hero__updated-at,
.project-overview-stat-item small,
.project-overview-metric-item p,
.project-service-card__head p,
.project-overview-activity-item p {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.project-overview-hero__stats {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  width: 100%;
}

.project-overview-hero__stats article,
.project-overview-stat-item,
.project-overview-metric-item,
.project-service-card,
.project-overview-activity-item {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  padding: var(--graft-density-gap-12);
}

.project-overview-hero__stats span,
.project-overview-stat-item span,
.project-overview-metric-item span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-overview-hero__stats strong,
.project-overview-stat-item strong,
.project-overview-metric-item strong,
.project-overview-activity-item strong {
  color: var(--td-text-color-primary);
  display: block;
  font: var(--td-font-title-medium);
}

.project-overview-stat-list {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
}

.project-overview-metric-item header,
.project-overview-activity-item header {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
}

.project-overview-meter {
  background: var(--td-gray-color-3);
  border-radius: 999px;
  height: 8px;
  overflow: hidden;
}

.project-overview-meter span {
  background: linear-gradient(
    90deg,
    var(--td-brand-color-6),
    color-mix(in srgb, var(--td-brand-color-6) 62%, var(--td-success-color-5))
  );
  display: block;
  height: 100%;
}

.project-service-snapshot-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
}

.project-service-card__head {
  align-items: flex-start;
  justify-content: space-between;
}

.project-service-card__metrics {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.project-service-card__metrics dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-service-card__metrics dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  margin: var(--graft-density-gap-4) 0 0;
}

.project-overview-resource-card,
.project-network-io-card,
.project-live-metric-card,
.project-logs-toolbar {
  display: flex;
}

.project-overview-resource-card,
.project-network-io-card {
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.project-live-metric-card {
  align-items: center;
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
  overflow: hidden;
  padding: var(--graft-density-gap-16);
  position: relative;
}

.project-live-metric-card--memory {
  align-items: stretch;
  flex-direction: column;
}

.project-live-metric-card__content {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.project-live-metric-card__content span,
.project-network-io-card__summary span,
.project-overview-card-footer {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-live-metric-card__content strong,
.project-network-io-card__summary strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
}

.project-live-metric-card__content em,
.project-network-io-card__summary em {
  color: var(--td-text-color-placeholder);
  font-style: normal;
}

.project-runtime-target-card {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.project-runtime-target-card__header,
.project-runtime-target-card__identity {
  align-items: flex-start;
  display: flex;
}

.project-runtime-target-card__header {
  gap: var(--graft-density-gap-12);
}

.project-runtime-target-card__identity {
  flex: 1;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
  min-width: 0;
}

.project-runtime-target-card__icon {
  flex: none;
  height: 30px;
  width: 30px;
}

.project-runtime-target-card__content {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.project-runtime-target-card__identity strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.project-runtime-target-card__identity p,
.project-runtime-target-card__facts dt,
.project-runtime-target-card__endpoint > span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-2) 0 0;
}

.project-runtime-target-card__facts {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  margin: 0;
}

.project-runtime-target-card__facts div {
  align-items: baseline;
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: 72px minmax(0, 1fr);
  min-width: 0;
}

.project-runtime-target-card__facts dt,
.project-runtime-target-card__facts dd {
  margin: 0;
}

.project-runtime-target-card__facts dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  min-width: 0;
}

.project-runtime-target-card__endpoint {
  border-top: 1px solid var(--td-border-level-1-color);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  padding-top: var(--graft-density-gap-12);
}

.project-runtime-target-card__endpoint > span {
  margin: 0;
}

.project-runtime-target-card__endpoint strong {
  color: var(--td-text-color-primary);
  display: block;
  font: var(--td-font-body-medium);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-overview-card-footer {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
}

.project-network-io-card__summary {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-network-io-card__summary article {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-width: 0;
  padding: var(--graft-density-gap-16);
}

.project-live-metric--up.project-live-metric--pulse-a::after,
.project-live-metric--up.project-live-metric--pulse-b::after,
.project-live-metric--down.project-live-metric--pulse-a::after,
.project-live-metric--down.project-live-metric--pulse-b::after {
  animation: project-live-metric-pulse 1.2s ease-out;
  content: '';
  inset: 0;
  pointer-events: none;
  position: absolute;
}

.project-live-metric--up::after {
  background: linear-gradient(90deg, color-mix(in srgb, var(--td-warning-color-5) 14%, transparent), transparent 70%);
}

.project-live-metric--down::after {
  background: linear-gradient(90deg, color-mix(in srgb, var(--td-success-color-5) 14%, transparent), transparent 70%);
}

@keyframes project-live-metric-pulse {
  0% {
    opacity: 0.9;
  }

  100% {
    opacity: 0;
  }
}

.project-overview-activity-list {
  gap: var(--graft-density-gap-12);
}

.project-inline-head,
.project-inline-head__hint {
  margin: 0;
}

.project-file-groups,
.project-activity-card,
.project-activity-entries,
.project-lifecycle-file-list {
  flex-direction: column;
}

.project-file-groups,
.project-service-name,
.project-activity-card {
  gap: var(--graft-density-gap-12);
}

.project-lifecycle-file-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.project-lifecycle-layout,
.project-lifecycle-summary,
.project-lifecycle-command-section,
.project-lifecycle-config,
.project-lifecycle-config-group,
.project-lifecycle-command-card,
.project-lifecycle-command-preview,
.project-lifecycle-option-list,
.project-lifecycle-option__content,
.project-lifecycle-field {
  display: flex;
  flex-direction: column;
}

.project-lifecycle-layout,
.project-lifecycle-config,
.project-lifecycle-command-section {
  gap: var(--graft-density-gap-16);
}

.project-lifecycle-summary,
.project-lifecycle-config-group,
.project-lifecycle-command-card,
.project-lifecycle-command-preview,
.project-lifecycle-option-list,
.project-lifecycle-option__content,
.project-lifecycle-field {
  gap: var(--graft-density-gap-12);
}

.project-lifecycle-alert {
  margin-bottom: var(--graft-density-gap-16);
}

.project-lifecycle-file-list code {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-small);
  padding: var(--graft-density-gap-4) var(--graft-density-gap-8);
}

.project-lifecycle-command-grid,
.project-lifecycle-field-grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-lifecycle-command-card {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  min-width: 0;
  padding: var(--graft-density-gap-16);
}

.project-lifecycle-command-card__header,
.project-lifecycle-config-group__header,
.project-lifecycle-actions,
.project-lifecycle-option {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
}

.project-lifecycle-command-card__header,
.project-lifecycle-option {
  justify-content: space-between;
}

.project-lifecycle-command-card__title,
.project-lifecycle-config-group__header > div,
.project-lifecycle-option__content,
.project-lifecycle-field {
  min-width: 0;
}

.project-lifecycle-command-card__title strong,
.project-lifecycle-config-group__header strong,
.project-lifecycle-option__label,
.project-lifecycle-field__label {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.project-lifecycle-option__label,
.project-lifecycle-field__label--with-help {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-6);
}

.project-lifecycle-command-card__title p,
.project-lifecycle-config-group__header p,
.project-lifecycle-option__content p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.project-lifecycle-command-meta {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-lifecycle-command-toolbar,
.project-lifecycle-command-toolbar__toggle {
  align-items: center;
  display: flex;
}

.project-lifecycle-command-toolbar {
  justify-content: flex-end;
}

.project-lifecycle-command-toolbar__toggle {
  color: var(--td-text-color-secondary);
  gap: var(--graft-density-gap-8);
}

.project-lifecycle-field :deep(.t-input__wrap),
.project-lifecycle-field :deep(.t-input) {
  width: 100%;
}

.project-lifecycle-option {
  align-items: center;
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  min-height: 52px;
  padding: var(--graft-density-gap-12) var(--graft-density-gap-16);
}

.project-lifecycle-option__control {
  align-items: center;
  display: flex;
  flex: none;
}

.project-lifecycle-option__control :deep(.t-switch) {
  flex: none;
  width: auto;
}

.project-lifecycle-actions {
  justify-content: flex-end;
  margin-top: var(--graft-density-gap-20);
}

.project-service-name span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-service-mobile-card {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.project-service-mobile-card__header {
  align-items: flex-start;
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: auto minmax(0, 1fr);
}

.project-service-mobile-card__identity {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.project-service-mobile-card__identity strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.project-service-mobile-card__identity span,
.project-service-mobile-card__facts dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-service-mobile-card__identity span,
.project-service-mobile-card__ports dd {
  overflow-wrap: anywhere;
}

.project-service-mobile-card__actions {
  align-self: flex-end;
  width: auto;
}

.project-service-mobile-card__facts {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.project-service-mobile-card__facts div {
  min-width: 0;
}

.project-service-mobile-card__facts dt,
.project-service-mobile-card__facts dd {
  margin: 0;
}

.project-service-mobile-card__facts dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  margin-top: var(--graft-density-gap-4);
}

.project-service-mobile-card__ports {
  grid-column: 1 / -1;
}

.project-service-selection-toolbar {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
  width: 100%;
}

.project-service-selection-toolbar__summary {
  color: var(--td-text-color-primary);
  min-width: 0;
}

.project-activity-card__head {
  align-items: flex-start;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.project-activity-card__head p {
  color: var(--td-text-color-secondary);
  margin: var(--graft-density-gap-4) 0 0;
}

.project-code-panel {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.project-code-panel__meta {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.project-code-panel pre,
.project-activity-entries article {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  margin: 0;
  padding: var(--graft-density-gap-12);
}

.project-code-panel pre {
  max-height: 420px;
  overflow: auto;
  white-space: pre-wrap;
}

@media (width <= 960px) {
  .project-lifecycle-command-grid,
  .project-lifecycle-field-grid {
    grid-template-columns: 1fr;
  }
}

@media (width <= 720px) {
  .project-detail-actions--wide {
    display: none;
  }

  .project-detail-actions--compact {
    display: inline-flex;
  }

  .project-lifecycle-command-card__header,
  .project-lifecycle-option {
    align-items: flex-start;
    flex-direction: column;
  }

  .project-lifecycle-option__control {
    align-self: flex-end;
  }
}

@container (max-width: 640px) {
  .project-detail-actions--wide {
    display: none;
  }

  .project-detail-actions--compact {
    display: inline-flex;
  }
}

.project-diagnostics-list,
.project-activity-list,
.project-diff-list,
.project-configuration-warning-list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.project-configuration-alert,
.project-diff-panel {
  margin-bottom: var(--graft-density-gap-12);
}

.project-diff-panel,
.project-diff-meta {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
}

.project-diff-meta {
  color: var(--td-text-color-secondary);
}

.project-activity-grid {
  align-items: flex-start;
}

.project-activity-grid > section {
  flex: 1 1 0;
  min-width: 0;
}

.project-activity-entries {
  gap: var(--graft-density-gap-8);
}

.project-activity-entries article header {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
  margin-bottom: var(--graft-density-gap-6);
}

.project-activity-toolbar__since {
  width: 140px;
}

.project-activity-toolbar__tail {
  width: 96px;
}

.project-logs-toolbar {
  justify-content: flex-end;
  margin-bottom: var(--graft-density-gap-12);
}

.project-logs-toolbar__summary {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

@media (width <= 768px) {
  .project-overview-grid,
  .project-runtime-grid,
  .project-configuration-grid,
  .project-activity-grid {
    flex-direction: column;
  }

  .project-network-io-card__summary {
    grid-template-columns: 1fr;
  }

  .project-overview-hero__body,
  .project-activity-card__head {
    flex-direction: column;
  }

  .project-service-card__metrics {
    grid-template-columns: 1fr;
  }

  .project-runtime-target-card__facts div {
    grid-template-columns: 64px minmax(0, 1fr);
  }
}
</style>
