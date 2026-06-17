// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, reactive } from 'vue';

import { LOCALE } from '@/contracts/i18n/locales';

import LayoutContent from './LayoutContent.vue';

const layoutStyleSource = readFileSync(join(process.cwd(), 'src/style/layout.less'), 'utf8');

type DropdownPopupProps = {
  onVisibleChange: (visible: boolean, context: { trigger: string }) => void;
  visible?: boolean;
};

const routeState = vi.hoisted(() => ({
  meta: {},
  path: '/server/runtime',
  fullPath: '/server/runtime',
}));

const routerMock = vi.hoisted(() => ({
  currentRoute: {
    value: routeState,
  },
  push: vi.fn(),
  replace: vi.fn(),
  resolve: vi.fn((target: { path?: string }) => ({
    href: target.path ?? '/',
  })),
}));

const storeState = vi.hoisted(() => ({
  settingStore: {
    isUseTabsRouter: true,
    showFooter: true,
  },
  tabsRouterStore: {
    activeTabKey: '/server/runtime',
    canReopenClosedTab: false,
    tabRouters: [] as Array<{
      fullPath?: string;
      isAlive?: boolean;
      isHome?: boolean;
      isPinned?: boolean;
      name?: string;
      path: string;
      query?: Record<string, string>;
      tabKey?: string;
      title?: Record<string, string>;
    }>,
    closeAllClosableTabs: vi.fn(),
    duplicateTab: vi.fn(),
    finishTabRefresh: vi.fn(),
    getNextRouteAfterClose: vi.fn(),
    reopenClosedTab: vi.fn(),
    resolveNavigationTarget: vi.fn((route?: { path?: string; query?: Record<string, string> }) =>
      route
        ? {
            path: route.path,
            query: route.query,
          }
        : null,
    ),
    setActiveTabKey: vi.fn((tabKey: string) => {
      storeState.tabsRouterStore.activeTabKey = tabKey;
    }),
    startTabRefresh: vi.fn(),
    subtractCurrentTabRouter: vi.fn(),
    subtractTabRouterAhead: vi.fn(),
    subtractTabRouterBehind: vi.fn(),
    subtractTabRouterOther: vi.fn(),
    togglePinnedTab: vi.fn(),
  },
}));

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => routerMock,
}));

vi.mock('@/locales', () => ({
  t: (key: string) => key,
}));

vi.mock('@/locales/useLocale', () => ({
  useLocale: () => ({
    locale: {
      value: LOCALE.ZH_CN,
    },
  }),
}));

vi.mock('@/shared/observability/copy', () => ({
  copyText: vi.fn(),
}));

vi.mock('@/store', async () => ({
  useSettingStore: () => reactive(storeState.settingStore),
  useTabsRouterStore: () => reactive(storeState.tabsRouterStore),
}));

const TDropdownStub = defineComponent({
  name: 'TDropdown',
  props: {
    hideAfterItemClick: {
      type: Boolean,
      default: false,
    },
    popupProps: {
      type: Object,
      default: () => ({}),
    },
  },
  setup(_, { slots }) {
    return () => h('div', { 'data-testid': 'tab-dropdown' }, [slots.default?.(), slots.dropdown?.()]);
  },
});

const TDropdownItemStub = defineComponent({
  name: 'TDropdownItem',
  props: {
    disabled: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          disabled: props.disabled,
          type: 'button',
          'data-testid': 'dropdown-item',
          onClick: () => {
            if (!props.disabled) {
              emit('click');
            }
          },
        },
        slots.default?.(),
      );
  },
});

const TDialogStub = defineComponent({
  name: 'TDialog',
  props: {
    attach: {
      type: String,
      default: '',
    },
    placement: {
      type: String,
      default: '',
    },
    visible: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['cancel', 'close', 'confirm', 'update:visible'],
  setup(props, { emit }) {
    return () =>
      props.visible
        ? h(
            'div',
            {
              'data-attach': props.attach,
              'data-placement': props.placement,
              'data-testid': 'close-all-dialog',
            },
            [
              h(
                'button',
                {
                  type: 'button',
                  'data-testid': 'close-all-confirm',
                  onClick: () => emit('confirm'),
                },
                'confirm',
              ),
              h(
                'button',
                {
                  type: 'button',
                  'data-testid': 'close-all-cancel',
                  onClick: () => emit('cancel'),
                },
                'cancel',
              ),
            ],
          )
        : null;
  },
});

const LContentStub = defineComponent({
  name: 'LContent',
  emits: ['page-surface-enter'],
  setup() {
    return () =>
      h('div', { class: 'route-view-host route-loading-host', 'data-testid': 'route-view-host' }, [
        h('div', { class: 'route-page-loading', 'data-testid': 'route-loading-host' }, [
          h('div', { 'data-testid': 'router-view-host' }, 'route content'),
        ]),
      ]);
  },
});

function createTab(path: string, name: string, isHome = false) {
  return {
    fullPath: path,
    isAlive: true,
    isHome,
    name,
    path,
    tabKey: path,
    title: {
      [LOCALE.ZH_CN]: name,
      [LOCALE.EN_US]: name,
    },
  };
}

function mountLayoutContent() {
  return mount(LayoutContent, {
    global: {
      stubs: {
        LContent: LContentStub,
        TContent: {
          template: '<main><slot /></main>',
        },
        TDialog: TDialogStub,
        TDropdown: TDropdownStub,
        TDropdownItem: TDropdownItemStub,
        TDropdownMenu: {
          template: '<div><slot /></div>',
        },
        TIcon: true,
        TLayout: {
          template: '<section><slot /></section>',
        },
        TFooter: {
          template: '<footer class="t-layout__footer"><slot /></footer>',
        },
        TTabPanel: {
          props: ['value'],
          template: '<div data-testid="tab-panel"><slot name="label" /></div>',
        },
        TTabs: {
          template: '<div data-testid="tabs"><slot /></div>',
        },
      },
    },
  });
}

async function openRuntimeTabMenu(wrapper: ReturnType<typeof mountLayoutContent>, tabIndex = 1) {
  const dropdown = wrapper.findAllComponents(TDropdownStub)[tabIndex];
  const popupProps = dropdown.vm.$props.popupProps as DropdownPopupProps;

  popupProps.onVisibleChange(true, { trigger: 'context-menu' });
  await nextTick();

  return dropdown;
}

async function clickCloseAll(wrapper: ReturnType<typeof mountLayoutContent>) {
  const closeAllItem = wrapper
    .findAll('[data-testid="dropdown-item"]')
    .find((item) => item.text().includes('layout.tagTabs.closeAll'));

  expect(closeAllItem).toBeTruthy();
  await closeAllItem!.trigger('click');
  await nextTick();
}

describe('LayoutContent', () => {
  beforeEach(() => {
    routeState.meta = {};
    routeState.path = '/server/runtime';
    routeState.fullPath = '/server/runtime';
    routerMock.currentRoute.value = routeState;
    routerMock.push.mockClear();
    routerMock.replace.mockClear();
    routerMock.resolve.mockClear();
    storeState.settingStore.isUseTabsRouter = true;
    storeState.settingStore.showFooter = true;
    storeState.tabsRouterStore.activeTabKey = '/server/runtime';
    storeState.tabsRouterStore.canReopenClosedTab = false;
    storeState.tabsRouterStore.tabRouters = [
      createTab('/', 'RootEntry', true),
      createTab('/server/runtime', 'ServerRuntime'),
      createTab('/audit/logs', 'AuditLogs'),
    ];
    storeState.tabsRouterStore.closeAllClosableTabs.mockImplementation(() => {
      storeState.tabsRouterStore.tabRouters = storeState.tabsRouterStore.tabRouters.filter(
        (tab) => tab.isHome || tab.isPinned,
      );
    });
    storeState.tabsRouterStore.duplicateTab.mockReset();
    storeState.tabsRouterStore.finishTabRefresh.mockReset();
    storeState.tabsRouterStore.getNextRouteAfterClose.mockReset();
    storeState.tabsRouterStore.reopenClosedTab.mockReset();
    storeState.tabsRouterStore.resolveNavigationTarget.mockClear();
    storeState.tabsRouterStore.setActiveTabKey.mockClear();
    storeState.tabsRouterStore.startTabRefresh.mockReset();
    storeState.tabsRouterStore.subtractCurrentTabRouter.mockReset();
    storeState.tabsRouterStore.subtractTabRouterAhead.mockReset();
    storeState.tabsRouterStore.subtractTabRouterBehind.mockReset();
    storeState.tabsRouterStore.subtractTabRouterOther.mockReset();
    storeState.tabsRouterStore.togglePinnedTab.mockReset();
  });

  it('opens the close-all dialog after the tab context menu is closed', async () => {
    const wrapper = mountLayoutContent();
    const dropdown = await openRuntimeTabMenu(wrapper);

    await clickCloseAll(wrapper);
    await nextTick();

    expect(dropdown.vm.$props.hideAfterItemClick).toBe(true);
    expect((dropdown.vm.$props.popupProps as DropdownPopupProps).visible).toBe(false);
    const dialog = wrapper.get('[data-testid="close-all-dialog"]');
    expect(dialog.attributes('data-attach')).toBe('body');
    expect(dialog.attributes('data-placement')).toBe('center');
    expect(storeState.tabsRouterStore.closeAllClosableTabs).not.toHaveBeenCalled();
  });

  it('keeps the leaving page surface until the entering view reports its surface', async () => {
    routeState.meta = {
      pageKind: 'list',
    };
    const wrapper = mountLayoutContent();

    expect(wrapper.get('.tdesign-starter-page-container').classes()).toContain(
      'tdesign-starter-page-container--paged-table',
    );

    routeState.meta = {
      pageSurface: 'form-detail',
    };
    await nextTick();

    expect(wrapper.get('.tdesign-starter-page-container').classes()).toContain(
      'tdesign-starter-page-container--paged-table',
    );

    wrapper.findComponent({ name: 'LContent' }).vm.$emit('page-surface-enter', 'form-detail');
    await nextTick();

    expect(wrapper.get('.tdesign-starter-page-container').classes()).toContain(
      'tdesign-starter-page-container--form-detail',
    );
  });

  it('renders non-cached route tabs even when their page instance is not alive', () => {
    storeState.tabsRouterStore.tabRouters = [
      createTab('/', 'RootEntry', true),
      {
        ...createTab('/ops/containers/container-1', 'ContainerDetail'),
        isAlive: false,
      },
    ];
    storeState.tabsRouterStore.activeTabKey = '/ops/containers/container-1';

    const wrapper = mountLayoutContent();

    expect(wrapper.findAll('[data-testid="tab-panel"]')).toHaveLength(2);
    expect(wrapper.text()).toContain('ContainerDetail');
  });

  it('keeps the page main surface from collapsing while route content transitions', () => {
    const wrapper = mountLayoutContent();
    const pageContainer = wrapper.get('.tdesign-starter-page-container');
    const pageMain = pageContainer.get('.tdesign-starter-page-container__main');
    const pageContent = pageMain.get('.tdesign-starter-page-container__content');
    const routeHost = pageContent.get('.route-view-host');
    const footer = pageContainer.get('.tdesign-starter-footer-layout');

    expect(pageContainer.classes()).toContain('page-scroll');
    expect(routeHost.element.compareDocumentPosition(footer.element) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(routeHost.element.parentElement).toBe(pageContent.element);
    expect(footer.element.parentElement).toBe(pageContainer.element);
    expect(routeHost.find('.tdesign-starter-footer-layout').exists()).toBe(false);
    expect(wrapper.get('[data-testid="route-loading-host"]').find('.tdesign-starter-footer-layout').exists()).toBe(
      false,
    );
    expect(wrapper.get('[data-testid="router-view-host"]').find('.tdesign-starter-footer-layout').exists()).toBe(false);

    expect(layoutStyleSource).toContain('&__main {');
    expect(layoutStyleSource).toContain('&__content {');
    expect(layoutStyleSource).toContain('&-footer-layout {');
    expect(layoutStyleSource).toContain('overflow: hidden auto;');
    expect(layoutStyleSource).not.toContain('overflow: auto hidden;');
    expect(layoutStyleSource).toContain('overflow: hidden;');
    expect(layoutStyleSource).toContain('flex: 1 0 auto;');
    expect(layoutStyleSource).toContain('min-height: 0;');
  });

  it('keeps close-all disabled when no tabs can be closed', async () => {
    storeState.tabsRouterStore.tabRouters = [createTab('/', 'RootEntry', true)];
    const wrapper = mountLayoutContent();
    await openRuntimeTabMenu(wrapper, 0);

    await clickCloseAll(wrapper);
    await nextTick();

    expect(wrapper.find('[data-testid="close-all-dialog"]').exists()).toBe(false);
    expect(storeState.tabsRouterStore.closeAllClosableTabs).not.toHaveBeenCalled();
  });

  it('does not create duplicate close-all dialogs for rapid consecutive clicks', async () => {
    const wrapper = mountLayoutContent();
    await openRuntimeTabMenu(wrapper);

    const closeAllItem = wrapper
      .findAll('[data-testid="dropdown-item"]')
      .find((item) => item.text().includes('layout.tagTabs.closeAll'));

    expect(closeAllItem).toBeTruthy();
    await closeAllItem!.trigger('click');
    await closeAllItem!.trigger('click');
    await nextTick();
    await nextTick();

    expect(wrapper.findAll('[data-testid="close-all-dialog"]')).toHaveLength(1);
    expect(storeState.tabsRouterStore.closeAllClosableTabs).not.toHaveBeenCalled();
  });

  it('does not reopen or close the close-all dialog when dropdown closes while the dialog is visible', async () => {
    const wrapper = mountLayoutContent();
    const dropdown = await openRuntimeTabMenu(wrapper);

    await clickCloseAll(wrapper);
    (dropdown.vm.$props.popupProps as DropdownPopupProps).onVisibleChange(false, { trigger: 'document' });
    await nextTick();

    expect(wrapper.findAll('[data-testid="close-all-dialog"]')).toHaveLength(1);
    expect(storeState.tabsRouterStore.closeAllClosableTabs).not.toHaveBeenCalled();
  });

  it('does not reopen the close-all dialog after cancel and a late dropdown close event', async () => {
    const wrapper = mountLayoutContent();
    const dropdown = await openRuntimeTabMenu(wrapper);

    await clickCloseAll(wrapper);
    await wrapper.get('[data-testid="close-all-cancel"]').trigger('click');
    (dropdown.vm.$props.popupProps as DropdownPopupProps).onVisibleChange(false, { trigger: 'document' });
    await nextTick();

    expect(wrapper.find('[data-testid="close-all-dialog"]').exists()).toBe(false);
    expect(storeState.tabsRouterStore.closeAllClosableTabs).not.toHaveBeenCalled();
  });

  it('closes all closable tabs only after dialog confirmation', async () => {
    const wrapper = mountLayoutContent();
    await openRuntimeTabMenu(wrapper);

    await clickCloseAll(wrapper);
    await wrapper.get('[data-testid="close-all-confirm"]').trigger('click');
    await nextTick();

    expect(storeState.tabsRouterStore.closeAllClosableTabs).toHaveBeenCalledTimes(1);
    expect(storeState.tabsRouterStore.setActiveTabKey).toHaveBeenCalledWith('/');
    expect(routerMock.push).toHaveBeenCalledWith({
      path: '/',
      query: undefined,
    });
    expect(wrapper.find('[data-testid="close-all-dialog"]').exists()).toBe(false);
  });
});
