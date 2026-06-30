<template>
  <div class="project-detail-page" data-page-type="list-form-detail">
    <management-page-header
      :title="pageTitle"
      description-key="project.detail.description"
      :description="t('project.detail.description')"
      :source="{ labelKey: 'project.detail.eyebrow', fallback: t('project.detail.eyebrow') }"
    >
      <template #actions>
        <t-button theme="primary" :loading="detailLoading" @click="refreshDetail">
          <template #icon><refresh-icon /></template>
          {{ t('project.detail.refresh') }}
        </t-button>
      </template>
      <template #meta>
        <t-space break-line size="small">
          <t-tag :theme="refreshStatusTheme(detailRecord?.last_refresh_status)" variant="light-outline">
            {{ detailRecord ? refreshStatusLabel(detailRecord.last_refresh_status) : '-' }}
          </t-tag>
          <t-tag :theme="driftStatusTheme(detailRecord?.drift_status)" variant="light-outline">
            {{ detailRecord ? driftStatusLabel(detailRecord.drift_status) : '-' }}
          </t-tag>
          <t-tag theme="default" variant="light-outline">
            {{ detailRecord?.canonical_project_name || fallbackCanonicalName }}
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
        <section class="project-detail-summary">
          <t-card
            class="project-detail-summary-card"
            size="small"
            :bordered="false"
            :title="t('project.detail.overview.summaryTitle')"
          >
            <t-descriptions bordered size="small" :column="2">
              <t-descriptions-item :label="t('project.detail.overview.displayName')">
                {{ detailRecord.display_name }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.detail.overview.canonicalName')">
                <code>{{ detailRecord.canonical_project_name }}</code>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.detail.overview.workingDirectory')">
                <div class="project-detail-copy-row">
                  <code>{{ detailRecord.working_directory }}</code>
                  <t-button
                    size="small"
                    theme="default"
                    variant="text"
                    @click="copyPath(detailRecord.working_directory)"
                  >
                    {{ t('project.detail.actions.copyPath') }}
                  </t-button>
                </div>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.detail.overview.serviceCount')">
                {{ detailRecord.service_count }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.detail.overview.sourceKind')">
                {{ sourceKindLabel(detailRecord.source_kind) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.detail.overview.ownershipMode')">
                {{ ownershipModeLabel(detailRecord.ownership_mode) }}
              </t-descriptions-item>
            </t-descriptions>
          </t-card>

          <t-card
            class="project-detail-summary-card"
            size="small"
            :bordered="false"
            :title="t('project.detail.overview.lifecycleTitle')"
          >
            <div class="project-detail-action-bar">
              <t-button
                theme="primary"
                variant="outline"
                :loading="actionLoading === 'up'"
                @click="runLifecycleAction('up')"
              >
                {{ t('project.detail.actions.up') }}
              </t-button>
              <t-button
                theme="warning"
                variant="outline"
                :loading="actionLoading === 'down'"
                @click="runLifecycleAction('down')"
              >
                {{ t('project.detail.actions.down') }}
              </t-button>
              <t-button
                theme="warning"
                variant="outline"
                :loading="actionLoading === 'restart'"
                @click="runLifecycleAction('restart')"
              >
                {{ t('project.detail.actions.restart') }}
              </t-button>
              <t-button
                theme="default"
                variant="outline"
                :loading="actionLoading === 'unregister'"
                @click="runLifecycleAction('unregister')"
              >
                {{ t('project.detail.actions.unregister') }}
              </t-button>
            </div>
          </t-card>
        </section>

        <t-card class="project-detail-tabs-card" :bordered="true">
          <t-tabs v-model:value="activeTab" theme="card">
            <t-tab-panel value="overview" :label="t('project.detail.tabs.overview')">
              <section class="project-tab-section">
                <div class="project-overview-grid">
                  <t-card size="small" :title="t('project.detail.overview.snapshotTitle')">
                    <t-descriptions size="small" :column="1">
                      <t-descriptions-item :label="t('project.detail.overview.runtimeStatus')">
                        {{ runtimeStatusLabel(detailRecord.runtime_status) }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.overview.containerCounts')">
                        {{ detailRecord.container_counts.running }}/{{ detailRecord.container_counts.total }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.overview.lastRefreshStatus')">
                        {{ refreshStatusLabel(detailRecord.last_refresh_status) }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.overview.lastRefreshAt')">
                        {{ formatTime(detailRecord.last_refresh_at) }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.overview.lastRefreshHash')">
                        <code>{{ detailRecord.last_refresh_config_hash || '-' }}</code>
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.overview.lastObservedHash')">
                        <code>{{ detailRecord.last_observed_config_hash || '-' }}</code>
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.overview.driftStatus')">
                        {{ driftStatusLabel(detailRecord.drift_status) }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.overview.lastDriftCheckedAt')">
                        {{ formatTime(detailRecord.last_drift_checked_at) }}
                      </t-descriptions-item>
                    </t-descriptions>
                  </t-card>
                  <t-card size="small" :title="t('project.detail.overview.diagnosticsTitle')">
                    <div v-if="configurationMetadata?.diagnostics_summary?.length" class="project-diagnostics-list">
                      <t-alert
                        v-for="(item, index) in configurationMetadata.diagnostics_summary"
                        :key="`${index}-${item}`"
                        theme="warning"
                        :message="item"
                      />
                    </div>
                    <t-empty v-else :description="t('project.detail.overview.diagnosticsEmpty')" />
                  </t-card>
                </div>
              </section>
            </t-tab-panel>

            <t-tab-panel value="services" :label="t('project.detail.tabs.services')">
              <section class="project-tab-section">
                <management-table-card>
                  <template #head>
                    <div class="project-inline-head">
                      <p>{{ t('project.detail.services.summary', { count: serviceItems.length }) }}</p>
                    </div>
                  </template>
                  <div v-if="servicesLoading" class="project-inline-loading">
                    <t-loading :loading="true" />
                  </div>
                  <t-alert v-else-if="servicesError" theme="error" :message="servicesError" />
                  <t-empty
                    v-else-if="!serviceItems.length"
                    :title="t('project.detail.services.emptyTitle')"
                    :description="t('project.detail.services.emptyDescription')"
                  />
                  <div v-else class="project-services-list">
                    <t-card v-for="service in serviceItems" :key="service.service_name" size="small" bordered>
                      <div class="project-service-card">
                        <div class="project-service-card__head">
                          <div>
                            <strong>{{ service.service_name }}</strong>
                            <p>{{ service.image || '-' }}</p>
                          </div>
                          <t-space break-line size="small">
                            <t-tag theme="success" variant="light">
                              {{ t('project.detail.services.runningCount', { count: service.running_count }) }}
                            </t-tag>
                            <t-tag theme="warning" variant="light">
                              {{ t('project.detail.services.stoppedCount', { count: service.stopped_count }) }}
                            </t-tag>
                          </t-space>
                        </div>
                        <div class="project-service-card__body">
                          <p>
                            <strong>{{ t('project.detail.services.declaredPorts') }}:</strong>
                            {{ joinList(service.declared_ports) }}
                          </p>
                          <p>
                            <strong>{{ t('project.detail.services.declaredVolumes') }}:</strong>
                            {{ joinList(service.declared_volumes) }}
                          </p>
                          <p>
                            <strong>{{ t('project.detail.services.declaredNetworks') }}:</strong>
                            {{ joinList(service.declared_networks) }}
                          </p>
                        </div>
                        <div class="project-service-members">
                          <span>{{ t('project.detail.services.members') }}</span>
                          <div class="project-service-members__items">
                            <t-button
                              v-for="member in service.container_members"
                              :key="member.container_id"
                              theme="default"
                              variant="outline"
                              size="small"
                              @click="openContainerDetail(member)"
                            >
                              {{ member.container_name }} · {{ member.state }}
                            </t-button>
                          </div>
                        </div>
                      </div>
                    </t-card>
                  </div>
                </management-table-card>
              </section>
            </t-tab-panel>

            <t-tab-panel value="configuration" :label="t('project.detail.tabs.configuration')">
              <section class="project-tab-section">
                <div class="project-configuration-grid">
                  <t-card size="small" :title="t('project.detail.configuration.title')">
                    <t-descriptions size="small" :column="1">
                      <t-descriptions-item :label="t('project.detail.configuration.composeFiles')">
                        {{ configurationMetadata?.compose_files.length || 0 }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.configuration.envFiles')">
                        {{ configurationMetadata?.env_files.length || 0 }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.configuration.ownershipMode')">
                        {{ configurationMetadata ? ownershipModeLabel(configurationMetadata.ownership_mode) : '-' }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.configuration.driftStatus')">
                        {{ configurationMetadata ? driftStatusLabel(configurationMetadata.drift_status) : '-' }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('project.detail.configuration.refreshStatus')">
                        {{
                          configurationMetadata ? refreshStatusLabel(configurationMetadata.last_refresh_status) : '-'
                        }}
                      </t-descriptions-item>
                    </t-descriptions>
                    <div class="project-file-groups">
                      <section>
                        <strong>{{ t('project.detail.configuration.composeFiles') }}</strong>
                        <t-space direction="vertical" size="small">
                          <t-button
                            v-for="file in configurationMetadata?.compose_files || []"
                            :key="file.id"
                            theme="default"
                            variant="text"
                            size="small"
                            @click="selectConfigurationFile(file.id)"
                          >
                            {{ file.display_path }}
                          </t-button>
                        </t-space>
                      </section>
                      <section>
                        <strong>{{ t('project.detail.configuration.envFiles') }}</strong>
                        <t-space direction="vertical" size="small">
                          <t-button
                            v-for="file in configurationMetadata?.env_files || []"
                            :key="file.id"
                            theme="default"
                            variant="text"
                            size="small"
                            @click="selectConfigurationFile(file.id)"
                          >
                            {{ file.display_path }}
                          </t-button>
                        </t-space>
                      </section>
                    </div>
                  </t-card>

                  <t-card size="small" :title="t('project.detail.configuration.previewTitle')">
                    <div v-if="configurationPreview?.normalized_compose_yaml" class="project-code-panel">
                      <div class="project-code-panel__meta">
                        <t-tag theme="default" variant="light-outline">
                          {{ t('project.detail.configuration.previewHash') }}: {{ configurationPreview.config_hash }}
                        </t-tag>
                        <span
                          >{{ t('project.detail.configuration.previewUpdatedAt') }}:
                          {{ formatTime(configurationPreview.refreshed_at) }}</span
                        >
                      </div>
                      <pre>{{ configurationPreview.normalized_compose_yaml }}</pre>
                    </div>
                    <t-empty v-else :description="t('project.detail.configuration.previewEmpty')" />
                  </t-card>

                  <t-card size="small" :title="t('project.detail.configuration.editorTitle')">
                    <template #actions>
                      <t-space size="small" break-line>
                        <t-button
                          size="small"
                          theme="default"
                          variant="outline"
                          :loading="configurationLoading"
                          @click="resetDraftFromCurrent"
                        >
                          {{ t('project.detail.configuration.resetDraft') }}
                        </t-button>
                        <t-button
                          size="small"
                          theme="primary"
                          variant="outline"
                          :loading="configurationDiffLoading"
                          :disabled="!managedConfigurationEnabled"
                          @click="runConfigurationDiff"
                        >
                          {{ t('project.detail.configuration.runDiff') }}
                        </t-button>
                        <t-button
                          size="small"
                          theme="primary"
                          variant="outline"
                          :loading="configurationValidateLoading"
                          :disabled="!managedConfigurationEnabled"
                          @click="runConfigurationValidate"
                        >
                          {{ t('project.detail.configuration.runValidate') }}
                        </t-button>
                        <t-button
                          size="small"
                          theme="primary"
                          :loading="configurationDeployLoading"
                          :disabled="!managedConfigurationEnabled"
                          @click="runConfigurationDeploy"
                        >
                          {{ t('project.detail.configuration.deploy') }}
                        </t-button>
                      </t-space>
                    </template>
                    <t-alert
                      v-if="configurationAuthorityNotice"
                      :theme="managedConfigurationEnabled ? 'info' : 'warning'"
                      :message="configurationAuthorityNotice"
                      class="project-configuration-alert"
                    />
                    <t-tabs v-model:value="configurationEditorTab" theme="card">
                      <t-tab-panel value="compose" :label="t('project.detail.configuration.composeEditorTab')">
                        <project-file-editor
                          v-model="configurationDraft.compose_file_content"
                          v-model:mode="composeEditorMode"
                          :title="t('project.detail.configuration.composeEditorTitle')"
                          :description="t('project.detail.configuration.composeEditorDescription')"
                          :placeholder="t('project.detail.configuration.composeEditorPlaceholder')"
                          :empty-label="t('project.detail.configuration.composeEditorEmpty')"
                          :edit-label="t('project.detail.configuration.backToEditor')"
                          :preview-label="t('project.detail.configuration.previewDraft')"
                          :format-label="t('project.detail.configuration.formatDraft')"
                          :fullscreen-label="t('project.detail.configuration.fullscreen')"
                          :exit-fullscreen-label="t('project.detail.configuration.exitFullscreen')"
                          :resize-handle-label="t('project.detail.configuration.resizeEditor')"
                          storage-key="graft.project.detail.compose.editor.height"
                          @format="formatComposeDraft"
                        />
                      </t-tab-panel>
                      <t-tab-panel value="env" :label="t('project.detail.configuration.envEditorTab')">
                        <project-file-editor
                          v-model="envDraftContent"
                          v-model:mode="envEditorMode"
                          :title="t('project.detail.configuration.envEditorTitle')"
                          :description="t('project.detail.configuration.envEditorDescription')"
                          :placeholder="t('project.detail.configuration.envEditorPlaceholder')"
                          :empty-label="t('project.detail.configuration.envEditorEmpty')"
                          :edit-label="t('project.detail.configuration.backToEditor')"
                          :preview-label="t('project.detail.configuration.previewDraft')"
                          :format-label="t('project.detail.configuration.formatDraft')"
                          :fullscreen-label="t('project.detail.configuration.fullscreen')"
                          :exit-fullscreen-label="t('project.detail.configuration.exitFullscreen')"
                          :resize-handle-label="t('project.detail.configuration.resizeEditor')"
                          storage-key="graft.project.detail.env.editor.height"
                          @format="formatEnvDraft"
                        />
                      </t-tab-panel>
                    </t-tabs>
                  </t-card>

                  <t-card size="small" :title="t('project.detail.configuration.diffTitle')">
                    <t-loading :loading="configurationDiffLoading">
                      <div v-if="configurationDiffResult" class="project-diff-list">
                        <t-alert
                          :theme="configurationDiffResult.has_changes ? 'warning' : 'success'"
                          :message="
                            configurationDiffResult.has_changes
                              ? t('project.detail.configuration.diffHasChanges')
                              : t('project.detail.configuration.diffNoChanges')
                          "
                        />
                        <t-space break-line size="small">
                          <t-tag theme="default" variant="light-outline">
                            {{ t('project.detail.configuration.currentHash') }}:
                            {{ configurationDiffResult.current_config_hash }}
                          </t-tag>
                          <t-tag theme="primary" variant="light-outline">
                            {{ t('project.detail.configuration.proposedHash') }}:
                            {{ configurationDiffResult.proposed_config_hash }}
                          </t-tag>
                        </t-space>
                        <t-collapse :value="expandedDiffPanels" @change="handleDiffPanelChange">
                          <t-collapse-panel
                            v-for="file in configurationDiffResult.files"
                            :key="`${file.kind}-${file.path}`"
                            :value="file.path"
                            :header="file.path"
                          >
                            <template #headerRightContent>
                              <t-tag :theme="file.changed ? 'warning' : 'success'" variant="light-outline">
                                {{
                                  file.changed
                                    ? t('project.detail.configuration.diffFileChanged')
                                    : t('project.detail.configuration.diffFileUnchanged')
                                }}
                              </t-tag>
                            </template>
                            <div class="project-diff-panel">
                              <div class="project-diff-meta">
                                <span
                                  >{{ t('project.detail.configuration.currentHash') }}: {{ file.current_hash }}</span
                                >
                                <span
                                  >{{ t('project.detail.configuration.proposedHash') }}: {{ file.proposed_hash }}</span
                                >
                              </div>
                              <pre>{{ file.proposed_content }}</pre>
                            </div>
                          </t-collapse-panel>
                        </t-collapse>
                        <div v-if="configurationDiffResult.warnings?.length" class="project-configuration-warning-list">
                          <t-alert
                            v-for="warning in configurationDiffResult.warnings"
                            :key="warning"
                            theme="warning"
                            :message="warning"
                          />
                        </div>
                      </div>
                      <t-empty v-else :description="t('project.detail.configuration.diffEmpty')" />
                    </t-loading>
                  </t-card>

                  <t-card size="small" :title="t('project.detail.configuration.validationTitle')">
                    <t-loading :loading="configurationValidateLoading">
                      <div v-if="configurationValidateResult" class="project-code-panel">
                        <div class="project-code-panel__meta">
                          <t-tag theme="primary" variant="light-outline">
                            {{ t('project.detail.configuration.proposedHash') }}:
                            {{ configurationValidateResult.proposed_config_hash }}
                          </t-tag>
                          <span>
                            {{ t('project.detail.configuration.declaredServices') }}:
                            {{ configurationValidateResult.declared_service_names.join(', ') || '-' }}
                          </span>
                        </div>
                        <pre>{{ configurationValidateResult.normalized_compose_yaml }}</pre>
                        <div
                          v-if="configurationValidateResult.warnings?.length"
                          class="project-configuration-warning-list"
                        >
                          <t-alert
                            v-for="warning in configurationValidateResult.warnings"
                            :key="warning"
                            theme="warning"
                            :message="warning"
                          />
                        </div>
                      </div>
                      <t-empty v-else :description="t('project.detail.configuration.validationEmpty')" />
                    </t-loading>
                  </t-card>

                  <t-card size="small" :title="t('project.detail.configuration.fileContentTitle')">
                    <div v-if="selectedConfigurationFile?.content" class="project-code-panel">
                      <div class="project-code-panel__meta">
                        <t-tag theme="default" variant="light-outline">
                          {{ selectedConfigurationFile.download_name }}
                        </t-tag>
                        <t-button size="small" theme="default" variant="text" @click="copyConfigurationContent">
                          {{ t('project.detail.configuration.copyContent') }}
                        </t-button>
                      </div>
                      <pre>{{ selectedConfigurationFile.content }}</pre>
                    </div>
                    <t-empty v-else :description="t('project.detail.configuration.fileEmpty')" />
                  </t-card>
                </div>
              </section>
            </t-tab-panel>

            <t-tab-panel value="activity" :label="t('project.detail.tabs.activity')">
              <section class="project-tab-section">
                <t-card size="small" :title="t('project.detail.activity.title')">
                  <template #actions>
                    <t-space size="small" align="center">
                      <t-input
                        v-model="activitySince"
                        class="project-activity-toolbar__since"
                        :placeholder="t('project.detail.activity.sinceLabel')"
                      />
                      <t-input
                        v-model="activityTail"
                        class="project-activity-toolbar__tail"
                        :placeholder="t('project.detail.activity.tailLabel')"
                      />
                      <t-button theme="primary" variant="outline" :loading="activityLoading" @click="loadActivity">
                        {{ t('project.list.refresh') }}
                      </t-button>
                    </t-space>
                  </template>
                  <p class="project-inline-head__hint">{{ t('project.detail.activity.summary') }}</p>
                  <t-alert v-if="activityError" theme="error" :message="activityError" class="project-activity-alert" />
                  <t-empty
                    v-else-if="!activityMembers.length"
                    :title="t('project.detail.activity.emptyTitle')"
                    :description="t('project.detail.activity.emptyDescription')"
                  />
                  <div v-else class="project-activity-list">
                    <t-card v-for="member in activityMembers" :key="member.container_id" size="small" bordered>
                      <div class="project-activity-card">
                        <div class="project-activity-card__head">
                          <div>
                            <strong>{{ member.container_name }}</strong>
                            <p>{{ member.container_id }}</p>
                          </div>
                          <t-button size="small" theme="default" variant="outline" @click="openContainerDetail(member)">
                            {{ t('project.detail.services.openContainer') }}
                          </t-button>
                        </div>
                        <div class="project-activity-grid">
                          <section>
                            <div class="project-activity-grid__title">
                              <span>{{ t('project.detail.activity.eventSection') }}</span>
                              <t-tag theme="default" variant="light-outline">
                                {{ t('project.detail.activity.eventCount', { count: member.events.length }) }}
                              </t-tag>
                            </div>
                            <div v-if="member.events.length" class="project-activity-entries">
                              <article
                                v-for="eventItem in member.events"
                                :key="`event-${member.container_id}-${eventItem.seq}`"
                              >
                                <header>
                                  <t-tag :theme="eventSeverityTheme(eventItem.event.severity)" variant="light-outline">
                                    {{ eventItem.event.severity }}
                                  </t-tag>
                                  <span>{{ formatTime(eventItem.event.occurred_at) }}</span>
                                </header>
                                <strong>{{ eventItem.event.event_type }}</strong>
                                <p>{{ summarizeEvent(eventItem) }}</p>
                              </article>
                            </div>
                            <t-empty v-else size="small" />
                          </section>
                          <section>
                            <div class="project-activity-grid__title">
                              <span>{{ t('project.detail.activity.logSection') }}</span>
                              <t-tag theme="default" variant="light-outline">
                                {{ t('project.detail.activity.logCount', { count: member.logs.length }) }}
                              </t-tag>
                            </div>
                            <div v-if="member.logs.length" class="project-activity-entries">
                              <article
                                v-for="(logItem, index) in member.logs"
                                :key="`log-${member.container_id}-${index}`"
                              >
                                <header>
                                  <t-tag
                                    :theme="logItem.stream === 'stderr' ? 'danger' : 'default'"
                                    variant="light-outline"
                                  >
                                    {{ logItem.stream }}
                                  </t-tag>
                                  <span>{{ formatTime(logItem.occurred_at) }}</span>
                                </header>
                                <p>{{ logItem.line }}</p>
                              </article>
                            </div>
                            <t-empty v-else size="small" />
                          </section>
                        </div>
                      </div>
                    </t-card>
                  </div>
                </t-card>
              </section>
            </t-tab-panel>
          </t-tabs>
        </t-card>
      </template>
    </section>
  </div>
</template>
<script setup lang="ts">
import { RefreshIcon } from 'tdesign-icons-vue-next';
import { DialogPlugin } from 'tdesign-vue-next/es/dialog';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { LOCALE, type LocalizedTitle } from '@/contracts/i18n/locales';
import { getContainerEvents, getContainerLogs } from '@/modules/container/api/container';
import { CONTAINER_BOOTSTRAP_ROUTE } from '@/modules/container/contract/bootstrap';
import type { ContainerLogEntry, ContainerRuntimeEventRecord } from '@/modules/container/types/container';
import { ManagementPageHeader, ManagementTableCard } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { copyText } from '@/shared/observability';
import { useTabsRouterStore } from '@/store/modules/tabs-router';
import { createLogger } from '@/utils/logger';

import {
  getProject,
  getProjectConfiguration,
  getProjectConfigurationFile,
  getProjectConfigurationPreview,
  getProjectServices,
  postProjectConfigurationDiff,
  postProjectConfigurationValidate,
  postProjectDeploy,
  postProjectDown,
  postProjectRestart,
  postProjectUnregister,
  postProjectUp,
} from '../../api/project';
import ProjectFileEditor from '../../components/ProjectFileEditor.vue';
import {
  formatProjectTime,
  projectDriftStatusLabel,
  projectDriftStatusTheme,
  projectOwnershipModeLabel,
  projectRefreshStatusLabel,
  projectRefreshStatusTheme,
  projectRuntimeStatusLabel,
  projectSourceKindLabel,
} from '../../shared/display';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../../shared/navigation';
import type {
  ProjectActionResponse,
  ProjectConfigurationDiffRequest,
  ProjectConfigurationDiffResponse,
  ProjectConfigurationFileResponse,
  ProjectConfigurationMetadataResponse,
  ProjectConfigurationPreviewResponse,
  ProjectConfigurationValidateRequest,
  ProjectConfigurationValidateResponse,
  ProjectDeployRequest,
  ProjectDetailResponse,
  ProjectServiceContainerMember,
  ProjectServiceItem,
} from '../../types/project';

defineOptions({
  name: 'ProjectDetailIndex',
});

type ActivityMember = ProjectServiceContainerMember & {
  events: ContainerRuntimeEventRecord[];
  logs: ContainerLogEntry[];
};
type EditorMode = 'edit' | 'preview';
type ConfigurationEditorTab = 'compose' | 'env';

const { locale, t } = useI18n();
const route = useRoute();
const router = useRouter();
const tabsRouterStore = useTabsRouterStore();
const logger = createLogger('project.detail');

const detailRecord = ref<ProjectDetailResponse | null>(null);
const detailLoading = ref(false);
const detailError = ref('');
const servicesLoading = ref(false);
const servicesError = ref('');
const serviceItems = ref<ProjectServiceItem[]>([]);
const configurationMetadata = ref<ProjectConfigurationMetadataResponse | null>(null);
const configurationPreview = ref<ProjectConfigurationPreviewResponse | null>(null);
const selectedConfigurationFile = ref<ProjectConfigurationFileResponse | null>(null);
const configurationDiffResult = ref<ProjectConfigurationDiffResponse | null>(null);
const configurationValidateResult = ref<ProjectConfigurationValidateResponse | null>(null);
const configurationEditorTab = ref<ConfigurationEditorTab>('compose');
const composeEditorMode = ref<EditorMode>('edit');
const envEditorMode = ref<EditorMode>('edit');
const configurationLoading = ref(false);
const configurationDiffLoading = ref(false);
const configurationValidateLoading = ref(false);
const configurationDeployLoading = ref(false);
const expandedDiffPanels = ref<Array<string | number>>([]);
const activityMembers = ref<ActivityMember[]>([]);
const activityLoading = ref(false);
const activityError = ref('');
const activeTab = ref(String(route.query.tab || 'overview'));
const actionLoading = ref<ProjectActionResponse['action'] | ''>('');
const activitySince = ref('1h');
const activityTail = ref('40');
const configurationDraft = reactive<ProjectDeployRequest>({
  compose_file_content: '',
  env_file_content: '',
});

const projectId = computed(() => Number(route.params.id));
const activeTabRoute = computed(() =>
  tabsRouterStore.tabRouterList.find(
    (tab) => tab.tabKey === route.path || tab.path === route.path || tab.fullPath === route.fullPath,
  ),
);
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
const managedConfigurationEnabled = computed(() => detailRecord.value?.ownership_mode === 'managed-root-dedicated');
const configurationAuthorityNotice = computed(() => {
  if (!detailRecord.value) {
    return '';
  }
  if (managedConfigurationEnabled.value) {
    return t('project.detail.configuration.managedAuthorityHint');
  }
  return t('project.detail.configuration.externalAuthorityHint');
});
const envDraftContent = computed({
  get: () => configurationDraft.env_file_content || '',
  set: (value: string) => {
    configurationDraft.env_file_content = value;
  },
});

watch(
  () => route.query.tab,
  (value) => {
    activeTab.value = typeof value === 'string' ? value : 'overview';
  },
);

watch(activeTab, async (value) => {
  if (value === 'services') {
    await loadServices();
  }
  if (value === 'configuration') {
    await loadConfiguration();
  }
  if (value === 'activity') {
    await loadActivity();
  }
});

onMounted(async () => {
  await refreshDetail();
});

function formatTime(value?: string | null) {
  return formatProjectTime(locale.value, value);
}

function sourceKindLabel(value: ProjectDetailResponse['source_kind']) {
  return projectSourceKindLabel(t, value);
}

function ownershipModeLabel(value: ProjectDetailResponse['ownership_mode']) {
  return projectOwnershipModeLabel(t, value);
}

function driftStatusLabel(value: ProjectDetailResponse['drift_status']) {
  return projectDriftStatusLabel(t, value);
}

function driftStatusTheme(value?: ProjectDetailResponse['drift_status']) {
  return projectDriftStatusTheme(value);
}

function refreshStatusLabel(value: ProjectDetailResponse['last_refresh_status']) {
  return projectRefreshStatusLabel(t, value);
}

function refreshStatusTheme(value?: ProjectDetailResponse['last_refresh_status']) {
  return projectRefreshStatusTheme(value);
}

function runtimeStatusLabel(value?: ProjectDetailResponse['runtime_status'] | null) {
  return projectRuntimeStatusLabel(t, value);
}

function joinList(items?: string[] | null) {
  return items && items.length > 0 ? items.join(', ') : '-';
}

async function refreshDetail() {
  if (!Number.isFinite(projectId.value)) {
    detailError.value = t('project.list.retry');
    return;
  }
  detailLoading.value = true;
  detailError.value = '';
  try {
    detailRecord.value = await getProject(projectId.value);
    updateCurrentTabTitle(buildDetailTitle(detailRecord.value.display_name));
    if (activeTab.value === 'services') {
      await loadServices();
    } else if (activeTab.value === 'configuration') {
      await loadConfiguration();
    } else if (activeTab.value === 'activity') {
      await loadActivity();
    }
  } catch (error) {
    logger.error('failed to load project detail', error);
    detailRecord.value = null;
    detailError.value = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
  } finally {
    detailLoading.value = false;
  }
}

async function loadServices() {
  if (!Number.isFinite(projectId.value)) return;
  servicesLoading.value = true;
  servicesError.value = '';
  try {
    const response = await getProjectServices(projectId.value);
    serviceItems.value = response.items;
  } catch (error) {
    logger.error('failed to load project services', error);
    serviceItems.value = [];
    servicesError.value = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
  } finally {
    servicesLoading.value = false;
  }
}

async function loadConfiguration() {
  if (!Number.isFinite(projectId.value)) return;
  configurationLoading.value = true;
  try {
    const [metadata, preview] = await Promise.all([
      getProjectConfiguration(projectId.value),
      getProjectConfigurationPreview(projectId.value),
    ]);
    configurationMetadata.value = metadata;
    configurationPreview.value = preview;
    const firstFile = metadata.compose_files[0]?.id ?? metadata.env_files[0]?.id;
    if (typeof firstFile === 'number') {
      await selectConfigurationFile(firstFile);
    }
    hydrateDraftFromCurrent(metadata);
  } catch (error) {
    logger.error('failed to load project configuration', error);
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.retry')));
  } finally {
    configurationLoading.value = false;
  }
}

async function selectConfigurationFile(fileId: number) {
  if (!Number.isFinite(projectId.value)) return;
  try {
    selectedConfigurationFile.value = await getProjectConfigurationFile(projectId.value, fileId);
  } catch (error) {
    logger.error('failed to load project configuration file', error);
    selectedConfigurationFile.value = null;
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.retry')));
  }
}

async function copyConfigurationContent() {
  if (!selectedConfigurationFile.value?.content) return;
  try {
    await copyText(selectedConfigurationFile.value.content);
    MessagePlugin.success(t('project.detail.configuration.copySuccess'));
  } catch {
    MessagePlugin.error(t('project.detail.configuration.copyError'));
  }
}

function hydrateDraftFromCurrent(metadata: ProjectConfigurationMetadataResponse) {
  const composeFileId = metadata.compose_files[0]?.id;
  const envFileId = metadata.env_files[0]?.id;
  const tasks: Promise<unknown>[] = [];
  if (typeof composeFileId === 'number') {
    tasks.push(
      getProjectConfigurationFile(projectId.value, composeFileId).then((response) => {
        configurationDraft.compose_file_content = response.content;
      }),
    );
  }
  if (typeof envFileId === 'number') {
    tasks.push(
      getProjectConfigurationFile(projectId.value, envFileId).then((response) => {
        configurationDraft.env_file_content = response.content;
      }),
    );
  } else {
    configurationDraft.env_file_content = '';
  }
  void Promise.all(tasks).catch((error) => {
    logger.error('failed to hydrate project draft', error);
  });
}

function resetDraftFromCurrent() {
  if (configurationMetadata.value) {
    hydrateDraftFromCurrent(configurationMetadata.value);
  }
  configurationDiffResult.value = null;
  configurationValidateResult.value = null;
  expandedDiffPanels.value = [];
  configurationEditorTab.value = 'compose';
  composeEditorMode.value = 'edit';
  envEditorMode.value = 'edit';
}

function buildConfigurationDraftRequest(): ProjectConfigurationDiffRequest &
  ProjectConfigurationValidateRequest &
  ProjectDeployRequest {
  return {
    compose_file_content: normalizeTextBlock(configurationDraft.compose_file_content || ''),
    env_file_content: normalizeTextBlock(configurationDraft.env_file_content || ''),
  };
}

function formatComposeDraft() {
  configurationDraft.compose_file_content = normalizeTextBlock(configurationDraft.compose_file_content || '');
}

function formatEnvDraft() {
  configurationDraft.env_file_content = normalizeTextBlock(configurationDraft.env_file_content || '');
}

async function runConfigurationDiff() {
  if (!Number.isFinite(projectId.value) || !managedConfigurationEnabled.value) {
    MessagePlugin.warning(configurationAuthorityNotice.value);
    return;
  }
  configurationDiffLoading.value = true;
  try {
    configurationDiffResult.value = await postProjectConfigurationDiff(
      projectId.value,
      buildConfigurationDraftRequest(),
    );
    expandedDiffPanels.value = configurationDiffResult.value.files
      .filter((item) => item.changed)
      .map((item) => item.path);
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.configuration.diffFailed')));
  } finally {
    configurationDiffLoading.value = false;
  }
}

async function runConfigurationValidate() {
  if (!Number.isFinite(projectId.value) || !managedConfigurationEnabled.value) {
    MessagePlugin.warning(configurationAuthorityNotice.value);
    return;
  }
  configurationValidateLoading.value = true;
  try {
    configurationValidateResult.value = await postProjectConfigurationValidate(
      projectId.value,
      buildConfigurationDraftRequest(),
    );
    MessagePlugin.success(t('project.detail.configuration.validateSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.configuration.validateFailed')));
  } finally {
    configurationValidateLoading.value = false;
  }
}

async function runConfigurationDeploy() {
  if (!Number.isFinite(projectId.value) || !managedConfigurationEnabled.value) {
    MessagePlugin.warning(configurationAuthorityNotice.value);
    return;
  }
  const dialog = DialogPlugin.confirm({
    header: t('project.detail.configuration.deployConfirmTitle'),
    body: t('project.detail.configuration.deployConfirmDescription'),
    confirmBtn: {
      content: t('project.detail.configuration.deploy'),
      theme: 'primary',
    },
    cancelBtn: t('project.list.actions.cancel'),
    onConfirm: async () => {
      configurationDeployLoading.value = true;
      try {
        const response = await postProjectDeploy(projectId.value, buildConfigurationDraftRequest());
        MessagePlugin.success(response.message || t('project.detail.configuration.deploySuccess'));
        configurationDiffResult.value = null;
        configurationValidateResult.value = null;
        await refreshDetail();
        await loadConfiguration();
      } catch (error) {
        MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.configuration.deployFailed')));
      } finally {
        configurationDeployLoading.value = false;
        dialog.destroy();
      }
    },
  });
}

function handleDiffPanelChange(value: Array<string | number>) {
  expandedDiffPanels.value = value;
}

async function loadActivity() {
  if (!Number.isFinite(projectId.value)) return;
  activityLoading.value = true;
  activityError.value = '';
  try {
    if (!serviceItems.value.length) {
      await loadServices();
    }
    const members = serviceItems.value.flatMap((item) => item.container_members);
    const tail = Number(activityTail.value) || 40;
    const since = activitySince.value.trim() || '1h';
    const fanout = await Promise.all(
      members.map(async (member) => {
        const [events, logs] = await Promise.all([
          getContainerEvents(member.container_id),
          getContainerLogs(member.container_id, {
            tail,
            since,
            timestamps: true,
            stdout: true,
            stderr: true,
          }),
        ]);
        return {
          ...member,
          events: events.items.slice(0, 8),
          logs: logs.entries.slice(-12),
        } satisfies ActivityMember;
      }),
    );
    activityMembers.value = fanout;
  } catch (error) {
    logger.error('failed to fan out project activity', error);
    activityMembers.value = [];
    activityError.value = resolveLocalizedErrorMessage(t, error, t('project.detail.activity.loadFailed'));
  } finally {
    activityLoading.value = false;
  }
}

function eventSeverityTheme(value: ContainerRuntimeEventRecord['event']['severity']) {
  if (value === 'error') return 'danger';
  if (value === 'warning') return 'warning';
  return 'default';
}

function summarizeEvent(record: ContainerRuntimeEventRecord) {
  const attributes = record.event.attributes || {};
  const joined = Object.entries(attributes)
    .slice(0, 3)
    .map(([key, value]) => `${key}=${value}`)
    .join(', ');
  return joined || record.event.event_type;
}

async function runLifecycleAction(action: 'up' | 'down' | 'restart' | 'unregister') {
  if (!Number.isFinite(projectId.value)) return;
  actionLoading.value = action;
  try {
    if (action === 'up') {
      await postProjectUp(projectId.value);
    } else if (action === 'down') {
      await postProjectDown(projectId.value);
    } else if (action === 'restart') {
      await postProjectRestart(projectId.value);
    } else {
      await postProjectUnregister(projectId.value);
    }
    MessagePlugin.success(t('project.list.actions.actionSuccess'));
    await refreshDetail();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.actions.actionFailed')));
  } finally {
    actionLoading.value = '';
  }
}

async function copyPath(path: string) {
  try {
    await copyText(path);
    MessagePlugin.success(t('project.detail.actions.copyPathSuccess'));
  } catch {
    MessagePlugin.error(t('project.detail.actions.copyPathError'));
  }
}

function normalizeTextBlock(value: string) {
  const normalized = String(value ?? '')
    .replace(/\r\n/g, '\n')
    .split('\n')
    .map((line) => line.replace(/\s+$/g, ''))
    .join('\n')
    .trim();
  return normalized ? `${normalized}\n` : '';
}

function buildDetailTitle(name: string): LocalizedTitle {
  return buildDetailTitleWithFallback('project.route.detail.title', name);
}

function readNameFromTabTitle(title?: LocalizedTitle) {
  if (!title) return '';
  const current = title[locale.value as keyof LocalizedTitle] || title[LOCALE.ZH_CN] || title[LOCALE.EN_US] || '';
  const parts = current.split(' - ');
  return parts.length > 1 ? parts.slice(1).join(' - ').trim() : '';
}

function updateCurrentTabTitle(title: LocalizedTitle) {
  const routePath = route.path;
  const routeFullPath = route.fullPath;
  tabsRouterStore.tabRouterList = tabsRouterStore.tabRouterList.map((tab) =>
    tab.tabKey === routePath || tab.path === routePath || tab.fullPath === routeFullPath ? { ...tab, title } : tab,
  );
}

function openContainerDetail(member: ProjectServiceContainerMember) {
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
.project-detail-summary,
.project-overview-grid,
.project-configuration-grid,
.project-service-card,
.project-service-card__head,
.project-service-members,
.project-service-members__items,
.project-file-groups,
.project-detail-copy-row,
.project-activity-card,
.project-activity-card__head,
.project-activity-grid,
.project-activity-grid__title,
.project-activity-entries {
  display: flex;
}

.project-detail-page,
.project-detail-body {
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.project-detail-summary,
.project-overview-grid,
.project-configuration-grid,
.project-activity-grid {
  gap: var(--graft-density-gap-16);
}

.project-detail-summary,
.project-overview-grid,
.project-configuration-grid {
  align-items: stretch;
}

.project-detail-summary-card,
.project-overview-grid > .t-card,
.project-configuration-grid > .t-card {
  flex: 1 1 0;
  min-width: 0;
}

.project-detail-action-bar,
.project-detail-copy-row,
.project-service-members__items,
.project-activity-grid__title {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.project-inline-head,
.project-inline-head__hint {
  margin: 0;
}

.project-file-groups,
.project-service-card,
.project-service-members,
.project-activity-card,
.project-activity-entries {
  flex-direction: column;
}

.project-file-groups,
.project-service-card,
.project-activity-card {
  gap: var(--graft-density-gap-12);
}

.project-service-card__head,
.project-activity-card__head {
  align-items: flex-start;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.project-service-card__head p,
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

.project-diagnostics-list,
.project-services-list,
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

@media (width <= 768px) {
  .project-detail-summary,
  .project-overview-grid,
  .project-configuration-grid,
  .project-activity-grid {
    flex-direction: column;
  }

  .project-service-card__head,
  .project-activity-card__head {
    flex-direction: column;
  }
}
</style>
