import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useSettingStore } from '@/store';

import SettingHost from './setting.vue';

vi.mock('@/utils/color', () => ({
  composeThemeTokenMap: (tokens: Record<string, string>) => tokens,
  generateBrandColorMap: (brandTheme: string) => ({ '--td-brand-color': brandTheme }),
  insertThemeStylesheet: vi.fn(),
  syncFaviconColor: vi.fn(),
}));

const mountHost = () =>
  mount(SettingHost, {
    global: {
      stubs: {
        ThemeWorkbenchDock: { template: '<div data-testid="theme-workbench-dock" />' },
        ThemeWorkbenchPanel: { template: '<div data-testid="theme-workbench-panel" />' },
      },
    },
  });

describe('theme workbench host', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it('renders the floating entry on every route when enabled', async () => {
    const store = useSettingStore();
    const wrapper = mountHost();

    expect(wrapper.find('[data-testid="theme-workbench-dock"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="theme-workbench-panel"]').exists()).toBe(true);

    store.updateConfig({ showThemeWorkbenchDock: true });
    await wrapper.vm.$nextTick();
    expect(wrapper.find('[data-testid="theme-workbench-dock"]').exists()).toBe(true);
  });
});
