import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import ResponsiveSidebar from './ResponsiveSidebar.vue';

const drawerStub = defineComponent({
  name: 'TDrawerStub',
  props: { visible: { type: Boolean, default: false } },
  emits: ['update:visible'],
  setup(props, { emit, slots }) {
    return () =>
      props.visible
        ? h('div', { 'data-drawer-visible': 'true', onClick: () => emit('update:visible', false) }, slots.default?.())
        : null;
  },
});

describe('ResponsiveSidebar', () => {
  it('keeps desktop and compact navigation in the shell while exposing compact state', () => {
    const wrapper = mount(ResponsiveSidebar, {
      props: { mode: 'compact' },
      slots: { default: ({ compact }: { compact: boolean }) => h('span', String(compact)) },
    });

    expect(wrapper.find('aside').exists()).toBe(true);
    expect(wrapper.text()).toBe('true');
  });

  it('renders the mobile navigation in a controlled drawer', async () => {
    const wrapper = mount(ResponsiveSidebar, {
      props: { mode: 'drawer', visible: true },
      slots: { drawer: '<nav>Navigation</nav>' },
      global: { stubs: { TDrawer: drawerStub } },
    });

    expect(wrapper.get('[data-drawer-visible="true"]').text()).toContain('Navigation');
    await wrapper.get('[data-drawer-visible="true"]').trigger('click');
    expect(wrapper.emitted('update:visible')).toEqual([[false]]);
  });
});
