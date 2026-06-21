import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';

import {
  EXCLUDED_DIRS,
  REPOSITORY_DIR,
  ROOT_DIR,
  SCANNED_EXTENSIONS,
  SERVER_KEY_DIRS,
  SRC_DIR,
  STRICT_I18N_KEY_FIRST,
} from './config';
import { buildLineIndex } from './text-utils';
import type { ScanContext, SourceFile, SourceKind } from './types';

function sourceKind(file: string): SourceKind | null {
  if (file.endsWith('.vue')) return 'vue';
  if (file.endsWith('.tsx')) return 'tsx';
  if (file.endsWith('.ts')) return 'ts';
  if (file.endsWith('.go')) return 'go';
  return null;
}

function shouldScanSourceFile(file: string): boolean {
  const normalized = relative(ROOT_DIR, file).replaceAll('\\', '/');
  const kind = sourceKind(file);
  if (!kind || !SCANNED_EXTENSIONS.has(`.${kind}`)) return false;
  if (
    /\.d\.ts$/.test(normalized) ||
    /\.test\.(?:ts|tsx|vue)$/.test(normalized) ||
    /\.spec\.(?:ts|tsx|vue)$/.test(normalized)
  ) {
    return false;
  }
  if (normalized.startsWith('src/contracts/openapi/generated/')) return false;
  return true;
}

function shouldScanServerFile(file: string): boolean {
  const normalized = relative(REPOSITORY_DIR, file).replaceAll('\\', '/');
  return (
    file.endsWith('.go') &&
    !normalized.endsWith('_test.go') &&
    (normalized.startsWith('server/internal/') || normalized.startsWith('server/modules/')) &&
    !normalized.includes('/contract/openapi/generated/')
  );
}

function walk(dir: string, shouldInclude: (file: string) => boolean): string[] {
  if (!existsSync(dir)) return [];
  const files: string[] = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (EXCLUDED_DIRS.has(entry.name)) continue;
    const fullPath = join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walk(fullPath, shouldInclude));
      continue;
    }
    if (shouldInclude(fullPath)) files.push(fullPath);
  }
  return files;
}

function readSourceFile(filePath: string, rootDir: string): SourceFile | null {
  const kind = sourceKind(filePath);
  if (!kind) return null;
  const source = readFileSync(filePath, 'utf8');
  return {
    absolutePath: filePath,
    relativePath: relative(rootDir, filePath).replaceAll('\\', '/'),
    kind,
    source,
    lineStarts: buildLineIndex(source),
  };
}

export function createScanContext(): ScanContext {
  return {
    rootDir: ROOT_DIR,
    repositoryDir: REPOSITORY_DIR,
    srcDir: SRC_DIR,
    sourceFiles: walk(SRC_DIR, shouldScanSourceFile)
      .map((file) => readSourceFile(file, ROOT_DIR))
      .filter((file): file is SourceFile => Boolean(file)),
    serverFiles: SERVER_KEY_DIRS.flatMap((dir) => walk(dir, shouldScanServerFile))
      .map((file) => readSourceFile(file, REPOSITORY_DIR))
      .filter((file): file is SourceFile => Boolean(file)),
    strictKeyFirst: STRICT_I18N_KEY_FIRST,
  };
}
