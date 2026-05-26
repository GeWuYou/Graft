import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import AssignmentDrawer from './AssignmentDrawer.vue';

describe('AssignmentDrawer', () => {
  it('syncs v-model visibility when the drawer requests a visibility change', async () => {
    const wrapper = mount(AssignmentDrawer, {
      props: {
        title: 'Assignments',
        visible: true,
      },
      global: {
        stubs: {
          TDrawer: {
            name: 'TDrawer',
            template: '<div><slot /></div>',
          },
        },
      },
    });

    await wrapper.getComponent({ name: 'TDrawer' }).vm.$emit('update:visible', false);

    expect(wrapper.emitted('update:visible')).toEqual([[false]]);
    expect(wrapper.emitted('close')).toBeUndefined();
  });
});
