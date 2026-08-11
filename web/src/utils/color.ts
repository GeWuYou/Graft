import { Color } from 'tvision-color';

import type { TColorToken } from '@/config/color';
import type { ThemeTokenMap } from '@/types/theme';
import type { ModeType } from '@/utils/types';

const THEME_STYLE_TAG_PREFIX = 'graft-theme-style';
const GRAFT_FAVICON_LINK_ID = 'graft-favicon';
const GRAFT_FAVICON_PATH =
  'M16 3C8.8203 3 3 8.8203 3 16C3 23.1797 8.8203 29 16 29C22.1963 29 27.3799 24.6641 28.6836 18.8594H18.0312V14.5469H28.8828C29.002 15.0078 29.0625 15.4941 29.0625 16C29.0625 23.1758 23.1719 29.0625 16 29.0625C8.82812 29.0625 2.9375 23.1758 2.9375 16C2.9375 8.82422 8.82812 2.9375 16 2.9375C20.3281 2.9375 24.1641 5.05078 26.5312 8.30469L23.1719 10.7539C21.5859 8.58203 18.9922 7.17188 16 7.17188C11.1641 7.17188 7.17188 11.1641 7.17188 16C7.17188 20.8359 11.1641 24.8281 16 24.8281C19.5781 24.8281 22.8086 22.6484 24.168 19.3516H16V15.0391H28.7109C28.7617 15.3594 28.7891 15.6836 28.7891 16C28.7891 23.0195 23.0195 28.7891 16 28.7891C8.98047 28.7891 3.21094 23.0195 3.21094 16C3.21094 8.98047 8.98047 3.21094 16 3.21094C20.1094 3.21094 23.7578 5.15625 26.0703 8.17188L22.707 10.6562C21.4258 8.93359 18.8555 7.44141 16 7.44141C11.3164 7.44141 7.44141 11.3164 7.44141 16C7.44141 20.6836 11.3164 24.5586 16 24.5586C19.4219 24.5586 22.4805 22.5 23.8398 19.3516H14.5781V11.8906H28.1172C27.1523 6.81641 22.6836 3 16 3Z';

function createFaviconSvg(color: string) {
  return `<svg width="64" height="64" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="${GRAFT_FAVICON_PATH}" fill="${color}" /></svg>`;
}

/**
 * 外部 favicon 无法继承页面 CSS token，因此在主题应用时序列化当前品牌色。
 */
export function syncFaviconColor(color: string) {
  if (typeof document === 'undefined' || !color.trim()) {
    return;
  }

  const favicon = document.getElementById?.(GRAFT_FAVICON_LINK_ID);
  if (!favicon || favicon.tagName !== 'LINK') {
    return;
  }

  const faviconLink = favicon as HTMLLinkElement;
  faviconLink.type = 'image/svg+xml';
  faviconLink.href = `data:image/svg+xml,${encodeURIComponent(createFaviconSvg(color))}`;
}

/**
 * 根据当前主题色、模式等情景 计算最后生成的色阶
 */
function generateColorMap(theme: string, colorPalette: Array<string>, mode: ModeType, brandColorIdx: number) {
  const isDarkMode = mode === 'dark';

  if (isDarkMode) {
    colorPalette.reverse().map((color) => {
      const [h, s, l] = Color.colorTransform(color, 'hex', 'hsl');
      return Color.colorTransform([h, Number(s) - 4, l], 'hsl', 'hex');
    });
    brandColorIdx = 5;
    colorPalette[0] = `${colorPalette[brandColorIdx]}20`;
  }

  const colorMap: TColorToken = {
    '--td-brand-color': colorPalette[brandColorIdx], // 主题色
    '--td-brand-color-1': colorPalette[0], // light
    '--td-brand-color-2': colorPalette[1], // focus
    '--td-brand-color-3': colorPalette[2], // disabled
    '--td-brand-color-4': colorPalette[3],
    '--td-brand-color-5': colorPalette[4],
    '--td-brand-color-6': colorPalette[5],
    '--td-brand-color-7': brandColorIdx > 0 ? colorPalette[brandColorIdx - 1] : theme, // hover
    '--td-brand-color-8': colorPalette[brandColorIdx], // 主题色
    '--td-brand-color-9': brandColorIdx > 8 ? theme : colorPalette[brandColorIdx + 1], // click
    '--td-brand-color-10': colorPalette[9],
  };
  return colorMap;
}

/**
 * 依据品牌色生成当前模式下的品牌 token。
 */
export function generateBrandColorMap(theme: string, mode: ModeType): TColorToken {
  const [{ colors: newPalette, primary: brandColorIndex }] = Color.getColorGradations({
    colors: [theme],
    step: 10,
    remainInput: false,
  });

  return generateColorMap(theme, newPalette, mode, brandColorIndex);
}

export function composeThemeTokenMap(...maps: Array<ThemeTokenMap | undefined>): ThemeTokenMap {
  return maps.reduce<ThemeTokenMap>((merged, current) => {
    if (!current) {
      return merged;
    }

    return {
      ...merged,
      ...current,
    };
  }, {});
}

function getThemeStyleTagId(mode: ModeType) {
  return `${THEME_STYLE_TAG_PREFIX}-${mode}`;
}

function ensureThemeStyleTag(mode: ModeType): HTMLStyleElement {
  const styleTagId = getThemeStyleTagId(mode);
  const existingStyleTag = document.getElementById(styleTagId);

  if (existingStyleTag instanceof HTMLStyleElement) {
    return existingStyleTag;
  }

  const styleSheet = document.createElement('style');
  styleSheet.id = styleTagId;
  styleSheet.type = 'text/css';
  document.head.appendChild(styleSheet);
  return styleSheet;
}

/**
 * 将生成的样式嵌入头部
 */
export function insertThemeStylesheet(theme: string, colorMap: ThemeTokenMap, mode: ModeType) {
  const isDarkMode = mode === 'dark';
  const root = !isDarkMode ? `:root[theme-color='${theme}']` : `:root[theme-color='${theme}'][theme-mode='dark']`;
  const styleSheet = ensureThemeStyleTag(mode);
  const declarations = Object.entries(colorMap)
    .map(([key, value]) => `    ${key}: ${value};`)
    .join('\n');

  styleSheet.textContent = `${root}{
${declarations}
  }`;
}
