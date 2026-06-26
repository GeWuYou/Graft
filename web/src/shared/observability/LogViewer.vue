<template>
  <content-viewer-frame
    v-if="viewerMode"
    class="log-viewer log-viewer--framed"
    :storage-key="viewerStorageKey"
    :fullscreen-label="fullscreenLabel"
    :exit-fullscreen-label="exitFullscreenLabel"
    :resize-handle-label="resizeHandleLabel"
    surface-padding="none"
    fullscreen-surface-padding="none"
  >
    <template #toolbar>
      <div class="log-viewer__toolbar">
        <div class="log-viewer__toolbar-group log-viewer__toolbar-left">
          <t-button
            theme="default"
            variant="outline"
            size="small"
            :disabled="!displayLines.length"
            data-testid="log-viewer-clear"
            @click="$emit('clear')"
          >
            {{ clearLabel }}
          </t-button>
          <t-button
            theme="default"
            variant="outline"
            size="small"
            :disabled="!displayLines.length"
            data-testid="log-viewer-copy"
            @click="copyContent"
          >
            {{ copyLabel }}
          </t-button>
          <t-button
            theme="default"
            variant="outline"
            size="small"
            :disabled="!displayLines.length"
            data-testid="log-viewer-download"
            @click="downloadContent"
          >
            {{ downloadLabel }}
          </t-button>
        </div>

        <div class="log-viewer__toolbar-group log-viewer__toolbar-middle">
          <t-select
            v-if="lineLimitOptions.length"
            v-model:value="selectedLineLimit"
            class="log-viewer__limit"
            :options="lineLimitOptions"
            size="small"
            @change="emitLimit"
          />
          <t-select
            v-model:value="selectedLevel"
            class="log-viewer__level-filter"
            :options="levelOptions"
            size="small"
          />
          <t-input
            v-model:value="searchKeyword"
            class="log-viewer__search"
            clearable
            type="search"
            :placeholder="searchPlaceholder"
          />
          <span v-if="normalizedSearchKeyword" class="log-viewer__match-count">
            {{ matchCountLabel.replace('{count}', String(searchMatchCount)) }}
          </span>
        </div>

        <div class="log-viewer__toolbar-group log-viewer__toolbar-right">
          <label class="log-viewer__switch">
            <span>{{ wrapLabel }}</span>
            <t-switch v-model:value="wrapLines" size="small" />
          </label>
          <label class="log-viewer__switch">
            <span>{{ autoScrollLabel }}</span>
            <t-tooltip v-if="autoScrollTooltipLabel" :content="autoScrollTooltipLabel" theme="light">
              <t-switch v-model:value="scrollAfterRefresh" size="small" />
            </t-tooltip>
            <t-switch v-else v-model:value="scrollAfterRefresh" size="small" />
          </label>
          <t-button
            size="small"
            theme="default"
            variant="outline"
            data-testid="log-viewer-pause-toggle"
            @click="togglePause"
          >
            {{ paused ? resumeLabel : pauseLabel }}
          </t-button>
          <t-button
            v-if="showReconnect"
            size="small"
            theme="primary"
            variant="outline"
            data-testid="log-viewer-reconnect"
            @click="$emit('reconnect')"
          >
            {{ reconnectLabel }}
          </t-button>
        </div>
      </div>
    </template>

    <template #default>
      <div class="log-viewer__body">
        <t-alert v-if="error" theme="error" :title="error">
          <template #operation>
            <t-button size="small" theme="danger" variant="text" @click="$emit('refresh')">
              {{ retryLabel }}
            </t-button>
          </template>
        </t-alert>
        <t-alert v-if="truncated" theme="warning" :title="truncatedLabel" />

        <div
          ref="viewport"
          :class="['log-viewer__viewport graft-scrollbar', { 'log-viewer__viewport--wrap': wrapLines }]"
          @scroll="handleViewportScroll"
        >
          <stream-viewport-state-surface
            v-if="shouldRenderViewportStateSurface"
            class="log-viewer__viewport-state"
            v-bind="viewportStateSurfaceModel"
          />
          <template v-else>
            <div class="log-viewer__header-row">
              <span class="log-viewer__header-cell">{{ timeLabel }}</span>
              <span class="log-viewer__header-cell">{{ levelLabel }}</span>
              <span class="log-viewer__header-cell">{{ streamLabel }}</span>
              <span class="log-viewer__header-cell">{{ messageLabel }}</span>
              <span class="log-viewer__header-cell log-viewer__header-cell--actions">{{ operationLabel }}</span>
            </div>
            <ol class="log-viewer__lines" :style="virtualListStyle">
              <li
                v-for="(line, lineIndex) in renderedLines"
                :key="line.lineNo"
                :ref="(element) => setRenderedLineRef(line.lineNo, element)"
                tabindex="0"
                :class="[
                  'log-viewer__line',
                  `log-viewer__line--${line.tone}`,
                  { 'log-viewer__line--active': isActive(line.lineNo) },
                ]"
                :style="virtualLineStyle(lineIndex)"
                @click="openLineDetail(line)"
                @keydown.enter.prevent="openLineDetail(line)"
                @keydown.space.prevent="openLineDetail(line)"
              >
                <div class="log-viewer__timestamp-cell">
                  <t-tooltip
                    v-if="line.timestamp"
                    :content="formattedFullTimestamp(line.timestamp)"
                    placement="top-left"
                    theme="light"
                  >
                    <time class="log-viewer__timestamp">{{ displayTimestamp(line.timestamp) }}</time>
                  </t-tooltip>
                  <span v-else class="log-viewer__timestamp log-viewer__timestamp--empty"></span>
                </div>
                <div class="log-viewer__level-cell">
                  <t-tag class="log-viewer__level" :theme="levelTheme(line.level)" size="small" variant="light-outline">
                    {{ line.level ?? 'LOG' }}
                  </t-tag>
                </div>
                <div class="log-viewer__stream-cell">
                  <span class="log-viewer__stream-pill" :class="`log-viewer__stream-pill--${line.stream}`">
                    {{ line.stream === 'stderr' ? stderrLabel : stdoutLabel }}
                  </span>
                </div>
                <div class="log-viewer__content">
                  <div class="log-viewer__message-row">
                    <code class="log-viewer__message">
                      <span
                        v-for="(token, tokenIndex) in line.messageTokens"
                        :key="`${line.lineNo}-message-${tokenIndex}`"
                        :class="tokenClass(token)"
                        >{{ token.text }}</span
                      >
                    </code>
                  </div>
                  <div v-if="visibleMetadataTags(line).length" class="log-viewer__metadata-tags" @click.stop>
                    <t-tag
                      v-for="[key, value] in visibleMetadataTags(line)"
                      :key="`${line.lineNo}-${key}`"
                      size="small"
                      theme="default"
                      variant="light"
                    >
                      {{ key }}={{ formatMetadataValue(value) }}
                    </t-tag>
                    <t-button
                      v-if="hiddenRowFieldCount(line)"
                      size="small"
                      theme="default"
                      variant="text"
                      @click="openLineDetail(line)"
                    >
                      +{{ hiddenRowFieldCount(line) }}
                    </t-button>
                  </div>
                </div>
                <div class="log-viewer__row-actions" @click.stop>
                  <t-tooltip :content="viewDetailLabel" theme="light">
                    <t-button
                      :aria-label="viewDetailLabel"
                      class="log-viewer__icon-action"
                      shape="square"
                      size="small"
                      theme="default"
                      variant="text"
                      @click="openLineDetail(line)"
                    >
                      <template #icon>
                        <browse-icon />
                      </template>
                    </t-button>
                  </t-tooltip>
                  <t-tooltip :content="copyLineLabel" theme="light">
                    <t-button
                      :aria-label="copyLineLabel"
                      class="log-viewer__icon-action"
                      shape="square"
                      size="small"
                      theme="default"
                      variant="text"
                      @click="copyLine(line.raw)"
                    >
                      <template #icon>
                        <copy-icon />
                      </template>
                    </t-button>
                  </t-tooltip>
                </div>
              </li>
            </ol>
          </template>
          <div v-if="showJumpBottom" class="log-viewer__jump-bottom-wrap">
            <t-button size="small" theme="primary" @click="jumpBottom">
              {{ jumpBottomLabel }}
            </t-button>
          </div>
        </div>
      </div>
    </template>
  </content-viewer-frame>

  <section v-else class="log-viewer">
    <div class="log-viewer__toolbar">
      <div class="log-viewer__toolbar-group log-viewer__toolbar-left">
        <t-button
          theme="default"
          variant="outline"
          size="small"
          :disabled="!displayLines.length"
          data-testid="log-viewer-clear"
          @click="$emit('clear')"
        >
          {{ clearLabel }}
        </t-button>
        <t-button
          theme="default"
          variant="outline"
          size="small"
          :disabled="!displayLines.length"
          data-testid="log-viewer-copy"
          @click="copyContent"
        >
          {{ copyLabel }}
        </t-button>
        <t-button
          theme="default"
          variant="outline"
          size="small"
          :disabled="!displayLines.length"
          data-testid="log-viewer-download"
          @click="downloadContent"
        >
          {{ downloadLabel }}
        </t-button>
      </div>

      <div class="log-viewer__toolbar-group log-viewer__toolbar-middle">
        <t-select
          v-if="lineLimitOptions.length"
          v-model:value="selectedLineLimit"
          class="log-viewer__limit"
          :options="lineLimitOptions"
          size="small"
          @change="emitLimit"
        />
        <t-select v-model:value="selectedLevel" class="log-viewer__level-filter" :options="levelOptions" size="small" />
        <t-input
          v-model:value="searchKeyword"
          class="log-viewer__search"
          clearable
          type="search"
          :placeholder="searchPlaceholder"
        />
        <span v-if="normalizedSearchKeyword" class="log-viewer__match-count">
          {{ matchCountLabel.replace('{count}', String(searchMatchCount)) }}
        </span>
      </div>

      <div class="log-viewer__toolbar-group log-viewer__toolbar-right">
        <label class="log-viewer__switch">
          <span>{{ wrapLabel }}</span>
          <t-switch v-model:value="wrapLines" size="small" />
        </label>
        <label class="log-viewer__switch">
          <span>{{ autoScrollLabel }}</span>
          <t-tooltip v-if="autoScrollTooltipLabel" :content="autoScrollTooltipLabel" theme="light">
            <t-switch v-model:value="scrollAfterRefresh" size="small" />
          </t-tooltip>
          <t-switch v-else v-model:value="scrollAfterRefresh" size="small" />
        </label>
        <t-button
          size="small"
          theme="default"
          variant="outline"
          data-testid="log-viewer-pause-toggle"
          @click="togglePause"
        >
          {{ paused ? resumeLabel : pauseLabel }}
        </t-button>
        <t-button
          v-if="showReconnect"
          size="small"
          theme="primary"
          variant="outline"
          data-testid="log-viewer-reconnect"
          @click="$emit('reconnect')"
        >
          {{ reconnectLabel }}
        </t-button>
      </div>
    </div>

    <t-alert v-if="error" theme="error" :title="error">
      <template #operation>
        <t-button size="small" theme="danger" variant="text" @click="$emit('refresh')">
          {{ retryLabel }}
        </t-button>
      </template>
    </t-alert>
    <t-alert v-if="truncated" theme="warning" :title="truncatedLabel" />

    <div
      ref="viewport"
      :class="['log-viewer__viewport graft-scrollbar', { 'log-viewer__viewport--wrap': wrapLines }]"
      @scroll="handleViewportScroll"
    >
      <stream-viewport-state-surface
        v-if="shouldRenderViewportStateSurface"
        class="log-viewer__viewport-state"
        v-bind="viewportStateSurfaceModel"
      />
      <template v-else>
        <div class="log-viewer__header-row">
          <span class="log-viewer__header-cell">{{ timeLabel }}</span>
          <span class="log-viewer__header-cell">{{ levelLabel }}</span>
          <span class="log-viewer__header-cell">{{ streamLabel }}</span>
          <span class="log-viewer__header-cell">{{ messageLabel }}</span>
          <span class="log-viewer__header-cell log-viewer__header-cell--actions">{{ operationLabel }}</span>
        </div>
        <ol class="log-viewer__lines" :style="virtualListStyle">
          <li
            v-for="(line, lineIndex) in renderedLines"
            :key="line.lineNo"
            :ref="(element) => setRenderedLineRef(line.lineNo, element)"
            tabindex="0"
            :class="[
              'log-viewer__line',
              `log-viewer__line--${line.tone}`,
              { 'log-viewer__line--active': isActive(line.lineNo) },
            ]"
            :style="virtualLineStyle(lineIndex)"
            @click="openLineDetail(line)"
            @keydown.enter.prevent="openLineDetail(line)"
            @keydown.space.prevent="openLineDetail(line)"
          >
            <div class="log-viewer__timestamp-cell">
              <t-tooltip
                v-if="line.timestamp"
                :content="formattedFullTimestamp(line.timestamp)"
                placement="top-left"
                theme="light"
              >
                <time class="log-viewer__timestamp">{{ displayTimestamp(line.timestamp) }}</time>
              </t-tooltip>
              <span v-else class="log-viewer__timestamp log-viewer__timestamp--empty"></span>
            </div>
            <div class="log-viewer__level-cell">
              <t-tag class="log-viewer__level" :theme="levelTheme(line.level)" size="small" variant="light-outline">
                {{ line.level ?? 'LOG' }}
              </t-tag>
            </div>
            <div class="log-viewer__stream-cell">
              <span class="log-viewer__stream-pill" :class="`log-viewer__stream-pill--${line.stream}`">
                {{ line.stream === 'stderr' ? stderrLabel : stdoutLabel }}
              </span>
            </div>
            <div class="log-viewer__content">
              <div class="log-viewer__message-row">
                <code class="log-viewer__message">
                  <span
                    v-for="(token, tokenIndex) in line.messageTokens"
                    :key="`${line.lineNo}-message-${tokenIndex}`"
                    :class="tokenClass(token)"
                    >{{ token.text }}</span
                  >
                </code>
              </div>
              <div v-if="visibleMetadataTags(line).length" class="log-viewer__metadata-tags" @click.stop>
                <t-tag
                  v-for="[key, value] in visibleMetadataTags(line)"
                  :key="`${line.lineNo}-${key}`"
                  size="small"
                  theme="default"
                  variant="light"
                >
                  {{ key }}={{ formatMetadataValue(value) }}
                </t-tag>
                <t-button
                  v-if="hiddenRowFieldCount(line)"
                  size="small"
                  theme="default"
                  variant="text"
                  @click="openLineDetail(line)"
                >
                  +{{ hiddenRowFieldCount(line) }}
                </t-button>
              </div>
            </div>
            <div class="log-viewer__row-actions" @click.stop>
              <t-tooltip :content="viewDetailLabel" theme="light">
                <t-button
                  :aria-label="viewDetailLabel"
                  class="log-viewer__icon-action"
                  shape="square"
                  size="small"
                  theme="default"
                  variant="text"
                  @click="openLineDetail(line)"
                >
                  <template #icon>
                    <browse-icon />
                  </template>
                </t-button>
              </t-tooltip>
              <t-tooltip :content="copyLineLabel" theme="light">
                <t-button
                  :aria-label="copyLineLabel"
                  class="log-viewer__icon-action"
                  shape="square"
                  size="small"
                  theme="default"
                  variant="text"
                  @click="copyLine(line.raw)"
                >
                  <template #icon>
                    <copy-icon />
                  </template>
                </t-button>
              </t-tooltip>
            </div>
          </li>
        </ol>
      </template>
      <div v-if="showJumpBottom" class="log-viewer__jump-bottom-wrap">
        <t-button size="small" theme="primary" @click="jumpBottom">
          {{ jumpBottomLabel }}
        </t-button>
      </div>
    </div>
  </section>

  <t-drawer
    v-model:visible="detailDrawerVisible"
    drawer-class-name="log-viewer__drawer"
    :footer="false"
    :header="detailTitleLabel"
    placement="right"
    size="min(600px, 100vw)"
    @close="closeLineDetail"
  >
    <div v-if="selectedLine" class="log-viewer__detail-drawer">
      <section class="log-viewer__summary">
        <div class="log-viewer__summary-main">
          <div class="log-viewer__summary-title">
            <t-tag
              class="log-viewer__summary-level"
              :theme="levelTheme(selectedLine.parsed.display.level)"
              size="small"
              variant="light-outline"
            >
              {{ selectedLine.parsed.display.level ?? 'LOG' }}
            </t-tag>
            <span
              class="log-viewer__stream-pill log-viewer__stream-pill--detail"
              :class="`log-viewer__stream-pill--${selectedLine.stream}`"
            >
              {{ selectedLine.stream === 'stderr' ? stderrLabel : stdoutLabel }}
            </span>
            <span class="log-viewer__summary-message">{{ selectedLine.parsed.display.title }}</span>
          </div>
          <div
            v-if="selectedLine.parsed.display.subtitleParts.length || selectedLine.timestamp"
            class="log-viewer__summary-meta"
          >
            <span v-if="selectedLine.timestamp">{{ formattedFullTimestamp(selectedLine.timestamp) }}</span>
            <template
              v-for="(part, partIndex) in selectedLine.parsed.display.subtitleParts"
              :key="`${selectedLine.lineNo}-summary-${partIndex}`"
            >
              <span aria-hidden="true">·</span>
              <t-tooltip
                v-if="part === selectedLine.source"
                :content="selectedLine.source"
                placement="top-left"
                theme="light"
              >
                <span class="log-viewer__summary-source">{{ part }}</span>
              </t-tooltip>
              <span v-else>{{ part }}</span>
            </template>
          </div>
        </div>
      </section>

      <section v-if="selectedLine.parsed.importantFields.length" class="log-viewer__drawer-section">
        <div class="log-viewer__drawer-section-title">{{ importantFieldsLabel }}</div>
        <div class="log-viewer__field-chips">
          <span v-for="field in selectedLine.parsed.importantFields" :key="field.key" class="log-viewer__field-chip">
            <span class="log-viewer__field-key">{{ field.key }}</span>
            <span class="log-viewer__field-equals">=</span>
            <t-tooltip :content="field.value" placement="top-left" theme="light">
              <span class="log-viewer__field-value">{{ field.value }}</span>
            </t-tooltip>
          </span>
        </div>
      </section>

      <section class="log-viewer__drawer-section">
        <div class="log-viewer__drawer-section-title">{{ basicInfoLabel }}</div>
        <div class="log-viewer__basic">
          <div class="log-viewer__descriptions">
            <template v-if="selectedLine.timestamp">
              <div class="log-viewer__description-label">{{ timeLabel }}</div>
              <div class="log-viewer__description-value">{{ formattedFullTimestamp(selectedLine.timestamp) }}</div>
            </template>

            <template v-if="selectedLine.level">
              <div class="log-viewer__description-label">{{ levelLabel }}</div>
              <div class="log-viewer__description-value log-viewer__level-value">
                <t-tag
                  class="log-viewer__detail-level"
                  :theme="levelTheme(selectedLine.level)"
                  size="small"
                  variant="light-outline"
                >
                  {{ selectedLine.level }}
                </t-tag>
              </div>
            </template>

            <div class="log-viewer__description-label">{{ streamLabel }}</div>
            <div class="log-viewer__description-value">
              <span class="log-viewer__stream-pill" :class="`log-viewer__stream-pill--${selectedLine.stream}`">
                {{ selectedLine.stream === 'stderr' ? stderrLabel : stdoutLabel }}
              </span>
            </div>

            <template v-if="selectedLine.source">
              <div class="log-viewer__description-label">{{ sourceLabel }}</div>
              <div class="log-viewer__description-value">{{ selectedLine.source }}</div>
            </template>

            <div class="log-viewer__description-label">{{ messageLabel }}</div>
            <div class="log-viewer__description-value">{{ selectedLine.message }}</div>
          </div>
        </div>
      </section>

      <section v-if="selectedLine.metadata" class="log-viewer__drawer-section">
        <div class="log-viewer__drawer-section-header">
          <div class="log-viewer__drawer-section-title">{{ metadataLabel }}</div>
          <t-button size="small" theme="default" variant="text" @click="copySelectedJson">
            {{ copyJsonLabel }}
          </t-button>
        </div>
        <pre class="log-viewer__code-block graft-scrollbar"><code>{{ formatJson(selectedLine.metadata) }}</code></pre>
      </section>

      <section class="log-viewer__drawer-section">
        <div class="log-viewer__drawer-section-header">
          <div class="log-viewer__drawer-section-title">{{ rawLabel }}</div>
          <t-button size="small" theme="default" variant="text" @click="copySelectedLine">
            {{ copyLineLabel }}
          </t-button>
        </div>
        <pre class="log-viewer__code-block log-viewer__code-block--raw graft-scrollbar">
          <code>{{ selectedLine.raw }}</code>
        </pre>
      </section>
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
import { BrowseIcon, CopyIcon } from 'tdesign-icons-vue-next';
import type { SelectProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import {
  type ComponentPublicInstance,
  computed,
  nextTick,
  onMounted,
  onUnmounted,
  ref,
  shallowRef,
  triggerRef,
  watch,
} from 'vue';
import { useI18n } from 'vue-i18n';

import ContentViewerFrame from '@/shared/components/viewer/ContentViewerFrame.vue';

import { copyText } from './copy';
import type { StructuredLogEntry } from './log-entry';
import type { LogLevel, LogToken } from './log-highlight';
import { type DisplayLogLine, formatLogMetadataValue, type ParsedLogMetadata, summarizeMetadata } from './log-parser';
import { LogViewCache } from './log-view-cache';
import type { StreamViewportState } from './stream-viewport-state';
import StreamViewportStateSurface from './StreamViewportStateSurface.vue';
import { formatLocaleDateTime, formatLogViewerTimestamp } from './time';

const props = withDefaults(
  defineProps<{
    entries: readonly StructuredLogEntry[];
    contentVersion?: number;
    loading?: boolean;
    error?: string;
    truncated?: boolean;
    lineLimit?: number;
    lineLimits?: number[];
    clearLabel: string;
    copyLabel: string;
    downloadLabel: string;
    retryLabel: string;
    searchPlaceholder: string;
    wrapLabel: string;
    autoScrollLabel: string;
    autoScrollTooltipLabel: string;
    pauseLabel: string;
    resumeLabel: string;
    reconnectLabel: string;
    jumpBottomLabel: string;
    levelFilterLabel: string;
    allLevelsLabel: string;
    matchCountLabel: string;
    emptyLabel: string;
    emptyDescriptionLabel?: string;
    truncatedLabel: string;
    detailTitleLabel: string;
    importantFieldsLabel: string;
    basicInfoLabel: string;
    timeLabel: string;
    levelLabel: string;
    streamLabel: string;
    sourceLabel: string;
    operationLabel: string;
    stdoutLabel: string;
    stderrLabel: string;
    viewDetailLabel: string;
    collapseDetailLabel: string;
    metadataLabel: string;
    messageLabel: string;
    rawLabel: string;
    copyMessageLabel: string;
    copyLineLabel: string;
    copyJsonLabel: string;
    copySuccessLabel: string;
    copyErrorLabel: string;
    paused?: boolean;
    showReconnect?: boolean;
    viewportState?: LogViewerViewportState | null;
    viewerMode?: boolean;
    viewerStorageKey?: string;
    fullscreenLabel?: string;
    exitFullscreenLabel?: string;
    resizeHandleLabel?: string;
  }>(),
  {
    contentVersion: undefined,
    loading: false,
    error: '',
    truncated: false,
    lineLimit: 200,
    lineLimits: () => [100, 200, 500, 1000],
    paused: false,
    showReconnect: false,
    viewportState: null,
    viewerMode: false,
    viewerStorageKey: 'graft.log-viewer.height',
    fullscreenLabel: 'Fullscreen',
    exitFullscreenLabel: 'Exit Fullscreen',
    resizeHandleLabel: 'Resize viewer',
    emptyDescriptionLabel: '',
  },
);

const emit = defineEmits<{
  refresh: [];
  clear: [];
  pause: [];
  resume: [];
  reconnect: [];
  'update:lineLimit': [value: number];
}>();

const { locale } = useI18n();

type SelectOption = NonNullable<SelectProps['options']>[number];
type LevelFilter = 'ALL' | LogLevel;
type VirtualMetrics = Readonly<{
  offsets: number[];
  ends: number[];
  totalHeight: number;
}>;
type LogViewerViewportState = Readonly<{
  state: StreamViewportState;
  ariaLabel?: string;
  badgeLabel?: string;
  busyLabel?: string;
  description?: string;
  fauxLineCount?: number;
  hint?: string;
  showBusy?: boolean | null;
  showCursor?: boolean | null;
  title?: string;
}>;

const DEFAULT_VIRTUAL_VIEWPORT_HEIGHT = 480;
const DEFAULT_LOG_ROW_HEIGHT = 52;
const DEFAULT_WRAPPED_LOG_ROW_HEIGHT = 88;
const DEFAULT_VIRTUAL_OVERSCAN_PX = 240;
const LOG_ROW_VERTICAL_MARGIN_PX = 2;
const AUTO_SCROLL_BOTTOM_THRESHOLD_PX = 32;

const searchKeyword = ref('');
const wrapLines = ref(true);
const scrollAfterRefresh = ref(true);
const selectedLineLimit = ref(props.lineLimit);
const selectedLevel = ref<LevelFilter>('ALL');
const viewport = ref<HTMLElement | null>(null);
const viewportScrollTop = ref(0);
const viewportHeight = ref(DEFAULT_VIRTUAL_VIEWPORT_HEIGHT);
const viewportPinnedToBottom = ref(true);
const hasAutoScrolledSinceLastEmpty = ref(false);
const selectedLineNo = ref<number | null>(null);
const measuredHeights = shallowRef(new Map<number, number>());
const logViewCache = new LogViewCache();
let scrollToBottomFrameId: number | null = null;

const levelOptions = computed<SelectOption[]>(() => [
  { label: `${props.levelFilterLabel}: ${props.allLevelsLabel}`, value: 'ALL' },
  { label: 'FATAL', value: 'FATAL' },
  { label: 'ERROR', value: 'ERROR' },
  { label: 'WARN', value: 'WARN' },
  { label: 'INFO', value: 'INFO' },
  { label: 'DEBUG', value: 'DEBUG' },
  { label: 'TRACE', value: 'TRACE' },
  { label: 'LOG', value: 'LOG' },
  { label: 'UNKNOWN', value: 'UNKNOWN' },
]);
const lineLimitOptions = computed<SelectOption[]>(() =>
  props.lineLimits.map((value) => ({ label: String(value), value })),
);
const effectiveContentVersion = computed(() => props.contentVersion ?? props.entries.length);
const normalizedSearchKeyword = computed(() => searchKeyword.value.trim());
const logView = computed(() =>
  logViewCache.buildView({
    entries: props.entries,
    lineLimit: selectedLineLimit.value,
    level: selectedLevel.value,
    keyword: normalizedSearchKeyword.value,
  }),
);
const displayLines = computed(() => logView.value.displayLines);
const defaultVirtualRowHeight = computed(() =>
  wrapLines.value ? DEFAULT_WRAPPED_LOG_ROW_HEIGHT : DEFAULT_LOG_ROW_HEIGHT,
);
const virtualMetrics = computed<VirtualMetrics>(() => {
  void measuredHeights.value;

  const offsets: number[] = [];
  const ends: number[] = [];
  let totalHeight = 0;

  for (const line of displayLines.value) {
    offsets.push(totalHeight);
    const rowHeight = measuredHeights.value.get(line.lineNo) ?? defaultVirtualRowHeight.value;
    totalHeight += rowHeight;
    ends.push(totalHeight);
  }

  return {
    offsets,
    ends,
    totalHeight,
  };
});
const visibleRange = computed(() => {
  const totalLines = displayLines.value.length;
  if (!totalLines) {
    return {
      start: 0,
      end: 0,
    };
  }

  const topBoundary = Math.max(0, viewportScrollTop.value - DEFAULT_VIRTUAL_OVERSCAN_PX);
  const bottomBoundary = viewportScrollTop.value + viewportHeight.value + DEFAULT_VIRTUAL_OVERSCAN_PX;
  const rawStart = findFirstEndAfter(virtualMetrics.value.ends, topBoundary);
  const start = Math.min(rawStart, Math.max(0, totalLines - 1));
  const end = Math.max(
    start + 1,
    Math.min(totalLines, findFirstOffsetAtOrAfter(virtualMetrics.value.offsets, bottomBoundary) + 1),
  );

  return {
    start,
    end,
  };
});
const renderedLines = computed(() => displayLines.value.slice(visibleRange.value.start, visibleRange.value.end));
const virtualListStyle = computed(() => ({
  height: `${virtualMetrics.value.totalHeight}px`,
}));
const searchMatchCount = computed(() => logView.value.matchCount);
const selectedLine = computed(() => displayLines.value.find((line) => line.lineNo === selectedLineNo.value) ?? null);
const shouldRenderViewportStateSurface = computed(() => !displayLines.value.length);
const viewportStateSurfaceModel = computed<LogViewerViewportState>(() => {
  if (props.viewportState) {
    return props.viewportState;
  }

  if (props.loading) {
    return {
      state: 'connecting',
      busyLabel: props.emptyDescriptionLabel || props.emptyLabel,
      description: props.emptyDescriptionLabel || '',
      title: props.emptyLabel,
    };
  }

  if (props.paused) {
    return {
      state: 'paused',
      description: props.emptyDescriptionLabel || props.emptyLabel,
      title: props.emptyLabel,
    };
  }

  return {
    state: 'empty',
    description: props.emptyDescriptionLabel || props.emptyLabel,
    title: props.emptyLabel,
  };
});
const detailDrawerVisible = computed({
  get: () => selectedLine.value !== null,
  set: (visible: boolean) => {
    if (!visible) {
      selectedLineNo.value = null;
    }
  },
});
const showJumpBottom = computed(() => Boolean(displayLines.value.length) && !viewportPinnedToBottom.value);

watch(
  () => props.lineLimit,
  (value) => {
    selectedLineLimit.value = value;
  },
);

watch(
  () => displayLines.value.length,
  (length) => {
    if (length === 0) {
      hasAutoScrolledSinceLastEmpty.value = false;
    }
  },
  { flush: 'post', immediate: true },
);

watch(
  () => [effectiveContentVersion.value, scrollAfterRefresh.value] as const,
  () => {
    if (!scrollAfterRefresh.value) return;
    if (!displayLines.value.length) return;
    if (!hasAutoScrolledSinceLastEmpty.value) {
      hasAutoScrolledSinceLastEmpty.value = true;
      scheduleScrollToBottom();
      return;
    }
    if (!viewportPinnedToBottom.value) return;
    scheduleScrollToBottom();
  },
  { flush: 'post', immediate: true },
);

watch(
  () => [props.entries, effectiveContentVersion.value, wrapLines.value] as const,
  () => {
    clearMeasuredHeights();
    void nextTick(syncViewportMetrics);
  },
  { flush: 'post' },
);

onMounted(() => {
  void nextTick(syncViewportMetrics);
  window.addEventListener('resize', syncViewportMetrics);
});

onUnmounted(() => {
  window.removeEventListener('resize', syncViewportMetrics);
  if (scrollToBottomFrameId !== null) {
    cancelAnimationFrame(scrollToBottomFrameId);
    scrollToBottomFrameId = null;
  }
});

function emitLimit(value: SelectProps['value']) {
  if (typeof value === 'number') {
    emit('update:lineLimit', value);
  }
}

async function copyContent() {
  await copyTextWithFeedback(displayLines.value.map((line) => line.raw).join('\n'));
}

async function copyLine(raw: string) {
  await copyTextWithFeedback(raw);
}

async function copyJson(metadata: ParsedLogMetadata) {
  await copyTextWithFeedback(formatJson(metadata));
}

function downloadContent() {
  const blob = new Blob([displayLines.value.map((line) => line.raw).join('\n')], { type: 'text/plain;charset=utf-8' });
  const link = document.createElement('a');
  link.href = URL.createObjectURL(blob);
  link.download = `container-logs-${new Date().toISOString().replace(/[:.]/g, '-')}.log`;
  link.click();
  URL.revokeObjectURL(link.href);
}

function togglePause() {
  if (props.paused) {
    emit('resume');
    return;
  }

  emit('pause');
}

function jumpBottom() {
  scrollToBottom();
}

function scrollToBottom() {
  const node = viewport.value;
  if (node) {
    node.scrollTop = node.scrollHeight;
    viewportScrollTop.value = node.scrollTop;
    viewportHeight.value = node.clientHeight || DEFAULT_VIRTUAL_VIEWPORT_HEIGHT;
    viewportPinnedToBottom.value = true;
  }
}

function handleViewportScroll(event: Event) {
  const node = event.target;
  if (!(node instanceof HTMLElement)) {
    return;
  }

  viewportScrollTop.value = node.scrollTop;
  viewportHeight.value = node.clientHeight || DEFAULT_VIRTUAL_VIEWPORT_HEIGHT;
  viewportPinnedToBottom.value = isViewportNearBottom(node);
}

function syncViewportMetrics() {
  const node = viewport.value;
  if (!node) {
    return;
  }

  viewportScrollTop.value = node.scrollTop;
  viewportHeight.value = node.clientHeight || DEFAULT_VIRTUAL_VIEWPORT_HEIGHT;
  viewportPinnedToBottom.value = isViewportNearBottom(node);
}

function virtualLineStyle(renderedIndex: number) {
  const index = visibleRange.value.start + renderedIndex;
  return {
    transform: `translateY(${virtualMetrics.value.offsets[index] ?? 0}px)`,
  };
}

function setRenderedLineRef(lineNo: number, element: Element | ComponentPublicInstance | null) {
  if (!(element instanceof HTMLElement)) {
    return;
  }

  const measuredHeight =
    element.getBoundingClientRect().height ||
    element.offsetHeight ||
    element.clientHeight ||
    defaultVirtualRowHeight.value;
  const nextHeight = Math.max(defaultVirtualRowHeight.value, Math.ceil(measuredHeight) + LOG_ROW_VERTICAL_MARGIN_PX);
  const currentHeight = measuredHeights.value.get(lineNo);
  if (currentHeight === nextHeight) {
    return;
  }

  measuredHeights.value.set(lineNo, nextHeight);
  triggerRef(measuredHeights);
}

function clearMeasuredHeights() {
  measuredHeights.value.clear();
  triggerRef(measuredHeights);
}

function scheduleScrollToBottom() {
  if (scrollToBottomFrameId !== null) {
    return;
  }

  scrollToBottomFrameId = requestAnimationFrame(() => {
    scrollToBottomFrameId = null;
    scrollToBottom();
  });
}

function openLineDetail(line: DisplayLogLine) {
  selectedLineNo.value = line.lineNo;
}

function closeLineDetail() {
  selectedLineNo.value = null;
}

function isActive(lineNo: number) {
  return selectedLineNo.value === lineNo;
}

async function copySelectedLine() {
  if (selectedLine.value) {
    await copyLine(selectedLine.value.raw);
  }
}

async function copySelectedJson() {
  if (selectedLine.value?.metadata) {
    await copyJson(selectedLine.value.metadata);
  }
}

function visibleMetadataTags(line: DisplayLogLine) {
  if (line.parsed.importantFields.length) {
    return visibleRowImportantFields(line).map((field) => [field.key, field.value] as [string, unknown]);
  }
  return summarizeMetadata(line.metadata).tags;
}

function hiddenMetadataCount(line: DisplayLogLine) {
  return summarizeMetadata(line.metadata).hiddenCount;
}

function hiddenRowFieldCount(line: DisplayLogLine) {
  if (!line.parsed.importantFields.length) {
    return hiddenMetadataCount(line);
  }
  if (line.parsed.format === 'logfmt') {
    return 0;
  }
  return Math.max(0, Object.keys(line.parsed.fields).length - visibleRowImportantFields(line).length);
}

function formatMetadataValue(value: unknown) {
  return formatLogMetadataValue(value);
}

function visibleRowImportantFields(line: DisplayLogLine) {
  const hiddenRowKeys = new Set(['level', 'severity', 'msg', 'message', 'event']);
  return line.parsed.importantFields
    .filter((field) => !hiddenRowKeys.has(field.key))
    .filter((field) => field.value !== line.message)
    .slice(0, 3);
}

function formatJson(value: unknown) {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

function displayTimestamp(timestamp: string) {
  return formatLogViewerTimestamp(timestamp, locale);
}

function formattedFullTimestamp(timestamp: string) {
  return formatLocaleDateTime(timestamp, locale);
}

function levelTheme(level: LogLevel | null | undefined) {
  if (level === 'ERROR' || level === 'FATAL') return 'danger';
  if (level === 'WARN') return 'warning';
  if (level === 'INFO') return 'primary';
  return 'default';
}

function tokenClass(token: LogToken) {
  return [
    'log-viewer__token',
    `log-viewer__token--${token.type}`,
    token.level ? `log-viewer__token--level-${token.level.toLowerCase()}` : '',
  ];
}

async function copyTextWithFeedback(value: string) {
  try {
    const copied = await copyText(value);
    if (!copied) {
      MessagePlugin.error(props.copyErrorLabel);
      return;
    }
    MessagePlugin.success(props.copySuccessLabel);
  } catch {
    MessagePlugin.error(props.copyErrorLabel);
  }
}

function findFirstEndAfter(ends: readonly number[], target: number) {
  let low = 0;
  let high = ends.length;

  while (low < high) {
    const mid = Math.floor((low + high) / 2);
    if ((ends[mid] ?? 0) <= target) {
      low = mid + 1;
    } else {
      high = mid;
    }
  }

  return low;
}

function findFirstOffsetAtOrAfter(offsets: readonly number[], target: number) {
  let low = 0;
  let high = offsets.length;

  while (low < high) {
    const mid = Math.floor((low + high) / 2);
    if ((offsets[mid] ?? 0) < target) {
      low = mid + 1;
    } else {
      high = mid;
    }
  }

  return low;
}

function isViewportNearBottom(node: HTMLElement) {
  const remainingDistance = node.scrollHeight - (node.scrollTop + node.clientHeight);
  return remainingDistance <= AUTO_SCROLL_BOTTOM_THRESHOLD_PX;
}
</script>
<style scoped lang="less">
.log-viewer {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
  min-width: 0;
}

.log-viewer--framed {
  gap: 0;
}

.log-viewer__body {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
  min-height: 0;
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.log-viewer__toolbar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
}

.log-viewer__toolbar-group,
.log-viewer__switch {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.log-viewer__toolbar-left {
  flex: 0 0 auto;
}

.log-viewer__toolbar-middle {
  flex: 1 1 420px;
}

.log-viewer__toolbar-right {
  flex: 0 0 auto;
  justify-content: flex-end;
}

.log-viewer__limit {
  width: 96px;
}

.log-viewer__level-filter {
  width: 140px;
}

.log-viewer__search {
  flex: 1 1 280px;
  min-width: 220px;
}

.log-viewer__match-count,
.log-viewer__switch {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.log-viewer__switch {
  white-space: nowrap;
}

.log-viewer__viewport {
  background: color-mix(in srgb, var(--td-bg-color-page) 78%, var(--td-bg-color-container) 22%);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  color: var(--td-text-color-primary);
  flex: 1 1 auto;
  min-height: 0;
  min-width: 0;
  overflow: auto;
  padding: var(--graft-density-gap-8);
  position: relative;
}

.log-viewer__viewport-state {
  min-height: 100%;
}

.log-viewer__header-row {
  backdrop-filter: blur(12px);
  background: color-mix(in srgb, var(--td-bg-color-container) 92%, transparent);
  border-bottom: 1px solid var(--td-component-stroke);
  column-gap: var(--graft-density-gap-6);
  display: grid;
  grid-template-columns: 19ch 72px 80px minmax(0, 1fr) 48px;
  padding: 0 var(--graft-density-gap-6) var(--graft-density-gap-8);
  position: sticky;
  top: 0;
  z-index: 2;
}

.log-viewer__header-cell {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  font-weight: 600;
  line-height: 24px;
}

.log-viewer__header-cell--actions {
  text-align: right;
}

.log-viewer__lines {
  list-style: none;
  margin: 0;
  min-width: max(100%, 760px);
  padding: var(--graft-density-gap-8) 0 0;
  position: relative;
}

.log-viewer__line {
  border-left: var(--graft-density-gap-2) solid transparent;
  border-radius: var(--td-radius-small);
  column-gap: var(--graft-density-gap-6);
  display: grid;
  grid-template-columns: 19ch 72px 80px minmax(0, 1fr) 48px;
  inset-inline: 0;
  margin-block: var(--graft-density-gap-1);
  min-height: 38px;
  padding: var(--graft-density-gap-6);
  position: absolute;
}

.log-viewer__line:hover,
.log-viewer__line:focus-within {
  background: color-mix(in srgb, var(--td-bg-color-container-hover) 54%, transparent);
  outline: none;
}

.log-viewer__line--active {
  background: var(--td-bg-color-container);
  box-shadow: inset 0 0 0 1px var(--td-component-stroke);
}

.log-viewer__line--active.log-viewer__line--default,
.log-viewer__line--active.log-viewer__line--info {
  border-left-color: var(--td-brand-color);
}

.log-viewer__line--active.log-viewer__line--muted {
  border-left-color: var(--td-text-color-placeholder);
}

.log-viewer__timestamp-cell,
.log-viewer__level-cell,
.log-viewer__stream-cell,
.log-viewer__content,
.log-viewer__row-actions {
  align-self: start;
  min-width: 0;
}

.log-viewer__timestamp,
.log-viewer__message {
  font-family: var(--td-font-family-monospace);
}

.log-viewer__timestamp-cell {
  width: 19ch;
}

.log-viewer__timestamp {
  color: var(--td-text-color-placeholder);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.log-viewer__level-cell {
  display: flex;
  justify-content: flex-start;
}

.log-viewer__level-cell :deep(.t-tag) {
  padding-inline: var(--graft-density-gap-4);
}

.log-viewer__stream-pill {
  align-items: center;
  border: 1px solid transparent;
  border-radius: 999px;
  display: inline-flex;
  font: var(--td-font-body-small);
  font-weight: 500;
  letter-spacing: 0.02em;
  line-height: 18px;
  padding: 0 var(--graft-density-gap-4);
}

.log-viewer__stream-pill--stdout {
  background: color-mix(in srgb, var(--td-success-color-1) 92%, transparent);
  border-color: color-mix(in srgb, var(--td-success-color-5) 30%, transparent);
  color: var(--td-success-color-7);
}

.log-viewer__stream-pill--stderr {
  background: color-mix(in srgb, var(--td-error-color-1) 92%, transparent);
  border-color: color-mix(in srgb, var(--td-error-color-5) 30%, transparent);
  color: var(--td-error-color-7);
}

.log-viewer__stream-pill--detail {
  flex: 0 0 auto;
}

.log-viewer__message-row {
  align-items: center;
  display: flex;
  min-width: 0;
}

.log-viewer__message {
  color: var(--td-text-color-primary);
  line-height: var(--td-line-height-body-medium);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-viewer__metadata-tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-4);
  margin-top: var(--graft-density-gap-4);
  max-width: 100%;
  min-width: 0;
}

.log-viewer__metadata-tags :deep(.t-tag) {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  line-height: 18px;
  max-width: 240px;
  overflow: hidden;
  padding: 0 var(--graft-density-gap-6);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-viewer__metadata-tags :deep(.t-button) {
  color: var(--td-text-color-secondary);
  min-width: 0;
  padding-inline: var(--graft-density-gap-4);
}

.log-viewer__row-actions {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-2);
  justify-content: flex-end;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.16s ease;
  width: 48px;
}

.log-viewer__line:hover .log-viewer__row-actions,
.log-viewer__line:focus-within .log-viewer__row-actions,
.log-viewer__line--active .log-viewer__row-actions {
  opacity: 1;
  pointer-events: auto;
}

.log-viewer__icon-action {
  color: var(--td-text-color-secondary);
}

.log-viewer__viewport--wrap .log-viewer__lines {
  min-width: 0;
}

.log-viewer__viewport--wrap .log-viewer__message {
  overflow: visible;
  overflow-wrap: anywhere;
  text-overflow: unset;
  white-space: pre-wrap;
}

.log-viewer__line--danger {
  background: color-mix(in srgb, var(--td-error-color-5) 4%, transparent);
  border-left-color: var(--td-error-color);
}

.log-viewer__line--warning {
  background: color-mix(in srgb, var(--td-warning-color-5) 4%, transparent);
  border-left-color: var(--td-warning-color);
}

.log-viewer__line--muted {
  color: var(--td-text-color-secondary);
}

.log-viewer__jump-bottom-wrap {
  bottom: var(--graft-density-gap-12);
  display: flex;
  justify-content: flex-end;
  pointer-events: none;
  position: sticky;
  z-index: 3;
}

.log-viewer__jump-bottom-wrap :deep(.t-button) {
  box-shadow: var(--td-shadow-2);
  pointer-events: auto;
}

.log-viewer :deep(.log-viewer__drawer .t-drawer__body) {
  padding: var(--graft-density-gap-24);
}

.log-viewer :deep(.log-viewer__drawer .t-drawer__content-wrapper) {
  max-width: min(720px, 100vw);
}

.log-viewer__detail-drawer {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.log-viewer__summary {
  border-bottom: 1px solid var(--td-component-stroke);
  margin-bottom: var(--graft-density-gap-18);
  min-width: 0;
  padding-bottom: var(--graft-density-gap-16);
}

.log-viewer__summary-level,
.log-viewer__detail-level {
  flex: 0 0 auto;
}

.log-viewer__summary-title {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.log-viewer__summary-main {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.log-viewer__summary-message {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
  min-width: 0;
  overflow-wrap: anywhere;
}

.log-viewer__summary-meta {
  align-items: center;
  color: var(--td-text-color-secondary);
  column-gap: var(--graft-density-gap-6);
  display: flex;
  flex-wrap: wrap;
  font: var(--td-font-body-small);
  margin-top: var(--graft-density-gap-6);
  min-width: 0;
}

.log-viewer__summary-source {
  display: inline-block;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.log-viewer__drawer-section-header {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  margin-bottom: var(--graft-density-gap-8);
  min-width: 0;
}

.log-viewer__drawer-section {
  display: flex;
  flex-direction: column;
  margin-top: var(--graft-density-gap-18);
  min-width: 0;
}

.log-viewer__drawer-section-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  font-weight: 600;
}

.log-viewer__field-chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  margin-top: var(--graft-density-gap-8);
  min-width: 0;
}

.log-viewer__field-chip {
  align-items: center;
  background: var(--td-bg-color-secondarycontainer);
  border-radius: var(--td-radius-small);
  color: var(--td-text-color-secondary);
  display: inline-flex;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-4);
  max-width: 100%;
  min-width: 0;
  padding: var(--graft-density-gap-4) var(--graft-density-gap-8);
}

.log-viewer__field-key {
  color: var(--td-text-color-placeholder);
  flex: 0 0 auto;
}

.log-viewer__field-value {
  color: var(--td-text-color-primary);
  display: inline-block;
  max-width: 260px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.log-viewer__basic {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  margin-top: var(--graft-density-gap-8);
  padding: var(--graft-density-gap-12) var(--graft-density-gap-14);
}

.log-viewer__descriptions {
  display: grid;
  gap: var(--graft-density-gap-8) var(--graft-density-gap-12);
  grid-template-columns: 72px minmax(0, 1fr);
}

.log-viewer__description-label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.log-viewer__description-value {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  min-width: 0;
  overflow-wrap: anywhere;
}

.log-viewer__level-value {
  align-items: center;
  display: inline-flex;
  justify-self: start;
}

.log-viewer__code-block {
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  font: var(--td-font-body-small);
  font-family: var(--td-font-family-mono, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace);
  line-height: 1.7;
  margin: 0;
  max-height: 260px;
  overflow: auto;
  padding: var(--graft-density-gap-12) var(--graft-density-gap-14);
  white-space: pre;
}

.log-viewer__code-block code {
  font-family: inherit;
}

.log-viewer__code-block--raw {
  max-height: 180px;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
  word-break: normal;
}

.log-viewer__token--keyword {
  background: color-mix(in srgb, var(--td-warning-color-5) 34%, transparent);
  border-radius: var(--td-radius-small);
  color: var(--td-text-color-primary);
}

.log-viewer__token--field-key {
  color: var(--td-brand-color);
}

.log-viewer__token--field-value {
  color: var(--td-text-color-secondary);
}

.log-viewer__token--level-error,
.log-viewer__token--level-fatal {
  color: var(--td-error-color);
  font-weight: 600;
}

.log-viewer__token--level-warn {
  color: var(--td-warning-color);
  font-weight: 600;
}

.log-viewer__token--level-info {
  color: var(--td-brand-color);
  font-weight: 600;
}

.log-viewer__token--level-debug,
.log-viewer__token--level-trace {
  color: var(--td-text-color-placeholder);
}

@media (width <= 1280px) {
  .log-viewer__toolbar-left,
  .log-viewer__toolbar-middle,
  .log-viewer__toolbar-right {
    width: 100%;
  }

  .log-viewer__toolbar-right {
    justify-content: flex-start;
  }
}

@media (width <= 1024px) {
  .log-viewer__header-row,
  .log-viewer__line {
    grid-template-columns: 17ch 68px 76px minmax(0, 1fr) 44px;
  }
}

@media (width <= 760px) {
  .log-viewer__lines {
    min-width: 680px;
  }
}
</style>
