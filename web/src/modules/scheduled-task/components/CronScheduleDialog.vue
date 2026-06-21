<template>
  <t-dialog
    :visible="visible"
    :header="t('scheduledTask.cronScheduleDialog.title')"
    width="820px"
    placement="center"
    attach="body"
    destroy-on-close
    :confirm-btn="confirmButton"
    :cancel-btn="t('scheduledTask.cronScheduleDialog.cancel')"
    @update:visible="handleVisibleUpdate"
    @confirm="confirmDraft"
    @cancel="cancelDraft"
    @close="cancelDraft"
  >
    <div class="cron-schedule-dialog">
      <aside class="cron-schedule-dialog__nav" :aria-label="t('scheduledTask.cronScheduleDialog.planTypeLabel')">
        <button
          v-for="option in planTypeOptions"
          :key="option.value"
          class="cron-schedule-dialog__nav-item"
          :class="{ 'cron-schedule-dialog__nav-item--active': activeMode === option.value }"
          type="button"
          @click="selectMode(option.value)"
        >
          {{ option.label }}
        </button>
      </aside>

      <section class="cron-schedule-dialog__content">
        <div class="cron-schedule-dialog__form">
          <t-form label-align="top" :data="formState">
            <template v-if="activeMode === 'intervalMinutes'">
              <t-form-item :label="t('scheduledTask.cronScheduleDialog.fields.intervalMinutes')">
                <t-input-number
                  :value="formState.intervalMinutes"
                  theme="normal"
                  :min="1"
                  :max="59"
                  :disabled="disabled"
                  @update:value="updateNumberField('intervalMinutes', $event)"
                  @change="updateNumberField('intervalMinutes', $event)"
                />
              </t-form-item>
              <div class="cron-schedule-dialog__quick-group">
                <span>{{ t('scheduledTask.cronScheduleDialog.quickSelect') }}</span>
                <div>
                  <t-button
                    v-for="minute in minuteQuickOptions"
                    :key="minute"
                    size="small"
                    variant="outline"
                    :disabled="disabled"
                    @click="setIntervalMinutes(minute)"
                  >
                    {{ t('scheduledTask.cronScheduleDialog.quickMinutes', { count: minute }) }}
                  </t-button>
                </div>
              </div>
            </template>

            <template v-if="activeMode === 'hourly'">
              <t-form-item :label="t('scheduledTask.cronScheduleDialog.fields.executionMinute')">
                <div class="cron-schedule-dialog__inline-field">
                  <span>{{ t('scheduledTask.cronScheduleDialog.hourlyPrefix') }}</span>
                  <t-input-number
                    :value="formState.minute"
                    theme="normal"
                    :min="0"
                    :max="59"
                    :disabled="disabled"
                    @update:value="updateNumberField('minute', $event)"
                    @change="updateNumberField('minute', $event)"
                  />
                  <span>{{ t('scheduledTask.cronScheduleDialog.hourlySuffix') }}</span>
                </div>
              </t-form-item>
            </template>

            <template v-if="activeMode === 'weekly'">
              <t-form-item :label="t('scheduledTask.cronScheduleDialog.fields.weekday')">
                <t-radio-group
                  v-model:value="formState.weekday"
                  theme="button"
                  variant="outline"
                  :options="weekdayOptions"
                  :disabled="disabled"
                  @change="syncExpressionFromMode"
                />
              </t-form-item>
            </template>

            <template v-if="activeMode === 'monthly'">
              <t-form-item :label="t('scheduledTask.cronScheduleDialog.fields.dayOfMonth')">
                <t-input-number
                  :value="formState.dayOfMonth"
                  theme="normal"
                  :min="1"
                  :max="31"
                  :disabled="disabled"
                  @update:value="updateNumberField('dayOfMonth', $event)"
                  @change="updateNumberField('dayOfMonth', $event)"
                />
                <p v-if="formState.dayOfMonth >= 29" class="cron-schedule-dialog__field-hint">
                  {{ t('scheduledTask.cronScheduleDialog.monthDayWarning') }}
                </p>
              </t-form-item>
            </template>

            <template v-if="showExecutionTimeFields">
              <t-form-item :label="t('scheduledTask.cronScheduleDialog.fields.executionTime')">
                <div class="cron-schedule-dialog__time-fields">
                  <template v-for="field in executionTimeFields" :key="field.key">
                    <t-input-number
                      class="cron-schedule-dialog__time-input"
                      :data-testid="`cron-time-${field.key}-input`"
                      :value="formState[field.key]"
                      theme="normal"
                      :min="field.min"
                      :max="field.max"
                      :disabled="disabled"
                      @update:value="updateNumberField(field.key, $event)"
                      @change="updateNumberField(field.key, $event)"
                    />
                    <span
                      v-if="field.separator"
                      class="cron-schedule-dialog__time-separator"
                      data-testid="cron-time-separator"
                    >
                      {{ field.separator }}
                    </span>
                  </template>
                </div>
              </t-form-item>
            </template>

            <template v-if="activeMode === 'advanced'">
              <t-form-item :label="t('scheduledTask.cronScheduleDialog.fields.rawExpression')">
                <t-input
                  v-model="draftExpression"
                  :disabled="disabled"
                  :status="validation.valid ? 'default' : 'error'"
                  :placeholder="t('scheduledTask.cronScheduleDialog.rawPlaceholder')"
                  clearable
                  @change="handleRawInput"
                  @blur="handleRawInput"
                />
              </t-form-item>
              <div class="cron-schedule-dialog__quick-group">
                <span>{{ t('scheduledTask.cronScheduleDialog.quickTemplates') }}</span>
                <div>
                  <t-button
                    v-for="example in exampleOptions"
                    :key="example.expression"
                    size="small"
                    variant="outline"
                    :disabled="disabled"
                    @click="applyExample(example.expression)"
                  >
                    {{ example.label }}
                  </t-button>
                </div>
              </div>
            </template>
          </t-form>
        </div>

        <cron-schedule-preview :expression="draftExpression" />
      </section>
    </div>
  </t-dialog>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  buildCronExpression,
  normalizeCronExpression,
  parseCronExpression,
  toUnixCronExpression,
  validateCronExpression,
} from '../utils/cron';
import CronSchedulePreview from './CronSchedulePreview.vue';

type PlanType = 'intervalMinutes' | 'hourly' | 'daily' | 'weekly' | 'monthly' | 'advanced';
type NumberFieldKey = 'intervalMinutes' | 'minute' | 'hour' | 'weekday' | 'dayOfMonth';
type ExecutionTimeField = {
  key: Extract<NumberFieldKey, 'hour' | 'minute'>;
  max: number;
  min: number;
  separator?: string;
};

type FormState = {
  intervalMinutes: number;
  minute: number;
  hour: number;
  weekday: number;
  dayOfMonth: number;
};

const props = withDefaults(
  defineProps<{
    visible: boolean;
    modelValue: string;
    disabled?: boolean;
  }>(),
  {
    disabled: false,
  },
);

const emit = defineEmits<{
  'update:visible': [value: boolean];
  confirm: [expression: string];
}>();

const { t } = useI18n();

const activeMode = ref<PlanType>('daily');
const draftExpression = ref('0 17 * * *');
const formState = reactive<FormState>({
  intervalMinutes: 5,
  minute: 0,
  hour: 17,
  weekday: 1,
  dayOfMonth: 1,
});

const validation = computed(() => validateCronExpression(draftExpression.value));
const showExecutionTimeFields = computed(() => ['daily', 'weekly', 'monthly'].includes(activeMode.value));
const confirmButton = computed(() => ({
  content: t('scheduledTask.cronScheduleDialog.confirm'),
  disabled: !validation.value.valid || props.disabled,
}));
const minuteQuickOptions = [5, 10, 15, 30];
const executionTimeFields: ExecutionTimeField[] = [
  { key: 'hour', min: 0, max: 23, separator: ':' },
  { key: 'minute', min: 0, max: 59 },
];
const planTypeOptions = computed<Array<{ value: PlanType; label: string }>>(() => [
  { value: 'intervalMinutes', label: t('scheduledTask.cronScheduleDialog.planTypes.intervalMinutes') },
  { value: 'hourly', label: t('scheduledTask.cronScheduleDialog.planTypes.hourly') },
  { value: 'daily', label: t('scheduledTask.cronScheduleDialog.planTypes.daily') },
  { value: 'weekly', label: t('scheduledTask.cronScheduleDialog.planTypes.weekly') },
  { value: 'monthly', label: t('scheduledTask.cronScheduleDialog.planTypes.monthly') },
  { value: 'advanced', label: t('scheduledTask.cronScheduleDialog.planTypes.advanced') },
]);
const weekdayOptions = computed(() =>
  [1, 2, 3, 4, 5, 6, 0].map((value) => ({
    value,
    label: t(`scheduledTask.cronScheduleDialog.weekdays.${value}`),
  })),
);
const exampleOptions = computed(() => [
  { label: t('scheduledTask.cronScheduleDialog.examples.everyFiveMinutes'), expression: '*/5 * * * *' },
  { label: t('scheduledTask.cronScheduleDialog.examples.dailyAtFive'), expression: '0 17 * * *' },
  { label: t('scheduledTask.cronScheduleDialog.examples.weeklyMonday'), expression: '0 9 * * 1' },
  { label: t('scheduledTask.cronScheduleDialog.examples.monthlyFirst'), expression: '0 0 1 * *' },
]);

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      initializeDraft(props.modelValue);
    }
  },
  { immediate: true },
);

function initializeDraft(expression: string) {
  const parsed = parseCronExpression(expression || '0 17 * * *');
  draftExpression.value = parsed.expression;
  activeMode.value = parsed.mode;
  Object.assign(formState, parsed.value);
}

function selectMode(type: PlanType) {
  activeMode.value = type;
  if (type !== 'advanced') {
    syncExpressionFromMode();
  }
}

function syncExpressionFromMode() {
  if (activeMode.value === 'advanced') {
    return;
  }

  draftExpression.value = buildCronExpression(activeMode.value, formState);
}

function setIntervalMinutes(value: number) {
  formState.intervalMinutes = value;
  syncExpressionFromMode();
}

function updateNumberField(key: NumberFieldKey, value: string | number) {
  const limits: Record<NumberFieldKey, [number, number]> = {
    dayOfMonth: [1, 31],
    hour: [0, 23],
    intervalMinutes: [1, 59],
    minute: [0, 59],
    weekday: [0, 6],
  };
  const [min, max] = limits[key];
  formState[key] = clampInteger(value, min, max);
  syncExpressionFromMode();
}

function handleRawInput(value: string | number) {
  draftExpression.value = toUnixCronExpression(String(value));
  if (validateCronExpression(draftExpression.value).valid) {
    const parsed = parseCronExpression(draftExpression.value);
    activeMode.value = parsed.mode;
    Object.assign(formState, parsed.value);
  }
}

function applyExample(expression: string) {
  draftExpression.value = expression;
  const parsed = parseCronExpression(expression);
  activeMode.value = parsed.mode;
  Object.assign(formState, parsed.value);
}

function confirmDraft() {
  if (!validation.value.valid || props.disabled) {
    return;
  }

  emit('confirm', normalizeCronExpression(draftExpression.value));
  emit('update:visible', false);
}

function cancelDraft() {
  emit('update:visible', false);
}

function handleVisibleUpdate(value: boolean) {
  emit('update:visible', value);
}

function clampInteger(value: number | string, min: number, max: number) {
  const numericValue = Number(value);
  if (!Number.isFinite(numericValue)) {
    return min;
  }

  return Math.min(Math.max(Math.trunc(numericValue), min), max);
}
</script>
<style scoped lang="less">
.cron-schedule-dialog {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: 132px minmax(0, 1fr);
  min-height: 420px;
}

.cron-schedule-dialog__nav {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
  padding: var(--graft-density-gap-8);
}

.cron-schedule-dialog__nav-item {
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: var(--td-font-body-medium);
  min-height: var(--td-comp-size-m);
  overflow-wrap: anywhere;
  padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-s);
  text-align: left;
}

.cron-schedule-dialog__nav-item:hover {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}

.cron-schedule-dialog__nav-item--active {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-brand-color);
  font-weight: 600;
}

.cron-schedule-dialog__content {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-rows: minmax(192px, auto) auto;
  min-width: 0;
}

.cron-schedule-dialog__form {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.cron-schedule-dialog__time-fields {
  align-items: center;
  display: grid;
  gap: var(--graft-density-gap-8) var(--graft-density-gap-10);
  grid-template-columns: 120px minmax(16px, auto) 120px;
  width: fit-content;
}

.cron-schedule-dialog__inline-field span,
.cron-schedule-dialog__quick-group > span {
  color: var(--td-text-color-secondary);
}

.cron-schedule-dialog__time-input,
.cron-schedule-dialog__time-fields :deep(.t-input-number) {
  min-width: 0;
  width: 100%;
}

.cron-schedule-dialog__time-separator {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  line-height: var(--td-line-height-body-medium);
  min-width: 16px;
  text-align: center;
  user-select: none;
}

.cron-schedule-dialog__inline-field {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.cron-schedule-dialog__inline-field :deep(.t-input-number) {
  width: 120px;
}

.cron-schedule-dialog__quick-group {
  display: grid;
  gap: var(--graft-density-gap-8);
}

.cron-schedule-dialog__quick-group > div {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.cron-schedule-dialog__field-hint {
  color: var(--td-text-color-secondary);
  margin: var(--graft-density-gap-8) 0 0;
}

@media (width <= 720px) {
  .cron-schedule-dialog {
    grid-template-columns: 1fr;
  }

  .cron-schedule-dialog__nav {
    flex-direction: row;
    overflow-x: auto;
    scrollbar-color: var(--td-scrollbar-color) transparent;
    scrollbar-width: thin;
  }

  .cron-schedule-dialog__nav::-webkit-scrollbar {
    background: transparent;
    height: 8px;
  }

  .cron-schedule-dialog__nav::-webkit-scrollbar-thumb {
    background-clip: content-box;
    background-color: var(--td-scrollbar-color);
    border: 2px solid transparent;
    border-radius: 6px;
  }

  .cron-schedule-dialog__nav-item {
    flex: 0 0 auto;
    white-space: nowrap;
  }

  .cron-schedule-dialog__time-fields {
    grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
    width: 100%;
  }
}
</style>
