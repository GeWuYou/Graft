import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import ManagementEntityCard from './ManagementEntityCard.vue';

describe('ManagementEntityCard', () => {
  it('presents an ordered subset of table columns and reacts to table heading changes', async () => {
    const wrapper = mount(ManagementEntityCard, {
      props: {
        columns: [
          { colKey: 'name', title: 'Name' },
          { colKey: 'size', title: 'Size' },
          { colKey: 'driver', title: 'Driver' },
        ],
        fields: ['driver', 'size', 'unknown'],
        row: { name: 'example', size: 0, driver: 'local' },
      },
      slots: { 'field-driver': '<strong>Custom driver</strong>', actions: '<button>Details</button>' },
    });

    expect(wrapper.findAll('dt').map((field) => field.text())).toEqual(['Driver', 'Size']);
    expect(wrapper.findAll('dd').map((field) => field.text())).toEqual(['Custom driver', '0']);
    expect(wrapper.get('button').text()).toBe('Details');

    await wrapper.setProps({ columns: [{ colKey: 'size', title: '容量' }] });
    expect(wrapper.findAll('dt').map((field) => field.text())).toEqual(['容量']);
  });
});
