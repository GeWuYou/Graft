// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { Color } from 'tvision-color';

import type { TColorToken } from '@/config/color';
import type { ThemeTokenMap } from '@/types/theme';
import type { ModeType } from '@/utils/types';

const THEME_STYLE_TAG_PREFIX = 'graft-theme-style';

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
