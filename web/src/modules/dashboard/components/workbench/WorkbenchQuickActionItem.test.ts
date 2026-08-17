import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { QuickAction } from '../../presentation/workbench';
import WorkbenchQuickActionItem from './WorkbenchQuickActionItem.vue';

vi.mock('@/locales', () => ({ t: (key: string) => key }));
vi.mock('@/shared/icons/MenuIcon.vue', () => ({
  default: defineComponent({
    name: 'GraftMenuIconStub',
    props: { iconKey: { type: String, default: '' } },
    setup(props) {
      return () => h('i', { 'data-icon-key': props.iconKey });
    },
  }),
}));

const action: QuickAction = {
  id: 'build',
  iconKey: 'build',
  kind: 'action',
  titleKey: 'dashboard.previewWorkbench.quickActions.createBuild.title',
  titleFallback: 'Create Build',
  descriptionKey: 'dashboard.previewWorkbench.quickActions.createBuild.description',
  descriptionFallback: 'Create a build job',
  route: '/build/create',
  showOnHome: true,
};

describe('WorkbenchQuickActionItem', () => {
  it('renders the shared icon, title, description, and full-button activation target', async () => {
    const wrapper = mount(WorkbenchQuickActionItem, { props: { action } });

    expect(wrapper.get('[data-icon-key="build"]').attributes('data-icon-key')).toBe('build');
    expect(wrapper.text()).toContain(action.titleFallback);
    expect(wrapper.text()).toContain(action.descriptionFallback);

    await wrapper.get('button').trigger('click');

    expect(wrapper.emitted('activate')?.[0]).toEqual([action]);
  });

  it('keeps an action indicator only for executable featured drawer actions', async () => {
    const wrapper = mount(WorkbenchQuickActionItem, { props: { action, variant: 'drawer-featured' } });

    expect(wrapper.get('button').classes()).toContain('workbench-quick-action-item--drawer-featured');
    expect(wrapper.find('.workbench-quick-action-item__arrow').exists()).toBe(true);

    await wrapper.setProps({ action: { ...action, kind: 'navigation' } });

    expect(wrapper.find('.workbench-quick-action-item__arrow').exists()).toBe(false);
  });
});
