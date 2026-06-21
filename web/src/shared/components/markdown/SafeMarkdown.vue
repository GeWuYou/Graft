<template>
  <div class="markdown-viewer" v-html="renderedHtml" />
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
.markdown-viewer {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  line-height: 1.7;
  overflow-wrap: break-word;
  text-align: left;
}

.markdown-viewer :deep(:first-child) {
  margin-top: 0;
}

.markdown-viewer :deep(:last-child) {
  margin-bottom: 0;
}

.markdown-viewer :deep(p),
.markdown-viewer :deep(ul),
.markdown-viewer :deep(ol),
.markdown-viewer :deep(pre),
.markdown-viewer :deep(blockquote),
.markdown-viewer :deep(table),
.markdown-viewer :deep(hr) {
  margin: 0 0 var(--graft-density-gap-10);
}

.markdown-viewer :deep(h1),
.markdown-viewer :deep(h2),
.markdown-viewer :deep(h3),
.markdown-viewer :deep(h4),
.markdown-viewer :deep(h5),
.markdown-viewer :deep(h6) {
  color: var(--td-text-color-primary);
  font-weight: 600;
  line-height: 1.35;
  margin: var(--graft-density-gap-18) 0 var(--graft-density-gap-10);
}

.markdown-viewer :deep(h1) {
  font: var(--td-font-title-large);
}

.markdown-viewer :deep(h2) {
  font: var(--td-font-title-medium);
}

.markdown-viewer :deep(h3),
.markdown-viewer :deep(h4) {
  font: var(--td-font-title-small);
}

.markdown-viewer :deep(h5),
.markdown-viewer :deep(h6) {
  font: var(--td-font-body-large);
}

.markdown-viewer :deep(ul),
.markdown-viewer :deep(ol) {
  padding-left: var(--graft-density-gap-24);
}

.markdown-viewer :deep(a) {
  color: var(--td-brand-color);
  text-decoration: none;
  word-break: break-all;
}

.markdown-viewer :deep(a:hover) {
  color: var(--td-brand-color-hover);
  text-decoration: underline;
}

.markdown-viewer :deep(code) {
  background: color-mix(in srgb, var(--td-bg-color-component) 86%, var(--td-brand-color) 14%);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-small);
  color: var(--td-text-color-primary);
  font-family: var(--td-font-family-mono);
  font-size: var(--td-font-size-body-small);
  padding: 0 var(--graft-density-gap-4);
}

.markdown-viewer :deep(pre) {
  background: color-mix(in srgb, var(--td-bg-color-component) 92%, var(--td-bg-color-page) 8%);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  max-width: 100%;
  overflow: auto;
  padding: var(--graft-density-gap-12);
  scrollbar-color: var(--td-scrollbar-color) transparent;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
}

.markdown-viewer :deep(pre::-webkit-scrollbar),
.markdown-viewer :deep(table::-webkit-scrollbar) {
  height: 8px;
  width: 8px;
}

.markdown-viewer :deep(pre::-webkit-scrollbar-track),
.markdown-viewer :deep(table::-webkit-scrollbar-track) {
  background: transparent;
}

.markdown-viewer :deep(pre::-webkit-scrollbar-thumb),
.markdown-viewer :deep(table::-webkit-scrollbar-thumb) {
  background-color: var(--td-scrollbar-color);
  border-radius: var(--td-radius-round);
}

.markdown-viewer :deep(pre code) {
  background: transparent;
  border: 0;
  display: block;
  min-width: max-content;
  padding: 0;
  white-space: pre;
}

.markdown-viewer :deep(blockquote) {
  background: color-mix(in srgb, var(--td-bg-color-container-hover) 76%, transparent);
  border-left: 3px solid color-mix(in srgb, var(--td-brand-color) 54%, var(--td-component-stroke));
  border-radius: 0 var(--td-radius-medium) var(--td-radius-medium) 0;
  color: var(--td-text-color-secondary);
  margin-left: 0;
  padding-left: var(--graft-density-gap-12);
}

.markdown-viewer :deep(table) {
  border-collapse: collapse;
  display: block;
  max-width: 100%;
  overflow: auto;
  scrollbar-color: var(--td-scrollbar-color) transparent;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
  white-space: nowrap;
}

.markdown-viewer :deep(th),
.markdown-viewer :deep(td) {
  border: 1px solid var(--td-component-stroke);
  padding: var(--graft-density-gap-6) var(--graft-density-gap-10);
}

.markdown-viewer :deep(th) {
  background: var(--td-bg-color-component);
  color: var(--td-text-color-primary);
  font-weight: 600;
}

.markdown-viewer :deep(img) {
  border-radius: var(--td-radius-medium);
  display: block;
  height: auto;
  max-width: 100%;
}

.markdown-viewer :deep(hr) {
  border: 0;
  border-top: 1px solid var(--td-component-stroke);
}
</style>
