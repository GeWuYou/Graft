import { readdirSync, readFileSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

type JsonRecord = Record<string, unknown>;

type VisibleScope = {
  label: string;
  paths: string[];
  filePath: string;
};

const rootZh = loadJson(resolve(process.cwd(), 'src/locales/lang/zh-CN.json'));
const rootEn = loadJson(resolve(process.cwd(), 'src/locales/lang/en-US.json'));
const auditZh = loadJson(resolve(process.cwd(), 'src/modules/audit/locales/zh-CN.json'));
const auditEn = loadJson(resolve(process.cwd(), 'src/modules/audit/locales/en-US.json'));

const visibleScopes: VisibleScope[] = [
  {
    label: 'root zh visible ui',
    filePath: resolve(process.cwd(), 'src/locales/lang/zh-CN.json'),
    paths: ['common.appName', 'common.copyright', 'layout', 'menu', 'app.result'],
  },
  {
    label: 'root en visible ui',
    filePath: resolve(process.cwd(), 'src/locales/lang/en-US.json'),
    paths: ['common.appName', 'common.copyright', 'layout', 'menu', 'app.result'],
  },
  {
    label: 'monitor zh visible ui',
    filePath: resolve(process.cwd(), 'src/modules/monitor/locales/zh-CN.json'),
    paths: ['monitor.serverStatus'],
  },
  {
    label: 'monitor en visible ui',
    filePath: resolve(process.cwd(), 'src/modules/monitor/locales/en-US.json'),
    paths: ['monitor.serverStatus'],
  },
  {
    label: 'rbac zh visible ui',
    filePath: resolve(process.cwd(), 'src/modules/rbac/locales/zh-CN.json'),
    paths: ['rbac.roleList'],
  },
  {
    label: 'rbac en visible ui',
    filePath: resolve(process.cwd(), 'src/modules/rbac/locales/en-US.json'),
    paths: ['rbac.roleList'],
  },
  {
    label: 'user zh visible ui',
    filePath: resolve(process.cwd(), 'src/modules/user/locales/zh-CN.json'),
    paths: ['user.userList'],
  },
  {
    label: 'user en visible ui',
    filePath: resolve(process.cwd(), 'src/modules/user/locales/en-US.json'),
    paths: ['user.userList'],
  },
  {
    label: 'access-control zh visible ui',
    filePath: resolve(process.cwd(), 'src/modules/access-control/locales/zh-CN.json'),
    paths: ['accessControl.overview'],
  },
  {
    label: 'access-control en visible ui',
    filePath: resolve(process.cwd(), 'src/modules/access-control/locales/en-US.json'),
    paths: ['accessControl.overview'],
  },
  {
    label: 'audit zh visible ui',
    filePath: resolve(process.cwd(), 'src/modules/audit/locales/zh-CN.json'),
    paths: ['audit.logList'],
  },
  {
    label: 'audit en visible ui',
    filePath: resolve(process.cwd(), 'src/modules/audit/locales/en-US.json'),
    paths: ['audit.logList'],
  },
];

const bannedVisibleCopyPatterns = [/starter/i, /demo/i, /最小闭环/, /真实契约/, /\bdebug\b/i, /调试/u];

describe('frontend visible-copy governance', () => {
  it.each(visibleScopes)('keeps banned visible copy out of $label', ({ paths, filePath }) => {
    const document = loadJson(filePath);
    const strings = paths.flatMap((path) => collectStrings(resolvePath(document, path)));

    expect(strings.length).toBeGreaterThan(0);

    strings.forEach((value) => {
      bannedVisibleCopyPatterns.forEach((pattern) => {
        expect(value).not.toMatch(pattern);
      });
    });
  });

  it('keeps owned menu title keys available in both locales', () => {
    const requiredKeys = [
      'menu.role_list.title',
      'menu.user_list.title',
      'menu.access_control.title',
      'menu.access_control.overview.title',
      'menu.server.title',
      'menu.server.overview.title',
      'menu.server.runtime.title',
      'menu.server.dependencies.title',
      'menu.modulesRuntime.title',
      'menu.logCenter.title',
    ];

    requiredKeys.forEach((key) => {
      expect(resolvePath(rootZh, key)).toEqual(expect.any(String));
      expect(resolvePath(rootEn, key)).toEqual(expect.any(String));
    });

    ['menu.audit.title', 'menu.audit.overview.title', 'menu.audit.logs.title'].forEach((key) => {
      expect(resolvePath(auditZh, key)).toEqual(expect.any(String));
      expect(resolvePath(auditEn, key)).toEqual(expect.any(String));
    });
  });

  it('keeps zh-CN and en-US locale key sets aligned', () => {
    collectLocalePairs(resolve(process.cwd(), 'src')).forEach(({ label, zhFile, enFile }) => {
      const zhKeys = collectStringKeys(loadJson(zhFile));
      const enKeys = collectStringKeys(loadJson(enFile));

      expect(
        [...zhKeys].filter((key) => !enKeys.has(key)),
        `${label} missing en-US keys`,
      ).toEqual([]);
      expect(
        [...enKeys].filter((key) => !zhKeys.has(key)),
        `${label} missing zh-CN keys`,
      ).toEqual([]);
    });
  });
});

function loadJson(filePath: string): JsonRecord {
  return JSON.parse(readFileSync(filePath, 'utf8')) as JsonRecord;
}

function resolvePath(document: JsonRecord, dottedPath: string): unknown {
  return dottedPath.split('.').reduce<unknown>((current, segment) => {
    if (!current || typeof current !== 'object' || Array.isArray(current)) {
      return undefined;
    }

    return (current as JsonRecord)[segment];
  }, document);
}

function collectStrings(value: unknown): string[] {
  if (typeof value === 'string') {
    return [value];
  }

  if (Array.isArray(value)) {
    return value.flatMap((entry) => collectStrings(entry));
  }

  if (value && typeof value === 'object') {
    return Object.values(value as JsonRecord).flatMap((entry) => collectStrings(entry));
  }

  return [];
}

function collectLocalePairs(rootDir: string): Array<{ label: string; zhFile: string; enFile: string }> {
  const groups = new Map<string, Partial<Record<'zh-CN' | 'en-US', string>>>();

  walkFiles(rootDir).forEach((filePath) => {
    const match = filePath.match(/(?:^|\/)(zh-CN|en-US)\.json$/);
    if (!match) {
      return;
    }

    const key = relative(rootDir, filePath).replace(/(?:zh-CN|en-US)\.json$/, '{locale}.json');
    const group = groups.get(key) ?? {};
    group[match[1] as 'zh-CN' | 'en-US'] = filePath;
    groups.set(key, group);
  });

  return [...groups.entries()].map(([label, group]) => {
    expect(group['zh-CN'], `${label} zh-CN file`).toEqual(expect.any(String));
    expect(group['en-US'], `${label} en-US file`).toEqual(expect.any(String));

    return {
      label,
      zhFile: group['zh-CN'] as string,
      enFile: group['en-US'] as string,
    };
  });
}

function walkFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const fullPath = join(dir, entry.name);
    return entry.isDirectory() ? walkFiles(fullPath) : [fullPath];
  });
}

function collectStringKeys(value: unknown, prefix = '', output = new Set<string>()): Set<string> {
  if (typeof value === 'string') {
    output.add(prefix);
    return output;
  }

  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return output;
  }

  Object.entries(value as JsonRecord).forEach(([key, child]) => {
    collectStringKeys(child, prefix ? `${prefix}.${key}` : key, output);
  });

  return output;
}
