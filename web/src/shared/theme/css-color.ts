/** 颜色解析结果；同时保留规范化 hex、CSS 文本和透明度，供主题编辑与预览共享。 */
export interface ParsedCssColor {
  alpha: number;
  blue: number;
  css: string;
  green: number;
  hex: string;
  red: number;
}

function toHexSegment(value: number) {
  return Math.max(0, Math.min(255, Math.round(value)))
    .toString(16)
    .padStart(2, '0');
}

function normalizeHexInput(value: string) {
  const normalized = value.trim().replace(/^#?/, '#');
  const compact = normalized.slice(1);

  if (/^[0-9a-fA-F]{3}$/.test(compact)) {
    return `#${compact
      .split('')
      .map((item) => `${item}${item}`)
      .join('')
      .toLowerCase()}`;
  }

  if (/^[0-9a-fA-F]{4}$/.test(compact)) {
    // 归一化 hex 的调用方只需要 RGB；alpha 由结构化解析结果单独保留。
    const [red, green, blue] = compact
      .slice(0, 3)
      .split('')
      .map((item) => `${item}${item}`);
    return `#${red}${green}${blue}`.toLowerCase();
  }

  if (/^[0-9a-fA-F]{6}$/.test(compact)) {
    return normalized.toLowerCase();
  }

  if (/^[0-9a-fA-F]{8}$/.test(compact)) {
    // 与简写分支一致：先归一化 RGB，结构化解析时再保留 alpha。
    return `#${compact.slice(0, 6)}`.toLowerCase();
  }

  return null;
}

function buildParsedCssColor(red: number, green: number, blue: number, alpha = 1): ParsedCssColor {
  const normalizedAlpha = Math.max(0, Math.min(1, alpha));
  const hex = `#${toHexSegment(red)}${toHexSegment(green)}${toHexSegment(blue)}`;

  return {
    alpha: normalizedAlpha,
    blue,
    css: normalizedAlpha >= 0.999 ? hex : `rgba(${red}, ${green}, ${blue}, ${normalizedAlpha})`,
    green,
    hex,
    red,
  };
}

function parseHexColor(value: string): ParsedCssColor | null {
  const normalized = value.trim().replace(/^#?/, '#');
  const compact = normalized.slice(1);

  if (!/^[0-9a-fA-F]{3,8}$/.test(compact) || ![3, 4, 6, 8].includes(compact.length)) {
    return null;
  }

  const expanded =
    compact.length === 3 || compact.length === 4
      ? compact
          .split('')
          .map((item) => `${item}${item}`)
          .join('')
      : compact;
  const red = Number.parseInt(expanded.slice(0, 2), 16);
  const green = Number.parseInt(expanded.slice(2, 4), 16);
  const blue = Number.parseInt(expanded.slice(4, 6), 16);
  const alpha = expanded.length === 8 ? Number.parseInt(expanded.slice(6, 8), 16) / 255 : 1;

  return buildParsedCssColor(red, green, blue, alpha);
}

function parseRgbChannel(value: string) {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed >= 0 && parsed <= 255 ? parsed : null;
}

function parseAlphaChannel(value: string | undefined) {
  if (value === undefined) {
    return 1;
  }

  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed >= 0 && parsed <= 1 ? parsed : null;
}

function buildParsedMatchedColor(matched: RegExpMatchArray | null, parseChannel: (value: string) => number | null) {
  if (!matched) {
    return null;
  }

  const red = parseChannel(matched[1]);
  const green = parseChannel(matched[2]);
  const blue = parseChannel(matched[3]);
  const alpha = parseAlphaChannel(matched[4]);

  if (red === null || green === null || blue === null || alpha === null) {
    return null;
  }

  return buildParsedCssColor(red, green, blue, alpha);
}

function parseRgbColor(value: string): ParsedCssColor | null {
  return buildParsedMatchedColor(
    value.match(
      /^rgba?\(\s*([+-]?\d+(?:\.\d+)?)\s*(?:,\s*|\s+)([+-]?\d+(?:\.\d+)?)\s*(?:,\s*|\s+)([+-]?\d+(?:\.\d+)?)(?:\s*(?:,|\/)\s*([+-]?\d*(?:\.\d+)?))?\s*\)$/i,
    ),
    parseRgbChannel,
  );
}

function parseSrgbChannel(value: string) {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed >= 0 && parsed <= 1 ? Math.round(parsed * 255) : null;
}

function parseSrgbColor(value: string): ParsedCssColor | null {
  return buildParsedMatchedColor(
    value.match(
      /^color\(\s*srgb\s+([+-]?\d*(?:\.\d+)?)\s+([+-]?\d*(?:\.\d+)?)\s+([+-]?\d*(?:\.\d+)?)(?:\s*\/\s*([+-]?\d*(?:\.\d+)?))?\s*\)$/i,
    ),
    parseSrgbChannel,
  );
}

/** 先处理无须浏览器的颜色语法，再用 DOM 解析命名色等浏览器支持的格式。 */
export function normalizeCssColorValue(value: string) {
  const trimmed = value.trim();

  if (!trimmed) {
    return null;
  }

  const normalizedHex = normalizeHexInput(trimmed);
  if (normalizedHex) {
    return normalizedHex;
  }

  if (parseRgbColor(trimmed) || parseSrgbColor(trimmed)) {
    return trimmed;
  }

  // SSR 或测试环境没有 DOM 时不能解析浏览器专属颜色名，必须返回 null 而不是伪造结果。
  if (typeof document === 'undefined') {
    return null;
  }

  const probe = document.createElement('span');
  probe.style.color = '';
  probe.style.color = trimmed;

  if (!probe.style.color) {
    return null;
  }

  probe.style.position = 'absolute';
  probe.style.pointerEvents = 'none';
  probe.style.opacity = '0';

  const parent = document.body ?? document.documentElement;
  if (!parent) {
    return probe.style.color;
  }

  parent.appendChild(probe);
  const resolved = getComputedStyle(probe).color || probe.style.color;
  probe.remove();

  return resolved || null;
}

/** 解析常见颜色语法，并在必要时借助浏览器计算样式把可识别颜色转为结构化结果。 */
export function parseResolvedCssColor(value: string) {
  const trimmed = value.trim();

  if (!trimmed) {
    return null;
  }

  return (
    parseHexColor(trimmed) ??
    parseRgbColor(trimmed) ??
    parseSrgbColor(trimmed) ??
    (() => {
      const normalized = normalizeCssColorValue(trimmed);
      if (!normalized || normalized === trimmed) {
        return null;
      }
      return parseHexColor(normalized) ?? parseRgbColor(normalized) ?? parseSrgbColor(normalized);
    })()
  );
}
