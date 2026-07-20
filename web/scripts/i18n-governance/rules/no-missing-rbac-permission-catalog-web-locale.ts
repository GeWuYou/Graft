import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

import { collectLocaleCatalogs, localeViolation } from '../locale-utils';
import type { I18nGovernanceRule, RuleViolation } from '../types';

const PERMISSION_KEY_PREFIX = 'rbac.permissionCatalog.';
const RBAC_WEB_LOCALE_PATHS = ['src/modules/rbac/locales/zh-CN.json', 'src/modules/rbac/locales/en-US.json'] as const;
const RBAC_PERMISSION_SOURCE_PATH_FRAGMENTS = [
  '/permission',
  '/module_registration',
  '/route_registration',
  '/bootstrap',
];
const PERMISSION_KEY_FIELD_PATTERN =
  /\b(?:DisplayKey|DescriptionKey|displayKey|descriptionKey|display_key|description_key)\s*:\s*"([^"$]+)"/g;
const PERMISSION_SQL_KEY_PATTERN = /'((?:rbac\.permissionCatalog\.)[^']+)'/g;

function collectRbacMigrationSources(repositoryDir: string) {
  const migrationDir = join(repositoryDir, 'server', 'modules', 'rbac', 'migrations');
  if (!existsSync(migrationDir)) return [];

  return readdirSync(migrationDir, { withFileTypes: true })
    .filter((entry) => entry.isFile() && entry.name.endsWith('.sql'))
    .map((entry) => ({
      relativePath: `server/modules/rbac/migrations/${entry.name}`,
      source: readFileSync(join(migrationDir, entry.name), 'utf8'),
    }));
}

export const noMissingRbacPermissionCatalogWebLocaleRule: I18nGovernanceRule = {
  id: 'no-missing-rbac-permission-catalog-web-locale',
  description:
    'Blocks RBAC permission catalog locale keys that exist only in backend authority but are missing from the web RBAC locale catalogs.',
  defaultSeverity: 'error',
  appliesTo: ['go', 'locale'],
  check(context) {
    const violations: RuleViolation[] = [];
    const catalogs = collectLocaleCatalogs(context);
    const rbacWebCatalogs = catalogs.filter((catalog) => RBAC_WEB_LOCALE_PATHS.includes(catalog.file as never));
    const webDefinedKeysByLocale = new Map<string, Set<string>>();

    for (const catalog of rbacWebCatalogs) {
      webDefinedKeysByLocale.set(catalog.locale, new Set(catalog.messages.keys()));
    }

    const requiredKeys = new Set<string>();
    for (const file of context.serverFiles) {
      if (!RBAC_PERMISSION_SOURCE_PATH_FRAGMENTS.some((fragment) => file.relativePath.includes(fragment))) {
        continue;
      }
      for (const match of file.source.matchAll(PERMISSION_KEY_FIELD_PATTERN)) {
        const key = match[1]?.trim();
        if (key?.startsWith(PERMISSION_KEY_PREFIX)) {
          requiredKeys.add(key);
        }
      }
    }

    for (const file of collectRbacMigrationSources(context.repositoryDir)) {
      if (!file.source.includes(PERMISSION_KEY_PREFIX)) continue;
      for (const match of file.source.matchAll(PERMISSION_SQL_KEY_PATTERN)) {
        const key = match[1]?.trim();
        if (key) requiredKeys.add(key);
      }
    }

    for (const key of [...requiredKeys].sort()) {
      for (const locale of ['zh-CN', 'en-US'] as const) {
        if (!webDefinedKeysByLocale.get(locale)?.has(key)) {
          violations.push(
            localeViolation(
              noMissingRbacPermissionCatalogWebLocaleRule.id,
              'error',
              join('src', 'modules', 'rbac', 'locales', `${locale}.json`),
              `RBAC permission locale key ${key} is missing from the ${locale} web RBAC locale catalog`,
              `Add the key to web/src/modules/rbac/locales/${locale}.json so permission pages can render localized labels.`,
            ),
          );
        }
      }
    }

    return violations;
  },
};
