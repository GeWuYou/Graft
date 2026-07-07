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
          <div class="rule-collection__row-main">
            <div class="rule-collection__row-title">
              <strong>{{ ruleTitle(rule, index) }}</strong>
              <small>{{ ruleSummary(rule) }}</small>
            </div>
            <div class="rule-collection__row-meta">
              <t-tag size="small" :theme="rule.enabled ? 'success' : 'default'" variant="light">
                {{ rule.enabled ? labels.ruleEnabledState : labels.ruleDisabledState }}
              </t-tag>
              <span v-if="editable" class="rule-collection__drag-hint">
                {{ labels.ruleDragHint }}
              </span>
            </div>
          </div>
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
          <div>
            <strong>{{ labels.detailTitle }}</strong>
            <p>{{ labels.detailDescription }}</p>
          </div>
          <t-tag size="small" :theme="selectedRule.enabled ? 'success' : 'default'" variant="light">
            {{ selectedRule.enabled ? labels.ruleEnabledState : labels.ruleDisabledState }}
          </t-tag>
        </div>

        <div class="rule-collection__detail-body">
          <div class="rule-collection__field">
            <span>{{ labels.ruleTooltipLabel }}</span>
            <t-textarea
              :model-value="selectedRule.tooltip"
              :disabled="disabled"
              :autosize="{ minRows: 2, maxRows: 4 }"
              :placeholder="labels.ruleTooltipPlaceholder"
              data-test-id="rule-tooltip-input"
              @change="(value) => updateSelectedRule('tooltip', String(value ?? ''))"
            />
          </div>

          <div class="rule-collection__field">
            <span>{{ labels.rulePatternLabel }}</span>
            <t-input
              :model-value="selectedRule.pattern"
              :disabled="disabled"
              clearable
              :placeholder="labels.rulePatternPlaceholder"
              data-test-id="rule-pattern-input"
              @change="(value) => updateSelectedRule('pattern', String(value ?? ''))"
            />
            <small>{{ labels.rulePatternDescription }}</small>
          </div>

          <div class="rule-collection__toggle">
            <div>
              <span>{{ labels.ruleEnabledLabel }}</span>
              <small>{{ labels.ruleEnabledDescription }}</small>
            </div>
            <t-switch
              :model-value="selectedRule.enabled"
              :disabled="disabled"
              data-test-id="rule-enabled-switch"
              @change="(value) => updateSelectedRule('enabled', Boolean(value))"
            />
          </div>

          <div class="rule-collection__tester">
            <span>{{ labels.ruleTestLabel }}</span>
            <t-input
              v-model="testValue"
              clearable
              :placeholder="labels.ruleTestPlaceholder"
              data-test-id="rule-test-input"
            />
            <div class="rule-collection__test-result">
              <t-tag size="small" :theme="testResultTheme" variant="light">
                {{ testResultLabel }}
              </t-tag>
              <small>{{ testResultDescription }}</small>
            </div>
          </div>
        </div>

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
      </template>

      <div v-else class="rule-collection__detail-empty">
        <strong>{{ labels.emptyTitle }}</strong>
        <p>{{ labels.emptyDescription }}</p>
      </div>
    </section>
  </div>
</template>
<script setup lang="ts">
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

const rules = computed(() => parseWorkspaceTooltipRules(props.modelValue));
const selectedRule = computed(() => rules.value[selectedIndex.value] ?? null);
const enabledCount = computed(() => rules.value.filter((rule) => rule.enabled).length);
const collectionMetaLabel = computed(
  () =>
    `${replaceCount(props.labels.collectionRuleCount, rules.value.length)} · ${replaceCount(props.labels.collectionEnabledCount, enabledCount.value)}`,
);

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

const testResultLabel = computed(() => {
  if (!testValue.value.trim()) {
    return props.labels.ruleTestEmptyLabel;
  }
  if (!testMatch.value?.valid) {
    return props.labels.invalidPatternLabel;
  }
  return testMatch.value.matched ? props.labels.ruleMatchedLabel : props.labels.ruleUnmatchedLabel;
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
  display: grid;
  gap: var(--graft-density-gap-16);
}

.rule-collection--editable {
  grid-template-columns: minmax(0, 280px) minmax(0, 1fr);
}

.rule-collection__list,
.rule-collection__detail {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  border-radius: var(--td-radius-large);
  min-width: 0;
  padding: var(--graft-density-gap-16);
}

.rule-collection__list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.rule-collection__list-head,
.rule-collection__row-main,
.rule-collection__detail-head,
.rule-collection__toggle,
.rule-collection__detail-actions {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.rule-collection__list-copy,
.rule-collection__row-title,
.rule-collection__detail-head > div,
.rule-collection__toggle > div,
.rule-collection__detail-empty,
.rule-collection__field,
.rule-collection__tester,
.rule-collection__test-result {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xs);
  min-width: 0;
}

.rule-collection__list-copy small,
.rule-collection__row-title small,
.rule-collection__detail-head p,
.rule-collection__toggle small,
.rule-collection__field small,
.rule-collection__test-result small,
.rule-collection__drag-hint {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.rule-collection__rows,
.rule-collection__detail-body {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.rule-collection__row {
  background: linear-gradient(180deg, var(--td-bg-color-container) 0%, var(--td-bg-color-container-hover) 100%);
  border: 1px solid var(--td-component-border);
  border-radius: var(--td-radius-medium);
  cursor: pointer;
  min-width: 0;
  padding: var(--graft-density-gap-12);
  text-align: left;
  transition:
    border-color 160ms ease,
    box-shadow 160ms ease,
    transform 160ms ease;
}

.rule-collection__row:hover {
  border-color: var(--td-brand-color-4);
  transform: translateY(-1px);
}

.rule-collection__row--active {
  border-color: var(--td-brand-color-6);
  box-shadow: 0 0 0 1px var(--td-brand-color-3);
}

.rule-collection__row--dragging {
  opacity: 0.72;
}

.rule-collection__row--disabled {
  opacity: 0.78;
}

.rule-collection__row-title strong,
.rule-collection__detail-head strong,
.rule-collection__field > span,
.rule-collection__toggle span,
.rule-collection__tester > span {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.rule-collection__row-title strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rule-collection__row-title small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rule-collection__row-meta {
  align-items: center;
  display: flex;
  flex: 0 0 auto;
  gap: var(--graft-density-gap-8);
}

.rule-collection__field :deep(.t-textarea__inner),
.rule-collection__field :deep(.t-input__inner),
.rule-collection__tester :deep(.t-input__inner) {
  font-family: ui-sans-serif, system-ui, sans-serif;
}

.rule-collection__detail {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.rule-collection__toggle {
  border: 1px dashed var(--td-component-border);
  border-radius: var(--td-radius-medium);
  padding: var(--graft-density-gap-12);
}

.rule-collection__test-result {
  background: var(--td-bg-color-page);
  border-radius: var(--td-radius-medium);
  padding: var(--graft-density-gap-8) var(--graft-density-gap-12);
}

.rule-collection__detail-empty {
  align-items: flex-start;
  justify-content: center;
  min-height: 220px;
}

@media (width <= 960px) {
  .rule-collection--editable {
    grid-template-columns: 1fr;
  }
}
</style>
