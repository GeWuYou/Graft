<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <advanced-query-list-page
    root-class="scheduled-task-page"
    page-type="list-form-detail"
    title-key="scheduledTask.list.title"
    :title="t('scheduledTask.list.title')"
    description-key="scheduledTask.list.description"
    :description="t('scheduledTask.list.description')"
    :error-message="errorMessage"
    :error-title="t('scheduledTask.list.loadError')"
    :loading="loading"
    :reload-label="t('scheduledTask.list.refresh')"
    :retry-label="t('scheduledTask.list.refresh')"
    :source="{ labelKey: 'scheduledTask.list.eyebrow', fallback: t('scheduledTask.list.eyebrow') }"
    @reload="refreshTasks"
  >
    <template #actions>
      <t-button theme="default" variant="outline" @click="columnDrawerVisible = true">
        {{ t('scheduledTask.list.columnSettings') }}
      </t-button>
      <t-button v-permission="permissionCodes.CREATE" theme="primary" @click="openCreateDrawer">
        <template #icon><add-icon /></template>
        {{ t('scheduledTask.list.create') }}
      </t-button>
    </template>
    <template #feedback-extra>
      <section class="scheduled-task-metrics" :aria-label="t('scheduledTask.list.metricsAriaLabel')">
        <t-card
          v-for="metric in overviewMetrics"
          :key="metric.key"
          class="scheduled-task-metric-card"
          size="small"
          :bordered="true"
        >
          <p>{{ metric.label }}</p>
          <strong>{{ metric.value }}</strong>
          <span>{{ metric.description }}</span>
        </t-card>
      </section>
    </template>
    <template #filters>
      <div class="scheduled-task-toolbar">
        <t-input
          v-model="filters.keyword"
          class="scheduled-task-toolbar__search"
          clearable
          :placeholder="t('scheduledTask.list.filters.searchPlaceholder')"
        >
          <template #prefix-icon><search-icon /></template>
        </t-input>
        <t-select
          v-model="filters.jobKey"
          class="scheduled-task-toolbar__select"
          :placeholder="t('scheduledTask.list.filters.jobType')"
        >
          <t-option value="all" :label="t('scheduledTask.list.filters.allJobTypes')" />
          <t-option-group
            v-for="group in groupedJobDefinitions"
            :key="group.module"
            :label="moduleDisplayName(group.module)"
          >
            <t-option v-for="job in group.items" :key="job.key" :value="job.key" :label="jobDefinitionTitle(job)" />
          </t-option-group>
        </t-select>
        <t-select
          v-model="filters.status"
          class="scheduled-task-toolbar__select"
          :placeholder="t('scheduledTask.list.filters.status')"
        >
          <t-option value="all" :label="t('scheduledTask.list.filters.allStatuses')" />
          <t-option
            v-for="statusOption in statusOptions"
            :key="statusOption"
            :value="statusOption"
            :label="statusLabel(statusOption)"
          />
        </t-select>
      </div>
    </template>
    <template #table>
      <t-card class="scheduled-task-table-card" :bordered="true">
        <template #header>
          <div class="scheduled-task-table-head">
            <div>
              <h2>{{ t('scheduledTask.list.tableTitle') }}</h2>
              <p>{{ t('scheduledTask.list.tableHint', { count: filteredTasks.length }) }}</p>
            </div>
          </div>
        </template>

        <t-table
          row-key="key"
          :data="filteredTasks"
          :columns="columns"
          :loading="loading"
          table-layout="fixed"
          :table-content-width="tableContentWidth"
          cell-empty-content="-"
          hover
        >
          <template #task="{ row }">
            <div class="scheduled-task-identity">
              <span class="scheduled-task-identity__name">{{ taskDisplayName(row) }}</span>
              <span class="scheduled-task-identity__key">{{ row.key }}</span>
            </div>
          </template>

          <template #job_key="{ row }">
            <t-tag variant="light-outline" theme="primary">
              {{ jobTypeLabel(row.job_key) }}
            </t-tag>
          </template>

          <template #status="{ row }">
            <task-status-tag :status="row.status" />
          </template>

          <template #schedule="{ row }">
            <t-tooltip
              placement="top-left"
              overlay-class-name="scheduled-task-cron-tooltip"
              :overlay-inner-style="cronTooltipOverlayInnerStyle"
            >
              <template #content>
                <div class="scheduled-task-cron-tooltip__content">
                  <div class="scheduled-task-cron-tooltip__item">
                    <span>{{ t('scheduledTask.cron.expression') }}</span>
                    <code>{{ scheduleExpressionText(row) }}</code>
                  </div>
                  <div class="scheduled-task-cron-tooltip__item">
                    <span>{{ t('scheduledTask.cron.description') }}</span>
                    <strong>{{ cronScheduleDescription(row.schedule) }}</strong>
                  </div>
                  <div class="scheduled-task-cron-tooltip__item">
                    <span>{{ t('scheduledTask.cron.timezone') }}</span>
                    <strong>{{ cronTimezone() }}</strong>
                  </div>
                </div>
              </template>
              <div class="scheduled-task-schedule">
                <code class="scheduled-task-mono">{{ scheduleExpressionText(row) }}</code>
                <span class="scheduled-task-schedule__next-run">{{ cronNextRunLine(row.schedule) }}</span>
              </div>
            </t-tooltip>
          </template>

          <template #recent_result="{ row }">
            <div v-if="row.last_run" class="scheduled-task-last-run">
              <task-status-tag :status="row.last_run.status" />
              <span>{{ runResultText(row.last_run) }}</span>
            </div>
            <span v-else class="scheduled-task-muted">{{ t('scheduledTask.list.detail.none') }}</span>
          </template>

          <template #recent_run="{ row }">
            {{ formatTimestamp(row.last_run?.started_at) }}
          </template>

          <template #success_rate="{ row }">
            {{ successRateLabel(row.key) }}
          </template>

          <template #operation="{ row }">
            <t-space class="scheduled-task-actions" size="small" align="center">
              <t-button theme="primary" variant="text" size="small" @click="openDetail(row)">
                <template #icon><browse-icon /></template>
                {{ t('scheduledTask.list.viewDetail') }}
              </t-button>
              <t-button
                v-permission="permissionCodes.RUN"
                theme="primary"
                variant="outline"
                size="small"
                :disabled="!canRunTask(row)"
                :loading="runningTaskKey === row.key"
                @click="openRunDialog(row)"
              >
                <template #icon><play-icon /></template>
                {{ t('scheduledTask.list.run') }}
              </t-button>
              <t-dropdown trigger="click" placement="bottom-right">
                <t-button theme="default" variant="outline" size="small">
                  <template #icon><ellipsis-icon /></template>
                  {{ t('scheduledTask.list.more') }}
                </t-button>
                <t-dropdown-menu>
                  <t-dropdown-item v-permission="permissionCodes.UPDATE" @click="openEditDrawer(row)">
                    <template #prefix-icon><edit-icon /></template>
                    {{ t('scheduledTask.list.edit') }}
                  </t-dropdown-item>
                  <t-dropdown-item
                    v-permission="permissionCodes.ENABLE"
                    :disabled="lifecycleTaskKey === row.key"
                    @click="toggleTaskEnabled(row)"
                  >
                    <template #prefix-icon>
                      <pause-icon v-if="row.enabled" />
                      <play-icon v-else />
                    </template>
                    {{ row.enabled ? t('scheduledTask.list.disable') : t('scheduledTask.list.enable') }}
                  </t-dropdown-item>
                  <t-dropdown-item
                    v-permission="permissionCodes.DELETE"
                    :disabled="isSystemTask(row) || deletingTaskKey === row.key"
                    theme="error"
                    @click="openDeleteDialog(row)"
                  >
                    <template #prefix-icon><delete-icon /></template>
                    {{ t('scheduledTask.list.delete') }}
                  </t-dropdown-item>
                </t-dropdown-menu>
              </t-dropdown>
            </t-space>
          </template>

          <template #empty>
            <div class="scheduled-task-empty">
              <t-empty
                :title="t('scheduledTask.list.emptyTitle')"
                :description="t('scheduledTask.list.emptyDescription')"
              >
                <template #action>
                  <t-button theme="primary" variant="outline" @click="refreshTasks">
                    {{ t('scheduledTask.list.refresh') }}
                  </t-button>
                </template>
              </t-empty>
            </div>
          </template>
        </t-table>
      </t-card>
    </template>

    <template #detail>
      <advanced-query-column-drawer
        v-model:visible="columnDrawerVisible"
        v-model:selected-keys="visibleColumnKeys"
        :columns="columnSettingOptions"
        :title="t('scheduledTask.list.columnSettings')"
      />
      <t-drawer v-model:visible="formVisible" :header="formTitle" size="720px" placement="right" destroy-on-close>
        <t-form :data="taskForm" label-align="top">
          <section v-if="formMode === 'create'" class="scheduled-task-form-section">
            <div class="scheduled-task-form-section__head">
              <h3>{{ t('scheduledTask.list.form.sectionJobType') }}</h3>
            </div>
            <t-form-item :label="t('scheduledTask.list.form.jobType')" name="jobKey">
              <t-select
                v-model="taskForm.jobKey"
                :loading="jobDefinitionsLoading"
                :placeholder="t('scheduledTask.list.form.jobTypePlaceholder')"
                filterable
                @change="handleJobDefinitionChange"
              >
                <t-option-group
                  v-for="group in groupedJobDefinitions"
                  :key="group.module"
                  :label="moduleDisplayName(group.module)"
                >
                  <t-option v-for="job in group.items" :key="job.key" :value="job.key" :label="jobDefinitionTitle(job)">
                    <div class="scheduled-task-job-option">
                      <div class="scheduled-task-job-option__main">
                        <strong>{{ jobDefinitionTitle(job) }}</strong>
                        <span>{{ job.key }}</span>
                      </div>
                      <t-tag size="small" variant="light">{{ moduleDisplayName(job.module) }}</t-tag>
                    </div>
                  </t-option>
                </t-option-group>
              </t-select>
            </t-form-item>
            <t-card
              v-if="selectedJobDefinition"
              class="scheduled-task-job-summary"
              size="small"
              :title="jobDefinitionTitle(selectedJobDefinition)"
              :bordered="true"
            >
              <p>{{ jobDefinitionDescription(selectedJobDefinition) }}</p>
              <t-descriptions size="small" :column="2" table-layout="auto">
                <t-descriptions-item :label="t('scheduledTask.list.form.jobKey')">
                  {{ selectedJobDefinition.key }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('scheduledTask.list.form.module')">
                  <t-tag size="small" variant="light">{{ moduleDisplayName(selectedJobDefinition.module) }}</t-tag>
                </t-descriptions-item>
                <t-descriptions-item :label="t('scheduledTask.list.form.defaultCron')">
                  <code>{{ normalizeCronForForm(selectedJobDefinition.default_cron_expression) }}</code>
                </t-descriptions-item>
                <t-descriptions-item :label="t('scheduledTask.list.form.owner')">
                  {{ selectedJobDefinition.owner }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('scheduledTask.list.form.defaultConfig')" :span="2">
                  <pre class="scheduled-task-json-preview">{{
                    formatJsonPreview(selectedJobDefinition.default_config_json) || t('scheduledTask.list.detail.none')
                  }}</pre>
                </t-descriptions-item>
              </t-descriptions>
              <t-collapse v-if="selectedJobDefinition.config_schema_json" expand-icon-placement="right">
                <t-collapse-panel value="configSchema" :header="t('scheduledTask.list.form.configSchema')">
                  <pre class="scheduled-task-json-preview">{{
                    formatJsonPreview(selectedJobDefinition.config_schema_json)
                  }}</pre>
                </t-collapse-panel>
              </t-collapse>
            </t-card>
          </section>

          <section class="scheduled-task-form-section">
            <div class="scheduled-task-form-section__head">
              <h3>{{ t('scheduledTask.list.form.sectionBasic') }}</h3>
            </div>
            <t-form-item :label="t('scheduledTask.list.form.taskKey')" name="taskKey">
              <t-input
                v-model="taskForm.taskKey"
                :disabled="formMode === 'edit'"
                :placeholder="t('scheduledTask.list.form.taskKeyPlaceholder')"
              />
            </t-form-item>
            <t-form-item :label="t('scheduledTask.list.form.title')" name="title">
              <t-input
                v-model="taskForm.title"
                :disabled="isSystemEdit"
                :placeholder="t('scheduledTask.list.form.titlePlaceholder')"
              />
            </t-form-item>
            <t-form-item :label="t('scheduledTask.list.form.description')" name="description">
              <t-textarea
                v-model="taskForm.description"
                :disabled="isSystemEdit"
                :autosize="{ minRows: 3, maxRows: 5 }"
                :placeholder="t('scheduledTask.list.form.descriptionPlaceholder')"
              />
            </t-form-item>
          </section>

          <section class="scheduled-task-form-section">
            <div class="scheduled-task-form-section__head">
              <h3>{{ t('scheduledTask.list.form.sectionExecutionPlan') }}</h3>
            </div>
            <t-form-item
              class="scheduled-task-cron-form-item"
              :label="t('scheduledTask.list.form.cronExpression')"
              name="cronExpression"
              :status="formFieldErrors.cronExpression ? 'error' : undefined"
              :show-error-message="false"
            >
              <cron-expression-field
                v-model="taskForm.cronExpression"
                :error="formFieldErrors.cronExpression"
                @update:model-value="handleCronEditorUpdate"
                @validate="handleCronEditorValidate"
              />
            </t-form-item>
          </section>

          <section class="scheduled-task-form-section">
            <div class="scheduled-task-form-section__head">
              <h3>{{ t('scheduledTask.list.form.sectionEnabled') }}</h3>
            </div>
            <t-form-item :label="t('scheduledTask.list.form.enabled')" name="enabled">
              <t-switch v-model="taskForm.enabled" />
            </t-form-item>
          </section>

          <section class="scheduled-task-form-section">
            <div class="scheduled-task-form-section__head">
              <h3>{{ t('scheduledTask.list.form.sectionConfig') }}</h3>
              <p>{{ t('scheduledTask.list.form.configHint') }}</p>
            </div>
            <div v-if="drawerConfigSummaryItems.length > 0" class="scheduled-task-config-list">
              <div v-for="item in drawerConfigSummaryItems" :key="item.key" class="scheduled-task-config-list__item">
                <strong>{{ item.label }}</strong>
                <span>{{ item.value }}</span>
                <small v-if="item.description">{{ item.description }}</small>
              </div>
            </div>
            <p v-else class="scheduled-task-muted">{{ t('scheduledTask.list.form.noConfigFields') }}</p>
            <t-button theme="default" variant="outline" @click="openConfigDialog">
              {{ t('scheduledTask.list.configDialog.open') }}
            </t-button>
          </section>

          <section v-if="formMode === 'edit' && jobDefinitionActions.length > 0" class="scheduled-task-form-section">
            <div class="scheduled-task-form-section__head">
              <h3>{{ t('scheduledTask.list.action.sectionTitle') }}</h3>
              <p>{{ t('scheduledTask.list.action.sectionHint') }}</p>
            </div>
            <t-space class="scheduled-task-action-buttons" size="small">
              <t-button
                v-for="action in jobDefinitionActions"
                :key="action.key"
                :theme="actionButtonTheme(action)"
                variant="outline"
                :loading="actionExecutingKey === action.key"
                @click="handleActionClick(action)"
              >
                {{ jobDefinitionActionTitle(action) }}
              </t-button>
            </t-space>
          </section>
        </t-form>

        <template #footer>
          <t-space class="scheduled-task-drawer-footer">
            <t-button theme="default" variant="outline" @click="formVisible = false">
              {{ t('scheduledTask.list.cancel') }}
            </t-button>
            <t-button theme="primary" :loading="submittingTask" @click="submitTaskForm">
              {{ formMode === 'create' ? t('scheduledTask.list.create') : t('scheduledTask.list.save') }}
            </t-button>
          </t-space>
        </template>
      </t-drawer>

      <t-dialog
        v-model:visible="configDialogVisible"
        :header="configDialogTitle"
        :confirm-btn="t('scheduledTask.list.configDialog.confirm')"
        :cancel-btn="t('scheduledTask.list.cancel')"
        :confirm-loading="configDialogSaving"
        width="760px"
        destroy-on-close
        @confirm="confirmConfigDialog"
      >
        <div class="scheduled-task-dialog-copy">
          <t-form :data="taskForm" label-align="top">
            <section class="scheduled-task-config-section">
              <div class="scheduled-task-config-section__head">
                <h3>{{ t('scheduledTask.list.form.sectionBasicConfig') }}</h3>
                <p>{{ configDialogBehaviorSummary }}</p>
              </div>
              <template v-if="persistentConfigSchemaFields.length > 0">
                <t-form-item
                  v-for="field in persistentConfigSchemaFields"
                  :key="field.key"
                  :label="configSchemaFieldTitle(field)"
                  :name="`config.${field.key}`"
                  :help="configSchemaFieldDescription(field)"
                  :required-mark="field.required"
                >
                  <div v-if="isRetentionDaysField(field)" class="scheduled-task-retention-field">
                    <t-radio-group
                      :model-value="retentionDaysOptionValue()"
                      variant="default-filled"
                      @change="handleRetentionDaysOptionChange"
                    >
                      <t-radio-button v-for="days in RETENTION_DAY_PRESETS" :key="days" :value="days">
                        {{ t('scheduledTask.list.configDialog.retentionDaysOption', { days }) }}
                      </t-radio-button>
                      <t-radio-button :value="CUSTOM_RETENTION_DAY_VALUE">
                        {{ t('scheduledTask.list.configDialog.customRetentionDays') }}
                      </t-radio-button>
                    </t-radio-group>
                    <t-input-number
                      v-if="retentionDaysOptionValue() === CUSTOM_RETENTION_DAY_VALUE"
                      :model-value="configNumberValue(field.key)"
                      :min="field.schema.minimum"
                      :max="field.schema.maximum"
                      :decimal-places="0"
                      :placeholder="t('scheduledTask.list.form.configNumberPlaceholder')"
                      @change="(value) => updateConfigField(field.key, value)"
                    />
                  </div>
                  <t-select
                    v-else-if="field.schema.enum"
                    :model-value="configSelectValue(field.key)"
                    :placeholder="t('scheduledTask.list.form.configSelectPlaceholder')"
                    clearable
                    @change="(value) => updateConfigField(field.key, value)"
                  >
                    <t-option
                      v-for="option in field.schema.enum"
                      :key="String(option)"
                      :value="option"
                      :label="String(option)"
                    />
                  </t-select>
                  <t-input-number
                    v-else-if="field.schema.type === 'integer' || field.schema.type === 'number'"
                    :model-value="configNumberValue(field.key)"
                    :min="field.schema.minimum"
                    :max="field.schema.maximum"
                    :decimal-places="field.schema.type === 'integer' ? 0 : undefined"
                    :placeholder="t('scheduledTask.list.form.configNumberPlaceholder')"
                    @change="(value) => updateConfigField(field.key, value)"
                  />
                  <t-switch
                    v-else-if="field.schema.type === 'boolean'"
                    :model-value="Boolean(taskForm.config[field.key])"
                    @change="(value) => updateConfigField(field.key, value)"
                  />
                  <t-input
                    v-else
                    :model-value="configStringValue(field.key)"
                    :minlength="field.schema.minLength"
                    :maxlength="field.schema.maxLength"
                    :placeholder="t('scheduledTask.list.form.configStringPlaceholder')"
                    clearable
                    @change="(value) => updateConfigField(field.key, value)"
                  />
                </t-form-item>
              </template>
              <p v-else class="scheduled-task-muted">{{ t('scheduledTask.list.form.noConfigFields') }}</p>
            </section>
            <t-collapse expand-icon-placement="right">
              <t-collapse-panel value="advancedConfig" :header="t('scheduledTask.list.form.sectionAdvancedConfig')">
                <div class="scheduled-task-advanced-config">
                  <config-json-editor
                    v-model="taskForm.configJson"
                    v-model:mode="configDialogJsonMode"
                    :done-label="t('scheduledTask.list.configDialog.doneJson')"
                    :edit-label="t('scheduledTask.list.configDialog.editJson')"
                    :editor-label="t('scheduledTask.list.form.configJson')"
                    :error="formFieldErrors.configJson"
                    :format-label="t('scheduledTask.list.form.formatJson')"
                    :placeholder="t('scheduledTask.list.form.configJsonPlaceholder')"
                    :preview-text="formatJsonPreview(taskForm.configJson) || t('scheduledTask.list.detail.none')"
                    :title="t('scheduledTask.list.configDialog.jsonPreview')"
                    @change="handleConfigJsonChange"
                    @format="formatConfigJson"
                  />
                  <section>
                    <strong>{{ t('scheduledTask.list.configDialog.schemaDebug') }}</strong>
                    <pre class="scheduled-task-json-preview">{{
                      formatJsonPreview(selectedJobDefinition?.config_schema_json) ||
                      t('scheduledTask.list.detail.none')
                    }}</pre>
                  </section>
                </div>
              </t-collapse-panel>
            </t-collapse>
          </t-form>
        </div>
      </t-dialog>

      <t-dialog
        v-model:visible="actionConfirmDialogVisible"
        :header="t('scheduledTask.list.action.confirmTitle')"
        :confirm-btn="t('scheduledTask.list.action.confirm')"
        :cancel-btn="t('scheduledTask.list.cancel')"
        :confirm-loading="Boolean(selectedAction && actionExecutingKey === selectedAction.key)"
        width="640px"
        destroy-on-close
        @confirm="confirmSelectedAction"
      >
        <div v-if="selectedAction && editingTask" class="scheduled-task-dialog-copy">
          <t-alert theme="info" :message="t('scheduledTask.list.action.previewWarning')" />
          <t-descriptions :column="1" bordered size="small">
            <t-descriptions-item :label="t('scheduledTask.list.action.taskName')">
              {{ taskDisplayName(editingTask) }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.action.behavior')">
              {{ jobDefinitionActionDescription(selectedAction) }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.action.affectedResource')">
              {{ jobDefinitionActionAffectedResource(selectedAction, editingTask) }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.action.currentConfig')">
              <div v-if="drawerConfigSummaryItems.length > 0" class="scheduled-task-config-list">
                <div v-for="item in drawerConfigSummaryItems" :key="item.key" class="scheduled-task-config-list__item">
                  <strong>{{ item.label }}</strong>
                  <span>{{ item.value }}</span>
                </div>
              </div>
              <span v-else>{{ t('scheduledTask.list.detail.none') }}</span>
            </t-descriptions-item>
          </t-descriptions>
        </div>
      </t-dialog>

      <t-dialog
        v-model:visible="actionResultDialogVisible"
        :header="t('scheduledTask.list.actionResult.title')"
        :confirm-btn="t('scheduledTask.list.actionResult.confirm')"
        :cancel-btn="null"
        width="720px"
        destroy-on-close
        @confirm="closeActionResultDialog"
        @close="closeActionResultDialog"
      >
        <div v-if="actionResult" class="scheduled-task-dialog-copy">
          <t-alert v-if="actionResultStructured.summary" theme="success" :message="actionResultStructured.summary" />
          <t-descriptions :column="1" bordered size="small">
            <t-descriptions-item v-if="actionResultStructured.stage" :label="t('scheduledTask.list.detail.stage')">
              <t-tag theme="primary" variant="light">{{ actionResultStructured.stage }}</t-tag>
            </t-descriptions-item>
            <t-descriptions-item
              v-if="actionResultStructured.affected_resource"
              :label="t('scheduledTask.list.detail.affectedResource')"
            >
              {{ actionResultStructured.affected_resource }}
            </t-descriptions-item>
            <t-descriptions-item
              v-if="Object.keys(actionResultStructured.metrics ?? {}).length > 0"
              :label="t('scheduledTask.list.detail.metrics')"
            >
              <pre class="scheduled-task-json-preview">{{
                JSON.stringify(actionResultStructured.metrics, null, 2)
              }}</pre>
            </t-descriptions-item>
            <t-descriptions-item
              v-if="Object.keys(actionResultStructured.details ?? {}).length > 0"
              :label="t('scheduledTask.list.detail.details')"
            >
              <pre class="scheduled-task-json-preview">{{
                JSON.stringify(actionResultStructured.details, null, 2)
              }}</pre>
            </t-descriptions-item>
            <t-descriptions-item
              v-if="(actionResultStructured.warnings?.length ?? 0) > 0"
              :label="t('scheduledTask.list.detail.warnings')"
            >
              <ul class="scheduled-task-warning-list">
                <li v-for="warning in actionResultStructured.warnings" :key="warning">{{ warning }}</li>
              </ul>
            </t-descriptions-item>
          </t-descriptions>
          <t-collapse expand-icon-placement="right">
            <t-collapse-panel value="rawResultJson" :header="t('scheduledTask.list.detail.rawResultJson')">
              <pre class="scheduled-task-json-preview">{{ actionResultRawJsonPreview }}</pre>
            </t-collapse-panel>
          </t-collapse>
        </div>
      </t-dialog>

      <t-drawer
        v-model:visible="detailVisible"
        :header="detailTitle"
        size="840px"
        placement="right"
        destroy-on-close
        :footer="false"
      >
        <div v-if="selectedTask" class="scheduled-task-detail">
          <section class="scheduled-task-detail__section">
            <div class="scheduled-task-detail__section-head">
              <h3>{{ t('scheduledTask.list.detail.basics') }}</h3>
              <task-status-tag :status="selectedTask.status" />
            </div>
            <t-descriptions :column="1" bordered size="small">
              <t-descriptions-item :label="t('scheduledTask.list.detail.key')">
                {{ selectedTask.key }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.title')">
                {{ taskDisplayName(selectedTask) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.description')">
                {{ taskDescription(selectedTask) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.owner')">
                {{ selectedTask.owner }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.module')">
                {{ selectedTask.module }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.jobType')">
                {{ jobTypeLabel(selectedTask.job_key) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.jobKey')">
                {{ selectedTask.job_key }}
              </t-descriptions-item>
            </t-descriptions>
          </section>

          <section class="scheduled-task-detail__section">
            <h3>{{ t('scheduledTask.list.detail.behavior') }}</h3>
            <t-descriptions :column="1" bordered size="small">
              <t-descriptions-item :label="t('scheduledTask.list.detail.jobBehavior')">
                {{ selectedTaskJobDescription }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.configItems')">
                <div v-if="selectedTaskConfigFields.length > 0" class="scheduled-task-config-list">
                  <div
                    v-for="field in selectedTaskConfigFields"
                    :key="field.key"
                    class="scheduled-task-config-list__item"
                  >
                    <strong>{{ configSchemaFieldTitle(field) }}</strong>
                    <span>{{ configValuePreview(selectedTaskEffectiveConfig[field.key]) }}</span>
                    <small v-if="configSchemaFieldDescription(field)">{{ configSchemaFieldDescription(field) }}</small>
                  </div>
                </div>
                <span v-else>{{ t('scheduledTask.list.detail.none') }}</span>
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.effectiveConfig')">
                <pre class="scheduled-task-json-preview">{{ selectedTaskEffectiveConfigPreview }}</pre>
              </t-descriptions-item>
            </t-descriptions>
            <t-collapse expand-icon-placement="right">
              <t-collapse-panel value="rawConfig" :header="t('scheduledTask.list.detail.advancedConfig')">
                <div class="scheduled-task-raw-config">
                  <strong>{{ t('scheduledTask.list.detail.configJson') }}</strong>
                  <pre class="scheduled-task-json-preview">{{
                    formatJsonPreview(selectedTask.config_json) || t('scheduledTask.list.detail.none')
                  }}</pre>
                  <strong>{{ t('scheduledTask.list.detail.effectiveConfig') }}</strong>
                  <pre class="scheduled-task-json-preview">{{ selectedTaskEffectiveConfigPreview }}</pre>
                </div>
              </t-collapse-panel>
            </t-collapse>
          </section>

          <section class="scheduled-task-detail__section">
            <h3>{{ t('scheduledTask.list.detail.schedule') }}</h3>
            <t-descriptions :column="1" bordered size="small">
              <t-descriptions-item :label="t('scheduledTask.list.detail.scheduleType')">
                {{ scheduleTypeLabel(selectedTask.schedule_type) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.scheduleRule')">
                <span class="scheduled-task-mono">{{ selectedTask.schedule }}</span>
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.nextRun')">
                {{ formatTimestamp(selectedTask.next_run_at) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.enabled')">
                {{ booleanLabel(selectedTask.enabled) }}
              </t-descriptions-item>
            </t-descriptions>
          </section>

          <section class="scheduled-task-detail__section">
            <h3>{{ t('scheduledTask.list.detail.runtime') }}</h3>
            <t-descriptions :column="1" bordered size="small">
              <t-descriptions-item :label="t('scheduledTask.list.detail.running')">
                {{ booleanLabel(selectedTask.running) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.status')">
                <task-status-tag :status="selectedTask.status" />
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.successRate')">
                {{ successRateLabel(selectedTask.key) }}
              </t-descriptions-item>
            </t-descriptions>
          </section>

          <section class="scheduled-task-detail__section">
            <h3>{{ t('scheduledTask.list.detail.latestResult') }}</h3>
            <t-descriptions v-if="selectedTask.last_run" :column="1" bordered size="small">
              <t-descriptions-item :label="t('scheduledTask.list.detail.runId')">
                {{ selectedTask.last_run.id }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.triggerType')">
                {{ triggerLabel(selectedTask.last_run.trigger_type) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.status')">
                <task-status-tag :status="selectedTask.last_run.status" />
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.startedAt')">
                {{ formatTimestamp(selectedTask.last_run.started_at) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.finishedAt')">
                {{ formatTimestamp(selectedTask.last_run.finished_at) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.duration')">
                {{ formatDuration(selectedTask.last_run.duration_ms) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('scheduledTask.list.detail.result')">
                {{ runResultText(selectedTask.last_run) }}
              </t-descriptions-item>
              <t-descriptions-item
                v-if="runResultStructured(selectedTask.last_run).stage"
                :label="t('scheduledTask.list.detail.stage')"
              >
                {{ runResultStructured(selectedTask.last_run).stage }}
              </t-descriptions-item>
              <t-descriptions-item
                v-if="runResultStructured(selectedTask.last_run).affected_resource"
                :label="t('scheduledTask.list.detail.affectedResource')"
              >
                {{ runResultStructured(selectedTask.last_run).affected_resource }}
              </t-descriptions-item>
              <t-descriptions-item
                v-if="Object.keys(runResultStructured(selectedTask.last_run).metrics ?? {}).length > 0"
                :label="t('scheduledTask.list.detail.metrics')"
              >
                <pre class="scheduled-task-json-preview">{{
                  JSON.stringify(runResultStructured(selectedTask.last_run).metrics, null, 2)
                }}</pre>
              </t-descriptions-item>
            </t-descriptions>
            <p v-else class="scheduled-task-muted">{{ t('scheduledTask.list.detail.none') }}</p>
          </section>

          <section class="scheduled-task-detail__section">
            <div class="scheduled-task-detail__section-head">
              <h3>{{ t('scheduledTask.list.detail.recentRuns') }}</h3>
              <t-button size="small" theme="default" variant="outline" :loading="runsLoading" @click="refreshRuns">
                {{ t('scheduledTask.list.refresh') }}
              </t-button>
            </div>
            <t-table
              row-key="id"
              size="small"
              :data="recentRuns"
              :columns="runColumns"
              :loading="runsLoading"
              table-layout="fixed"
              table-content-width="860"
              cell-empty-content="-"
            >
              <template #started_at="{ row }">
                {{ formatTimestamp(row.started_at) }}
              </template>
              <template #trigger_type="{ row }">
                {{ triggerLabel(row.trigger_type) }}
              </template>
              <template #status="{ row }">
                <task-status-tag :status="row.status" />
              </template>
              <template #duration_ms="{ row }">
                {{ formatDuration(row.duration_ms) }}
              </template>
              <template #result="{ row }">
                {{ runResultText(row) }}
              </template>
              <template #operation="{ row }">
                <t-button theme="primary" variant="text" size="small" @click="openRunDetail(row)">
                  {{ t('scheduledTask.list.detail.viewRun') }}
                </t-button>
              </template>
              <template #empty>
                <div class="scheduled-task-runs-empty">
                  {{ t('scheduledTask.list.detail.runsEmpty') }}
                </div>
              </template>
            </t-table>
          </section>
        </div>
      </t-drawer>

      <t-dialog
        v-model:visible="runDialogVisible"
        :header="t('scheduledTask.list.runDialog.title')"
        :confirm-btn="t('scheduledTask.list.runDialog.confirm')"
        :cancel-btn="t('scheduledTask.list.runDialog.cancel')"
        :confirm-loading="runningTaskKey === runDialogTask?.key"
        @confirm="confirmRunTask"
      >
        <div v-if="runDialogTask" class="scheduled-task-dialog-copy">
          <p>{{ t('scheduledTask.list.runDialog.taskLine', { taskName: taskDisplayName(runDialogTask) }) }}</p>
          <p>{{ immediateRunSummary(runDialogTask).description }}</p>
          <t-descriptions :column="1" bordered size="small">
            <t-descriptions-item
              v-for="item in immediateRunSummary(runDialogTask).items"
              :key="item.key"
              :label="item.label"
            >
              {{ item.value }}
            </t-descriptions-item>
          </t-descriptions>
        </div>
      </t-dialog>

      <t-dialog
        v-model:visible="deleteDialogVisible"
        :header="t('scheduledTask.list.deleteDialog.title')"
        :confirm-btn="t('scheduledTask.list.deleteDialog.confirm')"
        :cancel-btn="t('scheduledTask.list.cancel')"
        :confirm-loading="deletingTaskKey === deleteDialogTask?.key"
        @confirm="confirmDeleteTask"
      >
        <p v-if="deleteDialogTask">
          {{ t('scheduledTask.list.deleteDialog.description', { taskName: taskDisplayName(deleteDialogTask) }) }}
        </p>
      </t-dialog>

      <t-dialog
        v-model:visible="runDetailVisible"
        :header="t('scheduledTask.list.detail.runDetailTitle')"
        :footer="false"
        width="640px"
      >
        <t-descriptions v-if="selectedRun" :column="1" bordered size="small">
          <t-descriptions-item :label="t('scheduledTask.list.detail.runId')">
            {{ selectedRun.id }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('scheduledTask.list.columns.task')">
            {{ selectedRun.task_name || selectedRun.task_key }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('scheduledTask.list.detail.triggerType')">
            {{ triggerLabel(selectedRun.trigger_type) }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('scheduledTask.list.detail.status')">
            <task-status-tag :status="selectedRun.status" />
          </t-descriptions-item>
          <t-descriptions-item :label="t('scheduledTask.list.detail.startedAt')">
            {{ formatTimestamp(selectedRun.started_at) }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('scheduledTask.list.detail.finishedAt')">
            {{ formatTimestamp(selectedRun.finished_at) }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('scheduledTask.list.detail.duration')">
            {{ formatDuration(selectedRun.duration_ms) }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('scheduledTask.list.detail.result')">
            {{ runResultText(selectedRun) }}
          </t-descriptions-item>
          <t-descriptions-item
            v-if="runResultStructured(selectedRun).stage"
            :label="t('scheduledTask.list.detail.stage')"
          >
            {{ runResultStructured(selectedRun).stage }}
          </t-descriptions-item>
          <t-descriptions-item
            v-if="runResultStructured(selectedRun).affected_resource"
            :label="t('scheduledTask.list.detail.affectedResource')"
          >
            {{ runResultStructured(selectedRun).affected_resource }}
          </t-descriptions-item>
          <t-descriptions-item
            v-if="Object.keys(runResultStructured(selectedRun).metrics ?? {}).length > 0"
            :label="t('scheduledTask.list.detail.metrics')"
          >
            <pre class="scheduled-task-json-preview">{{
              JSON.stringify(runResultStructured(selectedRun).metrics, null, 2)
            }}</pre>
          </t-descriptions-item>
          <t-descriptions-item
            v-if="Object.keys(runResultStructured(selectedRun).details ?? {}).length > 0"
            :label="t('scheduledTask.list.detail.details')"
          >
            <pre class="scheduled-task-json-preview">{{
              JSON.stringify(runResultStructured(selectedRun).details, null, 2)
            }}</pre>
          </t-descriptions-item>
          <t-descriptions-item
            v-if="(runResultStructured(selectedRun).warnings?.length ?? 0) > 0"
            :label="t('scheduledTask.list.detail.warnings')"
          >
            <ul class="scheduled-task-warning-list">
              <li v-for="warning in runResultStructured(selectedRun).warnings" :key="warning">{{ warning }}</li>
            </ul>
          </t-descriptions-item>
          <t-descriptions-item :label="t('scheduledTask.list.detail.rawResultJson')">
            <t-collapse expand-icon-placement="right">
              <t-collapse-panel value="rawResultJson" :header="t('scheduledTask.list.detail.rawResultJson')">
                <pre class="scheduled-task-json-preview">{{
                  formatJsonPreview(selectedRun.result_json) || t('scheduledTask.list.detail.none')
                }}</pre>
              </t-collapse-panel>
            </t-collapse>
          </t-descriptions-item>
        </t-descriptions>
      </t-dialog>
    </template>
  </advanced-query-list-page>
</template>
<script setup lang="ts">
import {
  AddIcon,
  BrowseIcon,
  DeleteIcon,
  EditIcon,
  EllipsisIcon,
  PauseIcon,
  PlayIcon,
  SearchIcon,
} from 'tdesign-icons-vue-next';
import { MessagePlugin, Tag, type TdBaseTableProps } from 'tdesign-vue-next';
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { requestNotificationHeaderRefresh } from '@/modules/notification/contract/refresh';
import { readErrorField } from '@/modules/shared/error-field';
import { buildVisibleColumns, calculateTableContentWidth } from '@/shared/components/management';
import { AdvancedQueryColumnDrawer, AdvancedQueryListPage } from '@/shared/components/query-list';
import type { ApiRequestError } from '@/types/axios';
import { createLogger } from '@/utils/logger';

import {
  createScheduledTask,
  deleteScheduledTask,
  disableScheduledTask,
  enableScheduledTask,
  executeScheduledTaskAction,
  getScheduledTask,
  getScheduledTaskJobDefinition,
  getScheduledTaskJobDefinitions,
  getScheduledTaskRun,
  getScheduledTaskRuns,
  getScheduledTasks,
  runScheduledTask,
  updateScheduledTask,
} from '../../api/scheduled-task';
import ConfigJsonEditor from '../../components/ConfigJsonEditor.vue';
import CronExpressionField from '../../components/CronExpressionField.vue';
import { SCHEDULED_TASK_PERMISSION_CODE } from '../../contract/permissions';
import type {
  CreateScheduledTaskRequest,
  ScheduledTaskActionRequest,
  ScheduledTaskActionResult,
  ScheduledTaskItem,
  ScheduledTaskJobDefinitionAction,
  ScheduledTaskJobDefinitionItem,
  ScheduledTaskJobDefinitionItemWithActions,
  ScheduledTaskJobKey,
  ScheduledTaskRunItem,
  ScheduledTaskRunStatus,
  ScheduledTaskRunTriggerType,
  ScheduledTaskStatus,
  UpdateScheduledTaskRequest,
} from '../../types/scheduled-task';
import type { ConfigSchema, ConfigSchemaField, ConfigValidationIssue } from '../../utils/config-schema';
import {
  buildDefaultConfigFromSchema,
  getConfigSchemaFields,
  mergeConfigRecords,
  parseConfigSchema,
  validateConfigRecord,
} from '../../utils/config-schema';
import {
  type CronValidationResult,
  formatCronExpression,
  getCronDescription,
  getNextRunText,
  normalizeCronExpression,
  validateCronExpression,
} from '../../utils/cron';
import { translateCronValidation } from '../../utils/cron-i18n';
import { formatJsonPreview, type JsonRecord, parseJsonRecord } from '../../utils/json';
import { parseRunResult, type ScheduledTaskRunResult } from '../../utils/run-result';

defineOptions({
  name: 'ScheduledTaskListPage',
});

type TaskFormModel = {
  taskKey: string;
  title: string;
  description: string;
  cronExpression: string;
  enabled: boolean;
  jobKey: ScheduledTaskJobKey | '';
  config: JsonRecord;
  configJson: string;
  configDirty: boolean;
  taskConfigJson: string;
};

type FormFieldErrors = {
  cronExpression: string;
  configJson: string;
};

type FilterModel = {
  keyword: string;
  jobKey: ScheduledTaskJobKey | 'all';
  status: ScheduledTaskStatus | 'all';
};

type FormMode = 'create' | 'edit';

type RunSummary = {
  runs24h: number;
  failures24h: number;
};

type JobDefinitionGroup = {
  module: string;
  items: ScheduledTaskJobDefinitionItem[];
};

type ImmediateRunSummary = {
  description: string;
  items: Array<{
    key: string;
    label: string;
    value: string;
  }>;
};

const DEFAULT_CRON_EXPRESSION = '0 */5 * * * *';
const RETENTION_DAY_PRESETS = [1, 7, 30, 90] as const;
const CUSTOM_RETENTION_DAY_VALUE = 'custom';
const cronTooltipOverlayInnerStyle = {
  maxWidth: '280px',
  padding: 'var(--graft-density-gap-12)',
};

const statusOptions: ScheduledTaskStatus[] = ['idle', 'running', 'success', 'failed', 'unknown'];
const builtinTaskMessageKeys = [
  'scheduledTask.accessLogRetention.title',
  'scheduledTask.accessLogRetention.description',
  'scheduledTask.auditLogRetention.title',
  'scheduledTask.auditLogRetention.description',
  'scheduledTask.appLogRetention.title',
  'scheduledTask.appLogRetention.description',
  'scheduledTask.action.dryRun.title',
  'scheduledTask.action.dryRun.description',
  'scheduledTask.accessLogRetention.config.retentionDays.title',
  'scheduledTask.accessLogRetention.config.retentionDays.description',
  'scheduledTask.accessLogRetention.config.batchSize.title',
  'scheduledTask.accessLogRetention.config.batchSize.description',
  'scheduledTask.auditLogRetention.config.retentionDays.title',
  'scheduledTask.auditLogRetention.config.retentionDays.description',
  'scheduledTask.auditLogRetention.config.dryRun.title',
  'scheduledTask.auditLogRetention.config.dryRun.description',
  'scheduledTask.auditLogRetention.config.batchSize.title',
  'scheduledTask.auditLogRetention.config.batchSize.description',
  'scheduledTask.appLogRetention.config.retentionDays.title',
  'scheduledTask.appLogRetention.config.retentionDays.description',
  'scheduledTask.appLogRetention.config.dryRun.title',
  'scheduledTask.appLogRetention.config.dryRun.description',
  'scheduledTask.appLogRetention.config.batchSize.title',
  'scheduledTask.appLogRetention.config.batchSize.description',
] as const;

type BuiltinTaskMessageKey = (typeof builtinTaskMessageKeys)[number];

const TaskStatusTag = defineComponent({
  name: 'ScheduledTaskStatusTag',
  props: {
    status: {
      type: String,
      required: true,
    },
  },
  setup(props) {
    const { t } = useI18n();

    return () =>
      h(
        Tag,
        {
          theme: statusTheme(props.status as ScheduledTaskStatus | ScheduledTaskRunStatus),
          variant: 'light',
          class: 'scheduled-task-status-tag',
        },
        () => statusLabel(props.status as ScheduledTaskStatus | ScheduledTaskRunStatus, t),
      );
  },
});

const { t, te, locale } = useI18n();
const logger = createLogger('scheduled-task.list.page');
const permissionCodes = SCHEDULED_TASK_PERMISSION_CODE;

const tasks = ref<ScheduledTaskItem[]>([]);
const jobDefinitions = ref<ScheduledTaskJobDefinitionItem[]>([]);
const selectedTask = ref<ScheduledTaskItem | null>(null);
const recentRuns = ref<ScheduledTaskRunItem[]>([]);
const runHistoryByTaskKey = ref<Record<string, ScheduledTaskRunItem[]>>({});
const selectedRun = ref<ScheduledTaskRunItem | null>(null);
const loading = ref(false);
const jobDefinitionsLoading = ref(false);
const runsLoading = ref(false);
const detailVisible = ref(false);
const formVisible = ref(false);
const configDialogVisible = ref(false);
const runDialogVisible = ref(false);
const deleteDialogVisible = ref(false);
const runDetailVisible = ref(false);
const actionConfirmDialogVisible = ref(false);
const actionResultDialogVisible = ref(false);
const columnDrawerVisible = ref(false);
const errorMessage = ref('');
const runningTaskKey = ref('');
const actionExecutingKey = ref('');
const lifecycleTaskKey = ref('');
const deletingTaskKey = ref('');
const submittingTask = ref(false);
const configDialogSaving = ref(false);
const formMode = ref<FormMode>('create');
const editingTask = ref<ScheduledTaskItem | null>(null);
const runDialogTask = ref<ScheduledTaskItem | null>(null);
const deleteDialogTask = ref<ScheduledTaskItem | null>(null);
const selectedAction = ref<ScheduledTaskJobDefinitionAction | null>(null);
const actionResult = ref<ScheduledTaskActionResult | null>(null);
const customRetentionDaysSelected = ref(false);
const configDialogJsonMode = ref<'preview' | 'edit'>('preview');

const filters = reactive<FilterModel>({
  keyword: '',
  jobKey: 'all',
  status: 'all',
});

const taskForm = reactive<TaskFormModel>(createEmptyTaskForm());
const formFieldErrors = reactive<FormFieldErrors>({
  cronExpression: '',
  configJson: '',
});
const visibleColumnKeys = ref(['task', 'job_key', 'status', 'schedule', 'recent_result', 'recent_run']);

const filteredTasks = computed(() => {
  const keyword = filters.keyword.trim().toLowerCase();
  return tasks.value.filter((task) => {
    const matchesKeyword =
      !keyword ||
      [task.key, task.job_key, taskDisplayName(task), taskDescription(task), task.owner, task.module]
        .filter(Boolean)
        .some((value) => value.toLowerCase().includes(keyword));
    const matchesJob = filters.jobKey === 'all' || task.job_key === filters.jobKey;
    const matchesStatus = filters.status === 'all' || task.status === filters.status;
    return matchesKeyword && matchesJob && matchesStatus;
  });
});

const recentRunSummary = computed<RunSummary>(() => {
  const since = Date.now() - 24 * 60 * 60 * 1000;
  const allRuns = Object.values(runHistoryByTaskKey.value).flat();
  const recent = allRuns.filter((run) => new Date(run.started_at).getTime() >= since);

  return {
    runs24h: recent.length,
    failures24h: recent.filter((run) => run.status === 'failed').length,
  };
});

const overviewMetrics = computed(() => [
  {
    key: 'total',
    label: t('scheduledTask.list.metric.total'),
    value: tasks.value.length,
    description: t('scheduledTask.list.metric.totalDescription'),
  },
  {
    key: 'enabled',
    label: t('scheduledTask.list.metric.enabled'),
    value: tasks.value.filter((task) => task.enabled).length,
    description: t('scheduledTask.list.metric.enabledDescription'),
  },
  {
    key: 'runs24h',
    label: t('scheduledTask.list.metric.runs24h'),
    value: recentRunSummary.value.runs24h,
    description: t('scheduledTask.list.metric.runs24hDescription'),
  },
  {
    key: 'failures24h',
    label: t('scheduledTask.list.metric.failures24h'),
    value: recentRunSummary.value.failures24h,
    description: t('scheduledTask.list.metric.failures24hDescription'),
  },
]);

const detailTitle = computed(() =>
  selectedTask.value
    ? t('scheduledTask.list.detail.titleWithName', { name: taskDisplayName(selectedTask.value) })
    : t('scheduledTask.list.detail.title'),
);

const formTitle = computed(() =>
  formMode.value === 'create' ? t('scheduledTask.list.form.createTitle') : t('scheduledTask.list.form.editTitle'),
);

const isSystemEdit = computed(
  () => formMode.value === 'edit' && editingTask.value !== null && isSystemTask(editingTask.value),
);

const selectedJobDefinition = computed(() => {
  if (!taskForm.jobKey) {
    return null;
  }
  return jobDefinitions.value.find((job) => job.key === taskForm.jobKey) ?? null;
});

const selectedConfigSchema = computed(() => parseConfigSchema(selectedJobDefinition.value?.config_schema_json));

const configSchemaFields = computed(() => getConfigSchemaFields(selectedConfigSchema.value));

const selectedJobDefinitionWithActions = computed(() =>
  selectedJobDefinition.value ? (selectedJobDefinition.value as ScheduledTaskJobDefinitionItemWithActions) : null,
);

const jobDefinitionActions = computed(() => selectedJobDefinitionWithActions.value?.actions ?? []);

const persistentConfigSchemaFields = computed(() => configSchemaFields.value);

const drawerConfigSummaryItems = computed(() =>
  persistentConfigSchemaFields.value
    .map((field) => ({
      key: field.key,
      label: configSchemaFieldTitle(field),
      value: configValuePreview(taskForm.config[field.key]),
      description: configSchemaFieldDescription(field),
    }))
    .filter((item) => item.value !== t('scheduledTask.list.detail.none')),
);

const configDialogBehaviorSummary = computed(() =>
  selectedJobDefinition.value
    ? jobDefinitionDescription(selectedJobDefinition.value)
    : t('scheduledTask.list.configDialog.noJobDefinition'),
);

const configDialogTitle = computed(
  () =>
    taskForm.title.trim() ||
    (selectedJobDefinition.value
      ? jobDefinitionTitle(selectedJobDefinition.value)
      : t('scheduledTask.list.configDialog.title')),
);

const selectedTaskJobDefinition = computed(() =>
  selectedTask.value ? jobDefinitions.value.find((job) => job.key === selectedTask.value?.job_key) : null,
);

const selectedTaskConfigSchema = computed<ConfigSchema>(() =>
  parseConfigSchema(selectedTaskJobDefinition.value?.config_schema_json),
);

const selectedTaskConfigFields = computed(() => getConfigSchemaFields(selectedTaskConfigSchema.value));

const selectedTaskEffectiveConfig = computed<JsonRecord>(() => {
  if (!selectedTask.value) {
    return {};
  }
  if (selectedTask.value.effective_config?.trim()) {
    return parseJsonRecord(selectedTask.value.effective_config);
  }

  return mergeConfigRecords(
    parseJsonRecord(selectedTaskJobDefinition.value?.default_config_json),
    parseJsonRecord(selectedTask.value.config_json),
  );
});

const selectedTaskEffectiveConfigPreview = computed(() => JSON.stringify(selectedTaskEffectiveConfig.value, null, 2));

const selectedTaskJobDescription = computed(() =>
  selectedTaskJobDefinition.value
    ? jobDefinitionDescription(selectedTaskJobDefinition.value)
    : selectedTask.value
      ? taskDescription(selectedTask.value)
      : t('scheduledTask.list.detail.none'),
);

const actionResultStructured = computed<ScheduledTaskRunResult>(() => {
  if (!actionResult.value) {
    return {};
  }

  const parsed = actionResult.value.result_json ? parseRunResult(actionResult.value.result_json) : {};
  const result = actionResult.value.result ?? {};
  return {
    ...parsed,
    summary: result.summary ?? parsed.summary,
    stage: result.stage ?? parsed.stage,
    affected_resource: result.affected_resource ?? parsed.affected_resource,
    metrics: result.metrics ?? parsed.metrics,
    details: result.details ?? parsed.details,
    warnings: result.warnings ?? parsed.warnings,
  };
});

const actionResultRawJsonPreview = computed(() => {
  if (!actionResult.value) {
    return t('scheduledTask.list.detail.none');
  }

  return actionResult.value.result_json
    ? formatJsonPreview(actionResult.value.result_json) || actionResult.value.result_json
    : JSON.stringify(actionResult.value, null, 2);
});

const groupedJobDefinitions = computed<JobDefinitionGroup[]>(() => {
  const groups = new Map<string, ScheduledTaskJobDefinitionItem[]>();
  for (const job of jobDefinitions.value) {
    const moduleKey = job.module || job.owner || 'scheduler';
    groups.set(moduleKey, [...(groups.get(moduleKey) ?? []), job]);
  }

  return Array.from(groups.entries())
    .sort(([left], [right]) => moduleDisplayName(left).localeCompare(moduleDisplayName(right), locale.value))
    .map(([module, items]) => ({
      module,
      items: [...items].sort((left, right) =>
        jobDefinitionTitle(left).localeCompare(jobDefinitionTitle(right), locale.value),
      ),
    }));
});

const columnSettingOptions = computed(() => [
  { label: t('scheduledTask.list.columns.taskName'), value: 'task' },
  { label: t('scheduledTask.list.columns.jobType'), value: 'job_key' },
  { label: t('scheduledTask.list.columns.status'), value: 'status' },
  { label: t('scheduledTask.list.columns.cron'), value: 'schedule' },
  { label: t('scheduledTask.list.columns.recentResult'), value: 'recent_result' },
  { label: t('scheduledTask.list.columns.recentRun'), value: 'recent_run' },
  { label: t('scheduledTask.list.columns.successRate'), value: 'success_rate' },
]);

const allColumns = computed<TdBaseTableProps['columns']>(() => [
  {
    colKey: 'task',
    title: t('scheduledTask.list.columns.taskName'),
    width: 260,
    fixed: 'left',
  },
  {
    colKey: 'job_key',
    title: t('scheduledTask.list.columns.jobType'),
    width: 180,
  },
  {
    colKey: 'status',
    title: t('scheduledTask.list.columns.status'),
    width: 110,
  },
  {
    colKey: 'schedule',
    title: t('scheduledTask.list.columns.cron'),
    width: 230,
  },
  {
    colKey: 'recent_result',
    title: t('scheduledTask.list.columns.recentResult'),
    width: 220,
  },
  {
    colKey: 'recent_run',
    title: t('scheduledTask.list.columns.recentRun'),
    width: 210,
  },
  {
    colKey: 'success_rate',
    title: t('scheduledTask.list.columns.successRate'),
    width: 120,
  },
  {
    colKey: 'operation',
    title: t('scheduledTask.list.columns.operation'),
    align: 'right',
    width: 280,
    fixed: 'right',
  },
]);

const columns = computed<TdBaseTableProps['columns']>(() =>
  buildVisibleColumns(allColumns.value, visibleColumnKeys.value, ['operation']),
);
const tableContentWidth = computed(() => calculateTableContentWidth(columns.value));

const runColumns = computed<TdBaseTableProps['columns']>(() => [
  {
    colKey: 'started_at',
    title: t('scheduledTask.list.detail.time'),
    width: 170,
  },
  {
    colKey: 'trigger_type',
    title: t('scheduledTask.list.detail.triggerType'),
    width: 120,
  },
  {
    colKey: 'status',
    title: t('scheduledTask.list.detail.status'),
    width: 110,
  },
  {
    colKey: 'duration_ms',
    title: t('scheduledTask.list.detail.duration'),
    width: 110,
  },
  {
    colKey: 'result',
    title: t('scheduledTask.list.detail.result'),
    ellipsis: true,
  },
  {
    colKey: 'operation',
    title: t('scheduledTask.list.columns.operation'),
    width: 110,
  },
]);

onMounted(() => {
  void refreshTasks();
  void refreshJobDefinitions();
});

async function refreshJobDefinitions() {
  jobDefinitionsLoading.value = true;
  try {
    const response = await getScheduledTaskJobDefinitions();
    jobDefinitions.value = response.items;
  } catch (error) {
    logger.error(error instanceof Error ? error : 'load scheduled task job definitions failed', {
      operation: 'scheduled_task_jobs',
    });
    void MessagePlugin.error(t('scheduledTask.list.jobLoadError'));
  } finally {
    jobDefinitionsLoading.value = false;
  }
}

async function refreshTasks() {
  loading.value = true;
  errorMessage.value = '';

  try {
    const response = await getScheduledTasks({ limit: 100, offset: 0 });
    tasks.value = response.items;
    await refreshRunSummaries(response.items);
  } catch (error) {
    logger.error(error instanceof Error ? error : 'load scheduled tasks failed', {
      operation: 'scheduled_task_list',
    });
    errorMessage.value = t('scheduledTask.list.loadError');
  } finally {
    loading.value = false;
  }
}

async function refreshRunSummaries(items: ScheduledTaskItem[]) {
  // Summary runs are enrichment only; a per-task failure must not block the list.
  const entries = await Promise.all(
    items.map(async (task) => {
      try {
        const response = await getScheduledTaskRuns(task.key, { limit: 20, offset: 0 });
        return [task.key, response.items] as const;
      } catch (error) {
        logger.warn('load scheduled task summary runs failed', {
          error,
          taskKey: task.key,
          operation: 'scheduled_task_runs_summary',
        });
        return [task.key, []] as const;
      }
    }),
  );

  runHistoryByTaskKey.value = Object.fromEntries(entries);
}

async function openDetail(row: ScheduledTaskItem) {
  errorMessage.value = '';
  selectedTask.value = row;
  recentRuns.value = runHistoryByTaskKey.value[row.key] ?? [];
  detailVisible.value = true;

  try {
    const [detail] = await Promise.all([getScheduledTask(row.key), loadRuns(row.key)]);
    selectedTask.value = detail;
    syncTask(detail);
    await ensureJobDefinitionLoaded(detail.job_key);
  } catch (error) {
    logger.error(error instanceof Error ? error : 'load scheduled task detail failed', {
      taskKey: row.key,
      operation: 'scheduled_task_detail',
    });
    void MessagePlugin.error(t('scheduledTask.list.detailLoadError'));
  }
}

async function refreshRuns() {
  if (!selectedTask.value) {
    return;
  }

  try {
    await loadRuns(selectedTask.value.key);
  } catch (error) {
    logger.error(error instanceof Error ? error : 'load scheduled task runs failed', {
      taskKey: selectedTask.value.key,
      operation: 'scheduled_task_runs',
    });
    void MessagePlugin.error(t('scheduledTask.list.detailLoadError'));
  }
}

async function loadRuns(taskKey: string) {
  runsLoading.value = true;
  try {
    const response = await getScheduledTaskRuns(taskKey, { limit: 20, offset: 0 });
    recentRuns.value = response.items;
    runHistoryByTaskKey.value = {
      ...runHistoryByTaskKey.value,
      [taskKey]: response.items,
    };
    return response;
  } finally {
    runsLoading.value = false;
  }
}

function openCreateDrawer() {
  Object.assign(taskForm, createEmptyTaskForm());
  resetFormFieldErrors();
  formMode.value = 'create';
  editingTask.value = null;
  formVisible.value = true;
  if (jobDefinitions.value.length === 0) {
    void refreshJobDefinitions();
  }
}

async function openEditDrawer(row: ScheduledTaskItem) {
  editingTask.value = row;
  resetFormFieldErrors();
  formMode.value = 'edit';
  formVisible.value = true;
  try {
    const detail = await getScheduledTask(row.key);
    editingTask.value = detail;
    syncTask(detail);
    const job = await ensureJobDefinitionLoaded(detail.job_key);
    Object.assign(taskForm, taskToForm(detail, job ?? undefined));
  } catch (error) {
    logger.error(error instanceof Error ? error : 'load scheduled task edit detail failed', {
      taskKey: row.key,
      operation: 'scheduled_task_edit_detail',
    });
    Object.assign(taskForm, taskToForm(row));
    void MessagePlugin.error(t('scheduledTask.list.detailLoadError'));
  }
}

async function submitTaskForm() {
  const payload = buildTaskPayload();
  if (!payload) {
    return;
  }

  submittingTask.value = true;
  try {
    const saved =
      formMode.value === 'create'
        ? await createScheduledTask(payload as CreateScheduledTaskRequest)
        : await updateScheduledTask(editingTask.value?.key ?? taskForm.taskKey, payload as UpdateScheduledTaskRequest);

    syncTask(saved);
    formVisible.value = false;
    void MessagePlugin.success(
      formMode.value === 'create' ? t('scheduledTask.list.createSuccess') : t('scheduledTask.list.updateSuccess'),
    );
    await loadRuns(saved.key);
  } catch (error) {
    logger.error(error instanceof Error ? error : 'save scheduled task failed', {
      taskKey: taskForm.taskKey,
      operation: 'scheduled_task_save',
    });
    if (applyBackendFieldError(error)) {
      return;
    }
    void MessagePlugin.error(t('scheduledTask.list.saveError'));
  } finally {
    submittingTask.value = false;
  }
}

function buildTaskPayload(): CreateScheduledTaskRequest | UpdateScheduledTaskRequest | null {
  const cronExpression = normalizeCronForForm(taskForm.cronExpression);
  if (!cronExpression) {
    formFieldErrors.cronExpression = t('scheduledTask.list.form.cronRequiredHint');
    return null;
  }

  const cronResult = validateCronExpression(cronExpression);
  if (!cronResult.valid) {
    formFieldErrors.cronExpression =
      cronValidationMessageText(cronResult) || t('scheduledTask.list.form.cronInvalidHint');
    return null;
  }
  taskForm.cronExpression = cronExpression;
  formFieldErrors.cronExpression = '';

  const shouldPersistConfigJson = taskForm.configDirty;
  const persistentConfigJson = normalizeTaskLevelConfigJson();
  if (persistentConfigJson === null) {
    formFieldErrors.configJson = t('scheduledTask.list.form.configJsonInvalidHint');
    return null;
  }
  if (!validatePersistentConfigJson(persistentConfigJson)) {
    return null;
  }
  formFieldErrors.configJson = '';
  if (shouldPersistConfigJson) {
    syncTaskLevelConfigAfterSave(persistentConfigJson);
  }

  if (formMode.value === 'edit' && isSystemEdit.value) {
    // Builtin tasks keep their module-owned identity; users may tune schedule, enabled state, and schema-backed config.
    const payload = {
      cron_expression: cronExpression,
      enabled: taskForm.enabled,
    };
    return shouldPersistConfigJson ? withOptionalConfigJson(payload, persistentConfigJson) : payload;
  }

  if (!taskForm.title.trim()) {
    void MessagePlugin.warning(t('scheduledTask.list.form.titleRequiredHint'));
    return null;
  }

  if (!taskForm.jobKey.trim()) {
    void MessagePlugin.warning(t('scheduledTask.list.form.jobTypeRequiredHint'));
    return null;
  }

  if (formMode.value === 'create') {
    if (!taskForm.taskKey.trim()) {
      void MessagePlugin.warning(t('scheduledTask.list.form.taskKeyRequiredHint'));
      return null;
    }

    return withOptionalConfigJson(
      {
        task_key: taskForm.taskKey.trim(),
        job_key: taskForm.jobKey.trim(),
        title: taskForm.title.trim(),
        description: taskForm.description.trim() || undefined,
        cron_expression: cronExpression,
        enabled: taskForm.enabled,
      },
      persistentConfigJson,
    );
  }

  return withOptionalConfigJson(
    {
      title: taskForm.title.trim(),
      description: taskForm.description.trim() || undefined,
      cron_expression: cronExpression,
      enabled: taskForm.enabled,
    },
    persistentConfigJson,
  );
}

function openConfigDialog() {
  configDialogVisible.value = true;
  configDialogJsonMode.value = 'preview';
  customRetentionDaysSelected.value = !RETENTION_DAY_PRESETS.includes(
    configNumberValue('retentionDays') as (typeof RETENTION_DAY_PRESETS)[number],
  );
  if (jobDefinitions.value.length === 0) {
    void refreshJobDefinitions();
  }
}

async function confirmConfigDialog() {
  const persistentConfigJson = normalizePersistentConfigJson();
  if (persistentConfigJson === null) {
    formFieldErrors.configJson = t('scheduledTask.list.form.configJsonInvalidHint');
    configDialogJsonMode.value = 'edit';
    return;
  }
  if (!validatePersistentConfigJson(persistentConfigJson)) {
    configDialogJsonMode.value = 'edit';
    return;
  }

  taskForm.config = parseJsonRecord(persistentConfigJson);
  taskForm.configJson = persistentConfigJson ? JSON.stringify(taskForm.config, null, 2) : '';
  clearFormFieldError('configJson');

  if (formMode.value === 'create') {
    configDialogVisible.value = false;
    return;
  }

  if (!taskForm.configDirty) {
    configDialogVisible.value = false;
    return;
  }

  const taskKey = editingTask.value?.key ?? taskForm.taskKey;
  if (!taskKey) {
    return;
  }

  configDialogSaving.value = true;
  try {
    const saved = await updateScheduledTask(taskKey, {
      config_json: persistentConfigJson || undefined,
    } as UpdateScheduledTaskRequest);
    syncTask(saved);
    editingTask.value = saved;
    const savedForm = taskToForm(saved, selectedJobDefinition.value ?? undefined);
    taskForm.config = savedForm.config;
    taskForm.configJson = savedForm.configJson;
    taskForm.configDirty = savedForm.configDirty;
    taskForm.taskConfigJson = savedForm.taskConfigJson;
    configDialogVisible.value = false;
    void MessagePlugin.success(t('scheduledTask.list.configDialog.saveSuccess'));
  } catch (error) {
    logger.error(error instanceof Error ? error : 'save scheduled task config failed', {
      taskKey,
      operation: 'scheduled_task_config_save',
    });
    if (applyBackendFieldError(error)) {
      return;
    }
    void MessagePlugin.error(t('scheduledTask.list.configDialog.saveError'));
  } finally {
    configDialogSaving.value = false;
  }
}

function normalizePersistentConfigJson() {
  const configJson = normalizeJsonString(taskForm.configJson);
  if (configJson === null) {
    return null;
  }
  return buildPersistentConfigJson(configJson);
}

function normalizeTaskLevelConfigJson() {
  if (taskForm.configDirty) {
    return normalizePersistentConfigJson();
  }

  return taskForm.taskConfigJson;
}

function syncTaskLevelConfigAfterSave(persistentConfigJson: string) {
  taskForm.config = parseJsonRecord(persistentConfigJson);
  taskForm.configJson = persistentConfigJson ? JSON.stringify(taskForm.config, null, 2) : '';
  taskForm.taskConfigJson = persistentConfigJson;
  taskForm.configDirty = false;
}

function withOptionalConfigJson<T extends Record<string, unknown>>(
  payload: T,
  persistentConfigJson: string,
): T & {
  config_json?: string;
} {
  if (!persistentConfigJson) {
    return payload;
  }

  return {
    ...payload,
    config_json: persistentConfigJson,
  };
}

function buildPersistentConfigJson(configJson: string) {
  const config = sanitizeConfigBySelectedSchema(parseJsonRecord(configJson));
  return Object.keys(config).length > 0 ? JSON.stringify(config) : '';
}

function validatePersistentConfigJson(persistentConfigJson: string) {
  const job = selectedJobDefinition.value;
  if (!job || !persistentConfigJson) {
    clearFormFieldError('configJson');
    return true;
  }

  const schema = parseConfigSchema(job.config_schema_json);
  const result = validateConfigRecord(schema, parseJsonRecord(persistentConfigJson));
  if (result.valid) {
    clearFormFieldError('configJson');
    return true;
  }

  const message = configValidationIssueMessage(result.issues[0]);
  formFieldErrors.configJson = message;
  void MessagePlugin.warning(message);
  return false;
}

function configValidationIssueMessage(issue?: ConfigValidationIssue) {
  if (!issue) {
    return t('scheduledTask.list.form.configJsonInvalidHint');
  }

  const fieldLabel = configValidationIssueLabel(issue);
  switch (issue.reasonCode) {
    case 'required':
      return t('scheduledTask.list.validation.required', { field: fieldLabel });
    case 'additional_property':
      return t('scheduledTask.list.validation.additionalProperty', { field: fieldLabel });
    case 'type_mismatch':
      return t('scheduledTask.list.validation.typeMismatch', {
        field: fieldLabel,
        expected: configValidationExpectedLabel(issue.expected),
      });
    case 'enum':
      return t('scheduledTask.list.validation.enum', {
        field: fieldLabel,
        values: Array.isArray(issue.expected) ? issue.expected.join(', ') : '',
      });
    case 'below_minimum':
      return t('scheduledTask.list.validation.belowMinimum', { field: fieldLabel, minimum: issue.minimum });
    case 'above_maximum':
      return t('scheduledTask.list.validation.aboveMaximum', { field: fieldLabel, maximum: issue.maximum });
    case 'too_short':
      return t('scheduledTask.list.validation.tooShort', { field: fieldLabel, minimum: issue.minimum });
    case 'too_long':
      return t('scheduledTask.list.validation.tooLong', { field: fieldLabel, maximum: issue.maximum });
    default:
      return t('scheduledTask.list.form.configJsonInvalidHint');
  }
}

function configValidationIssueLabel(issue: ConfigValidationIssue) {
  if (issue.schema) {
    const field = { key: issue.key ?? issue.field, schema: issue.schema, required: false };
    return configSchemaFieldTitle(field);
  }
  return issue.key ?? issue.field;
}

function configValidationExpectedLabel(expected: unknown) {
  if (typeof expected === 'string') {
    return t(`scheduledTask.list.validation.types.${expected}`);
  }
  return String(expected ?? '');
}

function sanitizeConfigBySelectedSchema(config: JsonRecord): JsonRecord {
  if (!selectedJobDefinition.value) {
    return {};
  }

  return sanitizeConfigByJobDefinition(config, selectedJobDefinition.value);
}

function sanitizeConfigByJobDefinition(config: JsonRecord, job?: ScheduledTaskJobDefinitionItem): JsonRecord {
  if (!job) {
    return {};
  }

  const fields = getConfigSchemaFields(parseConfigSchema(job.config_schema_json));
  const allowedKeys = new Set(fields.map((field) => field.key));
  return Object.fromEntries(Object.entries(config).filter(([key]) => allowedKeys.has(key)));
}

function serializeConfigRecord(config: JsonRecord) {
  return Object.keys(config).length > 0 ? JSON.stringify(config) : '';
}

function openRunDialog(task: ScheduledTaskItem) {
  runDialogTask.value = task;
  runDialogVisible.value = true;
}

async function confirmRunTask() {
  if (!runDialogTask.value) {
    return;
  }
  await runTask(runDialogTask.value);
  runDialogVisible.value = false;
}

async function runTask(task: ScheduledTaskItem) {
  if (!canRunTask(task)) {
    return;
  }

  runningTaskKey.value = task.key;

  try {
    const run = await runScheduledTask(task.key);
    recentRuns.value = [run, ...recentRuns.value.filter((item) => item.id !== run.id)].slice(0, 20);
    runHistoryByTaskKey.value = {
      ...runHistoryByTaskKey.value,
      [task.key]: recentRuns.value,
    };
    const detail = await getScheduledTask(task.key);
    syncTask(detail);
    if (selectedTask.value?.key === detail.key) {
      selectedTask.value = detail;
    }
    requestNotificationHeaderRefresh();
    void MessagePlugin.success(t('scheduledTask.list.runSuccess'));
  } catch (error) {
    logger.error(error instanceof Error ? error : 'run scheduled task failed', {
      taskKey: task.key,
      operation: 'scheduled_task_run',
    });
    void MessagePlugin.error(t('scheduledTask.list.runError'));
  } finally {
    runningTaskKey.value = '';
  }
}

function handleActionClick(action: ScheduledTaskJobDefinitionAction) {
  selectedAction.value = action;
  if (action.confirm_required === false) {
    void executeSelectedAction(action);
    return;
  }
  actionConfirmDialogVisible.value = true;
}

async function confirmSelectedAction() {
  if (!selectedAction.value) {
    return;
  }
  await executeSelectedAction(selectedAction.value);
}

async function executeSelectedAction(action: ScheduledTaskJobDefinitionAction) {
  const taskKey = editingTask.value?.key ?? taskForm.taskKey;
  if (!taskKey) {
    return;
  }
  const payload = buildActionRequestPayload();
  if (payload === null) {
    return;
  }

  actionExecutingKey.value = action.key;
  try {
    actionResult.value = await executeScheduledTaskAction(taskKey, action.key, payload);
    actionConfirmDialogVisible.value = false;
    actionResultDialogVisible.value = true;
  } catch (error) {
    logger.error(error instanceof Error ? error : 'execute scheduled task action failed', {
      taskKey,
      actionKey: action.key,
      operation: 'scheduled_task_action_execute',
    });
    void MessagePlugin.error(t('scheduledTask.list.action.executeError'));
  } finally {
    actionExecutingKey.value = '';
  }
}

function buildActionRequestPayload(): ScheduledTaskActionRequest | undefined | null {
  const persistentConfigJson = normalizePersistentConfigJson();
  if (persistentConfigJson === null) {
    formFieldErrors.configJson = t('scheduledTask.list.form.configJsonInvalidHint');
    configDialogVisible.value = true;
    configDialogJsonMode.value = 'edit';
    return null;
  }
  if (!validatePersistentConfigJson(persistentConfigJson)) {
    configDialogVisible.value = true;
    configDialogJsonMode.value = 'edit';
    return null;
  }

  return persistentConfigJson ? { config_json: parseJsonRecord(persistentConfigJson) } : undefined;
}

function closeActionResultDialog() {
  actionResultDialogVisible.value = false;
  actionResult.value = null;
}

async function toggleTaskEnabled(task: ScheduledTaskItem) {
  lifecycleTaskKey.value = task.key;
  try {
    const updated = task.enabled ? await disableScheduledTask(task.key) : await enableScheduledTask(task.key);
    syncTask(updated);
    if (selectedTask.value?.key === updated.key) {
      selectedTask.value = updated;
    }
    void MessagePlugin.success(
      task.enabled ? t('scheduledTask.list.disableSuccess') : t('scheduledTask.list.enableSuccess'),
    );
  } catch (error) {
    logger.error(error instanceof Error ? error : 'toggle scheduled task enabled failed', {
      taskKey: task.key,
      operation: 'scheduled_task_lifecycle',
    });
    void MessagePlugin.error(t('scheduledTask.list.lifecycleError'));
  } finally {
    lifecycleTaskKey.value = '';
  }
}

function openDeleteDialog(task: ScheduledTaskItem) {
  deleteDialogTask.value = task;
  deleteDialogVisible.value = true;
}

async function confirmDeleteTask() {
  if (!deleteDialogTask.value) {
    return;
  }

  const task = deleteDialogTask.value;
  deletingTaskKey.value = task.key;
  try {
    await deleteScheduledTask(task.key);
    tasks.value = tasks.value.filter((item) => item.key !== task.key);
    const { [task.key]: _removed, ...remainingRuns } = runHistoryByTaskKey.value;
    runHistoryByTaskKey.value = remainingRuns;
    deleteDialogVisible.value = false;
    void MessagePlugin.success(t('scheduledTask.list.deleteSuccess'));
  } catch (error) {
    logger.error(error instanceof Error ? error : 'delete scheduled task failed', {
      taskKey: task.key,
      operation: 'scheduled_task_delete',
    });
    void MessagePlugin.error(t('scheduledTask.list.deleteError'));
  } finally {
    deletingTaskKey.value = '';
  }
}

async function openRunDetail(row: ScheduledTaskRunItem) {
  selectedRun.value = row;
  runDetailVisible.value = true;

  try {
    selectedRun.value = await getScheduledTaskRun(row.id);
  } catch (error) {
    logger.error(error instanceof Error ? error : 'load scheduled task run detail failed', {
      runId: row.id,
      operation: 'scheduled_task_run_detail',
    });
    void MessagePlugin.error(t('scheduledTask.list.detailLoadError'));
  }
}

function syncTask(detail: ScheduledTaskItem) {
  const index = tasks.value.findIndex((task) => task.key === detail.key);
  if (index === -1) {
    tasks.value = [detail, ...tasks.value];
    return;
  }

  tasks.value = tasks.value.map((task) => (task.key === detail.key ? detail : task));
}

function canRunTask(task: ScheduledTaskItem) {
  return task.enabled && !task.running && runningTaskKey.value !== task.key;
}

function isSystemTask(task: ScheduledTaskItem) {
  return task.builtin === true;
}

function createEmptyTaskForm(): TaskFormModel {
  return {
    taskKey: '',
    title: '',
    description: '',
    cronExpression: DEFAULT_CRON_EXPRESSION,
    enabled: true,
    jobKey: '',
    config: {},
    configJson: '',
    configDirty: false,
    taskConfigJson: '',
  };
}

function taskToForm(task: ScheduledTaskItem, job?: ScheduledTaskJobDefinitionItem): TaskFormModel {
  const expression = normalizeCronForForm(task.schedule || DEFAULT_CRON_EXPRESSION);
  const taskConfig = job
    ? sanitizeConfigByJobDefinition(parseJsonRecord(task.config_json), job)
    : parseJsonRecord(task.config_json);
  const defaultConfig = job
    ? mergeConfigRecords(
        buildDefaultConfigFromSchema(parseConfigSchema(job.config_schema_json)),
        parseJsonRecord(job.default_config_json),
      )
    : {};
  const effectiveConfig = task.effective_config?.trim()
    ? job
      ? sanitizeConfigByJobDefinition(parseJsonRecord(task.effective_config), job)
      : parseJsonRecord(task.effective_config)
    : mergeConfigRecords(defaultConfig, taskConfig);
  const taskConfigJson = serializeConfigRecord(taskConfig);
  return {
    taskKey: task.key,
    title: taskDisplayName(task),
    description: isSystemTask(task) ? taskDescription(task) : task.description || '',
    cronExpression: expression,
    enabled: task.enabled,
    jobKey: task.job_key,
    config: effectiveConfig,
    configJson: Object.keys(effectiveConfig).length > 0 ? JSON.stringify(effectiveConfig, null, 2) : '',
    configDirty: false,
    taskConfigJson,
  };
}

function normalizeJsonString(value: string): string | null {
  const trimmed = value.trim();
  if (!trimmed) {
    return '';
  }

  try {
    JSON.parse(trimmed);
    return trimmed;
  } catch {
    return null;
  }
}

function normalizeCronForForm(expression: string) {
  return normalizeCronExpression(expression || '');
}

function scheduleExpressionText(task: ScheduledTaskItem) {
  return formatCronExpression(task.schedule || DEFAULT_CRON_EXPRESSION);
}

function cronNextRunLine(expression: string) {
  const nextRun = getNextRunText(expression || DEFAULT_CRON_EXPRESSION, cronTimezone(), { locale: locale.value });
  return t('scheduledTask.cron.nextRun', {
    time: nextRun || t('scheduledTask.cron.nextRunUnavailable'),
  });
}

function cronScheduleDescription(expression: string) {
  return getCronDescription(expression || DEFAULT_CRON_EXPRESSION, locale.value, {
    advancedExpressionText: t('scheduledTask.cron.advancedExpression'),
    translate: (key, params) => t(key, params ?? {}),
  });
}

function cronTimezone() {
  return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
}

function cronValidationMessageText(result: CronValidationResult) {
  return translateCronValidation(result, t);
}

function handleCronEditorUpdate(value: string) {
  taskForm.cronExpression = normalizeCronForForm(value);
  if (validateCronExpression(taskForm.cronExpression).valid) {
    clearFormFieldError('cronExpression');
  }
}

function handleCronEditorValidate(result: CronValidationResult & { normalizedExpression: string }) {
  if (result.valid) {
    clearFormFieldError('cronExpression');
  }
}

function formatConfigJson() {
  const normalized = normalizeJsonString(taskForm.configJson);
  if (normalized === null) {
    formFieldErrors.configJson = t('scheduledTask.list.form.configJsonInvalidHint');
    configDialogJsonMode.value = 'edit';
    return;
  }

  taskForm.config = sanitizeConfigBySelectedSchema(parseJsonRecord(normalized));
  syncConfigJsonFromModel();
  markConfigDirty();
  clearFormFieldError('configJson');
}

function handleConfigJsonChange() {
  const normalized = normalizeJsonString(taskForm.configJson);
  if (normalized === null) {
    return;
  }
  taskForm.config = sanitizeConfigBySelectedSchema(parseJsonRecord(normalized));
  markConfigDirty();
  clearFormFieldError('configJson');
}

function updateConfigField(key: string, value: unknown) {
  taskForm.config = {
    ...taskForm.config,
    [key]: value,
  };
  syncConfigJsonFromModel();
  markConfigDirty();
  clearFormFieldError('configJson');
}

function syncConfigJsonFromModel() {
  taskForm.configJson = JSON.stringify(taskForm.config, null, 2);
}

function markConfigDirty() {
  taskForm.configDirty = true;
}

function configNumberValue(key: string) {
  const value = taskForm.config[key];
  return typeof value === 'number' ? value : undefined;
}

function configStringValue(key: string) {
  const value = taskForm.config[key];
  return typeof value === 'string' ? value : value === undefined || value === null ? '' : String(value);
}

function configSelectValue(key: string) {
  const value = taskForm.config[key];
  return typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean' ? value : undefined;
}

function clearFormFieldError(field: keyof FormFieldErrors) {
  formFieldErrors[field] = '';
}

function resetFormFieldErrors() {
  formFieldErrors.cronExpression = '';
  formFieldErrors.configJson = '';
}

function applyBackendFieldError(error: unknown) {
  if (!isApiRequestErrorShape(error)) {
    return false;
  }

  const field = readErrorField(error.responseData);
  const fieldMap: Record<string, keyof FormFieldErrors> = {
    cron_expression: 'cronExpression',
    config_json: 'configJson',
  };
  const formField = field ? fieldMap[field] : null;
  if (!formField) {
    return false;
  }

  formFieldErrors[formField] = error.message || t('scheduledTask.list.saveError');
  return true;
}

function isApiRequestErrorShape(error: unknown): error is ApiRequestError {
  return Boolean(error && typeof error === 'object' && (error as Partial<ApiRequestError>).isApiRequestError);
}

async function handleJobDefinitionChange(value: unknown) {
  if (typeof value !== 'string') {
    return;
  }

  const job = await ensureJobDefinitionLoaded(value);
  if (!job) {
    return;
  }

  taskForm.title = jobDefinitionTitle(job);
  taskForm.description = jobDefinitionDescription(job);
  taskForm.cronExpression = normalizeCronForForm(job.default_cron_expression || DEFAULT_CRON_EXPRESSION);
  taskForm.enabled = job.default_enabled;
  taskForm.config = mergeConfigRecords(
    buildDefaultConfigFromSchema(parseConfigSchema(job.config_schema_json)),
    parseJsonRecord(job.default_config_json),
  );
  syncConfigJsonFromModel();
  taskForm.configDirty = false;
  taskForm.taskConfigJson = '';
  resetFormFieldErrors();
}

async function ensureJobDefinitionLoaded(jobKey: string) {
  const cached = jobDefinitions.value.find((item) => item.key === jobKey);
  if (cached?.config_schema_json) {
    return cached;
  }

  try {
    const detail = await getScheduledTaskJobDefinition(jobKey);
    jobDefinitions.value = [detail, ...jobDefinitions.value.filter((item) => item.key !== jobKey)];
    return detail;
  } catch (error) {
    logger.warn('load scheduled task job definition detail failed', {
      error,
      jobKey,
      operation: 'scheduled_task_job_definition_detail',
    });
    return cached ?? null;
  }
}

function taskDisplayName(task: ScheduledTaskItem) {
  return localizedDisplayText(task.display_name_key, task.title, isSystemTask(task)) || task.key;
}

function taskDescription(task: ScheduledTaskItem) {
  return (
    localizedDisplayText(task.description_key, task.description, isSystemTask(task)) ||
    t('scheduledTask.list.detail.none')
  );
}

function localizeMessageKey(key?: string) {
  const trimmed = key?.trim();
  if (!trimmed) {
    return '';
  }

  if (isBuiltinTaskMessageKey(trimmed)) {
    return builtinTaskMessageText(trimmed);
  }

  return te(trimmed) ? t(trimmed) : '';
}

function isBuiltinTaskMessageKey(key: string): key is BuiltinTaskMessageKey {
  return (builtinTaskMessageKeys as readonly string[]).includes(key);
}

function builtinTaskMessageText(key: BuiltinTaskMessageKey) {
  const messages: Record<BuiltinTaskMessageKey, string> = {
    'scheduledTask.accessLogRetention.title': t('scheduledTask.accessLogRetention.title'),
    'scheduledTask.accessLogRetention.description': t('scheduledTask.accessLogRetention.description'),
    'scheduledTask.auditLogRetention.title': t('scheduledTask.auditLogRetention.title'),
    'scheduledTask.auditLogRetention.description': t('scheduledTask.auditLogRetention.description'),
    'scheduledTask.appLogRetention.title': t('scheduledTask.appLogRetention.title'),
    'scheduledTask.appLogRetention.description': t('scheduledTask.appLogRetention.description'),
    'scheduledTask.action.dryRun.title': t('scheduledTask.action.dryRun.title'),
    'scheduledTask.action.dryRun.description': t('scheduledTask.action.dryRun.description'),
    'scheduledTask.accessLogRetention.config.retentionDays.title': t(
      'scheduledTask.accessLogRetention.config.retentionDays.title',
    ),
    'scheduledTask.accessLogRetention.config.retentionDays.description': t(
      'scheduledTask.accessLogRetention.config.retentionDays.description',
    ),
    'scheduledTask.accessLogRetention.config.batchSize.title': t(
      'scheduledTask.accessLogRetention.config.batchSize.title',
    ),
    'scheduledTask.accessLogRetention.config.batchSize.description': t(
      'scheduledTask.accessLogRetention.config.batchSize.description',
    ),
    'scheduledTask.auditLogRetention.config.retentionDays.title': t(
      'scheduledTask.auditLogRetention.config.retentionDays.title',
    ),
    'scheduledTask.auditLogRetention.config.retentionDays.description': t(
      'scheduledTask.auditLogRetention.config.retentionDays.description',
    ),
    'scheduledTask.auditLogRetention.config.dryRun.title': t('scheduledTask.auditLogRetention.config.dryRun.title'),
    'scheduledTask.auditLogRetention.config.dryRun.description': t(
      'scheduledTask.auditLogRetention.config.dryRun.description',
    ),
    'scheduledTask.auditLogRetention.config.batchSize.title': t(
      'scheduledTask.auditLogRetention.config.batchSize.title',
    ),
    'scheduledTask.auditLogRetention.config.batchSize.description': t(
      'scheduledTask.auditLogRetention.config.batchSize.description',
    ),
    'scheduledTask.appLogRetention.config.retentionDays.title': t(
      'scheduledTask.appLogRetention.config.retentionDays.title',
    ),
    'scheduledTask.appLogRetention.config.retentionDays.description': t(
      'scheduledTask.appLogRetention.config.retentionDays.description',
    ),
    'scheduledTask.appLogRetention.config.dryRun.title': t('scheduledTask.appLogRetention.config.dryRun.title'),
    'scheduledTask.appLogRetention.config.dryRun.description': t(
      'scheduledTask.appLogRetention.config.dryRun.description',
    ),
    'scheduledTask.appLogRetention.config.batchSize.title': t('scheduledTask.appLogRetention.config.batchSize.title'),
    'scheduledTask.appLogRetention.config.batchSize.description': t(
      'scheduledTask.appLogRetention.config.batchSize.description',
    ),
  };
  return messages[key];
}

function localizeDisplayValue(value?: string | null) {
  const trimmed = value?.trim();
  if (!trimmed) {
    return '';
  }
  return localizeMessageKey(trimmed) || trimmed;
}

function localizedDisplayText(messageKey?: string, fallback?: string | null, preferMessageKey = false) {
  const localized = localizeMessageKey(messageKey);
  if (preferMessageKey && localized) {
    return localized;
  }

  // Custom tasks usually carry literal titles, while builtin jobs prefer translated message keys.
  return localizeDisplayValue(fallback) || localized;
}

function jobTypeLabel(jobKey: ScheduledTaskJobKey) {
  const job = jobDefinitions.value.find((item) => item.key === jobKey);
  return job ? jobDefinitionTitle(job) : jobKey;
}

function jobDefinitionTitle(job: ScheduledTaskJobDefinitionItem) {
  return localizedDisplayText(job.display_name_key, job.title, true) || job.key;
}

function jobDefinitionDescription(job: ScheduledTaskJobDefinitionItem) {
  return localizedDisplayText(job.description_key, job.description, true) || t('scheduledTask.list.detail.none');
}

function jobDefinitionActionTitle(action: ScheduledTaskJobDefinitionAction) {
  return localizedDisplayText(actionTranslationKey(action, 'title'), action.title, true) || action.key;
}

function jobDefinitionActionDescription(action: ScheduledTaskJobDefinitionAction) {
  return (
    localizedDisplayText(actionTranslationKey(action, 'description'), action.description, true) ||
    localizedDisplayText(
      action.behavior_key ?? action.behavior_summary_key,
      action.behavior ?? action.behavior_summary,
      true,
    ) ||
    jobDefinitionActionTitle(action)
  );
}

function actionTranslationKey(action: ScheduledTaskJobDefinitionAction, field: 'title' | 'description') {
  const compatibilityAction = action as ScheduledTaskJobDefinitionAction & {
    titleKey?: string;
    descriptionKey?: string;
  };
  return field === 'title'
    ? action.title_key || compatibilityAction.titleKey
    : action.description_key || compatibilityAction.descriptionKey;
}

function jobDefinitionActionAffectedResource(action: ScheduledTaskJobDefinitionAction, task: ScheduledTaskItem) {
  return (
    localizedDisplayText(action.affected_resource_key, action.affected_resource, true) || cleanupResourceLabel(task)
  );
}

function actionButtonTheme(action: ScheduledTaskJobDefinitionAction) {
  switch (action.theme) {
    case 'danger':
      return 'danger';
    case 'success':
      return 'success';
    case 'warning':
      return 'warning';
    case 'primary':
      return 'primary';
    case 'default':
    default:
      return 'default';
  }
}

function moduleDisplayName(moduleKey: string) {
  const messageKey = `module.${moduleKey}.title`;
  return te(messageKey) ? t(messageKey) : moduleKey;
}

function scheduleTypeLabel(type: ScheduledTaskItem['schedule_type']) {
  return t(`scheduledTask.list.scheduleType.${type}`);
}

function triggerLabel(type: ScheduledTaskRunTriggerType) {
  return t(`scheduledTask.list.trigger.${type}`);
}

function statusLabel(status: ScheduledTaskStatus | ScheduledTaskRunStatus, translate = t) {
  return translate(`scheduledTask.list.status.${status}`);
}

function booleanLabel(value: boolean) {
  return value ? t('scheduledTask.list.detail.enabledYes') : t('scheduledTask.list.detail.enabledNo');
}

function statusTheme(status: ScheduledTaskStatus | ScheduledTaskRunStatus) {
  switch (status) {
    case 'success':
      return 'success';
    case 'running':
      return 'primary';
    case 'failed':
      return 'danger';
    case 'idle':
    case 'unknown':
    default:
      return 'default';
  }
}

function successRateLabel(taskKey: string) {
  const runs = runHistoryByTaskKey.value[taskKey] ?? [];
  const finishedRuns = runs.filter((run) => run.status === 'success' || run.status === 'failed');
  if (finishedRuns.length === 0) {
    return t('scheduledTask.list.detail.notAvailable');
  }

  const success = finishedRuns.filter((run) => run.status === 'success').length;
  return `${Math.round((success / finishedRuns.length) * 100)}%`;
}

function runResultText(run: ScheduledTaskRunItem | NonNullable<ScheduledTaskItem['last_run']>) {
  const structured = runResultStructured(run);
  if (structured.summary) {
    return structured.summary;
  }

  if (run.status === 'success') {
    return run.result_summary || t('scheduledTask.list.detail.none');
  }

  if (run.status === 'failed') {
    return run.error_summary || run.result_summary || t('scheduledTask.list.detail.noError');
  }

  return run.result_summary || run.error_summary || t('scheduledTask.list.detail.noError');
}

function runResultStructured(run: ScheduledTaskRunItem | NonNullable<ScheduledTaskItem['last_run']>) {
  return parseRunResult(run.result_json);
}

function configSchemaFieldTitle(field: ConfigSchemaField) {
  return localizeMessageKey(field.schema.xI18n?.titleKey) || field.schema.title || field.key;
}

function configSchemaFieldDescription(field: ConfigSchemaField) {
  return localizeMessageKey(field.schema.xI18n?.descriptionKey) || field.schema.description || '';
}

function configValuePreview(value: unknown) {
  if (typeof value === 'boolean') {
    return booleanLabel(value);
  }
  if (value === undefined || value === null || value === '') {
    return t('scheduledTask.list.detail.none');
  }
  if (typeof value === 'object') {
    return JSON.stringify(value);
  }
  return String(value);
}

function isRetentionDaysField(field: ConfigSchemaField) {
  return field.key === 'retentionDays' && (field.schema.type === 'integer' || field.schema.type === 'number');
}

function retentionDaysOptionValue() {
  const value = configNumberValue('retentionDays');
  if (
    !customRetentionDaysSelected.value &&
    value &&
    RETENTION_DAY_PRESETS.includes(value as (typeof RETENTION_DAY_PRESETS)[number])
  ) {
    return value;
  }
  return CUSTOM_RETENTION_DAY_VALUE;
}

function handleRetentionDaysOptionChange(value: unknown) {
  if (value === CUSTOM_RETENTION_DAY_VALUE) {
    customRetentionDaysSelected.value = true;
    const currentValue = configNumberValue('retentionDays');
    if (!currentValue) {
      updateConfigField('retentionDays', 30);
    }
    return;
  }

  if (typeof value === 'number') {
    customRetentionDaysSelected.value = false;
    updateConfigField('retentionDays', value);
    return;
  }

  const numericValue = Number(value);
  if (Number.isFinite(numericValue)) {
    customRetentionDaysSelected.value = false;
    updateConfigField('retentionDays', numericValue);
  }
}

function immediateRunSummary(task: ScheduledTaskItem): ImmediateRunSummary {
  const job = jobDefinitions.value.find((item) => item.key === task.job_key);
  const effectiveConfig = task.effective_config?.trim()
    ? parseJsonRecord(task.effective_config)
    : mergeConfigRecords(parseJsonRecord(job?.default_config_json), parseJsonRecord(task.config_json));
  const items = [
    {
      key: 'resource',
      label: t('scheduledTask.list.runDialog.affectedResource'),
      value: cleanupResourceLabel(task, job),
    },
    {
      key: 'retentionDays',
      label: t('scheduledTask.list.runDialog.retentionDays'),
      value: configValuePreview(effectiveConfig.retentionDays ?? effectiveConfig.retention_days),
    },
    {
      key: 'cutoff',
      label: t('scheduledTask.list.runDialog.cutoffPolicy'),
      value: configValuePreview(effectiveConfig.cutoffTime ?? effectiveConfig.cutoff_policy),
    },
    {
      key: 'batchSize',
      label: t('scheduledTask.list.runDialog.batchSize'),
      value: configValuePreview(effectiveConfig.batchSize),
    },
  ].filter((item) => item.value !== t('scheduledTask.list.detail.none'));

  return {
    description:
      job && isCleanupJob(task.job_key)
        ? t('scheduledTask.list.runDialog.cleanupDescription', {
            behavior: jobDefinitionDescription(job),
          })
        : t('scheduledTask.list.runDialog.description'),
    items:
      items.length > 0
        ? items
        : [
            {
              key: 'behavior',
              label: t('scheduledTask.list.runDialog.expectedBehavior'),
              value: job ? jobDefinitionDescription(job) : taskDescription(task),
            },
          ],
  };
}

function isCleanupJob(jobKey: string) {
  return jobKey.includes('retention-cleanup');
}

function cleanupResourceLabel(task: ScheduledTaskItem, job?: ScheduledTaskJobDefinitionItem) {
  const key = job?.key ?? task.job_key;
  if (key.includes('access-log')) {
    return t('scheduledTask.list.resource.accessLog');
  }
  if (key.includes('audit-log')) {
    return t('scheduledTask.list.resource.auditLog');
  }
  if (key.includes('app-log')) {
    return t('scheduledTask.list.resource.appLog');
  }
  return moduleDisplayName(job?.module ?? task.module);
}

function formatTimestamp(value?: string | null) {
  if (!value) {
    return t('scheduledTask.list.detail.notAvailable');
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  }).format(date);
}

function formatDuration(value?: number | null) {
  if (value === undefined || value === null) {
    return t('scheduledTask.list.detail.notAvailable');
  }

  if (value < 1000) {
    return `${value} ms`;
  }

  const seconds = value / 1000;
  if (seconds < 60) {
    return `${seconds.toFixed(seconds >= 10 ? 0 : 1)} s`;
  }

  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = Math.round(seconds % 60);
  return `${minutes} min ${remainingSeconds} s`;
}
</script>
<style scoped lang="less">
.scheduled-task-page {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.scheduled-task-page__header,
.scheduled-task-table-head,
.scheduled-task-detail__section-head {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.scheduled-task-page__title-block {
  min-width: 0;
}

.scheduled-task-page__eyebrow {
  color: var(--td-brand-color);
  display: inline-block;
  font: var(--td-font-body-small);
  font-weight: 600;
  margin-bottom: var(--graft-density-gap-4);
}

.scheduled-task-page h1,
.scheduled-task-table-head h2,
.scheduled-task-detail h3 {
  color: var(--td-text-color-primary);
  margin: 0;
}

.scheduled-task-page h1 {
  font: var(--td-font-headline-small);
}

.scheduled-task-table-head h2,
.scheduled-task-detail h3 {
  font: var(--td-font-title-medium);
}

.scheduled-task-page__title-block p,
.scheduled-task-table-head p,
.scheduled-task-metric-card p,
.scheduled-task-metric-card span,
.scheduled-task-muted,
.scheduled-task-schedule__next-run,
.scheduled-task-identity__key,
.scheduled-task-last-run span,
.scheduled-task-form-hint {
  color: var(--td-text-color-secondary);
}

.scheduled-task-page__title-block p,
.scheduled-task-table-head p,
.scheduled-task-metric-card p,
.scheduled-task-form-hint {
  margin: var(--graft-density-gap-4) 0 0;
}

.scheduled-task-metrics {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.scheduled-task-metric-card {
  min-width: 0;
}

.scheduled-task-metric-card :deep(.t-card__body) {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
}

.scheduled-task-metric-card strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-headline-small);
}

.scheduled-task-table-card :deep(.t-card__body) {
  padding-top: 0;
}

.scheduled-task-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  margin-bottom: var(--graft-density-gap-12);
}

.scheduled-task-toolbar__search {
  max-width: 360px;
  min-width: 240px;
}

.scheduled-task-toolbar__select {
  width: 180px;
}

.scheduled-task-feedback {
  align-items: center;
  background: color-mix(in srgb, var(--td-error-color-5) 10%, var(--td-bg-color-container));
  border: 1px solid color-mix(in srgb, var(--td-error-color-5) 28%, var(--td-component-stroke));
  border-radius: var(--td-radius-medium);
  color: var(--td-error-color-7);
  display: flex;
  justify-content: space-between;
  margin-bottom: var(--graft-density-gap-12);
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

.scheduled-task-identity,
.scheduled-task-schedule,
.scheduled-task-last-run {
  display: flex;
  min-width: 0;
}

.scheduled-task-identity,
.scheduled-task-schedule {
  flex-direction: column;
}

.scheduled-task-last-run {
  align-items: center;
  gap: var(--graft-density-gap-8);
}

.scheduled-task-identity__name {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.scheduled-task-identity__key,
.scheduled-task-schedule span,
.scheduled-task-last-run span,
.scheduled-task-mono {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scheduled-task-mono {
  color: var(--td-text-color-primary);
  display: inline-block;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  max-width: 100%;
  user-select: text;
}

.scheduled-task-schedule {
  border-radius: var(--td-radius-small);
  cursor: default;
  gap: var(--graft-density-gap-4);
  max-width: 100%;
  padding: var(--graft-density-gap-2) 0;
}

.scheduled-task-schedule:hover .scheduled-task-mono {
  color: var(--td-brand-color);
}

.scheduled-task-schedule__next-run {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-small);
  line-height: var(--td-line-height-body-small);
}

.scheduled-task-cron-tooltip__content {
  display: grid;
  gap: var(--graft-density-gap-10);
  min-width: 220px;
}

.scheduled-task-cron-tooltip__item {
  display: grid;
  gap: var(--graft-density-gap-4);
}

.scheduled-task-cron-tooltip__item span {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-small);
}

.scheduled-task-cron-tooltip__item code,
.scheduled-task-cron-tooltip__item strong {
  color: var(--td-text-color-primary);
  font-size: var(--td-font-size-body-medium);
  font-weight: 500;
  line-height: var(--td-line-height-body-medium);
  overflow-wrap: anywhere;
}

.scheduled-task-cron-tooltip__item code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  font-weight: 600;
}

.scheduled-task-actions {
  display: inline-flex;
  flex-wrap: nowrap;
  gap: var(--graft-density-gap-4);
  justify-content: flex-end;
  white-space: nowrap;
  width: 100%;
}

.scheduled-task-actions :deep(.t-button),
.scheduled-task-actions :deep(.t-dropdown) {
  flex: 0 0 auto;
}

.scheduled-task-empty,
.scheduled-task-runs-empty {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  justify-content: center;
  min-height: 220px;
  padding: var(--graft-density-gap-24);
}

.scheduled-task-runs-empty {
  min-height: 120px;
}

.scheduled-task-detail {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-20);
}

.scheduled-task-detail__section {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.scheduled-task-form-section {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  margin-bottom: var(--graft-density-gap-20);
}

.scheduled-task-form-section__head h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0;
}

.scheduled-task-cron-form-item :deep(.t-form__controls),
.scheduled-task-cron-form-item :deep(.t-form__controls-content) {
  min-width: 0;
  width: 100%;
}

.scheduled-task-job-option {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
  width: 100%;
}

.scheduled-task-job-option__main {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.scheduled-task-job-option__main strong {
  color: var(--td-text-color-primary);
  font-weight: 600;
}

.scheduled-task-job-option__main span {
  color: var(--td-text-color-secondary);
  overflow-wrap: anywhere;
}

.scheduled-task-job-summary {
  background: var(--td-bg-color-container);
}

.scheduled-task-job-summary :deep(.t-card__body) {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.scheduled-task-job-summary p,
.scheduled-task-form-section p {
  margin: 0;
}

.scheduled-task-json-preview {
  background: var(--td-bg-color-page);
  border-radius: var(--td-radius-small);
  box-sizing: border-box;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  margin: 0;
  max-width: 100%;
  overflow: auto;
  overflow-wrap: anywhere;
  padding: var(--graft-density-gap-8);
  white-space: pre-wrap;
}

.scheduled-task-config-list,
.scheduled-task-config-section,
.scheduled-task-advanced-config,
.scheduled-task-raw-config {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
}

.scheduled-task-config-section__head {
  display: grid;
  gap: var(--graft-density-gap-4);
}

.scheduled-task-config-section__head h3 {
  color: var(--td-text-color-primary);
  font-size: var(--td-font-size-title-small);
  margin: 0;
}

.scheduled-task-config-section__head p {
  color: var(--td-text-color-secondary);
}

.scheduled-task-retention-field {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
}

.scheduled-task-retention-field :deep(.t-radio-group) {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.scheduled-task-advanced-config section {
  display: grid;
  gap: var(--graft-density-gap-6);
}

.scheduled-task-advanced-config strong {
  color: var(--td-text-color-primary);
}

.scheduled-task-config-list__item {
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-small);
  display: grid;
  gap: var(--graft-density-gap-4);
  padding: var(--graft-density-gap-8);
}

.scheduled-task-config-list__item strong,
.scheduled-task-raw-config strong {
  color: var(--td-text-color-primary);
}

.scheduled-task-config-list__item span {
  color: var(--td-text-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  overflow-wrap: anywhere;
}

.scheduled-task-config-list__item small {
  color: var(--td-text-color-secondary);
}

.scheduled-task-warning-list {
  margin: 0;
  padding-left: var(--graft-density-gap-20);
}

.scheduled-task-drawer-footer {
  justify-content: flex-end;
  width: 100%;
}

.scheduled-task-dialog-copy {
  color: var(--td-text-color-primary);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
}

.scheduled-task-dialog-copy p {
  margin: 0;
}

:deep(.scheduled-task-status-tag) {
  border-radius: 999px;
  font-weight: 600;
}

@media (width <= 900px) {
  .scheduled-task-page__header,
  .scheduled-task-table-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .scheduled-task-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .scheduled-task-toolbar__search,
  .scheduled-task-toolbar__select {
    max-width: none;
    width: 100%;
  }
}

@media (width <= 520px) {
  .scheduled-task-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
