import { readdirSync, readFileSync } from 'node:fs';
import { extname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

/**
 * 解析当前模块所在项目的默认根目录。
 *
 * @returns 当前模块上一级目录的路径；若 URL 解析失败，则返回当前工作目录。
 */
function resolveDefaultRootDir() {
  try {
    return fileURLToPath(new URL('..', import.meta.url));
  } catch {
    return process.cwd();
  }
}

const DEFAULT_ROOT_DIR = resolveDefaultRootDir();

const SCANNED_EXTENSIONS = new Set(['.vue', '.less', '.css', '.scss', '.sass']);
const EXCLUDED_DIRS = new Set(['node_modules', 'dist', 'coverage', 'mock', '__mocks__', 'ai-libs', 'assets']);
const CSS_BLOCK_PATTERN = /(?<selector>[^{}@;]+?)\s*\{(?<body>[\s\S]*?)\}/g;
const NATIVE_SCROLLBAR_PATTERN = /scrollbar-(?:color|width|gutter)|::-webkit-scrollbar(?:-(?:track|thumb))?/;
const GRAFT_SCROLLBAR_PATTERN = /\bgraft-scrollbar\b/;

type AllowlistEntry = {
  file: string;
  selector: string;
  reason: string;
  cleanupTrigger: string;
};

type InventoryItem = {
  file: string;
  line: number;
  selector: string;
  snippet: string;
  allowlisted: boolean;
  reason?: string;
  cleanupTrigger?: string;
};

type ScrollbarGovernanceAuditOptions = {
  rootDir?: string;
  srcDir?: string;
};

const ALLOWLIST: AllowlistEntry[] = [
  {
    file: 'src/style/scrollbar.less',
    selector: '.graft-scrollbar',
    reason: 'Shared runtime scrollbar utility; the only approved native scrollbar styling entrypoint.',
    cleanupTrigger: 'Keep all visible runtime scrollbar styling centralized in the shared scrollbar utility.',
  },
  {
    file: 'src/shared/components/markdown/SafeMarkdown.vue',
    selector: '*',
    reason: 'Generated markdown can emit arbitrary pre/table nodes that cannot receive a utility class directly.',
    cleanupTrigger: 'Replace the markdown renderer with a class-injection path or a shared scrollbar mixin hook.',
  },
];

/**
 * 递归收集指定目录下符合扫描扩展名的文件路径。
 *
 * @param dir - 要遍历的目录
 * @returns 匹配的文件路径数组
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

    if (!SCANNED_EXTENSIONS.has(extname(fullPath))) {
      continue;
    }

    files.push(fullPath);
  }

  return files;
}

/**
 * 判断文件是否应纳入扫描。
 *
 * @param rootDir - 仓库根目录
 * @param file - 待检查的文件路径
 * @returns `true` if the file should be scanned, `false` otherwise.
 */
function shouldScanFile(rootDir: string, file: string): boolean {
  const normalized = relative(rootDir, file).replaceAll('\\', '/');
  if (/\.d\.ts$/.test(normalized) || /\.(?:test|spec)\.(?:ts|tsx|vue)$/.test(normalized)) {
    return false;
  }

  return !normalized.startsWith('src/contracts/openapi/generated/');
}

/**
 * 将字符位置转换为行号。
 *
 * @param source - 源文本
 * @param index - 字符索引
 * @returns 对应的 1 基行号
 */
function lineNumberForIndex(source: string, index: number): number {
  return source.slice(0, index).split('\n').length;
}

/**
 * 查找文件和选择器对应的白名单项。
 *
 * @param file - 相对文件路径
 * @param selector - CSS 选择器
 * @returns 匹配的白名单项；未找到时返回 `null`
 */
function resolveAllowlist(file: string, selector: string): AllowlistEntry | null {
  return (
    ALLOWLIST.find(
      (entry) =>
        entry.file === file &&
        (entry.selector === '*' || entry.selector === selector || selector.includes(entry.selector)),
    ) ?? null
  );
}

/**
 * 收集仓库中命中的滚动条样式清单。
 *
 * @param rootDir - 仓库根目录
 * @param srcDir - 需要扫描的源代码目录
 * @returns 命中的清单项数组
 */
function collectInventory(rootDir: string, srcDir: string): InventoryItem[] {
  const inventory: InventoryItem[] = [];

  for (const file of walk(srcDir)) {
    const rel = relative(rootDir, file).replaceAll('\\', '/');
    if (!shouldScanFile(rootDir, file)) {
      continue;
    }

    const source = readFileSync(file, 'utf8');

    for (const match of source.matchAll(CSS_BLOCK_PATTERN)) {
      const selector = match.groups?.selector?.trim() || '<unknown>';
      const body = match.groups?.body ?? '';
      if (!NATIVE_SCROLLBAR_PATTERN.test(selector) && !NATIVE_SCROLLBAR_PATTERN.test(body)) {
        continue;
      }

      const index = match.index ?? 0;
      const snippet = `${selector} { ${body.trim().replace(/\s+/g, ' ')} }`;
      const allowlistEntry = resolveAllowlist(rel, selector);

      inventory.push({
        file: rel,
        line: lineNumberForIndex(source, index),
        selector,
        snippet,
        allowlisted:
          Boolean(allowlistEntry) || GRAFT_SCROLLBAR_PATTERN.test(selector) || GRAFT_SCROLLBAR_PATTERN.test(body),
        reason: allowlistEntry?.reason,
        cleanupTrigger: allowlistEntry?.cleanupTrigger,
      });
    }
  }

  return inventory;
}

/**
 * 审核仓库中的原生滚动条样式并生成汇总结果。
 *
 * @param options - 审核范围配置
 * @returns 包含违规项、允许项和格式化输出的审核结果
 */
export function runScrollbarGovernanceAudit(options: ScrollbarGovernanceAuditOptions = {}) {
  const rootDir = options.rootDir ?? DEFAULT_ROOT_DIR;
  const srcDir = options.srcDir ?? join(rootDir, 'src');
  const inventory = collectInventory(rootDir, srcDir);
  const debt = inventory.filter((item) => !item.allowlisted);
  const exceptions = inventory.filter((item) => item.allowlisted);

  let output = 'Scrollbar governance inventory:\n';

  if (exceptions.length > 0) {
    output += 'Allowed exceptions:\n';
    for (const item of exceptions) {
      const reason = item.reason ? ` reason=${item.reason}` : '';
      const cleanupTrigger = item.cleanupTrigger ? ` cleanup=${item.cleanupTrigger}` : '';
      output += `- ${item.file}:${item.line} ${item.selector}${reason}${cleanupTrigger}\n`;
    }
  }

  if (debt.length > 0) {
    output += 'Blacklisted native scrollbar styling found:\n';
    for (const item of debt) {
      output += `- ${item.file}:${item.line} ${item.selector} -> ${item.snippet}\n`;
    }
    output +=
      '\nOnly the shared `graft-scrollbar` utility and explicit allowlist entries may define native scrollbar styling. Do not add page-local `scrollbar-*` or `::-webkit-scrollbar*` rules.\n';
  } else {
    output += 'Scrollbar governance: no blacklisted native scrollbar styles found.\n';
  }

  return {
    debt,
    exceptions,
    output,
  };
}

if (import.meta.main) {
  const result = runScrollbarGovernanceAudit();
  process.stdout.write(result.output);
  if (result.debt.length > 0) {
    process.exitCode = 1;
  }
}
