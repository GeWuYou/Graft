import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, reactive } from 'vue';

import Header from './Header.vue';

const settingStore = reactive({
  isSidebarCompact: false,
  menuAlwaysExpanded: false,
  openThemeWorkbench: vi.fn(),
  updateConfig: vi.fn(),
});

vi.mock('@/store', () => ({
  useSettingStore: () => settingStore,
}));

vi.mock('@/router', () => ({
  getActive: () => '/infrastructure/docker/containers',
}));

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

vi.mock('@/layouts/useShellNavigation', () => ({
  useShellNavigation: () => ({ goHome: vi.fn() }),
}));

vi.mock('@/locales', () => ({
  t: (key: string) => key,
}));

vi.mock('@/modules/auth/store', () => ({
  useAuthSessionStore: () => ({ userInfo: { name: 'Admin' }, logout: vi.fn() }),
}));

vi.mock('@/shared/components/brand', () => ({
  BrandIdentity: defineComponent({ name: 'BrandIdentityStub', setup: () => () => h('span') }),
}));

vi.mock('./MenuContent.vue', () => ({
  default: defineComponent({ name: 'MenuContentStub', setup: () => () => h('span') }),
}));

vi.mock('./Notice.vue', () => ({
  default: defineComponent({ name: 'NoticeStub', setup: () => () => h('span') }),
}));

vi.mock('./Search.vue', () => ({
  default: defineComponent({ name: 'SearchStub', setup: () => () => h('span') }),
}));

vi.mock('@/shared/components/LanguageSwitcher.vue', () => ({
  default: defineComponent({ name: 'LanguageSwitcherStub', setup: () => () => h('span') }),
}));

const headMenuStub = defineComponent({
  name: 'THeadMenuStub',
  props: {
    expanded: { type: Array, default: () => [] },
  },
  emits: ['expand'],
  setup(props, { slots }) {
    return () => h('div', { 'data-menu-expanded': JSON.stringify(props.expanded) }, [slots.default?.()]);
  },
});

describe('Header', () => {
  beforeEach(() => {
    settingStore.menuAlwaysExpanded = false;
    settingStore.openThemeWorkbench.mockReset();
    settingStore.updateConfig.mockReset();
  });

  it('expands all descendants of the hovered top-level menu when enabled', async () => {
    const wrapper = mount(Header, {
      props: {
        layout: 'top',
        menu: [
          {
            path: 'infrastructure',
            children: [
              {
                path: 'docker',
                children: [{ path: 'containers' }],
              },
            ],
          },
          {
            path: 'security',
            children: [
              {
                path: 'audit',
                children: [{ path: 'access' }],
              },
            ],
          },
        ],
      },
      global: {
        stubs: {
          't-head-menu': headMenuStub,
          't-button': true,
          't-dropdown': true,
          't-dropdown-item': true,
          't-icon': true,
          't-tooltip': true,
        },
      },
    });

    const menu = wrapper.findComponent(headMenuStub);
    menu.vm.$emit('expand', ['infrastructure']);
    await wrapper.vm.$nextTick();
    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('["infrastructure"]');

    settingStore.menuAlwaysExpanded = true;
    await wrapper.vm.$nextTick();
    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('["infrastructure","docker"]');

    menu.vm.$emit('expand', ['security']);
    await wrapper.vm.$nextTick();
    expect(wrapper.get('[data-menu-expanded]').attributes('data-menu-expanded')).toBe('["security","audit"]');
  });
});
