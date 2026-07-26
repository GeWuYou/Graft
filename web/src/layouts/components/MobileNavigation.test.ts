import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, reactive } from 'vue';

import MobileNavigation from './MobileNavigation.vue';

const pushMock = vi.fn();
const route = reactive({ fullPath: '/applications/projects/alpha', path: '/applications/projects/alpha' });

beforeEach(() => {
  route.fullPath = '/applications/projects/alpha';
  route.path = '/applications/projects/alpha';
  pushMock.mockReset();
});

vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => ({ push: pushMock }),
}));

vi.mock('@/locales', () => ({ t: (key: string) => key }));
vi.mock('@/locales/useLocale', () => ({ useLocale: () => ({ locale: { value: 'zh-CN' } }) }));
vi.mock('@/store', () => ({
  useSettingStore: () => ({ displaySideMode: 'light', layout: 'side' }),
}));
vi.mock('@/shared/icons/MenuIcon.vue', () => ({
  default: defineComponent({
    props: { iconKey: { type: String, default: '' } },
    setup(props) {
      return () => h('i', { 'data-icon-key': props.iconKey });
    },
  }),
}));
vi.mock('./SideNav.vue', () => ({
  default: defineComponent({
    props: { menu: { type: Array, default: () => [] } },
    setup(props) {
      return () => h('div', { 'data-menu-count': String(props.menu.length) });
    },
  }),
}));

const buttonStub = defineComponent({
  emits: ['click'],
  setup(_, { attrs, emit, slots }) {
    return () =>
      h('button', { ...attrs, type: 'button', onClick: () => emit('click') }, [slots.icon?.(), slots.default?.()]);
  },
});

const drawerStub = defineComponent({
  props: {
    footer: { type: Boolean, default: true },
    placement: { type: String, default: 'right' },
    visible: { type: Boolean, default: false },
  },
  emits: ['update:visible'],
  setup(props, { slots }) {
    return () =>
      h(
        'section',
        {
          'data-drawer-footer': String(props.footer),
          'data-drawer-placement': props.placement,
          'data-drawer-visible': String(props.visible),
        },
        slots.default?.(),
      );
  },
});

function mountNavigation() {
  return mount(MobileNavigation, {
    props: {
      menu: [
        {
          path: 'applications',
          meta: { icon: 'application-portfolio', orderNo: 20, title: { 'en-US': 'Applications', 'zh-CN': '应用' } },
          children: [
            { path: 'projects', meta: { title: { 'en-US': 'Projects', 'zh-CN': '项目' } } },
            { path: 'templates', meta: { title: { 'en-US': 'Templates', 'zh-CN': '应用模板' } } },
          ],
        },
        {
          path: 'hidden',
          meta: { hidden: true, orderNo: 10, title: { 'en-US': 'Hidden', 'zh-CN': '隐藏' } },
        },
        { path: '/security', meta: { icon: 'security', orderNo: 30, title: { 'en-US': 'Security', 'zh-CN': '安全' } } },
        {
          path: '/observability',
          meta: { icon: 'chart', orderNo: 40, title: { 'en-US': 'Observability', 'zh-CN': '可观测性' } },
        },
        { path: '/platform', meta: { icon: 'setting', orderNo: 50, title: { 'en-US': 'Platform', 'zh-CN': '平台' } } },
      ],
      visible: false,
    },
    global: {
      stubs: {
        't-button': buttonStub,
        't-drawer': drawerStub,
        't-icon': true,
      },
    },
  });
}

describe('MobileNavigation', () => {
  it('derives four visible shortcuts in bootstrap order and resolves grouped routes', async () => {
    route.fullPath = '/applications/projects/alpha';
    route.path = '/applications/projects/alpha';
    const wrapper = mountNavigation();
    const shortcuts = wrapper.findAll('.graft-mobile-navigation__item');

    expect(shortcuts).toHaveLength(4);
    expect(shortcuts.map((item) => item.text())).toEqual(['应用', '安全', '可观测性', '平台']);
    expect(shortcuts[0]?.attributes('aria-current')).toBe('page');

    await shortcuts[0]?.trigger('click');
    expect(pushMock).toHaveBeenLastCalledWith('/applications/projects');
  });

  it('keeps Application Templates inside the Application shortcut instead of creating a top-level shortcut', () => {
    const wrapper = mountNavigation();

    expect(wrapper.findAll('.graft-mobile-navigation__item').map((item) => item.text())).not.toContain('应用模板');
    expect(wrapper.get('[data-menu-count]').attributes('data-menu-count')).toBe('5');
  });

  it('opens the full menu sheet from the permanent center action', async () => {
    const wrapper = mountNavigation();

    expect(wrapper.get('[data-drawer-placement]').attributes('data-drawer-placement')).toBe('bottom');
    expect(wrapper.get('[data-drawer-footer]').attributes('data-drawer-footer')).toBe('false');
    expect(wrapper.get('[data-menu-count]').attributes('data-menu-count')).toBe('5');

    await wrapper.get('.graft-mobile-navigation__all').trigger('click');
    expect(wrapper.emitted('update:visible')).toContainEqual([true]);
  });

  it('closes the sheet after a route change', async () => {
    const wrapper = mountNavigation();
    route.fullPath = '/security/users';
    route.path = '/security/users';
    await nextTick();

    expect(wrapper.emitted('update:visible')).toContainEqual([false]);
  });
});
