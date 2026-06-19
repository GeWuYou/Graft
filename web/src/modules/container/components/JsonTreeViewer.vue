<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="json-tree-viewer" data-testid="container-raw-tree-viewer">
    <div v-if="!visibleNodes.length" class="json-tree-viewer__empty">
      {{ emptyLabel }}
    </div>

    <div v-else class="json-tree-viewer__viewport graft-scrollbar">
      <div
        v-for="node in visibleNodes"
        :key="node.path"
        class="json-tree-viewer__row"
        :class="{
          'json-tree-viewer__row--match': node.matched,
          'json-tree-viewer__row--clickable': node.expandable,
        }"
        :style="{ '--json-tree-indent': `${node.depth}` }"
        @click="toggleNode(node.path, node.expandable)"
      >
        <button
          class="json-tree-viewer__toggle"
          :class="{ 'json-tree-viewer__toggle--hidden': !node.expandable }"
          type="button"
          :aria-label="node.expanded ? collapseLabel : expandLabel"
          @click.stop="toggleNode(node.path, node.expandable)"
        >
          <span>{{ node.expandable ? (node.expanded ? '▾' : '▸') : '' }}</span>
        </button>
        <span class="json-tree-viewer__key">{{ node.label }}</span>
        <div class="json-tree-viewer__value">
          <t-tag
            v-if="node.kind === 'boolean'"
            size="small"
            :theme="node.preview === 'true' ? 'success' : 'default'"
            variant="light-outline"
          >
            {{ node.preview }}
          </t-tag>
          <span v-else-if="node.kind === 'null'" class="json-tree-viewer__null">{{ node.preview }}</span>
          <span v-else-if="node.kind === 'number'" class="json-tree-viewer__number">{{ node.preview }}</span>
          <span v-else-if="node.kind === 'string'" class="json-tree-viewer__string" :title="node.preview">
            {{ node.preview }}
          </span>
          <span v-else class="json-tree-viewer__summary">{{ node.preview }}</span>
          <t-tag
            v-if="node.sensitive"
            size="small"
            theme="warning"
            variant="light-outline"
            class="json-tree-viewer__badge"
          >
            {{ sensitiveLabel }}
          </t-tag>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue';

type JsonTreeNodeKind = 'array' | 'boolean' | 'null' | 'number' | 'object' | 'string' | 'unknown';

type JsonTreeNode = {
  depth: number;
  expandable: boolean;
  expanded: boolean;
  kind: JsonTreeNodeKind;
  label: string;
  matched: boolean;
  path: string;
  preview: string;
  sensitive: boolean;
};

const props = defineProps<{
  collapseLabel: string;
  emptyLabel: string;
  expandLabel: string;
  expandedAll: boolean;
  expandAllToken: number;
  rootLabel: string;
  searchValue: string;
  sensitiveLabel: string;
  value: unknown;
}>();

const expandedPaths = ref(new Set<string>());

watch(() => props.expandAllToken, syncExpandedPaths, { immediate: true });

watch([() => props.value, () => props.rootLabel], () => {
  if (props.expandedAll) {
    syncExpandedPaths();
  }
});

const visibleNodes = computed(() => {
  const normalizedSearch = props.searchValue.trim().toLowerCase();
  return buildVisibleNodes(props.value, props.rootLabel, normalizedSearch, expandedPaths.value);
});

function syncExpandedPaths() {
  if (props.expandedAll) {
    const next = new Set<string>();
    collectExpandablePaths(props.value, props.rootLabel).forEach((path) => next.add(path));
    expandedPaths.value = next;
    return;
  }

  expandedPaths.value = new Set();
}

function buildVisibleNodes(
  value: unknown,
  rootLabel: string,
  normalizedSearch: string,
  expanded: Set<string>,
): JsonTreeNode[] {
  const nodes: JsonTreeNode[] = [];

  const visit = (currentValue: unknown, label: string, path: string, depth: number) => {
    const kind = resolveKind(currentValue);
    const expandable = kind === 'array' || kind === 'object';
    const summary = formatPreview(currentValue, kind);
    const sensitive = resolveSensitive(currentValue);
    const matched = normalizedSearch ? `${label} ${summary}`.toLowerCase().includes(normalizedSearch) : false;
    const currentExpanded = expandable ? expanded.has(path) : false;

    nodes.push({
      depth,
      expandable,
      expanded: currentExpanded,
      kind,
      label,
      matched,
      path,
      preview: summary,
      sensitive,
    });

    if (!expandable || !currentExpanded) {
      return;
    }

    const entries = Array.isArray(currentValue)
      ? currentValue.map((item, index) => [String(index), item] as const)
      : Object.entries((currentValue as Record<string, unknown>) ?? {});

    entries.forEach(([childKey, childValue]) => {
      visit(childValue, childKey, `${path}.${childKey}`, depth + 1);
    });
  };

  visit(value, rootLabel, rootLabel, 0);
  return nodes.filter((node) => !normalizedSearch || node.matched || node.expandable || node.path === rootLabel);
}

function collectExpandablePaths(value: unknown, rootLabel: string) {
  const paths: string[] = [];
  const visit = (currentValue: unknown, path: string) => {
    const kind = resolveKind(currentValue);
    if (kind !== 'array' && kind !== 'object') {
      return;
    }
    paths.push(path);
    const entries = Array.isArray(currentValue)
      ? currentValue.map((item, index) => [String(index), item] as const)
      : Object.entries((currentValue as Record<string, unknown>) ?? {});
    entries.forEach(([childKey, childValue]) => visit(childValue, `${path}.${childKey}`));
  };

  visit(value, rootLabel);
  return paths;
}

function toggleNode(path: string, expandable: boolean) {
  if (!expandable) {
    return;
  }

  const next = new Set(expandedPaths.value);
  if (next.has(path)) {
    next.delete(path);
  } else {
    next.add(path);
  }
  expandedPaths.value = next;
}

function resolveKind(value: unknown): JsonTreeNodeKind {
  if (value === null) {
    return 'null';
  }
  if (Array.isArray(value)) {
    return 'array';
  }
  if (typeof value === 'boolean') {
    return 'boolean';
  }
  if (typeof value === 'number') {
    return 'number';
  }
  if (typeof value === 'string') {
    return 'string';
  }
  if (value && typeof value === 'object') {
    return 'object';
  }
  return 'unknown';
}

function formatPreview(value: unknown, kind: JsonTreeNodeKind) {
  if (kind === 'array') {
    return `Array(${Array.isArray(value) ? value.length : 0})`;
  }
  if (kind === 'object') {
    return `Object(${Object.keys((value as Record<string, unknown>) ?? {}).length})`;
  }
  if (kind === 'string') {
    return `"${String(value)}"`;
  }
  if (kind === 'null') {
    return 'null';
  }
  if (kind === 'unknown') {
    return '-';
  }
  return String(value);
}

function resolveSensitive(value: unknown) {
  const record =
    value && typeof value === 'object' && !Array.isArray(value) ? (value as Record<string, unknown>) : null;
  return record?.sensitive === true || record?.masked === true;
}
</script>
<style scoped lang="less">
.json-tree-viewer {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
}

.json-tree-viewer__empty {
  align-items: center;
  color: var(--td-text-color-placeholder);
  display: flex;
  flex: 1;
  justify-content: center;
  min-height: 0;
}

.json-tree-viewer__viewport {
  min-height: 0;
  min-width: 0;
  overflow: auto;
}

.json-tree-viewer__row {
  align-items: center;
  border-radius: var(--td-radius-medium);
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: 18px minmax(160px, 240px) minmax(0, 1fr);
  margin-left: calc(var(--json-tree-indent) * var(--graft-density-gap-16));
  min-width: max-content;
  padding: var(--graft-density-gap-4) var(--graft-density-gap-8);
}

.json-tree-viewer__row:hover {
  background: color-mix(in srgb, var(--td-brand-color-1) 60%, transparent);
}

.json-tree-viewer__row--match {
  background: color-mix(in srgb, var(--td-warning-color-1) 72%, transparent);
}

.json-tree-viewer__row--clickable {
  cursor: pointer;
}

.json-tree-viewer__toggle {
  align-items: center;
  background: transparent;
  border: 0;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  display: inline-flex;
  height: 18px;
  justify-content: center;
  padding: 0;
  width: 18px;
}

.json-tree-viewer__toggle--hidden {
  cursor: default;
}

.json-tree-viewer__key {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  min-width: 0;
}

.json-tree-viewer__value {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.json-tree-viewer__string,
.json-tree-viewer__summary {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.json-tree-viewer__string {
  color: var(--td-success-color-6);
  font-family: var(
    --td-font-family-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    monospace
  );
}

.json-tree-viewer__number {
  color: var(--td-warning-color-7);
}

.json-tree-viewer__null {
  color: var(--td-text-color-placeholder);
}

.json-tree-viewer__badge {
  flex: 0 0 auto;
}
</style>
