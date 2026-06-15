// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import JsonViewer from './JsonViewer.vue';

vi.mock('./copy', () => ({
  copyText: vi.fn(async () => true),
}));

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

describe('JsonViewer', () => {
  it('renders objects, arrays, null values, and masked fields', () => {
    const wrapper = mountViewer({
      id: 'container-1',
      token: 'secret-token',
      ports: [80, 443],
      optional: null,
    });

    expect(wrapper.text()).toContain('container');
    expect(wrapper.text()).toContain('Object(4)');
    expect(wrapper.text()).toContain('ports');
    expect(wrapper.text()).toContain('Array(2)');
    expect(wrapper.text()).toContain('optional');
    expect(wrapper.text()).toContain('null');
    expect(wrapper.text()).toContain('******');
    expect(wrapper.text()).not.toContain('secret-token');
  });

  it('renders empty state for empty objects', () => {
    const wrapper = mountViewer({});

    expect(wrapper.text()).toContain('No JSON');
  });
});

function mountViewer(value: unknown) {
  return mount(JsonViewer, {
    props: {
      value,
      title: 'Raw JSON',
      description: 'Masked',
      rootLabel: 'container',
      sourceLabel: 'Source',
      treeLabel: 'Tree',
      copyLabel: 'Copy',
      copySuccessLabel: 'Copied',
      copyErrorLabel: 'Copy Failed',
      emptyLabel: 'No JSON',
      errorLabel: 'Invalid JSON',
    },
    global: {
      stubs: {
        't-alert': defineComponent({
          props: ['title'],
          setup: (props) => () => h('div', String(props.title ?? '')),
        }),
        't-button': defineComponent({
          emits: ['click'],
          setup:
            (_, { attrs, emit, slots }) =>
            () =>
              h('button', { ...attrs, onClick: () => emit('click') }, slots.default?.()),
        }),
        't-empty': defineComponent({
          props: ['description'],
          setup: (props) => () => h('div', String(props.description ?? '')),
        }),
      },
    },
  });
}
