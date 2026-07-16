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
const NATIVE_DIALOG_ALLOW_COMMENT = 'native-dialog-governance: allow';
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

type ParserFrame =
  | { kind: 'code' | 'line-comment' | 'block-comment' | 'single-quote' | 'double-quote' | 'template' }
  | { kind: 'template-expression'; braceDepth: number };

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

/**
 * 将源码中的注释和字面量内容替换为空格，同时保留换行结构。
 * 模板字符串中的 `${...}` 表达式继续按代码处理，避免漏掉真实调用。
 *
 * @param source - 原始源码文本
 * @returns 处理后的源码文本
 */
function sanitizeSource(source: string) {
  let result = '';
  let index = 0;
  const frames: ParserFrame[] = [{ kind: 'code' }];

  while (index < source.length) {
    const char = source[index] ?? '';
    const next = source[index + 1] ?? '';
    const current = frames[frames.length - 1] ?? { kind: 'code' as const };

    switch (current.kind) {
      case 'code':
      case 'template-expression':
        if (char === '/' && next === '/') {
          result += '  ';
          index += 2;
          frames.push({ kind: 'line-comment' });
          continue;
        }
        if (char === '/' && next === '*') {
          result += '  ';
          index += 2;
          frames.push({ kind: 'block-comment' });
          continue;
        }
        if (char === "'") {
          result += ' ';
          index += 1;
          frames.push({ kind: 'single-quote' });
          continue;
        }
        if (char === '"') {
          result += ' ';
          index += 1;
          frames.push({ kind: 'double-quote' });
          continue;
        }
        if (char === '`') {
          result += ' ';
          index += 1;
          frames.push({ kind: 'template' });
          continue;
        }
        if (current.kind === 'template-expression') {
          if (char === '{') {
            result += '{';
            current.braceDepth += 1;
            index += 1;
            continue;
          }
          if (char === '}') {
            if (current.braceDepth === 1) {
              result += ' ';
              frames.pop();
            } else {
              result += '}';
              current.braceDepth -= 1;
            }
            index += 1;
            continue;
          }
        }
        result += char;
        index += 1;
        continue;
      case 'line-comment':
        result += char === '\n' ? '\n' : ' ';
        index += 1;
        if (char === '\n') {
          frames.pop();
        }
        continue;
      case 'block-comment':
        if (char === '*' && next === '/') {
          result += '  ';
          index += 2;
          frames.pop();
          continue;
        }
        result += char === '\n' ? '\n' : ' ';
        index += 1;
        continue;
      case 'single-quote':
        if (char === '\\') {
          result += next === '\n' ? ' \n' : '  ';
          index += Math.min(2, source.length - index);
          continue;
        }
        result += char === '\n' ? '\n' : ' ';
        index += 1;
        if (char === "'") {
          frames.pop();
        }
        continue;
      case 'double-quote':
        if (char === '\\') {
          result += next === '\n' ? ' \n' : '  ';
          index += Math.min(2, source.length - index);
          continue;
        }
        result += char === '\n' ? '\n' : ' ';
        index += 1;
        if (char === '"') {
          frames.pop();
        }
        continue;
      case 'template':
        if (char === '\\') {
          result += next === '\n' ? ' \n' : '  ';
          index += Math.min(2, source.length - index);
          continue;
        }
        if (char === '$' && next === '{') {
          result += '  ';
          index += 2;
          frames.push({ kind: 'template-expression', braceDepth: 1 });
          continue;
        }
        result += char === '\n' ? '\n' : ' ';
        index += 1;
        if (char === '`') {
          frames.pop();
        }
        continue;
    }
  }

  return result;
}

/**
 * 计算源码在指定位置之前的解析状态栈。
 *
 * @param source - 原始源码文本
 * @param targetIndex - 目标字符位置
 * @returns 解析到目标位置前的状态栈
 */
function parserFramesAtIndex(source: string, targetIndex: number): ParserFrame[] {
  const frames: ParserFrame[] = [{ kind: 'code' }];
  let index = 0;

  while (index < targetIndex) {
    const char = source[index] ?? '';
    const next = source[index + 1] ?? '';
    const current = frames[frames.length - 1] ?? { kind: 'code' as const };

    switch (current.kind) {
      case 'code':
      case 'template-expression':
        if (char === '/' && next === '/') {
          index += 2;
          frames.push({ kind: 'line-comment' });
          continue;
        }
        if (char === '/' && next === '*') {
          index += 2;
          frames.push({ kind: 'block-comment' });
          continue;
        }
        if (char === "'") {
          index += 1;
          frames.push({ kind: 'single-quote' });
          continue;
        }
        if (char === '"') {
          index += 1;
          frames.push({ kind: 'double-quote' });
          continue;
        }
        if (char === '`') {
          index += 1;
          frames.push({ kind: 'template' });
          continue;
        }
        if (current.kind === 'template-expression') {
          if (char === '{') {
            current.braceDepth += 1;
            index += 1;
            continue;
          }
          if (char === '}') {
            if (current.braceDepth === 1) {
              frames.pop();
            } else {
              current.braceDepth -= 1;
            }
            index += 1;
            continue;
          }
        }
        index += 1;
        continue;
      case 'line-comment':
        index += 1;
        if (char === '\n') {
          frames.pop();
        }
        continue;
      case 'block-comment':
        if (char === '*' && next === '/') {
          index += 2;
          frames.pop();
          continue;
        }
        index += 1;
        continue;
      case 'single-quote':
        if (char === '\\') {
          index += Math.min(2, source.length - index);
          continue;
        }
        index += 1;
        if (char === "'") {
          frames.pop();
        }
        continue;
      case 'double-quote':
        if (char === '\\') {
          index += Math.min(2, source.length - index);
          continue;
        }
        index += 1;
        if (char === '"') {
          frames.pop();
        }
        continue;
      case 'template':
        if (char === '\\') {
          index += Math.min(2, source.length - index);
          continue;
        }
        if (char === '$' && next === '{') {
          index += 2;
          frames.push({ kind: 'template-expression', braceDepth: 1 });
          continue;
        }
        index += 1;
        if (char === '`') {
          frames.pop();
        }
        continue;
    }
  }

  return frames;
}

/**
 * 判断命中位置之后是否存在真实的行注释豁免标记。
 *
 * @param source - 原始源码文本
 * @param index - 命中的字符位置
 * @returns 若匹配位置后的真实行注释包含 allow 标记则返回 `true`，否则返回 `false`。
 */
function hasAllowCommentAfterIndex(source: string, index: number) {
  const frames = parserFramesAtIndex(source, index);
  const lineEnd = source.indexOf('\n', index);
  const end = lineEnd === -1 ? source.length : lineEnd;
  let cursor = index;

  while (cursor < end) {
    const char = source[cursor] ?? '';
    const next = source[cursor + 1] ?? '';
    const current = frames[frames.length - 1] ?? { kind: 'code' as const };

    switch (current.kind) {
      case 'code':
      case 'template-expression':
        if (char === '/' && next === '/') {
          return source.slice(cursor + 2, end).includes(NATIVE_DIALOG_ALLOW_COMMENT);
        }
        if (char === '/' && next === '*') {
          cursor += 2;
          frames.push({ kind: 'block-comment' });
          continue;
        }
        if (char === "'") {
          cursor += 1;
          frames.push({ kind: 'single-quote' });
          continue;
        }
        if (char === '"') {
          cursor += 1;
          frames.push({ kind: 'double-quote' });
          continue;
        }
        if (char === '`') {
          cursor += 1;
          frames.push({ kind: 'template' });
          continue;
        }
        if (current.kind === 'template-expression') {
          if (char === '{') {
            current.braceDepth += 1;
            cursor += 1;
            continue;
          }
          if (char === '}') {
            if (current.braceDepth === 1) {
              frames.pop();
            } else {
              current.braceDepth -= 1;
            }
            cursor += 1;
            continue;
          }
        }
        cursor += 1;
        continue;
      case 'line-comment':
        return source.slice(cursor, end).includes(NATIVE_DIALOG_ALLOW_COMMENT);
      case 'block-comment':
        if (char === '*' && next === '/') {
          cursor += 2;
          frames.pop();
          continue;
        }
        cursor += 1;
        continue;
      case 'single-quote':
        if (char === '\\') {
          cursor += Math.min(2, end - cursor);
          continue;
        }
        cursor += 1;
        if (char === "'") {
          frames.pop();
        }
        continue;
      case 'double-quote':
        if (char === '\\') {
          cursor += Math.min(2, end - cursor);
          continue;
        }
        cursor += 1;
        if (char === '"') {
          frames.pop();
        }
        continue;
      case 'template':
        if (char === '\\') {
          cursor += Math.min(2, end - cursor);
          continue;
        }
        if (char === '$' && next === '{') {
          cursor += 2;
          frames.push({ kind: 'template-expression', braceDepth: 1 });
          continue;
        }
        cursor += 1;
        if (char === '`') {
          frames.pop();
        }
        continue;
    }
  }

  return false;
}

function shouldIgnoreLine(source: string, index: number) {
  return hasAllowCommentAfterIndex(source, index);
}

function collectInventory(rootDir: string, srcDir: string): InventoryItem[] {
  const inventory: InventoryItem[] = [];

  for (const file of walk(srcDir)) {
    if (!shouldScanFile(rootDir, file)) {
      continue;
    }

    const rel = relative(rootDir, file).replaceAll('\\', '/');
    const source = readFileSync(file, 'utf8');
    const sanitizedSource = sanitizeSource(source);
    for (const pattern of NATIVE_DIALOG_PATTERNS) {
      for (const match of sanitizedSource.matchAll(pattern)) {
        const index = match.index ?? 0;
        if (shouldIgnoreLine(source, index)) {
          continue;
        }
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

/** 扫描浏览器原生对话框调用，并生成供治理门禁消费的命中报告。 */
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
