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
const SCANNED_EXTENSIONS = new Set(['.ts', '.tsx', '.js', '.jsx', '.vue']);
const EXCLUDED_DIRS = new Set(['node_modules', 'dist', 'coverage', 'mock', '__mocks__', 'assets', 'ai-libs']);
const NATIVE_DIALOG_PATTERNS = [
  /(?<![\w$.])(?<callee>alert|confirm|prompt)\s*\(/g,
  /(?<![\w$])(?:window|globalThis)\.(?<callee>alert|confirm|prompt)\s*\(/g,
];

type NativeDialogGovernanceAuditOptions = {
  rootDir?: string;
  srcDir?: string;
};

type InventoryItem = {
  callee: string;
  file: string;
  line: number;
  snippet: string;
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
  return !/\.(?:test|spec)\.(?:ts|tsx|js|jsx|vue)$/.test(normalized);
}

function lineNumberForIndex(source: string, index: number) {
  return source.slice(0, index).split('\n').length;
}

function snippetForIndex(source: string, index: number) {
  const line = source.split('\n')[lineNumberForIndex(source, index) - 1] ?? '';
  return line.trim();
}

function collectInventory(rootDir: string, srcDir: string): InventoryItem[] {
  const inventory: InventoryItem[] = [];

  for (const file of walk(srcDir)) {
    if (!shouldScanFile(rootDir, file)) {
      continue;
    }

    const rel = relative(rootDir, file).replaceAll('\\', '/');
    const source = readFileSync(file, 'utf8');
    for (const pattern of NATIVE_DIALOG_PATTERNS) {
      for (const match of source.matchAll(pattern)) {
        const index = match.index ?? 0;
        inventory.push({
          callee: match.groups?.callee ?? '<unknown>',
          file: rel,
          line: lineNumberForIndex(source, index),
          snippet: snippetForIndex(source, index),
        });
      }
    }
  }

  return inventory;
}

export function runNativeDialogGovernanceAudit(options: NativeDialogGovernanceAuditOptions = {}) {
  const rootDir = options.rootDir ?? DEFAULT_ROOT_DIR;
  const srcDir = options.srcDir ?? join(rootDir, 'src');
  const debt = collectInventory(rootDir, srcDir);

  let output = 'Native dialog governance inventory:\n';
  if (debt.length === 0) {
    output += 'Native dialog governance: no browser-native dialogs found.\n';
    return { debt, output };
  }

  output += 'Blacklisted browser-native dialogs found:\n';
  for (const item of debt) {
    output += `- ${item.file}:${item.line} ${item.callee} -> ${item.snippet}\n`;
  }
  output +=
    '\nUse TDesign `t-dialog` or `DialogPlugin` for runtime confirmation, alert, and prompt flows. Browser-native `alert` / `confirm` / `prompt` are forbidden in `web/src`.\n';

  return { debt, output };
}

if (import.meta.main) {
  const result = runNativeDialogGovernanceAudit();
  process.stdout.write(result.output);
  if (result.debt.length > 0) {
    process.exitCode = 1;
  }
}
