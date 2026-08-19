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
    expect(wrapper.get('.management-table-card__toolbar-row').find('[data-testid="refresh"]').exists()).toBe(true);
    expect(wrapper.get('[data-testid="body"]').text()).toBe('table');
  });

  it('keeps batch actions and list tools in the same toolbar row', () => {
    const wrapper = mount(ManagementTableCard, {
      slots: {
        batch: '<span data-testid="batch">2 selected</span>',
        toolbar: '<button data-testid="refresh">Refresh</button>',
      },
    });

    const row = wrapper.get('.management-table-card__toolbar-row');
    expect(row.find('[data-testid="batch"]').exists()).toBe(true);
    expect(row.find('[data-testid="refresh"]').exists()).toBe(true);
  });
});
