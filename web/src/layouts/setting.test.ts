import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  createScrollEdgeActionsContext,
  scrollEdgeActionsContextKey,
  type ScrollEdgeActionsController,
} from '@/shared/composables';
import { useSettingStore } from '@/store';

import SettingHost from './setting.vue';

vi.mock('@/utils/color', () => ({
  composeThemeTokenMap: (tokens: Record<string, string>) => tokens,
  generateBrandColorMap: (brandTheme: string) => ({ '--td-brand-color': brandTheme }),
  insertThemeStylesheet: vi.fn(),
  syncFaviconColor: vi.fn(),
}));

const mountHost = (context = createScrollEdgeActionsContext()) =>
  mount(SettingHost, {
    global: {
      provide: {
        [scrollEdgeActionsContextKey as symbol]: context,
      },
      stubs: {
        ScrollEdgeActions: {
          props: ['controller', 'labels'],
          template:
            '<div data-testid="scroll-edge-actions" :data-top="labels.toTop" :data-bottom="labels.toBottom" :data-group="labels.group" />',
        },
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

  it('mounts the scroll host beside the personalization entry when a controller is active', async () => {
    const context = createScrollEdgeActionsContext();
    const wrapper = mountHost(context);
    const unregister = context.register({} as ScrollEdgeActionsController);

    await wrapper.vm.$nextTick();

    const scrollActions = wrapper.get('[data-testid="scroll-edge-actions"]');
    expect(scrollActions.attributes('data-top')).toBe('到顶部');
    expect(scrollActions.attributes('data-bottom')).toBe('到底部');
    expect(scrollActions.attributes('data-group')).toBeUndefined();
    expect(
      scrollActions.element.compareDocumentPosition(wrapper.get('[data-testid="theme-workbench-panel"]').element),
    ).toBe(Node.DOCUMENT_POSITION_FOLLOWING);

    unregister();
    await wrapper.vm.$nextTick();
    expect(wrapper.find('[data-testid="scroll-edge-actions"]').exists()).toBe(false);
    wrapper.unmount();
  });
});
