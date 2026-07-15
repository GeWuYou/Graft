import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import ProjectCreateWorkspaceEditor from './ProjectCreateWorkspaceEditor.vue';

const messageMocks = vi.hoisted(() => ({ error: vi.fn(), info: vi.fn(), success: vi.fn(), warning: vi.fn() }));
const monacoDiagnostics = vi.hoisted(() => ({ resolver: (_modelKey: string) => [] as Array<{ severity: number }> }));

vi.mock('tdesign-vue-next/es/message', () => ({ MessagePlugin: messageMocks }));

vi.mock('./ProjectMonacoSurface.vue', () => ({
  default: defineComponent({
    name: 'ProjectMonacoSurfaceStub',
    props: { modelKey: { type: String, default: '' } },
    setup(props, { expose }) {
      expose({
        getModelKey: () => props.modelKey,
        waitForDiagnostics: () => Promise.resolve(monacoDiagnostics.resolver(props.modelKey)),
      });
      return () => h('textarea');
    },
  }),
}));

vi.mock('../shared/page-context', () => ({
  useProjectPageContext: () => ({ t: (key: string, values?: Record<string, string>) => values?.path || key }),
}));

describe('ProjectCreateWorkspaceEditor', () => {
  it('synchronizes the active workspace draft with Ctrl+S without leaving its workspace scope', async () => {
    messageMocks.success.mockClear();
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      attachTo: document.body,
      props: { files: [{ path: '.env', content: 'APP_PORT=8080' }] },
    });
    const editor = wrapper.findComponent({ name: 'ProjectWorkspaceEditor' });
    editor.vm.$emit('update-content', '.env', 'APP_PORT=3000');
    await nextTick();

    const saveEvent = new KeyboardEvent('keydown', {
      bubbles: true,
      cancelable: true,
      code: 'KeyS',
      ctrlKey: true,
      key: 's',
    });
    wrapper.find('textarea').element.dispatchEvent(saveEvent);

    expect(saveEvent.defaultPrevented).toBe(true);
    expect(messageMocks.success).toHaveBeenCalledWith('project.create.workspace.saveSuccess');
    expect(wrapper.props('files')).toEqual([{ path: '.env', content: 'APP_PORT=3000' }]);

    messageMocks.success.mockClear();
    document.body.dispatchEvent(
      new KeyboardEvent('keydown', { bubbles: true, cancelable: true, code: 'KeyS', ctrlKey: true, key: 's' }),
    );
    expect(messageMocks.success).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it('keeps hidden and nested files in the managed workspace manifest', async () => {
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: {
        files: [
          { path: 'compose.yaml', content: 'services: {}' },
          { path: '.env', content: '' },
          { path: 'nginx', node_type: 'directory' },
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

  it('renders a resizable file tree with an initial tab and editor for the creation workspace', () => {
    window.localStorage.removeItem('graft.project.create-workspace.sidebar.width');
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: { files: [{ path: '.env', content: 'APP_PORT=8080' }] },
    });

    expect(wrapper.find('.project-workspace-editor__main-grid').classes()).toContain(
      'project-workspace-editor__main-grid--with-splitter',
    );
    const splitter = wrapper.get('.project-workspace-editor__splitter');
    expect(splitter.attributes('role')).toBe('separator');
    expect(splitter.attributes('aria-valuemin')).toBe('208');
    expect(splitter.attributes('aria-valuemax')).toBe('360');
    expect(splitter.attributes('aria-valuenow')).toBe('256');
    expect(wrapper.find('.project-workspace-editor__tabbar').exists()).toBe(true);
    expect(wrapper.findComponent({ name: 'ProjectMonacoSurfaceStub' }).exists()).toBe(true);
    expect(wrapper.find('[data-testid="workspace-create-save"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="workspace-create-save-all"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="workspace-create-validate"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="workspace-create-format"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="workspace-create-copy"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="workspace-fullscreen-toggle"]').exists()).toBe(true);
  });

  it('persists file tree width after pointer and keyboard resizing', async () => {
    window.localStorage.removeItem('graft.project.create-workspace.sidebar.width');
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      attachTo: document.body,
      props: { files: [{ path: '.env', content: 'APP_PORT=8080' }] },
    });
    const splitter = wrapper.get('.project-workspace-editor__splitter');

    await splitter.trigger('pointerdown', { clientX: 256 });
    window.dispatchEvent(new MouseEvent('pointermove', { clientX: 320 }));
    window.dispatchEvent(new MouseEvent('pointerup'));
    await nextTick();

    expect(window.localStorage.getItem('graft.project.create-workspace.sidebar.width')).toBe('320');
    expect(wrapper.get('.project-workspace-editor__main-grid').attributes('style')).toContain(
      '--project-workspace-sidebar-width: 320px',
    );

    await splitter.trigger('keydown', { key: 'End' });
    expect(window.localStorage.getItem('graft.project.create-workspace.sidebar.width')).toBe('360');
    wrapper.unmount();
  });

  it('restores a persisted file tree width for the creation workspace', () => {
    window.localStorage.setItem('graft.project.create-workspace.sidebar.width', '320');
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: { files: [{ path: '.env', content: 'APP_PORT=8080' }] },
    });

    expect(wrapper.get('.project-workspace-editor__splitter').attributes('aria-valuenow')).toBe('320');
    wrapper.unmount();
    window.localStorage.removeItem('graft.project.create-workspace.sidebar.width');
  });

  it('saves the active workspace draft directly when syntax validation passes', async () => {
    monacoDiagnostics.resolver = () => [];
    messageMocks.success.mockClear();
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: { files: [{ path: 'compose.yaml', content: 'services: {}' }] },
    });
    const editor = wrapper.findComponent({ name: 'ProjectWorkspaceEditor' });
    editor.vm.$emit('update-content', 'compose.yaml', 'services:\n  app:\n    image: nginx');
    await nextTick();

    await wrapper.find('[data-testid="workspace-create-save"]').trigger('click');
    await nextTick();

    expect(messageMocks.success).toHaveBeenCalledWith('project.create.workspace.saveSuccess');
    expect(editor.props('activeBuffer')).toMatchObject({ dirty: false });
  });

  it('requires confirmation before saving a draft with syntax errors', async () => {
    monacoDiagnostics.resolver = () => [{ severity: 8 }];
    messageMocks.success.mockClear();
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: { files: [{ path: 'compose.yaml', content: 'services: {}' }] },
      global: {
        stubs: {
          't-dialog': {
            props: ['visible'],
            emits: ['confirm', 'update:visible'],
            template:
              '<section v-if="visible"><slot /><button class="dialog-confirm" @click="$emit(\'confirm\')">confirm</button></section>',
          },
        },
      },
    });
    const editor = wrapper.findComponent({ name: 'ProjectWorkspaceEditor' });
    editor.vm.$emit('update-content', 'compose.yaml', 'services: [broken');
    await nextTick();

    await wrapper.find('[data-testid="workspace-create-save"]').trigger('click');
    await nextTick();

    expect(wrapper.text()).toContain('project.create.workspace.syntaxErrorSaveBody');
    expect(editor.props('activeBuffer')).toMatchObject({ dirty: true });
    await wrapper.find('.dialog-confirm').trigger('click');
    await nextTick();

    expect(messageMocks.success).toHaveBeenCalledWith('project.create.workspace.saveSuccess');
    expect(editor.props('activeBuffer')).toMatchObject({ dirty: false });
  });

  it('uses F11 within the workspace to toggle its existing outer fullscreen mode', async () => {
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: { files: [{ path: '.env', content: 'APP_PORT=8080' }] },
    });

    const event = new KeyboardEvent('keydown', {
      bubbles: true,
      cancelable: true,
      code: 'F11',
      key: 'F11',
    });
    wrapper.find('.project-workspace-editor').element.dispatchEvent(event);
    await nextTick();

    expect(event.defaultPrevented).toBe(true);
    expect(wrapper.find('.project-workspace-editor').classes()).toContain('project-workspace-editor--fullscreen');
  });

  it('keeps file selection when toggling an empty folder', async () => {
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: {
        files: [
          { path: 'config', node_type: 'directory' },
          { path: 'compose.yaml', content: 'services: {}' },
        ],
      },
    });

    await wrapper
      .findAll('.project-workspace-editor__tree-entry')
      .find((entry) => entry.text() === 'compose.yaml')
      ?.trigger('click');
    expect(wrapper.findAll('.project-workspace-editor__tree-row--selected')).toHaveLength(1);
    await wrapper
      .findAll('.project-workspace-editor__tree-entry')
      .find((entry) => entry.text() === 'config')
      ?.trigger('click');

    expect(wrapper.findAll('.project-workspace-editor__tree-row--selected')).toHaveLength(1);
  });

  it('creates a file from the tree context menu with an inline name', async () => {
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
    await wrapper.find('input').setValue('start');
    wrapper.findComponent({ name: 'ProjectWorkspaceEditor' }).vm.$emit('inline-edit-submit');
    await nextTick();
    await nextTick();

    expect(wrapper.emitted('update:files')?.at(-1)?.[0]).toEqual(
      expect.arrayContaining([expect.objectContaining({ path: 'start', node_type: 'file' })]),
    );
    const editor = wrapper.findComponent({ name: 'ProjectWorkspaceEditor' });
    expect(editor.props('activePath')).toBe('start');
    expect(editor.props('selectedPath')).toBe('start');
    expect(editor.props('activeBuffer')).toMatchObject({ path: 'start' });
    expect(wrapper.find('.project-workspace-editor__tree-row--selected').text()).toContain('start');
  });

  it('keeps the root context menu available when the workspace is empty', async () => {
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: { files: [] },
      global: {
        stubs: {
          't-input': {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
          },
        },
      },
    });

    await wrapper.find('.project-workspace-editor__tree').trigger('contextmenu', { clientX: 20, clientY: 20 });
    expect(wrapper.findAll('[role="menuitem"]')).toHaveLength(4);

    await wrapper.findAll('[role="menuitem"]')[0].trigger('click');
    await wrapper.find('input').setValue('recover.yml');
    wrapper.findComponent({ name: 'ProjectWorkspaceEditor' }).vm.$emit('inline-edit-submit');

    expect(wrapper.emitted('update:files')?.at(-1)?.[0]).toEqual(
      expect.arrayContaining([expect.objectContaining({ path: 'recover.yml', node_type: 'file' })]),
    );
  });

  it('creates a directory entry without content', async () => {
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: { files: [{ path: 'compose.yaml', content: 'services: {}' }] },
      global: {
        stubs: {
          't-input': {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
          },
        },
      },
    });

    await wrapper.find('.project-workspace-editor__tree').trigger('contextmenu', { clientX: 20, clientY: 20 });
    await wrapper.findAll('[role="menuitem"]')[1].trigger('click');
    await wrapper.find('input').setValue('config');
    wrapper.findComponent({ name: 'ProjectWorkspaceEditor' }).vm.$emit('inline-edit-submit');
    await nextTick();

    const entries = wrapper.emitted('update:files')?.at(-1)?.[0] as Array<Record<string, unknown>>;
    expect(entries.find((entry) => entry.path === 'config')).toEqual({ path: 'config', node_type: 'directory' });
    expect(wrapper.findComponent({ name: 'ProjectWorkspaceEditor' }).props('activePath')).toBe('compose.yaml');
  });

  it('creates a file inside the selected directory instead of inferring a folder type', async () => {
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      props: {
        files: [
          { path: 'config', node_type: 'directory' },
          { path: 'config/app.yaml', content: 'name: app' },
        ],
      },
      global: {
        stubs: {
          't-input': {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
          },
        },
      },
    });

    await wrapper.find('.project-workspace-editor__tree-menu-trigger[aria-label="config"]').trigger('click');
    await wrapper.findAll('[role="menuitem"]')[0].trigger('click');
    await wrapper.find('input').setValue('dashboard.e2e.json');
    wrapper.findComponent({ name: 'ProjectWorkspaceEditor' }).vm.$emit('inline-edit-submit');
    await nextTick();
    await nextTick();

    expect(wrapper.emitted('update:files')?.at(-1)?.[0]).toEqual(
      expect.arrayContaining([expect.objectContaining({ path: 'config/dashboard.e2e.json', node_type: 'file' })]),
    );
    const editor = wrapper.findComponent({ name: 'ProjectWorkspaceEditor' });
    expect(editor.props('activePath')).toBe('config/dashboard.e2e.json');
    expect(editor.props('selectedPath')).toBe('config/dashboard.e2e.json');
  });

  it('preselects a file basename while preserving its extension for inline rename', async () => {
    const wrapper = mount(ProjectCreateWorkspaceEditor, {
      attachTo: document.body,
      props: { files: [{ path: 'compose.yaml', content: 'services: {}' }] },
      global: {
        stubs: {
          't-input': {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
          },
        },
      },
    });

    await wrapper.find('.project-workspace-editor__tree-menu-trigger[aria-label="compose.yaml"]').trigger('click');
    await wrapper.findAll('[role="menuitem"]')[2].trigger('click');
    await nextTick();
    await nextTick();

    const input = wrapper.find('input').element as HTMLInputElement;
    expect(input.value).toBe('compose.yaml');
    expect(input.selectionStart).toBe(0);
    expect(input.selectionEnd).toBe('compose'.length);
    wrapper.unmount();
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
    wrapper.findComponent({ name: 'ProjectWorkspaceEditor' }).vm.$emit('inline-edit-submit');

    expect(wrapper.emitted('update:files') ?? []).toHaveLength(initialUpdateCount);

    await wrapper.find('.project-workspace-editor__tree-menu-trigger[aria-label="other"]').trigger('click');
    await wrapper.findAll('[role="menuitem"]')[2].trigger('click');
    await wrapper.find('input').setValue('scripts');
    wrapper.findComponent({ name: 'ProjectWorkspaceEditor' }).vm.$emit('inline-edit-submit');

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
