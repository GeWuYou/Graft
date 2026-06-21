import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

export const ROOT_DIR = fileURLToPath(new URL('../..', import.meta.url));
export const REPOSITORY_DIR = fileURLToPath(new URL('../../..', import.meta.url));
export const SRC_DIR = join(ROOT_DIR, 'src');

export const SCANNED_EXTENSIONS = new Set(['.vue', '.ts', '.tsx']);
export const EXCLUDED_DIRS = new Set(['node_modules', 'dist', 'coverage', 'mock', '__mocks__', 'ai-libs']);
export const SERVER_KEY_DIRS = [join(REPOSITORY_DIR, 'server/internal'), join(REPOSITORY_DIR, 'server/modules')];

// Key-first governance tiers:
// - Runtime UI copy must use locale keys as the primary display source.
// - Registration/contract boundaries may keep key + fallback pairs, such as TitleKey + Title, where fallback is only
//   cross-client/config/registry resilience and not the localization source of truth.
// - Fallback-only declarations, such as Title without TitleKey, are risks: default mode reports warnings and
//   STRICT_I18N_KEY_FIRST=true promotes them to blockers.
// - Fallback/default-locale mismatch is a future warning track, not a blocker in this default gate.
// Allowlist: tests, mocks, generated artifacts, examples, logs/debug text, protocols, technical names, internal codes.
export const UI_COPY_FIELDS = new Set([
  'label',
  'title',
  'description',
  'placeholder',
  'content',
  'body',
  'header',
  'emptyText',
  'text',
  'message',
  'fallbackLabel',
  'moreLabelFallback',
  'semanticTitle',
  'breadcrumbTitle',
  'tabTitle',
  'helperText',
  'help',
  'tips',
  'tooltip',
  'confirmText',
  'cancelText',
  'confirmBtn',
  'cancelBtn',
  'okText',
  'closeText',
  'empty',
  'emptyTitle',
  'emptyDescription',
  'successMessage',
  'errorMessage',
  'fallbackMessage',
  'ariaLabel',
]);

export const KEY_FIELDS = new Set([
  'key',
  'labelKey',
  'titleKey',
  'title_key',
  'descriptionKey',
  'description_key',
  'messageKey',
  'message_key',
  'displayKey',
  'display_key',
  'emptyKey',
  'empty_key',
  'placeholderKey',
  'placeholder_key',
  'unitKey',
  'unit_key',
]);

export const KNOWN_NON_I18N_NAMES = new Set([
  'Axios',
  'Bun',
  'Casbin',
  'CSS',
  'Ent',
  'Gin',
  'Go',
  'Graft',
  'HTML',
  'HTTP',
  'HTTPS',
  'JSON',
  'HarmonyOS Sans',
  'Inter',
  'Pinia',
  'PostgreSQL',
  'Redis',
  'Source Han Sans',
  'TDesign',
  'TDesign Original',
  'Tencent Cloud',
  'TypeScript',
  'UnoCSS',
  'Vite',
  'Vue',
  'Zap',
]);

export const TECHNICAL_UNITS = new Set(['ms', 'px', 'em', 'rem', 'vh', 'vw']);

export const STRICT_I18N_KEY_FIRST = /^(?:1|true|yes)$/i.test(process.env.STRICT_I18N_KEY_FIRST ?? '');
