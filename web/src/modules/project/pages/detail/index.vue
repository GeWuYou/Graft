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
          <t-tag :theme="runtimeStatusTheme(detailRecord?.runtime_status)" variant="light-outline">
            {{ detailRecord ? runtimeStatusLabel(detailRecord.runtime_status) : '-' }}
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
                <t-card size="small" :title="t('project.detail.summary.configurationTitle')">
                  <t-descriptions size="small" :column="1">
                    <t-descriptions-item :label="t('project.detail.summary.canonicalName')">
                      <code>{{ detailRecord.canonical_project_name }}</code>
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.summary.composeFiles')">
                      {{ detailRecord.compose_files.length }}
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.summary.workingDirectory')">
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
                    <t-descriptions-item :label="t('project.detail.summary.envFiles')">
                      {{ detailRecord.env_files.length }}
                    </t-descriptions-item>
                  </t-descriptions>
                </t-card>

                <t-card size="small" :title="t('project.detail.summary.summaryTitle')">
                  <t-descriptions size="small" :column="1">
                    <t-descriptions-item :label="t('project.detail.summary.status')">
                      {{ runtimeStatusLabel(detailRecord.runtime_status) }}
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.summary.services')">
                      {{ detailRecord.service_count }}
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.summary.configHash')">
                      <code>{{
                        detailRecord.last_refresh_config_hash || detailRecord.last_observed_config_hash || '-'
                      }}</code>
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.summary.nameSource')">
                      {{ projectCanonicalNameSourceLabel(t, detailRecord.canonical_project_name_source) }}
                    </t-descriptions-item>
                  </t-descriptions>
                </t-card>

                <t-card size="small" :title="t('project.detail.summary.discoveryTitle')">
                  <t-descriptions size="small" :column="1">
                    <t-descriptions-item :label="t('project.detail.summary.runtimeMembers')">
                      {{ detailRecord.container_counts.total }}
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.summary.runningMembers')">
                      {{ detailRecord.container_counts.running }}
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.summary.hostScope')">
                      {{ projectHostScopeLabel(t, detailRecord.host_scope) }}
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.summary.lastRefreshAt')">
                      {{ formatTime(detailRecord.last_refresh_at) }}
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

          <t-tab-panel value="services" :destroy-on-hide="false" :label="t('project.detail.tabs.services')">
            <section class="project-section project-tab-panel">
              <div class="project-section-heading">
                <div>
                  <h2>{{ t('project.detail.services.title') }}</h2>
                  <p>{{ t('project.detail.services.description') }}</p>
                </div>
              </div>

              <t-card size="small">
                <template #actions>
                  <t-space size="small">
                    <t-tag theme="default" variant="light-outline">
                      {{ t('project.detail.services.summary', { count: serviceRows.length }) }}
                    </t-tag>
                    <t-button
                      theme="primary"
                      variant="outline"
                      :loading="serviceLoading"
                      @click="refreshProjectServices"
                    >
                      {{ t('project.detail.services.refresh') }}
                    </t-button>
                  </t-space>
                </template>

                <t-table
                  row-key="service_name"
                  :columns="serviceColumns"
                  :data="serviceRows"
                  :loading="serviceLoading"
                  cell-empty-content="-"
                  hover
                >
                  <template #service_name="{ row }">
                    <div class="project-service-name">
                      <strong>{{ row.service_name }}</strong>
                      <span>{{ row.image || '-' }}</span>
                    </div>
                  </template>

                  <template #build_context="{ row }">
                    <code>{{ row.build_context || '-' }}</code>
                  </template>

                  <template #declared_networks="{ row }">
                    <span>{{ joinList(row.declared_networks || []) }}</span>
                  </template>

                  <template #declared_ports="{ row }">
                    <span>{{ joinList(row.declared_ports || []) }}</span>
                  </template>

                  <template #declared_volumes="{ row }">
                    <span>{{ joinList(row.declared_volumes || []) }}</span>
                  </template>

                  <template #runtime="{ row }">
                    <div class="project-detail-action-bar">
                      <t-tag theme="success" variant="light-outline">{{ row.running_count }}</t-tag>
                      <t-tag theme="warning" variant="light-outline">{{ row.stopped_count }}</t-tag>
                    </div>
                  </template>

                  <template #operation="{ row }">
                    <t-button
                      size="small"
                      theme="default"
                      variant="outline"
                      :disabled="!row.container_members.length"
                      @click="openFirstServiceContainer(row)"
                    >
                      {{ t('project.detail.services.openContainer') }}
                    </t-button>
                  </template>

                  <template #empty>
                    <t-empty
                      :title="t('project.detail.services.emptyTitle')"
                      :description="t('project.detail.services.emptyDescription')"
                    />
                  </template>
                </t-table>
              </t-card>
            </section>
          </t-tab-panel>

          <t-tab-panel value="containers" :destroy-on-hide="false" :label="t('project.detail.tabs.containers')">
            <project-resources-section
              :active-resource="'containers'"
              :canonical-project-name="detailRecord.canonical_project_name"
              :project-id="detailRecord.id"
              :show-resource-switcher="false"
              @open-container-detail="openContainerDetail"
            />
          </t-tab-panel>

          <t-tab-panel value="networks" :destroy-on-hide="false" :label="t('project.detail.tabs.networks')">
            <project-resources-section
              :active-resource="'networks'"
              :canonical-project-name="detailRecord.canonical_project_name"
              :project-id="detailRecord.id"
              :show-resource-switcher="false"
              @open-container-detail="openContainerDetail"
            />
          </t-tab-panel>

          <t-tab-panel value="volumes" :destroy-on-hide="false" :label="t('project.detail.tabs.volumes')">
            <project-resources-section
              :active-resource="'volumes'"
              :canonical-project-name="detailRecord.canonical_project_name"
              :project-id="detailRecord.id"
              :show-resource-switcher="false"
              @open-container-detail="openContainerDetail"
            />
          </t-tab-panel>

          <t-tab-panel value="configuration" :destroy-on-hide="false" :label="t('project.detail.tabs.configuration')">
            <section class="project-section project-tab-panel">
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
                      {{ configurationMetadata ? refreshStatusLabel(configurationMetadata.last_refresh_status) : '-' }}
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
                      <span>
                        {{ t('project.detail.configuration.previewUpdatedAt') }}:
                        {{ formatTime(configurationPreview.refreshed_at) }}
                      </span>
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
                              <span>{{ t('project.detail.configuration.currentHash') }}: {{ file.current_hash }}</span>
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

          <t-tab-panel value="activity" :destroy-on-hide="false" :label="t('project.detail.tabs.activity')">
            <section class="project-section project-tab-panel">
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
                <t-alert
                  theme="info"
                  :message="activityAuthorityNotice(detailRecord.activity_authority)"
                  class="project-activity-alert"
                />
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

          <t-tab-panel value="runtime" :destroy-on-hide="false" :label="t('project.detail.tabs.runtime')">
            <section class="project-section project-tab-panel">
              <div class="project-section-heading">
                <div>
                  <h2>{{ t('project.detail.runtime.title') }}</h2>
                  <p>{{ t('project.detail.runtime.description') }}</p>
                </div>
              </div>

              <section class="project-lifecycle-bar">
                <div class="project-detail-action-bar">
                  <t-button
                    v-if="lifecycleActionVisibility.up"
                    data-testid="project-detail-action-up"
                    theme="primary"
                    variant="outline"
                    :loading="actionLoading === 'up'"
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
                    @click="runLifecycleAction('restart')"
                  >
                    {{ t('project.detail.actions.restart') }}
                  </t-button>
                  <t-button
                    v-if="lifecycleActionVisibility.unregister"
                    data-testid="project-detail-action-unregister"
                    theme="default"
                    variant="outline"
                    :loading="actionLoading === 'unregister'"
                    @click="runLifecycleAction('unregister')"
                  >
                    {{ t('project.detail.actions.unregister') }}
                  </t-button>
                </div>
              </section>

              <div class="project-runtime-grid">
                <t-card size="small" :title="t('project.detail.runtime.statusTitle')">
                  <t-descriptions size="small" :column="1">
                    <t-descriptions-item :label="t('project.detail.runtime.runtimeStatus')">
                      {{ runtimeStatusLabel(detailRecord.runtime_status) }}
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.runtime.refreshStatus')">
                      {{ refreshStatusLabel(detailRecord.last_refresh_status) }}
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.runtime.driftStatus')">
                      {{ driftStatusLabel(detailRecord.drift_status) }}
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.runtime.lastRefreshAt')">
                      {{ formatTime(detailRecord.last_refresh_at) }}
                    </t-descriptions-item>
                  </t-descriptions>
                </t-card>

                <t-card size="small" :title="t('project.detail.runtime.membersTitle')">
                  <t-descriptions size="small" :column="1">
                    <t-descriptions-item :label="t('project.detail.runtime.runningMembers')">
                      {{ detailRecord.container_counts.running }}
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.runtime.totalMembers')">
                      {{ detailRecord.container_counts.total }}
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.runtime.serviceCount')">
                      {{ detailRecord.service_count }}
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.runtime.hostScope')">
                      {{ projectHostScopeLabel(t, detailRecord.host_scope) }}
                    </t-descriptions-item>
                  </t-descriptions>
                </t-card>

                <t-card size="small" :title="t('project.detail.runtime.authorityTitle')">
                  <t-descriptions size="small" :column="1">
                    <t-descriptions-item :label="t('project.detail.runtime.activityAuthority')">
                      {{ activityAuthorityLabel(detailRecord.activity_authority) }}
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.runtime.workingDirectory')">
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
                    <t-descriptions-item :label="t('project.detail.runtime.canonicalName')">
                      <code>{{ detailRecord.canonical_project_name }}</code>
                    </t-descriptions-item>
                    <t-descriptions-item :label="t('project.detail.runtime.nameSource')">
                      {{ projectCanonicalNameSourceLabel(t, detailRecord.canonical_project_name_source) }}
                    </t-descriptions-item>
                  </t-descriptions>
                </t-card>
              </div>
            </section>
          </t-tab-panel>
        </t-tabs>
      </template>
    </section>
  </div>
</template>
<script setup lang="ts">
import { RefreshIcon } from 'tdesign-icons-vue-next';
import type { TableProps } from 'tdesign-vue-next';
import { DialogPlugin } from 'tdesign-vue-next/es/dialog';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { LOCALE, type LocalizedTitle } from '@/contracts/i18n/locales';
import { getContainerEvents, getContainerLogs } from '@/modules/container/api/container';
import { CONTAINER_BOOTSTRAP_ROUTE } from '@/modules/container/contract/bootstrap';
import type { ContainerLogEntry, ContainerRuntimeEventRecord } from '@/modules/container/types/container';
import { ManagementPageHeader } from '@/shared/components/management';
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
  postProjectRestart,
  postProjectStop,
  postProjectUnregister,
  postProjectUp,
} from '../../api/project';
import ProjectFileEditor from '../../components/ProjectFileEditor.vue';
import ProjectResourcesSection from '../../components/ProjectResourcesSection.vue';
import {
  formatProjectTime,
  projectCanonicalNameSourceLabel,
  projectDriftStatusLabel,
  projectDriftStatusTheme,
  projectHostScopeLabel,
  projectLifecycleActionVisibility,
  projectOwnershipModeLabel,
  projectRefreshStatusLabel,
  projectRefreshStatusTheme,
  projectRuntimeStatusLabel,
  projectRuntimeStatusTheme as runtimeStatusTheme,
} from '../../shared/display';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../../shared/navigation';
import type {
  ProjectActionResponse,
  ProjectActivityAuthority,
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
type ProjectDetailTab =
  'overview' | 'services' | 'containers' | 'networks' | 'volumes' | 'configuration' | 'activity' | 'runtime';

const { locale, t } = useI18n();
const route = useRoute();
const router = useRouter();
const tabsRouterStore = useTabsRouterStore();
const logger = createLogger('project.detail');

const detailRecord = ref<ProjectDetailResponse | null>(null);
const detailLoading = ref(false);
const detailError = ref('');
const activeDetailTab = ref<ProjectDetailTab>(normalizeDetailTab(route.query.tab));
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
const serviceRows = ref<ProjectServiceItem[]>([]);
const serviceLoading = ref(false);
const servicesLoaded = ref(false);
const configurationLoadRequestId = ref(0);
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
const lifecycleActionVisibility = computed(() => projectLifecycleActionVisibility(detailRecord.value?.runtime_status));
const envDraftContent = computed({
  get: () => configurationDraft.env_file_content || '',
  set: (value: string) => {
    configurationDraft.env_file_content = value;
  },
});
const serviceColumns = computed<TableProps['columns']>(() => [
  { colKey: 'service_name', title: t('project.detail.services.columns.service'), minWidth: 220 },
  { colKey: 'build_context', title: t('project.detail.services.columns.buildContext'), minWidth: 220 },
  { colKey: 'declared_networks', title: t('project.detail.services.columns.networks'), minWidth: 220 },
  { colKey: 'declared_ports', title: t('project.detail.services.columns.ports'), minWidth: 220 },
  { colKey: 'declared_volumes', title: t('project.detail.services.columns.volumes'), minWidth: 220 },
  { colKey: 'runtime', title: t('project.detail.services.columns.runtime'), width: 140, align: 'center' },
  { colKey: 'operation', title: t('project.detail.services.columns.operation'), width: 144, align: 'center' },
]);

onMounted(async () => {
  await refreshDetail();
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
  if (currentTab === value) {
    return;
  }
  void router.replace({
    query: {
      ...route.query,
      tab: value,
    },
  });
});

function formatTime(value?: string | null) {
  return formatProjectTime(locale.value, value);
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

function activityAuthorityLabel(value: ProjectActivityAuthority) {
  return value === 'backend-planned'
    ? t('project.detail.activity.authorityBackendPlanned')
    : t('project.detail.activity.authorityFrontendFanout');
}

function activityAuthorityNotice(value: ProjectActivityAuthority) {
  return activityAuthorityLabel(value);
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
    await Promise.all([loadConfiguration(), loadProjectServices(true)]);
    await loadActivity();
  } catch (error) {
    logger.error('failed to load project detail', error);
    detailRecord.value = null;
    activityMembers.value = [];
    activityError.value = '';
    detailError.value = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
  } finally {
    detailLoading.value = false;
  }
}

async function loadConfiguration() {
  if (!Number.isFinite(projectId.value)) return;
  const requestId = configurationLoadRequestId.value + 1;
  configurationLoadRequestId.value = requestId;
  configurationLoading.value = true;
  resetConfigurationState();
  try {
    const [metadata, preview] = await Promise.all([
      getProjectConfiguration(projectId.value),
      getProjectConfigurationPreview(projectId.value),
    ]);
    if (requestId !== configurationLoadRequestId.value) {
      return;
    }
    configurationMetadata.value = metadata;
    configurationPreview.value = preview;
    const firstFile = metadata.compose_files[0]?.id ?? metadata.env_files[0]?.id;
    if (typeof firstFile === 'number') {
      await selectConfigurationFile(firstFile, requestId);
    }
    await hydrateDraftFromCurrent(metadata, requestId);
  } catch (error) {
    logger.error('failed to load project configuration', error);
    if (requestId === configurationLoadRequestId.value) {
      resetConfigurationState();
    }
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.retry')));
  } finally {
    if (requestId === configurationLoadRequestId.value) {
      configurationLoading.value = false;
    }
  }
}

async function selectConfigurationFile(fileId: number, requestId?: number) {
  if (!Number.isFinite(projectId.value)) return;
  try {
    const response = await getProjectConfigurationFile(projectId.value, fileId);
    if (typeof requestId === 'number' && requestId !== configurationLoadRequestId.value) {
      return;
    }
    selectedConfigurationFile.value = response;
  } catch (error) {
    logger.error('failed to load project configuration file', error);
    if (typeof requestId !== 'number' || requestId === configurationLoadRequestId.value) {
      selectedConfigurationFile.value = null;
    }
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

async function hydrateDraftFromCurrent(metadata: ProjectConfigurationMetadataResponse, requestId?: number) {
  const composeFileId = metadata.compose_files[0]?.id;
  const envFileId = metadata.env_files[0]?.id;
  try {
    const [composeResponse, envResponse] = await Promise.all([
      typeof composeFileId === 'number'
        ? getProjectConfigurationFile(projectId.value, composeFileId)
        : Promise.resolve(null),
      typeof envFileId === 'number' ? getProjectConfigurationFile(projectId.value, envFileId) : Promise.resolve(null),
    ]);
    if (typeof requestId === 'number' && requestId !== configurationLoadRequestId.value) {
      return;
    }
    configurationDraft.compose_file_content = composeResponse?.content || '';
    configurationDraft.env_file_content = envResponse?.content || '';
  } catch (error) {
    resetConfigurationState();
    logger.error('failed to hydrate project draft', error);
    throw error;
  }
}

function resetConfigurationState() {
  configurationMetadata.value = null;
  configurationPreview.value = null;
  selectedConfigurationFile.value = null;
  configurationDiffResult.value = null;
  configurationValidateResult.value = null;
  expandedDiffPanels.value = [];
  configurationDraft.compose_file_content = '';
  configurationDraft.env_file_content = '';
}

function resetDraftFromCurrent() {
  if (configurationMetadata.value) {
    void hydrateDraftFromCurrent(configurationMetadata.value);
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
    const services = await loadProjectServices();
    const members = services.flatMap((item) => item.container_members);
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

async function loadProjectServices(forceRefresh = false) {
  if (!Number.isFinite(projectId.value)) {
    serviceRows.value = [];
    servicesLoaded.value = false;
    return [];
  }
  if (servicesLoaded.value && !forceRefresh) {
    return serviceRows.value;
  }

  serviceLoading.value = true;
  try {
    const response = await getProjectServices(projectId.value);
    serviceRows.value = response.items;
    servicesLoaded.value = true;
    return response.items;
  } catch (error) {
    logger.error('failed to load project services', error);
    serviceRows.value = [];
    servicesLoaded.value = false;
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.services.loadFailed')));
    return [];
  } finally {
    serviceLoading.value = false;
  }
}

async function refreshProjectServices() {
  try {
    await loadProjectServices(true);
  } catch {
    // loadProjectServices already reports user-facing feedback.
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

function openFirstServiceContainer(service: ProjectServiceItem) {
  const member = service.container_members[0];
  if (!member) {
    return;
  }
  openContainerDetail(member);
}

async function runLifecycleAction(action: 'up' | 'stop' | 'restart' | 'unregister') {
  if (!Number.isFinite(projectId.value)) return;
  actionLoading.value = action;
  try {
    if (action === 'up') {
      await postProjectUp(projectId.value);
    } else if (action === 'stop') {
      await postProjectStop(projectId.value);
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

function joinList(items: string[]) {
  return items.length > 0 ? items.join(', ') : '-';
}

function buildDetailTitle(name: string): LocalizedTitle {
  return buildDetailTitleWithFallback('project.route.detail.title', name);
}

function normalizeDetailTab(value: unknown): ProjectDetailTab {
  const raw = Array.isArray(value) ? value[0] : value;
  const tabs: ProjectDetailTab[] = [
    'overview',
    'services',
    'containers',
    'networks',
    'volumes',
    'configuration',
    'activity',
    'runtime',
  ];
  return typeof raw === 'string' && tabs.includes(raw as ProjectDetailTab) ? (raw as ProjectDetailTab) : 'overview';
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
.project-tab-panel,
.project-overview-grid,
.project-runtime-grid,
.project-configuration-grid,
.project-file-groups,
.project-detail-copy-row,
.project-service-name,
.project-activity-card,
.project-activity-card__head,
.project-activity-grid,
.project-activity-grid__title,
.project-activity-entries {
  display: flex;
}

.project-detail-page,
.project-detail-body,
.project-tab-panel,
.project-service-name {
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.project-overview-grid,
.project-runtime-grid,
.project-configuration-grid,
.project-activity-grid {
  gap: var(--graft-density-gap-16);
}

.project-overview-grid,
.project-runtime-grid,
.project-configuration-grid {
  align-items: stretch;
}

.project-overview-grid > .t-card,
.project-runtime-grid > .t-card,
.project-configuration-grid > .t-card {
  flex: 1 1 0;
  min-width: 0;
}

.project-detail-tabs :deep(.t-tabs__content) {
  padding-top: var(--graft-density-gap-16);
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
.project-activity-card,
.project-activity-entries {
  flex-direction: column;
}

.project-file-groups,
.project-service-name,
.project-activity-card {
  gap: var(--graft-density-gap-12);
}

.project-service-name span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
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

@media (width <= 768px) {
  .project-overview-grid,
  .project-runtime-grid,
  .project-configuration-grid,
  .project-activity-grid {
    flex-direction: column;
  }

  .project-activity-card__head {
    flex-direction: column;
  }
}
</style>
