import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import ProjectCreateWorkspaceEditor from './ProjectCreateWorkspaceEditor.vue';

vi.mock('./ProjectMonacoSurface.vue', () => ({
  default: { name: 'ProjectMonacoSurfaceStub', template: '<textarea />' },
}));

vi.mock('../shared/page-context', () => ({
  useProjectPageContext: () => ({ t: (key: string, values?: Record<string, string>) => values?.path || key }),
}));

describe('ProjectCreateWorkspaceEditor', () => {
  it('keeps hidden and nested files in the managed workspace manifest', async () => {
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: {
        files: [
          { path: 'compose.yaml', content: 'services: {}' },
          { path: '.env', content: '' },
        ],
      },
      global: {
        stubs: {
          'project-monaco-surface': { template: '<textarea />' },
          't-alert': { template: '<div><slot /></div>' },
          't-button': { template: '<button @click="$emit(\'click\')"><slot /></button>' },
          't-card': { template: '<section><slot name="actions" /><slot /></section>' },
          't-dialog': { template: '<section><slot /></section>' },
          't-form': { template: '<form><slot /></form>' },
          't-form-item': { template: '<label><slot /></label>' },
          't-input': { template: '<input />' },
        },
      },
    });

    expect(wrapper.text()).toContain('compose.yaml');
    expect(wrapper.text()).toContain('.env');
  });
});
