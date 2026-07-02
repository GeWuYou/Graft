import { flushPromises, shallowMount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ref } from 'vue';

import ProjectResourcesSection from './ProjectResourcesSection.vue';

const containerApiMocks = vi.hoisted(() => ({
  getContainers: vi.fn(),
}));

const projectApiMocks = vi.hoisted(() => ({
  getProjectServices: vi.fn(),
}));

vi.mock('@/modules/container/api/container', () => ({
  getContainers: containerApiMocks.getContainers,
}));

vi.mock('../api/project', () => ({
  getProjectServices: projectApiMocks.getProjectServices,
}));

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>();
  const locale = ref('en-US');
  return {
    ...actual,
    useI18n: () => ({
      locale,
      t: (key: string) => key,
    }),
  };
});

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
  }),
}));

describe('ProjectResourcesSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    containerApiMocks.getContainers.mockResolvedValue({
      items: [],
      total: 0,
    });
    projectApiMocks.getProjectServices.mockResolvedValue({
      items: [],
    });
  });

  it('reloads containers once when project identity props change together', async () => {
    const wrapper = shallowMount(ProjectResourcesSection, {
      props: {
        canonicalProjectName: 'compose-alpha',
        projectId: 1,
      },
    });

    await flushPromises();
    expect(containerApiMocks.getContainers).toHaveBeenCalledTimes(1);

    await wrapper.setProps({
      canonicalProjectName: 'compose-beta',
      projectId: 2,
    });
    await flushPromises();

    expect(containerApiMocks.getContainers).toHaveBeenCalledTimes(2);
    expect(containerApiMocks.getContainers).toHaveBeenLastCalledWith({
      limit: 10,
      offset: 0,
      orchestrator: 'compose',
      source_scope: 'compose-beta',
      source_scope_kind: 'compose_project',
    });
  });
});
