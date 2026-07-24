import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { UpdateRelease } from '../types/update';
import UpdateReleaseDetailDialog from './UpdateReleaseDetailDialog.vue';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'en-US' },
    t: (key: string) =>
      ({
        'update.center.channels.beta': 'Beta',
        'update.center.release.notesEmpty': 'No release notes',
        'update.preview.detail.close': 'Close details',
        'update.preview.detail.publishedAt': 'Published',
        'update.preview.detail.releaseDescription': 'A verified Graft release',
        'update.preview.detail.title': 'Release details',
        'update.preview.detail.upgradeNotes': 'Upgrade notes',
        'update.preview.viewRelease': 'View release',
      })[key] ?? key,
  }),
}));

vi.mock('@/shared/components/markdown', () => ({
  MarkdownViewer: defineComponent({
    props: { source: { type: String, default: '' } },
    setup(props) {
      return () => h('article', { 'data-testid': 'markdown-viewer' }, props.source);
    },
  }),
}));

vi.mock('@/shared/observability', () => ({
  formatLocaleDateTime: () => '2026-07-24 12:00:00',
}));

const release = {
  version: '1.1.0',
  channel: 'beta',
  notes: '# Release notes\n\n- Fixes',
  notes_url: 'https://github.com/GeWuYou/Graft/releases/tag/v1.1.0',
  upgrade_notes: 'Restart after migration.',
  published_at: '2026-07-24T04:00:00Z',
} as UpdateRelease;

const dialogStub = defineComponent({
  props: { visible: Boolean },
  emits: ['close'],
  setup(_, { slots }) {
    return () => h('section', { 'data-testid': 'release-dialog' }, [slots.default?.()]);
  },
});

const buttonStub = defineComponent({
  emits: ['click'],
  setup(_, { emit, slots }) {
    return () => h('button', { type: 'button', onClick: () => emit('click') }, slots.default?.());
  },
});

describe('UpdateReleaseDetailDialog', () => {
  it('renders release metadata, markdown notes, and upgrade notes', () => {
    const wrapper = mount(UpdateReleaseDetailDialog, {
      props: { release, visible: true },
      global: {
        stubs: {
          't-dialog': dialogStub,
          't-link': defineComponent({
            setup:
              (_, { slots }) =>
              () =>
                h('a', slots.default?.()),
          }),
          't-tag': defineComponent({
            setup:
              (_, { slots }) =>
              () =>
                h('span', slots.default?.()),
          }),
          't-button': buttonStub,
        },
      },
    });

    expect(wrapper.text()).toContain('1.1.0');
    expect(wrapper.text()).toContain('Beta');
    expect(wrapper.text()).toContain('2026-07-24 12:00:00');
    expect(wrapper.get('[data-testid="markdown-viewer"]').text()).toContain('# Release notes\n\n- Fixes');
    expect(wrapper.text()).toContain('Restart after migration.');
  });

  it('does not render an external link for an unsafe release URL', () => {
    const wrapper = mount(UpdateReleaseDetailDialog, {
      props: { release: { ...release, notes_url: 'javascript:alert(1)' }, visible: true },
      global: {
        stubs: {
          't-dialog': dialogStub,
          't-link': defineComponent({
            setup:
              (_, { slots }) =>
              () =>
                h('a', slots.default?.()),
          }),
          't-tag': defineComponent({
            setup:
              (_, { slots }) =>
              () =>
                h('span', slots.default?.()),
          }),
          't-button': buttonStub,
        },
      },
    });

    expect(wrapper.find('a').exists()).toBe(false);
  });
});
