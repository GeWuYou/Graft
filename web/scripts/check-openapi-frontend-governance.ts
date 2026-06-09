// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';

const ROOT_DIR = new URL('..', import.meta.url);
const SRC_DIR = join(ROOT_DIR.pathname, 'src');

const ALLOWED_RUNTIME_FILES = new Set([
  'src/utils/request.ts',
  'src/contracts/api/envelope.ts',
  'src/types/axios.d.ts',
]);

const DTO_NAME_PATTERN = /\b(?:export\s+)?(?:interface|type)\s+([A-Za-z0-9_]*(?:Request|Response|DTO|Payload))\b/g;
const GENERATED_IMPORT_PATTERN =
  /from\s+['"]@\/contracts\/openapi\/generated\/schema['"]|from\s+['"]\.\.\/(?:\.\.\/)*contracts\/openapi\/generated\/schema['"]/;
const GENERATED_BOUNDARY_USAGE_PATTERN = /(?:paths\s*\[|components\s*\[\s*['"]schemas['"]\s*\])/;
const REQUEST_METHOD_CALL_PATTERN = /\brequest\.(?:get|post|put|delete)\s*(?:<|\()/;
const FETCH_PATTERN = /\bfetch\s*\(/;
const AXIOS_CREATE_PATTERN = /\baxios\.create\s*\(/;
const AXIOS_IMPORT_PATTERN = /from\s+['"]axios['"]/;
const GENERATED_RUNTIME_IMPORT_PATTERN =
  /from\s+['"][^'"]*(?:generated[^'"]*(?:client|runtime)|client[^'"]*generated|runtime[^'"]*generated)[^'"]*['"]/;
const REQUEST_IMPORT_PATTERN = /import\s*\{\s*([^}]+)\s*\}\s*from\s*['"]@\/utils\/request['"]/g;

type Finding = {
  file: string;
  detail: string;
};

function walk(dir: string): string[] {
  const entries = readdirSync(dir, { withFileTypes: true });
  const files: string[] = [];

  for (const entry of entries) {
    if (entry.name === 'node_modules' || entry.name === 'dist' || entry.name === 'coverage' || entry.name === '.tmp') {
      continue;
    }

    const fullPath = join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walk(fullPath));
      continue;
    }

    if (!/\.(ts|tsx|vue|d\.ts)$/.test(entry.name)) {
      continue;
    }

    files.push(fullPath);
  }

  return files;
}

function isPageOrStore(relativePath: string) {
  return relativePath.startsWith('src/store/') || relativePath.includes('/pages/') || relativePath.includes('/store/');
}

function isPotentialDtoFile(relativePath: string) {
  if (ALLOWED_RUNTIME_FILES.has(relativePath)) {
    return false;
  }

  return (
    relativePath.startsWith('src/modules/') ||
    relativePath.startsWith('src/contracts/') ||
    relativePath.startsWith('src/types/')
  );
}

function collectFindings() {
  const files = walk(SRC_DIR);
  const staleManualDtos: Finding[] = [];
  const pageStoreDirectRequestCalls: Finding[] = [];
  const pageStoreRequestHelperCoupling: Finding[] = [];
  const runtimeBypasses: Finding[] = [];

  for (const file of files) {
    const rel = relative(ROOT_DIR.pathname, file).replaceAll('\\', '/');
    const source = readFileSync(file, 'utf8');

    if (isPotentialDtoFile(rel)) {
      const matches = [...source.matchAll(DTO_NAME_PATTERN)];
      const usesGeneratedBoundary =
        GENERATED_IMPORT_PATTERN.test(source) || GENERATED_BOUNDARY_USAGE_PATTERN.test(source);

      if (matches.length > 0 && !usesGeneratedBoundary) {
        for (const match of matches) {
          staleManualDtos.push({
            file: rel,
            detail: `suspected stale manual API DTO: ${match[1]}`,
          });
        }
      }
    }

    if (isPageOrStore(rel)) {
      if (REQUEST_METHOD_CALL_PATTERN.test(source)) {
        pageStoreDirectRequestCalls.push({
          file: rel,
          detail: 'direct request.<method>() call from page/store',
        });
      }

      for (const match of source.matchAll(REQUEST_IMPORT_PATTERN)) {
        const importedNames = match[1]
          .split(',')
          .map((name) => name.trim())
          .filter(Boolean);
        if (importedNames.some((name) => /^(request)$/.test(name) || /^request\s+as\s+/.test(name))) {
          pageStoreRequestHelperCoupling.push({
            file: rel,
            detail: `imports request from @/utils/request: ${importedNames.join(', ')}`,
          });
        } else if (importedNames.length > 0) {
          pageStoreRequestHelperCoupling.push({
            file: rel,
            detail: `imports request.ts helper(s): ${importedNames.join(', ')}`,
          });
        }
      }
    }

    if (rel !== 'src/utils/request.ts') {
      if (FETCH_PATTERN.test(source)) {
        runtimeBypasses.push({
          file: rel,
          detail: 'uses fetch() outside request.ts',
        });
      }

      if (AXIOS_CREATE_PATTERN.test(source)) {
        runtimeBypasses.push({
          file: rel,
          detail: 'uses axios.create() outside request.ts',
        });
      }

      if (GENERATED_RUNTIME_IMPORT_PATTERN.test(source)) {
        runtimeBypasses.push({
          file: rel,
          detail: 'imports suspected generated runtime client',
        });
      }

      if (AXIOS_IMPORT_PATTERN.test(source) && !rel.endsWith('axios.d.ts')) {
        runtimeBypasses.push({
          file: rel,
          detail: 'imports axios outside request.ts/types boundary',
        });
      }
    }
  }

  return {
    staleManualDtos,
    pageStoreDirectRequestCalls,
    pageStoreRequestHelperCoupling,
    runtimeBypasses,
  };
}

function printSection(title: string, findings: Finding[]) {
  if (findings.length === 0) {
    process.stdout.write(`${title}: none\n`);
    return;
  }

  process.stdout.write(`${title}:\n`);
  for (const finding of findings) {
    process.stdout.write(`- ${finding.file}: ${finding.detail}\n`);
  }
}

const findings = collectFindings();

printSection('Suspected stale manual API DTO', findings.staleManualDtos);
printSection('Page/store direct request calls', findings.pageStoreDirectRequestCalls);
printSection('Page/store request.ts coupling', findings.pageStoreRequestHelperCoupling);
printSection('Runtime bypasses', findings.runtimeBypasses);

if (
  findings.staleManualDtos.length > 0 ||
  findings.pageStoreDirectRequestCalls.length > 0 ||
  findings.runtimeBypasses.length > 0
) {
  process.exitCode = 1;
}
