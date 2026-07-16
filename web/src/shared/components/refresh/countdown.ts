/** 将刷新倒计时按秒、分秒或时分展示；非法值统一返回 `--`，避免把缺失状态误显示为有效倒计时。 */

export function formatRefreshCountdown(seconds: number | null | undefined): string {
  if (typeof seconds !== 'number' || !Number.isFinite(seconds) || seconds < 0) {
    return '--';
  }

  const normalizedSeconds = Math.floor(seconds);
  if (normalizedSeconds < 60) {
    return `${normalizedSeconds}s`;
  }

  if (normalizedSeconds < 3600) {
    const minutes = Math.floor(normalizedSeconds / 60);
    const remainingSeconds = normalizedSeconds % 60;
    return `${minutes}m ${padTimeUnit(remainingSeconds)}s`;
  }

  const hours = Math.floor(normalizedSeconds / 3600);
  const minutes = Math.floor((normalizedSeconds % 3600) / 60);
  return `${hours}h ${padTimeUnit(minutes)}m`;
}

/** 为分钟和秒补齐两位，保持倒计时在单位切换时的视觉宽度稳定。 */
function padTimeUnit(value: number) {
  return String(value).padStart(2, '0');
}
