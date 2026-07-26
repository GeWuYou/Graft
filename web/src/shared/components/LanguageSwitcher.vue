<template>
  <t-dropdown trigger="click">
    <t-tooltip :content="t('layout.header.language')" placement="bottom">
      <t-button :aria-label="t('layout.header.language')" theme="default" shape="square" variant="text">
        <translate-icon />
      </t-button>
    </t-tooltip>
    <t-dropdown-menu>
      <t-dropdown-item
        v-for="(lang, index) in languageList"
        :key="index"
        :value="lang.value"
        @click="(options) => changeLang(options.value as string)"
        >{{ lang.content }}</t-dropdown-item
      ></t-dropdown-menu
    >
  </t-dropdown>
</template>
<script setup lang="ts">
import { TranslateIcon } from 'tdesign-icons-vue-next';

import { languageList, t } from '@/locales';
import { useLocale } from '@/locales/useLocale';

const { changeLocale } = useLocale();

const changeLang = (lang: string) => {
  changeLocale(lang);
};
</script>
