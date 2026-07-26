<template>
  <t-dropdown v-if="mode === 'dropdown'" trigger="click">
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
  <template v-else>
    <t-tooltip v-if="showTrigger" :content="t('layout.header.language')" placement="bottom">
      <t-button :aria-label="t('layout.header.language')" theme="default" shape="square" variant="text" @click="open">
        <translate-icon />
      </t-button>
    </t-tooltip>
    <t-dialog
      v-model:visible="dialogVisible"
      attach="body"
      :footer="false"
      :header="t('layout.header.language')"
      width="360px"
    >
      <t-select
        v-model="selectedLanguage"
        data-testid="language-dialog-select"
        filterable
        :options="languageOptions"
        :placeholder="t('layout.header.language')"
      />
    </t-dialog>
  </template>
</template>
<script setup lang="ts">
import { TranslateIcon } from 'tdesign-icons-vue-next';
import { computed, ref } from 'vue';

import { languageList, t } from '@/locales';
import { useLocale } from '@/locales/useLocale';

const { changeLocale, locale } = useLocale();
const { mode = 'dropdown', showTrigger = true } = defineProps<{
  mode?: 'dropdown' | 'dialog';
  showTrigger?: boolean;
}>();
const dialogVisible = ref(false);
const languageOptions = computed(() =>
  languageList.value.map((lang) => ({ label: lang.content, value: String(lang.value) })),
);
const selectedLanguage = computed({
  get: () => String(locale.value),
  set: (value: string) => changeLang(value),
});

const changeLang = (lang: string) => {
  changeLocale(lang);
  dialogVisible.value = false;
};

const open = () => {
  dialogVisible.value = true;
};

defineExpose({ open });
</script>
