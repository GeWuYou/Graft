// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import AssignmentDrawer from './AssignmentDrawer.vue';

describe('AssignmentDrawer', () => {
  it('emits close requests instead of directly hiding the drawer', async () => {
    const wrapper = mount(AssignmentDrawer, {
      props: {
        title: 'Assignments',
        visible: true,
      },
      global: {
        stubs: {
          TDrawer: {
            name: 'TDrawer',
            props: ['closeOnEscKeydown', 'closeOnOverlayClick'],
            emits: ['close', 'close-btn-click', 'esc-keydown', 'overlay-click', 'update:visible'],
            template:
              '<div :data-close-on-esc="String(closeOnEscKeydown)" :data-close-on-overlay="String(closeOnOverlayClick)"><slot /></div>',
          },
        },
      },
    });

    await wrapper.getComponent({ name: 'TDrawer' }).vm.$emit('update:visible', false);

    expect(wrapper.emitted('update:visible')).toBeUndefined();
    expect(wrapper.emitted('close')).toEqual([[]]);
    expect(wrapper.getComponent({ name: 'TDrawer' }).attributes('data-close-on-esc')).toBe('false');
    expect(wrapper.getComponent({ name: 'TDrawer' }).attributes('data-close-on-overlay')).toBe('false');
  });
});
