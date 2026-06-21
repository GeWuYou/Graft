<template>
  <result
    :title="t('app.result.browserIncompatible.title')"
    :tip="t('app.result.browserIncompatible.subtitle')"
    type="ie"
  >
    <div class="browser-incompatible-actions">
      <t-button @click="() => router.push(ROOT_ENTRY_PATH)">
        {{ t('app.result.browserIncompatible.back') }}
      </t-button>
      <section class="browser-recommendations" :aria-label="t('app.result.browserIncompatible.recommend')">
        <div class="browser-recommendations__title">{{ t('app.result.browserIncompatible.recommend') }}</div>
        <div class="browser-recommendations__list">
          <div class="browser-recommendations__item">
            <thumbnail-image class="browser-recommendations__icon" :src="chromeIcon" />
            <span>{{ t('app.result.browserIncompatible.chrome') }}</span>
          </div>
          <div class="browser-recommendations__item">
            <thumbnail-image class="browser-recommendations__icon" :src="qqBrowserIcon" />
            <span>{{ t('app.result.browserIncompatible.qqBrowser') }}</span>
          </div>
        </div>
      </section>
    </div>
  </result>
</template>
<script setup lang="ts">
import { useRouter } from 'vue-router';

import chromeIcon from '@/assets/browser-chrome.svg';
import qqBrowserIcon from '@/assets/browser-qq.svg';
import { ROOT_ENTRY_PATH } from '@/contracts/app/routes';
import { t } from '@/locales';
import Result from '@/shared/components/ResultView.vue';
import ThumbnailImage from '@/shared/components/ThumbnailImage.vue';

defineOptions({
  name: 'ResultBrowserIncompatible',
});

const router = useRouter();
</script>
<style lang="less" scoped>
.browser-incompatible-actions {
  align-items: center;
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xl);
  width: min(640px, 100%);
}

.browser-recommendations {
  align-items: center;
  background: var(--td-bg-color-container);
  border-radius: var(--td-radius-medium);
  box-shadow: var(--td-shadow-1);
  color: var(--td-text-color-secondary);
  display: flex;
  gap: var(--td-comp-margin-xl);
  justify-content: space-between;
  padding: var(--td-comp-paddingTB-xl) var(--td-comp-paddingLR-xxl);
  width: 100%;

  &__title {
    color: var(--td-text-color-primary);
    font: var(--td-font-body-medium);
  }

  &__list {
    display: flex;
    gap: var(--td-comp-margin-xxxl);
  }

  &__item {
    align-items: center;
    display: flex;
    flex-direction: column;
    font: var(--td-font-body-small);
    gap: var(--td-comp-margin-xs);
    min-width: 88px;
  }

  &__icon {
    height: var(--td-comp-size-xl);
    width: var(--td-comp-size-xl);
  }
}

@media (width <= 640px) {
  .browser-recommendations {
    align-items: stretch;
    flex-direction: column;

    &__list {
      justify-content: space-between;
    }
  }
}
</style>
