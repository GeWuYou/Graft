<template>
  <div ref="rootRef" class="header-menu-search-left" :class="{ 'is-open': isSearchFocus }">
    <t-tooltip :content="t('global.search.trigger')" placement="bottom" :disabled="isSearchFocus">
      <t-button
        class="header-menu-search-left__button"
        :class="{ 'search-icon-hide': isSearchFocus }"
        theme="default"
        shape="square"
        variant="text"
        :aria-label="t('global.search.trigger')"
        @click="changeSearchFocus(true)"
      >
        <t-icon name="search" />
      </t-button>
    </t-tooltip>
    <t-input
      ref="searchInputRef"
      v-model="keyword"
      class="header-search"
      :class="{ 'width-zero': !isSearchFocus }"
      clearable
      :placeholder="t('global.search.placeholder')"
      :autofocus="isSearchFocus"
      @focus="changeSearchFocus(true)"
      @keydown="handleKeydown"
      @enter="handleEnter"
    >
      <template #prefix-icon>
        <t-icon name="search" size="16" />
      </template>
    </t-input>

    <teleport to="body">
      <div v-if="showPanel" ref="panelRef" class="global-menu-search__panel-layer" :style="panelStyle">
        <div class="global-menu-search__panel">
          <div v-if="!routesInitialized" class="global-menu-search__state">
            <t-loading :loading="true" size="small" />
          </div>

          <div v-else-if="!searchIndex.length" class="global-menu-search__state">
            {{ t('global.search.empty') }}
          </div>

          <div v-else-if="!normalizedKeyword" class="global-menu-search__state">
            {{ t('global.search.idle') }}
          </div>

          <div v-else-if="visibleResults.length === 0" class="global-menu-search__state">
            {{ t('global.search.noResults') }}
          </div>

          <button
            v-for="(item, index) in visibleResults"
            v-else
            :key="item.key"
            type="button"
            class="global-menu-search-result"
            :class="{ 'is-active': index === activeIndex }"
            @mouseenter="activeIndex = index"
            @mousedown.prevent
            @click="navigateToItem(item)"
          >
            <div class="global-menu-search-result__title">
              <span>{{ item.title }}</span>
            </div>

            <div v-if="item.parentTitles.length" class="global-menu-search-result__parents">
              <span>{{ item.parentTitles.join(' / ') }}</span>
            </div>

            <t-tooltip v-if="item.path" :content="item.path" placement="bottom-left">
              <div class="global-menu-search-result__path">
                <span>{{ item.path }}</span>
              </div>
            </t-tooltip>
          </button>
        </div>
      </div>
    </teleport>
  </div>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { isNavigationFailure, NavigationFailureType, useRoute, useRouter } from 'vue-router';

import { t } from '@/locales';
import {
  type GlobalMenuSearchItem,
  normalizeGlobalMenuSearchKeyword,
  useGlobalMenuSearch,
} from '@/shared/composables/useGlobalMenuSearch';

const router = useRouter();
const route = useRoute();
const { routesInitialized, searchIndex, searchItems } = useGlobalMenuSearch();

const rootRef = ref<HTMLElement | null>(null);
const searchInputRef = ref<{ focus?: () => void } | null>(null);
const panelRef = ref<HTMLElement | null>(null);
const isSearchFocus = ref(false);
const keyword = ref('');
const activeIndex = ref(0);
const panelStyle = ref<Record<string, string>>({});

const normalizedKeyword = computed(() => normalizeGlobalMenuSearchKeyword(keyword.value));
const visibleResults = computed(() => searchItems(keyword.value));
const showPanel = computed(() => isSearchFocus.value);

watch(visibleResults, (results) => {
  if (results.length === 0) {
    activeIndex.value = 0;
    return;
  }

  activeIndex.value = Math.min(activeIndex.value, results.length - 1);
});

watch(showPanel, async (visible) => {
  if (!visible) {
    keyword.value = '';
    activeIndex.value = 0;
    return;
  }

  await nextTick();
  updatePanelPosition();
  searchInputRef.value?.focus?.();
});

function changeSearchFocus(value: boolean) {
  if (!value) {
    keyword.value = '';
    activeIndex.value = 0;
  }

  isSearchFocus.value = value;

  if (value) {
    nextTick(() => {
      updatePanelPosition();
      searchInputRef.value?.focus?.();
    });
  }
}

function handleKeydown(_: string | number, context: { e: KeyboardEvent }) {
  if (context.e.key === 'Escape') {
    context.e.preventDefault();
    changeSearchFocus(false);
    return;
  }

  if (visibleResults.value.length === 0) {
    return;
  }

  if (context.e.key === 'ArrowDown') {
    context.e.preventDefault();
    activeIndex.value = (activeIndex.value + 1) % visibleResults.value.length;
    return;
  }

  if (context.e.key === 'ArrowUp') {
    context.e.preventDefault();
    activeIndex.value = (activeIndex.value - 1 + visibleResults.value.length) % visibleResults.value.length;
  }
}

function handleEnter() {
  const item = visibleResults.value[activeIndex.value];
  if (!item) {
    return;
  }

  void navigateToItem(item);
}

async function navigateToItem(item: GlobalMenuSearchItem) {
  changeSearchFocus(false);

  try {
    if (route.path === item.navigationPath) {
      return;
    }

    await router.push(item.navigationPath);
  } catch (error) {
    if (isNavigationFailure(error, NavigationFailureType.duplicated)) {
      return;
    }

    throw error;
  }
}

function handleDocumentPointerDown(event: PointerEvent) {
  if (!showPanel.value) {
    return;
  }

  const target = event.target;
  if (!(target instanceof Node)) {
    return;
  }

  if (rootRef.value?.contains(target) || panelRef.value?.contains(target)) {
    return;
  }

  changeSearchFocus(false);
}

function updatePanelPosition() {
  const rootElement = rootRef.value;
  if (!rootElement || typeof window === 'undefined') {
    return;
  }

  const inputWrap = rootElement.querySelector('.header-search') as HTMLElement | null;
  const anchorRect = (inputWrap ?? rootElement).getBoundingClientRect();
  const viewportPadding = 16;
  const width = Math.min(Math.max(280, anchorRect.width), window.innerWidth - viewportPadding * 2);
  const maxLeft = Math.max(viewportPadding, window.innerWidth - width - viewportPadding);
  const left = Math.min(Math.max(viewportPadding, anchorRect.right - width), maxLeft);

  panelStyle.value = {
    left: `${left}px`,
    top: `${Math.max(viewportPadding, anchorRect.bottom + 8)}px`,
    width: `${width}px`,
  };
}

function handleViewportChange() {
  if (!showPanel.value) {
    return;
  }

  updatePanelPosition();
}

onMounted(() => {
  document.addEventListener('pointerdown', handleDocumentPointerDown, true);
  window.addEventListener('resize', handleViewportChange);
  window.addEventListener('scroll', handleViewportChange, true);
});

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleDocumentPointerDown, true);
  window.removeEventListener('resize', handleViewportChange);
  window.removeEventListener('scroll', handleViewportChange, true);
});
</script>
<style lang="less" scoped>
.header-menu-search-left {
  align-items: center;
  display: flex;
  flex: 0 0 auto;
  line-height: normal;

  .header-search {
    opacity: 1;
    transition:
      opacity @anim-duration-base @anim-time-fn-easing,
      width @anim-duration-base @anim-time-fn-easing;
    width: 200px;

    :deep(.t-input) {
      transition:
        background @anim-duration-base linear,
        border-color @anim-duration-base linear,
        box-shadow @anim-duration-base linear;
    }

    &.width-zero {
      opacity: 0;
      width: 0;
    }
  }
}

.header-menu-search-left__button {
  margin: 0;
  transition: opacity @anim-duration-base @anim-time-fn-easing;

  :deep(.t-icon) {
    font-size: var(--td-font-size-title-medium);
  }
}

.search-icon-hide {
  opacity: 0;
  pointer-events: none;
}

.global-menu-search__panel-layer {
  position: fixed;
  z-index: 1200;
}

.global-menu-search__panel {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  border-radius: var(--td-radius-large);
  box-shadow: var(--td-shadow-3);
  max-height: min(60vh, 560px);
  overflow: auto;
  padding: var(--graft-density-gap-8);
  scrollbar-color: var(--td-scrollbar-color) transparent;
  scrollbar-width: thin;
  width: 100%;
}

.global-menu-search__panel::-webkit-scrollbar {
  background: transparent;
  width: 8px;
}

.global-menu-search__panel::-webkit-scrollbar-track {
  background: transparent;
}

.global-menu-search__panel::-webkit-scrollbar-thumb {
  background-clip: content-box;
  background-color: var(--td-scrollbar-color);
  border: 2px solid transparent;
  border-radius: 6px;
}

.global-menu-search__state {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  justify-content: center;
  min-height: 104px;
  padding: var(--graft-density-gap-24);
  text-align: center;
}

.global-menu-search-result {
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-medium);
  color: inherit;
  cursor: pointer;
  display: block;
  padding: var(--graft-density-gap-12);
  text-align: left;
  width: 100%;

  & + & {
    margin-top: var(--graft-density-gap-8);
  }

  &:hover,
  &.is-active {
    background: var(--td-bg-color-container-hover);
    box-shadow: inset 0 0 0 1px var(--td-brand-color-light);
  }
}

.global-menu-search-result__title,
.global-menu-search-result__parents,
.global-menu-search-result__path {
  min-width: 0;

  span {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.global-menu-search-result__title {
  color: var(--td-text-color-primary);
  font-size: var(--td-font-size-title-small);
  font-weight: 600;
  line-height: 22px;
}

.global-menu-search-result__parents {
  color: var(--td-text-color-secondary);
  font-size: var(--td-font-size-body-small);
  line-height: 20px;
  margin-top: var(--graft-density-gap-4);
}

.global-menu-search-result__path {
  color: var(--td-text-color-placeholder);
  font-family: SFMono-Regular, 'JetBrains Mono', 'Cascadia Code', monospace;
  font-size: var(--td-font-size-body-small);
  line-height: 18px;
  margin-top: var(--graft-density-gap-4);
}
</style>
