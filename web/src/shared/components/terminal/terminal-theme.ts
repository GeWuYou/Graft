import type { ITerminalOptions } from '@xterm/xterm';

const TERMINAL_FONT_FAMILY = [
  'ui-monospace',
  'SFMono-Regular',
  'Menlo',
  'Monaco',
  'Consolas',
  '"Liberation Mono"',
  'monospace',
].join(', ');

/**
 * 创建终端的主题和样式配置选项。
 *
 * @returns 包含终端外观、字体、光标和颜色主题的配置对象
 */
export function createTerminalThemeOptions(): Pick<
  ITerminalOptions,
  | 'allowProposedApi'
  | 'convertEol'
  | 'cursorBlink'
  | 'cursorStyle'
  | 'drawBoldTextInBrightColors'
  | 'fontFamily'
  | 'fontSize'
  | 'lineHeight'
  | 'scrollback'
  | 'theme'
> {
  return {
    allowProposedApi: false,
    convertEol: false,
    cursorBlink: true,
    cursorStyle: 'block',
    drawBoldTextInBrightColors: true,
    fontFamily: TERMINAL_FONT_FAMILY,
    fontSize: 13,
    lineHeight: 1.35,
    scrollback: 5000,
    theme: {
      background: '#0f1720',
      black: '#141b24',
      blue: '#6fb1ff',
      brightBlack: '#556270',
      brightBlue: '#8bc2ff',
      brightCyan: '#7fe9d7',
      brightGreen: '#78f0ab',
      brightMagenta: '#f4a6ff',
      brightRed: '#ff9a91',
      brightWhite: '#eef3f8',
      brightYellow: '#ffe08a',
      cursor: '#e6edf5',
      cyan: '#53d4c0',
      foreground: '#d9e2ec',
      green: '#57d989',
      magenta: '#d593ff',
      red: '#ff7b72',
      selectionBackground: 'rgb(118 170 255 / 28%)',
      white: '#c5d1dd',
      yellow: '#d7b24d',
    },
  };
}
