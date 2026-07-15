import type { ProjectWorkspaceFileKind, ProjectWorkspaceLanguageHint } from '../types/project';

export type ProjectWorkspaceMonacoLanguage =
  'dockerfile' | 'hcl' | 'ini' | 'json' | 'markdown' | 'plaintext' | 'powershell' | 'shell' | 'sql' | 'xml' | 'yaml';

const WORKSPACE_LANGUAGE_HINT_TO_MONACO_LANGUAGE: Partial<
  Record<ProjectWorkspaceLanguageHint, ProjectWorkspaceMonacoLanguage>
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

const WORKSPACE_FILE_NAME_TO_MONACO_LANGUAGE: Record<string, ProjectWorkspaceMonacoLanguage> = {
  '.editorconfig': 'ini',
  '.gitattributes': 'plaintext',
  '.gitconfig': 'ini',
  '.gitignore': 'plaintext',
  caddyfile: 'plaintext',
  dockerfile: 'dockerfile',
  makefile: 'plaintext',
};

const WORKSPACE_EXTENSION_TO_MONACO_LANGUAGE: Record<string, ProjectWorkspaceMonacoLanguage> = {
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

/**
 * 判断当前内容与已保存内容是否存在未保存的变化。
 *
 * @param current - 当前内容
 * @param saved - 已保存的内容
 * @returns `true` 表示标准化后的内容不同，`false` 表示相同
 */
export function hasWorkspaceUnsavedChanges(current: string, saved: string) {
  return normalizeWorkspaceContent(current) !== normalizeWorkspaceContent(saved);
}

/**
 * 判断工作区语言是否支持显式语法校验。
 *
 * @param language - 要检查的 Monaco 语言标识
 * @returns `true` 表示语言支持显式语法校验，`false` 表示不支持
 */
export function supportsExplicitWorkspaceSyntaxValidation(language: ProjectWorkspaceMonacoLanguage) {
  return language === 'json' || language === 'yaml';
}

/**
 * 从路径中解析工作区文件名。
 *
 * @param path - 文件路径；为空时使用 `untitled`
 * @returns 路径末尾的文件名，或 `untitled`
 */
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
  if (hint && WORKSPACE_LANGUAGE_HINT_TO_MONACO_LANGUAGE[hint as ProjectWorkspaceLanguageHint]) {
    return WORKSPACE_LANGUAGE_HINT_TO_MONACO_LANGUAGE[hint as ProjectWorkspaceLanguageHint]!;
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
