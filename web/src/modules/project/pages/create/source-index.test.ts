import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import ApplicationCreateSourceIndex from './source-index.vue';
import sourceText from './source-index.vue?raw';

const mocks = vi.hoisted(() => ({
  getApplicationCreationMethods: vi.fn(),
  push: vi.fn(),
  replace: vi.fn(),
}));

vi.mock('../../api/project', () => ({ getApplicationCreationMethods: mocks.getApplicationCreationMethods }));
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: { deployment: 'compose', runtime_target_id: '7' } }),
  useRouter: () => ({ push: mocks.push, replace: mocks.replace }),
}));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('../../shared/navigation', () => ({
  useApplicationCreateRouteNavigation: () => (target: unknown) => mocks.push(target),
}));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

describe('ApplicationCreateSourceIndex', () => {
  it('keeps creation card actions anchored to the bottom of equal-height cards', () => {
    expect(sourceText).toContain('.project-creation-card :deep(.t-card__body)');
    expect(sourceText).toContain('flex: 1;');
    expect(sourceText).toContain('.project-creation-card__body :deep(.t-space-item:last-child)');
    expect(sourceText).toContain('margin-top: auto;');
  });

  it('returns to runtime target selection with the current query', async () => {
    mocks.getApplicationCreationMethods.mockResolvedValue({ items: [] });
    const wrapper = mount(ApplicationCreateSourceIndex, {
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
      name: 'ApplicationCreateRuntimeTargetIndex',
      query: { deployment: 'compose', runtime_target_id: '7' },
    });
  });

  it('keeps unavailable creation methods non-actionable', async () => {
    mocks.push.mockClear();
    mocks.getApplicationCreationMethods.mockResolvedValue({
      items: [
        {
          method: 'blank',
          availability: 'blocked',
          blocked_reason: 'managed_root_invalid',
        },
      ],
    });
    const wrapper = mount(ApplicationCreateSourceIndex, {
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
