import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import ProjectRuntimeIndex from './runtime-index.vue';

const push = vi.fn();

vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

describe('ProjectRuntimeIndex', () => {
  it('offers only Docker Compose as an actionable runtime', async () => {
    const wrapper = mount(ProjectRuntimeIndex, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot /></header>' },
          't-card': { template: '<section><slot /></section>' },
          't-tag': { template: '<span><slot /></span>' },
          't-tooltip': { template: '<div><slot /></div>' },
          't-button': { template: '<button @click="$emit(\'click\')"><slot /></button>' },
        },
      },
    });

    expect(wrapper.text()).toContain('project.runtime.items.dockerCompose.title');
    expect(wrapper.text()).toContain('project.runtime.items.dockerSwarm.title');
    expect(wrapper.findAll('button')).toHaveLength(1);

    await wrapper.get('[data-testid="project-runtime-docker-compose"]').trigger('click');

    expect(push).toHaveBeenCalledWith({ name: 'ProjectCreateSourceIndex', query: { runtime: 'docker-compose' } });
  });
});
