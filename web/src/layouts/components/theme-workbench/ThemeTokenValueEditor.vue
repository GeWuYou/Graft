<template>
  <div class="theme-token-value-editor">
    <div v-if="isColorToken" class="editor-row">
      <div class="editor-row__label">{{ t('layout.setting.workbench.token.preview') }}</div>
      <color-value-preview :token-key="token.key" :value="rawValue" />
      <t-color-picker
        class="editor-color-picker"
        :color-modes="colorPickerModes"
        enable-alpha
        format="CSS"
        :model-value="rawValue"
        :popup-props="{ placement: 'bottom-right' }"
        @change="handlePickerChange"
      />
    </div>

    <div v-if="isColorToken" class="editor-row">
      <div class="editor-row__label">{{ t('layout.setting.workbench.token.hex') }}</div>
      <t-input
        :model-value="hexDraft"
        placeholder="#0052D9"
        :status="hexInputStatus"
        @update:model-value="(value) => (hexDraft = String(value ?? ''))"
        @change="commitColorDraft"
        @blur="commitColorDraft"
      />
    </div>

    <div v-if="showOpacityInput" class="editor-row">
      <div class="editor-row__label">{{ t('layout.setting.workbench.token.opacity') }}</div>
      <t-input
        :model-value="opacityDraft"
        placeholder="100"
        suffix="%"
        type="number"
        :status="opacityInputStatus"
        @update:model-value="(value) => (opacityDraft = String(value ?? ''))"
        @change="commitColorDraft"
        @blur="commitColorDraft"
      />
    </div>

    <div v-if="!isColorToken" class="editor-row">
      <div class="editor-row__label">{{ t('layout.setting.workbench.token.value') }}</div>
      <t-input
        :model-value="rawValue"
        @update:model-value="(value) => (rawValue = String(value ?? ''))"
        @change="commitRawValue"
        @blur="commitRawValue"
      />
    </div>

    <div class="editor-actions">
      <t-button size="small" variant="outline" @click="copyVariableName">
        {{ t('layout.setting.workbench.token.copyVariable') }}
      </t-button>
      <t-button size="small" variant="text" :disabled="!hasOverride" @click="$emit('reset')">
        {{ t('layout.setting.workbench.token.reset') }}
      </t-button>
    </div>
  </div>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, ref, watch } from 'vue';

import { t } from '@/locales';
import { copyText } from '@/shared/observability';
import type { ThemeTokenDefinition } from '@/types/theme';

import ColorValuePreview from './ColorValuePreview.vue';
import { buildThemeTokenColorValue, isThemeTokenColorKey, parseThemeTokenColor } from './theme-token-color';

const props = defineProps<{
  hasOverride: boolean;
  token: ThemeTokenDefinition;
  value: string;
}>();

const emit = defineEmits<{
  commit: [value: string];
  reset: [];
}>();

const colorPickerModes: Array<'monochrome'> = ['monochrome'];
const rawValue = ref('');
const hexDraft = ref('');
const opacityDraft = ref('100');

const isColorToken = computed(() => isThemeTokenColorKey(props.token.key));
const parsedColor = computed(() => parseThemeTokenColor(rawValue.value) ?? parseThemeTokenColor(props.value));
const showOpacityInput = computed(() => isColorToken.value && parsedColor.value !== null);
const hexInputStatus = computed(() => {
  if (!isColorToken.value || !hexDraft.value.trim()) {
    return 'default';
  }

  return buildThemeTokenColorValue(hexDraft.value, parseOpacityValue(opacityDraft.value) ?? 100) ? 'default' : 'error';
});
const opacityInputStatus = computed(() => {
  if (!showOpacityInput.value || !opacityDraft.value.trim()) {
    return 'default';
  }

  const parsed = parseOpacityValue(opacityDraft.value);
  return parsed === null ? 'error' : 'default';
});

function syncDrafts() {
  rawValue.value = props.value;

  const color = parseThemeTokenColor(props.value);
  hexDraft.value = color?.hex ?? '';
  opacityDraft.value = String(color ? Math.round(color.alpha * 100) : 100);
}

function parseOpacityValue(value: string) {
  const trimmed = value.trim().replace(/%$/, '');

  if (!trimmed) {
    return 100;
  }

  const parsed = Number(trimmed);

  if (!Number.isFinite(parsed) || parsed < 0 || parsed > 100) {
    return null;
  }

  return parsed;
}

function commitRawValue() {
  emit('commit', rawValue.value);
}

function commitColorDraft() {
  if (!hexDraft.value.trim()) {
    emit('reset');
    return;
  }

  const opacity = parseOpacityValue(opacityDraft.value);

  if (opacity === null) {
    return;
  }

  const nextValue = buildThemeTokenColorValue(hexDraft.value, opacity);

  if (!nextValue) {
    return;
  }

  rawValue.value = nextValue;
  emit('commit', nextValue);
}

function handlePickerChange(value: string) {
  rawValue.value = value;
  const color = parseThemeTokenColor(value);

  if (color) {
    hexDraft.value = color.hex;
    opacityDraft.value = String(Math.round(color.alpha * 100));
  }

  emit('commit', value);
}

async function copyVariableName() {
  const copied = await copyText(props.token.key);

  if (copied) {
    MessagePlugin.success(t('layout.setting.workbench.token.copyVariableSuccess'));
    return;
  }

  MessagePlugin.error(t('layout.setting.workbench.token.copyVariableFail'));
}

watch(() => props.value, syncDrafts, { immediate: true });
watch(() => props.token.key, syncDrafts);
</script>
<style lang="less" scoped>
.theme-token-value-editor {
  display: grid;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.editor-row {
  display: grid;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.editor-row__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  font-weight: 600;
}

.editor-color-picker {
  width: 100%;
}

.editor-color-picker :deep(.t-input) {
  width: 100%;
}

.editor-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.editor-actions :deep(.t-button) {
  margin: 0;
}
</style>
