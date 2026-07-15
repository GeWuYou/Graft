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

const ROOT_DIR = resolveDefaultRootDir();
const SCANNED_EXTENSIONS = new Set(['.vue', '.less', '.css', '.scss', '.sass']);
const EXCLUDED_DIRS = new Set(['node_modules', 'dist', 'coverage', 'mock', '__mocks__', 'ai-libs', 'assets']);
const CSS_BLOCK_PATTERN = /(?<selector>[^{}@;]+?)\s*\{(?<body>[^{}]*)\}/g;
const COLOR_DECLARATION_PATTERN = /(?:^|[;\n\r])\s*color\s*:\s*(?<value>[^;{}]+);?/g;
const SAFE_COLOR_PATTERN = /^(?:inherit|currentColor|var\(\s*--(?:td|graft)-[\w-]+(?:\s*,\s*[^)]*)?\s*\))$/;

type ThemeGovernanceFinding = {
  file: string;
  line: number;
  selector: string;
  detail: string;
};

type AuditOptions = {
  rootDir?: string;
  srcDir?: string;
};

type StyleBlock = {
  source: string;
  offset: number;
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

    if (SCANNED_EXTENSIONS.has(extname(file))) {
      files.push(file);
    }
  }

  return files;
}

function shouldScanFile(rootDir: string, file: string): boolean {
  const normalized = relative(rootDir, file).replaceAll('\\', '/');

  return (
    !/\.(?:test|spec)\.(?:vue|less|css|scss|sass)$/.test(normalized) &&
    !normalized.startsWith('src/contracts/openapi/generated/')
  );
}

function getStyleBlocks(file: string, source: string): StyleBlock[] {
  if (extname(file) !== '.vue') {
    return [{ offset: 0, source }];
  }

  return [...source.matchAll(/<style\b[^>]*>(?<body>[\s\S]*?)<\/style>/gi)].map((match) => ({
    offset: (match.index ?? 0) + match[0].indexOf(match.groups?.body ?? ''),
    source: match.groups?.body ?? '',
  }));
}

function lineNumberForIndex(source: string, index: number): number {
  return source.slice(0, index).split('\n').length;
}

function isNativeButtonSelector(selector: string): boolean {
  const normalized = selector.replace(/\s+/g, ' ').trim();
  if (!/\bbutton\b/.test(normalized)) {
    return false;
  }

  return !/(?:\.t-button\b|:deep\(\s*\.t-button\b|\bt-button\b)/.test(normalized);
}

function isExactButtonSelector(selector: string): boolean {
  return selector
    .split(',')
    .map((item) => item.trim())
    .some((item) => item === 'button');
}

function isSafeColor(value: string): boolean {
  return SAFE_COLOR_PATTERN.test(value.replace(/\s+/g, ' ').trim());
}

function addFinding(findings: ThemeGovernanceFinding[], finding: ThemeGovernanceFinding) {
  if (
    findings.some(
      (item) =>
        item.file === finding.file &&
        item.line === finding.line &&
        item.selector === finding.selector &&
        item.detail === finding.detail,
    )
  ) {
    return;
  }

  findings.push(finding);
}

function runStyleBlockAudit(
  rootDir: string,
  file: string,
  source: string,
  block: StyleBlock,
  findings: ThemeGovernanceFinding[],
) {
  for (const match of block.source.matchAll(CSS_BLOCK_PATTERN)) {
    const selector = match.groups?.selector?.trim() ?? '';
    if (!isNativeButtonSelector(selector)) {
      continue;
    }

    const body = match.groups?.body ?? '';
    for (const colorMatch of body.matchAll(COLOR_DECLARATION_PATTERN)) {
      const value = colorMatch.groups?.value?.trim() ?? '';
      if (isSafeColor(value)) {
        continue;
      }

      const declarationIndex = (match.index ?? 0) + (colorMatch.index ?? 0);
      addFinding(findings, {
        file: relative(rootDir, file).replaceAll('\\', '/'),
        line: lineNumberForIndex(source, block.offset + declarationIndex),
        selector,
        detail: `native button color must use inherit, currentColor, or a --td/--graft theme token; found ${value}`,
      });
    }
  }
}

export function runNativeButtonThemeGovernanceAudit(options: AuditOptions = {}) {
  const rootDir = options.rootDir ?? ROOT_DIR;
  const srcDir = options.srcDir ?? join(rootDir, 'src');
  const findings: ThemeGovernanceFinding[] = [];
  const resetFile = join(srcDir, 'style/reset.less');
  const resetSource = readFileSync(resetFile, 'utf8');
  const buttonBlock = [...resetSource.matchAll(CSS_BLOCK_PATTERN)].find((match) =>
    isExactButtonSelector(match.groups?.selector?.trim() ?? ''),
  );

  if (!buttonBlock) {
    addFinding(findings, {
      file: 'src/style/reset.less',
      line: 1,
      selector: 'button',
      detail: 'global native button theme baseline is missing',
    });
  } else {
    const body = buttonBlock?.groups?.body ?? '';
    const hasColorInheritance = /(?:^|[;\n\r])\s*color\s*:\s*inherit\s*;?/m.test(body);
    const hasFontInheritance = /(?:^|[;\n\r])\s*font\s*:\s*inherit\s*;?/m.test(body);

    if (!hasColorInheritance || !hasFontInheritance) {
      addFinding(findings, {
        file: 'src/style/reset.less',
        line: buttonBlock ? lineNumberForIndex(resetSource, buttonBlock.index ?? 0) : 1,
        selector: buttonBlock?.groups?.selector?.trim() ?? 'button',
        detail: 'global native button theme baseline must declare color: inherit and font: inherit',
      });
    }
  }

  for (const file of walk(srcDir)) {
    if (!shouldScanFile(rootDir, file)) {
      continue;
    }

    const source = readFileSync(file, 'utf8');
    for (const block of getStyleBlocks(file, source)) {
      runStyleBlockAudit(rootDir, file, source, block, findings);
    }
  }

  let output = 'Native button theme governance:\n';
  if (findings.length > 0) {
    output += 'Theme-incompatible native button styles found:\n';
    for (const finding of findings) {
      output += `- ${finding.file}:${finding.line} ${finding.selector} -> ${finding.detail}\n`;
    }
  } else {
    output += 'No theme-incompatible native button styles found.\n';
  }

  return { findings, output };
}

if (import.meta.main) {
  const result = runNativeButtonThemeGovernanceAudit();
  process.stdout.write(result.output);
  if (result.findings.length > 0) {
    process.exitCode = 1;
  }
}
