import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h, nextTick, provide } from 'vue';

import { createScrollEdgeActionsContext, scrollEdgeActionsContextKey } from '@/shared/composables';

import PageContainer from './PageContainer.vue';

describe('PageContainer scroll registration', () => {
  it('registers its scroll viewport and unregisters it on unmount', async () => {
    const context = createScrollEdgeActionsContext();
    const Host = defineComponent({
      setup() {
        provide(scrollEdgeActionsContextKey, context);
        return () => h(PageContainer, { showFooter: false });
      },
    });

    const wrapper = mount(Host, {
      global: {
        stubs: {
          Breadcrumb: true,
          LFooter: true,
        },
      },
    });

    await nextTick();
    expect(context.activeController.value).not.toBeNull();
    expect(context.activeController.value?.metrics.value.target).toBe(wrapper.find('section').element);

    wrapper.unmount();
    expect(context.activeController.value).toBeNull();
  });
});
