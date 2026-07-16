import { parseResolvedCssColor } from '@/shared/theme/css-color';

export type ParsedThemeTokenColor = NonNullable<ReturnType<typeof parseResolvedCssColor>>;

const FALLBACK_HEX = '#0052d9';

/** 判断主题 token 是否属于颜色相关属性；阴影 token 虽可能包含颜色，但不进入颜色编辑流程。 */
export function isThemeTokenColorKey(tokenKey: string) {
  return /color|background|border/i.test(tokenKey) && !/shadow/i.test(tokenKey);
}

/** 将主题 token 的颜色值解析为编辑器所需的通道、透明度和 CSS 表示；无法解析时返回 null。 */
export function parseThemeTokenColor(value: string) {
  const trimmed = value.trim();

  if (!trimmed) {
    return null;
  }

  return parseResolvedCssColor(trimmed);
}

/** 根据颜色和百分比透明度生成 token 值；透明度会收敛到 0-100，并在完全不透明时保持 hex 格式。 */
export function buildThemeTokenColorValue(hexValue: string, opacityValue: number) {
  const parsed = parseResolvedCssColor(hexValue);

  if (!parsed) {
    return null;
  }

  const opacity = Math.max(0, Math.min(100, Math.round(opacityValue)));

  if (opacity >= 100) {
    return parsed.hex;
  }

  return `rgba(${parsed.red}, ${parsed.green}, ${parsed.blue}, ${opacity / 100})`;
}

/** 格式化工作台中的 token 摘要；颜色值以 hex 和透明度展示，其他 token 保留原值。 */
export function formatThemeTokenSummaryValue(tokenKey: string, value: string) {
  const trimmed = value.trim();

  if (!trimmed) {
    return '--';
  }

  if (!isThemeTokenColorKey(tokenKey)) {
    return trimmed;
  }

  const parsed = parseThemeTokenColor(trimmed);

  if (!parsed) {
    return trimmed;
  }

  const opacity = Math.round(parsed.alpha * 100);
  return opacity >= 100 ? parsed.hex : `${parsed.hex} / ${opacity}%`;
}

/** 为主题 token 预览提供稳定的 `#RRGGBB` 值，解析失败时使用工作台默认色避免预览失效。 */
export function resolveThemeTokenPreviewHex(value: string) {
  return parseThemeTokenColor(value)?.hex ?? FALLBACK_HEX;
}
