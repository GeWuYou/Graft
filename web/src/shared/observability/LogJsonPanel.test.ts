import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import LogJsonPanel from './LogJsonPanel.vue';

const collapseStub = defineComponent({
  name: 'TCollapseStub',
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const collapsePanelStub = defineComponent({
  name: 'TCollapsePanelStub',
  setup(_, { slots }) {
    return () => h('section', [slots.header?.(), slots.headerRightContent?.(), slots.default?.()]);
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_, { attrs, emit, slots }) {
    return () => h('button', { ...attrs, onClick: () => emit('click') }, slots.default?.());
  },
});

vi.mock('./copy', () => ({
  copyText: vi.fn(async () => true),
}));

vi.mock('tdesign-vue-next', () => ({
  MessagePlugin: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

describe('LogJsonPanel', () => {
  it('renders formatted JSON and copy action for non-empty values', () => {
    const wrapper = mount(LogJsonPanel, {
      props: {
        title: 'Metadata',
        expandLabel: 'Expand JSON',
        collapseLabel: 'Collapse JSON',
        copyLabel: 'Copy JSON',
        copySuccessLabel: 'Copied',
        copyFailLabel: 'Failed',
        emptyText: 'No data',
        value: { request_id: 'req-1' },
      },
      global: {
        stubs: {
          't-collapse': collapseStub,
          't-collapse-panel': collapsePanelStub,
          't-button': buttonStub,
        },
      },
    });

    expect(wrapper.text()).toContain('Metadata');
    expect(wrapper.text()).toContain('Copy JSON');
    expect(wrapper.text()).toContain('"request_id": "req-1"');
  });

  it('renders the empty text when value is empty', () => {
    const wrapper = mount(LogJsonPanel, {
      props: {
        title: 'Metadata',
        expandLabel: 'Expand JSON',
        collapseLabel: 'Collapse JSON',
        copyLabel: 'Copy JSON',
        copySuccessLabel: 'Copied',
        copyFailLabel: 'Failed',
        emptyText: 'No data',
        value: {},
      },
      global: {
        stubs: {
          't-collapse': collapseStub,
          't-collapse-panel': collapsePanelStub,
          't-button': buttonStub,
        },
      },
    });

    expect(wrapper.text()).toContain('No data');
    expect(wrapper.text()).not.toContain('Copy JSON');
  });
});
