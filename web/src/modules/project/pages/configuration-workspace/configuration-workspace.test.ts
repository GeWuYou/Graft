import { describe, expect, it } from 'vitest';

import {
  hasWorkspaceUnsavedChanges,
  normalizeWorkspaceContent,
  resolveWorkspaceFileName,
  resolveWorkspaceMonacoLanguage,
} from '../../shared/configuration-workspace';

describe('configuration workspace helpers', () => {
  it('normalizes line endings without trimming file content', () => {
    expect(normalizeWorkspaceContent('services:\r\n  api:\r\n')).toBe('services:\n  api:\n');
  });

  it('detects dirty buffers from normalized content', () => {
    expect(hasWorkspaceUnsavedChanges('APP_ENV=prod\r\n', 'APP_ENV=prod\n')).toBe(false);
    expect(hasWorkspaceUnsavedChanges('APP_ENV=prod\n', 'APP_ENV=dev\n')).toBe(true);
  });

  it('resolves monaco language from backend hints and fallback file metadata', () => {
    expect(
      resolveWorkspaceMonacoLanguage({ fileKind: 'compose', languageHint: 'yaml', path: 'docker-compose.yml' }),
    ).toBe('yaml');
    expect(resolveWorkspaceMonacoLanguage({ fileKind: 'env', languageHint: 'dotenv', path: '.env' })).toBe('ini');
    expect(resolveWorkspaceMonacoLanguage({ fileKind: 'config', languageHint: 'json', path: 'app.json' })).toBe('json');
    expect(resolveWorkspaceMonacoLanguage({ fileKind: 'config', languageHint: null, path: 'Dockerfile' })).toBe(
      'dockerfile',
    );
    expect(resolveWorkspaceMonacoLanguage({ fileKind: 'config', languageHint: null, path: 'terraform/main.tf' })).toBe(
      'hcl',
    );
    expect(resolveWorkspaceMonacoLanguage({ fileKind: 'config', languageHint: null, path: 'scripts/app.ps1' })).toBe(
      'powershell',
    );
    expect(resolveWorkspaceMonacoLanguage({ fileKind: 'config', languageHint: null, path: 'config/app.xml' })).toBe(
      'xml',
    );
    expect(resolveWorkspaceMonacoLanguage({ fileKind: 'config', languageHint: null, path: 'queries/seed.sql' })).toBe(
      'sql',
    );
    expect(resolveWorkspaceMonacoLanguage({ fileKind: 'text', languageHint: null, path: 'docs/README.md' })).toBe(
      'markdown',
    );
    expect(resolveWorkspaceMonacoLanguage({ fileKind: 'config', languageHint: null, path: '.editorconfig' })).toBe(
      'ini',
    );
    expect(resolveWorkspaceMonacoLanguage({ fileKind: 'text', languageHint: null, path: '.gitignore' })).toBe(
      'plaintext',
    );
    expect(resolveWorkspaceMonacoLanguage({ fileKind: 'config', languageHint: null, path: 'app.jsonc' })).toBe('json');
  });

  it('extracts a presentable file name from a relative path', () => {
    expect(resolveWorkspaceFileName('config/.env')).toBe('.env');
  });
});
