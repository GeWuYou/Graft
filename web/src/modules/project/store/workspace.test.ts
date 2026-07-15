import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it } from 'vitest';

import { useProjectWorkspaceStore } from './workspace';

describe('project workspace store', () => {
  beforeEach(() => setActivePinia(createPinia()));

  it('normalizes arbitrary nested paths into one parent-child tree', () => {
    const store = useProjectWorkspaceStore();
    store.activateSession('test');
    store.replaceTree([
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

    expect(store.workspaceTree.map((node) => node.relative_path)).toEqual(['app', 'sql']);
    store.setExpanded('sql', true);
    expect(store.visibleTreeRows.map((row) => row.item.relative_path)).toEqual([
      'app',
      'sql',
      'sql/data.sql',
      'sql/init.sql',
    ]);
  });

  it('keeps editor state separate while opening a deep file expands its ancestors', () => {
    const store = useProjectWorkspaceStore();
    store.activateSession('test');
    store.replaceTree([
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

    store.openFile('app/config.yaml', { content: 'enabled: true', loaded: true, savedContent: 'enabled: true' });
    store.setFileContent('app/config.yaml', 'enabled: false');

    expect(store.activeFile?.content).toBe('enabled: false');
    expect(store.activeSession.expandedKeys).toContain('app');
    expect(store.activeSession.dirtyFiles).toEqual(['app/config.yaml']);
    expect(store.activeSession.nodesByKey['app/config.yaml'].childKeys).toEqual([]);
  });
});
