<template>
  <advanced-query-list-page
    root-class="scheduled-task-page"
    page-type="list-form-detail"
    :title="t('scheduledTask.list.title')"
    :description="t('scheduledTask.list.description')"
    :error-message="errorMessage"
    :error-title="t('scheduledTask.list.loadError')"
    :loading="loading"
    :reload-label="t('scheduledTask.list.refresh')"
    :retry-label="t('scheduledTask.list.refresh')"
    @reload="refreshTasks"
  >
    <template #eyebrow>{{ t('scheduledTask.list.eyebrow') }}</template>
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
      <section class="scheduled-task-metrics" aria-label="scheduled task metrics">
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
            <div class="scheduled-task-schedule">
              <span class="scheduled-task-mono">{{ scheduleExpressionText(row) }}</span>
              <span v-if="cronScheduleDescriptionText(row.schedule)">{{
                cronScheduleDescriptionText(row.schedule)
              }}</span>
            </div>
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
                <t-descriptions-item :label="t('scheduledTask.list.form.defaultParams')" :span="2">
                  <pre class="scheduled-task-json-preview">{{
                    formatJsonPreview(selectedJobDefinition.default_params_json) || t('scheduledTask.list.detail.none')
                  }}</pre>
                </t-descriptions-item>
              </t-descriptions>
              <t-collapse v-if="selectedJobDefinition.params_schema_json" expand-icon-placement="right">
                <t-collapse-panel value="paramsSchema" :header="t('scheduledTask.list.form.paramsSchema')">
                  <pre class="scheduled-task-json-preview">{{
                    formatJsonPreview(selectedJobDefinition.params_schema_json)
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

          <section v-if="!isSystemEdit" class="scheduled-task-form-section">
            <t-collapse expand-icon-placement="right">
              <t-collapse-panel value="advancedParams" :header="t('scheduledTask.list.form.sectionAdvancedParams')">
                <t-form-item
                  :label="t('scheduledTask.list.form.paramsJson')"
                  name="paramsJson"
                  :status="formFieldErrors.paramsJson ? 'error' : undefined"
                  :tips="formFieldErrors.paramsJson"
                >
                  <div class="scheduled-task-params-field">
                    <t-button size="small" variant="outline" @click="formatParamsJson">
                      {{ t('scheduledTask.list.form.formatJson') }}
                    </t-button>
                    <t-textarea
                      v-model="taskForm.paramsJson"
                      :autosize="{ minRows: 4, maxRows: 8 }"
                      :placeholder="t('scheduledTask.list.form.paramsJsonPlaceholder')"
                      @change="clearFormFieldError('paramsJson')"
                    />
                  </div>
                </t-form-item>
              </t-collapse-panel>
            </t-collapse>
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
              <t-descriptions-item v-if="selectedTask.params_json" :label="t('scheduledTask.list.detail.paramsJson')">
                <pre class="scheduled-task-json-preview">{{ formatJsonPreview(selectedTask.params_json) }}</pre>
              </t-descriptions-item>
            </t-descriptions>
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
          <p>{{ t('scheduledTask.list.runDialog.description') }}</p>
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
  getScheduledTask,
  getScheduledTaskJobs,
  getScheduledTaskRun,
  getScheduledTaskRuns,
  getScheduledTasks,
  runScheduledTask,
  updateScheduledTask,
} from '../../api/scheduled-task';
import CronExpressionField from '../../components/CronExpressionField.vue';
import { SCHEDULED_TASK_PERMISSION_CODE } from '../../contract/permissions';
import type {
  CreateScheduledTaskRequest,
  ScheduledTaskItem,
  ScheduledTaskJobDefinitionItem,
  ScheduledTaskJobKey,
  ScheduledTaskRunItem,
  ScheduledTaskRunStatus,
  ScheduledTaskRunTriggerType,
  ScheduledTaskStatus,
  UpdateScheduledTaskRequest,
} from '../../types/scheduled-task';
import {
  type CronDescriptionResult,
  type CronValidationResult,
  describeCronExpression,
  normalizeCronExpression,
  validateCronExpression,
} from '../../utils/cron';
import { translateCronDescription, translateCronValidation } from '../../utils/cron-i18n';

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
  paramsJson: string;
};

type FormFieldErrors = {
  cronExpression: string;
  paramsJson: string;
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

const DEFAULT_CRON_EXPRESSION = '0 */5 * * * *';

const statusOptions: ScheduledTaskStatus[] = ['idle', 'running', 'success', 'failed', 'unknown'];
const builtinTaskMessageKeys = [
  'scheduledTask.accessLogRetention.title',
  'scheduledTask.accessLogRetention.description',
  'scheduledTask.auditLogRetention.title',
  'scheduledTask.auditLogRetention.description',
  'scheduledTask.appLogRetention.title',
  'scheduledTask.appLogRetention.description',
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
const runDialogVisible = ref(false);
const deleteDialogVisible = ref(false);
const runDetailVisible = ref(false);
const columnDrawerVisible = ref(false);
const errorMessage = ref('');
const runningTaskKey = ref('');
const lifecycleTaskKey = ref('');
const deletingTaskKey = ref('');
const submittingTask = ref(false);
const formMode = ref<FormMode>('create');
const editingTask = ref<ScheduledTaskItem | null>(null);
const runDialogTask = ref<ScheduledTaskItem | null>(null);
const deleteDialogTask = ref<ScheduledTaskItem | null>(null);

const filters = reactive<FilterModel>({
  keyword: '',
  jobKey: 'all',
  status: 'all',
});

const taskForm = reactive<TaskFormModel>(createEmptyTaskForm());
const formFieldErrors = reactive<FormFieldErrors>({
  cronExpression: '',
  paramsJson: '',
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
    width: 220,
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
    const response = await getScheduledTaskJobs();
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
    Object.assign(taskForm, taskToForm(detail));
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

  if (formMode.value === 'edit' && isSystemEdit.value) {
    // Builtin tasks keep their module-owned identity; users may only tune schedule and enabled state.
    return {
      cron_expression: cronExpression,
      enabled: taskForm.enabled,
    };
  }

  if (!taskForm.title.trim()) {
    void MessagePlugin.warning(t('scheduledTask.list.form.titleRequiredHint'));
    return null;
  }

  if (!taskForm.jobKey.trim()) {
    void MessagePlugin.warning(t('scheduledTask.list.form.jobTypeRequiredHint'));
    return null;
  }

  const paramsJson = normalizeJsonString(taskForm.paramsJson);
  if (paramsJson === null) {
    formFieldErrors.paramsJson = t('scheduledTask.list.form.paramsJsonInvalidHint');
    return null;
  }
  formFieldErrors.paramsJson = '';

  if (formMode.value === 'create') {
    if (!taskForm.taskKey.trim()) {
      void MessagePlugin.warning(t('scheduledTask.list.form.taskKeyRequiredHint'));
      return null;
    }

    return {
      task_key: taskForm.taskKey.trim(),
      job_key: taskForm.jobKey.trim(),
      title: taskForm.title.trim(),
      description: taskForm.description.trim() || undefined,
      cron_expression: cronExpression,
      enabled: taskForm.enabled,
      params_json: paramsJson || undefined,
    };
  }

  return {
    title: taskForm.title.trim(),
    description: taskForm.description.trim() || undefined,
    cron_expression: cronExpression,
    enabled: taskForm.enabled,
    params_json: paramsJson || undefined,
  };
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
    paramsJson: '',
  };
}

function taskToForm(task: ScheduledTaskItem): TaskFormModel {
  const expression = normalizeCronForForm(task.schedule || DEFAULT_CRON_EXPRESSION);
  return {
    taskKey: task.key,
    title: taskDisplayName(task),
    description: isSystemTask(task) ? taskDescription(task) : task.description || '',
    cronExpression: expression,
    enabled: task.enabled,
    jobKey: task.job_key,
    paramsJson: formatJsonPreview(task.params_json || ''),
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
  return normalizeCronForForm(task.schedule || DEFAULT_CRON_EXPRESSION);
}

function cronScheduleDescriptionText(expression: string) {
  const description = describeCronExpression(expression) as CronDescriptionResult;
  if (!description.valid || description.key === 'scheduledTask.cronDescription.custom') {
    return '';
  }

  return translateCronDescription(description, t);
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

function formatParamsJson() {
  const normalized = normalizeJsonString(taskForm.paramsJson);
  if (normalized === null) {
    formFieldErrors.paramsJson = t('scheduledTask.list.form.paramsJsonInvalidHint');
    return;
  }

  taskForm.paramsJson = formatJsonPreview(normalized);
  clearFormFieldError('paramsJson');
}

function clearFormFieldError(field: keyof FormFieldErrors) {
  formFieldErrors[field] = '';
}

function resetFormFieldErrors() {
  formFieldErrors.cronExpression = '';
  formFieldErrors.paramsJson = '';
}

function applyBackendFieldError(error: unknown) {
  if (!isApiRequestErrorShape(error)) {
    return false;
  }

  const field = readErrorField(error.responseData);
  const fieldMap: Record<string, keyof FormFieldErrors> = {
    cron_expression: 'cronExpression',
    params_json: 'paramsJson',
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

function handleJobDefinitionChange(value: unknown) {
  if (typeof value !== 'string') {
    return;
  }

  const job = jobDefinitions.value.find((item) => item.key === value);
  if (!job) {
    return;
  }

  taskForm.title = jobDefinitionTitle(job);
  taskForm.description = jobDefinitionDescription(job);
  taskForm.cronExpression = normalizeCronForForm(job.default_cron_expression || DEFAULT_CRON_EXPRESSION);
  taskForm.enabled = job.default_enabled;
  taskForm.paramsJson = formatJsonPreview(job.default_params_json);
  resetFormFieldErrors();
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

function moduleDisplayName(moduleKey: string) {
  const messageKey = `module.${moduleKey}.title`;
  return te(messageKey) ? t(messageKey) : moduleKey;
}

function formatJsonPreview(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return '';
  }

  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2);
  } catch {
    return trimmed;
  }
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
  if (run.status === 'success') {
    return run.result_summary || t('scheduledTask.list.detail.noError');
  }

  if (run.status === 'failed') {
    return run.error_summary || run.result_summary || t('scheduledTask.list.detail.none');
  }

  return run.result_summary || run.error_summary || t('scheduledTask.list.detail.none');
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
.scheduled-task-schedule span:last-child,
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

.scheduled-task-params-field {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
}

.scheduled-task-params-field > .t-button {
  align-self: flex-start;
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
