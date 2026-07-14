import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import ProjectCreateSourceIndex from './source-index.vue';

const mocks = vi.hoisted(() => ({
  getProjectCreationMethods: vi.fn(),
  push: vi.fn(),
  replace: vi.fn(),
}));

vi.mock('../../api/project', () => ({ getProjectCreationMethods: mocks.getProjectCreationMethods }));
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: { deployment: 'compose', runtime_target_id: '7' } }),
  useRouter: () => ({ push: mocks.push, replace: mocks.replace }),
}));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('../../shared/navigation', () => ({
  useProjectCreateRouteNavigation: () => (target: unknown) => mocks.push(target),
}));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

describe('ProjectCreateSourceIndex', () => {
  it('returns to runtime target selection with the current query', async () => {
    mocks.getProjectCreationMethods.mockResolvedValue({ items: [] });
    const wrapper = mount(ProjectCreateSourceIndex, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot name="actions" /></header>' },
          't-button': { inheritAttrs: false, template: '<button v-bind="$attrs"><slot /></button>' },
          't-alert': true,
          't-card': { template: '<section><slot /></section>' },
          't-collapse': { template: '<div><slot /></div>' },
          't-collapse-panel': { template: '<div><slot /></div>' },
          't-space': { template: '<div><slot /></div>' },
          't-tag': { template: '<span><slot /></span>' },
          't-tooltip': { template: '<div><slot /></div>' },
          'project-back-icon': true,
        },
      },
    });
    await flushPromises();

    await wrapper.get('[data-testid="project-creation-back"]').trigger('click');

    expect(mocks.push).toHaveBeenCalledWith({
      name: 'ProjectCreateRuntimeTargetIndex',
      query: { deployment: 'compose', runtime_target_id: '7' },
    });
  });

  it('keeps unavailable creation methods non-actionable', async () => {
    mocks.push.mockClear();
    mocks.getProjectCreationMethods.mockResolvedValue({
      items: [
        {
          method: 'blank',
          availability: 'blocked',
          blocked_reason: 'managed_root_invalid',
        },
      ],
    });
    const wrapper = mount(ProjectCreateSourceIndex, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot name="actions" /></header>' },
          't-button': { inheritAttrs: false, template: '<button v-bind="$attrs"><slot /></button>' },
          't-alert': true,
          't-card': { template: '<section><slot /></section>' },
          't-collapse': { template: '<div><slot /></div>' },
          't-collapse-panel': { template: '<div><slot /></div>' },
          't-space': { template: '<div><slot /></div>' },
          't-tag': { template: '<span><slot /></span>' },
          't-tooltip': { template: '<div><slot /></div>' },
          'project-back-icon': true,
        },
      },
    });
    await flushPromises();

    await wrapper.get('[data-testid="project-creation-method-blank"]').trigger('click');

    expect(mocks.push).not.toHaveBeenCalled();
  });
});
