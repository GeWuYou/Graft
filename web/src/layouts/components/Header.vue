<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div :class="layoutCls">
    <t-head-menu :class="menuCls" :theme="menuTheme" expand-type="popup" :value="active">
      <template #logo>
        <span v-if="showLogo" class="header-logo-container" @click="goHome">
          <logo-full class="t-logo" />
        </span>
        <div v-else class="header-operate-left">
          <t-button theme="default" shape="square" variant="text" @click="changeCollapsed">
            <t-icon class="collapsed-icon" name="view-list" />
          </t-button>
          <search />
        </div>
      </template>
      <template v-if="layout !== 'side'" #default>
        <menu-content class="header-menu" :nav-data="menu" />
      </template>
      <template #operations>
        <div class="operations-container">
          <!-- 搜索框 -->
          <search v-if="layout !== 'side'" />

          <!-- 全局通知 -->
          <notice />

          <t-tooltip placement="bottom" :content="t('layout.header.code')">
            <t-button theme="default" shape="square" variant="text" @click="navToGitHub">
              <t-icon name="logo-github" />
            </t-button>
          </t-tooltip>
          <t-tooltip placement="bottom" :content="t('layout.header.apiDocs')">
            <t-button theme="default" shape="square" variant="text" @click="navToDocs">
              <t-icon name="book-open" />
            </t-button>
          </t-tooltip>
          <t-tooltip placement="bottom" :content="t('layout.header.help')">
            <t-button theme="default" shape="square" variant="text" @click="navToHelper">
              <t-icon name="help-circle" />
            </t-button>
          </t-tooltip>
          <language-switcher />
          <t-dropdown :min-column-width="120" trigger="click">
            <template #dropdown>
              <t-dropdown-item class="operations-dropdown-container-item" @click="handleNav(USER_ROUTE_PATH.LIST)">
                <user-circle-icon />{{ t('layout.header.user') }}
              </t-dropdown-item>
              <t-dropdown-item class="operations-dropdown-container-item" @click="handleLogout">
                <poweroff-icon />{{ t('layout.header.signOut') }}
              </t-dropdown-item>
            </template>
            <t-button class="header-user-btn" theme="default" variant="text">
              <template #icon>
                <t-icon class="header-user-avatar" name="user-circle" />
              </template>
              <div class="header-user-account">{{ user.userInfo.name }}</div>
              <template #suffix><chevron-down-icon /></template>
            </t-button>
          </t-dropdown>
          <t-tooltip placement="bottom" :content="t('layout.header.setting')">
            <t-button theme="default" shape="square" variant="text" @click="toggleSettingPanel">
              <setting-icon />
            </t-button>
          </t-tooltip>
        </div>
      </template>
    </t-head-menu>
  </div>
</template>
<script setup lang="ts">
import { ChevronDownIcon, PoweroffIcon, SettingIcon, UserCircleIcon } from 'tdesign-icons-vue-next';
import type { PropType } from 'vue';
import { computed } from 'vue';
import { useRouter } from 'vue-router';

import LogoFull from '@/assets/assets-logo-full.svg?component';
import { prefix } from '@/config/global';
import { useShellNavigation } from '@/layouts/useShellNavigation';
import { t } from '@/locales';
import { AUTH_ROUTE_PATH } from '@/modules/auth/contract/routes';
import { useAuthSessionStore } from '@/modules/auth/store';
import { USER_ROUTE_PATH } from '@/modules/user/contract/paths';
import { getActive } from '@/router';
import LanguageSwitcher from '@/shared/components/LanguageSwitcher.vue';
import { useSettingStore } from '@/store';
import type { MenuRoute, ModeType } from '@/utils/types';

import MenuContent from './MenuContent.vue';
import Notice from './Notice.vue';
import Search from './Search.vue';

const { theme, layout, showLogo, menu, isFixed, isCompact } = defineProps({
  theme: {
    type: String,
    default: 'light',
  },
  layout: {
    type: String,
    default: 'top',
  },
  showLogo: {
    type: Boolean,
    default: true,
  },
  menu: {
    type: Array as PropType<MenuRoute[]>,
    default: () => [],
  },
  isFixed: {
    type: Boolean,
    default: false,
  },
  isCompact: {
    type: Boolean,
    default: false,
  },
  maxLevel: {
    type: Number,
    default: 3,
  },
});

const router = useRouter();
const settingStore = useSettingStore();
const user = useAuthSessionStore();
const { goHome } = useShellNavigation();

const toggleSettingPanel = () => {
  settingStore.openThemeWorkbench('overview');
};

const active = computed(() => getActive());

const layoutCls = computed(() => [`${prefix}-header-layout`]);

const menuCls = computed(() => {
  return [
    {
      [`${prefix}-header-menu`]: !isFixed,
      [`${prefix}-header-menu-fixed`]: isFixed,
      [`${prefix}-header-menu-fixed-side`]: layout === 'side' && isFixed,
      [`${prefix}-header-menu-fixed-side-compact`]: layout === 'side' && isFixed && isCompact,
    },
  ];
});
const menuTheme = computed(() => theme as ModeType);

// 切换语言
const changeCollapsed = () => {
  settingStore.updateConfig({
    isSidebarCompact: !settingStore.isSidebarCompact,
  });
};

const handleNav = (url: string) => {
  router.push(url);
};

const handleLogout = async () => {
  try {
    await user.logout();
  } finally {
    await router.push({
      path: AUTH_ROUTE_PATH.LOGIN,
    });
  }
};

const navToGitHub = () => {
  window.open('https://github.com/GeWuYou/Graft');
};

const navToDocs = () => {
  window.open('/docs');
};

const navToHelper = () => {
  window.open('https://tdesign.tencent.com/starter/docs/vue-next/get-started');
};
</script>
<style lang="less" scoped>
.@{starter-prefix}-header {
  &-menu-fixed {
    position: fixed;
    top: 0;
    z-index: 1001;

    :deep(.t-head-menu__inner) {
      padding-right: var(--td-comp-margin-xl);
    }

    &-side {
      left: 232px;
      right: 0;
      transition: all 0.3s;
      width: auto;
      z-index: 10;

      &-compact {
        left: 64px;
      }
    }
  }

  &-logo-container {
    cursor: pointer;
    display: inline-flex;
  }
}

.header-menu {
  display: inline-flex;
  flex: 1 1 auto;

  :deep(.t-menu__item) {
    min-width: unset;
  }
}

.operations-container {
  align-items: center;
  display: flex;
  height: 100%;

  :deep(.t-popup__reference) {
    align-items: center;
    display: flex;
    height: var(--td-comp-size-m);
    justify-content: center;
  }

  :deep(.t-button) {
    align-items: center;
    display: inline-flex;
    height: var(--td-comp-size-m);
    justify-content: center;
    margin-left: var(--td-comp-margin-l);
    min-width: var(--td-comp-size-m);
    padding: 0;
  }
}

.header-operate-left {
  align-items: normal;
  background: transparent;
  display: flex;
  line-height: 0;
}

.header-logo-container {
  color: var(--td-text-color-primary);
  display: flex;
  height: 26px;
  margin-left: var(--graft-density-gap-24);
  width: 184px;

  .t-logo {
    height: 100%;
    width: 100%;

    &:hover {
      cursor: pointer;
    }
  }

  &:hover {
    cursor: pointer;
  }
}

.header-user-account {
  align-items: center;
  color: var(--td-text-color-primary);
  display: inline-flex;
}

:deep(.t-head-menu__inner) {
  border-bottom: 1px solid var(--td-component-stroke);
}

.t-menu--light {
  .header-user-account {
    color: var(--td-text-color-primary);
  }
}

.t-menu--dark {
  background: var(--graft-shell-header-bg);

  .t-head-menu__inner {
    border-bottom: 1px solid var(--graft-shell-border-color);
  }

  :deep(.t-head-menu__inner),
  :deep(.t-menu__logo),
  :deep(.t-menu),
  :deep(.t-menu__operations),
  .header-operate-left,
  .operations-container,
  .header-menu,
  .header-logo-container {
    background: var(--graft-shell-header-bg);
  }

  :deep(.t-menu__item) {
    background: transparent;
  }

  :deep(.t-button.t-button--variant-text) {
    background: transparent;
    border-color: transparent;
    color: var(--td-text-color-secondary);
  }

  :deep(.t-button.t-button--variant-text:hover) {
    background: var(--graft-dark-header-button-hover-bg);
    border-color: transparent;
    color: var(--td-text-color-primary);
  }

  :deep(.t-button.t-button--variant-text:active),
  :deep(.t-button.t-button--variant-text.t-is-active) {
    background: var(--graft-dark-header-button-active-bg);
    border-color: transparent;
    color: var(--td-brand-color);
  }

  .header-user-account {
    color: var(--td-text-color-secondary);
  }
}

.operations-dropdown-container-item {
  align-items: center;
  display: flex;
  width: 100%;

  :deep(.t-dropdown__item-text) {
    align-items: center;
    display: flex;
  }

  .t-icon {
    font-size: var(--td-comp-size-xxxs);
    margin-right: var(--td-comp-margin-s);
  }

  :deep(.t-dropdown__item) {
    margin-bottom: 0;
    width: 100%;
  }

  &:last-child {
    :deep(.t-dropdown__item) {
      margin-bottom: var(--graft-density-gap-8);
    }
  }
}
</style>
<!-- eslint-disable-next-line vue-scoped-css/enforce-style-type -->
<style lang="less">
.operations-dropdown-container-item {
  .t-dropdown__item-text {
    align-items: center;
    display: flex;
  }
}
</style>
