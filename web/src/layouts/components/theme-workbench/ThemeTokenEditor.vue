<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="theme-token-editor">
    <div class="editor-toolbar">
      <t-button size="small" variant="text" theme="danger" class="clear-button" @click="clearCurrentGroup">
        {{ t('layout.setting.workbench.actions.clearGroup') }}
      </t-button>
    </div>

    <div v-if="tokenDefinitions.length" class="token-list">
      <theme-token-item
        v-for="token in tokenDefinitions"
        :key="token.key"
        :has-override="hasTokenOverride(token.key)"
        :token="token"
        :value="getResolvedTokenValue(token.key)"
        @commit="(value) => commitToken(token.key, value)"
        @reset="resetToken(token.key)"
      />
    </div>

    <t-empty v-else :description="t('layout.setting.workbench.token.empty')" />
  </div>
</template>
<script setup lang="ts">
import { t } from '@/locales';
import { useSettingStore } from '@/store';
import type { ThemeTokenDefinition, ThemeTokenGroupKey } from '@/types/theme';
import type { ModeType } from '@/utils/types';

import ThemeTokenItem from './ThemeTokenItem.vue';

const props = defineProps<{
  tokenDefinitions: ThemeTokenDefinition[];
  groupKey: ThemeTokenGroupKey;
  mode: ModeType;
}>();

const settingStore = useSettingStore();

const getResolvedTokenValue = (tokenKey: string) => {
  const modeTokens = settingStore.themeResolvedTokens[props.mode];
  return modeTokens[tokenKey] ?? '';
};

const hasTokenOverride = (tokenKey: string) => {
  return Object.prototype.hasOwnProperty.call(settingStore.themeTokenOverrides[props.mode], tokenKey);
};

const resetToken = (tokenKey: string) => {
  settingStore.clearThemeTokenGroup(props.mode, [tokenKey]);
};

const commitToken = (tokenKey: string, tokenValue: string) => {
  const resolvedValue = tokenValue.trim();

  if (!resolvedValue) {
    resetToken(tokenKey);
    return;
  }

  settingStore.updateThemeToken(props.mode, tokenKey, resolvedValue);
};

const clearCurrentGroup = () => {
  settingStore.clearThemeTokenGroup(
    props.mode,
    props.tokenDefinitions.map((token) => token.key),
  );
};
</script>
<style lang="less" scoped>
.theme-token-editor {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
}

.editor-toolbar {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: flex-end;
}

.clear-button {
  opacity: 0.72;
}

.token-list {
  display: grid;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}
</style>
