<template>
  <header class="login-header">
    <logo-full-icon class="logo" />
    <div class="operations-container">
      <t-button theme="default" shape="square" variant="text" @click="navToGitHub">
        <t-icon name="logo-github" class="icon" />
      </t-button>
      <t-button theme="default" shape="square" variant="text" @click="navToHelper">
        <t-icon name="help-circle" class="icon" />
      </t-button>
      <language-switcher />
      <t-button theme="default" shape="square" variant="text" @click="toggleSettingPanel">
        <t-icon name="setting" class="icon" />
      </t-button>
    </div>
  </header>
</template>
<script setup lang="ts">
import LogoFullIcon from '@/assets/assets-logo-full.svg?component';
import LanguageSwitcher from '@/shared/components/LanguageSwitcher.vue';
import { useSettingStore } from '@/store';

const settingStore = useSettingStore();

const toggleSettingPanel = () => {
  settingStore.openThemeWorkbench('overview');
};

const navToGitHub = () => {
  window.open('https://github.com/tencent/tdesign-vue-next-starter');
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
    height: var(--td-comp-size-s);
    width: 178px;
  }

  .operations-container {
    align-items: center;
    display: flex;

    .t-button {
      margin-left: var(--td-comp-margin-l);
    }
  }
}
</style>
