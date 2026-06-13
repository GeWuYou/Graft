// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import ManagementTableCard from './ManagementTableCard.vue';

describe('ManagementTableCard', () => {
  it('renders a shared table header with left summary and right toolbar slots', () => {
    const wrapper = mount(ManagementTableCard, {
      slots: {
        default: '<div data-testid="body">table</div>',
        head: '<p data-testid="summary">Current page 20 / 42</p>',
        toolbar: '<button data-testid="refresh">Refresh</button>',
      },
    });

    expect(wrapper.get('.management-table-card__head-main').text()).toContain('Current page 20 / 42');
    expect(wrapper.get('.management-table-card__toolbar').text()).toContain('Refresh');
    expect(wrapper.get('[data-testid="body"]').text()).toBe('table');
  });
});
