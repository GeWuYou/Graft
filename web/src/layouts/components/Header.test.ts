import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { useSettingStore } from '@/store';

import Header from './Header.vue';

vi.mock('@/utils/color', () => ({
  composeThemeTokenMap: (tokens: Record<string, string>) => tokens,
  generateBrandColorMap: (brandTheme: string) => ({ '--td-brand-color': brandTheme }),
  insertThemeStylesheet: vi.fn(),
}));

vi.mock('tdesign-icons-vue-next', () => ({
  ChevronDownIcon: defineComponent({ template: '<i />' }),
  PaletteIcon: defineComponent({ name: 'PaletteIcon', template: '<i data-testid="palette-icon" />' }),
  PoweroffIcon: defineComponent({ template: '<i />' }),
  UserCircleIcon: defineComponent({ template: '<i />' }),
}));

vi.mock('@/config/global', () => ({ prefix: 'graft' }));
vi.mock('@/layouts/useShellNavigation', () => ({ useShellNavigation: () => ({ goHome: vi.fn() }) }));
vi.mock('@/locales', () => ({
  i18n: { global: { getLocaleMessage: () => ({}) } },
  t: (key: string) => key,
}));
vi.mock('@/modules/auth/store', () => ({
  useAuthSessionStore: () => ({ logout: vi.fn(), userInfo: { name: 'Graft Admin' } }),
}));
vi.mock('@/router', () => ({ getActive: () => '' }));

const headMenuStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', [slots.logo?.(), slots.default?.(), slots.operations?.()]);
  },
});
const buttonStub = defineComponent({
  emits: ['click'],
  setup(_, { emit, slots }) {
    return () => h('button', { type: 'button', onClick: () => emit('click') }, [slots.icon?.(), slots.default?.()]);
  },
});
const tooltipStub = defineComponent({
  props: { content: { type: String, required: false, default: '' } },
  setup(props, { slots }) {
    return () => h('div', { 'data-tooltip-content': props.content }, slots.default?.());
  },
});

describe('Header', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it('uses the personalization tooltip and palette icon to open the workbench', async () => {
    const store = useSettingStore();
    const openWorkbench = vi.spyOn(store, 'openThemeWorkbench');
    const wrapper = mount(Header, {
      props: { layout: 'side' },
      global: {
        stubs: {
          't-head-menu': headMenuStub,
          't-button': buttonStub,
          't-dropdown': { template: '<div><slot /><slot name="dropdown" /></div>' },
          't-dropdown-item': { template: '<button type="button"><slot /></button>' },
          't-icon': { template: '<i />' },
          't-tooltip': tooltipStub,
          BrandIdentity: true,
          LanguageSwitcher: true,
          MenuContent: true,
          Notice: true,
          Search: true,
        },
      },
    });

    const personalizationEntry = wrapper.get('[data-tooltip-content="layout.header.personalization"]');
    expect(personalizationEntry.find('[data-testid="palette-icon"]').exists()).toBe(true);

    await personalizationEntry.get('button').trigger('click');
    expect(openWorkbench).toHaveBeenCalledWith('overview');
  });
});
