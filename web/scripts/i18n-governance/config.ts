import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

export const ROOT_DIR = fileURLToPath(new URL('../..', import.meta.url));
export const REPOSITORY_DIR = fileURLToPath(new URL('../../..', import.meta.url));
export const SRC_DIR = join(ROOT_DIR, 'src');

export const SCANNED_EXTENSIONS = new Set(['.vue', '.ts', '.tsx']);
export const EXCLUDED_DIRS = new Set(['node_modules', 'dist', 'coverage', 'mock', '__mocks__', 'ai-libs']);
export const SERVER_KEY_DIRS = [join(REPOSITORY_DIR, 'server/internal'), join(REPOSITORY_DIR, 'server/modules')];

// Key-first 治理分层：
// - 运行时界面文案必须以 locale key 作为主要展示来源。
// - 注册/契约边界可以保留 key + fallback，例如 TitleKey + Title；fallback 仅用于跨客户端、配置或注册表的韧性，不能成为本地化真值。
// - 只有 fallback 的声明（例如只有 Title 没有 TitleKey）属于风险：默认模式报告 warning，STRICT_I18N_KEY_FIRST=true 时升级为 blocker。
// - fallback 与默认语言不一致属于后续 warning 轨道，在默认门禁中不是 blocker。
// Allowlist：测试、mock、生成产物、示例、日志/调试文本、协议、技术名称和内部代码。
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
