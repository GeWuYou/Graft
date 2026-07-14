<template>
  <header class="login-header">
    <brand-identity class="logo" :label="t('common.appName')" />
    <div class="operations-container">
      <t-button theme="default" shape="square" variant="text" @click="navToGitHub">
        <t-icon name="logo-github" class="icon" />
      </t-button>
      <t-button theme="default" shape="square" variant="text" @click="navToDocs">
        <t-icon name="book-open" class="icon" />
      </t-button>
      <t-button theme="default" shape="square" variant="text" @click="navToHelper">
        <t-icon name="help-circle" class="icon" />
      </t-button>
      <language-switcher />
      <t-tooltip placement="bottom" :content="t('layout.header.personalization')">
        <t-button theme="default" shape="square" variant="text" @click="toggleSettingPanel">
          <t-icon name="palette" class="icon" />
        </t-button>
      </t-tooltip>
    </div>
  </header>
</template>
<script setup lang="ts">
import { t } from '@/locales';
import { BrandIdentity } from '@/shared/components/brand';
import LanguageSwitcher from '@/shared/components/LanguageSwitcher.vue';
import { useSettingStore } from '@/store';

const settingStore = useSettingStore();

const toggleSettingPanel = () => {
  settingStore.openThemeWorkbench('overview');
};

const navToGitHub = () => {
  window.open('https://github.com/GeWuYou/Graft');
};

const navToDocs = () => {
  window.open('/docs');
};

const navToHelper = () => {
  window.open('http://tdesign.tencent.com/starter/docs/get-started');
};
</script>
<style lang="less" scoped>
.login-header {
  align-items: center;
  backdrop-filter: blur(10px);
  color: var(--td-text-color-primary);
  display: flex;
  height: var(--td-comp-size-xxxl);
  justify-content: space-between;
  padding: 0 var(--td-comp-paddingLR-xl);

  .logo {
    height: 32px;
    width: 168px;
  }

  .operations-container {
    align-items: center;
    display: inline-flex;
    gap: var(--graft-density-gap-4);
    line-height: 0;
  }
}
</style>
