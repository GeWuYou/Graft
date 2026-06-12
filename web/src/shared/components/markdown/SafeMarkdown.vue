<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="safe-markdown" v-html="renderedHtml" />
</template>
<script setup lang="ts">
import DOMPurify from 'dompurify';
import MarkdownIt from 'markdown-it';
import { computed } from 'vue';

const props = withDefaults(
  defineProps<{
    source?: string | null;
  }>(),
  {
    source: '',
  },
);

const markdown = new MarkdownIt({
  breaks: false,
  html: false,
  linkify: true,
  typographer: false,
});

const defaultLinkOpen =
  markdown.renderer.rules.link_open ?? ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options));

markdown.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  const token = tokens[idx];
  const hrefIndex = token.attrIndex('href');
  const href = hrefIndex >= 0 ? token.attrs?.[hrefIndex]?.[1] : '';
  const isExternal = href ? /^(https?:)?\/\//iu.test(href) : false;

  if (isExternal && token.attrIndex('target') < 0) {
    token.attrPush(['target', '_blank']);
  }
  if (isExternal) {
    const relIndex = token.attrIndex('rel');
    if (relIndex < 0) {
      token.attrPush(['rel', 'noopener noreferrer']);
    } else if (token.attrs) {
      token.attrs[relIndex][1] = 'noopener noreferrer';
    }
  }

  return defaultLinkOpen(tokens, idx, options, env, self);
};

const renderedHtml = computed(() =>
  DOMPurify.sanitize(markdown.render(props.source ?? ''), {
    USE_PROFILES: { html: true },
  }),
);
</script>
<style scoped lang="less">
.safe-markdown {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  line-height: 1.7;
  overflow-wrap: anywhere;
  text-align: left;
}

.safe-markdown :deep(:first-child) {
  margin-top: 0;
}

.safe-markdown :deep(:last-child) {
  margin-bottom: 0;
}

.safe-markdown :deep(p),
.safe-markdown :deep(ul),
.safe-markdown :deep(ol),
.safe-markdown :deep(pre),
.safe-markdown :deep(blockquote),
.safe-markdown :deep(table),
.safe-markdown :deep(hr) {
  margin: 0 0 var(--graft-density-gap-10);
}

.safe-markdown :deep(h1),
.safe-markdown :deep(h2),
.safe-markdown :deep(h3),
.safe-markdown :deep(h4),
.safe-markdown :deep(h5),
.safe-markdown :deep(h6) {
  color: var(--td-text-color-primary);
  font-weight: 600;
  line-height: 1.35;
  margin: var(--graft-density-gap-18) 0 var(--graft-density-gap-10);
}

.safe-markdown :deep(h1) {
  font: var(--td-font-title-large);
}

.safe-markdown :deep(h2) {
  font: var(--td-font-title-medium);
}

.safe-markdown :deep(h3),
.safe-markdown :deep(h4) {
  font: var(--td-font-title-small);
}

.safe-markdown :deep(h5),
.safe-markdown :deep(h6) {
  font: var(--td-font-body-large);
}

.safe-markdown :deep(ul),
.safe-markdown :deep(ol) {
  padding-left: var(--graft-density-gap-24);
}

.safe-markdown :deep(a) {
  color: var(--td-brand-color);
  text-decoration: none;
}

.safe-markdown :deep(a:hover) {
  color: var(--td-brand-color-hover);
  text-decoration: underline;
}

.safe-markdown :deep(code) {
  background: var(--td-bg-color-component);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-small);
  color: var(--td-text-color-primary);
  font-family: var(--td-font-family-mono);
  padding: 0 var(--graft-density-gap-4);
}

.safe-markdown :deep(pre) {
  background: var(--td-bg-color-component);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  max-width: 100%;
  overflow: auto;
  padding: var(--graft-density-gap-12);
}

.safe-markdown :deep(pre code) {
  background: transparent;
  border: 0;
  padding: 0;
}

.safe-markdown :deep(blockquote) {
  border-left: 3px solid var(--td-brand-color);
  color: var(--td-text-color-secondary);
  padding-left: var(--graft-density-gap-12);
}

.safe-markdown :deep(table) {
  border-collapse: collapse;
  display: block;
  max-width: 100%;
  overflow: auto;
}

.safe-markdown :deep(th),
.safe-markdown :deep(td) {
  border: 1px solid var(--td-component-stroke);
  padding: var(--graft-density-gap-6) var(--graft-density-gap-10);
}

.safe-markdown :deep(th) {
  background: var(--td-bg-color-component);
  color: var(--td-text-color-primary);
  font-weight: 600;
}

.safe-markdown :deep(hr) {
  border: 0;
  border-top: 1px solid var(--td-component-stroke);
}
</style>
