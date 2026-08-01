import { join } from 'node:path';

import { collectLocaleCatalogs, localeViolation } from '../locale-utils';
import type { I18nGovernanceRule } from '../types';

const SERVER_MODULE_LOCALE_PATTERN = /^server\/modules\/([^/]+)\/locales\/(?:zh-CN|en-US)\.yaml$/;
const WEB_MODULE_LOCALE_PATTERN = /^src\/modules\/([^/]+)\/locales\/(zh-CN|en-US)\.json$/;
const ERROR_KEY_PATTERN = /^ops\..+\.error\./;

export const noMissingModuleErrorWebLocaleRule: I18nGovernanceRule = {
  id: 'no-missing-module-error-web-locale',
  description:
    'Blocks module-owned backend user error keys that are missing from the matching web module locale catalogs.',
  defaultSeverity: 'error',
  appliesTo: ['locale'],
  check(context) {
    const catalogs = collectLocaleCatalogs(context);
    const webCatalogs = new Map<string, Set<string>>();

    for (const catalog of catalogs) {
      const match = catalog.file.match(WEB_MODULE_LOCALE_PATTERN);
      if (!match) continue;
      webCatalogs.set(`${match[1]}:${match[2]}`, new Set(catalog.messages.keys()));
    }

    return catalogs
      .filter((catalog) => SERVER_MODULE_LOCALE_PATTERN.test(catalog.file))
      .flatMap((catalog) => {
        const match = catalog.file.match(SERVER_MODULE_LOCALE_PATTERN);
        if (!match) return [];

        const locale = catalog.locale;
        const webKeys = webCatalogs.get(`${match[1]}:${locale}`) ?? new Set<string>();
        return [...catalog.messages.keys()]
          .filter((key) => ERROR_KEY_PATTERN.test(key) && !webKeys.has(key))
          .sort()
          .map((key) =>
            localeViolation(
              noMissingModuleErrorWebLocaleRule.id,
              'error',
              join('src', 'modules', match[1], 'locales', `${locale}.json`),
              `module error locale key ${key} is missing from the ${locale} web locale catalog`,
              `Add the key to web/src/modules/${match[1]}/locales/${locale}.json so frontend API errors remain localized.`,
            ),
          );
      });
  },
};
