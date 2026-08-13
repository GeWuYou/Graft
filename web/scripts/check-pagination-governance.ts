import { readdirSync, readFileSync } from 'node:fs';
import { extname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

function resolveDefaultRootDir() {
  try {
    return fileURLToPath(new URL('..', import.meta.url));
  } catch {
    return process.cwd();
  }
}

const DEFAULT_ROOT_DIR = resolveDefaultRootDir();
const SCANNED_EXTENSIONS = new Set(['.ts', '.tsx', '.vue']);
const EXCLUDED_DIRS = new Set(['node_modules', 'dist', 'coverage', 'mock', '__mocks__', 'assets', 'ai-libs']);
const MANAGEMENT_TABLE_TAG_PATTERN =
  /<(?:management-paged-table|advanced-query-paged-table)\b(?<attributes>[\s\S]*?)>/gi;
const TABLE_ROWS_BINDING_PATTERN = /:(?:rows|data)\s*=\s*"(?<expression>[^"]+)"/gi;
const DERIVED_LIST_NAME_PATTERN = /\b(?:filtered|paged)[A-Z_$][\w$]*/g;
const EXEMPT_PATH_PATTERNS = [
  /\/pages\/detail\//,
  /\/pages\/import\//,
  /(?:^|\/)(?:app-log|access-log|audit)(?:\/|$)/,
  /(?:preview|inspect)/i,
];

type PaginationGovernanceAuditOptions = {
  rootDir?: string;
  srcDir?: string;
};

type PaginationGovernanceFinding = {
  file: string;
  line: number;
  method: 'filter' | 'slice';
  variable: string;
};

type TableRowsBinding = {
  expression: string;
  index: number;
};

function walk(dir: string): string[] {
  const entries = readdirSync(dir, { withFileTypes: true });
  const files: string[] = [];

  for (const entry of entries) {
    if (EXCLUDED_DIRS.has(entry.name)) {
      continue;
    }

    const fullPath = join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walk(fullPath));
      continue;
    }

    if (SCANNED_EXTENSIONS.has(extname(fullPath))) {
      files.push(fullPath);
    }
  }

  return files;
}

function shouldScanFile(rootDir: string, file: string) {
  const normalized = relative(rootDir, file).replaceAll('\\', '/');
  if (!normalized.startsWith('src/')) {
    return false;
  }
  if (normalized.startsWith('src/contracts/openapi/generated/')) {
    return false;
  }
  if (/\.(?:test|spec)\.(?:ts|tsx|vue)$/.test(normalized)) {
    return false;
  }
  return !EXEMPT_PATH_PATTERNS.some((pattern) => pattern.test(normalized));
}

function lineNumberForIndex(source: string, index: number) {
  return source.slice(0, index).split('\n').length;
}

function collectTableRowsBindings(source: string) {
  const bindings: TableRowsBinding[] = [];

  for (const tagMatch of source.matchAll(MANAGEMENT_TABLE_TAG_PATTERN)) {
    const attributes = tagMatch.groups?.attributes ?? '';
    for (const bindingMatch of attributes.matchAll(TABLE_ROWS_BINDING_PATTERN)) {
      bindings.push({
        expression: bindingMatch.groups?.expression ?? '',
        index: (tagMatch.index ?? 0) + (bindingMatch.index ?? 0),
      });
    }
  }

  return bindings;
}

function findDerivedListMethod(source: string, variable: string) {
  const declarationPattern = new RegExp(
    `\\b(?:const|let)\\s+${variable}\\s*=\\s*(?:computed(?:<[^>]*>)?\\s*\\()?`,
    'g',
  );
  const declaration = declarationPattern.exec(source);
  if (!declaration) {
    return null;
  }

  // 限制声明搜索范围，避免同文件后续无关 helper 的数组操作被误判为该表格数据源。
  const declarationSource = source.slice(declaration.index, declaration.index + 2400);
  const filterIndex = declarationSource.search(/\.filter\s*\(/);
  const sliceIndex = declarationSource.search(/\.slice\s*\(/);
  if (filterIndex < 0 && sliceIndex < 0) {
    return null;
  }

  if (filterIndex >= 0 && (sliceIndex < 0 || filterIndex < sliceIndex)) {
    return { index: declaration.index + filterIndex, method: 'filter' as const };
  }
  return { index: declaration.index + sliceIndex, method: 'slice' as const };
}

function collectFindings(rootDir: string, srcDir: string): PaginationGovernanceFinding[] {
  const findings: PaginationGovernanceFinding[] = [];

  for (const file of walk(srcDir)) {
    if (!shouldScanFile(rootDir, file)) {
      continue;
    }

    const source = readFileSync(file, 'utf8');
    const bindings = collectTableRowsBindings(source);
    const variables = new Set<string>();
    for (const binding of bindings) {
      const directMethod = binding.expression.match(/\.(filter|slice)\s*\(/)?.[1] as
        PaginationGovernanceFinding['method'] | undefined;
      if (directMethod) {
        findings.push({
          file: relative(rootDir, file).replaceAll('\\', '/'),
          line: lineNumberForIndex(source, binding.index),
          method: directMethod,
          variable: '<table rows binding>',
        });
      }
      for (const nameMatch of binding.expression.matchAll(DERIVED_LIST_NAME_PATTERN)) {
        variables.add(nameMatch[0]);
      }
    }
    for (const variable of variables) {
      const result = findDerivedListMethod(source, variable);
      if (!result) {
        continue;
      }
      findings.push({
        file: relative(rootDir, file).replaceAll('\\', '/'),
        line: lineNumberForIndex(source, result.index),
        method: result.method,
        variable,
      });
    }
  }

  return findings.sort((left, right) =>
    `${left.file}:${left.line}:${left.variable}`.localeCompare(`${right.file}:${right.line}:${right.variable}`),
  );
}

/** 扫描管理表格是否把本地搜索或分页后的派生列表作为运行时数据源。 */
export function runPaginationGovernanceAudit(options: PaginationGovernanceAuditOptions = {}) {
  const rootDir = options.rootDir ?? DEFAULT_ROOT_DIR;
  const srcDir = options.srcDir ?? join(rootDir, 'src');
  const debt = collectFindings(rootDir, srcDir);

  let output = 'Pagination governance inventory:\n';
  if (debt.length === 0) {
    output += 'Pagination governance: no table-bound local filtering or pagination found.\n';
    return { debt, output };
  }

  output += 'Table-bound local filtering or pagination found:\n';
  for (const item of debt) {
    output += `- ${item.file}:${item.line} ${item.variable}.${item.method}()\n`;
  }
  output +=
    '\nManagement resource lists must send their filters and limit/offset to the server. Detail previews, log windows, and import previews are exempt from this runtime-list rule.\n';

  return { debt, output };
}

if (import.meta.main) {
  const result = runPaginationGovernanceAudit();
  process.stdout.write(result.output);
  if (result.debt.length > 0) {
    process.exitCode = 1;
  }
}
