import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import { useSettingStore } from '@/store';

import Header from './Header.vue';

vi.mock('@/utils/color', () => ({
  composeThemeTokenMap: (tokens: Record<string, string>) => tokens,
  generateBrandColorMap: (brandTheme: string) => ({ '--td-brand-color': brandTheme }),
  insertThemeStylesheet: vi.fn(),
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
        't-dropdown': { template: '<div><slot /><slot name="dropdown" /></div>' },
        't-dialog': {
          props: { visible: { type: Boolean, default: false } },
          template: '<section data-testid="language-dialog" :data-visible="String(visible)"><slot /></section>',
        },
        't-dropdown-item': { template: '<button v-bind="$attrs" type="button"><slot /></button>' },
        't-icon': { template: '<i />' },
        't-radio': true,
        't-radio-group': true,
        't-tooltip': tooltipStub,
        BrandIdentity: true,
        LanguageSwitcher: true,
        MenuContent: true,
        Notice: true,
        Search: defineComponent({ template: '<div data-testid="header-search" />' }),
      },
    },
  });
}

describe('Header', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  afterEach(() => {
    restoreFullscreenApi();
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
    expect(wrapper.find('.header-menu').exists()).toBe(false);
    expect(wrapper.find('[data-testid="header-document-fullscreen-toggle"]').exists()).toBe(false);
    expect(wrapper.get('[data-testid="header-more-tools"]').attributes('aria-label')).toBe('layout.header.moreTools');
    expect(wrapper.find('.header-more-tools-search [data-testid="header-search"]').exists()).toBe(true);
    expect(wrapper.find('.header-user-account').exists()).toBe(false);
    expect(wrapper.get('[data-testid="header-user-menu"]').classes()).toContain('header-user-btn--compact');

    expect(wrapper.get('[data-testid="language-dialog"]').attributes('data-visible')).toBe('false');
    await wrapper.get('[data-testid="header-language-selector"]').trigger('click');
    expect(wrapper.get('[data-testid="language-dialog"]').attributes('data-visible')).toBe('true');

    await wrapper.get('[data-testid="header-navigation-toggle"]').trigger('click');
    expect(wrapper.emitted('open-navigation')).toHaveLength(1);
    wrapper.unmount();
  });

  it('renders the narrow-screen navigation trigger for drawer navigation', () => {
    const wrapper = mountHeader({ navigationPresentation: 'drawer' });

    expect(wrapper.find('[data-testid="header-navigation-toggle"]').exists()).toBe(true);
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
