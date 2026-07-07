import { parseResolvedCssColor } from '@/shared/theme/css-color';

export function toMonacoColor(color: string, fallback: string) {
  const parsed = parseResolvedCssColor(color) ?? parseResolvedCssColor(fallback);

  if (!parsed) {
    return '#0052d9';
  }

  const alphaChannel =
    parsed.alpha >= 0.999
      ? ''
      : Math.round(parsed.alpha * 255)
          .toString(16)
          .padStart(2, '0');

  return `${parsed.hex}${alphaChannel}`;
}
