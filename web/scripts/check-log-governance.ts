import { readdirSync, readFileSync } from 'node:fs';
import { extname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT_DIR = fileURLToPath(new URL('..', import.meta.url));
const SRC_DIR = join(ROOT_DIR, 'src');
const CONSOLE_PATTERN = /\bconsole\.(?:log|debug|info|warn|error|trace|group|groupCollapsed|groupEnd|table)\b/;
const CONSOLA_IMPORT_PATTERN = /from\s+['"]consola['"]/;
const ALLOWED_CONSOLA_IMPORT = 'src/utils/logger/transports/consola.ts';
const EXCLUDED_DIRS = new Set(['node_modules', 'dist', 'coverage', 'mock', '__mocks__', 'ai-libs', 'assets']);
const SOURCE_EXTENSIONS = new Set(['.ts', '.tsx', '.vue', '.js', '.mjs', '.cjs']);

type Finding = {
  file: string;
  detail: string;
};

function walk(dir: string): string[] {
  const files: string[] = [];

  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (EXCLUDED_DIRS.has(entry.name)) {
      continue;
    }

    const file = join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walk(file));
      continue;
    }

    const normalized = relative(ROOT_DIR, file).replaceAll('\\', '/');
    if (
      SOURCE_EXTENSIONS.has(extname(file)) &&
      !/\.(?:test|spec)\.(?:ts|tsx|js|mjs|cjs)$/.test(normalized) &&
      !normalized.endsWith('.d.ts')
    ) {
      files.push(file);
    }
  }

  return files;
}

const findings: Finding[] = [];

for (const file of walk(SRC_DIR)) {
  const relativePath = relative(ROOT_DIR, file).replaceAll('\\', '/');
  const source = readFileSync(file, 'utf8');

  if (CONSOLE_PATTERN.test(source)) {
    findings.push({ file: relativePath, detail: 'uses a raw console method' });
  }
  if (relativePath !== ALLOWED_CONSOLA_IMPORT && CONSOLA_IMPORT_PATTERN.test(source)) {
    findings.push({ file: relativePath, detail: 'imports consola outside the logger transport' });
  }
}

if (findings.length === 0) {
  process.stdout.write('Log governance: no raw console or direct consola usage found.\n');
} else {
  process.stderr.write('Log governance violations:\n');
  for (const finding of findings) {
    process.stderr.write(`- ${finding.file}: ${finding.detail}\n`);
  }
  process.exitCode = 1;
}
