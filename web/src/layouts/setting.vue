<template>
  <t-drawer
    v-model:visible="showSettingPanel"
    size="408px"
    :footer="false"
    :header="t('layout.setting.title')"
    :close-btn="true"
    class="setting-drawer-container"
    @close-btn-click="handleCloseDrawer"
  >
    <div class="setting-container">
      <t-form :data="formData" label-align="left">
        <div class="setting-group-title">{{ t('layout.setting.theme.mode') }}</div>
        <t-radio-group v-model="formData.mode">
          <div v-for="(item, index) in MODE_OPTIONS" :key="index" class="setting-layout-drawer">
            <div>
              <t-radio-button :key="index" :value="item.type"
                ><component :is="getModeIcon(item.type)"
              /></t-radio-button>
              <p :style="{ textAlign: 'center', marginTop: '8px' }">{{ item.text }}</p>
            </div>
          </div>
        </t-radio-group>
        <div class="setting-group-title">{{ t('layout.setting.theme.color') }}</div>
        <t-radio-group v-model="formData.brandTheme">
          <div v-for="(item, index) in DEFAULT_COLOR_OPTIONS" :key="index" class="setting-layout-drawer">
            <t-radio-button :key="index" :value="item" class="setting-layout-color-group">
              <color-container :value="item" />
            </t-radio-button>
          </div>
          <div class="setting-layout-drawer">
            <t-popup
              destroy-on-close
              expand-animation
              placement="bottom-right"
              trigger="click"
              :visible="isColoPickerDisplay"
              :overlay-style="{ padding: 0 }"
              @visible-change="onPopupVisibleChange"
            >
              <template #content>
                <t-color-picker-panel
                  :on-change="changeColor"
                  :color-modes="['monochrome']"
                  format="HEX"
                  :swatch-colors="[]"
                />
              </template>
              <t-radio-button :value="dynamicColor" class="setting-layout-color-group dynamic-color-btn">
                <color-container :value="dynamicColor" />
              </t-radio-button>
            </t-popup>
          </div>
        </t-radio-group>
        <div class="setting-group-title">{{ t('layout.setting.navigationLayout') }}</div>
        <t-radio-group v-model="formData.layout">
          <div v-for="(item, index) in LAYOUT_OPTION" :key="index" class="setting-layout-drawer">
            <t-radio-button :key="index" :value="item">
              <thumbnail :src="getThumbnailUrl(item)" />
            </t-radio-button>
          </div>
        </t-radio-group>

        <t-form-item v-show="formData.layout === 'mix'" :label="t('layout.setting.splitMenu')" name="splitMenu">
          <t-switch v-model="formData.splitMenu" />
        </t-form-item>
        <t-form-item v-show="formData.layout === 'mix'" :label="t('layout.setting.fixedSidebar')" name="isSidebarFixed">
          <t-switch v-model="formData.isSidebarFixed" />
        </t-form-item>

        <div class="setting-group-title">{{ t('layout.setting.element.title') }}</div>
        <t-form-item :label="t('layout.setting.sideMode')" name="sideMode">
          <t-radio-group v-model="formData.sideMode" class="side-mode-radio">
            <t-radio-button key="light" value="light" :label="t('layout.setting.theme.options.light')" />
            <t-radio-button key="dark" value="dark" :label="t('layout.setting.theme.options.dark')" />
          </t-radio-group>
        </t-form-item>
        <t-form-item
          v-show="formData.layout === 'side'"
          :label="t('layout.setting.element.showHeader')"
          name="showHeader"
        >
          <t-switch v-model="formData.showHeader" />
        </t-form-item>
        <t-form-item :label="t('layout.setting.element.showBreadcrumb')" name="showBreadcrumb">
          <t-switch v-model="formData.showBreadcrumb" />
        </t-form-item>
        <t-form-item :label="t('layout.setting.element.showFooter')" name="showFooter">
          <t-switch v-model="formData.showFooter" />
        </t-form-item>
        <t-form-item :label="t('layout.setting.element.useTagTabs')" name="isUseTabsRouter">
          <t-switch v-model="formData.isUseTabsRouter"></t-switch>
        </t-form-item>
        <t-form-item :label="t('layout.setting.element.menuAutoCollapsed')" name="menuAutoCollapsed">
          <t-switch v-model="formData.menuAutoCollapsed"></t-switch>
        </t-form-item>
      </t-form>
      <div class="setting-info">
        <p>{{ t('layout.setting.tips') }}</p>
        <t-button theme="primary" variant="text" @click="handleCopy">
          {{ t('layout.setting.copy.title') }}
        </t-button>
      </div>
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
import { useClipboard } from '@vueuse/core';
import type { PopupVisibleChangeContext } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, ref, watchEffect } from 'vue';

import SettingAutoIcon from '@/assets/assets-setting-auto.svg';
import SettingDarkIcon from '@/assets/assets-setting-dark.svg';
import SettingLightIcon from '@/assets/assets-setting-light.svg';
import ColorContainer from '@/components/color/index.vue';
import Thumbnail from '@/components/thumbnail/index.vue';
import { DEFAULT_COLOR_OPTIONS } from '@/config/color';
import STYLE_CONFIG from '@/config/style';
import { t } from '@/locales';
import { useSettingStore } from '@/store';

const settingStore = useSettingStore();

const LAYOUT_OPTION = ['side', 'top', 'mix'];

const MODE_OPTIONS = computed(() => [
  { type: 'light', text: t('layout.setting.theme.options.light') },
  { type: 'dark', text: t('layout.setting.theme.options.dark') },
  { type: 'auto', text: t('layout.setting.theme.options.auto') },
]);

const initStyleConfig = () => {
  const styleConfig = STYLE_CONFIG;
  for (const key in styleConfig) {
    if (Object.prototype.hasOwnProperty.call(styleConfig, key)) {
      (styleConfig[key as keyof typeof STYLE_CONFIG] as any) = settingStore[key as keyof typeof STYLE_CONFIG];
    }
  }

  return styleConfig;
};

const dynamicColor = computed(() => {
  const isDynamic = DEFAULT_COLOR_OPTIONS.includes(formData.value.brandTheme);
  return isDynamic ? formData.value.brandTheme : '';
});
const formData = ref({ ...initStyleConfig() });
const isColoPickerDisplay = ref(false);

const showSettingPanel = computed({
  get() {
    return settingStore.showSettingPanel;
  },
  set(newVal: boolean) {
    settingStore.updateConfig({
      showSettingPanel: newVal,
    });
  },
});

const changeColor = (hex: string) => {
  formData.value.brandTheme = hex;
};

onMounted(() => {
  document.querySelector('.dynamic-color-btn')?.addEventListener('click', () => {
    isColoPickerDisplay.value = true;
  });
});

const onPopupVisibleChange = (visible: boolean, context: PopupVisibleChangeContext) => {
  if (!visible && context.trigger === 'document') {
    isColoPickerDisplay.value = visible;
  }
};

const handleCopy = () => {
  const sourceText = JSON.stringify(formData.value, null, 4);
  const { copy } = useClipboard({ source: sourceText });
  copy()
    .then(() => {
      MessagePlugin.closeAll();
      MessagePlugin.success(t('components.copySuccess'));
    })
    .catch(() => {
      MessagePlugin.closeAll();
      MessagePlugin.error(t('components.copyFail'));
    });
};
const getModeIcon = (mode: string) => {
  if (mode === 'light') {
    return SettingLightIcon;
  }
  if (mode === 'dark') {
    return SettingDarkIcon;
  }
  return SettingAutoIcon;
};

const handleCloseDrawer = () => {
  settingStore.updateConfig({
    showSettingPanel: false,
  });
};

const getThumbnailUrl = (name: string): string => {
  return `https://tdesign.gtimg.com/tdesign-pro/setting/${name}.png`;
};

watchEffect(() => {
  if (formData.value.brandTheme) settingStore.updateConfig(formData.value);
});
</script>
<!-- teleport导致drawer 内 scoped样式问题无法生效 先规避下 -->
<!-- eslint-disable-next-line vue-scoped-css/enforce-style-type -->
<style lang="less">
.tdesign-setting {
  border-radius: 20px 0 0 20px;
  bottom: 200px;
  height: 40px;
  position: fixed;
  right: 0;
  transition: all 0.3s;
  width: 40px;
  z-index: 100;

  .t-icon {
    margin-left: 8px;
  }

  .tdesign-setting-text {
    display: none;
    font-size: 12px;
  }

  &:hover {
    width: 96px;

    .tdesign-setting-text {
      display: inline-block;
    }
  }
}

.setting-layout-color-group {
  align-items: center;
  border: 2px solid transparent !important;
  border-radius: 50% !important;
  display: inline-flex;
  justify-content: center;
  padding: 6px !important;

  > .t-radio-button__label {
    display: inline-flex;
  }
}

.tdesign-setting-close {
  bottom: 200px;
  position: fixed;
  right: 300px;
}

.setting-group-title {
  color: var(--td-text-color-primary);
  font-family: 'PingFang SC', var(--td-font-family);
  font-size: 14px;
  font-style: normal;
  font-weight: 500;
  line-height: 22px;
  margin: 32px 0 24px;
  text-align: left;
}

.setting-link {
  color: var(--td-brand-color);
  cursor: pointer;
  margin-bottom: 8px;
}

.setting-info {
  background: var(--td-bg-color-container);
  bottom: 0;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  left: 0;
  line-height: 20px;
  padding: 24px;
  position: absolute;
  text-align: center;
  width: 100%;
}

.setting-drawer-container {
  .setting-container {
    padding-bottom: 100px;
  }

  .t-radio-group.t-size-m {
    align-items: center;
    justify-content: space-between;
    min-height: 32px;
    width: 100%;

    &.side-mode-radio {
      justify-content: end;
    }
  }

  .t-radio-group.t-size-m .t-radio-button {
    height: auto;
  }

  .setting-layout-drawer {
    align-items: center;
    display: flex;
    flex-direction: column;
    margin-bottom: 16px;

    .t-radio-button {
      border: 2px solid var(--td-component-border);
      border-radius: var(--td-radius-default);
      display: inline-flex;
      max-height: 78px;
      padding: 8px;

      > .t-radio-button__label {
        display: inline-flex;
      }
    }

    .t-is-checked {
      border: 2px solid var(--td-brand-color) !important;
    }

    .t-form__controls-content {
      justify-content: end;
    }
  }

  .t-form__controls-content {
    justify-content: end;
  }
}

.setting-route-theme {
  .t-form__label {
    color: var(--td-text-color-secondary);
    min-width: 310px !important;
  }
}

.setting-color-theme {
  .setting-layout-drawer {
    .t-radio-button {
      height: 32px;
    }

    &:last-child {
      margin-right: 0;
    }
  }
}
</style>
