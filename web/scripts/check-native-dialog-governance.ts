import { readdirSync, readFileSync } from 'node:fs';
import { extname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

/**
 * 解析默认仓库根目录。
 *
 * @returns 当前模块上一级目录；解析失败时返回当前工作目录。
 */
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

/**
 * 递归收集目录中符合条件的文件路径。
 *
 * @param dir - 要遍历的目录
 * @returns 符合扫描扩展名且未被排除的文件路径数组
 */
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

/**
 * 判断文件是否属于需要扫描的源码文件。
 *
 * @param rootDir - 仓库根目录
 * @param file - 待检查的文件路径
 * @returns `true` if 文件位于 `src/` 下，且不在生成的 OpenAPI 目录或测试/规范文件范围内，`false` otherwise.
 */
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

/**
 * 计算源码中指定位置对应的行号。
 *
 * @param source - 源码文本
 * @param index - 字符索引
 * @returns 该索引所在的 1-based 行号
 */
function lineNumberForIndex(source: string, index: number) {
  return source.slice(0, index).split('\n').length;
}

/**
 * 提取指定位置所在行的文本片段。
 *
 * @param source - 源文本
 * @param index - 目标字符位置
 * @returns 该位置所在行去除首尾空白后的内容
 */
function snippetForIndex(source: string, index: number) {
  const line = source.split('\n')[lineNumberForIndex(source, index) - 1] ?? '';
  return line.trim();
}

/**
 * 将源码中的注释和字面量内容替换为空格，同时保留换行结构。
 *
 * @param source - 原始源码文本
 * @returns 处理后的源码文本
 */
function sanitizeSource(source: string) {
  let result = '';
  let index = 0;
  let state: 'code' | 'line-comment' | 'block-comment' | 'single-quote' | 'double-quote' | 'template' = 'code';

  while (index < source.length) {
    const char = source[index] ?? '';
    const next = source[index + 1] ?? '';

    switch (state) {
      case 'code':
        if (char === '/' && next === '/') {
          result += '  ';
          index += 2;
          state = 'line-comment';
          continue;
        }
        if (char === '/' && next === '*') {
          result += '  ';
          index += 2;
          state = 'block-comment';
          continue;
        }
        if (char === "'") {
          result += ' ';
          index += 1;
          state = 'single-quote';
          continue;
        }
        if (char === '"') {
          result += ' ';
          index += 1;
          state = 'double-quote';
          continue;
        }
        if (char === '`') {
          result += ' ';
          index += 1;
          state = 'template';
          continue;
        }
        result += char;
        index += 1;
        continue;
      case 'line-comment':
        result += char === '\n' ? '\n' : ' ';
        index += 1;
        if (char === '\n') {
          state = 'code';
        }
        continue;
      case 'block-comment':
        if (char === '*' && next === '/') {
          result += '  ';
          index += 2;
          state = 'code';
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
          state = 'code';
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
          state = 'code';
        }
        continue;
      case 'template':
        if (char === '\\') {
          result += next === '\n' ? ' \n' : '  ';
          index += Math.min(2, source.length - index);
          continue;
        }
        result += char === '\n' ? '\n' : ' ';
        index += 1;
        if (char === '`') {
          state = 'code';
        }
        continue;
    }
  }

  return result;
}

/**
 * 判断包含指定位置的行是否带有允许注释。
 *
 * @param source - 源码文本
 * @param index - 目标位置的字符索引
 * @returns `true` if the line containing `index` includes `NATIVE_DIALOG_ALLOW_COMMENT`, `false` otherwise.
 */
function shouldIgnoreLine(source: string, index: number) {
  const line = source.split('\n')[lineNumberForIndex(source, index) - 1] ?? '';
  return line.includes(NATIVE_DIALOG_ALLOW_COMMENT);
}

/**
 * 收集项目中浏览器原生对话框调用的命中清单。
 *
 * 遍历 `srcDir` 下符合扫描条件的文件，提取 `alert`、`confirm` 和 `prompt` 的调用位置，并生成对应的文件、行号和代码片段。
 *
 * @returns 原生对话框调用命中项列表。
 */
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

/**
 * 扫描项目中的浏览器原生对话框调用并生成治理报告。
 *
 * @param options - 扫描根目录及源目录配置。
 * @returns 包含命中条目列表和格式化报告文本的结果。
 */
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
