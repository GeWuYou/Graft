import type {
  ProjectWorkspaceFileKind,
  ProjectWorkspaceLanguageHint,
  ProjectWorkspaceTreeItem,
} from '../types/project';

export type ProjectWorkspaceMonacoLanguage = 'dockerfile' | 'ini' | 'json' | 'plaintext' | 'shell' | 'yaml';

export function normalizeWorkspaceContent(value: string) {
  return String(value ?? '').replace(/\r\n/g, '\n');
}

export function normalizeTextBlock(value: string) {
  const normalized = normalizeWorkspaceContent(value)
    .split('\n')
    .map((line) => line.replace(/\s+$/g, ''))
    .join('\n')
    .trim();

  return normalized ? `${normalized}\n` : '';
}

export function hasWorkspaceUnsavedChanges(current: string, saved: string) {
  return normalizeWorkspaceContent(current) !== normalizeWorkspaceContent(saved);
}

export function resolveWorkspaceFileName(path: string) {
  const normalized = String(path || '').trim();
  if (!normalized) {
    return 'untitled';
  }
  return normalized.split('/').at(-1) || normalized;
}

export function resolveWorkspaceMonacoLanguage(options: {
  fileKind?: ProjectWorkspaceFileKind | null;
  languageHint?: ProjectWorkspaceLanguageHint | null;
  path?: string;
}): ProjectWorkspaceMonacoLanguage {
  const hint = String(options.languageHint || '')
    .trim()
    .toLowerCase();
  const extension = String(options.path || '')
    .split('.')
    .at(-1)
    ?.trim()
    .toLowerCase();

  if (hint === 'yaml' || hint === 'yml') {
    return 'yaml';
  }
  if (hint === 'json') {
    return 'json';
  }
  if (hint === 'shell' || hint === 'sh' || hint === 'bash') {
    return 'shell';
  }
  if (hint === 'dockerfile') {
    return 'dockerfile';
  }
  if (hint === 'dotenv' || hint === 'ini' || hint === 'toml' || hint === 'properties' || hint === 'conf') {
    return 'ini';
  }

  if (options.fileKind === 'compose') {
    return 'yaml';
  }
  if (options.fileKind === 'env') {
    return 'ini';
  }
  if (options.fileKind === 'config') {
    if (extension === 'json') {
      return 'json';
    }
    if (extension === 'sh') {
      return 'shell';
    }
    if (extension === 'dockerfile') {
      return 'dockerfile';
    }
    return 'ini';
  }

  if (extension === 'json') {
    return 'json';
  }
  if (extension === 'yaml' || extension === 'yml') {
    return 'yaml';
  }
  if (
    extension === 'env' ||
    extension === 'ini' ||
    extension === 'toml' ||
    extension === 'properties' ||
    extension === 'conf'
  ) {
    return 'ini';
  }
  if (extension === 'sh') {
    return 'shell';
  }

  return 'plaintext';
}

export function canOpenWorkspaceFile(item: Pick<ProjectWorkspaceTreeItem, 'node_type'>) {
  return item.node_type === 'file';
}
