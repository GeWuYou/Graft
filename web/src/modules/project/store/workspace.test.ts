import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it } from 'vitest';

import { useProjectWorkspaceStore } from './workspace';

describe('project workspace store', () => {
  beforeEach(() => setActivePinia(createPinia()));

  it('normalizes arbitrary nested paths into one parent-child tree', () => {
    const store = useProjectWorkspaceStore();
    store.ensureSession('test');
    store.replaceTree('test', [
      {
        editable: true,
        file_kind: 'text',
        has_children: false,
        name: 'init.sql',
        node_type: 'file',
        readable: true,
        relative_path: 'sql/init.sql',
      },
      {
        editable: true,
        file_kind: 'text',
        has_children: false,
        name: 'data.sql',
        node_type: 'file',
        readable: true,
        relative_path: 'sql/data.sql',
      },
      {
        editable: true,
        file_kind: 'config',
        has_children: false,
        name: 'config.yaml',
        node_type: 'file',
        readable: true,
        relative_path: 'app/config.yaml',
      },
    ]);

    expect(store.workspaceTree('test').map((node) => node.relative_path)).toEqual(['app', 'sql']);
    store.setExpanded('test', 'sql', true);
    expect(store.visibleTreeRows('test').map((row) => row.item.relative_path)).toEqual([
      'app',
      'sql',
      'sql/data.sql',
      'sql/init.sql',
    ]);
  });

  it('keeps editor state separate while opening a deep file expands its ancestors', () => {
    const store = useProjectWorkspaceStore();
    store.ensureSession('test');
    store.replaceTree('test', [
      {
        editable: true,
        file_kind: 'config',
        has_children: false,
        name: 'config.yaml',
        node_type: 'file',
        readable: true,
        relative_path: 'app/config.yaml',
      },
    ]);

    store.openFile('test', 'app/config.yaml', {
      content: 'enabled: true',
      loaded: true,
      savedContent: 'enabled: true',
    });
    store.setFileContent('test', 'app/config.yaml', 'enabled: false');

    expect(store.activeFile('test')?.content).toBe('enabled: false');
    expect(store.session('test').expandedKeys).toContain('app');
    expect(store.session('test').dirtyFiles).toEqual(['app/config.yaml']);
    expect(store.session('test').nodesByKey['app/config.yaml'].childKeys).toEqual([]);
  });

  it('keeps selection separate from directory expansion', () => {
    const store = useProjectWorkspaceStore();
    store.replaceTree('test', [
      {
        editable: true,
        file_kind: 'text',
        has_children: false,
        name: 'compose.yaml',
        node_type: 'file',
        readable: true,
        relative_path: 'compose.yaml',
      },
      {
        editable: false,
        file_kind: 'directory',
        has_children: false,
        name: 'config',
        node_type: 'directory',
        readable: true,
        relative_path: 'config',
      },
    ]);
    store.selectNode('test', 'compose.yaml');
    store.setExpanded('test', 'config', true);

    expect(store.session('test').selectedKey).toBe('compose.yaml');
  });

  it('focuses the next tab and expands its ancestors after closing the active file', () => {
    const store = useProjectWorkspaceStore();
    store.replaceTree('test', [
      {
        editable: true,
        file_kind: 'config',
        has_children: false,
        name: 'dashboard.json',
        node_type: 'file',
        readable: true,
        relative_path: 'config/dashboard.json',
      },
      {
        editable: true,
        file_kind: 'compose',
        has_children: false,
        name: 'compose.yaml',
        node_type: 'file',
        readable: true,
        relative_path: 'compose.yaml',
      },
    ]);
    store.openFile('test', 'config/dashboard.json');
    store.openFile('test', 'compose.yaml');
    store.setExpanded('test', 'config', false);

    store.closeFile('test', 'compose.yaml');

    expect(store.session('test').activeFileKey).toBe('config/dashboard.json');
    expect(store.session('test').selectedKey).toBe('config/dashboard.json');
    expect(store.session('test').expandedKeys).toContain('config');
  });

  it('keeps configuration and creation sessions isolated when both workspaces stay mounted', () => {
    const store = useProjectWorkspaceStore();
    store.replaceTree('project:shared-postgres', [
      {
        editable: true,
        file_kind: 'compose',
        has_children: false,
        name: 'docker-compose.yml',
        node_type: 'file',
        readable: true,
        relative_path: 'docker-compose.yml',
      },
    ]);
    store.openFile('project:shared-postgres', 'docker-compose.yml', {
      content: 'services:\n  postgres: {}',
      loaded: true,
      savedContent: 'services:\n  postgres: {}',
    });

    store.replaceTree('project-create-workspace', [
      {
        editable: true,
        file_kind: 'env',
        has_children: false,
        name: '.env',
        node_type: 'file',
        readable: true,
        relative_path: '.env',
      },
    ]);
    store.openFile('project-create-workspace', '.env', {
      content: 'APP_PORT=8080',
      loaded: true,
      savedContent: 'APP_PORT=8080',
    });
    store.setFileContent('project-create-workspace', '.env', 'APP_PORT=3000');

    expect(store.activeFile('project:shared-postgres')?.content).toBe('services:\n  postgres: {}');
    expect(store.openedFiles('project:shared-postgres').map((file) => file.path)).toEqual(['docker-compose.yml']);
    expect(store.activeFile('project-create-workspace')?.content).toBe('APP_PORT=3000');
    expect(store.session('project:shared-postgres').dirtyFiles).toEqual([]);
    expect(store.session('project-create-workspace').dirtyFiles).toEqual(['.env']);
  });

  it('keeps __proto__ session and file paths as ordinary workspace keys', () => {
    const store = useProjectWorkspaceStore();
    store.replaceTree('__proto__', [
      {
        editable: true,
        file_kind: 'text',
        has_children: false,
        name: '__proto__',
        node_type: 'file',
        readable: true,
        relative_path: '__proto__',
      },
    ]);
    store.openFile('__proto__', '__proto__', { content: 'safe', loaded: true, savedContent: 'safe' });

    expect(Object.getPrototypeOf(store.$state.sessions)).toBeNull();
    expect(Object.getPrototypeOf(store.session('__proto__').nodesByKey)).toBeNull();
    expect(Object.getPrototypeOf(store.session('__proto__').fileContents)).toBeNull();
    expect(store.activeFile('__proto__')?.content).toBe('safe');
  });
});
