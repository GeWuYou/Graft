/**
 * 基于当前页面主机生成实时 WebSocket 地址。
 *
 * @param relativePath - 相对于当前主机的 WebSocket 路径
 * @returns 生成的绝对 WebSocket URL 字符串
 */
export function toRealtimeWebSocketUrl(relativePath: string) {
  const base = new URL(window.location.href);
  const protocol = base.protocol === 'https:' ? 'wss:' : 'ws:';
  return new URL(relativePath, `${protocol}//${base.host}`).toString();
}
