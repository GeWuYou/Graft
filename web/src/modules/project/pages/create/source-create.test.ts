import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import ProjectSourceCreate from './source-create.vue';

const mocks = vi.hoisted(() => ({
  push: vi.fn(),
  replace: vi.fn(),
  postProjectCreateTemplate: vi.fn(),
  navigateToProjectCreateSource: vi.fn(),
}));

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: { deployment: 'compose', runtime_target_id: '7' } }),
  useRouter: () => ({ push: mocks.push, replace: mocks.replace }),
}));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('tdesign-vue-next', () => ({ MessagePlugin: { error: vi.fn(), success: vi.fn(), warning: vi.fn() } }));
vi.mock('../../api/project', () => ({ postProjectCreateTemplate: mocks.postProjectCreateTemplate }));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('../../shared/navigation', () => ({
  navigateToProjectCreateSource: mocks.navigateToProjectCreateSource,
  refreshProjectCreatePage: mocks.replace,
}));

describe('ProjectSourceCreate', () => {
  it('returns to source selection with deployment and runtime target context', async () => {
    const wrapper = mount(ProjectSourceCreate, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot name="actions" /></header>' },
          't-button': { inheritAttrs: false, template: '<button v-bind="$attrs"><slot /></button>' },
          't-space': { template: '<div><slot /></div>' },
          't-card': { template: '<section><slot /></section>' },
          't-form': { template: '<form><slot /></form>' },
          't-form-item': { template: '<label><slot /></label>' },
          't-input': true,
          't-select': true,
        },
      },
    });
    await flushPromises();

    await wrapper.get('button').trigger('click');

    expect(mocks.navigateToProjectCreateSource).toHaveBeenCalledWith(expect.anything(), {
      deployment: 'compose',
      runtime_target_id: '7',
    });
  });
});
