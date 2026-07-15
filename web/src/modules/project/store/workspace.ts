import { defineStore } from 'pinia';

import {
  hasWorkspaceUnsavedChanges,
  type ProjectWorkspaceMonacoLanguage,
  resolveWorkspaceFileName,
  resolveWorkspaceMonacoLanguage,
} from '../shared/configuration-workspace';
import { emitProjectWorkspaceDebug } from '../shared/project-workspace-debug';
import type { ProjectWorkspaceFileKind, ProjectWorkspaceTreeItem } from '../types/project';

export type WorkspaceNodeType = 'directory' | 'file';

export type WorkspaceNode = ProjectWorkspaceTreeItem & {
  childKeys: string[];
  childrenLoaded: boolean;
  parentKey: string | null;
};

export type OpenedWorkspaceFile = {
  content: string;
  editable: boolean;
  error: string;
  fileKind: ProjectWorkspaceFileKind;
  language: ProjectWorkspaceMonacoLanguage;
  loaded: boolean;
  loading: boolean;
  name: string;
  path: string;
  savedContent: string;
  saving: boolean;
};

export type WorkspaceTreeRow = {
  depth: number;
  expanded: boolean;
  item: WorkspaceNode;
};

export type WorkspacePendingOperation = {
  key: string;
  kind: 'create' | 'delete' | 'load' | 'rename' | 'save';
};

type WorkspaceSession = {
  activeFileKey: string;
  dirtyFiles: string[];
  expandedKeys: string[];
  fileContents: Record<string, OpenedWorkspaceFile>;
  nodesByKey: Record<string, WorkspaceNode>;
  openedTabs: string[];
  pendingOperations: WorkspacePendingOperation[];
  rootKeys: string[];
  selectedKey: string;
};

type WorkspaceState = {
  sessions: Record<string, WorkspaceSession>;
};

function createSession(): WorkspaceSession {
  return {
    activeFileKey: '',
    dirtyFiles: [],
    expandedKeys: [],
    fileContents: {},
    nodesByKey: {},
    openedTabs: [],
    pendingOperations: [],
    rootKeys: [],
    selectedKey: '',
  };
}

function normalizePath(path: string) {
  return String(path || '')
    .trim()
    .replace(/^\.\//, '')
    .replace(/\/+$/, '');
}

function parentKeyForPath(path: string) {
  const parts = normalizePath(path).split('/');
  return parts.length > 1 ? parts.slice(0, -1).join('/') : null;
}

function sortKeys(nodesByKey: Record<string, WorkspaceNode>, keys: string[]) {
  return [...keys].sort((left, right) => {
    const leftNode = nodesByKey[left];
    const rightNode = nodesByKey[right];
    if (!leftNode || !rightNode) return left.localeCompare(right);
    if (leftNode.node_type !== rightNode.node_type) return leftNode.node_type === 'directory' ? -1 : 1;
    return leftNode.name.localeCompare(rightNode.name, undefined, { sensitivity: 'base' });
  });
}

function ensureDirectory(session: WorkspaceSession, key: string) {
  const normalizedKey = normalizePath(key);
  if (!normalizedKey) return;
  const existing = session.nodesByKey[normalizedKey];
  if (existing) return;
  const parentKey = parentKeyForPath(normalizedKey);
  session.nodesByKey[normalizedKey] = {
    childKeys: [],
    childrenLoaded: false,
    editable: false,
    file_kind: 'directory',
    has_children: false,
    name: normalizedKey.split('/').at(-1) || normalizedKey,
    node_type: 'directory',
    parentKey,
    readable: true,
    relative_path: normalizedKey,
  };
  if (parentKey) {
    ensureDirectory(session, parentKey);
    const parent = session.nodesByKey[parentKey];
    if (parent && !parent.childKeys.includes(normalizedKey)) parent.childKeys.push(normalizedKey);
  } else if (!session.rootKeys.includes(normalizedKey)) {
    session.rootKeys.push(normalizedKey);
  }
}

function ensureAncestorsExpanded(session: WorkspaceSession, key: string) {
  let parentKey = session.nodesByKey[key]?.parentKey ?? parentKeyForPath(key);
  while (parentKey) {
    if (!session.expandedKeys.includes(parentKey)) session.expandedKeys.push(parentKey);
    parentKey = session.nodesByKey[parentKey]?.parentKey ?? parentKeyForPath(parentKey);
  }
}

function logWorkspaceSession(
  event: string,
  sessionKey: string,
  session: WorkspaceSession,
  detail: Record<string, unknown> = {},
) {
  emitProjectWorkspaceDebug(event, {
    activeFileKey: session.activeFileKey || '-',
    bufferCount: Object.keys(session.fileContents).length,
    openedTabCount: session.openedTabs.length,
    selectedKey: session.selectedKey || '-',
    sessionKey,
    treeNodeCount: Object.keys(session.nodesByKey).length,
    ...detail,
  });
}

export const useProjectWorkspaceStore = defineStore('project-workspace', {
  state: (): WorkspaceState => ({ sessions: {} }),
  getters: {
    session:
      (state) =>
      (key: string): WorkspaceSession => {
        return state.sessions[key] ?? createSession();
      },
    workspaceTree:
      (state) =>
      (sessionKey: string): WorkspaceNode[] => {
        const session = state.sessions[sessionKey] ?? createSession();
        return session.rootKeys
          .map((key) => session.nodesByKey[key])
          .filter((node): node is WorkspaceNode => Boolean(node));
      },
    visibleTreeRows:
      (state) =>
      (sessionKey: string): WorkspaceTreeRow[] => {
        const session = state.sessions[sessionKey] ?? createSession();
        const rows: WorkspaceTreeRow[] = [];
        const visit = (keys: string[], depth: number) => {
          for (const key of sortKeys(session.nodesByKey, keys)) {
            const node = session.nodesByKey[key];
            if (!node) continue;
            const expanded = node.node_type === 'directory' && session.expandedKeys.includes(key);
            rows.push({ depth, expanded, item: node });
            if (expanded) visit(node.childKeys, depth + 1);
          }
        };
        visit(session.rootKeys, 0);
        return rows;
      },
    openedFiles:
      (state) =>
      (sessionKey: string): OpenedWorkspaceFile[] => {
        const session = state.sessions[sessionKey] ?? createSession();
        return session.openedTabs
          .map((key) => session.fileContents[key])
          .filter((file): file is OpenedWorkspaceFile => Boolean(file));
      },
    activeFile:
      (state) =>
      (sessionKey: string): OpenedWorkspaceFile | null => {
        const session = state.sessions[sessionKey] ?? createSession();
        return session.fileContents[session.activeFileKey] ?? null;
      },
  },
  actions: {
    ensureSession(key: string) {
      if (!this.sessions[key]) {
        this.sessions[key] = createSession();
        logWorkspaceSession('session-created', key, this.sessions[key]);
      }
      return this.sessions[key];
    },
    clearSession(key: string) {
      delete this.sessions[key];
    },
    ingestTree(
      sessionKey: string,
      entries: ProjectWorkspaceTreeItem[],
      parentKey: string | null = null,
      childrenLoaded = true,
    ) {
      const session = this.ensureSession(sessionKey);
      const normalizedParentKey = parentKey ? normalizePath(parentKey) : null;
      if (normalizedParentKey) ensureDirectory(session, normalizedParentKey);
      const incomingKeys: string[] = [];
      for (const entry of entries) {
        const key = normalizePath(entry.relative_path);
        if (!key) continue;
        const inferredParent = parentKeyForPath(key);
        if (inferredParent) ensureDirectory(session, inferredParent);
        const resolvedParent = inferredParent ?? normalizedParentKey;
        const existing = session.nodesByKey[key];
        session.nodesByKey[key] = {
          ...existing,
          ...entry,
          childKeys: existing?.childKeys ?? [],
          childrenLoaded: existing?.childrenLoaded ?? false,
          parentKey: resolvedParent,
          relative_path: key,
        };
        if (resolvedParent) {
          const parent = session.nodesByKey[resolvedParent];
          if (parent && !parent.childKeys.includes(key)) parent.childKeys.push(key);
        } else if (!session.rootKeys.includes(key)) {
          session.rootKeys.push(key);
        }
        incomingKeys.push(key);
      }
      if (normalizedParentKey) {
        const parent = session.nodesByKey[normalizedParentKey];
        if (parent) {
          parent.childKeys = sortKeys(session.nodesByKey, incomingKeys);
          parent.childrenLoaded = childrenLoaded;
          parent.has_children = incomingKeys.length > 0;
        }
      } else {
        session.rootKeys = sortKeys(
          session.nodesByKey,
          Object.keys(session.nodesByKey).filter((key) => !session.nodesByKey[key]?.parentKey),
        );
      }
    },
    replaceTree(sessionKey: string, entries: ProjectWorkspaceTreeItem[]) {
      const session = this.ensureSession(sessionKey);
      const preservedExpanded = session.expandedKeys;
      const preservedSelected = session.selectedKey;
      session.nodesByKey = {};
      session.rootKeys = [];
      this.ingestTree(sessionKey, entries);
      session.expandedKeys = preservedExpanded.filter((key) => Boolean(session.nodesByKey[key]));
      session.selectedKey = session.nodesByKey[preservedSelected] ? preservedSelected : '';
      logWorkspaceSession('tree-replaced', sessionKey, session, { entryCount: entries.length });
    },
    setExpanded(sessionKey: string, key: string, expanded: boolean) {
      const session = this.ensureSession(sessionKey);
      const normalizedKey = normalizePath(key);
      if (!session.nodesByKey[normalizedKey] || session.nodesByKey[normalizedKey].node_type !== 'directory') return;
      session.expandedKeys = expanded
        ? [...new Set([...session.expandedKeys, normalizedKey])]
        : session.expandedKeys.filter((item) => item !== normalizedKey);
    },
    patchNode(sessionKey: string, item: ProjectWorkspaceTreeItem) {
      const key = normalizePath(item.relative_path);
      const session = this.ensureSession(sessionKey);
      const existing = session.nodesByKey[key];
      if (!existing) return;
      session.nodesByKey[key] = {
        ...existing,
        ...item,
        childKeys: existing.childKeys,
        parentKey: existing.parentKey,
        relative_path: key,
      };
    },
    selectNode(sessionKey: string, key: string) {
      const normalizedKey = normalizePath(key);
      const session = this.ensureSession(sessionKey);
      if (session.nodesByKey[normalizedKey]) session.selectedKey = normalizedKey;
    },
    focusFile(sessionKey: string, key: string) {
      const normalizedKey = normalizePath(key);
      const session = this.ensureSession(sessionKey);
      if (session.nodesByKey[normalizedKey]?.node_type !== 'file') return;
      session.selectedKey = normalizedKey;
      ensureAncestorsExpanded(session, normalizedKey);
    },
    openFile(sessionKey: string, key: string, content?: Partial<OpenedWorkspaceFile>) {
      const session = this.ensureSession(sessionKey);
      const normalizedKey = normalizePath(key);
      const node = session.nodesByKey[normalizedKey];
      if (!node || node.node_type !== 'file') return;
      if (!session.fileContents[normalizedKey]) {
        session.fileContents[normalizedKey] = {
          content: '',
          editable: node.editable,
          error: '',
          fileKind: node.file_kind,
          language: resolveWorkspaceMonacoLanguage({
            fileKind: node.file_kind,
            languageHint: node.language_hint,
            path: normalizedKey,
          }),
          loaded: false,
          loading: false,
          name: resolveWorkspaceFileName(normalizedKey),
          path: normalizedKey,
          savedContent: '',
          saving: false,
          ...content,
        };
      }
      if (!session.openedTabs.includes(normalizedKey)) session.openedTabs.push(normalizedKey);
      session.activeFileKey = normalizedKey;
      this.focusFile(sessionKey, normalizedKey);
      logWorkspaceSession('file-opened', sessionKey, session, { path: normalizedKey });
    },
    setFileContent(sessionKey: string, key: string, content: string) {
      const session = this.ensureSession(sessionKey);
      const file = session.fileContents[normalizePath(key)];
      if (!file || !file.editable) return;
      file.content = content;
      this.syncDirtyFile(sessionKey, file.path);
    },
    markFileSaved(sessionKey: string, key: string, content?: string) {
      const session = this.ensureSession(sessionKey);
      const file = session.fileContents[normalizePath(key)];
      if (!file) return;
      file.content = content ?? file.content;
      file.savedContent = file.content;
      file.loaded = true;
      this.syncDirtyFile(sessionKey, file.path);
    },
    syncOpenedFiles(sessionKey: string, files: OpenedWorkspaceFile[], activeFileKey: string) {
      const session = this.ensureSession(sessionKey);
      session.openedTabs = files.map((file) => file.path);
      for (const file of files) {
        session.fileContents[file.path] = { ...file };
        this.syncDirtyFile(sessionKey, file.path);
      }
      for (const path of Object.keys(session.fileContents)) {
        if (!session.openedTabs.includes(path)) delete session.fileContents[path];
      }
      session.activeFileKey = session.openedTabs.includes(activeFileKey)
        ? activeFileKey
        : (session.openedTabs.at(-1) ?? '');
      if (session.activeFileKey) this.focusFile(sessionKey, session.activeFileKey);
      logWorkspaceSession('opened-files-synced', sessionKey, session, { inputFileCount: files.length });
    },
    syncDirtyFile(sessionKey: string, key: string) {
      const session = this.ensureSession(sessionKey);
      const file = session.fileContents[normalizePath(key)];
      if (!file) return;
      const dirty = hasWorkspaceUnsavedChanges(file.content, file.savedContent);
      session.dirtyFiles = dirty
        ? [...new Set([...session.dirtyFiles, file.path])]
        : session.dirtyFiles.filter((path) => path !== file.path);
    },
    closeFile(sessionKey: string, key: string) {
      const session = this.ensureSession(sessionKey);
      const normalizedKey = normalizePath(key);
      session.openedTabs = session.openedTabs.filter((path) => path !== normalizedKey);
      if (session.activeFileKey === normalizedKey) {
        session.activeFileKey = session.openedTabs.at(-1) ?? '';
        if (session.activeFileKey) this.focusFile(sessionKey, session.activeFileKey);
      }
      logWorkspaceSession('file-closed', sessionKey, session, { path: normalizedKey });
    },
  },
});
