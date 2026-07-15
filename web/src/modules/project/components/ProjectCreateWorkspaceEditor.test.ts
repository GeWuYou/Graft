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
          { path: 'nginx', content: '', node_type: 'directory' },
          { path: 'nginx/nginx.conf', content: 'events {}' },
        ],
      },
      global: {
        stubs: {
          'project-monaco-surface': { template: '<textarea />' },
          't-alert': { template: '<div><slot /></div>' },
          't-button': { template: '<button @click="$emit(\'click\')"><slot /></button>' },
          't-card': { template: '<section><slot name="actions" /><slot /></section>' },
          't-dialog': {
            props: ['visible'],
            emits: ['confirm', 'update:visible'],
            template:
              '<section v-if="visible"><slot /><button type="button" class="dialog-confirm" @click="$emit(\'confirm\')">confirm</button></section>',
          },
          't-form': { template: '<form><slot /></form>' },
          't-form-item': { template: '<label><slot /></label>' },
          't-input': {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
          },
        },
      },
    });

    expect(wrapper.text()).toContain('compose.yaml');
    expect(wrapper.text()).toContain('.env');
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('nginx'))
      ?.trigger('click');
    expect(wrapper.text()).toContain('nginx.conf');
  });

  it('provides a stable editor-height storage key to the shared viewer frame', () => {
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: { files: [{ path: '.env', content: 'APP_PORT=8080' }] },
    });

    expect(wrapper.findComponent({ name: 'ContentViewerFrame' }).props('storageKey')).toBe(
      'graft.project.create-workspace.editor.height',
    );
  });

  it('creates unrestricted file names through the tree context menu', async () => {
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: { files: [{ path: 'compose.yaml', content: 'services: {}' }] },
      global: {
        stubs: {
          'project-monaco-surface': { template: '<textarea />' },
          't-alert': { template: '<div><slot /></div>' },
          't-card': { template: '<section><slot /></section>' },
          't-empty': { template: '<div />' },
          't-form': { template: '<form><slot /></form>' },
          't-form-item': { template: '<label><slot /></label>' },
          't-dialog': {
            props: ['visible'],
            emits: ['confirm'],
            template:
              '<section v-if="visible"><slot /><button class="dialog-confirm" @click="$emit(\'confirm\')">confirm</button></section>',
          },
          't-input': {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
          },
        },
      },
    });

    await wrapper.find('.project-workspace-editor__tree').trigger('contextmenu', { clientX: 20, clientY: 20 });
    await wrapper.findAll('[role="menuitem"]')[0].trigger('click');
    await wrapper.find('input').setValue('scripts/start');
    await wrapper.find('.dialog-confirm').trigger('click');

    expect(wrapper.emitted('update:files')?.at(-1)?.[0]).toEqual(
      expect.arrayContaining([expect.objectContaining({ path: 'scripts/start', node_type: 'file' })]),
    );
  });

  it('rejects workspace paths that would be ancestors or descendants of existing files', async () => {
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: {
        files: [
          { path: 'scripts/start', content: '#!/bin/sh' },
          { path: 'other', content: 'keep' },
        ],
      },
      global: {
        stubs: {
          'project-monaco-surface': { template: '<textarea />' },
          't-alert': { template: '<div><slot /></div>' },
          't-card': { template: '<section><slot /></section>' },
          't-dialog': {
            props: ['visible'],
            emits: ['confirm'],
            template:
              '<section v-if="visible"><slot /><button class="dialog-confirm" @click="$emit(\'confirm\')">confirm</button></section>',
          },
          't-empty': { template: '<div />' },
          't-form': { template: '<form><slot /></form>' },
          't-form-item': { template: '<label><slot /></label>' },
          't-input': {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
          },
        },
      },
    });

    const initialUpdateCount = wrapper.emitted('update:files')?.length ?? 0;
    await wrapper.find('.project-workspace-editor__tree').trigger('contextmenu', { clientX: 20, clientY: 20 });
    await wrapper.findAll('[role="menuitem"]')[0].trigger('click');
    await wrapper.find('input').setValue('scripts');
    await wrapper.find('.dialog-confirm').trigger('click');

    expect(wrapper.emitted('update:files') ?? []).toHaveLength(initialUpdateCount);

    await wrapper.find('.project-workspace-editor__tree-menu-trigger[aria-label="other"]').trigger('click');
    await wrapper.findAll('[role="menuitem"]')[2].trigger('click');
    await wrapper.find('input').setValue('scripts');
    await wrapper.find('.dialog-confirm').trigger('click');

    expect(wrapper.emitted('update:files') ?? []).toHaveLength(initialUpdateCount);
  });

  it('opens the workspace action menu from a keyboard-focusable trigger and restores focus on escape', async () => {
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      attachTo: document.body,
      props: { files: [{ path: 'compose.yaml', content: 'services: {}' }] },
      global: {
        stubs: {
          'project-monaco-surface': { template: '<textarea />' },
          't-alert': { template: '<div><slot /></div>' },
          't-card': { template: '<section><slot /></section>' },
          't-empty': { template: '<div />' },
        },
      },
    });

    const trigger = wrapper.find('.project-workspace-editor__tree-menu-trigger');
    await trigger.trigger('click');
    expect(document.activeElement).toBe(wrapper.findAll('[role="menuitem"]')[0].element);

    await wrapper.find('[role="menu"]').trigger('keydown', { key: 'ArrowDown' });
    expect(document.activeElement).toBe(wrapper.findAll('[role="menuitem"]')[1].element);

    await wrapper.find('[role="menu"]').trigger('keydown', { key: 'Escape' });
    expect(document.activeElement).toBe(trigger.element);
    wrapper.unmount();
  });
});
