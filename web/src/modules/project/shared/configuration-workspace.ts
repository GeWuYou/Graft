import type { ApplicationWorkspaceFileKind, ApplicationWorkspaceLanguageHint } from '../types/project';

export type ApplicationWorkspaceMonacoLanguage =
  'dockerfile' | 'hcl' | 'ini' | 'json' | 'markdown' | 'plaintext' | 'powershell' | 'shell' | 'sql' | 'xml' | 'yaml';

const WORKSPACE_LANGUAGE_HINT_TO_MONACO_LANGUAGE: Partial<
  Record<ApplicationWorkspaceLanguageHint, ApplicationWorkspaceMonacoLanguage>
> = {
  dockerfile: 'dockerfile',
  dotenv: 'ini',
  hcl: 'hcl',
  ini: 'ini',
  json: 'json',
  markdown: 'markdown',
  plaintext: 'plaintext',
  powershell: 'powershell',
  properties: 'ini',
  shell: 'shell',
  sql: 'sql',
  toml: 'ini',
  xml: 'xml',
  yaml: 'yaml',
};

const WORKSPACE_FILE_NAME_TO_MONACO_LANGUAGE: Record<string, ApplicationWorkspaceMonacoLanguage> = {
  '.editorconfig': 'ini',
  '.gitattributes': 'plaintext',
  '.gitconfig': 'ini',
  '.gitignore': 'plaintext',
  caddyfile: 'plaintext',
  dockerfile: 'dockerfile',
  makefile: 'plaintext',
};

const WORKSPACE_EXTENSION_TO_MONACO_LANGUAGE: Record<string, ApplicationWorkspaceMonacoLanguage> = {
  bash: 'shell',
  cfg: 'ini',
  conf: 'ini',
  dockerfile: 'dockerfile',
  hcl: 'hcl',
  ini: 'ini',
  json: 'json',
  jsonc: 'json',
  log: 'plaintext',
  markdown: 'markdown',
  md: 'markdown',
  properties: 'ini',
  ps1: 'powershell',
  psd1: 'powershell',
  psm1: 'powershell',
  sh: 'shell',
  sql: 'sql',
  tf: 'hcl',
  tfvars: 'hcl',
  toml: 'ini',
  txt: 'plaintext',
  xml: 'xml',
  yaml: 'yaml',
  yml: 'yaml',
  zsh: 'shell',
};

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

export function supportsExplicitWorkspaceSyntaxValidation(language: ApplicationWorkspaceMonacoLanguage) {
  return language === 'json' || language === 'yaml';
}

export function resolveWorkspaceFileName(path: string) {
  const normalized = String(path || '').trim();
  if (!normalized) {
    return 'untitled';
  }
  return normalized.split('/').at(-1) || normalized;
}

export function resolveWorkspaceMonacoLanguage(options: {
  fileKind?: ApplicationWorkspaceFileKind | null;
  languageHint?: ApplicationWorkspaceLanguageHint | null;
  path?: string;
}): ApplicationWorkspaceMonacoLanguage {
  const hint = String(options.languageHint || '')
    .trim()
    .toLowerCase();
  const fileName = resolveWorkspaceFileName(options.path || '').toLowerCase();
  const extension = String(options.path || '')
    .split('.')
    .at(-1)
    ?.trim()
    .toLowerCase();

  if (hint === 'yaml' || hint === 'yml') {
    return 'yaml';
  }
  if (hint === 'shell' || hint === 'sh' || hint === 'bash' || hint === 'zsh') {
    return 'shell';
  }
  if (hint === 'conf' || hint === 'cfg') {
    return 'ini';
  }
  if (hint && WORKSPACE_LANGUAGE_HINT_TO_MONACO_LANGUAGE[hint as ApplicationWorkspaceLanguageHint]) {
    return WORKSPACE_LANGUAGE_HINT_TO_MONACO_LANGUAGE[hint as ApplicationWorkspaceLanguageHint]!;
  }

  if (options.fileKind === 'compose') {
    return 'yaml';
  }
  if (options.fileKind === 'env') {
    return 'ini';
  }
  if (options.fileKind === 'config') {
    if (WORKSPACE_FILE_NAME_TO_MONACO_LANGUAGE[fileName]) {
      return WORKSPACE_FILE_NAME_TO_MONACO_LANGUAGE[fileName];
    }
    if (WORKSPACE_EXTENSION_TO_MONACO_LANGUAGE[extension || '']) {
      return WORKSPACE_EXTENSION_TO_MONACO_LANGUAGE[extension || ''];
    }
    return 'ini';
  }

  if (fileName === '.env' || fileName.startsWith('.env.')) {
    return 'ini';
  }
  if (WORKSPACE_FILE_NAME_TO_MONACO_LANGUAGE[fileName]) {
    return WORKSPACE_FILE_NAME_TO_MONACO_LANGUAGE[fileName];
  }
  if (WORKSPACE_EXTENSION_TO_MONACO_LANGUAGE[extension || '']) {
    return WORKSPACE_EXTENSION_TO_MONACO_LANGUAGE[extension || ''];
  }

  return 'plaintext';
}
