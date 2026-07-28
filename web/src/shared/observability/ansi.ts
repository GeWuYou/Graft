const ESC = String.fromCharCode(27);
const BEL = String.fromCharCode(7);
const ANSI_CSI_PATTERN = new RegExp(`(?:${ESC}\\[|\\u009B)[0-?]*[ -/]*[@-~]`, 'g');
const ANSI_OSC_PATTERN = new RegExp(`(?:${ESC}\\]|\\u009D).*?(?:${BEL}|${ESC}\\\\|\\u009C)`, 'gs');

/**
 * 移除终端控制序列，保留可读的日志文本。
 *
 * @returns 不包含 ANSI CSI 或 OSC 序列的文本
 */
export function stripAnsiControlSequences(value: string) {
  return value.replace(ANSI_OSC_PATTERN, '').replace(ANSI_CSI_PATTERN, '');
}
