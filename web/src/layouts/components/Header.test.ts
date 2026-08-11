import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import { useSettingStore } from '@/store';

import Header from './Header.vue';

const headerSource = readFileSync(join(process.cwd(), 'src/layouts/components/Header.vue'), 'utf8');
const shellSource = readFileSync(join(process.cwd(), 'src/layouts/index.vue'), 'utf8');

vi.mock('@/utils/color', () => ({
  composeThemeTokenMap: (tokens: Record<string, string>) => tokens,
  generateBrandColorMap: (brandTheme: string) => ({ '--td-brand-color': brandTheme }),
  insertThemeStylesheet: vi.fn(),
  syncFaviconColor: vi.fn(),
}));

vi.mock('tdesign-icons-vue-next', () => ({
  ChevronDownIcon: defineComponent({ template: '<i />' }),
  EllipsisIcon: defineComponent({ template: '<i data-testid="ellipsis-icon" />' }),
  FullscreenExitIcon: defineComponent({ template: '<i data-testid="fullscreen-exit-icon" />' }),
  FullscreenIcon: defineComponent({ template: '<i data-testid="fullscreen-icon" />' }),
  PaletteIcon: defineComponent({ name: 'PaletteIcon', template: '<i data-testid="palette-icon" />' }),
  PoweroffIcon: defineComponent({ template: '<i />' }),
  UserCircleIcon: defineComponent({ template: '<i />' }),
}));

vi.mock('@/config/global', () => ({ prefix: 'graft' }));
vi.mock('@/layouts/useShellNavigation', () => ({ useShellNavigation: () => ({ goHome: vi.fn() }) }));
vi.mock('@/locales', () => ({
  i18n: { global: { getLocaleMessage: () => ({}) } },
  languageList: [
    { content: '简体中文', value: 'zh-CN' },
    { content: 'English', value: 'en-US' },
  ],
  t: (key: string) => key,
}));
vi.mock('@/locales/useLocale', () => ({
  useLocale: () => ({ changeLocale: vi.fn(), locale: ref('zh-CN') }),
}));
vi.mock('@/modules/auth/store', () => ({
  useAuthSessionStore: () => ({ logout: vi.fn(), userInfo: { name: 'Graft Admin' } }),
}));
vi.mock('@/modules/update', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/modules/update')>()),
  updateVersionEntry: defineComponent({ template: '<span data-testid="header-version-entry">version</span>' }),
}));
vi.mock('@/router', () => ({ getActive: () => '', useRouter: () => ({ push: vi.fn() }) }));

const headMenuStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', [slots.logo?.(), slots.default?.(), slots.operations?.()]);
  },
});
const buttonStub = defineComponent({
  props: {
    disabled: { type: Boolean, default: false },
  },
  emits: ['click'],
  setup(props, { attrs, emit, slots }) {
    return () =>
      h('button', { ...attrs, disabled: props.disabled, type: 'button', onClick: () => emit('click') }, [
        slots.icon?.(),
        slots.default?.(),
      ]);
  },
});
const tooltipStub = defineComponent({
  props: { content: { type: String, required: false, default: '' } },
  setup(props, { slots }) {
    return () => h('div', { 'data-tooltip-content': props.content }, slots.default?.());
  },
});
const dropdownStub = defineComponent({
  setup(_, { slots }) {
    const visible = ref(false);
    return () =>
      h('div', { 'data-testid': 'dropdown' }, [
        h('div', { onClick: () => (visible.value = true) }, slots.default?.()),
        visible.value ? h('div', { 'data-testid': 'dropdown-panel' }, slots.dropdown?.()) : null,
      ]);
  },
});
const languageSwitcherOpenMock = vi.fn();
const languageSwitcherStub = defineComponent({
  setup(_, { expose }) {
    expose({ open: languageSwitcherOpenMock });
    return () => h('div', { 'data-testid': 'language-switcher' });
  },
});

const originalFullscreenElement = Object.getOwnPropertyDescriptor(document, 'fullscreenElement');
const originalExitFullscreen = Object.getOwnPropertyDescriptor(document, 'exitFullscreen');
const originalRequestFullscreen = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'requestFullscreen');
let fullscreenElement: Element | null = null;

function restoreFullscreenApi() {
  fullscreenElement = null;

  if (originalFullscreenElement) {
    Object.defineProperty(document, 'fullscreenElement', originalFullscreenElement);
  } else {
    Reflect.deleteProperty(document, 'fullscreenElement');
  }

  if (originalExitFullscreen) {
    Object.defineProperty(document, 'exitFullscreen', originalExitFullscreen);
  } else {
    Reflect.deleteProperty(document, 'exitFullscreen');
  }

  if (originalRequestFullscreen) {
    Object.defineProperty(HTMLElement.prototype, 'requestFullscreen', originalRequestFullscreen);
  } else {
    Reflect.deleteProperty(HTMLElement.prototype, 'requestFullscreen');
  }
}

function installFullscreenApi() {
  Object.defineProperty(document, 'fullscreenElement', {
    configurable: true,
    get: () => fullscreenElement,
  });
  Object.defineProperty(document, 'exitFullscreen', {
    configurable: true,
    value: vi.fn(async () => {
      fullscreenElement = null;
      document.dispatchEvent(new Event('fullscreenchange'));
    }),
  });
  Object.defineProperty(HTMLElement.prototype, 'requestFullscreen', {
    configurable: true,
    value: vi.fn(async () => {
      fullscreenElement = document.documentElement;
      document.dispatchEvent(new Event('fullscreenchange'));
    }),
  });
}

function installUnsupportedFullscreenApi() {
  Object.defineProperty(document, 'exitFullscreen', {
    configurable: true,
    value: undefined,
  });
  Object.defineProperty(HTMLElement.prototype, 'requestFullscreen', {
    configurable: true,
    value: undefined,
  });
}

function mountHeader(props: Record<string, unknown> = {}) {
  return mount(Header, {
    props: { layout: 'side', ...props },
    global: {
      stubs: {
        't-head-menu': headMenuStub,
        't-button': buttonStub,
        't-dropdown': dropdownStub,
        't-dropdown-item': { template: '<button v-bind="$attrs" type="button"><slot /></button>' },
        't-icon': { template: '<i />' },
        't-tooltip': tooltipStub,
        BrandIdentity: true,
        LanguageSwitcher: languageSwitcherStub,
        MenuContent: true,
        Notice: true,
        Search: defineComponent({ template: '<input data-testid="header-search" />' }),
      },
    },
  });
}

describe('Header', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    languageSwitcherOpenMock.mockClear();
  });

  afterEach(() => {
    restoreFullscreenApi();
  });

  it('keeps the operations surface transparent so the shared header separator remains visible', () => {
    expect(headerSource).toContain(':deep(.t-head-menu__inner) {');
    expect(headerSource).toContain('border-bottom: 1px solid var(--graft-shell-border-color);');
    expect(headerSource).toContain('.t-menu--dark {\n  background: var(--graft-shell-header-bg);');
    expect(headerSource).toContain(
      ':deep(.t-head-menu__inner),\n  :deep(.t-menu__logo),\n  :deep(.t-menu),\n  .header-operate-left,',
    );
    expect(headerSource).toContain(':deep(.t-head-menu__operations) {\n    background: transparent;');
    expect(headerSource).not.toContain('box-shadow: inset 0 -1px 0 var(--graft-shell-border-color);');
    expect(headerSource).not.toContain(':deep(.t-head-menu) {\n  position: relative;');
    expect(headerSource).not.toContain(':deep(.t-menu__operations),\n  .header-operate-left,');
    expect(shellSource).not.toContain('border-bottom: 1px solid var(--graft-shell-border-color);');
  });

  it('uses the personalization tooltip and palette icon to open the workbench', async () => {
    const store = useSettingStore();
    const openWorkbench = vi.spyOn(store, 'openThemeWorkbench');
    const wrapper = mountHeader();

    const personalizationEntry = wrapper.get('[data-tooltip-content="layout.header.personalization"]');
    expect(personalizationEntry.find('[data-testid="palette-icon"]').exists()).toBe(true);

    await personalizationEntry.get('button').trigger('click');
    expect(openWorkbench).toHaveBeenCalledWith('overview');
    wrapper.unmount();
  });

  it('collects header tools into the drawer presentation while preserving navigation, notices, and account access', async () => {
    const wrapper = mountHeader({ layout: 'top', navigationPresentation: 'drawer', showLogo: true });

    expect(wrapper.get('[data-testid="header-navigation-toggle"]').attributes('aria-label')).toBe(
      'layout.header.openNavigation',
    );
    expect(wrapper.find('.header-logo-container').exists()).toBe(false);
    expect(wrapper.find('[data-testid="header-version-entry"]').exists()).toBe(false);
    expect(wrapper.find('.header-menu').exists()).toBe(false);
    expect(wrapper.find('[data-testid="header-document-fullscreen-toggle"]').exists()).toBe(false);
    expect(wrapper.get('[data-testid="header-more-tools"]').attributes('aria-label')).toBe('layout.header.moreTools');
    expect(wrapper.find('[data-testid="dropdown-panel"]').exists()).toBe(false);
    expect(wrapper.find('.header-user-account').exists()).toBe(false);
    expect(wrapper.get('[data-testid="header-user-menu"]').classes()).toContain('header-user-btn--compact');

    await wrapper.get('[data-testid="header-more-tools"]').trigger('click');
    await wrapper.vm.$nextTick();
    const search = wrapper.get('.header-more-tools-search [data-testid="header-search"]');
    await search.trigger('focus');
    await search.setValue('runtime target');
    expect((search.element as HTMLInputElement).value).toBe('runtime target');

    await wrapper.get('[data-testid="header-language-selector"]').trigger('click');
    expect(languageSwitcherOpenMock).toHaveBeenCalledTimes(1);

    await wrapper.get('[data-testid="header-navigation-toggle"]').trigger('click');
    expect(wrapper.emitted('open-navigation')).toHaveLength(1);
    wrapper.unmount();
  });

  it('renders the narrow-screen navigation trigger for drawer navigation', () => {
    const wrapper = mountHeader({ navigationPresentation: 'drawer' });

    expect(wrapper.find('[data-testid="header-navigation-toggle"]').exists()).toBe(true);
    wrapper.unmount();
  });

  it.each(['top', 'mix'])('renders the version entry beside the desktop %s layout brand', (layout) => {
    const wrapper = mountHeader({ layout, navigationPresentation: 'desktop' });

    expect(wrapper.find('.header-brand-container .header-logo-container').exists()).toBe(true);
    expect(wrapper.find('.header-brand-container [data-testid="header-version-entry"]').exists()).toBe(true);
    wrapper.unmount();
  });

  it('keeps the version entry out of the side layout header because the side navigation owns it', () => {
    const wrapper = mountHeader({ layout: 'side', navigationPresentation: 'desktop' });

    expect(wrapper.find('[data-testid="header-version-entry"]').exists()).toBe(false);
    wrapper.unmount();
  });

  it('toggles document fullscreen through its operation button and browser keyboard chords', async () => {
    installFullscreenApi();
    expect(typeof document.exitFullscreen).toBe('function');
    expect(typeof document.documentElement.requestFullscreen).toBe('function');
    const wrapper = mountHeader();
    const toggleButton = wrapper.get('[data-testid="header-document-fullscreen-toggle"]');

    try {
      await wrapper.vm.$nextTick();
      expect(toggleButton.attributes('disabled')).toBeUndefined();
      expect(wrapper.find('[data-tooltip-content="layout.header.enterFullscreen"]').exists()).toBe(true);

      await toggleButton.trigger('click');
      await flushPromises();
      expect(fullscreenElement).toBe(document.documentElement);
      expect(wrapper.find('[data-tooltip-content="layout.header.exitFullscreen"]').exists()).toBe(true);

      const f11Event = new KeyboardEvent('keydown', {
        bubbles: true,
        cancelable: true,
        code: 'F11',
        key: 'F11',
      });
      window.dispatchEvent(f11Event);
      await flushPromises();
      expect(f11Event.defaultPrevented).toBe(true);
      expect(fullscreenElement).toBeNull();

      const macFullscreenEvent = new KeyboardEvent('keydown', {
        bubbles: true,
        cancelable: true,
        code: 'KeyF',
        ctrlKey: true,
        key: 'f',
        metaKey: true,
      });
      window.dispatchEvent(macFullscreenEvent);
      await flushPromises();
      expect(macFullscreenEvent.defaultPrevented).toBe(true);
      expect(fullscreenElement).toBe(document.documentElement);
    } finally {
      wrapper.unmount();
    }
  });

  it('disables the document fullscreen operation when the browser API is unavailable', async () => {
    installUnsupportedFullscreenApi();
    expect(typeof document.exitFullscreen).not.toBe('function');
    expect(typeof document.documentElement.requestFullscreen).not.toBe('function');
    const wrapper = mountHeader();

    await wrapper.vm.$nextTick();
    expect(wrapper.get('[data-testid="header-document-fullscreen-toggle"]').attributes('disabled')).toBeDefined();
    wrapper.unmount();
  });
});
