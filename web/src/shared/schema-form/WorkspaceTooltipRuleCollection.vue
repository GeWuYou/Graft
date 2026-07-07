<template>
  <div :class="['rule-collection', { 'rule-collection--editable': editable }]">
    <section class="rule-collection__list">
      <div class="rule-collection__list-head">
        <div class="rule-collection__list-copy">
          <strong v-if="labels.collectionTitle">{{ labels.collectionTitle }}</strong>
          <small>{{ collectionMetaLabel }}</small>
        </div>
        <t-button
          v-if="editable"
          size="small"
          variant="outline"
          :disabled="disabled"
          data-test-id="rule-collection-add"
          @click="appendRule"
        >
          <template #icon><add-icon /></template>
          {{ labels.ruleAddAction }}
        </t-button>
      </div>

      <div v-if="rules.length" class="rule-collection__rows">
        <button
          v-for="(rule, index) in rules"
          :key="`rule-${index}-${rule.pattern}-${rule.tooltip}`"
          :class="[
            'rule-collection__row',
            {
              'rule-collection__row--active': editable && selectedIndex === index,
              'rule-collection__row--dragging': draggingIndex === index,
              'rule-collection__row--disabled': !rule.enabled,
            },
          ]"
          type="button"
          :draggable="editable && !disabled"
          :data-test-id="`rule-collection-row-${index}`"
          @click="selectRule(index)"
          @dragstart="handleDragStart(index, $event)"
          @dragend="handleDragEnd"
          @dragover.prevent="handleDragOver(index)"
          @drop.prevent="handleDrop(index)"
        >
          <span v-if="editable" class="rule-collection__drag-handle" :aria-label="labels.ruleDragHint">
            <drag-move-icon />
          </span>
          <span class="rule-collection__row-icon">
            <file-icon />
          </span>
          <div class="rule-collection__row-copy">
            <strong>{{ ruleTitle(rule, index) }}</strong>
            <small>{{ ruleSummary(rule) }}</small>
          </div>
          <t-tag
            v-if="editable"
            size="small"
            :theme="rule.enabled ? 'success' : 'default'"
            variant="light"
            class="rule-collection__row-badge"
          >
            {{ rule.enabled ? labels.ruleEnabledState : labels.ruleDisabledState }}
          </t-tag>
        </button>
      </div>

      <t-empty
        v-else
        :title="labels.emptyTitle"
        :description="labels.emptyDescription"
        class="rule-collection__empty"
      />
    </section>

    <section v-if="editable" class="rule-collection__detail">
      <template v-if="selectedRule">
        <div class="rule-collection__detail-head">
          <div class="rule-collection__detail-title">
            <strong>{{ ruleTitle(selectedRule, selectedIndex) }}</strong>
            <p>{{ labels.detailDescription }}</p>
          </div>
          <t-tag size="small" theme="primary" variant="light-outline">
            {{ labels.detailTitle }}
          </t-tag>
        </div>

        <div class="rule-collection__detail-body">
          <section class="rule-collection__panel">
            <header class="rule-collection__panel-head">
              <span>{{ labels.basicInfoTitle }}</span>
            </header>
            <div class="rule-collection__panel-body">
              <div class="rule-collection__field">
                <label>{{ labels.ruleTooltipLabel }}</label>
                <t-input
                  :model-value="selectedRule.tooltip"
                  :disabled="disabled"
                  clearable
                  :placeholder="labels.ruleTooltipPlaceholder"
                  data-test-id="rule-tooltip-input"
                  @change="(value) => updateSelectedRule('tooltip', String(value ?? ''))"
                />
              </div>

              <div class="rule-collection__field">
                <div class="rule-collection__field-head">
                  <label>{{ labels.rulePatternLabel }}</label>
                  <t-button size="small" variant="text" :disabled="disabled" @click="togglePatternExpanded">
                    {{ labels.expandPatternEditorAction }}
                  </t-button>
                </div>
                <t-textarea
                  :model-value="selectedRule.pattern"
                  :disabled="disabled"
                  :autosize="patternAutosize"
                  :placeholder="labels.rulePatternPlaceholder"
                  data-test-id="rule-pattern-input"
                  @change="(value) => updateSelectedRule('pattern', String(value ?? ''))"
                />
                <small>{{ labels.rulePatternDescription }}</small>
              </div>
            </div>
          </section>

          <section class="rule-collection__panel">
            <header class="rule-collection__panel-head">
              <span>{{ labels.sectionToggleTitle }}</span>
            </header>
            <div class="rule-collection__toggle">
              <div>
                <strong>{{ labels.ruleEnabledLabel }}</strong>
                <small>{{ labels.ruleEnabledDescription }}</small>
              </div>
              <t-switch
                :model-value="selectedRule.enabled"
                :disabled="disabled"
                data-test-id="rule-enabled-switch"
                @change="(value) => updateSelectedRule('enabled', Boolean(value))"
              />
            </div>
          </section>

          <section class="rule-collection__panel">
            <header class="rule-collection__panel-head">
              <span>{{ labels.sectionTestTitle }}</span>
            </header>
            <div class="rule-collection__panel-body">
              <div class="rule-collection__field">
                <label>{{ labels.ruleTestLabel }}</label>
                <t-input
                  v-model="testValue"
                  clearable
                  :placeholder="labels.ruleTestPlaceholder"
                  data-test-id="rule-test-input"
                />
              </div>
              <div class="rule-collection__test-result" :class="`rule-collection__test-result--${testResultTheme}`">
                <span class="rule-collection__test-result-icon">
                  <check-circle-filled-icon v-if="testResultTheme === 'success'" />
                  <error-circle-filled-icon v-else-if="testResultTheme === 'warning'" />
                  <info-circle-icon v-else />
                </span>
                <div>
                  <strong>{{ testResultHeadline }}</strong>
                  <small>{{ testResultDescription }}</small>
                </div>
              </div>
            </div>
          </section>

          <section class="rule-collection__panel">
            <header class="rule-collection__panel-head">
              <span>{{ labels.sectionDangerTitle }}</span>
            </header>
            <div class="rule-collection__detail-actions">
              <t-button
                size="small"
                variant="text"
                :disabled="disabled || selectedIndex <= 0"
                data-test-id="rule-move-up"
                @click="moveSelectedRule(-1)"
              >
                {{ labels.ruleUpAction }}
              </t-button>
              <t-button
                size="small"
                variant="text"
                :disabled="disabled || selectedIndex >= rules.length - 1"
                data-test-id="rule-move-down"
                @click="moveSelectedRule(1)"
              >
                {{ labels.ruleDownAction }}
              </t-button>
              <t-button
                size="small"
                theme="danger"
                variant="text"
                :disabled="disabled"
                data-test-id="rule-remove"
                @click="removeSelectedRule"
              >
                {{ labels.ruleRemoveAction }}
              </t-button>
            </div>
          </section>

          <t-collapse borderless expand-icon-placement="right" class="rule-collection__advanced">
            <t-collapse-panel value="json-preview" :header="labels.rulePreviewJsonTitle">
              <pre>{{ serializedPreview }}</pre>
            </t-collapse-panel>
          </t-collapse>
        </div>
      </template>

      <div v-else class="rule-collection__detail-empty">
        <strong>{{ labels.emptyTitle }}</strong>
        <p>{{ labels.emptyDescription }}</p>
      </div>
    </section>
  </div>
</template>
<script setup lang="ts">
import {
  AddIcon,
  CheckCircleFilledIcon,
  DragMoveIcon,
  ErrorCircleFilledIcon,
  FileIcon,
  InfoCircleIcon,
} from 'tdesign-icons-vue-next';
import { computed, ref, watch } from 'vue';

import {
  emptyWorkspaceTooltipRule,
  matchWorkspaceTooltipRule,
  moveWorkspaceTooltipRule,
  parseWorkspaceTooltipRules,
  serializeWorkspaceTooltipRules,
  summarizeWorkspaceTooltipPattern,
  type WorkspaceTooltipRule,
  type WorkspaceTooltipRuleCollectionLabels,
} from './workspace-tooltip-rules';

const props = withDefaults(
  defineProps<{
    disabled?: boolean;
    editable?: boolean;
    labels: WorkspaceTooltipRuleCollectionLabels;
    modelValue: unknown;
  }>(),
  {
    disabled: false,
    editable: false,
  },
);

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

const draggingIndex = ref<number | null>(null);
const selectedIndex = ref(0);
const testValue = ref('');
const patternExpanded = ref(false);

const rules = computed(() => parseWorkspaceTooltipRules(props.modelValue));
const selectedRule = computed(() => rules.value[selectedIndex.value] ?? null);
const enabledCount = computed(() => rules.value.filter((rule) => rule.enabled).length);
const collectionMetaLabel = computed(
  () =>
    `${replaceCount(props.labels.collectionRuleCount, rules.value.length)} · ${replaceCount(props.labels.collectionEnabledCount, enabledCount.value)}`,
);
const patternAutosize = computed(() =>
  patternExpanded.value ? { minRows: 6, maxRows: 10 } : { minRows: 3, maxRows: 5 },
);
const serializedPreview = computed(() => serializeWorkspaceTooltipRules(rules.value));

const testMatch = computed(() => {
  if (!selectedRule.value || !testValue.value.trim()) {
    return null;
  }
  return matchWorkspaceTooltipRule(selectedRule.value.pattern, testValue.value.trim());
});

const testResultTheme = computed<'default' | 'success' | 'warning'>(() => {
  if (!testValue.value.trim()) {
    return 'default';
  }
  if (!testMatch.value?.valid) {
    return 'warning';
  }
  return testMatch.value.matched ? 'success' : 'default';
});

const testResultHeadline = computed(() => {
  if (!testValue.value.trim()) {
    return props.labels.ruleTestEmptyLabel;
  }
  if (!testMatch.value?.valid) {
    return props.labels.invalidPatternLabel;
  }
  return testMatch.value.matched ? props.labels.ruleMatchedRuleLabel : props.labels.ruleUnmatchedLabel;
});

const testResultDescription = computed(() => {
  if (!testValue.value.trim()) {
    return props.labels.ruleTestEmptyDescription;
  }
  if (!testMatch.value?.valid) {
    return props.labels.rulePatternDescription;
  }
  return testMatch.value.matched ? props.labels.ruleMatchedDescription : props.labels.ruleUnmatchedDescription;
});

watch(
  rules,
  (nextRules) => {
    if (!nextRules.length) {
      selectedIndex.value = 0;
      return;
    }
    if (selectedIndex.value >= nextRules.length) {
      selectedIndex.value = nextRules.length - 1;
    }
  },
  { immediate: true },
);

function replaceCount(template: string, count: number) {
  return template.replace('{count}', String(count));
}

function ruleTitle(rule: WorkspaceTooltipRule, index: number) {
  const tooltipTitle = rule.tooltip
    .split('\n')
    .map((line) => line.trim())
    .find((line) => line.length > 0);
  if (tooltipTitle) {
    return tooltipTitle;
  }

  const patternSummary = summarizeWorkspaceTooltipPattern(rule.pattern, 32);
  if (patternSummary) {
    return patternSummary;
  }

  return props.labels.ruleFallbackTitle.replace('{index}', String(index + 1));
}

function ruleSummary(rule: WorkspaceTooltipRule) {
  return summarizeWorkspaceTooltipPattern(rule.pattern) || props.labels.noPatternSummary;
}

function commit(nextRules: WorkspaceTooltipRule[]) {
  emit('update:modelValue', serializeWorkspaceTooltipRules(nextRules));
}

function selectRule(index: number) {
  if (!props.editable) {
    return;
  }
  selectedIndex.value = index;
}

function appendRule() {
  const nextRules = [...rules.value, emptyWorkspaceTooltipRule()];
  commit(nextRules);
  selectedIndex.value = nextRules.length - 1;
}

function updateSelectedRule(field: keyof WorkspaceTooltipRule, value: string | boolean) {
  commit(rules.value.map((rule, index) => (index === selectedIndex.value ? { ...rule, [field]: value } : rule)));
}

function moveSelectedRule(offset: -1 | 1) {
  const targetIndex = selectedIndex.value + offset;
  const nextRules = moveWorkspaceTooltipRule(rules.value, selectedIndex.value, targetIndex);
  commit(nextRules);
  selectedIndex.value = Math.max(0, Math.min(targetIndex, nextRules.length - 1));
}

function removeSelectedRule() {
  const nextRules = rules.value.filter((_, index) => index !== selectedIndex.value);
  commit(nextRules);
  selectedIndex.value = Math.max(0, Math.min(selectedIndex.value, nextRules.length - 1));
}

function togglePatternExpanded() {
  patternExpanded.value = !patternExpanded.value;
}

function handleDragStart(index: number, event: DragEvent) {
  if (!props.editable || props.disabled) {
    return;
  }
  draggingIndex.value = index;
  event.dataTransfer?.setData('text/plain', String(index));
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move';
  }
}

function handleDragOver(index: number) {
  if (!props.editable || props.disabled || draggingIndex.value === null) {
    return;
  }
  selectedIndex.value = index;
}

function handleDrop(index: number) {
  if (!props.editable || props.disabled || draggingIndex.value === null) {
    return;
  }
  const fromIndex = draggingIndex.value;
  const nextRules = moveWorkspaceTooltipRule(rules.value, fromIndex, index);
  commit(nextRules);
  selectedIndex.value = index;
  draggingIndex.value = null;
}

function handleDragEnd() {
  draggingIndex.value = null;
}
</script>
<style scoped>
.rule-collection {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.rule-collection--editable {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: minmax(220px, 0.9fr) minmax(0, 2.1fr);
}

.rule-collection__list,
.rule-collection__panel,
.rule-collection__detail-head,
.rule-collection__detail-empty {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
}

.rule-collection__list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  padding: var(--graft-density-gap-12);
}

.rule-collection__list-head,
.rule-collection__detail-head,
.rule-collection__field-head,
.rule-collection__toggle,
.rule-collection__detail-actions {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
}

.rule-collection__list-copy,
.rule-collection__row-copy,
.rule-collection__detail-title,
.rule-collection__panel-body,
.rule-collection__field,
.rule-collection__test-result,
.rule-collection__detail-empty {
  display: flex;
  flex-direction: column;
}

.rule-collection__list-copy strong,
.rule-collection__row-copy strong,
.rule-collection__detail-title strong,
.rule-collection__panel-head span,
.rule-collection__toggle strong,
.rule-collection__test-result strong,
.rule-collection__detail-empty strong {
  color: var(--td-text-color-primary);
}

.rule-collection__list-copy small,
.rule-collection__row-copy small,
.rule-collection__detail-title p,
.rule-collection__field small,
.rule-collection__toggle small,
.rule-collection__test-result small,
.rule-collection__detail-empty p {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.rule-collection__rows {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
}

.rule-collection__row {
  align-items: center;
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  cursor: pointer;
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: auto auto minmax(0, 1fr) auto;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
  text-align: left;
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease,
    transform 0.2s ease;
}

.rule-collection__row:hover,
.rule-collection__row--active {
  background: color-mix(in srgb, var(--td-bg-color-container-hover) 78%, var(--td-bg-color-secondarycontainer));
  border-color: color-mix(in srgb, var(--td-brand-color) 42%, var(--td-border-level-2-color));
}

.rule-collection__row--dragging {
  opacity: 0.7;
}

.rule-collection__row--disabled {
  opacity: 0.72;
}

.rule-collection__drag-handle,
.rule-collection__row-icon {
  align-items: center;
  color: var(--td-text-color-placeholder);
  display: inline-flex;
  font-size: var(--td-font-size-body-medium);
  justify-content: center;
}

.rule-collection__drag-handle {
  cursor: grab;
}

.rule-collection__row-copy {
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.rule-collection__row-copy strong,
.rule-collection__row-copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rule-collection__row-badge {
  flex: 0 0 auto;
}

.rule-collection__detail {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.rule-collection__detail-head,
.rule-collection__detail-empty {
  padding: var(--graft-density-gap-14) var(--graft-density-gap-16);
}

.rule-collection__detail-title {
  gap: var(--graft-density-gap-4);
}

.rule-collection__detail-body {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.rule-collection__panel {
  background: var(--td-bg-color-container);
  overflow: hidden;
}

.rule-collection__panel-head {
  border-bottom: 1px solid var(--td-border-level-1-color);
  padding: var(--graft-density-gap-12) var(--graft-density-gap-16);
}

.rule-collection__panel-body,
.rule-collection__toggle,
.rule-collection__detail-actions {
  gap: var(--graft-density-gap-12);
  padding: var(--graft-density-gap-16);
}

.rule-collection__field {
  gap: var(--graft-density-gap-8);
}

.rule-collection__field label {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.rule-collection__toggle {
  align-items: center;
}

.rule-collection__toggle > div {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
}

.rule-collection__test-result {
  align-items: flex-start;
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-small);
  flex-direction: row;
  gap: var(--graft-density-gap-10);
  padding: var(--graft-density-gap-12);
}

.rule-collection__test-result-icon {
  align-items: center;
  display: inline-flex;
  font-size: var(--td-font-size-body-medium);
  justify-content: center;
}

.rule-collection__test-result--success {
  border-color: var(--td-success-color-3);
}

.rule-collection__test-result--success .rule-collection__test-result-icon {
  color: var(--td-success-color-6);
}

.rule-collection__test-result--warning {
  border-color: var(--td-warning-color-3);
}

.rule-collection__test-result--warning .rule-collection__test-result-icon {
  color: var(--td-warning-color-6);
}

.rule-collection__test-result--default .rule-collection__test-result-icon {
  color: var(--td-text-color-placeholder);
}

.rule-collection__detail-actions {
  justify-content: flex-start;
}

.rule-collection__advanced :deep(.t-collapse-panel__content) {
  padding: 0;
}

.rule-collection__advanced pre {
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-small);
  box-sizing: border-box;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  margin: var(--graft-density-gap-8) 0 0;
  max-height: 220px;
  overflow: auto;
  padding: var(--graft-density-gap-12);
  white-space: pre-wrap;
}

.rule-collection__empty {
  padding: var(--graft-density-gap-20) 0;
}

@media (width <= 900px) {
  .rule-collection--editable {
    grid-template-columns: minmax(0, 1fr);
  }

  .rule-collection__row {
    grid-template-columns: auto auto minmax(0, 1fr);
  }

  .rule-collection__row-badge {
    grid-column: 2 / 4;
    justify-self: start;
  }

  .rule-collection__field-head,
  .rule-collection__toggle,
  .rule-collection__detail-actions,
  .rule-collection__detail-head {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
